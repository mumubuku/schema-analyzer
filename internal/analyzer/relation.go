package analyzer

import (
	"fmt"
	"math"
	"schema-analyzer/internal/adapter"
	"schema-analyzer/internal/graph"
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// RelationshipInferer 关系推断器
type RelationshipInferer struct {
	adapter adapter.DBAdapter
}

// NewRelationshipInferer 创建推断器
func NewRelationshipInferer(adapter adapter.DBAdapter) *RelationshipInferer {
	return &RelationshipInferer{adapter: adapter}
}

// InferRelationships 推断表间关系
func (r *RelationshipInferer) InferRelationships(meta *adapter.SchemaMetadata) ([]*graph.Edge, error) {
	var edges []*graph.Edge
	
	fmt.Printf("  正在分析 %d 个表的关系...\n", len(meta.Tables))
	
	// 构建主键映射
	pkMap := make(map[string][]string) // table -> pk columns
	for _, table := range meta.Tables {
		for _, col := range table.Columns {
			if col.IsPrimaryKey {
				pkMap[table.Name] = append(pkMap[table.Name], col.Name)
			}
		}
	}
	
	totalComparisons := 0
	completedComparisons := 0
	
	// 预计算总比较次数
	for _, fromTable := range meta.Tables {
		for _, fromCol := range fromTable.Columns {
			if !fromCol.IsPrimaryKey {
				for _, toTable := range meta.Tables {
					if fromTable.Name != toTable.Name {
						for _, toCol := range toTable.Columns {
							if toCol.IsPrimaryKey {
								totalComparisons++
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("  需要进行 %d 次列比较...\n", totalComparisons)
	
	// 遍历所有表的所有列，寻找可能的外键
	for _, fromTable := range meta.Tables {
		for _, fromCol := range fromTable.Columns {
			// 跳过主键列
			if fromCol.IsPrimaryKey {
				continue
			}
			
			// 与所有其他表的主键比较
			for _, toTable := range meta.Tables {
				if fromTable.Name == toTable.Name {
					continue
				}
				
				for _, toCol := range toTable.Columns {
					if !toCol.IsPrimaryKey {
						continue
					}
					
					completedComparisons++
					if completedComparisons%100 == 0 {
						progress := float64(completedComparisons) / float64(totalComparisons) * 100
						fmt.Printf("  进度: %.1f%% (%d/%d)\n", progress, completedComparisons, totalComparisons)
					}
					
					// 计算关系置信度
					edge := r.calculateRelationship(
						fromTable.Name, fromCol,
						toTable.Name, toCol,
					)
					
					// 降低置信度阈值到 0.3
					if edge != nil && edge.Confidence > 0.3 {
						edges = append(edges, edge)
					}
				}
			}
		}
	}
	
	fmt.Printf("  完成！共发现 %d 个关系\n", len(edges))
	
	return edges, nil
}

// calculateRelationship 计算两列之间的关系
func (r *RelationshipInferer) calculateRelationship(
	fromTable string, fromCol adapter.Column,
	toTable string, toCol adapter.Column,
) *graph.Edge {
	var evidences []graph.Evidence
	totalScore := 0.0
	
	// 1. 命名相似度 (权重 0.3)
	nameScore := r.calculateNameSimilarity(fromCol.Name, toCol.Name)
	if nameScore > 0.3 {
		evidences = append(evidences, graph.Evidence{
			Type:        "naming_similarity",
			Score:       nameScore,
			Description: "列名相似度",
			Details:     fmt.Sprintf("%s ↔ %s (%.2f)", fromCol.Name, toCol.Name, nameScore),
		})
		totalScore += nameScore * 0.3
	}
	
	// 2. 类型匹配 (权重 0.2)
	typeScore := r.calculateTypeMatch(fromCol, toCol)
	if typeScore > 0 {
		evidences = append(evidences, graph.Evidence{
			Type:        "type_match",
			Score:       typeScore,
			Description: "数据类型匹配",
			Details:     fmt.Sprintf("%s(%d) ↔ %s(%d)", fromCol.DataType, fromCol.Length, toCol.DataType, toCol.Length),
		})
		totalScore += typeScore * 0.2
	}
	
	// 3. 值集合包含 (权重 0.5) - 最重要的证据
	containmentScore, err := r.calculateValueContainment(fromTable, fromCol.Name, toTable, toCol.Name)
	if err == nil && containmentScore > 0.3 {
		evidences = append(evidences, graph.Evidence{
			Type:        "value_containment",
			Score:       containmentScore,
			Description: "值集合包含度",
			Details:     fmt.Sprintf("%.1f%% 的值存在于目标表", containmentScore*100),
		})
		totalScore += containmentScore * 0.5
	}
	
	if len(evidences) == 0 {
		return nil
	}
	
	// 降低置信度阈值到 0.3，更容易发现关系
	if totalScore < 0.3 {
		return nil
	}
	
	edge := &graph.Edge{
		ID:         fmt.Sprintf("%s.%s->%s.%s", fromTable, fromCol.Name, toTable, toCol.Name),
		Type:       graph.EdgeTypeInferredFK,
		From:       fmt.Sprintf("%s.%s", fromTable, fromCol.Name),
		To:         fmt.Sprintf("%s.%s", toTable, toCol.Name),
		Confidence: totalScore,
		Evidence:   evidences,
		Properties: map[string]interface{}{
			"from_table":  fromTable,
			"from_column": fromCol.Name,
			"to_table":    toTable,
			"to_column":   toCol.Name,
		},
	}
	
	return edge
}

// calculateNameSimilarity 计算命名相似度
func (r *RelationshipInferer) calculateNameSimilarity(name1, name2 string) float64 {
	// 标准化命名
	n1 := strings.ToLower(strings.TrimPrefix(name1, "c"))
	n2 := strings.ToLower(strings.TrimPrefix(name2, "c"))
	
	// 完全匹配
	if n1 == n2 {
		return 1.0
	}
	
	// 包含关系
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return 0.8
	}
	
	// Levenshtein 距离
	maxLen := math.Max(float64(len(n1)), float64(len(n2)))
	if maxLen == 0 {
		return 0
	}
	
	distance := levenshtein.DistanceForStrings([]rune(n1), []rune(n2), levenshtein.DefaultOptions)
	similarity := 1.0 - float64(distance)/maxLen
	
	if similarity > 0.7 {
		return similarity
	}
	
	return 0
}

// calculateTypeMatch 计算类型匹配度
func (r *RelationshipInferer) calculateTypeMatch(col1, col2 adapter.Column) float64 {
	// 类型必须兼容
	if !r.isTypeCompatible(col1.DataType, col2.DataType) {
		return 0
	}
	
	// 长度匹配
	if col1.Length > 0 && col2.Length > 0 {
		if col1.Length == col2.Length {
			return 1.0
		}
		// 长度接近
		ratio := float64(min(col1.Length, col2.Length)) / float64(max(col1.Length, col2.Length))
		if ratio > 0.8 {
			return 0.8
		}
	}
	
	return 0.6 // 类型兼容但长度不确定
}

// isTypeCompatible 判断类型是否兼容
func (r *RelationshipInferer) isTypeCompatible(type1, type2 string) bool {
	t1 := strings.ToLower(type1)
	t2 := strings.ToLower(type2)
	
	// 完全匹配
	if t1 == t2 {
		return true
	}
	
	// 字符串类型组
	stringTypes := map[string]bool{
		"varchar": true, "nvarchar": true, "char": true, "nchar": true, "text": true,
	}
	if stringTypes[t1] && stringTypes[t2] {
		return true
	}
	
	// 整数类型组
	intTypes := map[string]bool{
		"int": true, "bigint": true, "smallint": true, "tinyint": true,
	}
	if intTypes[t1] && intTypes[t2] {
		return true
	}
	
	return false
}

// calculateValueContainment 计算值包含度
func (r *RelationshipInferer) calculateValueContainment(fromTable, fromCol, toTable, toCol string) (float64, error) {
	// 采样检查
	sampleSize := 1000
	
	// 获取 from 列的样本值
	fromStats, err := r.adapter.SampleColumnStats(fromTable, fromCol, sampleSize)
	if err != nil {
		return 0, err
	}
	
	// 获取 to 列的所有唯一值（假设是主键，数量不会太大）
	toStats, err := r.adapter.SampleColumnStats(toTable, toCol, 10000)
	if err != nil {
		return 0, err
	}
	
	// 构建目标值集合
	toValues := make(map[string]bool)
	for _, v := range toStats.TopValues {
		toValues[v.Value] = true
	}
	
	// 计算包含度
	matchCount := 0
	totalCount := 0
	for _, v := range fromStats.TopValues {
		totalCount += int(v.Count)
		if toValues[v.Value] {
			matchCount += int(v.Count)
		}
	}
	
	if totalCount == 0 {
		return 0, nil
	}
	
	return float64(matchCount) / float64(totalCount), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

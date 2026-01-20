package analyzer

import (
	"fmt"
	"schema-analyzer/internal/adapter"
	"schema-analyzer/internal/ai"
	"schema-analyzer/internal/graph"
	"strings"
)

// HybridAnalyzer 混合分析器（算法 + AI）
type HybridAnalyzer struct {
	adapter   adapter.DBAdapter
	aiClient  ai.Client
	inferer   *RelationshipInferer
}

// NewHybridAnalyzer 创建混合分析器
func NewHybridAnalyzer(adapter adapter.DBAdapter, aiClient ai.Client) *HybridAnalyzer {
	return &HybridAnalyzer{
		adapter:  adapter,
		aiClient: aiClient,
		inferer:  NewRelationshipInferer(adapter),
	}
}

// AnalyzeWithAI 使用 AI 增强的分析
func (h *HybridAnalyzer) AnalyzeWithAI(meta *adapter.SchemaMetadata) (*EnhancedSchema, error) {
	enhanced := &EnhancedSchema{
		Tables: make(map[string]*EnhancedTable),
	}

	// 1. 先做关系推断（算法）
	fmt.Println("🔗 推断表间关系...")
	edges, err := h.inferer.InferRelationships(meta)
	if err != nil {
		return nil, err
	}
	enhanced.Relationships = edges

	// 2. 分类字段：标准字段 vs 自定义字段
	standardFields := []ai.FieldContext{}
	customFields := make(map[string][]string) // table -> custom columns

	for _, table := range meta.Tables {
		enhancedTable := &EnhancedTable{
			Name:    table.Name,
			Columns: make(map[string]*EnhancedColumn),
		}

		for _, col := range table.Columns {
			if isCustomField(col.Name) {
				// 自定义字段：记录下来，稍后基于关系推断
				customFields[table.Name] = append(customFields[table.Name], col.Name)
			} else {
				// 标准字段：加入批量解释队列
				standardFields = append(standardFields, ai.FieldContext{
					TableName:  table.Name,
					ColumnName: col.Name,
					DataType:   col.DataType,
				})
			}

			enhancedTable.Columns[col.Name] = &EnhancedColumn{
				Name:     col.Name,
				DataType: col.DataType,
			}
		}

		enhanced.Tables[table.Name] = enhancedTable
	}

	// 3. 批量解释标准字段（AI）
	if len(standardFields) > 0 {
		fmt.Printf("🤖 AI 解释 %d 个标准字段...\n", len(standardFields))
		explanations, err := h.aiClient.BatchExplain(standardFields)
		if err != nil {
			fmt.Printf("⚠️  AI 解释失败: %v，继续使用算法\n", err)
		} else {
			// 应用 AI 解释
			for _, field := range standardFields {
				if exp, ok := explanations[field.ColumnName]; ok {
					col := enhanced.Tables[field.TableName].Columns[field.ColumnName]
					col.Explanation = exp
				}
			}
		}
	}

	// 4. 推断自定义字段（基于关系 + AI）
	fmt.Printf("🔍 推断 %d 个表的自定义字段...\n", len(customFields))
	for tableName, columns := range customFields {
		for _, colName := range columns {
			explanation := h.inferCustomFieldMeaning(tableName, colName, edges, enhanced)
			col := enhanced.Tables[tableName].Columns[colName]
			col.Explanation = explanation
		}
	}

	return enhanced, nil
}

// inferCustomFieldMeaning 推断自定义字段含义
func (h *HybridAnalyzer) inferCustomFieldMeaning(
	tableName, columnName string,
	edges []*graph.Edge,
	enhanced *EnhancedSchema,
) *ai.FieldExplanation {
	// 1. 查找该字段的关联关系
	relatedFields := h.findRelatedFields(tableName, columnName, edges, enhanced)

	if len(relatedFields) == 0 {
		// 没有关联关系，只能给个默认说明
		return &ai.FieldExplanation{
			ColumnName:      columnName,
			ChineseName:     "自定义字段",
			Description:     "未发现关联关系",
			BusinessMeaning: "需要业务人员确认",
			Confidence:      0.1,
			Source:          "relation",
		}
	}

	// 2. 如果有 AI 客户端，让 AI 基于关联关系推断
	if h.aiClient != nil {
		explanation, err := h.aiClient.InferCustomField(columnName, relatedFields)
		if err == nil {
			return explanation
		}
		fmt.Printf("⚠️  AI 推断失败: %v\n", err)
	}

	// 3. 降级：基于关联关系生成简单说明
	return h.generateRelationBasedExplanation(columnName, relatedFields)
}

// findRelatedFields 查找关联字段
func (h *HybridAnalyzer) findRelatedFields(
	tableName, columnName string,
	edges []*graph.Edge,
	enhanced *EnhancedSchema,
) []ai.RelatedField {
	var related []ai.RelatedField

	for _, edge := range edges {
		props := edge.Properties
		fromTable := props["from_table"].(string)
		fromCol := props["from_column"].(string)
		toTable := props["to_table"].(string)
		toCol := props["to_column"].(string)

		// 如果这个自定义字段参与了关系
		if fromTable == tableName && fromCol == columnName {
			// 查找目标字段的解释
			if targetTable, ok := enhanced.Tables[toTable]; ok {
				if targetCol, ok := targetTable.Columns[toCol]; ok && targetCol.Explanation != nil {
					related = append(related, ai.RelatedField{
						TableName:   toTable,
						ColumnName:  toCol,
						ChineseName: targetCol.Explanation.ChineseName,
						Relation:    string(edge.Type),
						Confidence:  edge.Confidence,
					})
				}
			}
		}
	}

	return related
}

// generateRelationBasedExplanation 基于关系生成说明（降级方案）
func (h *HybridAnalyzer) generateRelationBasedExplanation(
	columnName string,
	relatedFields []ai.RelatedField,
) *ai.FieldExplanation {
	if len(relatedFields) == 0 {
		return &ai.FieldExplanation{
			ColumnName:  columnName,
			ChineseName: "自定义字段",
			Description: "未发现关联",
			Confidence:  0.1,
			Source:      "relation",
		}
	}

	// 使用置信度最高的关联
	best := relatedFields[0]
	for _, rf := range relatedFields {
		if rf.Confidence > best.Confidence {
			best = rf
		}
	}

	return &ai.FieldExplanation{
		ColumnName:      columnName,
		ChineseName:     fmt.Sprintf("关联%s", best.ChineseName),
		Description:     fmt.Sprintf("与 %s.%s 关联", best.TableName, best.ColumnName),
		BusinessMeaning: fmt.Sprintf("基于关联推断：可能是%s相关字段", best.ChineseName),
		Confidence:      best.Confidence * 0.7, // 降低置信度
		Source:          "relation",
	}
}

// isCustomField 判断是否为自定义字段
func isCustomField(columnName string) bool {
	lower := strings.ToLower(columnName)
	
	// cFree1-10
	if strings.HasPrefix(lower, "cfree") {
		return true
	}
	
	// cDefine1-37
	if strings.HasPrefix(lower, "cdefine") {
		return true
	}
	
	// ufts (用友自定义时间戳)
	if lower == "ufts" {
		return true
	}
	
	return false
}

// EnhancedSchema 增强的 Schema（包含 AI 解释）
type EnhancedSchema struct {
	Tables        map[string]*EnhancedTable
	Relationships []*graph.Edge
}

// EnhancedTable 增强的表
type EnhancedTable struct {
	Name    string
	Columns map[string]*EnhancedColumn
}

// EnhancedColumn 增强的列
type EnhancedColumn struct {
	Name        string
	DataType    string
	Explanation *ai.FieldExplanation
}

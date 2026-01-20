package analyzer

import (
	"schema-analyzer/internal/adapter"
	"strings"
)

// EnumDetector 枚举/码表检测器
type EnumDetector struct {
	adapter adapter.DBAdapter
}

// NewEnumDetector 创建检测器
func NewEnumDetector(adapter adapter.DBAdapter) *EnumDetector {
	return &EnumDetector{adapter: adapter}
}

// EnumTable 枚举表
type EnumTable struct {
	Name          string
	RowCount      int64
	KeyColumn     string
	ValueColumn   string
	Confidence    float64
	ReferencedBy  []string // 被哪些表引用
}

// DetectEnumTables 检测枚举表
func (e *EnumDetector) DetectEnumTables(meta *adapter.SchemaMetadata) ([]EnumTable, error) {
	var enumTables []EnumTable
	
	for _, table := range meta.Tables {
		// 估算行数
		rowCount, err := e.adapter.EstimateRowCount(table.Name)
		if err != nil {
			continue
		}
		
		// 枚举表特征：行数少（< 1000）
		if rowCount > 1000 {
			continue
		}
		
		// 检查列结构：需要有 code/id 和 name/label 这样的组合
		keyCol, valueCol := e.findEnumColumns(table.Columns)
		if keyCol == "" {
			continue
		}
		
		confidence := e.calculateEnumConfidence(table, rowCount, keyCol, valueCol)
		if confidence > 0.6 {
			enumTables = append(enumTables, EnumTable{
				Name:        table.Name,
				RowCount:    rowCount,
				KeyColumn:   keyCol,
				ValueColumn: valueCol,
				Confidence:  confidence,
			})
		}
	}
	
	return enumTables, nil
}

// findEnumColumns 查找枚举列
func (e *EnumDetector) findEnumColumns(columns []adapter.Column) (keyCol, valueCol string) {
	keyPatterns := []string{"code", "id", "key", "type"}
	valuePatterns := []string{"name", "label", "desc", "description", "value"}
	
	for _, col := range columns {
		colLower := toLower(col.Name)
		
		// 查找 key 列
		if keyCol == "" {
			for _, pattern := range keyPatterns {
				if contains(colLower, pattern) {
					keyCol = col.Name
					break
				}
			}
		}
		
		// 查找 value 列
		if valueCol == "" {
			for _, pattern := range valuePatterns {
				if contains(colLower, pattern) {
					valueCol = col.Name
					break
				}
			}
		}
	}
	
	return
}

// calculateEnumConfidence 计算枚举表置信度
func (e *EnumDetector) calculateEnumConfidence(table adapter.Table, rowCount int64, keyCol, valueCol string) float64 {
	score := 0.0
	
	// 行数少加分
	if rowCount < 100 {
		score += 0.4
	} else if rowCount < 500 {
		score += 0.3
	} else {
		score += 0.2
	}
	
	// 有 key 和 value 列加分
	if keyCol != "" && valueCol != "" {
		score += 0.4
	} else if keyCol != "" {
		score += 0.2
	}
	
	// 列数少加分（典型枚举表列数 2-5）
	if len(table.Columns) <= 5 {
		score += 0.2
	}
	
	return score
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

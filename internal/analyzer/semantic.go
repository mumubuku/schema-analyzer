package analyzer

import (
	"fmt"
	"schema-analyzer/internal/adapter"
	"strings"
)

// SemanticAnalyzer 字段语义分析器（AI 增强）
type SemanticAnalyzer struct {
	adapter adapter.DBAdapter
	aiClient AIClient // 可选的 AI 客户端
}

// AIClient AI 服务接口
type AIClient interface {
	// InferColumnSemantic 推断字段语义
	InferColumnSemantic(columnName string, stats *adapter.ColumnStats) (string, error)
	
	// SuggestComment 建议字段注释
	SuggestComment(columnName, dataType string, stats *adapter.ColumnStats) (string, error)
}

// NewSemanticAnalyzer 创建分析器
func NewSemanticAnalyzer(adapter adapter.DBAdapter, aiClient AIClient) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		adapter: adapter,
		aiClient: aiClient,
	}
}

// AnalyzeColumn 分析列语义
func (s *SemanticAnalyzer) AnalyzeColumn(table, column string) (*ColumnSemantic, error) {
	// 1. 先获取统计特征（证据）
	stats, err := s.adapter.SampleColumnStats(table, column, 1000)
	if err != nil {
		return nil, err
	}
	
	semantic := &ColumnSemantic{
		Table:  table,
		Column: column,
		Stats:  buildStatsSummary(stats),
	}
	
	// 2. 如果有 AI 客户端，让 AI 基于证据生成解释
	if s.aiClient != nil {
		// AI 只负责"组织语言"，不负责"拍脑袋"
		comment, err := s.aiClient.SuggestComment(column, "", stats)
		if err == nil {
			semantic.AIComment = comment
		}
		
		semanticType, err := s.aiClient.InferColumnSemantic(column, stats)
		if err == nil {
			semantic.SemanticType = semanticType
		}
	}
	
	return semantic, nil
}

// ColumnSemantic 列语义信息
type ColumnSemantic struct {
	Table        string            `json:"table"`
	Column       string            `json:"column"`
	Stats        map[string]interface{} `json:"stats"`        // 统计特征（证据）
	SemanticType string            `json:"semantic_type"` // AI 推断的语义类型
	AIComment    string            `json:"ai_comment"`    // AI 建议的注释
}

// buildStatsSummary 构建统计摘要（给 AI 的输入）
func buildStatsSummary(stats *adapter.ColumnStats) map[string]interface{} {
	summary := make(map[string]interface{})
	
	if stats.TotalRows > 0 {
		summary["null_ratio"] = float64(stats.NullCount) / float64(stats.TotalRows)
		summary["distinct_ratio"] = float64(stats.DistinctCount) / float64(stats.TotalRows)
	}
	
	// Top 值（脱敏：只给数量和模式，不给实际值）
	if len(stats.TopValues) > 0 {
		topPatterns := make([]map[string]interface{}, 0)
		for _, v := range stats.TopValues {
			topPatterns = append(topPatterns, map[string]interface{}{
				"length":  len(v.Value),
				"count":   v.Count,
				"pattern": detectPattern(v.Value), // 例如：数字/日期/邮箱
			})
		}
		summary["top_patterns"] = topPatterns
	}
	
	return summary
}

// detectPattern 检测值的模式（不暴露实际值）
func detectPattern(value string) string {
	// 简单的模式识别
	if len(value) == 0 {
		return "empty"
	}
	
	// 数字
	if isNumeric(value) {
		return "numeric"
	}
	
	// 日期格式
	if isDateLike(value) {
		return "date"
	}
	
	// 邮箱
	if isEmailLike(value) {
		return "email"
	}
	
	return "text"
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isDateLike(s string) bool {
	// 简单判断：包含 - 或 /
	return len(s) >= 8 && (stringContains(s, "-") || stringContains(s, "/"))
}

func isEmailLike(s string) bool {
	return stringContains(s, "@") && stringContains(s, ".")
}

func stringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// MockAIClient 示例：调用 OpenAI/Claude 等
type MockAIClient struct {
	apiKey string
	model  string
}

func (m *MockAIClient) SuggestComment(columnName, dataType string, stats *adapter.ColumnStats) (string, error) {
	// 构建 prompt（只给统计摘要，不给实际数据）
	_ = fmt.Sprintf(`
基于以下字段特征，建议一个简洁的中文注释：

字段名: %s
数据类型: %s
Null 比例: %.2f%%
唯一值比例: %.2f%%
样本数: %d

要求：
1. 只输出注释文本，不要解释
2. 不要超过 20 字
3. 基于统计特征推断，不要猜测
`, 
		columnName,
		dataType,
		float64(stats.NullCount)/float64(stats.TotalRows)*100,
		float64(stats.DistinctCount)/float64(stats.TotalRows)*100,
		stats.TotalRows,
	)
	
	// 调用 AI API（这里只是示例）
	// response := callAIAPI(prompt)
	
	// 示例返回
	return "用户部门编码", nil
}

func (m *MockAIClient) InferColumnSemantic(columnName string, stats *adapter.ColumnStats) (string, error) {
	// 基于统计特征推断语义类型
	summary := buildStatsSummary(stats)
	
	distinctRatio := summary["distinct_ratio"].(float64)
	
	// 简单规则（实际可以用 AI）
	if distinctRatio > 0.95 {
		return "identifier", nil // 主键/唯一标识
	} else if distinctRatio < 0.1 {
		return "category", nil // 分类/枚举
	} else {
		return "attribute", nil // 普通属性
	}
}

// 使用示例（在 main.go 中）：
/*
// 可选：启用 AI 增强
var aiClient analyzer.AIClient
if enableAI {
	aiClient = &analyzer.MockAIClient{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		model:  "gpt-4",
	}
}

semanticAnalyzer := analyzer.NewSemanticAnalyzer(dbAdapter, aiClient)

for _, table := range meta.Tables {
	for _, col := range table.Columns {
		semantic, _ := semanticAnalyzer.AnalyzeColumn(table.Name, col.Name)
		// 将 AI 注释添加到输出
		if semantic.AIComment != "" {
			fmt.Printf("  %s: %s\n", col.Name, semantic.AIComment)
		}
	}
}
*/

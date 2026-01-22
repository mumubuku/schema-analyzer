package renderer

import (
	"fmt"
	"schema-analyzer/internal/graph"
	"strings"
)

// EnhancedMarkdownRenderer 增强的 Markdown 渲染器（包含 AI 解释）
type EnhancedMarkdownRenderer struct{}

// NewEnhancedMarkdownRenderer 创建渲染器
func NewEnhancedMarkdownRenderer() *EnhancedMarkdownRenderer {
	return &EnhancedMarkdownRenderer{}
}

// Render 渲染为 Markdown 格式（包含 AI 解释）
func (m *EnhancedMarkdownRenderer) Render(g *graph.SchemaGraph) string {
	var sb strings.Builder
	
	sb.WriteString("# 数据库结构文档（AI 增强版）\n\n")
	sb.WriteString("## 表结构\n\n")
	
	// 按表组织列信息
	tables := make(map[string][]*graph.Node)
	for _, node := range g.Nodes {
		if node.Type == graph.NodeTypeColumn {
			tableName := node.Properties["table"].(string)
			tables[tableName] = append(tables[tableName], node)
		}
	}
	
	// 输出每个表
	for tableName, columns := range tables {
		sb.WriteString(fmt.Sprintf("### %s\n\n", tableName))
		
		// 检查是否有 AI 解释
		hasAI := false
		for _, col := range columns {
			if _, ok := col.Properties["ai_chinese_name"]; ok {
				hasAI = true
				break
			}
		}
		
		// 表头
		if hasAI {
			sb.WriteString("| 列名 | 中文名 | 类型 | 可空 | 主键 | 业务含义 | 来源 | 置信度 |\n")
			sb.WriteString("|------|--------|------|------|------|----------|------|--------|\n")
		} else {
			sb.WriteString("| 列名 | 类型 | 长度 | 可空 | 主键 | Null率 | 唯一值率 |\n")
			sb.WriteString("|------|------|------|------|------|--------|----------|\n")
		}
		
		// 列信息
		for _, col := range columns {
			props := col.Properties
			
			if hasAI && props["ai_chinese_name"] != nil {
				// AI 增强版
				nullable := "否"
				if props["nullable"].(bool) {
					nullable = "是"
				}
				pk := ""
				if props["is_primary_key"].(bool) {
					pk = "✓"
				}
				
				chineseName := props["ai_chinese_name"].(string)
				businessMeaning := props["ai_business_meaning"].(string)
				source := props["ai_source"].(string)
				confidence := props["ai_confidence"].(float64)
				
				// 来源标记
				sourceLabel := ""
				switch source {
				case "ai_standard":
					sourceLabel = "🤖标准"
				case "ai_inferred":
					sourceLabel = "🔍推断"
				case "relation":
					sourceLabel = "🔗关联"
				}
				
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %.0f%% |\n",
					col.Name,
					chineseName,
					props["data_type"].(string),
					nullable,
					pk,
					businessMeaning,
					sourceLabel,
					confidence*100,
				))
			} else {
				// 标准版
				nullable := "否"
				if props["nullable"].(bool) {
					nullable = "是"
				}
				pk := ""
				if props["is_primary_key"].(bool) {
					pk = "✓"
				}
				
				nullRatio := props["null_ratio"].(float64)
				distinctRate := props["distinct_rate"].(float64)
				
				sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s | %.1f%% | %.1f%% |\n",
					col.Name,
					props["data_type"].(string),
					props["length"].(int),
					nullable,
					pk,
					nullRatio*100,
					distinctRate*100,
				))
			}
		}
		
		sb.WriteString("\n")
		
		// 输出该表的关系
		m.renderTableRelations(&sb, g, tableName)
	}
	
	// 添加图例说明
	if hasAnyAI(g) {
		sb.WriteString("\n## 图例说明\n\n")
		sb.WriteString("- 🤖标准：AI 直接识别的 U8 标准字段\n")
		sb.WriteString("- 🔍推断：AI 基于关联关系推断的自定义字段\n")
		sb.WriteString("- 🔗关联：仅基于关系推断的字段\n")
		sb.WriteString("- 置信度：AI 对解释的确定程度（0-100%）\n")
	}
	
	return sb.String()
}

// renderTableRelations 渲染表关系
func (m *EnhancedMarkdownRenderer) renderTableRelations(sb *strings.Builder, g *graph.SchemaGraph, tableName string) {
	var relations []*graph.Edge
	
	for _, edge := range g.Edges {
		props := edge.Properties
		if props["from_table"].(string) == tableName || props["to_table"].(string) == tableName {
			relations = append(relations, edge)
		}
	}
	
	if len(relations) == 0 {
		return
	}
	
	sb.WriteString("#### 关系\n\n")
	
	for _, rel := range relations {
		props := rel.Properties
		
		// 检查是否是 AI 推断的表关系（只有表级别的关系）
		if _, hasFromCol := props["from_column"]; !hasFromCol {
			// AI 推断的表关系
			fromTable := props["from_table"].(string)
			toTable := props["to_table"].(string)
			relType := props["relation_type"].(string)
			description := props["description"].(string)
			
			sb.WriteString(fmt.Sprintf("- **%s** `%s` → `%s` (置信度: %.2f)\n",
				relType, fromTable, toTable, rel.Confidence))
			sb.WriteString(fmt.Sprintf("  - 描述: %s\n", description))
		} else {
			// 传统的列级别关系
			fromTable := props["from_table"].(string)
			fromCol := props["from_column"].(string)
			toTable := props["to_table"].(string)
			toCol := props["to_column"].(string)
			
			relType := "外键"
			if rel.Type == graph.EdgeTypeInferredFK {
				relType = "推断外键"
			}
			
			sb.WriteString(fmt.Sprintf("- **%s** `%s.%s` → `%s.%s` (置信度: %.2f)\n",
				relType, fromTable, fromCol, toTable, toCol, rel.Confidence))
			
			// 输出证据
			if len(rel.Evidence) > 0 {
				sb.WriteString("  - 证据:\n")
				for _, ev := range rel.Evidence {
					sb.WriteString(fmt.Sprintf("    - %s (%.2f): %s\n", 
						ev.Description, ev.Score, ev.Details))
				}
			}
		}
	}
	
	sb.WriteString("\n")
}

// hasAnyAI 检查是否有任何 AI 解释
func hasAnyAI(g *graph.SchemaGraph) bool {
	for _, node := range g.Nodes {
		if node.Type == graph.NodeTypeColumn {
			if _, ok := node.Properties["ai_chinese_name"]; ok {
				return true
			}
		}
	}
	return false
}

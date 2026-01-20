package renderer

import (
	"fmt"
	"schema-analyzer/internal/graph"
	"strings"
)

// MarkdownRenderer Markdown 数据字典渲染器
type MarkdownRenderer struct{}

// NewMarkdownRenderer 创建渲染器
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render 渲染为 Markdown 格式
func (m *MarkdownRenderer) Render(g *graph.SchemaGraph) string {
	var sb strings.Builder
	
	sb.WriteString("# 数据库结构文档\n\n")
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
		
		// 表头
		sb.WriteString("| 列名 | 类型 | 长度 | 可空 | 主键 | Null率 | 唯一值率 |\n")
		sb.WriteString("|------|------|------|------|------|--------|----------|\n")
		
		// 列信息
		for _, col := range columns {
			props := col.Properties
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
		
		sb.WriteString("\n")
		
		// 输出该表的关系
		m.renderTableRelations(&sb, g, tableName)
	}
	
	return sb.String()
}

// renderTableRelations 渲染表关系
func (m *MarkdownRenderer) renderTableRelations(sb *strings.Builder, g *graph.SchemaGraph, tableName string) {
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
	
	sb.WriteString("\n")
}

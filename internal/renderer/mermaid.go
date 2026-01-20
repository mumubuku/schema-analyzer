package renderer

import (
	"fmt"
	"schema-analyzer/internal/graph"
	"strings"
)

// MermaidRenderer Mermaid ER 图渲染器
type MermaidRenderer struct{}

// NewMermaidRenderer 创建渲染器
func NewMermaidRenderer() *MermaidRenderer {
	return &MermaidRenderer{}
}

// Render 渲染为 Mermaid 格式
func (m *MermaidRenderer) Render(g *graph.SchemaGraph) string {
	var sb strings.Builder
	
	sb.WriteString("erDiagram\n")
	
	// 渲染表节点
	tables := make(map[string][]string)
	for _, node := range g.Nodes {
		if node.Type == graph.NodeTypeColumn {
			props := node.Properties
			tableName := props["table"].(string)
			
			dataType := props["data_type"].(string)
			nullable := ""
			if props["nullable"].(bool) {
				nullable = " NULL"
			}
			pk := ""
			if props["is_primary_key"].(bool) {
				pk = " PK"
			}
			
			colDef := fmt.Sprintf("        %s %s%s%s", node.Name, dataType, pk, nullable)
			tables[tableName] = append(tables[tableName], colDef)
		}
	}
	
	// 输出表定义
	for tableName, columns := range tables {
		sb.WriteString(fmt.Sprintf("    %s {\n", tableName))
		for _, col := range columns {
			sb.WriteString(col + "\n")
		}
		sb.WriteString("    }\n")
	}
	
	sb.WriteString("\n")
	
	// 渲染关系
	for _, edge := range g.Edges {
		if edge.Type == graph.EdgeTypeFK || edge.Type == graph.EdgeTypeInferredFK {
			props := edge.Properties
			fromTable := props["from_table"].(string)
			toTable := props["to_table"].(string)
			
			// 关系类型
			relType := "||--o{"
			if edge.Type == graph.EdgeTypeInferredFK {
				relType = "||..o{" // 虚线表示推断关系
			}
			
			label := fmt.Sprintf("\"%.2f\"", edge.Confidence)
			sb.WriteString(fmt.Sprintf("    %s %s %s : %s\n", 
				toTable, relType, fromTable, label))
		}
	}
	
	return sb.String()
}

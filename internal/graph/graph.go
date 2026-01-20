package graph

import (
	"encoding/json"
	"sync"
)

// SchemaGraph 数据库结构图
type SchemaGraph struct {
	mu    sync.RWMutex
	Nodes map[string]*Node `json:"nodes"`
	Edges map[string]*Edge `json:"edges"`
}

// NewSchemaGraph 创建新图
func NewSchemaGraph() *SchemaGraph {
	return &SchemaGraph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string]*Edge),
	}
}

// AddNode 添加节点
func (g *SchemaGraph) AddNode(node *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Nodes[node.ID] = node
}

// AddEdge 添加边
func (g *SchemaGraph) AddEdge(edge *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Edges[edge.ID] = edge
}

// GetNode 获取节点
func (g *SchemaGraph) GetNode(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Nodes[id]
}

// ToJSON 导出为JSON
func (g *SchemaGraph) ToJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return json.MarshalIndent(g, "", "  ")
}

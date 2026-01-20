package graph

// NodeType 节点类型
type NodeType string

const (
	NodeTypeTable  NodeType = "table"
	NodeTypeColumn NodeType = "column"
	NodeTypeIndex  NodeType = "index"
	NodeTypeView   NodeType = "view"
)

// Node 图节点
type Node struct {
	ID         string                 `json:"id"`
	Type       NodeType               `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}

// TableNode 表节点属性
type TableNode struct {
	Schema   string `json:"schema"`
	RowCount int64  `json:"row_count"`
}

// ColumnNode 列节点属性
type ColumnNode struct {
	Table        string  `json:"table"`
	DataType     string  `json:"data_type"`
	Length       int     `json:"length"`
	Nullable     bool    `json:"nullable"`
	IsPrimaryKey bool    `json:"is_primary_key"`
	NullRatio    float64 `json:"null_ratio"`
	DistinctRate float64 `json:"distinct_rate"`
	TopValues    []Value `json:"top_values"`
}

// Value 值统计
type Value struct {
	Value string  `json:"value"`
	Count int64   `json:"count"`
	Ratio float64 `json:"ratio"`
}

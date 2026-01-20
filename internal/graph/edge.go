package graph

// EdgeType 边类型
type EdgeType string

const (
	EdgeTypeFK         EdgeType = "foreign_key"      // 真外键
	EdgeTypeInferredFK EdgeType = "inferred_fk"      // 推断外键
	EdgeTypeDependency EdgeType = "dependency"       // 依赖关系
	EdgeTypeEnum       EdgeType = "enum_reference"   // 枚举表引用
)

// Edge 图的边
type Edge struct {
	ID         string                 `json:"id"`
	Type       EdgeType               `json:"type"`
	From       string                 `json:"from"` // 节点ID
	To         string                 `json:"to"`   // 节点ID
	Confidence float64                `json:"confidence"` // 置信度 0-1
	Evidence   []Evidence             `json:"evidence"`
	Properties map[string]interface{} `json:"properties"`
}

// Evidence 证据
type Evidence struct {
	Type        string  `json:"type"`        // naming/value_containment/type_match
	Score       float64 `json:"score"`       // 0-1
	Description string  `json:"description"`
	Details     string  `json:"details"`
}

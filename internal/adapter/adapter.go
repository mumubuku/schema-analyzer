package adapter

import "database/sql"

// DBAdapter 数据库适配器接口
type DBAdapter interface {
	// IntrospectSchema 获取元数据
	IntrospectSchema() (*SchemaMetadata, error)
	
	// EstimateRowCount 估算行数
	EstimateRowCount(table string) (int64, error)
	
	// SampleColumnStats 采样列统计
	SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error)
	
	// GetPrimaryKeys 获取主键
	GetPrimaryKeys(table string) ([]string, error)
	
	// GetForeignKeys 获取外键约束
	GetForeignKeys() ([]ForeignKey, error)
	
	// Close 关闭连接
	Close() error
}

// SchemaMetadata 元数据
type SchemaMetadata struct {
	Tables  []Table
	Indexes []Index
}

// Table 表信息
type Table struct {
	Schema  string
	Name    string
	Columns []Column
}

// Column 列信息
type Column struct {
	Name         string
	DataType     string
	Length       int
	Nullable     bool
	IsPrimaryKey bool
	DefaultValue sql.NullString
}

// Index 索引信息
type Index struct {
	Table   string
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKey 外键
type ForeignKey struct {
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// ColumnStats 列统计
type ColumnStats struct {
	TotalRows    int64
	NullCount    int64
	DistinctCount int64
	TopValues    []ValueCount
	MinValue     sql.NullString
	MaxValue     sql.NullString
}

// ValueCount 值计数
type ValueCount struct {
	Value string
	Count int64
}

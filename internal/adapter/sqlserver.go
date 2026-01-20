package adapter

import (
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
)

// SQLServerAdapter SQL Server 适配器
type SQLServerAdapter struct {
	db *sql.DB
}

// NewSQLServerAdapter 创建 SQL Server 适配器
func NewSQLServerAdapter(connStr string) (*SQLServerAdapter, error) {
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SQLServerAdapter{db: db}, nil
}

// IntrospectSchema 获取元数据
func (a *SQLServerAdapter) IntrospectSchema() (*SchemaMetadata, error) {
	meta := &SchemaMetadata{}
	
	// 获取表列表
	tables, err := a.getTables()
	if err != nil {
		return nil, err
	}
	
	// 获取每个表的列信息
	for i := range tables {
		columns, err := a.getColumns(tables[i].Schema, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Columns = columns
	}
	
	meta.Tables = tables
	
	// 获取索引
	indexes, err := a.getIndexes()
	if err != nil {
		return nil, err
	}
	meta.Indexes = indexes
	
	return meta, nil
}

func (a *SQLServerAdapter) getTables() ([]Table, error) {
	query := `
		SELECT TABLE_SCHEMA, TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_SCHEMA, TABLE_NAME
	`
	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []Table
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.Schema, &t.Name); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (a *SQLServerAdapter) getColumns(schema, table string) ([]Column, error) {
	query := `
		SELECT 
			c.COLUMN_NAME,
			c.DATA_TYPE,
			COALESCE(c.CHARACTER_MAXIMUM_LENGTH, 0) as LENGTH,
			CASE WHEN c.IS_NULLABLE = 'YES' THEN 1 ELSE 0 END as NULLABLE,
			CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END as IS_PK
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN (
			SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
				ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		) pk ON c.TABLE_SCHEMA = pk.TABLE_SCHEMA 
			AND c.TABLE_NAME = pk.TABLE_NAME 
			AND c.COLUMN_NAME = pk.COLUMN_NAME
		WHERE c.TABLE_SCHEMA = @p1 AND c.TABLE_NAME = @p2
		ORDER BY c.ORDINAL_POSITION
	`
	rows, err := a.db.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var columns []Column
	for rows.Next() {
		var c Column
		var nullable, isPK int
		if err := rows.Scan(&c.Name, &c.DataType, &c.Length, &nullable, &isPK); err != nil {
			return nil, err
		}
		c.Nullable = nullable == 1
		c.IsPrimaryKey = isPK == 1
		columns = append(columns, c)
	}
	return columns, nil
}

func (a *SQLServerAdapter) getIndexes() ([]Index, error) {
	query := `
		SELECT 
			t.name as TABLE_NAME,
			i.name as INDEX_NAME,
			c.name as COLUMN_NAME,
			i.is_unique
		FROM sys.indexes i
		JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		JOIN sys.tables t ON i.object_id = t.object_id
		WHERE i.is_primary_key = 0
		ORDER BY t.name, i.name, ic.key_ordinal
	`
	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	indexMap := make(map[string]*Index)
	for rows.Next() {
		var tableName, indexName, columnName string
		var isUnique bool
		if err := rows.Scan(&tableName, &indexName, &columnName, &isUnique); err != nil {
			return nil, err
		}
		
		key := tableName + "." + indexName
		if idx, exists := indexMap[key]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[key] = &Index{
				Table:   tableName,
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  isUnique,
			}
		}
	}
	
	var indexes []Index
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}
	return indexes, nil
}

// EstimateRowCount 估算行数
func (a *SQLServerAdapter) EstimateRowCount(table string) (int64, error) {
	query := `
		SELECT SUM(p.rows) 
		FROM sys.partitions p
		JOIN sys.tables t ON p.object_id = t.object_id
		WHERE t.name = @p1 AND p.index_id IN (0,1)
	`
	var count sql.NullInt64
	err := a.db.QueryRow(query, table).Scan(&count)
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return count.Int64, nil
}

// SampleColumnStats 采样列统计
func (a *SQLServerAdapter) SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error) {
	stats := &ColumnStats{}
	
	// 总行数和NULL计数
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN [%s] IS NULL THEN 1 ELSE 0 END) as nulls,
			COUNT(DISTINCT [%s]) as distincts
		FROM [%s] TABLESAMPLE (%d ROWS)
	`, column, column, table, sampleSize)
	
	err := a.db.QueryRow(query).Scan(&stats.TotalRows, &stats.NullCount, &stats.DistinctCount)
	if err != nil {
		return nil, err
	}
	
	// TopN值
	topQuery := fmt.Sprintf(`
		SELECT TOP 10 [%s], COUNT(*) as cnt
		FROM [%s] TABLESAMPLE (%d ROWS)
		WHERE [%s] IS NOT NULL
		GROUP BY [%s]
		ORDER BY cnt DESC
	`, column, table, sampleSize, column, column)
	
	rows, err := a.db.Query(topQuery)
	if err != nil {
		return stats, nil // 不影响主流程
	}
	defer rows.Close()
	
	for rows.Next() {
		var vc ValueCount
		if err := rows.Scan(&vc.Value, &vc.Count); err != nil {
			continue
		}
		stats.TopValues = append(stats.TopValues, vc)
	}
	
	return stats, nil
}

// GetPrimaryKeys 获取主键
func (a *SQLServerAdapter) GetPrimaryKeys(table string) ([]string, error) {
	query := `
		SELECT c.name
		FROM sys.indexes i
		JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		JOIN sys.tables t ON i.object_id = t.object_id
		WHERE t.name = @p1 AND i.is_primary_key = 1
		ORDER BY ic.key_ordinal
	`
	rows, err := a.db.Query(query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// GetForeignKeys 获取外键约束
func (a *SQLServerAdapter) GetForeignKeys() ([]ForeignKey, error) {
	query := `
		SELECT 
			OBJECT_NAME(fk.parent_object_id) as from_table,
			COL_NAME(fkc.parent_object_id, fkc.parent_column_id) as from_column,
			OBJECT_NAME(fk.referenced_object_id) as to_table,
			COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) as to_column
		FROM sys.foreign_keys fk
		JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
	`
	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.FromTable, &fk.FromColumn, &fk.ToTable, &fk.ToColumn); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, nil
}

// Close 关闭连接
func (a *SQLServerAdapter) Close() error {
	return a.db.Close()
}

package adapter

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLAdapter MySQL 适配器
type MySQLAdapter struct {
	db     *sql.DB
	schema string
}

// NewMySQLAdapter 创建 MySQL 适配器
func NewMySQLAdapter(connStr, schema string) (*MySQLAdapter, error) {
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLAdapter{db: db, schema: schema}, nil
}

// IntrospectSchema 获取元数据
func (a *MySQLAdapter) IntrospectSchema() (*SchemaMetadata, error) {
	meta := &SchemaMetadata{}
	
	tables, err := a.getTables()
	if err != nil {
		return nil, err
	}
	
	for i := range tables {
		columns, err := a.getColumns(tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Columns = columns
	}
	
	meta.Tables = tables
	
	indexes, err := a.getIndexes()
	if err != nil {
		return nil, err
	}
	meta.Indexes = indexes
	
	return meta, nil
}

func (a *MySQLAdapter) getTables() ([]Table, error) {
	query := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`
	rows, err := a.db.Query(query, a.schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []Table
	for rows.Next() {
		var t Table
		t.Schema = a.schema
		if err := rows.Scan(&t.Name); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (a *MySQLAdapter) getColumns(table string) ([]Column, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			COALESCE(CHARACTER_MAXIMUM_LENGTH, 0),
			IS_NULLABLE = 'YES',
			COLUMN_KEY = 'PRI'
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := a.db.Query(query, a.schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var columns []Column
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.Name, &c.DataType, &c.Length, &c.Nullable, &c.IsPrimaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, c)
	}
	return columns, nil
}

func (a *MySQLAdapter) getIndexes() ([]Index, error) {
	query := `
		SELECT 
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE = 0
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = ? AND INDEX_NAME != 'PRIMARY'
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX
	`
	rows, err := a.db.Query(query, a.schema)
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
func (a *MySQLAdapter) EstimateRowCount(table string) (int64, error) {
	query := `
		SELECT TABLE_ROWS
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	`
	var count sql.NullInt64
	err := a.db.QueryRow(query, a.schema, table).Scan(&count)
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return count.Int64, nil
}

// SampleColumnStats 采样列统计
func (a *MySQLAdapter) SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error) {
	stats := &ColumnStats{}
	
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) as nulls,
			COUNT(DISTINCT %s) as distincts
		FROM %s
		ORDER BY RAND()
		LIMIT %d
	`, column, column, table, sampleSize)
	
	err := a.db.QueryRow(query).Scan(&stats.TotalRows, &stats.NullCount, &stats.DistinctCount)
	if err != nil {
		return nil, err
	}
	
	topQuery := fmt.Sprintf(`
		SELECT %s, COUNT(*) as cnt
		FROM (SELECT %s FROM %s ORDER BY RAND() LIMIT %d) sample
		WHERE %s IS NOT NULL
		GROUP BY %s
		ORDER BY cnt DESC
		LIMIT 10
	`, column, column, table, sampleSize, column, column)
	
	rows, err := a.db.Query(topQuery)
	if err != nil {
		return stats, nil
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
func (a *MySQLAdapter) GetPrimaryKeys(table string) ([]string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION
	`
	rows, err := a.db.Query(query, a.schema, table)
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
func (a *MySQLAdapter) GetForeignKeys() ([]ForeignKey, error) {
	query := `
		SELECT 
			kcu.TABLE_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		WHERE kcu.TABLE_SCHEMA = ? 
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
	`
	rows, err := a.db.Query(query, a.schema)
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
func (a *MySQLAdapter) Close() error {
	return a.db.Close()
}

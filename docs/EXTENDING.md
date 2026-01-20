# 扩展指南

## 添加新数据库支持

### 1. 创建适配器文件

在 `internal/adapter/` 创建新文件，例如 `postgres.go`：

```go
package adapter

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type PostgresAdapter struct {
	db     *sql.DB
	schema string
}

func NewPostgresAdapter(connStr, schema string) (*PostgresAdapter, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresAdapter{db: db, schema: schema}, nil
}
```

### 2. 实现接口方法

```go
func (a *PostgresAdapter) IntrospectSchema() (*SchemaMetadata, error) {
	// 使用 pg_catalog 查询表和列
	query := `
		SELECT 
			table_schema,
			table_name
		FROM information_schema.tables
		WHERE table_schema = $1 
			AND table_type = 'BASE TABLE'
	`
	// ... 实现逻辑
}

func (a *PostgresAdapter) SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error) {
	// PostgreSQL 使用 TABLESAMPLE SYSTEM
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT %s) as distincts
		FROM %s TABLESAMPLE SYSTEM (10)
	`, column, table)
	// ... 实现逻辑
}
```

### 3. 注册到 CLI

在 `cmd/analyzer/main.go` 添加：

```go
case "postgres":
	if schema == "" {
		log.Fatal("PostgreSQL 需要指定 --schema 参数")
	}
	dbAdapter, err = adapter.NewPostgresAdapter(connStr, schema)
```

### 4. 更新文档

在 README.md 添加使用示例：

```bash
./schema-analyzer scan \
  --type postgres \
  --conn "host=localhost user=postgres password=pass dbname=mydb" \
  --schema public \
  --output ./output
```

## 添加新分析器

### 1. 创建分析器文件

在 `internal/analyzer/` 创建新文件，例如 `dependency.go`：

```go
package analyzer

import (
	"schema-analyzer/internal/adapter"
	"schema-analyzer/internal/graph"
)

type DependencyAnalyzer struct {
	adapter adapter.DBAdapter
}

func NewDependencyAnalyzer(adapter adapter.DBAdapter) *DependencyAnalyzer {
	return &DependencyAnalyzer{adapter: adapter}
}

func (d *DependencyAnalyzer) AnalyzeViews() ([]*graph.Edge, error) {
	// 1. 获取所有视图定义
	// 2. 解析 SQL 提取依赖的表
	// 3. 创建 Dependency 类型的边
	var edges []*graph.Edge
	
	// ... 实现逻辑
	
	return edges, nil
}
```

### 2. 在主流程中调用

在 `cmd/analyzer/main.go` 添加：

```go
// 分析视图依赖
fmt.Println("\n🔗 分析视图依赖...")
depAnalyzer := analyzer.NewDependencyAnalyzer(dbAdapter)
depEdges, err := depAnalyzer.AnalyzeViews()
if err != nil {
	log.Printf("分析视图时出错: %v", err)
} else {
	for _, edge := range depEdges {
		g.AddEdge(edge)
	}
	fmt.Printf("✓ 发现 %d 个依赖关系\n", len(depEdges))
}
```

## 添加新输出格式

### 1. 创建渲染器文件

在 `internal/renderer/` 创建新文件，例如 `html.go`：

```go
package renderer

import (
	"fmt"
	"html/template"
	"schema-analyzer/internal/graph"
	"strings"
)

type HTMLRenderer struct {
	template *template.Template
}

func NewHTMLRenderer() *HTMLRenderer {
	tmpl := template.Must(template.New("schema").Parse(htmlTemplate))
	return &HTMLRenderer{template: tmpl}
}

func (h *HTMLRenderer) Render(g *graph.SchemaGraph) string {
	var sb strings.Builder
	
	data := struct {
		Tables []TableData
		Edges  []EdgeData
	}{
		Tables: h.extractTables(g),
		Edges:  h.extractEdges(g),
	}
	
	h.template.Execute(&sb, data)
	return sb.String()
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
	<title>Schema Viewer</title>
	<style>
		/* CSS 样式 */
	</style>
</head>
<body>
	<h1>数据库结构</h1>
	{{range .Tables}}
	<div class="table">
		<h2>{{.Name}}</h2>
		<!-- 表格内容 -->
	</div>
	{{end}}
</body>
</html>
`
```

### 2. 在主流程中使用

```go
// HTML 输出
htmlRenderer := renderer.NewHTMLRenderer()
htmlContent := htmlRenderer.Render(g)
os.WriteFile(fmt.Sprintf("%s/schema.html", outputDir), []byte(htmlContent), 0644)
fmt.Printf("✓ %s/schema.html\n", outputDir)
```

## 添加新边类型

### 1. 在 graph/edge.go 添加类型

```go
const (
	EdgeTypeFK         EdgeType = "foreign_key"
	EdgeTypeInferredFK EdgeType = "inferred_fk"
	EdgeTypeDependency EdgeType = "dependency"
	EdgeTypeEnum       EdgeType = "enum_reference"
	EdgeTypeInheritance EdgeType = "inheritance"  // 新增
)
```

### 2. 创建分析逻辑

```go
func (a *InheritanceAnalyzer) DetectInheritance(meta *adapter.SchemaMetadata) ([]*graph.Edge, error) {
	// 检测表继承关系（例如：相同前缀、相似结构）
	var edges []*graph.Edge
	
	for _, table1 := range meta.Tables {
		for _, table2 := range meta.Tables {
			if similarity := a.calculateStructureSimilarity(table1, table2); similarity > 0.8 {
				edge := &graph.Edge{
					Type:       EdgeTypeInheritance,
					From:       table1.Name,
					To:         table2.Name,
					Confidence: similarity,
				}
				edges = append(edges, edge)
			}
		}
	}
	
	return edges, nil
}
```

## 自定义评分算法

### 修改关系推断权重

在 `internal/analyzer/relation.go` 修改：

```go
// 原始权重
totalScore += nameScore * 0.3
totalScore += typeScore * 0.2
totalScore += containmentScore * 0.5

// 自定义权重（更重视命名）
totalScore += nameScore * 0.5
totalScore += typeScore * 0.2
totalScore += containmentScore * 0.3
```

### 添加新的证据类型

```go
// 4. 索引证据（新增）
indexScore := r.calculateIndexEvidence(fromTable, fromCol.Name, toTable, toCol.Name)
if indexScore > 0 {
	evidences = append(evidences, graph.Evidence{
		Type:        "index_evidence",
		Score:       indexScore,
		Description: "索引关联",
		Details:     "两列都有索引且名称相似",
	})
	totalScore += indexScore * 0.1
}
```

## 添加配置文件支持

### 1. 使用 Viper 读取配置

```go
import "github.com/spf13/viper"

func loadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("配置文件未找到: %v", err)
	}
}

func main() {
	loadConfig()
	
	// 从配置读取
	minConfidence := viper.GetFloat64("analysis.min_confidence")
	sampleSize := viper.GetInt("analysis.sample_size")
}
```

### 2. 配置文件示例

```yaml
# config.yaml
database:
  type: sqlserver
  connection: "server=localhost;..."
  
analysis:
  min_confidence: 0.6
  sample_size: 2000
  weights:
    naming: 0.3
    type: 0.2
    value: 0.5
    
output:
  formats:
    - json
    - markdown
    - html
  directory: ./output
```

## 添加缓存支持

### 1. 使用 SQLite 缓存

```go
// internal/storage/cache.go
package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Cache struct {
	db *sql.DB
}

func NewCache(path string) (*Cache, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	
	// 创建表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS column_stats (
			table_name TEXT,
			column_name TEXT,
			stats JSON,
			updated_at TIMESTAMP,
			PRIMARY KEY (table_name, column_name)
		)
	`)
	
	return &Cache{db: db}, nil
}

func (c *Cache) GetColumnStats(table, column string) (*ColumnStats, error) {
	// 从缓存读取
}

func (c *Cache) SetColumnStats(table, column string, stats *ColumnStats) error {
	// 写入缓存
}
```

### 2. 在分析器中使用

```go
cache, _ := storage.NewCache(".cache.db")

// 先查缓存
if stats, err := cache.GetColumnStats(table, column); err == nil {
	return stats, nil
}

// 缓存未命中，采样
stats, _ := adapter.SampleColumnStats(table, column, sampleSize)
cache.SetColumnStats(table, column, stats)
```

## 测试新功能

### 单元测试

```go
// internal/analyzer/myanalyzer_test.go
package analyzer

import "testing"

func TestMyAnalyzer(t *testing.T) {
	// 创建 mock adapter
	mockAdapter := &MockAdapter{}
	
	analyzer := NewMyAnalyzer(mockAdapter)
	result, err := analyzer.Analyze()
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}
```

### 集成测试

```bash
# 使用 Docker 启动测试数据库
docker run -d --name test-mysql \
  -e MYSQL_ROOT_PASSWORD=test \
  -e MYSQL_DATABASE=testdb \
  -p 3306:3306 \
  mysql:8

# 运行测试
go test ./... -v
```

## 贡献代码

1. Fork 项目
2. 创建特性分支：`git checkout -b feature/my-feature`
3. 提交更改：`git commit -am 'Add my feature'`
4. 推送分支：`git push origin feature/my-feature`
5. 提交 Pull Request

## 代码规范

- 使用 `gofmt` 格式化代码
- 添加必要的注释
- 编写单元测试
- 更新相关文档

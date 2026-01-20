# 架构设计

## 核心理念

Schema Analyzer 采用 **"证据驱动 + 插件化方言 + 图谱输出"** 的设计理念：

1. **证据驱动**：每个推断都有明确的证据链和置信度
2. **插件化方言**：数据库适配层完全解耦，易于扩展
3. **图谱输出**：统一的 Schema Graph 表示，支持多种输出格式

## 架构分层

```
┌─────────────────────────────────────────┐
│           CLI / API Layer               │  用户接口
├─────────────────────────────────────────┤
│         Renderer Layer                  │  输出层
│  (Markdown, Mermaid, JSON, HTML)        │
├─────────────────────────────────────────┤
│         Analyzer Layer                  │  分析引擎
│  - RelationshipInferer                  │
│  - EnumDetector                         │
│  - KeyDetector                          │
│  - SemanticSummarizer                   │
├─────────────────────────────────────────┤
│         Schema Graph                    │  核心数据结构
│  (Nodes + Edges + Evidence)             │
├─────────────────────────────────────────┤
│         Adapter Layer                   │  数据库适配
│  (SQLServer, MySQL, Postgres, ...)     │
└─────────────────────────────────────────┘
```

## 核心组件

### 1. Adapter Layer（适配层）

**职责**：屏蔽不同数据库的差异，提供统一接口

**接口定义**：

```go
type DBAdapter interface {
    IntrospectSchema() (*SchemaMetadata, error)
    EstimateRowCount(table string) (int64, error)
    SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error)
    GetPrimaryKeys(table string) ([]string, error)
    GetForeignKeys() ([]ForeignKey, error)
}
```

**实现要点**：
- 使用 INFORMATION_SCHEMA 或系统表
- 采样查询要高效（TABLESAMPLE / RAND()）
- 错误处理要优雅（部分失败不影响整体）

### 2. Schema Graph（核心数据结构）

**设计思想**：把数据库看成图

```go
type SchemaGraph struct {
    Nodes map[string]*Node  // 表、列、索引
    Edges map[string]*Edge  // 关系、依赖
}

type Node struct {
    ID         string
    Type       NodeType  // table/column/index/view
    Properties map[string]interface{}
}

type Edge struct {
    ID         string
    Type       EdgeType  // fk/inferred_fk/dependency
    From       string
    To         string
    Confidence float64
    Evidence   []Evidence
}
```

**优势**：
- 统一表示，易于查询和遍历
- 支持复杂关系（多对多、自引用）
- 可直接导出为图数据库（Neo4j）

### 3. Analyzer Layer（分析引擎）

#### RelationshipInferer（关系推断器）

**算法流程**：

```
1. 遍历所有非主键列
2. 与所有表的主键列比较
3. 计算三个维度的得分：
   - 命名相似度（Levenshtein 距离）
   - 类型匹配度（类型 + 长度）
   - 值包含度（采样检查）
4. 加权求和得到置信度
5. 过滤低置信度关系（< 0.5）
```

**评分公式**：

```
Confidence = 
    NameSimilarity × 0.3 +
    TypeMatch × 0.2 +
    ValueContainment × 0.5
```

**为什么这样设计**：
- 值包含度权重最高（0.5）：最可靠的证据
- 命名相似度次之（0.3）：命名规范很重要
- 类型匹配最低（0.2）：必要但不充分

#### EnumDetector（枚举表检测器）

**识别规则**：

```
1. 行数 < 1000（可配置）
2. 有 code/id + name/label 列组合
3. 列数少（2-5 列）
4. 被多个表引用（可选）
```

**置信度计算**：

```
Score = 
    RowCountScore (0.2-0.4) +
    ColumnStructureScore (0.2-0.4) +
    ColumnCountScore (0.2)
```

### 4. Renderer Layer（输出层）

**职责**：将 Schema Graph 转换为不同格式

**支持格式**：
- JSON：完整数据，供程序集成
- Markdown：人类可读的数据字典
- Mermaid：ER 图可视化
- HTML：交互式 Web 页面（规划中）

## 数据流

```
数据库
  ↓
Adapter.IntrospectSchema()
  ↓
SchemaMetadata (表/列/索引)
  ↓
GraphBuilder.Build()
  ↓
SchemaGraph (节点)
  ↓
Analyzer.Infer()
  ↓
SchemaGraph (节点 + 边 + 证据)
  ↓
Renderer.Render()
  ↓
输出文件 (JSON/MD/MMD)
```

## 扩展点

### 1. 添加新数据库

```go
// 1. 创建适配器
type PostgresAdapter struct {
    db *sql.DB
}

// 2. 实现接口
func (a *PostgresAdapter) IntrospectSchema() (*SchemaMetadata, error) {
    // 使用 pg_catalog 查询
}

// 3. 注册到 main.go
case "postgres":
    dbAdapter, err = adapter.NewPostgresAdapter(connStr)
```

### 2. 添加新分析器

```go
// 1. 创建分析器
type DependencyAnalyzer struct {
    adapter adapter.DBAdapter
}

// 2. 实现分析逻辑
func (d *DependencyAnalyzer) AnalyzeViews() ([]*graph.Edge, error) {
    // 解析 VIEW 定义，提取依赖关系
}

// 3. 在 main.go 中调用
depAnalyzer := analyzer.NewDependencyAnalyzer(dbAdapter)
depEdges, _ := depAnalyzer.AnalyzeViews()
```

### 3. 添加新输出格式

```go
// 1. 创建渲染器
type HTMLRenderer struct{}

// 2. 实现渲染逻辑
func (h *HTMLRenderer) Render(g *graph.SchemaGraph) string {
    // 生成交互式 HTML
}

// 3. 在 main.go 中使用
htmlRenderer := renderer.NewHTMLRenderer()
htmlContent := htmlRenderer.Render(g)
```

## 性能优化

### 当前实现

- 串行处理每个表
- 采样大小固定（1000 行）
- 内存中构建完整图

### 优化方向

1. **并发采样**：
```go
var wg sync.WaitGroup
for _, table := range tables {
    wg.Add(1)
    go func(t Table) {
        defer wg.Done()
        stats, _ := adapter.SampleColumnStats(t.Name, ...)
    }(table)
}
wg.Wait()
```

2. **增量分析**：
```go
// 只分析变更的表
if cache.HasTable(tableName) && !cache.IsModified(tableName) {
    continue
}
```

3. **流式输出**：
```go
// 边分析边输出，不等全部完成
for edge := range edgeChan {
    renderer.WriteEdge(edge)
}
```

## 安全考虑

1. **只读操作**：所有查询都是 SELECT
2. **参数化查询**：防止 SQL 注入
3. **采样限制**：避免全表扫描
4. **超时控制**：防止长时间阻塞
5. **脱敏选项**：不输出实际数据值

## 测试策略

1. **单元测试**：每个分析器独立测试
2. **集成测试**：使用 Docker 启动测试数据库
3. **基准测试**：性能回归检测
4. **端到端测试**：完整流程验证

## 未来规划

### Phase 2: 血缘分析
- 解析 VIEW / PROC 定义
- 提取列级依赖
- 生成数据血缘图

### Phase 3: Schema Diff
- 对比两次扫描结果
- 识别结构变更
- 生成迁移建议

### Phase 4: Web UI
- 本地 Web 服务器
- 交互式图谱浏览
- 关系路径查询

### Phase 5: AI 增强
- 字段语义推断
- 命名规范建议
- 数据质量评分

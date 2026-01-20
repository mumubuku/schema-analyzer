# Schema Analyzer - 通用数据库结构分析器

一个用 Go 编写的通用数据库结构分析工具，能够自动推断表关系、检测枚举表、生成数据字典和 ER 图。

## 核心特性

### 1. 自动推断隐式外键
- **命名相似度分析**：识别 `cDepCode` ↔ `Department.code` 这样的关联
- **值集合包含检测**：检查列值是否存在于目标表主键中
- **类型匹配验证**：确保数据类型和长度兼容
- **置信度评分**：每个推断关系都有 0-1 的置信度分数
- **证据链**：详细记录推断依据（命名/类型/值包含）

### 2. 枚举/码表自动识别
- 识别小表（< 1000 行）
- 检测 code/name 或 id/label 结构
- 分析被引用情况

### 3. Schema Graph 输出
- **节点**：Table、Column、Index
- **边**：FK、InferredFK、Dependency
- **格式**：JSON、Markdown、Mermaid

### 4. 多数据库支持
- SQL Server（适配 U8 等系统）
- MySQL
- 插件化设计，易于扩展

## 快速开始

### 安装依赖

```bash
go mod download
```

### 构建

```bash
go build -o schema-analyzer cmd/analyzer/main.go
```

### 使用示例

#### SQL Server (U8)

```bash
./schema-analyzer scan \
  --type sqlserver \
  --conn "server=localhost;user id=sa;password=yourpass;database=U8" \
  --output ./output
```

#### MySQL

```bash
./schema-analyzer scan \
  --type mysql \
  --conn "user:pass@tcp(localhost:3306)/dbname" \
  --schema dbname \
  --output ./output
```

### 输出文件

- `schema.json` - 完整的 Schema Graph（节点+边+证据）
- `dict.md` - Markdown 数据字典（表结构+关系+证据链）
- `er.mmd` - Mermaid ER 图（可用 Mermaid Live Editor 查看）

## 项目结构

```
schema-analyzer/
├── cmd/analyzer/          # CLI 入口
├── internal/
│   ├── adapter/          # 数据库适配层（插件化）
│   │   ├── adapter.go   # 统一接口
│   │   ├── sqlserver.go # SQL Server 实现
│   │   └── mysql.go     # MySQL 实现
│   ├── analyzer/        # 分析引擎
│   │   ├── relation.go  # 关系推断（核心算法）
│   │   └── enum.go      # 枚举表检测
│   ├── graph/           # Schema Graph 核心
│   │   ├── graph.go
│   │   ├── node.go
│   │   └── edge.go
│   └── renderer/        # 输出渲染
│       ├── markdown.go
│       └── mermaid.go
└── go.mod
```

## 推断算法

### 关系置信度计算

```
总分 = 命名相似度 × 0.3 + 类型匹配 × 0.2 + 值包含度 × 0.5
```

- **命名相似度**：Levenshtein 距离 + 前缀处理（去除 `c` 前缀）
- **类型匹配**：数据类型兼容性 + 长度一致性
- **值包含度**：采样检查源列值在目标列中的存在比例（最重要）

### 枚举表识别

```
置信度 = 行数评分 + 列结构评分 + 列数评分
```

- 行数 < 100：0.4 分
- 有 key + value 列：0.4 分
- 列数 ≤ 5：0.2 分

## 扩展新数据库

实现 `adapter.DBAdapter` 接口：

```go
type DBAdapter interface {
    IntrospectSchema() (*SchemaMetadata, error)
    EstimateRowCount(table string) (int64, error)
    SampleColumnStats(table, column string, sampleSize int) (*ColumnStats, error)
    GetPrimaryKeys(table string) ([]string, error)
    GetForeignKeys() ([]ForeignKey, error)
}
```

## 配置参数

- `--type`: 数据库类型 (sqlserver/mysql)
- `--conn`: 连接字符串
- `--schema`: 数据库 schema (MySQL 必需)
- `--output`: 输出目录 (默认 ./output)
- `--sample`: 采样大小 (默认 1000)

## 安全特性

- **只读操作**：仅执行 SELECT 查询
- **采样分析**：避免全表扫描
- **脱敏支持**：统计摘要不包含敏感数据

## AI 增强（可选）

工具**默认不使用 AI**，完全基于算法和统计。但你可以选择启用 AI 来：

- 生成字段注释建议
- 推断字段语义类型
- 优化命名建议

**重要**：AI 只负责"组织语言"，不负责"拍脑袋"。所有推断都基于统计证据。

```bash
# 启用 AI（需要 API Key）
export OPENAI_API_KEY="sk-..."
./schema-analyzer scan --type sqlserver --conn "..." --enable-ai
```

详见 `internal/analyzer/semantic.go` 的实现示例。

## 后续规划

- [ ] SQL 依赖/血缘分析（View/Proc）
- [ ] Schema Diff（版本对比）
- [ ] Web UI（本地可视化）
- [ ] PostgreSQL/Oracle 支持
- [ ] 并发优化（goroutine 池）
- [ ] AI 增强集成（可选）

## License

MIT

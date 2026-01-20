# Schema Analyzer - 项目总览

## 项目定位

一个**通用的数据库结构分析工具**，专注于：
- 自动推断隐式外键关系
- 检测枚举/码表
- 生成数据字典和 ER 图
- 提供证据驱动的分析结果

## 核心价值

### 1. 解决真实痛点
很多老系统（如用友 U8）没有建立外键约束，表关系靠"约定"维护。这个工具能：
- 自动发现这些隐式关系
- 给出置信度和证据链
- 生成可读的文档

### 2. 通用性强
- 支持多种数据库（SQL Server、MySQL，易扩展）
- 插件化架构
- 统一的 Schema Graph 输出

### 3. 证据驱动
每个推断都有：
- 置信度分数（0-1）
- 详细证据（命名/类型/值包含）
- 可解释性强

## 技术亮点

### 推断算法
```
置信度 = 命名相似度×0.3 + 类型匹配×0.2 + 值包含度×0.5
```
- 使用 Levenshtein 距离计算命名相似度
- 采样检查值集合包含关系
- 综合多维度证据

### 架构设计
```
CLI → Renderer → Analyzer → Graph → Adapter → Database
```
- 分层清晰
- 易于扩展
- 职责单一

## 项目结构

```
schema-analyzer/
├── cmd/analyzer/          # CLI 入口
├── internal/
│   ├── adapter/          # 数据库适配层（插件化）
│   ├── analyzer/         # 分析引擎（关系推断、枚举检测）
│   ├── graph/            # Schema Graph 核心数据结构
│   └── renderer/         # 输出层（Markdown、Mermaid、JSON）
├── docs/                 # 文档
├── examples/             # 使用示例
└── go.mod
```

## 已实现功能

✅ SQL Server 适配器
✅ MySQL 适配器
✅ 关系推断（命名+类型+值包含）
✅ 枚举表检测
✅ Markdown 数据字典输出
✅ Mermaid ER 图输出
✅ JSON 完整数据输出
✅ CLI 命令行工具

## 待实现功能

🔲 SQL 依赖/血缘分析（View/Proc）
🔲 Schema Diff（版本对比）
🔲 Web UI（本地可视化）
🔲 PostgreSQL/Oracle 支持
🔲 并发优化
🔲 缓存机制
🔲 AI 增强（字段语义推断）

## 使用场景

1. **遗留系统分析**：理解没有文档的老系统
2. **数据库迁移**：梳理表关系，规划迁移方案
3. **新人上手**：快速了解数据库结构
4. **数据治理**：发现数据质量问题
5. **BI 建模**：识别维度表和事实表

## 快速开始

```bash
# 构建
make build

# 分析 SQL Server
./schema-analyzer scan --type sqlserver --conn "..." --output ./output

# 分析 MySQL
./schema-analyzer scan --type mysql --conn "..." --schema mydb --output ./output
```

## 文档导航

- [快速开始](QUICKSTART.md) - 5 分钟上手
- [使用指南](docs/USAGE.md) - 详细使用说明
- [架构设计](docs/ARCHITECTURE.md) - 技术架构
- [扩展指南](docs/EXTENDING.md) - 如何扩展功能

## 技术栈

- **语言**: Go 1.17+
- **数据库驱动**: go-mssqldb, go-sql-driver/mysql
- **CLI 框架**: cobra
- **字符串相似度**: golang-levenshtein

## 性能指标

- 100 表数据库：约 30 秒
- 500 表数据库：约 2-3 分钟
- 内存占用：< 100MB

## 贡献

欢迎提交 Issue 和 Pull Request！

## License

MIT

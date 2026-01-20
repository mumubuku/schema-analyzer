# 快速参考

## 命令速查

### 基础分析（不用 AI）

```bash
# SQL Server (U8)
./schema-analyzer scan \
  --type sqlserver \
  --conn "server=localhost;user id=sa;password=pass;database=U8" \
  --output ./output

# MySQL
./schema-analyzer scan \
  --type mysql \
  --conn "root:pass@tcp(localhost:3306)/db" \
  --schema db \
  --output ./output
```

### AI 增强分析

```bash
# 设置 API Key
export DASHSCOPE_API_KEY="sk-xxxxx"

# 运行分析
./schema-analyzer scan \
  --type sqlserver \
  --conn "..." \
  --enable-ai \
  --output ./output

# 或直接传入 Key
./schema-analyzer scan \
  --type sqlserver \
  --conn "..." \
  --enable-ai \
  --ai-key "sk-xxxxx"
```

## 参数说明

| 参数 | 说明 | 默认值 | 必需 |
|------|------|--------|------|
| `--type` | 数据库类型 | sqlserver | 否 |
| `--conn` | 连接字符串 | - | 是 |
| `--schema` | 数据库 schema | - | MySQL 必需 |
| `--output` | 输出目录 | ./output | 否 |
| `--sample` | 采样大小 | 1000 | 否 |
| `--enable-ai` | 启用 AI | false | 否 |
| `--ai-key` | AI API Key | - | AI 时需要 |

## 连接字符串格式

### SQL Server

```
server=HOST;user id=USER;password=PASS;database=DB
server=192.168.1.100;user id=sa;password=Pass123;database=U8
server=localhost;user id=sa;password=Pass123;database=UFDATA_001_2023
```

### MySQL

```
USER:PASS@tcp(HOST:PORT)/DB
root:password@tcp(localhost:3306)/mydb
admin:pass123@tcp(192.168.1.100:3306)/business_db
```

## 输出文件

| 文件 | 格式 | 说明 |
|------|------|------|
| `schema.json` | JSON | 完整的 Schema Graph 数据 |
| `dict.md` | Markdown | 数据字典（表结构+关系+证据） |
| `er.mmd` | Mermaid | ER 图（可在 mermaid.live 查看） |

## 环境变量

```bash
# 阿里云 API Key
export DASHSCOPE_API_KEY="sk-xxxxx"

# 数据库连接（可选）
export DB_CONN="server=...;database=..."
export DB_SCHEMA="mydb"
```

## 常用场景

### 1. 快速分析

```bash
make build
./schema-analyzer scan --type sqlserver --conn "..." --output ./output
cat output/dict.md
```

### 2. AI 增强分析

```bash
export DASHSCOPE_API_KEY="sk-xxxxx"
./schema-analyzer scan --type sqlserver --conn "..." --enable-ai
```

### 3. 大型数据库

```bash
# 减少采样，提高速度
./schema-analyzer scan --type sqlserver --conn "..." --sample 500
```

### 4. 使用示例脚本

```bash
chmod +x examples/*.sh
./examples/u8_example.sh          # 基础分析
./examples/u8_ai_example.sh       # AI 增强
./examples/mysql_example.sh       # MySQL
```

## 输出示例

### 不启用 AI

```markdown
### Department

| 列名 | 类型 | 长度 | 可空 | 主键 | Null率 | 唯一值率 |
|------|------|------|------|------|--------|----------|
| cDepCode | varchar | 20 | 否 | ✓ | 0.0% | 100.0% |
| cDepName | varchar | 60 | 否 |  | 0.0% | 98.5% |

#### 关系
- **推断外键** `Employee.cDepCode` → `Department.cDepCode` (置信度: 0.93)
```

### 启用 AI

```markdown
### Department

| 列名 | 中文名 | 类型 | 业务含义 | 来源 | 置信度 |
|------|--------|------|----------|------|--------|
| cDepCode | 部门编码 | varchar | 用于标识部门的唯一编码 | 🤖标准 | 95% |
| cFree1 | 关联部门 | varchar | 基于关联推断的部门字段 | 🔍推断 | 75% |
```

## 故障排除

### 连接失败

```bash
# 测试连接
# SQL Server
sqlcmd -S localhost -U sa -P pass -d U8 -Q "SELECT @@VERSION"

# MySQL
mysql -h localhost -u root -p -e "SHOW DATABASES"
```

### AI 调用失败

```bash
# 检查 API Key
echo $DASHSCOPE_API_KEY

# 测试 API
curl -X POST https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen-turbo","input":{"messages":[{"role":"user","content":"test"}]}}'
```

### 权限不足

确保数据库用户有以下权限：
- SELECT on INFORMATION_SCHEMA
- SELECT on 目标数据库的所有表

### 采样太慢

```bash
# 减少采样大小
--sample 500

# 或只分析特定表（修改代码）
WHERE TABLE_NAME LIKE 'User%'
```

## 性能参考

| 数据库规模 | 采样大小 | 预计时间 | 内存占用 |
|-----------|---------|---------|---------|
| 50 表 | 1000 | ~15 秒 | < 50MB |
| 100 表 | 1000 | ~30 秒 | < 100MB |
| 500 表 | 1000 | ~2-3 分钟 | < 200MB |
| 500 表 | 500 | ~1-2 分钟 | < 150MB |

## 成本参考（AI）

| 数据库规模 | 字段数 | Tokens | 成本（qwen-plus） |
|-----------|--------|--------|------------------|
| 小型（50表） | ~250 | ~25K | ~¥0.5 |
| 中型（100表） | ~500 | ~50K | ~¥1 |
| 大型（500表） | ~2500 | ~250K | ~¥5 |

## 文档导航

- [README.md](README.md) - 项目介绍
- [README_AI.md](README_AI.md) - AI 功能指南
- [QUICKSTART.md](QUICKSTART.md) - 快速开始
- [docs/USAGE.md](docs/USAGE.md) - 详细使用
- [docs/AI_INTEGRATION.md](docs/AI_INTEGRATION.md) - AI 集成
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - 架构设计
- [docs/EXTENDING.md](docs/EXTENDING.md) - 扩展开发

## 获取帮助

```bash
# 查看帮助
./schema-analyzer --help
./schema-analyzer scan --help

# 查看版本
./schema-analyzer version
```

## 联系方式

- GitHub Issues
- Pull Requests
- 文档反馈

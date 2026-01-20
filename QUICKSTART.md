# 快速开始

## 5 分钟上手

### 1. 构建项目

```bash
# 下载依赖
go mod download

# 构建
make build
```

### 2. 准备数据库连接

#### 选项 A: SQL Server (U8)

```bash
export DB_CONN="server=localhost;user id=sa;password=YourPass;database=U8"
```

#### 选项 B: MySQL

```bash
export DB_CONN="root:password@tcp(localhost:3306)/mydb"
export DB_SCHEMA="mydb"
```

### 3. 运行分析

#### SQL Server

```bash
./schema-analyzer scan \
  --type sqlserver \
  --conn "$DB_CONN" \
  --output ./output
```

#### MySQL

```bash
./schema-analyzer scan \
  --type mysql \
  --conn "$DB_CONN" \
  --schema "$DB_SCHEMA" \
  --output ./output
```

### 4. 查看结果

```bash
# 查看数据字典
cat output/dict.md

# 查看 JSON 数据
cat output/schema.json | jq .

# 查看 ER 图（复制内容到 https://mermaid.live/）
cat output/er.mmd
```

## 输出示例

### 推断关系示例

```
Employee.cDepCode → Department.cDepCode (置信度: 0.93)
证据:
  - 列名相似度 (1.00): cDepCode ↔ cDepCode
  - 数据类型匹配 (1.00): varchar(20) ↔ varchar(20)
  - 值集合包含度 (0.98): 98.0% 的值存在于目标表
```

### 枚举表示例

```
发现枚举表:
  - CodeTable_Status (行数: 15, 置信度: 0.90)
  - CodeTable_Type (行数: 32, 置信度: 0.85)
```

## 常用命令

```bash
# 使用示例脚本
./examples/u8_example.sh        # U8 数据库
./examples/mysql_example.sh     # MySQL 数据库

# 调整采样大小
./schema-analyzer scan --type sqlserver --conn "..." --sample 5000

# 查看帮助
./schema-analyzer --help
./schema-analyzer scan --help
```

## 下一步

- 📖 阅读 [完整使用指南](docs/USAGE.md)
- 🏗️ 了解 [架构设计](docs/ARCHITECTURE.md)
- 🔧 查看 [配置示例](config.example.yaml)

## 故障排除

### 连接失败

```bash
# 测试数据库连接
# SQL Server
sqlcmd -S localhost -U sa -P YourPass -d U8 -Q "SELECT @@VERSION"

# MySQL
mysql -h localhost -u root -p -e "SHOW DATABASES"
```

### 权限不足

确保数据库用户有以下权限：
- SELECT on INFORMATION_SCHEMA
- SELECT on 目标数据库的所有表

### 采样太慢

减少采样大小：
```bash
--sample 500  # 默认 1000
```

## 联系方式

- 提交 Issue
- 发起 Pull Request
- 查看文档

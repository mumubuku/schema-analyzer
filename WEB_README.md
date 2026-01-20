# Schema Analyzer Web 版使用指南

## 🌐 Web 界面

除了命令行工具，Schema Analyzer 还提供了友好的 Web 界面！

### 特性

- 📝 **表单输入**：无需记忆命令行参数
- 📊 **实时进度**：WebSocket 实时显示分析进度
- 🎨 **在线查看**：直接在浏览器查看结果
- 💾 **多种格式**：数据字典、ER 图、JSON 数据

## 🚀 快速开始

### 1. 启动服务器

```bash
# 构建 Web 服务器
go build -o schema-analyzer-server cmd/server/main.go

# 启动服务（默认端口 8080）
./schema-analyzer-server

# 或指定端口
PORT=3000 ./schema-analyzer-server
```

### 2. 打开浏览器

访问 http://localhost:8080

### 3. 填写表单

1. 选择数据库类型（MySQL / SQL Server）
2. 填写连接信息
3. （可选）启用 AI 增强
4. 点击"开始分析"

### 4. 查看结果

- **数据字典**：表结构 + 字段解释 + 关系
- **ER 图**：Mermaid 格式（可复制到 mermaid.live 查看）
- **JSON 数据**：完整的 Schema Graph

## 📸 界面预览

### 输入表单

```
┌─────────────────────────────────────┐
│  数据库类型: [MySQL ▼]              │
│  主机地址:   localhost               │
│  端口:       3306                    │
│  用户名:     root                    │
│  密码:       ********                │
│  数据库名:   mydb                    │
│  采样大小:   1000                    │
│  □ 启用 AI 增强                      │
│  [开始分析]                          │
└─────────────────────────────────────┘
```

### 分析进度

```
████████████████░░░░░░░░░░ 60%
正在推断表间关系...
```

### 结果展示

```
┌─────────────────────────────────────┐
│  [数据字典] [ER 图] [JSON 数据]      │
├─────────────────────────────────────┤
│  📊 统计信息                         │
│  ┌──────┐ ┌──────┐ ┌──────┐        │
│  │  50  │ │  23  │ │  12  │        │
│  │ 数据表│ │ 关系 │ │枚举表│        │
│  └──────┘ └──────┘ └──────┘        │
│                                     │
│  ### Department（部门档案）          │
│  | 列名 | 中文名 | 类型 | ...        │
│  |------|--------|------|...        │
│  | cDepCode | 部门编码 | varchar |  │
└─────────────────────────────────────┘
```

## 🔧 配置

### 环境变量

```bash
# 服务端口
export PORT=8080

# 阿里云 API Key（可选，也可在界面输入）
export DASHSCOPE_API_KEY="sk-xxxxx"
```

### 自定义配置

编辑 `cmd/server/main.go` 可以修改：
- 默认采样大小
- 超时时间
- 并发限制

## 🌟 使用技巧

### 1. 测试连接

先用小数据库测试，确保连接正常。

### 2. 采样大小

- 小数据库（< 100 表）：1000
- 中型数据库（100-500 表）：500
- 大型数据库（> 500 表）：200

### 3. AI 增强

- 标准字段多：启用 AI，效果明显
- 自定义字段多：启用 AI，基于关联推断
- 没有 API Key：不启用，仍可使用基础功能

### 4. 查看 ER 图

1. 切换到"ER 图"标签
2. 复制 Mermaid 代码
3. 打开 https://mermaid.live/
4. 粘贴代码查看图形

## 🔒 安全建议

### 生产环境部署

1. **使用 HTTPS**
```bash
# 使用 Nginx 反向代理
server {
    listen 443 ssl;
    server_name schema-analyzer.example.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

2. **添加认证**
```go
// 在 main.go 中添加中间件
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 验证 token
        token := r.Header.Get("Authorization")
        if !validateToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

3. **限制访问**
```bash
# 只允许内网访问
./schema-analyzer-server -bind 127.0.0.1:8080
```

### 数据安全

- ✅ 密码不会被记录
- ✅ 只发送元数据给 AI
- ✅ 结果存储在内存（重启清空）

## 🐛 故障排除

### 端口被占用

```bash
# 查看占用端口的进程
lsof -i :8080

# 使用其他端口
PORT=3000 ./schema-analyzer-server
```

### WebSocket 连接失败

浏览器会自动降级到轮询模式，不影响使用。

### 分析超时

增加采样大小或分批分析：
```bash
# 只分析特定表
WHERE TABLE_NAME LIKE 'User%'
```

## 📦 Docker 部署

```dockerfile
FROM golang:1.17 AS builder
WORKDIR /app
COPY . .
RUN go build -o schema-analyzer-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/schema-analyzer-server .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./schema-analyzer-server"]
```

```bash
# 构建镜像
docker build -t schema-analyzer-web .

# 运行容器
docker run -p 8080:8080 schema-analyzer-web
```

## 🔄 与 CLI 版本对比

| 特性 | CLI 版本 | Web 版本 |
|------|---------|---------|
| 使用方式 | 命令行 | 浏览器 |
| 学习成本 | 需要记忆参数 | 表单填写 |
| 进度显示 | 文本输出 | 实时进度条 |
| 结果查看 | 文件 | 在线查看 |
| 批量分析 | 支持脚本 | 单次分析 |
| 适用场景 | 自动化、CI/CD | 临时分析、演示 |

## 🎯 最佳实践

1. **开发环境**：使用 Web 版快速分析
2. **生产环境**：使用 CLI 版集成到流程
3. **演示场景**：使用 Web 版展示效果
4. **定期分析**：使用 CLI 版 + cron 定时任务

## 📞 获取帮助

- GitHub Issues
- 查看文档
- 提交 PR

---

**💡 提示**：Web 版和 CLI 版使用相同的核心代码，功能完全一致！

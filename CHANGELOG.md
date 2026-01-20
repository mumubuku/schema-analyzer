# 更新日志

## [v1.1.0] - 2026-01-20

### 🌐 新增 Web 界面

#### 核心功能
- **表单输入**：友好的 Web 界面，无需记忆命令行参数
- **实时进度**：WebSocket 实时显示分析进度，支持降级轮询
- **在线查看**：直接在浏览器查看分析结果
- **多标签切换**：数据字典、ER 图、JSON 数据一键切换

#### 增强功能
- 🔌 **测试连接**：验证数据库连接是否正常
- 📋 **选择数据库**：自动列出所有可用数据库，点击选择
- 💡 **采样大小说明**：详细解释参数含义和建议值
- 🎨 **自动填充**：MySQL schema 自动同步 database 名称

#### 技术实现
- Go Web 服务器（Gorilla WebSocket）
- 异步任务处理
- 实时进度推送
- 优雅的错误处理和降级策略

### 🐛 Bug 修复
- 修复进度条显示 `undefined%` 的问题
- 修复 MySQL `unexpected EOF` 连接超时错误
- 优化 WebSocket 连接稳定性
- 添加连接超时配置（30秒）

### 📚 文档更新
- 新增 `WEB_README.md` - Web 版详细使用指南
- 新增 `start-web.sh` - 一键启动脚本
- 更新主 `README.md` 添加 Web 版说明
- 更新 `.gitignore` 排除二进制文件

### 🎯 使用方式

#### 启动 Web 服务器
```bash
# 方式 1：使用启动脚本
./start-web.sh

# 方式 2：手动启动
go build -o schema-analyzer-server cmd/server/main.go
./schema-analyzer-server

# 访问
open http://localhost:8080
```

#### 使用流程
1. 填写数据库连接信息
2. 点击"测试连接"验证
3. 点击"选择"按钮选择数据库
4. 调整采样大小（可选）
5. 启用 AI（可选）
6. 点击"开始分析"
7. 查看实时进度
8. 在线查看结果

---

## [v1.0.0] - 2026-01-20

### 🎉 首次发布

#### 核心功能
- ✅ 自动推断隐式外键关系
- ✅ AI 增强字段解释（阿里云通义千问）
- ✅ 混合策略：算法 + AI
- ✅ 多数据库支持（SQL Server、MySQL）
- ✅ 多种输出格式（JSON、Markdown、Mermaid）

#### 关系推断算法
- 命名相似度（Levenshtein 距离）
- 类型匹配度
- 值集合包含度
- 综合置信度评分
- 详细证据链

#### AI 增强
- 标准字段直接识别
- 自定义字段基于关联推断
- 批量调用优化
- 降级策略保障

#### 输出格式
- `schema.json` - 完整 Schema Graph
- `dict.md` - 数据字典
- `er.mmd` - Mermaid ER 图

#### 文档
- 完整的使用文档
- 架构设计文档
- AI 集成指南
- 扩展开发指南

---

## 路线图

### v1.2.0（计划中）
- [ ] PostgreSQL 支持
- [ ] Oracle 支持
- [ ] Schema Diff（版本对比）
- [ ] 批量分析多个数据库

### v1.3.0（计划中）
- [ ] SQL 依赖分析（View/Proc）
- [ ] 数据血缘分析
- [ ] 列级依赖关系

### v2.0.0（计划中）
- [ ] 交互式图谱浏览
- [ ] 关系路径查询
- [ ] 数据质量评分
- [ ] 导出多种格式（Excel、PDF）

---

## 贡献

欢迎提交 Issue 和 Pull Request！

## License

MIT License

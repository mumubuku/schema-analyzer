# AI 集成指南

## 设计思路

### 混合策略：算法 + AI

Schema Analyzer 采用**混合策略**来处理 U8 数据库的字段：

1. **标准字段**（如 `cDepCode`、`cInvCode`）
   - AI 直接识别
   - 基于 U8 标准命名规范
   - 置信度高（0.8-0.95）

2. **自定义字段**（如 `cFree1-10`、`cDefine1-37`）
   - 先用算法推断关联关系
   - 再让 AI 基于关联推断含义
   - 置信度中等（0.5-0.8）

3. **降级策略**
   - AI 失败时，使用纯算法推断
   - 保证工具始终可用

## 使用阿里云通义千问

### 1. 获取 API Key

访问 [阿里云 DashScope](https://dashscope.console.aliyun.com/)：
1. 注册/登录阿里云账号
2. 开通 DashScope 服务
3. 创建 API Key

### 2. 设置环境变量

```bash
export DASHSCOPE_API_KEY="sk-xxxxx"
```

### 3. 运行 AI 增强分析

```bash
./schema-analyzer scan \
  --type sqlserver \
  --conn "server=localhost;user id=sa;password=pass;database=U8" \
  --output ./output \
  --enable-ai
```

或者直接传入 API Key：

```bash
./schema-analyzer scan \
  --type sqlserver \
  --conn "..." \
  --enable-ai \
  --ai-key "sk-xxxxx"
```

## 输出示例

### 标准字段（AI 直接识别）

| 列名 | 中文名 | 类型 | 业务含义 | 来源 | 置信度 |
|------|--------|------|----------|------|--------|
| cDepCode | 部门编码 | varchar | 用于标识部门的唯一编码 | 🤖标准 | 95% |
| cInvCode | 存货编码 | varchar | 存货档案的唯一标识 | 🤖标准 | 95% |

### 自定义字段（AI 推断）

| 列名 | 中文名 | 类型 | 业务含义 | 来源 | 置信度 |
|------|--------|------|----------|------|--------|
| cFree1 | 关联部门 | varchar | 基于关联推断：与部门编码关联 | 🔍推断 | 75% |
| cDefine1 | 自定义项目 | varchar | 基于关联推断：与项目编码关联 | 🔍推断 | 70% |

### 关系推断（纯算法）

| 列名 | 中文名 | 类型 | 业务含义 | 来源 | 置信度 |
|------|--------|------|----------|------|--------|
| cFree10 | 关联字段 | varchar | 与 XX 表关联，具体含义待确认 | 🔗关联 | 50% |

## 工作流程

```
1. 扫描数据库元数据
   ↓
2. 分类字段
   ├─ 标准字段 → AI 批量解释
   └─ 自定义字段 → 先推断关系
   ↓
3. 推断关联关系（算法）
   ↓
4. AI 基于关联推断自定义字段
   ↓
5. 生成增强版数据字典
```

## API 调用示例

### 标准字段解释

**输入 Prompt**:
```
你是用友 U8 ERP 系统的数据库专家。请解释以下字段：

表名: Department
字段名: cDepCode
数据类型: varchar(20)

请以 JSON 格式返回：
{
  "chinese_name": "部门编码",
  "description": "用于标识部门的唯一编码",
  "business_meaning": "在组织架构中唯一标识一个部门",
  "confidence": 0.95
}
```

**AI 响应**:
```json
{
  "chinese_name": "部门编码",
  "description": "部门档案的唯一标识",
  "business_meaning": "用于在系统中唯一标识一个部门，关联到组织架构",
  "confidence": 0.95
}
```

### 自定义字段推断

**输入 Prompt**:
```
你是用友 U8 ERP 系统的数据库专家。

字段 cFree1 是一个自定义字段，需要基于关联关系推断其含义。

已知关联关系：
- 与 Department.cDepCode (部门编码) 关联，置信度 0.85
- 与 Person.cPersonCode (人员编码) 关联，置信度 0.72

请基于这些关联关系，推断该字段的业务含义...
```

**AI 响应**:
```json
{
  "chinese_name": "关联部门",
  "description": "自定义部门关联字段",
  "business_meaning": "基于与部门编码的强关联，推断为扩展的部门关联字段",
  "confidence": 0.75
}
```

## 成本估算

### 阿里云通义千问定价（参考）

- qwen-turbo: ¥0.008/千tokens
- qwen-plus: ¥0.02/千tokens
- qwen-max: ¥0.12/千tokens

### 典型场景

**100 表数据库**：
- 标准字段：~500 个
- 自定义字段：~200 个
- 总 tokens：~50,000
- 成本（qwen-plus）：~¥1

**500 表数据库**：
- 标准字段：~2500 个
- 自定义字段：~1000 个
- 总 tokens：~250,000
- 成本（qwen-plus）：~¥5

## 优化建议

### 1. 批量调用

```go
// 一次调用解释多个字段
fields := []FieldContext{
    {TableName: "Department", ColumnName: "cDepCode"},
    {TableName: "Department", ColumnName: "cDepName"},
    // ... 最多 20 个
}
explanations := aiClient.BatchExplain(fields)
```

### 2. 缓存结果

```go
// 缓存 AI 解释，避免重复调用
cache.Set("Department.cDepCode", explanation)
```

### 3. 选择合适的模型

- **qwen-turbo**: 快速、便宜，适合大批量
- **qwen-plus**: 平衡性能和成本（推荐）
- **qwen-max**: 最准确，适合关键字段

## 故障处理

### AI 调用失败

工具会自动降级到纯算法模式：

```
⚠️  AI 解释失败: API 超时，继续使用算法
✓ 使用关系推断生成字段说明
```

### API Key 无效

```
⚠️  未提供 API Key，跳过 AI 分析
   提示：使用 --ai-key 或设置环境变量 DASHSCOPE_API_KEY
```

### 配额用尽

```
⚠️  API 调用失败: 配额不足
   已处理 150/700 个字段
   建议：升级套餐或稍后重试
```

## 扩展其他 AI 服务

### OpenAI

```go
type OpenAIClient struct {
    apiKey string
}

func (c *OpenAIClient) ExplainStandardField(...) {
    // 调用 OpenAI API
    // endpoint: https://api.openai.com/v1/chat/completions
}
```

### 本地模型

```go
type LocalLLMClient struct {
    endpoint string // http://localhost:11434
}

func (c *LocalLLMClient) ExplainStandardField(...) {
    // 调用本地 Ollama/LM Studio
}
```

## 最佳实践

1. **先运行不带 AI 的分析**，了解数据库结构
2. **启用 AI 增强**，获取字段解释
3. **人工审核**高价值表的 AI 解释
4. **导出结果**作为数据字典文档
5. **定期更新**，跟踪结构变化

## 隐私和安全

### 数据脱敏

AI 只接收：
- 表名、字段名
- 数据类型
- 统计摘要（null 率、唯一值率）

**不会发送**：
- 实际数据值
- 敏感业务数据

### 本地优先

如果担心数据安全，可以：
1. 使用本地 LLM（Ollama）
2. 部署私有化 AI 服务
3. 只对非敏感字段启用 AI

## 示例脚本

完整示例见：
- `examples/u8_ai_example.sh` - U8 AI 分析
- `internal/ai/client.go` - AI 客户端实现
- `internal/analyzer/hybrid.go` - 混合分析器

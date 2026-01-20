package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"schema-analyzer/internal/adapter"
)

// Client AI 客户端接口
type Client interface {
	// ExplainStandardField 解释标准字段（U8 通用字段）
	ExplainStandardField(tableName, columnName, dataType string) (*FieldExplanation, error)
	
	// InferCustomField 推断自定义字段（基于关联关系）
	InferCustomField(columnName string, relatedFields []RelatedField) (*FieldExplanation, error)
	
	// BatchExplain 批量解释（提高效率）
	BatchExplain(fields []FieldContext) (map[string]*FieldExplanation, error)
}

// FieldExplanation 字段解释
type FieldExplanation struct {
	ColumnName   string  `json:"column_name"`
	ChineseName  string  `json:"chinese_name"`   // 中文名称
	Description  string  `json:"description"`    // 详细说明
	BusinessMeaning string `json:"business_meaning"` // 业务含义
	Confidence   float64 `json:"confidence"`     // 置信度
	Source       string  `json:"source"`         // 来源：ai_standard/ai_inferred/relation
}

// FieldContext 字段上下文
type FieldContext struct {
	TableName  string
	ColumnName string
	DataType   string
	Stats      *adapter.ColumnStats
}

// RelatedField 关联字段（用于推断自定义字段）
type RelatedField struct {
	TableName   string
	ColumnName  string
	ChineseName string  // 已知的中文名
	Relation    string  // 关系类型：value_match/naming_similar
	Confidence  float64
}

// AlibabaClient 阿里云通义千问客户端
type AlibabaClient struct {
	apiKey    string
	endpoint  string
	model     string
	httpClient *http.Client
}

// NewAlibabaClient 创建阿里云 AI 客户端
func NewAlibabaClient(apiKey string) *AlibabaClient {
	return &AlibabaClient{
		apiKey:   apiKey,
		endpoint: "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation",
		model:    "qwen-plus", // 或 qwen-turbo, qwen-max
		httpClient: &http.Client{},
	}
}

// ExplainStandardField 解释 U8 标准字段
func (c *AlibabaClient) ExplainStandardField(tableName, columnName, dataType string) (*FieldExplanation, error) {
	// 构建 prompt
	prompt := fmt.Sprintf(`你是用友 U8 ERP 系统的数据库专家。请解释以下字段：

表名: %s
字段名: %s
数据类型: %s

请以 JSON 格式返回：
{
  "chinese_name": "字段的中文名称（5字以内）",
  "description": "字段的详细说明（20字以内）",
  "business_meaning": "业务含义（30字以内）",
  "confidence": 0.95
}

注意：
1. 只返回 JSON，不要其他文字
2. 如果不确定，confidence 设为 0.5 以下
3. 基于 U8 标准字段命名规范`, tableName, columnName, dataType)

	response, err := c.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	var explanation FieldExplanation
	if err := json.Unmarshal([]byte(response), &explanation); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %v", err)
	}

	explanation.ColumnName = columnName
	explanation.Source = "ai_standard"
	
	return &explanation, nil
}

// InferCustomField 推断自定义字段（cFree1-10, cDefine1-37）
func (c *AlibabaClient) InferCustomField(columnName string, relatedFields []RelatedField) (*FieldExplanation, error) {
	// 构建关联信息
	relationsDesc := ""
	for _, rf := range relatedFields {
		relationsDesc += fmt.Sprintf("- 与 %s.%s (%s) 关联，置信度 %.2f\n", 
			rf.TableName, rf.ColumnName, rf.ChineseName, rf.Confidence)
	}

	prompt := fmt.Sprintf(`你是用友 U8 ERP 系统的数据库专家。

字段 %s 是一个自定义字段（cFree 或 cDefine），需要基于关联关系推断其含义。

已知关联关系：
%s

请基于这些关联关系，推断该字段的业务含义，以 JSON 格式返回：
{
  "chinese_name": "推断的中文名称（5字以内）",
  "description": "推断的说明（20字以内）",
  "business_meaning": "基于关联关系的业务含义（30字以内）",
  "confidence": 0.75
}

注意：
1. 只返回 JSON
2. confidence 应该低于标准字段（0.5-0.8）
3. 说明中要提到"基于关联推断"`, columnName, relationsDesc)

	response, err := c.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	var explanation FieldExplanation
	if err := json.Unmarshal([]byte(response), &explanation); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %v", err)
	}

	explanation.ColumnName = columnName
	explanation.Source = "ai_inferred"
	
	return &explanation, nil
}

// BatchExplain 批量解释（提高效率）
func (c *AlibabaClient) BatchExplain(fields []FieldContext) (map[string]*FieldExplanation, error) {
	// 构建批量 prompt
	fieldsDesc := ""
	for i, f := range fields {
		fieldsDesc += fmt.Sprintf("%d. 表: %s, 字段: %s, 类型: %s\n", 
			i+1, f.TableName, f.ColumnName, f.DataType)
	}

	prompt := fmt.Sprintf(`你是用友 U8 ERP 系统的数据库专家。请批量解释以下字段：

%s

请以 JSON 数组格式返回，每个字段一个对象：
[
  {
    "column_name": "字段名",
    "chinese_name": "中文名称",
    "description": "说明",
    "business_meaning": "业务含义",
    "confidence": 0.95
  }
]

只返回 JSON 数组，不要其他文字。`, fieldsDesc)

	response, err := c.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	var explanations []FieldExplanation
	if err := json.Unmarshal([]byte(response), &explanations); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %v", err)
	}

	result := make(map[string]*FieldExplanation)
	for i := range explanations {
		explanations[i].Source = "ai_standard"
		result[explanations[i].ColumnName] = &explanations[i]
	}

	return result, nil
}

// callAPI 调用阿里云 API
func (c *AlibabaClient) callAPI(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.model,
		"input": map[string]interface{}{
			"messages": []map[string]string{
				{
					"role":    "system",
					"content": "你是用友 U8 ERP 系统的数据库专家，精通 U8 的表结构和字段命名规范。",
				},
				{
					"role":    "user",
					"content": prompt,
				},
			},
		},
		"parameters": map[string]interface{}{
			"result_format": "message",
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 调用失败: %s, 响应: %s", resp.Status, string(body))
	}

	// 解析响应
	var apiResp struct {
		Output struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if len(apiResp.Output.Choices) == 0 {
		return "", fmt.Errorf("API 返回空响应")
	}

	return apiResp.Output.Choices[0].Message.Content, nil
}

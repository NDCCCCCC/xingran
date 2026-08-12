package rpa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// rpaAIClientDefaultTimeout RPA AI 客户端默认请求超时
const rpaAIClientDefaultTimeout = 30 * time.Second

// AIClient AI API 客户端（共享）
type AIClient struct {
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	timeout   time.Duration
}

// NewAIClient 创建 AI 客户端
func NewAIClient(baseURL, apiKey, model string, maxTokens int, timeout time.Duration) *AIClient {
	return &AIClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		timeout:   timeout,
	}
}

// Message 聊天消息
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// Call 调用 OpenAI 兼容 API
func (c *AIClient) Call(ctx context.Context, messages []Message) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.model,
		"messages":   messages,
		"max_tokens": c.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建带超时的请求
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("无效的 API 响应：缺少 choices")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("无效的响应内容")
	}

	return content, nil
}

// ConvertToMessages 转换为消息格式（兼容旧代码）
func ConvertToMessages(messages []map[string]interface{}) []Message {
	result := make([]Message, len(messages))
	for i, m := range messages {
		result[i] = Message{
			Role:    m["role"].(string),
			Content: m["content"],
		}
	}
	return result
}

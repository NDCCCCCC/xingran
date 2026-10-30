package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// ErrorAnalyzer 错误分析服务接口
type ErrorAnalyzer interface {
	AnalyzeFailure(ctx context.Context, req *AnalyzeFailureRequest) (*FailureAnalysisResult, error)
	SuggestFix(ctx context.Context, req *SuggestFixRequest) (*FixAction, error)
	ClassifyError(ctx context.Context, errorMessage string) (*ErrorClassification, error)
}

// errorAnalyzerImpl 错误分析服务实现
type errorAnalyzerImpl struct {
	config          *config.Config
	agentClient     *AIClient
	generatorClient *AIClient
}

// NewErrorAnalyzer 创建错误分析服务
func NewErrorAnalyzer(cfg *config.Config) ErrorAnalyzer {
	agentClient := NewAIClient(
		cfg.RPA.AI.Agent.BaseURL,
		cfg.RPA.AI.Agent.APIKey,
		cfg.RPA.AI.Agent.Model,
		cfg.RPA.AI.Agent.MaxTokens,
		30*time.Second,
	)

	generatorClient := NewAIClient(
		cfg.RPA.AI.Generator.BaseURL,
		cfg.RPA.AI.Generator.APIKey,
		cfg.RPA.AI.Generator.Model,
		cfg.RPA.AI.Generator.MaxTokens,
		30*time.Second,
	)

	return &errorAnalyzerImpl{
		config:          cfg,
		agentClient:     agentClient,
		generatorClient: generatorClient,
	}
}

// FailureAnalysisResult 失败分析结果
type FailureAnalysisResult struct {
	ErrorType        string   `json:"errorType"`        // 错误类型：selector_timeout, element_not_found, network_error等
	Severity         string   `json:"severity"`         // 严重程度：critical, high, medium, low
	RootCause        string   `json:"rootCause"`        // 根本原因
	SuggestedActions []string `json:"suggestedActions"` // 建议的修复操作列表
	PreventionTips   []string `json:"preventionTips"`   // 预防建议
	CanAutoFix       bool     `json:"canAutoFix"`       // 是否可以自动修复
	Confidence       float64  `json:"confidence"`       // 分析置信度
}

// SuggestFixRequest 请求修复建议
type SuggestFixRequest struct {
	TaskDescription   string        `json:"taskDescription" binding:"required"`
	CurrentStep       int           `json:"currentStep"`
	OriginalAction    interface{}   `json:"originalAction"`
	ErrorMessage      string        `json:"errorMessage" binding:"required"`
	Selector          string        `json:"selector,omitempty"`
	ScreenshotBase64  string        `json:"screenshotBase64,omitempty"`
	HTMLSnippet       string        `json:"htmlSnippet,omitempty"`
	AvailableElements []interface{} `json:"availableElements,omitempty"`
	PageURL           string        `json:"pageUrl,omitempty"`
}

// ErrorClassification 错误分类
type ErrorClassification struct {
	Category    string `json:"category"`    // 错误分类：selector, timing, network, content, logic
	SubCategory string `json:"subCategory"` // 子分类
	Recoverable bool   `json:"recoverable"` // 是否可恢复
	Suggestion  string `json:"suggestion"`  // 修复建议
}

// AnalyzeFailure 分析失败原因
func (a *errorAnalyzerImpl) AnalyzeFailure(ctx context.Context, req *AnalyzeFailureRequest) (*FailureAnalysisResult, error) {
	if !a.config.RPA.AI.Agent.Enabled {
		return nil, fmt.Errorf("AI 错误分析未启用")
	}

	// 先进行本地快速分类
	classification := a.classifyErrorLocally(req.ErrorMessage)
	result := &FailureAnalysisResult{
		ErrorType:  classification.Category,
		Confidence: 0.7,
	}

	// 构建详细的 AI 提示词
	prompt := a.buildAnalysisPrompt(req, classification)

	messages := []Message{
		{
			Role: "system",
			Content: "你是 RPA 错误分析专家。分析执行失败的原因并提供详细的修复建议。返回 JSON 格式：\n" +
				"{\n" +
				"  \"errorType\": \"selector_timeout\",\n" +
				"  \"severity\": \"high\",\n" +
				"  \"rootCause\": \"元素选择器不唯一，页面加载时动态ID变化\",\n" +
				"  \"suggestedActions\": [\"使用更稳定的选择器如data-testid\", \"增加显式等待\"],\n" +
				"  \"preventionTips\": [\"使用稳定属性选择器\", \"添加加载检测\"],\n" +
				"  \"canAutoFix\": true,\n" +
				"  \"confidence\": 0.95\n" +
				"}",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// 调用 AI 进行深度分析
	if req.ScreenshotBase64 != "" || req.HTMLSnippet != "" {
		// 有视觉信息时使用 Agent 模型
		resp, err := a.agentClient.Call(ctx, messages)
		if err == nil {
			if err := json.Unmarshal([]byte(resp), result); err == nil {
				return result, nil
			}
		}
	}

	// 降级使用 Generator 模型
	resp, err := a.generatorClient.Call(ctx, messages)
	if err != nil {
		// AI 调用失败，返回本地分类结果
		a.enhanceLocalClassification(result, classification)
		return result, nil
	}

	if err := json.Unmarshal([]byte(resp), result); err != nil {
		a.enhanceLocalClassification(result, classification)
		return result, nil
	}

	return result, nil
}

// SuggestFix 提供修复建议
func (a *errorAnalyzerImpl) SuggestFix(ctx context.Context, req *SuggestFixRequest) (*FixAction, error) {
	if !a.config.RPA.AI.Agent.Enabled {
		return nil, fmt.Errorf("AI 修复建议未启用")
	}

	prompt := a.buildFixPrompt(req)

	messages := []Message{
		{
			Role: "system",
			Content: "你是 RPA 修复专家。分析失败的操作并提供修复后的版本。返回 JSON 格式：\n" +
				"{\n" +
				"  \"originalAction\": {...},\n" +
				"  \"fixedAction\": {...},\n" +
				"  \"reason\": \"修改原因说明\",\n" +
				"  \"confidence\": 0.95\n" +
				"}",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	var resp string
	var err error

	// 如果有截图，使用视觉模型
	if req.ScreenshotBase64 != "" {
		resp, err = a.agentClient.Call(ctx, messages)
	} else {
		resp, err = a.generatorClient.Call(ctx, messages)
	}

	if err != nil {
		return nil, fmt.Errorf("AI 请求失败: %w", err)
	}

	var result FixAction
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %w", err)
	}

	return &result, nil
}

// ClassifyError 分类错误（轻量级，不调用 AI）
func (a *errorAnalyzerImpl) ClassifyError(ctx context.Context, errorMessage string) (*ErrorClassification, error) {
	classification := a.classifyErrorLocally(errorMessage)
	return &classification, nil
}

// classifyErrorLocally 本地错误分类
func (a *errorAnalyzerImpl) classifyErrorLocally(errorMessage string) ErrorClassification {
	msg := strings.ToLower(errorMessage)

	classification := ErrorClassification{
		Category:    "unknown",
		SubCategory: "unknown",
		Recoverable: true,
	}

	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "timed out"):
		classification.Category = "timing"
		classification.SubCategory = "timeout"
		classification.Suggestion = "增加等待时间或使用显式等待"
		classification.Recoverable = true

	case strings.Contains(msg, "not found") || strings.Contains(msg, "no element") || strings.Contains(msg, "selector"):
		classification.Category = "selector"
		classification.SubCategory = "element_not_found"
		classification.Suggestion = "检查选择器是否正确，或使用更稳定的选择器"
		classification.Recoverable = true

	case strings.Contains(msg, "network") || strings.Contains(msg, "connection") || strings.Contains(msg, "econnrefused"):
		classification.Category = "network"
		classification.SubCategory = "connection_error"
		classification.Suggestion = "检查网络连接，添加重试机制"
		classification.Recoverable = true

	case strings.Contains(msg, "blocked") || strings.Contains(msg, "captcha") || strings.Contains(msg, "rate limit"):
		classification.Category = "network"
		classification.SubCategory = "access_denied"
		classification.Suggestion = "需要处理验证码或降低请求频率"
		classification.Recoverable = false

	case strings.Contains(msg, "invalid") && strings.Contains(msg, "value"):
		classification.Category = "content"
		classification.SubCategory = "invalid_input"
		classification.Suggestion = "检查输入数据的格式和内容"
		classification.Recoverable = true

	case strings.Contains(msg, "permission") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "unauthorized"):
		classification.Category = "logic"
		classification.SubCategory = "access_denied"
		classification.Suggestion = "检查权限配置或登录状态"
		classification.Recoverable = false

	default:
		classification.Suggestion = "需要人工介入分析"
		classification.Recoverable = false
	}

	return classification
}

// enhanceLocalClassification 增强本地分类结果
func (a *errorAnalyzerImpl) enhanceLocalClassification(result *FailureAnalysisResult, classification ErrorClassification) {
	result.RootCause = a.getDetailedRootCause(classification)
	result.SuggestedActions = []string{classification.Suggestion}
	result.Severity = a.getSeverityByCategory(classification.Category)
	result.CanAutoFix = classification.Recoverable
}

// getDetailedRootCause 获取详细的根本原因
func (a *errorAnalyzerImpl) getDetailedRootCause(classification ErrorClassification) string {
	reasons := map[string]string{
		"timing_timeout":             "页面加载或元素响应时间超过预设阈值",
		"selector_element_not_found": "指定的元素未能在页面中找到，可能原因：选择器错误、页面结构变化、元素未加载",
		"network_connection_error":   "无法建立网络连接，可能是网络问题或服务器不可达",
		"network_access_denied":      "访问被拒绝，可能由于IP被封、验证码、请求频率限制",
		"content_invalid_input":      "提交的数据格式或内容不符合预期",
		"logic_access_denied":        "权限不足或未登录，无法执行操作",
	}

	key := classification.Category + "_" + classification.SubCategory
	if reason, ok := reasons[key]; ok {
		return reason
	}
	return "需要进一步分析确定根本原因"
}

// getSeverityByCategory 根据分类获取严重程度
func (a *errorAnalyzerImpl) getSeverityByCategory(category string) string {
	severities := map[string]string{
		"timing":   "medium",
		"selector": "high",
		"network":  "high",
		"content":  "medium",
		"logic":    "critical",
		"unknown":  "medium",
	}

	if severity, ok := severities[category]; ok {
		return severity
	}
	return "medium"
}

// buildAnalysisPrompt 构建分析提示词
func (a *errorAnalyzerImpl) buildAnalysisPrompt(req *AnalyzeFailureRequest, classification ErrorClassification) string {
	var prompt strings.Builder

	prompt.WriteString("请分析以下 RPA 执行失败的情况：\n\n")
	prompt.WriteString(fmt.Sprintf("任务描述：%s\n", req.TaskDescription))
	prompt.WriteString(fmt.Sprintf("当前步骤：%d\n", req.CurrentStep))
	prompt.WriteString(fmt.Sprintf("错误信息：%s\n", req.ErrorMessage))

	if req.FailedAction != nil {
		actionJSON, _ := json.Marshal(req.FailedAction)
		prompt.WriteString(fmt.Sprintf("失败的操作：%s\n", string(actionJSON)))
	}

	if req.ScreenshotBase64 != "" {
		prompt.WriteString("[已附上页面截图]\n")
	}

	if req.HTMLSnippet != "" {
		prompt.WriteString(fmt.Sprintf("\n页面 HTML 片段：\n%s\n", req.HTMLSnippet))
	}

	prompt.WriteString(fmt.Sprintf("\n初步分类：%s - %s\n", classification.Category, classification.SubCategory))
	prompt.WriteString("\n请提供详细的分析和修复建议。")

	return prompt.String()
}

// buildFixPrompt 构建修复提示词
func (a *errorAnalyzerImpl) buildFixPrompt(req *SuggestFixRequest) string {
	var prompt strings.Builder

	prompt.WriteString("请提供以下操作的修复版本：\n\n")
	prompt.WriteString(fmt.Sprintf("任务描述：%s\n", req.TaskDescription))
	prompt.WriteString(fmt.Sprintf("当前步骤：%d\n", req.CurrentStep))

	originalJSON, _ := json.Marshal(req.OriginalAction)
	prompt.WriteString(fmt.Sprintf("原始操作：%s\n", string(originalJSON)))
	prompt.WriteString(fmt.Sprintf("错误信息：%s\n", req.ErrorMessage))

	if req.Selector != "" {
		prompt.WriteString(fmt.Sprintf("失败的选择器：%s\n", req.Selector))
	}

	if req.ScreenshotBase64 != "" {
		prompt.WriteString("[已附上页面截图]\n")
	}

	if req.HTMLSnippet != "" {
		prompt.WriteString(fmt.Sprintf("\n页面 HTML 片段：\n%s\n", req.HTMLSnippet))
	}

	if len(req.AvailableElements) > 0 {
		elementsJSON, _ := json.Marshal(req.AvailableElements)
		prompt.WriteString(fmt.Sprintf("\n可用的页面元素：%s\n", string(elementsJSON)))
	}

	if req.PageURL != "" {
		prompt.WriteString(fmt.Sprintf("\n页面 URL：%s\n", req.PageURL))
	}

	prompt.WriteString("\n请提供修复后的操作和详细的修改原因。")

	return prompt.String()
}

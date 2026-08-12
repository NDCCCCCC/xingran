package rpa

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// AIService AI 辅助服务接口
type AIService interface {
	// 脚本生成与优化
	GenerateScript(ctx context.Context, req *AIScriptGenerateRequest) (*AIScriptGenerateResponse, error)
	OptimizeScript(ctx context.Context, req *AIScriptOptimizeRequest) (*AIScriptOptimizeResponse, error)

	// AI Agent 决策
	DecideNextAction(ctx context.Context, req *AIAgentDecisionRequest) (*AIAgentAction, error)

	// 错误分析
	AnalyzeFailure(ctx context.Context, req *AnalyzeFailureRequest) (*FailureAnalysisResult, error)
	SuggestFix(ctx context.Context, req *SuggestFixRequest) (*FixAction, error)
	ClassifyError(ctx context.Context, errorMessage string) (*ErrorClassification, error)

	// 选择器学习
	RecordSelectorSuccess(ctx context.Context, record *SelectorSuccessRecord) error
	RecordSelectorFailure(ctx context.Context, record *SelectorFailureRecord) error
	GetBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error)
	ScoreSelector(ctx context.Context, selector, pageURL string) (float64, error)
	GetSelectorAlternatives(ctx context.Context, selector, pageURL string) ([]string, error)
}

// AIScriptGenerateResponse AI 脚本生成响应
type AIScriptGenerateResponse struct {
	Actions     []json.RawMessage `json:"actions"`
	Explanation string            `json:"explanation"`
	Confidence  float64           `json:"confidence"`
}

// AIScriptOptimizeResponse AI 脚本优化响应
type AIScriptOptimizeResponse struct {
	Actions     []json.RawMessage `json:"actions"`
	Explanation string            `json:"explanation"`
	Changes     []string          `json:"changes"`
	Confidence  float64           `json:"confidence"`
}

// aiServiceImpl AI 服务实现
type aiServiceImpl struct {
	config          *config.Config
	generatorClient *AIClient
	agentClient     *AIClient
	errorAnalyzer   ErrorAnalyzer
	selectorLearner SelectorLearner
}

// NewAIService 创建 AI 服务
func NewAIService(cfg *config.Config, db *gorm.DB, cache cache.Cache) AIService {
	// 创建脚本生成客户端
	generatorClient := NewAIClient(
		cfg.RPA.AI.Generator.BaseURL,
		cfg.RPA.AI.Generator.APIKey,
		cfg.RPA.AI.Generator.Model,
		cfg.RPA.AI.Generator.MaxTokens,
		rpaAIClientDefaultTimeout,
	)

	// 创建 AI Agent 客户端
	agentClient := NewAIClient(
		cfg.RPA.AI.Agent.BaseURL,
		cfg.RPA.AI.Agent.APIKey,
		cfg.RPA.AI.Agent.Model,
		cfg.RPA.AI.Agent.MaxTokens,
		rpaAIClientDefaultTimeout,
	)

	return &aiServiceImpl{
		config:          cfg,
		generatorClient: generatorClient,
		agentClient:     agentClient,
		errorAnalyzer:   NewErrorAnalyzer(cfg),
		selectorLearner: NewSelectorLearner(db, cache, cfg),
	}
}

// GenerateScript 从描述生成脚本
func (s *aiServiceImpl) GenerateScript(ctx context.Context, req *AIScriptGenerateRequest) (*AIScriptGenerateResponse, error) {
	if !s.config.RPA.AI.Generator.Enabled {
		return nil, fmt.Errorf("AI 脚本生成未启用")
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个 RPA 脚本生成助手。根据用户描述生成 Playwright 脚本。返回 JSON 格式。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("请生成以下任务的 RPA 脚本：\n%s\n\n目标页面：%s", req.Description, req.URL),
		},
	}

	resp, err := s.generatorClient.Call(ctx, messages)
	if err != nil {
		return nil, err
	}

	var result AIScriptGenerateResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %w", err)
	}

	return &result, nil
}

// OptimizeScript 优化脚本
func (s *aiServiceImpl) OptimizeScript(ctx context.Context, req *AIScriptOptimizeRequest) (*AIScriptOptimizeResponse, error) {
	if !s.config.RPA.AI.Generator.Enabled {
		return nil, fmt.Errorf("AI 脚本优化未启用")
	}

	// 分析优化目标
	optimizationGoals := s.analyzeOptimizationGoals(req.Goals)
	prompt := s.buildOptimizationPrompt(req, optimizationGoals)

	scriptJSON, _ := json.Marshal(req.Script)

	messages := []Message{
		{
			Role: "system",
			Content: "你是一个 RPA 脚本优化助手。分析并优化提供的脚本。返回 JSON 格式：\n" +
				"{\n" +
				"  \"actions\": [...],\n" +
				"  \"explanation\": \"优化说明\",\n" +
				"  \"changes\": [\"具体修改列表\"],\n" +
				"  \"confidence\": 0.95\n" +
				"}",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("%s\n\n脚本内容：\n%s", prompt, string(scriptJSON)),
		},
	}

	resp, err := s.generatorClient.Call(ctx, messages)
	if err != nil {
		return nil, err
	}

	var result AIScriptOptimizeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %w", err)
	}

	return &result, nil
}

// analyzeOptimizationGoals 分析优化目标并生成详细指导
func (s *aiServiceImpl) analyzeOptimizationGoals(goals []string) map[string]string {
	goalMap := make(map[string]string)

	for _, goal := range goals {
		switch goal {
		case "speed", "performance", "性能":
			goalMap["speed"] = "移除不必要的等待，使用更高效的选择器，并行执行独立操作"
		case "reliability", "robustness", "可靠性":
			goalMap["reliability"] = "添加重试机制，使用更稳定的选择器，增加等待和断言"
		case "maintainability", "可维护性":
			goalMap["maintainability"] = "添加注释，简化复杂逻辑，提取复用组件"
		case "readability", "可读性":
			goalMap["readability"] = "优化变量命名，添加说明文档，规范动作顺序"
		default:
			goalMap[goal] = "针对该目标进行优化"
		}
	}

	// 默认优化目标
	if len(goalMap) == 0 {
		goalMap["default"] = "平衡性能、可靠性和可维护性的综合优化"
	}

	return goalMap
}

// buildOptimizationPrompt 构建优化提示词
func (s *aiServiceImpl) buildOptimizationPrompt(req *AIScriptOptimizeRequest, goals map[string]string) string {
	prompt := "请优化以下 RPA 脚本，优化目标如下：\n\n"

	for goal, description := range goals {
		prompt += fmt.Sprintf("- %s: %s\n", goal, description)
	}

	if req.Description != "" {
		prompt += fmt.Sprintf("\n脚本用途：%s\n", req.Description)
	}

	prompt += "\n请返回优化后的脚本和详细的修改说明。"

	return prompt
}

// DecideNextAction AI 决策下一步动作
func (s *aiServiceImpl) DecideNextAction(ctx context.Context, req *AIAgentDecisionRequest) (*AIAgentAction, error) {
	if !s.config.RPA.AI.Agent.Enabled {
		return nil, fmt.Errorf("AI Agent 未启用")
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "你是 RPA 执行助手，帮助解决页面操作问题。返回 JSON 格式。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("任务: %s\n当前步骤: %d\n失败选择器: %s\n最后错误: %s", req.TaskDescription, req.CurrentStep, req.FailedSelector, req.LastError),
		},
	}

	resp, err := s.agentClient.Call(ctx, messages)
	if err != nil {
		return nil, err
	}

	var action AIAgentAction
	if err := json.Unmarshal([]byte(resp), &action); err != nil {
		return nil, fmt.Errorf("解析 AI 响应失败: %w", err)
	}

	return &action, nil
}

// AnalyzeFailure 分析失败原因（委托给错误分析器）
func (s *aiServiceImpl) AnalyzeFailure(ctx context.Context, req *AnalyzeFailureRequest) (*FailureAnalysisResult, error) {
	return s.errorAnalyzer.AnalyzeFailure(ctx, req)
}

// SuggestFix 提供修复建议（委托给错误分析器）
func (s *aiServiceImpl) SuggestFix(ctx context.Context, req *SuggestFixRequest) (*FixAction, error) {
	return s.errorAnalyzer.SuggestFix(ctx, req)
}

// ClassifyError 分类错误（委托给错误分析器）
func (s *aiServiceImpl) ClassifyError(ctx context.Context, errorMessage string) (*ErrorClassification, error) {
	return s.errorAnalyzer.ClassifyError(ctx, errorMessage)
}

// RecordSelectorSuccess 记录选择器成功（委托给选择器学习器）
func (s *aiServiceImpl) RecordSelectorSuccess(ctx context.Context, record *SelectorSuccessRecord) error {
	return s.selectorLearner.RecordSuccess(ctx, record)
}

// RecordSelectorFailure 记录选择器失败（委托给选择器学习器）
func (s *aiServiceImpl) RecordSelectorFailure(ctx context.Context, record *SelectorFailureRecord) error {
	return s.selectorLearner.RecordFailure(ctx, record)
}

// GetBestSelector 获取最佳选择器（委托给选择器学习器）
func (s *aiServiceImpl) GetBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
	return s.selectorLearner.GetBestSelector(ctx, pageURL, elementID)
}

// ScoreSelector 对选择器进行评分（委托给选择器学习器）
func (s *aiServiceImpl) ScoreSelector(ctx context.Context, selector, pageURL string) (float64, error) {
	return s.selectorLearner.ScoreSelector(ctx, selector, pageURL)
}

// GetSelectorAlternatives 获取选择器的替代方案（委托给选择器学习器）
func (s *aiServiceImpl) GetSelectorAlternatives(ctx context.Context, selector, pageURL string) ([]string, error) {
	return s.selectorLearner.GetSelectorAlternatives(ctx, selector, pageURL)
}

package rpa

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
)

// AIHandler AI 处理器
type AIHandler struct {
	aiService rpa.AIService
	core      *core.Core
}

// NewAIHandler 创建 AI 处理器
func NewAIHandler(aiService rpa.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *AIHandler) WithCore(core *core.Core) *AIHandler {
	h.core = core
	return h
}

// GenerateScript 生成脚本
func (h *AIHandler) GenerateScript(c *gin.Context) {
	var req rpa.AIScriptGenerateRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.GenerateScript(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "脚本生成失败") {
		return
	}

	// RecordWithBody：prompt 可能含用户提供的密钥/令牌，统一屏蔽
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// OptimizeScript 优化脚本
func (h *AIHandler) OptimizeScript(c *gin.Context) {
	var req rpa.AIScriptOptimizeRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.OptimizeScript(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "脚本优化失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// Decide AI 决策下一步动作
func (h *AIHandler) Decide(c *gin.Context) {
	var req rpa.AIAgentDecisionRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.DecideNextAction(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "AI 决策失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// AnalyzeFailure 分析失败原因
func (h *AIHandler) AnalyzeFailure(c *gin.Context) {
	var req rpa.AnalyzeFailureRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.AnalyzeFailure(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "错误分析失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// SuggestFix 提供修复建议
func (h *AIHandler) SuggestFix(c *gin.Context) {
	var req rpa.SuggestFixRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.SuggestFix(c.Request.Context(), &req)
	if handleError(c, err, http.StatusInternalServerError, "修复建议失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// ClassifyError 分类错误
func (h *AIHandler) ClassifyError(c *gin.Context) {
	var req struct {
		ErrorMessage string `json:"errorMessage" binding:"required"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.ClassifyError(c.Request.Context(), req.ErrorMessage)
	if handleError(c, err, http.StatusInternalServerError, "错误分类失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	success(c, result)
}

// RecordSelectorSuccess 记录选择器成功
func (h *AIHandler) RecordSelectorSuccess(c *gin.Context) {
	var req rpa.SelectorSuccessRecord
	if !bindAndValidate(c, &req) {
		return
	}

	if handleError(c, h.aiService.RecordSelectorSuccess(c.Request.Context(), &req), http.StatusInternalServerError, "记录失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	successMsg(c, "记录成功")
}

// RecordSelectorFailure 记录选择器失败
func (h *AIHandler) RecordSelectorFailure(c *gin.Context) {
	var req rpa.SelectorFailureRecord
	if !bindAndValidate(c, &req) {
		return
	}

	if handleError(c, h.aiService.RecordSelectorFailure(c.Request.Context(), &req), http.StatusInternalServerError, "记录失败") {
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA AI", operlog.OperTypeOther)

	successMsg(c, "记录成功")
}

// GetBestSelector 获取最佳选择器
func (h *AIHandler) GetBestSelector(c *gin.Context) {
	var req struct {
		PageURL   string `json:"pageUrl" binding:"required"`
		ElementID string `json:"elementId" binding:"required"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.GetBestSelector(c.Request.Context(), req.PageURL, req.ElementID)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, result)
}

// ScoreSelector 对选择器进行评分
func (h *AIHandler) ScoreSelector(c *gin.Context) {
	var req struct {
		Selector string `json:"selector" binding:"required"`
		PageURL  string `json:"pageUrl" binding:"required"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.ScoreSelector(c.Request.Context(), req.Selector, req.PageURL)
	if handleError(c, err, http.StatusInternalServerError, "评分失败") {
		return
	}

	success(c, gin.H{"score": result})
}

// GetSelectorAlternatives 获取选择器的替代方案
func (h *AIHandler) GetSelectorAlternatives(c *gin.Context) {
	var req struct {
		Selector string `json:"selector" binding:"required"`
		PageURL  string `json:"pageUrl" binding:"required"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.aiService.GetSelectorAlternatives(c.Request.Context(), req.Selector, req.PageURL)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, gin.H{"alternatives": result})
}

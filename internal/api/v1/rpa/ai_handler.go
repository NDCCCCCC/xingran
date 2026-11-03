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
// @Summary AI生成RPA脚本
// @Description 由自然语言描述生成RPA自动化脚本
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "生成要求"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/generate [post]
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
// @Summary AI优化RPA脚本
// @Description 优化已有RPA自动化脚本
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "脚本与优化要求"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/optimize [post]
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
// @Summary AI决策
// @Description RPA流程中的 AI 决策端点
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "决策上下文"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/decide [post]
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
// @Summary AI分析执行失败
// @Description 分析RPA执行失败原因
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "失败上下文"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/analyze-failure [post]
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
// @Summary AI建议修复
// @Description 针对失败给出修复建议
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "失败上下文"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/suggest-fix [post]
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
// @Summary AI错误分类
// @Description 对RPA错误进行分类
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "错误信息"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/classify-error [post]
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
// @Summary 记录选择器成功
// @Description 记录元素选择器命中成功（选择器学习）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "选择器信息"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/selector/record-success [post]
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
// @Summary 记录选择器失败
// @Description 记录元素选择器命中失败（选择器学习）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "选择器信息"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/selector/record-failure [post]
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
// @Summary 获取最优选择器
// @Description 获取当前学习到的最优元素选择器
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "查询条件"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/selector/best [post]
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
// @Summary 选择器评分
// @Description 对候选元素选择器评分
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "候选选择器"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/selector/score [post]
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
// @Summary 获取备选选择器
// @Description 获取备选元素选择器列表
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "查询条件"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/ai/selector/alternatives [post]
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

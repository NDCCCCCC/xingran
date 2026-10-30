package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemRequests "github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type ColumnConfigHandler struct {
	service systemServices.ColumnConfigService
	core    *core.Core
}

func NewColumnConfigHandler(service systemServices.ColumnConfigService) *ColumnConfigHandler {
	return &ColumnConfigHandler{service: service}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
func (h *ColumnConfigHandler) WithCore(core *core.Core) *ColumnConfigHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// GetByPageKey 获取页面列配置
func (h *ColumnConfigHandler) GetByPageKey(c *gin.Context) {
	pageKey := c.Param("page_key")
	if pageKey == "" {
		response.Error(c, response.ErrBadRequest, "页面标识不能为空")
		return
	}

	// 从上下文获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized, "用户未认证")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, response.ErrUnauthorized, "无效的用户标识")
		return
	}

	config, err := h.service.GetByPageKey(c.Request.Context(), userIDStr, pageKey)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, config)
}

// Save 保存列配置
func (h *ColumnConfigHandler) Save(c *gin.Context) {
	var req systemRequests.ColumnConfigSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	// 从上下文获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized, "用户未认证")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, response.ErrUnauthorized, "无效的用户标识")
		return
	}

	if err := h.service.Save(c.Request.Context(), userIDStr, &req); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "列自定义配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "保存成功"})
}

// Reset 重置列配置
func (h *ColumnConfigHandler) Reset(c *gin.Context) {
	pageKey := c.Param("page_key")
	if pageKey == "" {
		response.Error(c, response.ErrBadRequest, "页面标识不能为空")
		return
	}

	// 从上下文获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized, "用户未认证")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, response.ErrUnauthorized, "无效的用户标识")
		return
	}

	if err := h.service.Reset(c.Request.Context(), userIDStr, pageKey); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "列自定义配置", OperTypeReset)

	response.Success(c, gin.H{"message": "重置成功"})
}

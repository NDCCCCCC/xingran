package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	systemRequests "github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
)

type ColumnConfigHandler struct {
	service systemServices.ColumnConfigService
}

func NewColumnConfigHandler(service systemServices.ColumnConfigService) *ColumnConfigHandler {
	return &ColumnConfigHandler{service: service}
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

	response.Success(c, gin.H{"message": "重置成功"})
}

package system

import (

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

// ConfigHandler 参数配置处理器
type ConfigHandler struct {
	service systemServices.ConfigService
}

// NewConfigHandler 创建参数配置处理器实例
func NewConfigHandler(service systemServices.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
}

// Create 创建参数配置
func (h *ConfigHandler) Create(c *gin.Context) {
	var req requests.ConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询参数配置列表
func (h *ConfigHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := requests.DefaultConfigListParams()

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["configName"].(string); ok && val != "" {
		params.ConfigName = &val
	}
	if val, ok := rawReq["configKey"].(string); ok && val != "" {
		params.ConfigKey = &val
	}
	if val, ok := rawReq["configType"].(string); ok && val != "" {
		params.ConfigType = &val
	}
	if val, ok := rawReq["beginTime"].(string); ok && val != "" {
		params.BeginTime = &val
	}
	if val, ok := rawReq["endTime"].(string); ok && val != "" {
		params.EndTime = &val
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取参数配置详情
func (h *ConfigHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("参数配置ID"))
		return
	}

	config, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, config)
}

// GetByKey 根据键获取参数配置
func (h *ConfigHandler) GetByKey(c *gin.Context) {
	configKey := c.Param("configKey")
	if configKey == "" {
		response.Error(c, apperrors.ParamMissing("配置键"))
		return
	}

	config, err := h.service.GetByKey(c.Request.Context(), configKey)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, config)
}

// Update 更新参数配置
func (h *ConfigHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("参数配置ID"))
		return
	}

	var req requests.ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除参数配置
func (h *ConfigHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("参数配置ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除参数配置
func (h *ConfigHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RefreshCache 刷新参数缓存
func (h *ConfigHandler) RefreshCache(c *gin.Context) {
	if err := h.service.RefreshCache(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "刷新缓存成功"})
}

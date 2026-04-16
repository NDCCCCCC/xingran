package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
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
// @Summary 创建参数配置
// @Description 创建新的系统参数配置
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body object{configName=string,configKey=string,configValue=string,configType=string,remark=string} true "配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs [post]
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
// @Summary 查询参数配置列表
// @Description 查询系统参数配置列表，支持多条件筛选和分页
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body object{configName=string,configKey=string,configType=string,beginTime=string,endTime=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/list [post]
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
// @Summary 获取参数配置详情
// @Description 根据配置ID获取参数配置详细信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/:id [post]
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
// @Summary 根据键获取参数配置
// @Description 根据配置键获取参数配置值
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param configKey path string true "配置键"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/key/:configKey [post]
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
// @Summary 更新参数配置
// @Description 更新参数配置信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{configName=string,configKey=string,configValue=string,configType=string,remark=string} true "配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/:id/update [post]
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
// @Summary 删除参数配置
// @Description 删除指定的参数配置
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/:id/delete [post]
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
// @Summary 批量删除参数配置
// @Description 批量删除多个参数配置
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "配置ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/batch [post]
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
// @Summary 刷新参数缓存
// @Description 刷新系统参数配置缓存
// @Tags 配置管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/configs/refresh-cache [post]
func (h *ConfigHandler) RefreshCache(c *gin.Context) {
	if err := h.service.RefreshCache(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "刷新缓存成功"})
}

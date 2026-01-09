package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ConfigListRequest 参数配置列表请求
type ConfigListRequest struct {
	ConfigName *string `json:"configName,omitempty"`
	ConfigKey  *string `json:"configKey,omitempty"`
	ConfigType *string `json:"configType,omitempty"`
	BeginTime  *string `json:"beginTime,omitempty"`
	EndTime    *string `json:"endTime,omitempty"`
	Current    int     `json:"current" binding:"min=1"`
	PageSize   int     `json:"pageSize" binding:"min=1,max=100"`
}

// ConfigCreateRequest 创建参数配置请求
type ConfigCreateRequest struct {
	ConfigName  string            `json:"configName" binding:"required,min=2,max=100"`
	ConfigKey   string            `json:"configKey" binding:"required,min=2,max=100"`
	ConfigValue string            `json:"configValue" binding:"max=500"`
	ConfigType  models.ConfigType `json:"configType" binding:"oneof=Y N"`
	IsSystem    int               `json:"isSystem" binding:"min=0,max=1,oneof=0 1"`
	Remark      *string           `json:"remark,omitempty"`
}

// ConfigUpdateRequest 更新参数配置请求
type ConfigUpdateRequest struct {
	ID          string            `json:"id,omitempty"`
	ConfigName  string            `json:"configName" binding:"required,min=2,max=100"`
	ConfigValue string            `json:"configValue" binding:"max=500"`
	ConfigType  models.ConfigType `json:"configType" binding:"oneof=Y N"`
	Remark      *string           `json:"remark,omitempty"`
}

// listConfigs 查询参数配置列表
// @Summary 查询参数配置列表
// @Description 分页查询参数配置列表
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param request body ConfigListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /configs/list [post]
func listConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := ConfigListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["configName"].(string); ok && val != "" {
			req.ConfigName = &val
		}
		if val, ok := rawReq["configKey"].(string); ok && val != "" {
			req.ConfigKey = &val
		}
		if val, ok := rawReq["configType"].(string); ok && val != "" {
			req.ConfigType = &val
		}
		if val, ok := rawReq["beginTime"].(string); ok && val != "" {
			req.BeginTime = &val
		}
		if val, ok := rawReq["endTime"].(string); ok && val != "" {
			req.EndTime = &val
		}

		db := core.DB.GetDB().Model(&models.Config{})

		if req.ConfigName != nil && *req.ConfigName != "" {
			db = db.Where("config_name LIKE ?", "%"+*req.ConfigName+"%")
		}
		if req.ConfigKey != nil && *req.ConfigKey != "" {
			db = db.Where("config_key LIKE ?", "%"+*req.ConfigKey+"%")
		}
		if req.ConfigType != nil && *req.ConfigType != "" {
			db = db.Where("config_type = ?", *req.ConfigType)
		}
		if req.BeginTime != nil && *req.BeginTime != "" {
			db = db.Where("created_at >= ?", *req.BeginTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("created_at <= ?", *req.EndTime)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询参数配置总数失败")
			return
		}

		var configs []models.Config
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Find(&configs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询参数配置列表失败")
			return
		}

		pageResp := response.PageResponse{
			List:     configs,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createConfig 创建参数配置
// @Summary 创建参数配置
// @Description 创建新参数配置
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param request body ConfigCreateRequest true "参数配置信息"
// @Success 200 {object} response.Response
// @Router /configs [post]
func createConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConfigCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查配置键是否已存在
		var existConfig models.Config
		if err := core.DB.GetDB().Where("config_key = ?", req.ConfigKey).First(&existConfig).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "配置键已存在")
			return
		}

		config := models.Config{
			ConfigName:  req.ConfigName,
			ConfigKey:   req.ConfigKey,
			ConfigValue: req.ConfigValue,
			ConfigType:  req.ConfigType,
			IsSystem:    models.ConfigIsSystem(req.IsSystem),
			Remark:      func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&config).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建参数配置失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"参数配置",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"id": config.ID})
	}
}

// getConfig 获取参数配置详情
// @Summary 获取参数配置详情
// @Description 根据ID获取参数配置详情
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param id path string true "参数配置ID"
// @Success 200 {object} response.Response{data=models.Config}
// @Router /configs/:id [post]
func getConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var config models.Config
		if err := core.DB.GetDB().Where("id = ?", id).First(&config).Error; err != nil {
			response.Error(c, response.ErrNotFound, "参数配置不存在")
			return
		}

		response.Success(c, config)
	}
}

// getConfigByKey 根据键获取参数配置
// @Summary 根据键获取参数配置
// @Description 根据配置键获取参数值
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param configKey path string true "配置键"
// @Success 200 {object} response.Response{data=models.Config}
// @Router /configs/key/:configKey [get]
func getConfigByKey(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		configKey := c.Param("configKey")
		if configKey == "" {
			response.Error(c, response.ErrBadRequest, "配置键不能为空")
			return
		}

		var config models.Config
		if err := core.DB.GetDB().Where("config_key = ?", configKey).First(&config).Error; err != nil {
			response.Error(c, response.ErrNotFound, "参数配置不存在")
			return
		}

		response.Success(c, config)
	}
}

// updateConfig 更新参数配置
// @Summary 更新参数配置
// @Description 更新参数配置信息
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param id path string true "参数配置ID"
// @Param request body ConfigUpdateRequest true "参数配置信息"
// @Success 200 {object} response.Response
// @Router /configs/:id/update [post]
func updateConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req ConfigUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var config models.Config
		if err := core.DB.GetDB().Where("id = ?", id).First(&config).Error; err != nil {
			response.Error(c, response.ErrNotFound, "参数配置不存在")
			return
		}

		// 系统内置参数不能修改键名
		if config.IsSystem == models.ConfigIsSystemYes && req.ConfigName != config.ConfigName {
			response.Error(c, response.ErrBadRequest, "系统内置参数不能修改")
			return
		}

		updates := map[string]interface{}{
			"config_name":  req.ConfigName,
			"config_value": req.ConfigValue,
			"config_type":  req.ConfigType,
		}
		if req.Remark != nil {
			updates["remark"] = req.Remark
		}

		if err := core.DB.GetDB().Model(&config).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新参数配置失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"参数配置",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// deleteConfig 删除参数配置
// @Summary 删除参数配置
// @Description 删除指定参数配置
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param id path string true "参数配置ID"
// @Success 200 {object} response.Response
// @Router /configs/:id/delete [post]
func deleteConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var config models.Config
		if err := core.DB.GetDB().Where("id = ?", id).First(&config).Error; err != nil {
			response.Error(c, response.ErrNotFound, "参数配置不存在")
			return
		}

		// 系统内置参数不能删除
		if config.IsSystem == models.ConfigIsSystemYes {
			response.Error(c, response.ErrBadRequest, "系统内置参数不能删除")
			return
		}

		if err := core.DB.GetDB().Delete(&config).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除参数配置失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"参数配置",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// batchDeleteConfigs 批量删除参数配置
// @Summary 批量删除参数配置
// @Description 批量删除多个参数配置
// @Tags 参数配置
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /configs/batch-delete [post]
func batchDeleteConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查是否有系统内置参数
		var systemCount int64
		if err := core.DB.GetDB().Model(&models.Config{}).Where("id IN ? AND is_system = 1", req.Ids).Count(&systemCount).Error; err == nil && systemCount > 0 {
			response.Error(c, response.ErrBadRequest, "系统内置参数不能删除")
			return
		}

		if err := core.DB.GetDB().Where("id IN ? AND is_system = 0", req.Ids).Delete(&models.Config{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除参数配置失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"参数配置",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, nil)
	}
}

// refreshConfigCache 刷新参数缓存
// @Summary 刷新参数缓存
// @Description 刷新系统参数缓存
// @Tags 参数配置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /configs/refresh-cache [post]
func refreshConfigCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现缓存刷新逻辑
		response.Success(c, gin.H{"message": "刷新缓存成功"})
	}
}

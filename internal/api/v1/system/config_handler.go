package system

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ConfigHandler 参数配置处理器
type ConfigHandler struct {
	service        systemServices.ConfigService
	captchaService *core.CaptchaService
	core           *core.Core
}

// NewConfigHandler 创建参数配置处理器实例
func NewConfigHandler(service systemServices.ConfigService, captchaService *core.CaptchaService) *ConfigHandler {
	return &ConfigHandler{
		service:        service,
		captchaService: captchaService,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// Phase 34 操作日志全模块覆盖新增，用于 operlog.Record 访问 core.OperLogService 与
// core.GetDB()。不改写 NewConfigHandler 双参构造器签名，避免破坏既有调用点。
func (h *ConfigHandler) WithCore(core *core.Core) *ConfigHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 参数配置统计(读操作,不记操作日志)
func (h *ConfigHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// isCaptchaConfig 检查配置键是否与验证码相关
func (h *ConfigHandler) isCaptchaConfig(configKey string) bool {
	captchaKeys := []string{
		"sys.account.captchaEnabled",
		"sys.account.captchaType",
		"sys.account.captchaExpireTime",
		"sys.account.captchaMaxAttempts",
		"sys.account.ipRateLimit",
		"sys.account.loginMaxRetry",
		"sys.account.loginLockTime",
		"sys.account.captchaBackgroundMode",
		"sys.account.captchaPieceShape",
		"sys.account.captchaDifficulty",
	}
	for _, key := range captchaKeys {
		if configKey == key {
			return true
		}
	}
	return false
}

// validateCaptchaConfigValue 校验验证码相关配置值的合法性。
// 当前覆盖 captchaEnabled（三态枚举）；非法值返回 400，阻止写入。
// 纵深防御的"写入校验"层：与 core/captcha.go 的读取兜底配合，
// 防止非法值（如 "off"）让登录页陷入"既不弹模态框也无输入框"的死状态。
func (h *ConfigHandler) validateCaptchaConfigValue(configKey, configValue string) error {
	if configKey != "sys.account.captchaEnabled" {
		return nil
	}
	switch configValue {
	case "disabled", "normal", "slider":
		return nil
	default:
		return apperrors.BadRequest(fmt.Sprintf("验证码开关值非法：%q（合法值：disabled / normal / slider）", configValue))
	}
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

	// 写入校验：验证码相关配置值合法性（纵深防御）
	if err := h.validateCaptchaConfigValue(req.ConfigKey, req.ConfigValue); err != nil {
		response.Error(c, err)
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeCreate)
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

	// DEBUG: 打印实际接收的分页参数
	applogger.WithFields(map[string]interface{}{
		"rawReq_pageSize": rawReq["pageSize"],
		"params_pageSize":  params.PageSize,
		"params_Current":   params.Current,
	}).Info("[DEBUG] Config List received pagination parameters")

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

	// 服务端排序参数（透传给 service.List → base.ApplySort 白名单）
	if val, ok := rawReq["orderByColumn"].(string); ok && val != "" {
		params.OrderByColumn = val
	}
	if val, ok := rawReq["isAsc"]; ok {
		if b, ok := val.(bool); ok {
			params.IsAsc = &b
		}
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
	// 写入校验：验证码相关配置值合法性（纵深防御）
	if err := h.validateCaptchaConfigValue(req.ConfigKey, req.ConfigValue); err != nil {
		response.Error(c, err)
		return
	}

	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// 热重载：如果更新的是验证码相关配置，立即重新加载
	config, err := h.service.GetByID(c.Request.Context(), id)
	if err == nil {
		// 请求加密开关的处理
		if config.ConfigKey == "sys.request.encryption.enabled" {
			// 立即刷新中间件缓存
			middleware.RefreshEncryptionConfigCache()

			applogger.WithFields(map[string]interface{}{
				"config_key":   config.ConfigKey,
				"config_value": config.ConfigValue,
			}).Info("请求加密配置已更新，中间件缓存已刷新")
		}

		// 检查是否是验证码相关配置
		if h.isCaptchaConfig(config.ConfigKey) {
			if h.captchaService != nil {
				if loadErr := h.captchaService.LoadConfig(c.Request.Context()); loadErr != nil {
					// 记录错误但不影响更新成功的响应
					// 配置已经保存到数据库，下次重启时会生效
				}
			}
		}
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeUpdate)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeDelete)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeBatch)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeClean)
	response.Success(c, gin.H{"message": "刷新缓存成功"})
}

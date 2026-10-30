package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// NotificationConfigHandler 通知配置处理器
type NotificationConfigHandler struct {
	emailConfigService     systemServices.EmailConfigService
	apiNotificationService systemServices.APINotificationConfigService
	emailSenderService     *services.EmailSenderService
	core                   *core.Core
}

// NewNotificationConfigHandler 创建通知配置处理器实例
func NewNotificationConfigHandler(
	emailConfigService systemServices.EmailConfigService,
	apiNotificationService systemServices.APINotificationConfigService,
	emailSenderService *services.EmailSenderService,
) *NotificationConfigHandler {
	return &NotificationConfigHandler{
		emailConfigService:     emailConfigService,
		apiNotificationService: apiNotificationService,
		emailSenderService:     emailSenderService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
func (h *NotificationConfigHandler) WithCore(core *core.Core) *NotificationConfigHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ============= 邮箱配置 Handler =============

// ListEmailConfigs 查询邮箱配置列表
// @Summary 查询邮箱配置列表
// @Description 查询邮箱配置列表，支持状态筛选和分页
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param request body object{status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email/list [post]
func (h *NotificationConfigHandler) ListEmailConfigs(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := systemServices.DefaultEmailConfigListParams()

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

	// 处理状态参数
	if val, ok := rawReq["status"]; ok {
		switch v := val.(type) {
		case float64:
			status := int(v)
			params.Status = &status
		case int:
			params.Status = &v
		}
	}

	result, err := h.emailConfigService.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 隐藏密码
	if configs, ok := result.List.([]systemServices.EmailConfigDTO); ok {
		for i := range configs {
			configs[i].Password = "******"
		}
		result.List = configs
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetEmailConfig 获取邮箱配置详情
// @Summary 获取邮箱配置详情
// @Description 根据配置ID获取邮箱配置详细信息
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email/:id [get]
func (h *NotificationConfigHandler) GetEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	config, err := h.emailConfigService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 隐藏密码
	config.Password = "******"

	response.Success(c, config)
}

// CreateEmailConfig 创建邮箱配置
// @Summary 创建邮箱配置
// @Description 创建新的邮箱配置
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param request body object{configName=string,host=string,port=int,username=string,password=string,fromEmail=string,fromName=string,encryption=string,status=int} true "邮箱配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email [post]
func (h *NotificationConfigHandler) CreateEmailConfig(c *gin.Context) {
	var req systemServices.EmailConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.emailConfigService.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// RecordWithBody so SMTP password in the request body is masked before
	// being persisted to sys_oper_log.oper_param (Phase 34 sensitive-masking).
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "通知配置", OperTypeCreate)

	response.Success(c, gin.H{"message": "创建成功"})
}

// UpdateEmailConfig 更新邮箱配置
// @Summary 更新邮箱配置
// @Description 更新邮箱配置信息
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{configName=string,host=string,port=int,username=string,password=string,fromEmail=string,fromName=string,encryption=string,status=int} true "邮箱配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email/:id/update [post]
func (h *NotificationConfigHandler) UpdateEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	var req systemServices.EmailConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.emailConfigService.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// RecordWithBody: SMTP password may be in the update payload — mask it.
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "通知配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteEmailConfig 删除邮箱配置
// @Summary 删除邮箱配置
// @Description 删除指定的邮箱配置
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email/:id/delete [post]
func (h *NotificationConfigHandler) DeleteEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	if err := h.emailConfigService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "通知配置", OperTypeDelete)

	response.Success(c, nil)
}

// TestEmailConfig 测试邮箱配置
// @Summary 测试邮箱配置
// @Description 测试邮箱配置是否正常工作
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{testTo=string} true "测试收件人地址"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/email/:id/test [post]
func (h *NotificationConfigHandler) TestEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	var req struct {
		TestTo string `json:"testTo" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	result := h.emailSenderService.TestEmailConfig(c.Request.Context(), id, req.TestTo)

	if result.Success {
		// TestEmailConfig sends an outbound test email — record the trigger
		// as Other (one-shot action, not CRUD). Only success is recorded;
		// failures are surfaced via response.Error below.
		recordOperLog(c, h.core, "通知配置", OperTypeOther)
		response.Success(c, gin.H{"message": "测试邮件已发送，请检查收件箱"})
	} else {
		errMsg := result.Message
		if result.Error != nil {
			errMsg += ": " + result.Error.Error()
		}
		response.Error(c, apperrors.InternalServerErrorWithMsg(errMsg))
	}
}

// ============= API通知配置 Handler =============

// ListAPINotificationConfigs 查询API通知配置列表
// @Summary 查询API通知配置列表
// @Description 查询API通知配置列表，支持配置类型和状态筛选
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param request body object{configType=string,status=int,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/api/list [post]
func (h *NotificationConfigHandler) ListAPINotificationConfigs(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := systemServices.DefaultAPINotificationConfigListParams()

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

	// 处理配置类型参数
	if val, ok := rawReq["configType"].(string); ok && val != "" {
		params.ConfigType = &val
	}

	// 处理状态参数
	if val, ok := rawReq["status"]; ok {
		switch v := val.(type) {
		case float64:
			status := int(v)
			params.Status = &status
		case int:
			params.Status = &v
		}
	}

	result, err := h.apiNotificationService.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetAPINotificationConfig 获取API通知配置详情
// @Summary 获取API通知配置详情
// @Description 根据配置ID获取API通知配置详细信息
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/api/:id [get]
func (h *NotificationConfigHandler) GetAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	config, err := h.apiNotificationService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, config)
}

// CreateAPINotificationConfig 创建API通知配置
// @Summary 创建API通知配置
// @Description 创建新的API通知配置
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param request body object{configName=string,configType=string,apiUrl=string,method=string,headers=object,bodyTemplate=string,status=int} true "API通知配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/api [post]
func (h *NotificationConfigHandler) CreateAPINotificationConfig(c *gin.Context) {
	var req systemServices.APINotificationConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.apiNotificationService.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// RecordWithBody: API headers/bodyTemplate may embed secrets (Bearer tokens,
	// HMAC keys). Mask them before persisting to sys_oper_log.oper_param.
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "通知配置", OperTypeCreate)

	response.Success(c, gin.H{"message": "创建成功"})
}

// UpdateAPINotificationConfig 更新API通知配置
// @Summary 更新API通知配置
// @Description 更新API通知配置信息
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body object{configName=string,configType=string,apiUrl=string,method=string,headers=object,bodyTemplate=string,status=int} true "API通知配置信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/api/:id/update [post]
func (h *NotificationConfigHandler) UpdateAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	var req systemServices.APINotificationConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.apiNotificationService.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// RecordWithBody: API headers/bodyTemplate may embed secrets — mask them.
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "通知配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteAPINotificationConfig 删除API通知配置
// @Summary 删除API通知配置
// @Description 删除指定的API通知配置
// @Tags 通知配置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notification-configs/api/:id/delete [post]
func (h *NotificationConfigHandler) DeleteAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("配置ID"))
		return
	}

	if err := h.apiNotificationService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "通知配置", OperTypeDelete)

	response.Success(c, nil)
}

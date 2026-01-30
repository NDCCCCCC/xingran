package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// NotificationConfigHandler 通知配置处理器
type NotificationConfigHandler struct {
	emailConfigService      systemServices.EmailConfigService
	apiNotificationService  systemServices.APINotificationConfigService
	emailSenderService      *services.EmailSenderService
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

// ============= 邮箱配置 Handler =============

// ListEmailConfigs 查询邮箱配置列表
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
		response.Error(c, http.StatusInternalServerError, err.Error())
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
func (h *NotificationConfigHandler) GetEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	config, err := h.emailConfigService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 隐藏密码
	config.Password = "******"

	response.Success(c, config)
}

// CreateEmailConfig 创建邮箱配置
func (h *NotificationConfigHandler) CreateEmailConfig(c *gin.Context) {
	var req systemServices.EmailConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.emailConfigService.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// UpdateEmailConfig 更新邮箱配置
func (h *NotificationConfigHandler) UpdateEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	var req systemServices.EmailConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req.ID = id
	if err := h.emailConfigService.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteEmailConfig 删除邮箱配置
func (h *NotificationConfigHandler) DeleteEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	if err := h.emailConfigService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// TestEmailConfig 测试邮箱配置
func (h *NotificationConfigHandler) TestEmailConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	var req struct {
		TestTo string `json:"testTo" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	result := h.emailSenderService.TestEmailConfig(c.Request.Context(), id, req.TestTo)

	if result.Success {
		response.Success(c, gin.H{"message": "测试邮件已发送，请检查收件箱"})
	} else {
		errMsg := result.Message
		if result.Error != nil {
			errMsg += ": " + result.Error.Error()
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
	}
}

// ============= API通知配置 Handler =============

// ListAPINotificationConfigs 查询API通知配置列表
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
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetAPINotificationConfig 获取API通知配置详情
func (h *NotificationConfigHandler) GetAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	config, err := h.apiNotificationService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, config)
}

// CreateAPINotificationConfig 创建API通知配置
func (h *NotificationConfigHandler) CreateAPINotificationConfig(c *gin.Context) {
	var req systemServices.APINotificationConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.apiNotificationService.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// UpdateAPINotificationConfig 更新API通知配置
func (h *NotificationConfigHandler) UpdateAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	var req systemServices.APINotificationConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req.ID = id
	if err := h.apiNotificationService.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteAPINotificationConfig 删除API通知配置
func (h *NotificationConfigHandler) DeleteAPINotificationConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "配置ID不能为空")
		return
	}

	if err := h.apiNotificationService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

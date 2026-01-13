package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ============= 邮箱配置 API =============

// EmailConfigListRequest 邮箱配置列表请求
type EmailConfigListRequest struct {
	Page     int  `json:"page" binding:"min=1"`
	PageSize int  `json:"pageSize" binding:"min=1,max=100"`
	Status   *int `json:"status"`
}

// EmailConfigCreateRequest 创建邮箱配置请求
type EmailConfigCreateRequest struct {
	ConfigName  string `json:"configName" binding:"required,min=2,max=100"`
	Host        string `json:"host" binding:"required,max=255"`
	Port        int    `json:"port" binding:"required,min=1,max=65535"`
	Username    string `json:"username" binding:"required,max=255"`
	Password    string `json:"password" binding:"required"`
	FromName    string `json:"fromName" binding:"max=100"`
	FromEmail   string `json:"fromEmail" binding:"email,max=255"`
	UseSSL      bool   `json:"useSsl"`
	UseSTARTTLS bool   `json:"useStartTls"`
	IsDefault   bool   `json:"isDefault"`
	Status      int    `json:"status" binding:"oneof=0 1"`
	Remark      string `json:"remark" binding:"max=500"`
}

// EmailConfigUpdateRequest 更新邮箱配置请求
type EmailConfigUpdateRequest struct {
	ConfigName  *string `json:"configName" binding:"omitempty,min=2,max=100"`
	Host        *string `json:"host" binding:"omitempty,max=255"`
	Port        *int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    *string `json:"username" binding:"omitempty,max=255"`
	Password    *string `json:"password" binding:"omitempty"`
	FromName    *string `json:"fromName" binding:"omitempty,max=100"`
	FromEmail   *string `json:"fromEmail" binding:"omitempty,max=255"` // 移除email验证，允许空字符串
	UseSSL      *bool   `json:"useSsl"`
	UseSTARTTLS *bool   `json:"useStartTls"`
	IsDefault   *bool   `json:"isDefault"`
	Status      *int    `json:"status" binding:"omitempty,oneof=0 1"`
	Remark      *string `json:"remark" binding:"omitempty,max=500"`
}

// listEmailConfigs 获取邮箱配置列表
// @Summary 获取邮箱配置列表
// @Description 分页获取邮箱配置列表
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body EmailConfigListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /settings/email-configs/list [post]
func listEmailConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req EmailConfigListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Page = 1
			req.PageSize = 10
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		configs, total, err := configService.ListEmailConfigs(c.Request.Context(), req.Page, req.PageSize, req.Status)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 隐藏密码
		for i := range configs {
			configs[i].Password = "******"
		}

		pageResp := response.PageResponse{
			List:     configs,
			Total:    total,
			Current:  req.Page,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getEmailConfig 获取邮箱配置详情
// @Summary 获取邮箱配置详情
// @Description 根据ID获取邮箱配置详情
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response{data=models.EmailConfig}
// @Router /settings/email-configs/:id [get]
func getEmailConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		config, err := configService.GetEmailConfigByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		// 隐藏密码
		config.Password = "******"

		response.Success(c, config)
	}
}

// createEmailConfig 创建邮箱配置
// @Summary 创建邮箱配置
// @Description 创建新的邮箱配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body EmailConfigCreateRequest true "邮箱配置"
// @Success 200 {object} response.Response
// @Router /settings/email-configs [post]
func createEmailConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req EmailConfigCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		// 获取当前用户
		userID, _ := c.Get("user_id")

		// 加密密码
		encryptedPassword, err := services.EncryptPassword(req.Password, "")
		if err != nil {
			response.Error(c, response.ErrServerError, "密码加密失败")
			return
		}

		config := &models.EmailConfig{
			ConfigName:  req.ConfigName,
			Host:        req.Host,
			Port:        req.Port,
			Username:    req.Username,
			Password:    encryptedPassword,
			FromName:    req.FromName,
			FromEmail:   req.FromEmail,
			UseSSL:      req.UseSSL,
			UseSTARTTLS: req.UseSTARTTLS,
			IsDefault:   req.IsDefault,
			Status:      req.Status,
			Remark:      req.Remark,
			CreatedBy:   userID.(string),
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		if err := configService.CreateEmailConfig(c.Request.Context(), config); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 隐藏密码后返回
		config.Password = "******"
		response.Success(c, config)
	}
}

// updateEmailConfig 更新邮箱配置
// @Summary 更新邮箱配置
// @Description 更新邮箱配置信息
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body EmailConfigUpdateRequest true "邮箱配置"
// @Success 200 {object} response.Response
// @Router /settings/email-configs/:id [put]
func updateEmailConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req EmailConfigUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 获取当前用户
		userID, _ := c.Get("user_id")

		// 获取现有配置
		configService := services.NewNotificationConfigService(core.DB.GetDB())
		existing, err := configService.GetEmailConfigByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		// 构建更新数据
		updates := &models.EmailConfig{
			UpdatedBy: userID.(string),
		}

		if req.ConfigName != nil {
			updates.ConfigName = *req.ConfigName
		}
		if req.Host != nil {
			updates.Host = *req.Host
		}
		if req.Port != nil {
			updates.Port = *req.Port
		}
		if req.Username != nil {
			updates.Username = *req.Username
		}
		if req.Password != nil {
			// 加密新密码
			encryptedPassword, err := services.EncryptPassword(*req.Password, "")
			if err != nil {
				response.Error(c, response.ErrServerError, "密码加密失败")
				return
			}
			updates.Password = encryptedPassword
		} else {
			updates.Password = existing.Password
		}
		if req.FromName != nil {
			updates.FromName = *req.FromName
		}
		if req.FromEmail != nil {
			updates.FromEmail = *req.FromEmail
		}
		if req.UseSSL != nil {
			updates.UseSSL = *req.UseSSL
		}
		if req.UseSTARTTLS != nil {
			updates.UseSTARTTLS = *req.UseSTARTTLS
		}
		if req.IsDefault != nil {
			updates.IsDefault = *req.IsDefault
		}
		if req.Status != nil {
			updates.Status = *req.Status
		}
		if req.Remark != nil {
			updates.Remark = *req.Remark
		}

		if err := configService.UpdateEmailConfig(c.Request.Context(), id, updates); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteEmailConfig 删除邮箱配置
// @Summary 删除邮箱配置
// @Description 删除指定的邮箱配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Router /settings/email-configs/:id [delete]
func deleteEmailConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		if err := configService.DeleteEmailConfig(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// testEmailConfigRequest 测试邮件配置请求
type testEmailConfigRequest struct {
	TestTo string `json:"testTo" binding:"required,email"`
}

// testEmailConfig 测试邮箱配置
// @Summary 测试邮箱配置
// @Description 发送测试邮件验证配置是否正确
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body testEmailConfigRequest true "测试收件人"
// @Success 200 {object} response.Response
// @Router /settings/email-configs/:id/test [post]
func testEmailConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req testEmailConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		emailSender := services.NewEmailSenderService(core.DB.GetDB())
		result := emailSender.TestEmailConfig(c.Request.Context(), id, req.TestTo)

		if result.Success {
			response.Success(c, gin.H{"message": "测试邮件已发送，请检查收件箱"})
		} else {
			// 返回详细的错误信息
			errMsg := result.Message
			if result.Error != nil {
				errMsg += ": " + result.Error.Error()
			}
			response.Error(c, response.ErrServerError, errMsg)
		}
	}
}

// ============= API通知配置 API =============

// APINotificationConfigListRequest API通知配置列表请求
type APINotificationConfigListRequest struct {
	Page       int                   `json:"page" binding:"min=1"`
	PageSize   int                   `json:"pageSize" binding:"min=1,max=100"`
	ConfigType *models.APIConfigType `json:"configType"`
	Status     *int                  `json:"status"`
}

// APINotificationConfigCreateRequest 创建API通知配置请求
type APINotificationConfigCreateRequest struct {
	ConfigName   string                `json:"configName" binding:"required,min=2,max=100"`
	ConfigType   models.APIConfigType  `json:"configType" binding:"required,oneof=sms webhook push"`
	APIURL       string                `json:"apiUrl" binding:"required,max=500"`
	APIMethod    string                `json:"apiMethod" binding:"required,oneof=GET POST"`
	Headers      map[string]interface{} `json:"headers"`
	TemplateBody string                `json:"templateBody"`
	AuthType     models.AuthType       `json:"authType" binding:"oneof=none basic bearer apikey"`
	AuthConfig   map[string]interface{} `json:"authConfig"`
	RetryCount   int                   `json:"retryCount" binding:"min=0,max=10"`
	Timeout      int                   `json:"timeout" binding:"min=1,max=300"`
	IsDefault    bool                  `json:"isDefault"`
	Status       int                   `json:"status" binding:"oneof=0 1"`
	Remark       string                `json:"remark" binding:"max=500"`
}

// APINotificationConfigUpdateRequest 更新API通知配置请求
type APINotificationConfigUpdateRequest struct {
	ConfigName   *string                `json:"configName" binding:"omitempty,min=2,max=100"`
	APIURL       *string                `json:"apiUrl" binding:"omitempty,max=500"`
	APIMethod    *string                `json:"apiMethod" binding:"omitempty,oneof=GET POST"`
	Headers      map[string]interface{} `json:"headers"`
	TemplateBody *string                `json:"templateBody"`
	AuthType     *models.AuthType       `json:"authType" binding:"omitempty,oneof=none basic bearer apikey"`
	AuthConfig   map[string]interface{} `json:"authConfig"`
	RetryCount   *int                   `json:"retryCount" binding:"omitempty,min=0,max=10"`
	Timeout      *int                   `json:"timeout" binding:"omitempty,min=1,max=300"`
	IsDefault    *bool                  `json:"isDefault"`
	Status       *int                   `json:"status" binding:"omitempty,oneof=0 1"`
	Remark       *string                `json:"remark" binding:"omitempty,max=500"`
}

// listAPINotificationConfigs 获取API通知配置列表
// @Summary 获取API通知配置列表
// @Description 分页获取API通知配置列表
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body APINotificationConfigListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /settings/api-notification-configs/list [post]
func listAPINotificationConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req APINotificationConfigListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Page = 1
			req.PageSize = 10
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		configs, total, err := configService.ListAPINotificationConfigs(c.Request.Context(), req.Page, req.PageSize, req.ConfigType, req.Status)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     configs,
			Total:    total,
			Current:  req.Page,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getAPINotificationConfig 获取API通知配置详情
// @Summary 获取API通知配置详情
// @Description 根据ID获取API通知配置详情
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response{data=models.APINotificationConfig}
// @Router /settings/api-notification-configs/:id [get]
func getAPINotificationConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		config, err := configService.GetAPINotificationConfigByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, config)
	}
}

// createAPINotificationConfig 创建API通知配置
// @Summary 创建API通知配置
// @Description 创建新的API通知配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body APINotificationConfigCreateRequest true "API通知配置"
// @Success 200 {object} response.Response
// @Router /settings/api-notification-configs [post]
func createAPINotificationConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req APINotificationConfigCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 获取当前用户
		userID, _ := c.Get("user_id")

		config := &models.APINotificationConfig{
			ConfigName:   req.ConfigName,
			ConfigType:   req.ConfigType,
			APIURL:       req.APIURL,
			APIMethod:    req.APIMethod,
			Headers:      req.Headers,
			TemplateBody: req.TemplateBody,
			AuthType:     req.AuthType,
			AuthConfig:   req.AuthConfig,
			RetryCount:   req.RetryCount,
			Timeout:      req.Timeout,
			IsDefault:    req.IsDefault,
			Status:       req.Status,
			Remark:       req.Remark,
			CreatedBy:    userID.(string),
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		if err := configService.CreateAPINotificationConfig(c.Request.Context(), config); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"id": config.ID, "message": "创建成功"})
	}
}

// updateAPINotificationConfig 更新API通知配置
// @Summary 更新API通知配置
// @Description 更新API通知配置信息
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Param request body APINotificationConfigUpdateRequest true "API通知配置"
// @Success 200 {object} response.Response
// @Router /settings/api-notification-configs/:id [put]
func updateAPINotificationConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req APINotificationConfigUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 获取当前用户
		userID, _ := c.Get("user_id")

		// 获取现有配置
		configService := services.NewNotificationConfigService(core.DB.GetDB())
		existing, err := configService.GetAPINotificationConfigByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		// 构建更新数据
		updates := &models.APINotificationConfig{
			UpdatedBy: userID.(string),
		}

		if req.ConfigName != nil {
			updates.ConfigName = *req.ConfigName
		}
		if req.APIURL != nil {
			updates.APIURL = *req.APIURL
		}
		if req.APIMethod != nil {
			updates.APIMethod = *req.APIMethod
		}
		if req.Headers != nil {
			updates.Headers = req.Headers
		} else {
			updates.Headers = existing.Headers
		}
		if req.TemplateBody != nil {
			updates.TemplateBody = *req.TemplateBody
		} else {
			updates.TemplateBody = existing.TemplateBody
		}
		if req.AuthType != nil {
			updates.AuthType = *req.AuthType
		}
		if req.AuthConfig != nil {
			updates.AuthConfig = req.AuthConfig
		} else {
			updates.AuthConfig = existing.AuthConfig
		}
		if req.RetryCount != nil {
			updates.RetryCount = *req.RetryCount
		}
		if req.Timeout != nil {
			updates.Timeout = *req.Timeout
		}
		if req.IsDefault != nil {
			updates.IsDefault = *req.IsDefault
		}
		if req.Status != nil {
			updates.Status = *req.Status
		}
		if req.Remark != nil {
			updates.Remark = *req.Remark
		}

		if err := configService.UpdateAPINotificationConfig(c.Request.Context(), id, updates); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteAPINotificationConfig 删除API通知配置
// @Summary 删除API通知配置
// @Description 删除指定的API通知配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Response
// @Router /settings/api-notification-configs/:id [delete]
func deleteAPINotificationConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		configService := services.NewNotificationConfigService(core.DB.GetDB())
		if err := configService.DeleteAPINotificationConfig(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

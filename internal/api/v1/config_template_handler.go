package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ConfigTemplateListRequest 配置模板列表请求
type ConfigTemplateListRequest struct {
	TemplateName *string                `json:"templateName,omitempty"`
	TemplateType *models.TemplateType   `json:"templateType,omitempty"`
	Vendor       *models.DeviceVendor  `json:"vendor,omitempty"`
	DeviceType   *models.DeviceType     `json:"deviceType,omitempty"`
	IsSystem     *bool                  `json:"isSystem,omitempty"`
	Current      int                    `json:"current" binding:"min=1"`
	PageSize     int                    `json:"pageSize" binding:"min=1,max=100"`
}

// ConfigTemplateCreateRequest 创建配置模板请求
type ConfigTemplateCreateRequest struct {
	TemplateName    string                     `json:"templateName" binding:"required,min=2,max=100"`
	TemplateCode    string                     `json:"templateCode" binding:"required,min=2,max=50"`
	TemplateType    models.TemplateType        `json:"templateType" binding:"required,oneof=init config backup"`
	Vendor          models.DeviceVendor        `json:"vendor" binding:"required,oneof=huawei h3c ruijie maipu"`
	DeviceType      models.DeviceType          `json:"deviceType" binding:"required,oneof=router switch firewall ap loadbalancer"`
	TemplateContent string                     `json:"templateContent" binding:"required"`
	Variables       []models.TemplateVariable  `json:"variables,omitempty"`
	Description     string                     `json:"description,omitempty"`
	IsSystem        bool                       `json:"isSystem"`
}

// ConfigTemplateUpdateRequest 更新配置模板请求
type ConfigTemplateUpdateRequest struct {
	ID              string                     `json:"id" binding:"required"`
	TemplateName    string                     `json:"templateName" binding:"required,min=2,max=100"`
	TemplateType    models.TemplateType        `json:"templateType" binding:"required,oneof=init config backup"`
	Vendor          models.DeviceVendor        `json:"vendor" binding:"required,oneof=huawei h3c ruijie maipu"`
	DeviceType      models.DeviceType          `json:"deviceType" binding:"required,oneof=router switch firewall ap loadbalancer"`
	TemplateContent string                     `json:"templateContent" binding:"required"`
	Variables       []models.TemplateVariable  `json:"variables,omitempty"`
	Description     string                     `json:"description,omitempty"`
}

// listConfigTemplates 查询配置模板列表
// @Summary 查询配置模板列表
// @Description 分页查询配置模板列表
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body ConfigTemplateListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/templates/list [post]
func listConfigTemplates(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := ConfigTemplateListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["templateName"].(string); ok && val != "" {
			req.TemplateName = &val
		}
		if val, ok := rawReq["templateType"].(string); ok && val != "" {
			tt := models.TemplateType(val)
			req.TemplateType = &tt
		}
		if val, ok := rawReq["vendor"].(string); ok && val != "" {
			vendor := models.DeviceVendor(val)
			req.Vendor = &vendor
		}
		if val, ok := rawReq["deviceType"].(string); ok && val != "" {
			dt := models.DeviceType(val)
			req.DeviceType = &dt
		}
		if val, ok := rawReq["isSystem"].(bool); ok {
			req.IsSystem = &val
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		templates, total, err := svc.List(c.Request.Context(), &services.ListTemplateRequest{
			TemplateName: req.TemplateName,
			TemplateType: req.TemplateType,
			Vendor:       req.Vendor,
			DeviceType:   req.DeviceType,
			IsSystem:     req.IsSystem,
			Current:      req.Current,
			PageSize:     req.PageSize,
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     templates,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createConfigTemplate 创建配置模板
// @Summary 创建配置模板
// @Description 创建新配置模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body ConfigTemplateCreateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /network/templates [post]
func createConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConfigTemplateCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		template, err := svc.Create(c.Request.Context(), &services.CreateTemplateRequest{
			TemplateName:    req.TemplateName,
			TemplateCode:    req.TemplateCode,
			TemplateType:    req.TemplateType,
			Vendor:          req.Vendor,
			DeviceType:      req.DeviceType,
			TemplateContent: req.TemplateContent,
			Variables:       req.Variables,
			Description:     req.Description,
			IsSystem:        req.IsSystem,
			CreatedBy:       c.GetString("userId"),
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, template)
	}
}

// getConfigTemplate 获取模板详情
// @Summary 获取模板详情
// @Description 根据ID获取模板详情
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response{data=models.ConfigTemplate}
// @Router /network/templates/:id [post]
func getConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		template, err := svc.GetByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, "模板不存在")
			return
		}

		response.Success(c, template)
	}
}

// updateConfigTemplate 更新配置模板
// @Summary 更新配置模板
// @Description 更新模板信息
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body ConfigTemplateUpdateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/update [post]
func updateConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req ConfigTemplateUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}
		req.ID = id

		svc := services.NewTemplateService(core.DB.GetDB())
		if err := svc.Update(c.Request.Context(), &services.UpdateTemplateRequest{
			ID:              req.ID,
			TemplateName:    req.TemplateName,
			TemplateType:    req.TemplateType,
			Vendor:          req.Vendor,
			DeviceType:      req.DeviceType,
			TemplateContent: req.TemplateContent,
			Variables:       req.Variables,
			Description:     req.Description,
			UpdatedBy:       c.GetString("userId"),
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// deleteConfigTemplate 删除配置模板
// @Summary 删除配置模板
// @Description 删除指定模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/delete [post]
func deleteConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		if err := svc.Delete(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteConfigTemplates 批量删除模板
// @Summary 批量删除模板
// @Description 批量删除多个模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /network/templates/batch-delete [post]
func batchDeleteConfigTemplates(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		if err := svc.BatchDelete(c.Request.Context(), req.Ids); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// previewConfigTemplate 预览模板渲染
// @Summary 预览模板渲染
// @Description 使用示例变量预览模板渲染结果
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body map[string]string true "变量值"
// @Success 200 {object} response.Response{data=string}
// @Router /network/templates/:id/preview [post]
func previewConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var variables map[string]string
		if err := c.ShouldBindJSON(&variables); err != nil {
			variables = make(map[string]string)
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		result, err := svc.Preview(c.Request.Context(), id, variables)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"content": result})
	}
}

// cloneConfigTemplate 克隆模板
// @Summary 克隆模板
// @Description 克隆现有模板创建新模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body map[string]string true "新模板名称和编码"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/clone [post]
func cloneConfigTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req struct {
			NewName string `json:"newName" binding:"required,min=2,max=100"`
			NewCode string `json:"newCode" binding:"required,min=2,max=50"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		template, err := svc.Clone(c.Request.Context(), id, req.NewName, req.NewCode, c.GetString("userId"))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, template)
	}
}

// getTemplateVariables 获取模板变量定义
// @Summary 获取模板变量定义
// @Description 获取模板的变量定义列表
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response{data=[]models.TemplateVariable}
// @Router /network/templates/:id/variables [post]
func getTemplateVariables(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewTemplateService(core.DB.GetDB())
		variables, err := svc.GetVariables(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, variables)
	}
}

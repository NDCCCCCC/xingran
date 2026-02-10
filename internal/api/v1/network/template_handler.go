package network

import (
	"github.com/gin-gonic/gin"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// TemplateHandler 配置模板处理器
type TemplateHandler struct {
	templateService *services.TemplateService
}

// NewTemplateHandler 创建配置模板处理器实例
func NewTemplateHandler(templateService *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// List 查询配置模板列表
// @Summary 查询配置模板列表
// @Description 分页查询配置模板列表
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body services.ListTemplateRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/templates/list [post]
func (h *TemplateHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	req := services.ListTemplateRequest{
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

	templates, total, err := h.templateService.List(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
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

// GetByID 获取模板详情
// @Summary 获取模板详情
// @Description 根据ID获取模板详情
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response
// @Router /network/templates/:id [post]
func (h *TemplateHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	template, err := h.templateService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, template)
}

// Create 创建配置模板
// @Summary 创建配置模板
// @Description 创建新配置模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body services.CreateTemplateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /network/templates [post]
func (h *TemplateHandler) Create(c *gin.Context) {
	var req services.CreateTemplateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	template, err := h.templateService.Create(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "创建模板") {
		return
	}

	response.Success(c, template)
}

// Update 更新配置模板
// @Summary 更新配置模板
// @Description 更新模板信息
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body services.UpdateTemplateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/update [post]
func (h *TemplateHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	var req services.UpdateTemplateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	req.ID = id

	userID, _ := c.Get("user_id")
	req.UpdatedBy = userID.(string)

	err := h.templateService.Update(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "更新模板") {
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除配置模板
// @Summary 删除配置模板
// @Description 删除指定模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/delete [post]
func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	err := h.templateService.Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除模板") {
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除模板
// @Summary 批量删除模板
// @Description 批量删除多个模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "模板ID列表"
// @Success 200 {object} response.Response
// @Router /network/templates/batch-delete [post]
func (h *TemplateHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.templateService.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除模板") {
		return
	}

	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
}

// Preview 预览模板渲染
// @Summary 预览模板渲染
// @Description 使用示例变量预览模板渲染结果
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body map[string]string true "变量值"
// @Success 200 {object} response.Response{data=string}
// @Router /network/templates/:id/preview [post]
func (h *TemplateHandler) Preview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	var variables map[string]string
	if err := c.ShouldBindJSON(&variables); err != nil {
		variables = make(map[string]string)
	}

	result, err := h.templateService.Preview(c.Request.Context(), id, variables)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, gin.H{"content": result})
}

// Clone 克隆模板
// @Summary 克隆模板
// @Description 克隆现有模板创建新模板
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body object{newName=string,newCode=string} true "新模板名称和编码"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/clone [post]
func (h *TemplateHandler) Clone(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	var req struct {
		NewName string `json:"newName" binding:"required,min=2,max=100"`
		NewCode string `json:"newCode" binding:"required,min=2,max=50"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	template, err := h.templateService.Clone(c.Request.Context(), id, req.NewName, req.NewCode, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "克隆模板") {
		return
	}

	response.Success(c, template)
}

// GetVariables 获取模板变量定义
// @Summary 获取模板变量定义
// @Description 获取模板的变量定义列表
// @Tags 配置模板管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response
// @Router /network/templates/:id/variables [post]
func (h *TemplateHandler) GetVariables(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	variables, err := h.templateService.GetVariables(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, variables)
}

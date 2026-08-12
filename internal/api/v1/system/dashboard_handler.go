package system

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// 定义错误类型（与服务层共享）
var (
	ErrUnauthorized = errors.New("unauthorized")
)

// UserContext 用户上下文信息
type UserContext struct {
	UserID     string
	UserDeptID string
	DataScope  string
	IsAdmin    bool
}

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	service systemServices.DashboardService
	core    *core.Core
}

// NewDashboardHandler 创建仪表盘处理器实例
func NewDashboardHandler(service systemServices.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
func (h *DashboardHandler) WithCore(core *core.Core) *DashboardHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// getUserContext 从gin上下文中提取用户信息
func (h *DashboardHandler) getUserContext(c *gin.Context) (*UserContext, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, ErrUnauthorized
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return nil, ErrUnauthorized
	}

	return &UserContext{
		UserID:     userIDStr,
		UserDeptID: c.GetString("dept_id"),
		DataScope:  c.GetString("data_scope"),
		IsAdmin:    c.GetBool("is_admin"),
	}, nil
}

// requireUserContext 获取用户上下文，失败时返回错误响应
func (h *DashboardHandler) requireUserContext(c *gin.Context) (*UserContext, bool) {
	ctx, err := h.getUserContext(c)
	if err != nil {
		response.Error(c, apperrors.Unauthorized())
		return nil, false
	}
	return ctx, true
}

// handleServiceError 统一处理服务错误
func (h *DashboardHandler) handleServiceError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	// 直接返回apperror，因为服务层已经使用了统一错误处理
	response.Error(c, err)
	return true
}

// requireID 获取并验证路径参数ID
func (h *DashboardHandler) requireID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return "", false
	}
	return id, true
}

// ==================== 仪表盘 CRUD 操作 ====================

// GetDefault 获取可访问的默认仪表盘
// @Summary 获取可访问的默认仪表盘
// @Description 获取用户可访问的默认仪表盘（用户默认 > 系统仪表盘）
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/dashboards/default [get]
func (h *DashboardHandler) GetDefault(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	dashboard, err := h.service.GetAccessibleDefaultDashboard(
		c.Request.Context(),
		userCtx.UserID,
		userCtx.UserDeptID,
		userCtx.DataScope,
	)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("获取默认仪表盘失败"))
		return
	}

	if dashboard == nil {
		response.Success(c, gin.H{"dashboard": nil})
		return
	}

	response.Success(c, gin.H{"dashboard": dashboard})
}

// List 获取仪表盘列表（权限感知）
// @Summary 获取仪表盘列表
// @Description 获取用户可访问的仪表盘列表
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,keyword=string,isTemplate=bool,status=int} true "查询条件"
// @Success 200 {object} response.Response{data=object{list=[]object,total=int,current=int,pageSize=int}}
// @Router /system/dashboards/list [post]
func (h *DashboardHandler) List(c *gin.Context) {
	var req systemServices.DashboardListParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	result, err := h.service.GetAccessibleDashboards(
		c.Request.Context(),
		req,
		userCtx.UserID,
		userCtx.UserDeptID,
		userCtx.DataScope,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetByID 获取仪表盘详情
// @Summary 获取仪表盘详情
// @Description 根据仪表盘ID获取仪表盘详细信息
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id [get]
func (h *DashboardHandler) GetByID(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.DashboardNotFound())
		return
	}

	response.Success(c, gin.H{"dashboard": dashboard})
}

// Create 创建仪表盘（带权限验证）
// @Summary 创建仪表盘
// @Description 创建新的仪表盘（带权限验证）
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param request body object{name=string,description=string,layout=object,refreshInterval=int,isTemplate=bool,templateScope=string,scope=string,deptId=string,isSystem=bool} true "仪表盘信息"
// @Success 200 {object} response.Response
// @Router /system/dashboards [post]
func (h *DashboardHandler) Create(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	var req systemServices.CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	dashboard, err := h.service.CreateDashboardWithPermissions(
		c.Request.Context(),
		userCtx.UserID,
		userCtx.UserDeptID,
		userCtx.DataScope,
		userCtx.IsAdmin,
		req,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "仪表盘配置", OperTypeCreate)

	response.Success(c, gin.H{"dashboard": dashboard})
}

// Update 更新仪表盘
// @Summary 更新仪表盘
// @Description 更新仪表盘信息
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Param request body object{name=string,description=string,layout=object,refreshInterval=int,status=int} true "仪表盘信息"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/update [post]
func (h *DashboardHandler) Update(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	id, ok := h.requireID(c)
	if !ok {
		return
	}

	var req systemServices.UpdateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	err := h.service.UpdateDashboard(c.Request.Context(), userCtx.UserID, id, req)
	if h.handleServiceError(c, err) {
		return
	}

	recordOperLog(c, h.core, "仪表盘配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除仪表盘
// @Summary 删除仪表盘
// @Description 删除指定仪表盘
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id [delete]
func (h *DashboardHandler) Delete(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	id, ok := h.requireID(c)
	if !ok {
		return
	}

	err := h.service.DeleteDashboard(c.Request.Context(), userCtx.UserID, id)
	if h.handleServiceError(c, err) {
		return
	}

	recordOperLog(c, h.core, "仪表盘配置", OperTypeDelete)

	response.Success(c, gin.H{"message": "删除成功"})
}

// Duplicate 复制仪表盘
// @Summary 复制仪表盘
// @Description 复制指定仪表盘
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/duplicate [post]
func (h *DashboardHandler) Duplicate(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	id, ok := h.requireID(c)
	if !ok {
		return
	}

	dashboard, err := h.service.DuplicateDashboard(c.Request.Context(), userCtx.UserID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Duplicate creates a new dashboard — record as Create.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeCreate)

	response.Success(c, gin.H{"dashboard": dashboard})
}

// SetDefault 设置默认仪表盘
// @Summary 设置默认仪表盘
// @Description 设置指定仪表盘为默认仪表盘
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/set-default [post]
func (h *DashboardHandler) SetDefault(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	id, ok := h.requireID(c)
	if !ok {
		return
	}

	err := h.service.SetDefaultDashboard(c.Request.Context(), userCtx.UserID, id)
	if h.handleServiceError(c, err) {
		return
	}

	// SetDefault changes the default-state of one dashboard vs another — record as Update.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "设置成功"})
}

// ==================== 仪表盘模板操作 ====================

// GetTemplates 获取仪表盘模板列表
// @Summary 获取仪表盘模板列表
// @Description 获取仪表盘模板列表
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param request body object{scope=string} false "模板范围"
// @Success 200 {object} response.Response
// @Router /system/dashboards/templates [post]
func (h *DashboardHandler) GetTemplates(c *gin.Context) {
	var req struct {
		Scope *string `json:"scope"`
	}
	_ = c.ShouldBindJSON(&req)

	var scope *string
	if req.Scope != nil {
		scope = req.Scope
	}

	templates, err := h.service.GetTemplates(c.Request.Context(), scope)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"templates": templates})
}

// CreateFromTemplate 从模板创建仪表盘
// @Summary 从模板创建仪表盘
// @Description 从模板创建仪表盘
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body object{name=string} true "仪表盘名称"
// @Success 200 {object} response.Response
// @Router /system/dashboards/templates/:id/create [post]
func (h *DashboardHandler) CreateFromTemplate(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	templateID, ok := h.requireID(c)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	dashboard, err := h.service.CreateFromTemplate(c.Request.Context(), userCtx.UserID, templateID, req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}

	// CreateFromTemplate instantiates a new dashboard — record as Create.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeCreate)

	response.Success(c, gin.H{"dashboard": dashboard})
}

// ==================== 仪表盘版本操作 ====================

// GetVersions 获取仪表盘版本历史
// @Summary 获取仪表盘版本历史
// @Description 获取仪表盘版本历史
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/versions [get]
func (h *DashboardHandler) GetVersions(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	versions, err := h.service.GetVersions(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"versions": versions})
}

// CreateVersion 创建版本快照
// @Summary 创建版本快照
// @Description 创建仪表盘版本快照
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Param request body object{comment=string} false "版本备注"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/versions [post]
func (h *DashboardHandler) CreateVersion(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	id, ok := h.requireID(c)
	if !ok {
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&req)

	version, err := h.service.CreateVersion(c.Request.Context(), userCtx.UserID, id, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}

	// CreateVersion persists a new version snapshot — record as Create.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeCreate)

	response.Success(c, gin.H{"version": version})
}

// RestoreVersion 从版本恢复仪表盘
// @Summary 从版本恢复仪表盘
// @Description 从版本恢复仪表盘
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Param versionId path string true "版本ID"
// @Success 200 {object} response.Response
// @Router /system/dashboards/:id/versions/:versionId/restore [post]
func (h *DashboardHandler) RestoreVersion(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	dashboardID := c.Param("id")
	versionID := c.Param("versionId")
	if dashboardID == "" || versionID == "" {
		response.Error(c, apperrors.ParamMissing("仪表盘ID和版本ID"))
		return
	}

	err := h.service.RestoreVersion(c.Request.Context(), userCtx.UserID, dashboardID, versionID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// RestoreVersion overwrites the dashboard layout from a snapshot — record as Update.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeUpdate)

	response.Success(c, gin.H{"message": "恢复成功"})
}

// ==================== 仪表盘导入导出 ====================

// Export 导出仪表盘配置
// @Summary 导出仪表盘配置
// @Description 导出仪表盘配置
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "仪表盘ID"
// @Success 200 {object} response.Response{data=string}
// @Router /system/dashboards/:id/export [get]
func (h *DashboardHandler) Export(c *gin.Context) {
	id, ok := h.requireID(c)
	if !ok {
		return
	}

	config, err := h.service.ExportDashboard(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "仪表盘配置", OperTypeExport)

	response.Success(c, gin.H{"config": config})
}

// Import 导入仪表盘配置
// @Summary 导入仪表盘配置
// @Description 导入仪表盘配置
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param request body object{config=string} true "仪表盘配置"
// @Success 200 {object} response.Response
// @Router /system/dashboards/import [post]
func (h *DashboardHandler) Import(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	var req struct {
		Config string `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	dashboard, err := h.service.ImportDashboard(c.Request.Context(), userCtx.UserID, req.Config)
	if err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "仪表盘配置", OperTypeImport)

	response.Success(c, gin.H{"dashboard": dashboard})
}

// ==================== Widget 数据获取 ====================

// GetWidgetData 获取 Widget 数据
// @Summary 获取 Widget 数据
// @Description 获取 Widget 数据
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param id path string true "Widget ID"
// @Param request body object false "端点和参数"
// @Success 200 {object} response.Response{data=interface{}}
// @Router /system/dashboards/widgets/:id/data [post]
func (h *DashboardHandler) GetWidgetData(c *gin.Context) {
	widgetID := c.Param("id")
	if widgetID == "" {
		response.Error(c, apperrors.ParamMissing("Widget ID"))
		return
	}

	var req struct {
		Endpoint string                 `json:"endpoint"`
		Params   map[string]interface{} `json:"params"`
	}
	_ = c.ShouldBindJSON(&req)

	data, err := h.service.GetWidgetData(c.Request.Context(), widgetID, req.Endpoint, req.Params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"data": data})
}

// GetBatchWidgetData 批量获取 Widget 数据
// @Summary 批量获取 Widget 数据
// @Description 批量获取 Widget 数据
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param request body object{widgetIds=[]string,bypassCache=bool} true "Widget ID列表"
// @Success 200 {object} response.Response{data=map[string]WidgetDataResult}
// @Router /system/dashboards/widgets/batch-data [post]
func (h *DashboardHandler) GetBatchWidgetData(c *gin.Context) {
	var req struct {
		WidgetIds   []string `json:"widgetIds" binding:"required,min=1"`
		BypassCache bool     `json:"bypassCache"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	data, err := h.service.GetBatchWidgetData(c.Request.Context(), req.WidgetIds, req.BypassCache)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"data": data})
}

// ==================== API端点元数据 ====================

// GetAvailableEndpoints 获取可用的API端点列表
// @Summary 获取可用的API端点列表
// @Description 根据用户权限返回可访问的API端点，按模块分类
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=object}
// @Router /system/dashboards/endpoints [get]
func (h *DashboardHandler) GetAvailableEndpoints(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	endpoints, err := h.service.GetUserAccessibleEndpoints(c.Request.Context(), userCtx.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"categories": endpoints,
		"total":      len(endpoints),
	})
}

// ValidateEndpoint 验证API端点配置
// @Summary 验证API端点配置
// @Description 验证指定的API端点是否存在
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param route query string true "端点路径"
// @Param method query string true "请求方法"
// @Success 200 {object} response.Response{data=object}
// @Router /system/dashboards/endpoints/validate [get]
func (h *DashboardHandler) ValidateEndpoint(c *gin.Context) {
	route := c.Query("route")
	method := c.Query("method")

	if route == "" || method == "" {
		response.Error(c, apperrors.ParamMissing("端点路径和请求方法"))
		return
	}

	endpoint, err := h.service.ValidateEndpoint(route, method)
	if err != nil {
		response.Error(c, apperrors.NotFound("端点不存在"))
		return
	}

	response.Success(c, endpoint)
}

// GetUserEndpointsWithFilter 获取过滤后的端点列表
// @Summary 获取过滤后的端点列表
// @Description 根据Widget类型过滤可用的API端点
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Param widgetType query string false "Widget类型"
// @Success 200 {object} response.Response{data=object}
// @Router /system/dashboards/endpoints/filter [get]
func (h *DashboardHandler) GetUserEndpointsWithFilter(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	widgetType := c.Query("widgetType")

	categories, err := h.service.GetUserAccessibleEndpoints(c.Request.Context(), userCtx.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 如果指定了Widget类型，过滤端点
	if widgetType != "" {
		filteredCategories := h.service.FilterEndpointsByWidgetType(categories, widgetType)
		categories = filteredCategories
	}

	response.Success(c, gin.H{
		"categories": categories,
		"total":      len(categories),
	})
}

// InvalidateEndpointCache 清除用户端点缓存
// @Summary 清除用户端点缓存
// @Description 清除当前用户的端点列表缓存
// @Tags 仪表盘管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/dashboards/endpoints/cache/invalidate [post]
func (h *DashboardHandler) InvalidateEndpointCache(c *gin.Context) {
	userCtx, ok := h.requireUserContext(c)
	if !ok {
		return
	}

	h.service.InvalidateUserCache(c.Request.Context(), userCtx.UserID)

	// InvalidateEndpointCache is a maintenance action — record as Other.
	recordOperLog(c, h.core, "仪表盘配置", OperTypeOther)

	response.Success(c, gin.H{
		"message": "缓存已清除",
	})
}

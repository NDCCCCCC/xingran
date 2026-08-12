package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// DashboardService 仪表盘服务接口
type DashboardService interface {
	GetDashboards(ctx context.Context, params DashboardListParams) (*DashboardListResponse, error)
	GetAccessibleDashboards(ctx context.Context, params DashboardListParams, userID string, userDeptID string, dataScope string) (*DashboardListResponse, error)
	GetDashboard(ctx context.Context, id string) (*models.Dashboard, error)
	GetAccessibleDefaultDashboard(ctx context.Context, userID string, userDeptID string, dataScope string) (*models.Dashboard, error)
	CreateDashboard(ctx context.Context, userID string, req CreateDashboardRequest) (*models.Dashboard, error)
	CreateDashboardWithPermissions(ctx context.Context, userID string, userDeptID string, dataScope string, isAdmin bool, req CreateDashboardRequest) (*models.Dashboard, error)
	UpdateDashboard(ctx context.Context, userID string, id string, req UpdateDashboardRequest) error
	DeleteDashboard(ctx context.Context, userID string, id string) error
	DuplicateDashboard(ctx context.Context, userID string, id string) (*models.Dashboard, error)
	SetDefaultDashboard(ctx context.Context, userID string, id string) error
	GetTemplates(ctx context.Context, scope *string) ([]models.Dashboard, error)
	CreateFromTemplate(ctx context.Context, userID string, templateID string, name string) (*models.Dashboard, error)
	GetVersions(ctx context.Context, dashboardID string) ([]models.DashboardVersion, error)
	CreateVersion(ctx context.Context, userID string, dashboardID string, comment string) (*models.DashboardVersion, error)
	RestoreVersion(ctx context.Context, userID string, dashboardID string, versionID string) error
	ExportDashboard(ctx context.Context, id string) (string, error)
	ImportDashboard(ctx context.Context, userID string, config string) (*models.Dashboard, error)
	GetWidgetData(ctx context.Context, widgetID string, apiEndpoint string, params map[string]interface{}) (interface{}, error)
	GetBatchWidgetData(ctx context.Context, widgetIDs []string, bypassCache bool) (map[string]WidgetDataResult, error)
	GetUserAccessibleEndpoints(ctx context.Context, userID string) ([]services.CategoryEndpoints, error)
	ValidateEndpoint(route, method string) (*services.EndpointDetail, error)
	InvalidateUserCache(ctx context.Context, userID string)
	FilterEndpointsByWidgetType(categories []services.CategoryEndpoints, widgetType string) []services.CategoryEndpoints
}

// DashboardServiceImpl 仪表盘服务实现
type DashboardServiceImpl struct {
	db              *gorm.DB
	cache           cache.Cache
	endpointService EndpointService
	widgetFetcher   WidgetDataFetcher // Widget 数据获取器
}

// EndpointService 端点服务接口（抽象）
type EndpointService interface {
	GetUserAccessibleEndpoints(ctx context.Context, userID string) ([]services.CategoryEndpoints, error)
	ValidateEndpoint(route, method string) (*services.EndpointDetail, error)
	InvalidateUserCache(ctx context.Context, userID string)
}

// endpointServiceAdapter 端点服务适配器
type endpointServiceAdapter struct {
	*services.APIEndpointService
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(db *gorm.DB, cache cache.Cache, endpointService *services.APIEndpointService) DashboardService {
	var endpointSvc EndpointService
	if endpointService != nil {
		endpointSvc = &endpointServiceAdapter{APIEndpointService: endpointService}
	}

	// 创建 WidgetDataFetcher
	serviceRegistry := NewDefaultServiceRegistry()
	widgetFetcher := NewWidgetDataFetcher(db, cache, endpointSvc, serviceRegistry)

	return &DashboardServiceImpl{
		db:              db,
		cache:           cache,
		endpointService: endpointSvc,
		widgetFetcher:   widgetFetcher,
	}
}

// DashboardListParams 仪表盘列表请求参数
type DashboardListParams struct {
	Current    int                     `json:"current" binding:"required,min=1"`
	PageSize   int                     `json:"pageSize" binding:"required,min=1,max=100"`
	Keyword    string                  `json:"keyword,omitempty"`
	IsTemplate bool                    `json:"isTemplate,omitempty"`
	Status     *models.DashboardStatus `json:"status,omitempty"`
}

// DashboardListResponse 仪表盘列表响应
type DashboardListResponse struct {
	List     []models.Dashboard `json:"list"`
	Total    int64              `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"pageSize"`
}

// CreateDashboardRequest 创建仪表盘请求
type CreateDashboardRequest struct {
	Name            string               `json:"name" binding:"required,min=1,max=100"`
	Description     string               `json:"description" binding:"max=500"`
	Layout          models.LayoutConfig  `json:"layout" binding:"required"`
	RefreshInterval int                  `json:"refreshInterval" binding:"min=0,max=3600"`
	IsTemplate      bool                 `json:"isTemplate"`
	TemplateScope   models.TemplateScope `json:"templateScope" binding:"omitempty,oneof=global dept personal"`
	// 权限相关
	Scope    models.DashboardScope `json:"scope" binding:"omitempty,oneof=private dept global"`
	DeptID   *string               `json:"deptId" binding:"omitempty"`
	IsSystem bool                  `json:"isSystem"`
}

// UpdateDashboardRequest 更新仪表盘请求
type UpdateDashboardRequest struct {
	Name            *string                 `json:"name" binding:"omitempty,min=1,max=100"`
	Description     *string                 `json:"description" binding:"omitempty,max=500"`
	Layout          *models.LayoutConfig    `json:"layout"`
	RefreshInterval *int                    `json:"refreshInterval" binding:"omitempty,min=0,max=3600"`
	Status          *models.DashboardStatus `json:"status" binding:"omitempty,min=0,max=1"`
}

const (
	dashboardCachePrefix = "dashboard:"
)

// ==================== 辅助方法 ====================

// getDashboardWithOwner 获取仪表盘并验证所有权
func (s *DashboardServiceImpl) getDashboardWithOwner(ctx context.Context, id, ownerID string) (*models.Dashboard, error) {
	var dashboard models.Dashboard
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.DashboardNotFound()
		}
		return nil, fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	if !dashboard.IsTemplate && dashboard.OwnerID != ownerID {
		return nil, apperrors.PermissionDenied()
	}

	return &dashboard, nil
}

// invalidateDashboardCache 清除仪表盘缓存
func (s *DashboardServiceImpl) invalidateDashboardCache(ctx context.Context, dashboardID string) {
	_ = s.cache.Delete(ctx, dashboardCachePrefix+dashboardID)
}

// invalidateDashboardListCache 清除仪表盘列表缓存
func (s *DashboardServiceImpl) invalidateDashboardListCache(ctx context.Context) {
	// 清除所有列表缓存（简化处理）
	// 生产环境可以考虑使用缓存标签或模式匹配
}

// AccessContext 访问上下文
type AccessContext struct {
	UserID     string
	UserDeptID string
	DataScope  string
	IsAdmin    bool
}

// CheckAccess 检查访问权限
func (s *DashboardServiceImpl) CheckAccess(dashboard *models.Dashboard, ctx *AccessContext) bool {
	// 创建者可以访问
	if dashboard.OwnerID == ctx.UserID {
		return true
	}

	// 全局可见
	if dashboard.Scope == models.DashboardScopeGlobal {
		return true
	}

	// 部门可见
	if dashboard.Scope == models.DashboardScopeDept {
		return s.checkDeptAccess(dashboard, ctx)
	}

	return false
}

// checkDeptAccess 检查部门访问权限
func (s *DashboardServiceImpl) checkDeptAccess(dashboard *models.Dashboard, ctx *AccessContext) bool {
	// 用户可以看到所有部门仪表盘
	if ctx.DataScope == "all" || ctx.DataScope == "custom" {
		return true
	}

	// 用户只能看到本部门的仪表盘
	return dashboard.DeptID != nil && *dashboard.DeptID == ctx.UserDeptID
}

// ==================== 公共方法 ====================

// GetDashboards 获取仪表盘列表
func (s *DashboardServiceImpl) GetDashboards(ctx context.Context, params DashboardListParams) (*DashboardListResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Dashboard{})

	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	if params.IsTemplate {
		query = query.Where("is_template = ?", true)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count dashboards: %w", err)
	}

	offset := (params.Current - 1) * params.PageSize
	var dashboards []models.Dashboard
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&dashboards).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}

	return &DashboardListResponse{
		List:     dashboards,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetDashboard 获取仪表盘详情
func (s *DashboardServiceImpl) GetDashboard(ctx context.Context, id string) (*models.Dashboard, error) {
	// 尝试从缓存获取
	cacheKey := dashboardCachePrefix + id
	var dashboard models.Dashboard
	if err := s.cache.GetJSON(ctx, cacheKey, &dashboard); err == nil {
		return &dashboard, nil
	}

	// 从数据库查询
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.DashboardNotFound()
		}
		return nil, fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	_ = s.cache.SetJSON(ctx, cacheKey, dashboard, 5*time.Minute)

	return &dashboard, nil
}

// CreateDashboard 创建仪表盘
func (s *DashboardServiceImpl) CreateDashboard(ctx context.Context, userID string, req CreateDashboardRequest) (*models.Dashboard, error) {
	dashboard := &models.Dashboard{
		Name:            req.Name,
		Description:     req.Description,
		OwnerID:         userID,
		Layout:          req.Layout,
		RefreshInterval: req.RefreshInterval,
		IsTemplate:      req.IsTemplate,
		TemplateScope:   req.TemplateScope,
		Status:          models.DashboardStatusNormal,
	}

	dashboard.CreatedBy = userID
	dashboard.UpdatedBy = userID

	if err := s.db.WithContext(ctx).Create(dashboard).Error; err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	return dashboard, nil
}

// UpdateDashboard 更新仪表盘
func (s *DashboardServiceImpl) UpdateDashboard(ctx context.Context, userID string, id string, req UpdateDashboardRequest) error {
	dashboard, err := s.getDashboardWithOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Layout != nil {
		updates["layout"] = *req.Layout
	}
	if req.RefreshInterval != nil {
		updates["refresh_interval"] = *req.RefreshInterval
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	updates["updated_by"] = userID

	if err := s.db.WithContext(ctx).Model(dashboard).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	s.invalidateDashboardCache(ctx, id)
	s.invalidateDashboardListCache(ctx)

	return nil
}

// DeleteDashboard 删除仪表盘
func (s *DashboardServiceImpl) DeleteDashboard(ctx context.Context, userID string, id string) error {
	dashboard, err := s.getDashboardWithOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Delete(dashboard).Error; err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	s.invalidateDashboardCache(ctx, id)
	s.invalidateDashboardListCache(ctx)

	return nil
}

// DuplicateDashboard 复制仪表盘
func (s *DashboardServiceImpl) DuplicateDashboard(ctx context.Context, userID string, id string) (*models.Dashboard, error) {
	original, err := s.GetDashboard(ctx, id)
	if err != nil {
		return nil, err
	}

	duplicate := &models.Dashboard{
		Name:            original.Name + " (副本)",
		Description:     original.Description,
		OwnerID:         userID,
		Layout:          original.Layout,
		RefreshInterval: original.RefreshInterval,
		IsTemplate:      false,
		Status:          models.DashboardStatusNormal,
	}

	duplicate.CreatedBy = userID
	duplicate.UpdatedBy = userID

	if err := s.db.WithContext(ctx).Create(duplicate).Error; err != nil {
		return nil, fmt.Errorf("failed to duplicate dashboard: %w", err)
	}

	return duplicate, nil
}

// SetDefaultDashboard 设置默认仪表盘
func (s *DashboardServiceImpl) SetDefaultDashboard(ctx context.Context, userID string, id string) error {
	dashboard, err := s.getDashboardWithOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).
		Model(&models.Dashboard{}).
		Where("owner_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("failed to clear default dashboards: %w", err)
	}

	if err := s.db.WithContext(ctx).
		Model(dashboard).
		Update("is_default", true).Error; err != nil {
		return fmt.Errorf("failed to set default dashboard: %w", err)
	}

	s.invalidateDashboardCache(ctx, id)

	return nil
}

// GetTemplates 获取仪表盘模板列表
func (s *DashboardServiceImpl) GetTemplates(ctx context.Context, scope *string) ([]models.Dashboard, error) {
	query := s.db.WithContext(ctx).
		Where("is_template = ?", true).
		Where("status = ?", models.DashboardStatusNormal)

	// F-21: scope 为 nil 或无效值时,默认只返回 Global 模板,避免泄露私人/部门模板。
	// 任何登录用户调用本接口时若未传 scope,只应看到公开(Global)模板。
	if scope != nil {
		scopeEnum := models.TemplateScope(*scope)
		if scopeEnum == models.TemplateScopeGlobal || scopeEnum == models.TemplateScopeDept || scopeEnum == models.TemplateScopePersonal {
			query = query.Where("template_scope = ? OR template_scope = ?", scopeEnum, models.TemplateScopeGlobal)
		} else {
			// 无效 scope 值同样降级为仅 Global,而不是放行全部
			query = query.Where("template_scope = ?", models.TemplateScopeGlobal)
		}
	} else {
		query = query.Where("template_scope = ?", models.TemplateScopeGlobal)
	}

	var templates []models.Dashboard
	if err := query.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}

	return templates, nil
}

// CreateFromTemplate 从模板创建仪表盘
func (s *DashboardServiceImpl) CreateFromTemplate(ctx context.Context, userID string, templateID string, name string) (*models.Dashboard, error) {
	template, err := s.GetDashboard(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	if !template.IsTemplate {
		return nil, fmt.Errorf("not a template")
	}

	dashboard := &models.Dashboard{
		Name:            name,
		Description:     template.Description,
		OwnerID:         userID,
		Layout:          template.Layout,
		RefreshInterval: template.RefreshInterval,
		IsTemplate:      false,
		Status:          models.DashboardStatusNormal,
	}

	dashboard.CreatedBy = userID
	dashboard.UpdatedBy = userID

	if err := s.db.WithContext(ctx).Create(dashboard).Error; err != nil {
		return nil, fmt.Errorf("failed to create dashboard from template: %w", err)
	}

	return dashboard, nil
}

// GetVersions 获取仪表盘版本历史
func (s *DashboardServiceImpl) GetVersions(ctx context.Context, dashboardID string) ([]models.DashboardVersion, error) {
	var versions []models.DashboardVersion
	if err := s.db.WithContext(ctx).
		Where("dashboard_id = ?", dashboardID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	return versions, nil
}

// CreateVersion 创建版本快照
func (s *DashboardServiceImpl) CreateVersion(ctx context.Context, userID string, dashboardID string, comment string) (*models.DashboardVersion, error) {
	dashboard, err := s.GetDashboard(ctx, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("dashboard not found: %w", err)
	}

	version := &models.DashboardVersion{
		DashboardID: dashboardID,
		Layout:      dashboard.Layout,
		Comment:     comment,
		CreatedBy:   userID,
	}

	if err := s.db.WithContext(ctx).Create(version).Error; err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, nil
}

// RestoreVersion 从版本恢复仪表盘
func (s *DashboardServiceImpl) RestoreVersion(ctx context.Context, userID string, dashboardID string, versionID string) error {
	var version models.DashboardVersion
	if err := s.db.WithContext(ctx).
		Where("id = ? AND dashboard_id = ?", versionID, dashboardID).
		First(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("version not found")
		}
		return fmt.Errorf("failed to fetch version: %w", err)
	}

	if err := s.db.WithContext(ctx).
		Model(&models.Dashboard{}).
		Where("id = ?", dashboardID).
		Updates(map[string]interface{}{
			"layout":     version.Layout,
			"updated_by": userID,
		}).Error; err != nil {
		return fmt.Errorf("failed to restore version: %w", err)
	}

	_ = s.cache.Delete(ctx, dashboardCachePrefix+dashboardID)

	return nil
}

// ExportDashboard 导出仪表盘配置
func (s *DashboardServiceImpl) ExportDashboard(ctx context.Context, id string) (string, error) {
	dashboard, err := s.GetDashboard(ctx, id)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(dashboard)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard: %w", err)
	}

	return string(data), nil
}

// ImportDashboard 导入仪表盘配置
func (s *DashboardServiceImpl) ImportDashboard(ctx context.Context, userID string, config string) (*models.Dashboard, error) {
	var dashboard models.Dashboard
	if err := json.Unmarshal([]byte(config), &dashboard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	dashboard.ID = ""
	dashboard.OwnerID = userID
	dashboard.IsTemplate = false
	dashboard.CreatedBy = userID
	dashboard.UpdatedBy = userID

	if err := s.db.WithContext(ctx).Create(&dashboard).Error; err != nil {
		return nil, fmt.Errorf("failed to import dashboard: %w", err)
	}

	return &dashboard, nil
}

// GetWidgetData 获取 Widget 数据（调用其他服务的 API）
func (s *DashboardServiceImpl) GetWidgetData(ctx context.Context, widgetID string, apiEndpoint string, params map[string]interface{}) (interface{}, error) {
	// 1. 查询 Widget 配置
	var widget models.WidgetConfig
	if err := s.db.WithContext(ctx).Where("id = ?", widgetID).First(&widget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("widget not found: %s", widgetID)
		}
		return nil, fmt.Errorf("failed to fetch widget: %w", err)
	}

	// 2. 从 Context 提取用户 ID
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// 3. 调用 WidgetDataFetcher 获取数据
	if s.widgetFetcher == nil {
		return nil, fmt.Errorf("widget fetcher not available")
	}

	data, _, err := s.widgetFetcher.FetchWidgetData(ctx, &widget, userID, false)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetBatchWidgetData 批量获取 Widget 数据
func (s *DashboardServiceImpl) GetBatchWidgetData(ctx context.Context, widgetIDs []string, bypassCache bool) (map[string]WidgetDataResult, error) {
	// 从 Context 提取用户 ID
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// 调用 WidgetDataFetcher 批量获取数据
	if s.widgetFetcher == nil {
		return nil, fmt.Errorf("widget fetcher not available")
	}

	return s.widgetFetcher.FetchBatchWidgetData(ctx, widgetIDs, userID, bypassCache)
}

// GetUserAccessibleEndpoints 获取用户可访问的端点
func (s *DashboardServiceImpl) GetUserAccessibleEndpoints(ctx context.Context, userID string) ([]services.CategoryEndpoints, error) {
	if s.endpointService == nil {
		return []services.CategoryEndpoints{}, nil
	}
	return s.endpointService.GetUserAccessibleEndpoints(ctx, userID)
}

// ValidateEndpoint 验证端点
func (s *DashboardServiceImpl) ValidateEndpoint(route, method string) (*services.EndpointDetail, error) {
	if s.endpointService == nil {
		return nil, fmt.Errorf("endpoint service not available")
	}
	return s.endpointService.ValidateEndpoint(route, method)
}

// InvalidateUserCache 清除用户缓存
func (s *DashboardServiceImpl) InvalidateUserCache(ctx context.Context, userID string) {
	if s.endpointService != nil {
		s.endpointService.InvalidateUserCache(ctx, userID)
	}
}

// FilterEndpointsByWidgetType 根据Widget类型过滤端点
func (s *DashboardServiceImpl) FilterEndpointsByWidgetType(categories []services.CategoryEndpoints, widgetType string) []services.CategoryEndpoints {
	filteredCategories := make([]services.CategoryEndpoints, 0)
	for _, category := range categories {
		filteredEndpoints := make([]services.EndpointDetail, 0)
		for _, endpoint := range category.Endpoints {
			// 检查是否支持该Widget类型
			for _, supported := range endpoint.SupportedWidgets {
				if supported == widgetType {
					filteredEndpoints = append(filteredEndpoints, endpoint)
					break
				}
			}
		}
		if len(filteredEndpoints) > 0 {
			filteredCategories = append(filteredCategories, services.CategoryEndpoints{
				Module:    category.Module,
				Category:  category.Category,
				Icon:      category.Icon,
				Endpoints: filteredEndpoints,
			})
		}
	}
	return filteredCategories
}

// ============= 权限感知方法 =============

// GetAccessibleDefaultDashboard 获取用户可访问的默认仪表盘
// 优先级：用户默认 > 系统仪表盘
func (s *DashboardServiceImpl) GetAccessibleDefaultDashboard(ctx context.Context, userID string, userDeptID string, dataScope string) (*models.Dashboard, error) {
	// 1. 查找用户的默认仪表盘
	var userDefault models.Dashboard
	err := s.db.WithContext(ctx).
		Where("owner_id = ? AND is_default = ? AND status = ?", userID, true, models.DashboardStatusNormal).
		First(&userDefault).Error

	if err == nil {
		// 验证用户是否有权访问该仪表盘
		if s.canAccessDashboard(&userDefault, userID, userDeptID, dataScope) {
			return &userDefault, nil
		}
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 2. 无用户默认，查找系统仪表盘（管理员创建的全局仪表盘）
	var systemDashboard models.Dashboard
	err = s.db.WithContext(ctx).
		Where("is_system = ? AND scope = ? AND status = ?", true, models.DashboardScopeGlobal, models.DashboardStatusNormal).
		Order("created_at ASC").
		First(&systemDashboard).Error

	if err == nil {
		return &systemDashboard, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, nil // 无可用仪表盘
	}

	return nil, err
}

// GetAccessibleDashboards 获取用户可访问的仪表盘列表（权限过滤）
func (s *DashboardServiceImpl) GetAccessibleDashboards(ctx context.Context, params DashboardListParams, userID string, userDeptID string, dataScope string) (*DashboardListResponse, error) {
	var dashboards []models.Dashboard
	var total int64

	// 基础查询：只查询正常状态的
	baseQuery := s.db.WithContext(ctx).Model(&models.Dashboard{}).Where("status = ?", models.DashboardStatusNormal)

	// 构建权限过滤条件
	// 1. 私有：仅自己创建的
	// 2. 部门：自己创建的 或 本部门可见的
	// 3. 全局：任何人可见
	permissionFilter := s.db.WithContext(ctx).Where("owner_id = ?", userID)                      // 自己创建的
	permissionFilter = permissionFilter.Or(s.db.Where("scope = ?", models.DashboardScopeGlobal)) // 全局可见

	// 部门可见性（根据用户数据范围）
	if dataScope == "all" || dataScope == "custom" {
		// 用户可以看到所有部门仪表盘
		permissionFilter = permissionFilter.Or(s.db.Where("scope = ?", models.DashboardScopeDept))
	} else {
		// 用户只能看到本部门的仪表盘
		permissionFilter = permissionFilter.Or(s.db.Where(
			"scope = ? AND dept_id = ?", models.DashboardScopeDept, userDeptID))
	}

	query := baseQuery.Where(permissionFilter)

	// 搜索关键词
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	// 模板过滤
	if params.IsTemplate {
		query = query.Where("is_template = ?", true)
	}

	// 状态过滤
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count dashboards: %w", err)
	}

	// 分页查询
	offset := (params.Current - 1) * params.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&dashboards).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}

	return &DashboardListResponse{
		List:     dashboards,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// CreateDashboardWithPermissions 创建仪表盘（带权限验证）
func (s *DashboardServiceImpl) CreateDashboardWithPermissions(ctx context.Context, userID string, userDeptID string, dataScope string, isAdmin bool, req CreateDashboardRequest) (*models.Dashboard, error) {
	// 设置默认值
	scope := req.Scope
	if scope == "" {
		scope = models.DashboardScopePrivate
	}

	// 权限验证：部门选择受数据范围限制
	if scope == models.DashboardScopeDept {
		if req.DeptID == nil {
			return nil, fmt.Errorf("部门仪表盘必须指定部门")
		}

		// 验证部门选择是否在用户数据范围内
		if !isAdmin && dataScope != "all" && dataScope != "custom" {
			if *req.DeptID != userDeptID {
				return nil, fmt.Errorf("无权限选择该部门")
			}
		}
	}

	// 全局仪表盘仅管理员可创建
	if scope == models.DashboardScopeGlobal && !isAdmin {
		return nil, fmt.Errorf("仅管理员可创建全局仪表盘")
	}

	// 系统仪表盘仅管理员可创建
	if req.IsSystem && !isAdmin {
		return nil, fmt.Errorf("仅管理员可创建系统仪表盘")
	}

	dashboard := &models.Dashboard{
		Name:            req.Name,
		Description:     req.Description,
		OwnerID:         userID,
		OwnerDeptID:     userDeptID,
		Layout:          req.Layout,
		RefreshInterval: req.RefreshInterval,
		IsTemplate:      req.IsTemplate,
		TemplateScope:   req.TemplateScope,
		Scope:           scope,
		DeptID:          req.DeptID,
		IsSystem:        req.IsSystem,
		Status:          models.DashboardStatusNormal,
	}

	dashboard.CreatedBy = userID
	dashboard.UpdatedBy = userID

	if err := s.db.WithContext(ctx).Create(dashboard).Error; err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	return dashboard, nil
}

// canAccessDashboard 检查用户是否有权访问仪表盘
func (s *DashboardServiceImpl) canAccessDashboard(dashboard *models.Dashboard, userID string, userDeptID string, dataScope string) bool {
	ctx := &AccessContext{
		UserID:     userID,
		UserDeptID: userDeptID,
		DataScope:  dataScope,
	}
	return s.CheckAccess(dashboard, ctx)
}

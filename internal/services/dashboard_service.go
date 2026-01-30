package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
)

// DashboardService 仪表盘服务
type DashboardService struct {
	db    *gorm.DB
	cache cache.Cache
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(db *gorm.DB, cache cache.Cache) *DashboardService {
	return &DashboardService{
		db:    db,
		cache: cache,
	}
}

// DashboardListParams 仪表盘列表请求参数
type DashboardListParams struct {
	Current    int                       `json:"current" binding:"required,min=1"`
	PageSize   int                       `json:"pageSize" binding:"required,min=1,max=100"`
	Keyword    string                    `json:"keyword,omitempty"`
	IsTemplate bool                      `json:"isTemplate,omitempty"`
	Status     *models.DashboardStatus   `json:"status,omitempty"`
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
	Name            string                  `json:"name" binding:"required,min=1,max=100"`
	Description     string                  `json:"description" binding:"max=500"`
	Layout          models.LayoutConfig     `json:"layout" binding:"required"`
	RefreshInterval int                     `json:"refreshInterval" binding:"min=0,max=3600"`
	IsTemplate      bool                    `json:"isTemplate"`
	TemplateScope   models.TemplateScope    `json:"templateScope" binding:"omitempty,oneof=global dept personal"`
}

// UpdateDashboardRequest 更新仪表盘请求
type UpdateDashboardRequest struct {
	Name            *string                  `json:"name" binding:"omitempty,min=1,max=100"`
	Description     *string                  `json:"description" binding:"omitempty,max=500"`
	Layout          *models.LayoutConfig     `json:"layout"`
	RefreshInterval *int                     `json:"refreshInterval" binding:"omitempty,min=0,max=3600"`
	Status          *models.DashboardStatus `json:"status" binding:"omitempty,min=0,max=1"`
}

const (
	dashboardCachePrefix     = "dashboard:"
	dashboardListCachePrefix = "dashboard:list:"
)

// GetDashboards 获取仪表盘列表
func (s *DashboardService) GetDashboards(ctx context.Context, params DashboardListParams) (*DashboardListResponse, error) {
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
func (s *DashboardService) GetDashboard(ctx context.Context, id string) (*models.Dashboard, error) {
	// 尝试从缓存获取
	cacheKey := dashboardCachePrefix + id
	var dashboard models.Dashboard
	if err := s.cache.GetJSON(ctx, cacheKey, &dashboard); err == nil {
		return &dashboard, nil
	}

	// 从数据库查询
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("dashboard not found")
		}
		return nil, fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	s.cache.SetJSON(ctx, cacheKey, dashboard, 5*time.Minute)

	return &dashboard, nil
}

// CreateDashboard 创建仪表盘
func (s *DashboardService) CreateDashboard(ctx context.Context, userID string, req CreateDashboardRequest) (*models.Dashboard, error) {
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
func (s *DashboardService) UpdateDashboard(ctx context.Context, userID string, id string, req UpdateDashboardRequest) error {
	var dashboard models.Dashboard
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("dashboard not found")
		}
		return fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	if !dashboard.IsTemplate && dashboard.OwnerID != userID {
		return fmt.Errorf("permission denied: not the owner")
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

	if err := s.db.WithContext(ctx).Model(&dashboard).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	s.cache.Delete(ctx, dashboardCachePrefix+id)

	return nil
}

// DeleteDashboard 删除仪表盘
func (s *DashboardService) DeleteDashboard(ctx context.Context, userID string, id string) error {
	var dashboard models.Dashboard
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("dashboard not found")
		}
		return fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	if !dashboard.IsTemplate && dashboard.OwnerID != userID {
		return fmt.Errorf("permission denied: not the owner")
	}

	if err := s.db.WithContext(ctx).Delete(&dashboard).Error; err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	s.cache.Delete(ctx, dashboardCachePrefix+id)

	return nil
}

// DuplicateDashboard 复制仪表盘
func (s *DashboardService) DuplicateDashboard(ctx context.Context, userID string, id string) (*models.Dashboard, error) {
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
func (s *DashboardService) SetDefaultDashboard(ctx context.Context, userID string, id string) error {
	var dashboard models.Dashboard
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("dashboard not found")
		}
		return fmt.Errorf("failed to fetch dashboard: %w", err)
	}

	if !dashboard.IsTemplate && dashboard.OwnerID != userID {
		return fmt.Errorf("permission denied: not the owner")
	}

	if err := s.db.WithContext(ctx).
		Model(&models.Dashboard{}).
		Where("owner_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("failed to clear default dashboards: %w", err)
	}

	if err := s.db.WithContext(ctx).
		Model(&dashboard).
		Update("is_default", true).Error; err != nil {
		return fmt.Errorf("failed to set default dashboard: %w", err)
	}

	s.cache.Delete(ctx, dashboardCachePrefix+id)

	return nil
}

// GetTemplates 获取仪表盘模板列表
func (s *DashboardService) GetTemplates(ctx context.Context, scope *models.TemplateScope) ([]models.Dashboard, error) {
	query := s.db.WithContext(ctx).
		Where("is_template = ?", true).
		Where("status = ?", models.DashboardStatusNormal)

	if scope != nil {
		query = query.Where("template_scope = ? OR template_scope = ?", *scope, models.TemplateScopeGlobal)
	}

	var templates []models.Dashboard
	if err := query.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}

	return templates, nil
}

// CreateFromTemplate 从模板创建仪表盘
func (s *DashboardService) CreateFromTemplate(ctx context.Context, userID string, templateID string, name string) (*models.Dashboard, error) {
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
func (s *DashboardService) GetVersions(ctx context.Context, dashboardID string) ([]models.DashboardVersion, error) {
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
func (s *DashboardService) CreateVersion(ctx context.Context, userID string, dashboardID string, comment string) (*models.DashboardVersion, error) {
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
func (s *DashboardService) RestoreVersion(ctx context.Context, userID string, dashboardID string, versionID string) error {
	var version models.DashboardVersion
	if err := s.db.WithContext(ctx).
		Where("id = ? AND dashboard_id = ?", versionID, dashboardID).
		First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	s.cache.Delete(ctx, dashboardCachePrefix+dashboardID)

	return nil
}

// ExportDashboard 导出仪表盘配置
func (s *DashboardService) ExportDashboard(ctx context.Context, id string) (string, error) {
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
func (s *DashboardService) ImportDashboard(ctx context.Context, userID string, config string) (*models.Dashboard, error) {
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
func (s *DashboardService) GetWidgetData(ctx context.Context, widgetID string, apiEndpoint string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("widget data fetching not implemented yet")
}

// GetBatchWidgetData 批量获取 Widget 数据
func (s *DashboardService) GetBatchWidgetData(ctx context.Context, widgetIDs []string) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// APIEndpointService API端点服务
type APIEndpointService struct {
	metadataConfig *config.APIMetadataConfig
	cache          cache.Cache
	db             *gorm.DB
}

// EndpointDetail 端点详情
type EndpointDetail struct {
	Route            string            `json:"route"`
	Method           string            `json:"method"`
	DisplayName      string            `json:"displayName"`
	Description      string            `json:"description"`
	Module           string            `json:"module"`
	Category         string            `json:"category"`
	Icon             string            `json:"icon"`
	DataType         string            `json:"dataType"`
	DataPath         string            `json:"dataPath"`
	SupportedWidgets []string          `json:"supportedWidgets"`
	ExampleParams    map[string]string `json:"exampleParams"`
	RequiredPerms    []string          `json:"requiredPerms"`
}

// CategoryEndpoints 分类端点列表
type CategoryEndpoints struct {
	Module    string           `json:"module"`
	Category  string           `json:"category"`
	Icon      string           `json:"icon"`
	Endpoints []EndpointDetail `json:"endpoints"`
}

// NewAPIEndpointService 创建API端点服务
func NewAPIEndpointService(
	metadata *config.APIMetadataConfig,
	cache cache.Cache,
	db *gorm.DB,
) *APIEndpointService {
	return &APIEndpointService{
		metadataConfig: metadata,
		cache:          cache,
		db:             db,
	}
}

// GetUserAccessibleEndpoints 获取用户可访问的端点（带权限过滤）
func (s *APIEndpointService) GetUserAccessibleEndpoints(
	ctx context.Context,
	userID string,
) ([]CategoryEndpoints, error) {
	cacheKey := fmt.Sprintf("user_endpoints:%s", userID)
	var cachedResult []CategoryEndpoints
	if err := s.cache.GetJSON(ctx, cacheKey, &cachedResult); err == nil && len(cachedResult) > 0 {
		return cachedResult, nil
	}

	userPerms, err := s.getUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户权限失败: %w", err)
	}

	permSet := make(map[string]bool)
	for _, p := range userPerms {
		permSet[p] = true
	}

	var result []CategoryEndpoints
	for _, module := range s.metadataConfig.Metadata {
		var accessibleEndpoints []EndpointDetail

		for _, endpoint := range module.Endpoints {
			if s.hasPermission(permSet, endpoint.Permissions) {
				accessibleEndpoints = append(accessibleEndpoints, EndpointDetail{
					Route:            endpoint.Route,
					Method:           endpoint.Method,
					DisplayName:      endpoint.DisplayName,
					Description:      endpoint.Description,
					Module:           module.Module,
					Category:         module.Category,
					Icon:             module.Icon,
					DataType:         endpoint.DataType,
					DataPath:         endpoint.DataPath,
					SupportedWidgets: endpoint.SupportedWidgets,
					ExampleParams:    endpoint.ExampleParams,
					RequiredPerms:    endpoint.Permissions,
				})
			}
		}

		if len(accessibleEndpoints) > 0 {
			result = append(result, CategoryEndpoints{
				Module:    module.Module,
				Category:  module.Category,
				Icon:      module.Icon,
				Endpoints: accessibleEndpoints,
			})
		}
	}

	// 缓存写入失败不影响业务，显式忽略错误
	_ = s.cache.SetJSON(ctx, cacheKey, result, 5*time.Minute)

	return result, nil
}

// getUserPermissions 获取用户权限列表
func (s *APIEndpointService) getUserPermissions(ctx context.Context, userID string) ([]string, error) {
	var permissions []string

	err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT m.perms
		FROM sys_menu m
		INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND m.perms IS NOT NULL
		AND m.perms != ''
		AND m.status = 0
	`, userID).Scan(&permissions).Error

	return permissions, err
}

// hasPermission 检查用户是否有所需权限
func (s *APIEndpointService) hasPermission(
	userPerms map[string]bool,
	required []string,
) bool {
	if len(required) == 0 {
		return true
	}

	for _, req := range required {
		if userPerms[req] {
			return true
		}
	}

	return false
}

// ValidateEndpoint 验证端点是否存在
func (s *APIEndpointService) ValidateEndpoint(route, method string) (*EndpointDetail, error) {
	endpointMeta := s.metadataConfig.GetEndpointByRoute(route, method)
	if endpointMeta == nil {
		return nil, fmt.Errorf("endpoint not found: %s %s", method, route)
	}

	var category, icon, module string
	for _, mod := range s.metadataConfig.Metadata {
		for _, ep := range mod.Endpoints {
			if ep.Route == route && ep.Method == method {
				category = mod.Category
				icon = mod.Icon
				module = mod.Module
				break
			}
		}
		if category != "" {
			break
		}
	}

	return &EndpointDetail{
		Route:            endpointMeta.Route,
		Method:           endpointMeta.Method,
		DisplayName:      endpointMeta.DisplayName,
		Description:      endpointMeta.Description,
		Module:           module,
		Category:         category,
		Icon:             icon,
		DataType:         endpointMeta.DataType,
		DataPath:         endpointMeta.DataPath,
		SupportedWidgets: endpointMeta.SupportedWidgets,
		ExampleParams:    endpointMeta.ExampleParams,
		RequiredPerms:    endpointMeta.Permissions,
	}, nil
}

// InvalidateUserCache 清除用户端点缓存
func (s *APIEndpointService) InvalidateUserCache(ctx context.Context, userID string) {
	cacheKey := fmt.Sprintf("user_endpoints:%s", userID)
	_ = s.cache.Delete(ctx, cacheKey)
}

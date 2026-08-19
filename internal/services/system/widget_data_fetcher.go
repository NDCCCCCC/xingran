package system

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/transform"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// WidgetDataFetcher Widget 数据获取接口
type WidgetDataFetcher interface {
	FetchWidgetData(ctx context.Context, widget *models.WidgetConfig, userID string, bypassCache bool) (interface{}, bool, error)
	FetchBatchWidgetData(ctx context.Context, widgetIDs []string, userID string, bypassCache bool) (map[string]WidgetDataResult, error)
}

// WidgetDataResult Widget 数据获取结果
type WidgetDataResult struct {
	Data   interface{} `json:"data,omitempty"`   // 成功时的数据
	Error  string      `json:"error,omitempty"`  // 失败时的错误消息
	Code   int         `json:"code,omitempty"`   // 错误码（可选）
	Cached bool        `json:"cached"`           // 是否来自缓存
}

// WidgetDataFetcherImpl Widget 数据获取实现
type WidgetDataFetcherImpl struct {
	db               *gorm.DB
	cache            cache.Cache
	endpointService  EndpointService
	serviceRegistry  ServiceRegistry
	jsonataEvaluator *transform.JSONataEvaluator
}

// ServiceRegistry 服务注册表接口，用于根据端点路由分发到对应 Service
type ServiceRegistry interface {
	CallService(ctx context.Context, endpoint string, method string, params map[string]interface{}, userID string) (interface{}, error)
}

// NewWidgetDataFetcher 创建 Widget 数据获取器
func NewWidgetDataFetcher(db *gorm.DB, cache cache.Cache, endpointService EndpointService, serviceRegistry ServiceRegistry) *WidgetDataFetcherImpl {
	return &WidgetDataFetcherImpl{
		db:               db,
		cache:            cache,
		endpointService:  endpointService,
		serviceRegistry:  serviceRegistry,
		jsonataEvaluator: transform.NewJSONataEvaluator(),
	}
}

// buildWidgetCacheKey 构建缓存 Key
func buildWidgetCacheKey(widgetID string, params map[string]interface{}) string {
	if len(params) == 0 {
		return fmt.Sprintf("widget:data:%s", widgetID)
	}
	paramsJSON, _ := json.Marshal(params)
	paramsHash := sha256.Sum256(paramsJSON)
	return fmt.Sprintf("widget:data:%s:%x", widgetID, paramsHash[:8])
}

// widgetDefaultCacheTTL Widget 默认缓存 TTL（per D-06 决定）
const widgetDefaultCacheTTL = 5 * time.Minute

// widgetMaxConcurrency Widget 并发抓取上限（per D-03 决定）
const widgetMaxConcurrency = 10

// widgetFetchTimeout Widget 单个抓取超时
const widgetFetchTimeout = 5 * time.Second

// calculateTTL 计算缓存 TTL
func calculateTTL(refreshInterval int) time.Duration {
	if refreshInterval > 0 {
		return time.Duration(refreshInterval) * time.Second
	}
	return widgetDefaultCacheTTL
}

// FetchWidgetData 获取 Widget 数据
func (f *WidgetDataFetcherImpl) FetchWidgetData(ctx context.Context, widget *models.WidgetConfig, userID string, bypassCache bool) (interface{}, bool, error) {
	// 1. 检查缓存（如果未绕过）
	if !bypassCache && f.cache != nil {
		cacheKey := buildWidgetCacheKey(widget.ID, f.extractParams(widget))
		var cachedData interface{}
		if err := f.cache.GetJSON(ctx, cacheKey, &cachedData); err == nil {
			return cachedData, true, nil // 返回缓存数据和 cached=true 标志
		}
	}

	// 2. 根据数据源类型获取数据
	var data interface{}
	var err error

	switch {
	case widget.DataSource.Api != nil:
		data, err = f.fetchFromAPISource(ctx, widget.DataSource.Api, userID)
		if err != nil {
			return nil, false, err
		}
		// 应用数据转换（per D-02）
		if widget.DataSource.Api.Transform != nil {
			data, err = f.jsonataEvaluator.Transform(data, widget.DataSource.Api.Transform)
			if err != nil {
				return nil, false, fmt.Errorf("transform failed: %w", err)
			}
		}
	case widget.DataSource.Static != nil:
		data = widget.DataSource.Static.Data
		err = nil
	case widget.DataSource.WebSocket != nil:
		// Phase 6 实现，当前返回占位符
		data = map[string]interface{}{"status": "websocket_not_implemented"}
		err = nil
	default:
		return nil, false, fmt.Errorf("no data source configured for widget %s", widget.ID)
	}

	if err != nil {
		return nil, false, err
	}

	// 3. 写入缓存（如果未绕过）
	if !bypassCache && f.cache != nil {
		cacheKey := buildWidgetCacheKey(widget.ID, f.extractParams(widget))
		ttl := calculateTTL(widget.RefreshInterval)
		_ = f.cache.SetJSON(ctx, cacheKey, data, ttl)
	}

	return data, false, nil // 返回数据和 cached=false 标志
}

// extractParams 从 Widget 配置中提取参数
func (f *WidgetDataFetcherImpl) extractParams(widget *models.WidgetConfig) map[string]interface{} {
	if widget.DataSource.Api != nil && widget.DataSource.Api.Params != nil {
		return widget.DataSource.Api.Params
	}
	return nil
}

// fetchFromAPISource 从 API 数据源获取数据
func (f *WidgetDataFetcherImpl) fetchFromAPISource(ctx context.Context, apiConfig *models.ApiDataSourceConfig, userID string) (interface{}, error) {
	// 1. 验证端点存在
	if f.endpointService == nil {
		return nil, fmt.Errorf("endpoint service not available")
	}

	endpointMeta, err := f.endpointService.ValidateEndpoint(apiConfig.Endpoint, apiConfig.Method)
	if err != nil {
		return nil, fmt.Errorf("endpoint not found: %w", err)
	}

	// 2. 获取用户权限
	userPerms, err := f.getUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 3. 检查权限（用户必须有该端点的访问权限）
	permSet := make(map[string]bool)
	for _, p := range userPerms {
		permSet[p] = true
	}

	for _, required := range endpointMeta.RequiredPerms {
		if !permSet[required] {
			return nil, fmt.Errorf("permission denied: missing %s", required)
		}
	}

	// 4. 调用内部服务（per D-01）
	if f.serviceRegistry == nil {
		return nil, fmt.Errorf("service registry not available")
	}

	return f.serviceRegistry.CallService(ctx, apiConfig.Endpoint, apiConfig.Method, apiConfig.Params, userID)
}

// getUserPermissions 获取用户权限列表
func (f *WidgetDataFetcherImpl) getUserPermissions(ctx context.Context, userID string) ([]string, error) {
	if f.db == nil {
		return []string{}, nil
	}

	var permissions []string
	err := f.db.WithContext(ctx).Raw(`
		SELECT DISTINCT m.perms
		FROM sys_menu m
		INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND m.perms IS NOT NULL
		AND m.perms != ''
		AND m.status = ?
	`, userID, models.MenuStatusNormal).Scan(&permissions).Error

	return permissions, err
}

// FetchBatchWidgetData 批量获取 Widget 数据
func (f *WidgetDataFetcherImpl) FetchBatchWidgetData(ctx context.Context, widgetIDs []string, userID string, bypassCache bool) (map[string]WidgetDataResult, error) {
	results := make(map[string]WidgetDataResult)
	resultsMutex := sync.Mutex{}

	// 创建 errgroup，设置最大并发数（per D-03）
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(widgetMaxConcurrency)

	for _, widgetID := range widgetIDs {
		widgetID := widgetID // 捕获循环变量

		g.Go(func() error {
			// 设置单个 Widget 超时
			widgetCtx, cancel := context.WithTimeout(ctx, widgetFetchTimeout)
			defer cancel()

			// 查询 Widget 配置
			var widget models.WidgetConfig
			if err := f.db.WithContext(widgetCtx).Where("id = ?", widgetID).First(&widget).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					resultsMutex.Lock()
					results[widgetID] = WidgetDataResult{
						Error: fmt.Sprintf("widget not found: %s", widgetID),
						Code:  404,
					}
					resultsMutex.Unlock()
					return nil // 不中断其他 goroutine（per D-04）
				}
				resultsMutex.Lock()
				results[widgetID] = WidgetDataResult{
					Error: fmt.Sprintf("failed to fetch widget: %v", err),
					Code:  500,
				}
				resultsMutex.Unlock()
				return nil
			}

			// 获取 Widget 数据
			data, cached, err := f.FetchWidgetData(widgetCtx, &widget, userID, bypassCache)

			resultsMutex.Lock()
			if err != nil {
				results[widgetID] = WidgetDataResult{
					Error:  err.Error(),
					Code:   500,
					Cached: false,
				}
			} else {
				results[widgetID] = WidgetDataResult{
					Data:   data,
					Cached: cached,
				}
			}
			resultsMutex.Unlock()

			return nil // 不中断其他 goroutine（per D-04）
		})
	}

	// 等待所有 goroutine 完成
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// DefaultServiceRegistry 默认服务注册表实现
type DefaultServiceRegistry struct {
	// 注入常用 Service（根据项目实际情况添加）
	// workorderService WorkOrderService
	// networkService   NetworkService
	// monitorService   MonitorService
}

// NewDefaultServiceRegistry 创建默认服务注册表
func NewDefaultServiceRegistry() *DefaultServiceRegistry {
	return &DefaultServiceRegistry{}
}

// CallService 调用服务
func (r *DefaultServiceRegistry) CallService(ctx context.Context, endpoint string, method string, params map[string]interface{}, userID string) (interface{}, error) {
	// 根据端点路由分发到对应 Service
	// 当前返回占位符，实际实现需要根据项目 Service 结构添加
	switch endpoint {
	// case "/workorder/orders/list":
	//     return r.workorderService.List(ctx, params)
	// case "/network/devices/list":
	//     return r.networkService.ListDevices(ctx, params)
	default:
		return map[string]interface{}{
			"endpoint": endpoint,
			"method":   method,
			"params":   params,
			"message":  "service call placeholder - implement based on project services",
		}, nil
	}
}

// Ensure DefaultServiceRegistry implements ServiceRegistry
var _ ServiceRegistry = (*DefaultServiceRegistry)(nil)

// Ensure WidgetDataFetcherImpl implements WidgetDataFetcher
var _ WidgetDataFetcher = (*WidgetDataFetcherImpl)(nil)

// EndpointDetailAlias 是 services.EndpointDetail 的别名，用于避免循环导入
type EndpointDetailAlias = services.EndpointDetail

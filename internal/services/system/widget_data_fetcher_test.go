package system

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-07 收尾:widget_data_fetcher 私有方法 + cache hit + stub
// endpoint/registry 路径。WidgetConfig 通过缓存 roundtrip 规避 GORM 读
// 写 widget_configs 表(Position 无 Valuer/Scanner 的 QUIRK,见
// dashboard_service_test.go)。
// =====================================================================

func newWDFTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// cache=shared + 同名 named-memory:同进程内多次 Open 同 DSN 共享同一内存库,
	// 避免 widget_configs 表创建/查询跨连接看不到。
	db, err := gorm.Open(sqlite.Open("file:wdf_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	// getUserPermissions 需 sys_menu.perms + status 列,搭最小菜单/关联 schema
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_menu (id TEXT PRIMARY KEY, perms TEXT, status INTEGER DEFAULT 0)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_role_menu (role_id TEXT, menu_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	return db
}

func newWidgetFetcher(t *testing.T, db *gorm.DB, endpoint EndpointService, registry ServiceRegistry) *WidgetDataFetcherImpl {
	t.Helper()
	if registry == nil {
		registry = NewDefaultServiceRegistry()
	}
	return NewWidgetDataFetcher(db, pkgcache.NewMemoryCache(50, 0), endpoint, registry)
}

// stubEndpointService 满足 EndpointService 的最小实现。
type stubEndpointService struct {
	EndpointService
	validateFn func(route, method string) (*services.EndpointDetail, error)
}

func (s *stubEndpointService) ValidateEndpoint(route, method string) (*services.EndpointDetail, error) {
	return s.validateFn(route, method)
}

// stubServiceRegistry 把 endpoint 字符串返回固定结果用于覆盖 CallService 分支。
type stubServiceRegistry struct {
	ServiceRegistry
	result interface{}
	err    error
}

func (s *stubServiceRegistry) CallService(ctx context.Context, endpoint, method string, params map[string]interface{}, userID string) (interface{}, error) {
	return s.result, s.err
}

func TestWidgetFetcher_ExtractParams(t *testing.T) {
	f := newWidgetFetcher(t, newWDFTestDB(t), nil, nil)

	// Api 非 nil 且有 params → 返 params
	want := map[string]interface{}{"k": "v"}
	got := f.extractParams(&models.WidgetConfig{
		DataSource: models.DataSourceConfig{
			Api: &models.ApiDataSourceConfig{Params: want},
		},
	})
	assert.Equal(t, want, got)

	// Api 为 nil → nil
	got = f.extractParams(&models.WidgetConfig{DataSource: models.DataSourceConfig{}})
	assert.Nil(t, got)

	// Api 非 nil 但 params nil → nil
	got = f.extractParams(&models.WidgetConfig{
		DataSource: models.DataSourceConfig{Api: &models.ApiDataSourceConfig{}},
	})
	assert.Nil(t, got)
}

func TestWidgetFetcher_GetUserPermissions_DBShortCircuit(t *testing.T) {
	f := &WidgetDataFetcherImpl{} // db=nil
	perms, err := f.getUserPermissions(context.Background(), "u1")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestWidgetFetcher_GetUserPermissions_SQLHit(t *testing.T) {
	db := newWDFTestDB(t)
	// 种 2 菜单:menu-1 有 perms 'sys:user:list', menu-2 无 perms
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m1', 'sys:user:list', 0)`)
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m2', '', 0)`)
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m3', 'sys:user:edit', 1)`) // disabled
	db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r1', 'm1'), ('r1', 'm2'), ('r1', 'm3')`)
	db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('u1', 'r1')`)

	f := newWidgetFetcher(t, db, nil, nil)
	perms, err := f.getUserPermissions(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"sys:user:list"}, perms, "应只返回启用菜单的非空 perms")
}

func TestWidgetFetcher_FetchWidgetData_CacheHit(t *testing.T) {
	mem := pkgcache.NewMemoryCache(50, 0)
	f := NewWidgetDataFetcher(nil, mem, nil, nil)
	ctx := context.Background()

	w := &models.WidgetConfig{
		ID: "w1", Type: "stat", Title: "t",
		DataSource: models.DataSourceConfig{Static: &models.StaticDataSourceConfig{Data: "fresh"}},
	}
	// 预写缓存
	require.NoError(t, mem.SetJSON(ctx, buildWidgetCacheKey("w1", nil), "cached", 0))

	got, cached, err := f.FetchWidgetData(ctx, w, "u1", false)
	require.NoError(t, err)
	assert.True(t, cached)
	assert.Equal(t, "cached", got, "应命中缓存而非走 data source")
}

func TestWidgetFetcher_FetchWidgetData_StaticAndWebSocket(t *testing.T) {
	mem := pkgcache.NewMemoryCache(10, 0)
	f := newWidgetFetcher(t, newWDFTestDB(t), nil, nil)
	f.cache = mem
	ctx := context.Background()

	// Static
	w := &models.WidgetConfig{
		ID: "ws", Type: "stat", Title: "s",
		DataSource: models.DataSourceConfig{Static: &models.StaticDataSourceConfig{Data: map[string]int{"v": 1}}},
	}
	got, _, err := f.FetchWidgetData(ctx, w, "u1", true)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"v": 1}, got)

	// WebSocket 占位
	w = &models.WidgetConfig{
		ID: "ww", Type: "table", Title: "w",
		DataSource: models.DataSourceConfig{WebSocket: &models.WebSocketDataSourceConfig{Channel: "ch"}},
	}
	got, _, err = f.FetchWidgetData(ctx, w, "u1", true)
	require.NoError(t, err)
	assert.Equal(t, "websocket_not_implemented", got.(map[string]interface{})["status"])

	// Default(无任何 datasource)→ err
	w = &models.WidgetConfig{ID: "wx", Type: "x", Title: "x"}
	_, _, err = f.FetchWidgetData(ctx, w, "u1", true)
	require.ErrorContains(t, err, "no data source configured")
}

func TestWidgetFetcher_FetchFromAPISource(t *testing.T) {
	ep := &stubEndpointService{}
	registry := &stubServiceRegistry{result: "data", err: nil}

	// endpointService=nil → err
	f := &WidgetDataFetcherImpl{}
	_, err := f.fetchFromAPISource(context.Background(), &models.ApiDataSourceConfig{Endpoint: "/x", Method: "GET"}, "u1")
	require.ErrorContains(t, err, "endpoint service not available")

	// ValidateEndpoint err
	ep.validateFn = func(_, _ string) (*services.EndpointDetail, error) { return nil, errors.New("bad") }
	f = newWidgetFetcher(t, newWDFTestDB(t), ep, registry)
	_, err = f.fetchFromAPISource(context.Background(), &models.ApiDataSourceConfig{Endpoint: "/x", Method: "GET"}, "u1")
	require.ErrorContains(t, err, "endpoint not found")

	// 端点需要 perm 但用户没权限
	ep.validateFn = func(_, _ string) (*services.EndpointDetail, error) {
		return &services.EndpointDetail{RequiredPerms: []string{"sys:user:list"}}, nil
	}
	_, err = f.fetchFromAPISource(context.Background(), &models.ApiDataSourceConfig{Endpoint: "/x", Method: "GET"}, "u1")
	require.ErrorContains(t, err, "permission denied")

	// 完整成功路径: validateFn 切回无 perm 要求,registry 桩返回 "data"
	ep.validateFn = func(_, _ string) (*services.EndpointDetail, error) { return &services.EndpointDetail{}, nil }
	f = newWidgetFetcher(t, newWDFTestDB(t), ep, registry)
	got, err := f.fetchFromAPISource(context.Background(), &models.ApiDataSourceConfig{Endpoint: "/x", Method: "GET"}, "u1")
	require.NoError(t, err)
	assert.Equal(t, "data", got)
}

func TestWidgetFetcher_FetchBatchWidgetData(t *testing.T) {
	// widget_configs 表的 Position 列既无 Valuer 也无 Scanner(Q-12 QUIRK),
	// GORM 读写 widget 行均不可达,只测 widget-not-found 分支。
	db := newWDFTestDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE widget_configs (
		id TEXT PRIMARY KEY, type TEXT, title TEXT,
		position TEXT, data_source TEXT, display TEXT,
		refresh_interval INTEGER DEFAULT 0, enabled INTEGER DEFAULT 1
	)`).Error)
	f := newWidgetFetcher(t, db, nil, nil)
	res, err := f.FetchBatchWidgetData(context.Background(), []string{"ghost-a", "ghost-b"}, "u1", false)
	require.NoError(t, err)
	require.Len(t, res, 2)
	for _, id := range []string{"ghost-a", "ghost-b"} {
		assert.Equal(t, 404, res[id].Code)
		assert.Contains(t, res[id].Error, "widget not found")
	}
}
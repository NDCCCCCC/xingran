package monitor

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// SetupCacheRouter 的 task-4 权限接线段（读 ANY-of / 写 strict 端点级中间件）
// 此前无运行时触达。pkg/cache.Cache 用内存 MemoryCache，
// DataCacheService 走 NewDataCacheService 包装。

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

func ci87MonitorSmokeCore(t *testing.T, withCache bool) *core.Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	}
	if withCache {
		c.DataCacheService = services.NewDataCacheService(cache.NewMemoryCache(64, time.Minute))
	}
	return c
}

// TestSetupCacheRouter_RegistersPermissionWiredRoutes 缓存监控全路由注册
//（DataCacheService 非 nil 分支）。
func TestSetupCacheRouter_RegistersPermissionWiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testCore := ci87MonitorSmokeCore(t, true)

	engine := gin.New()
	rg := engine.Group("/monitor")

	require.NotPanics(t, func() {
		SetupCacheRouter(rg, testCore)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodPost + " /monitor/cache/list",
		http.MethodGet + " /monitor/cache/:key",
		http.MethodPost + " /monitor/cache/operate",
		http.MethodPost + " /monitor/cache/batch",
		http.MethodPost + " /monitor/cache/clear",
		http.MethodPost + " /monitor/cache/stats/list",
		http.MethodPost + " /monitor/cache/monitor",
		http.MethodPost + " /monitor/cache/export",
		http.MethodGet + " /monitor/cache/config",
		http.MethodPut + " /monitor/cache/config",
		http.MethodPost + " /monitor/cache/config/reload",
		http.MethodGet + " /monitor/cache/test",
		http.MethodGet + " /monitor/cache/debug/raw-keys",
		http.MethodGet + " /monitor/cache/debug/l1",
		http.MethodPost + " /monitor/cache/stats",
		http.MethodPost + " /monitor/cache/invalidate",
		http.MethodPost + " /monitor/cache/invalidate-pattern",
		http.MethodPost + " /monitor/cache/warmup",
		http.MethodPost + " /monitor/cache/key-info",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}

// TestSetupCacheRouter_NilDataCacheService 兜底分支：CacheConfigService 为 nil
// 时 SetupCacheRouter 自建 CacheConfigService，仍应全量注册。
func TestSetupCacheRouter_NilDataCacheService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testCore := ci87MonitorSmokeCore(t, false)

	engine := gin.New()
	rg := engine.Group("/monitor")
	require.NotPanics(t, func() {
		SetupCacheRouter(rg, testCore)
	})
	assert.GreaterOrEqual(t, len(engine.Routes()), 19)
}

package system

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// SetupDefaultThemeRouter 的 NewConfigServiceWithCache 分支
// （core.DataCacheService != nil 路径）此前无运行时触达。

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

// TestSetupDefaultThemeRouter_WithCacheBranch 走 NewConfigServiceWithCache
// 分支（DataCacheService 非 nil）并注册 3 个主题端点。
func TestSetupDefaultThemeRouter_WithCacheBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}
	c.DataCacheService = services.NewDataCacheService(cache.NewMemoryCache(64, time.Minute))

	engine := gin.New()
	rg := engine.Group("/system/settings/config/theme")

	require.NotPanics(t, func() {
		SetupDefaultThemeRouter(rg, c)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodGet + " /system/settings/config/theme/default",
		http.MethodPost + " /system/settings/config/theme/default",
		http.MethodPost + " /system/settings/config/theme/sync",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}

// TestSetupDefaultThemeRouter_NoCacheBranch 走 else 分支
// （NewConfigService 裸 DB 版本）。
func TestSetupDefaultThemeRouter_NoCacheBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newRouterSmokeCore(t)

	engine := gin.New()
	rg := engine.Group("/system/settings/config/theme")

	require.NotPanics(t, func() {
		SetupDefaultThemeRouter(rg, c)
	})
	assert.Len(t, engine.Routes(), 3)
}

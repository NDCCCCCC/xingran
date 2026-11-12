package network

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-4 权限接线改动（SetupMACRouter / SetupMACHistoryRouter /
// SetupTopologyRouter 的 MAC 规则段）此前无运行时触达。
// 模式对齐 internal/api/v1/system/setup_routers_full_test.go。

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
)

func newCI87NetSmokeCore(t *testing.T) (*core.Core, *gorm.DB) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}, gdb
}

func ci87NetRouteSet(engine *gin.Engine) map[string]bool {
	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	return set
}

// TestSetupMACRouter_RegistersPermissionWiredRoutes MAC 地址 7 端点。
func TestSetupMACRouter_RegistersPermissionWiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := newCI87NetSmokeCore(t)
	engine := gin.New()
	rg := engine.Group("/network/mac")

	require.NotPanics(t, func() {
		SetupMACRouter(rg, c, NewNetworkExportHandler(c))
	})

	routes := ci87NetRouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /network/mac/list",
		http.MethodPost + " /network/mac/collect",
		http.MethodPost + " /network/mac/collect/all",
		http.MethodGet + " /network/mac/statistics",
		http.MethodPost + " /network/mac/clean",
		http.MethodPost + " /network/mac/batch-delete",
		http.MethodPost + " /network/mac/export",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupMACHistoryRouter_RegistersPermissionWiredRoutes MAC 历史查询 7 端点。
func TestSetupMACHistoryRouter_RegistersPermissionWiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := newCI87NetSmokeCore(t)
	engine := gin.New()
	rg := engine.Group("/network/mac")

	require.NotPanics(t, func() {
		SetupMACHistoryRouter(rg, c)
	})

	routes := ci87NetRouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /network/mac/history/port",
		http.MethodPost + " /network/mac/history/device",
		http.MethodPost + " /network/mac/history/stats",
		http.MethodPost + " /network/mac/history/vendor",
		http.MethodPost + " /network/mac/history/list",
		http.MethodGet + " /network/mac/history/list",
		http.MethodPost + " /network/mac/history/heatmap",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupTopologyRouter_RegistersMACRuleRoutes 拓扑 MAC 过滤规则段
// （task-4：规则读写拆 ANY-of 读 / strict 写）。
func TestSetupTopologyRouter_RegistersMACRuleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, gdb := newCI87NetSmokeCore(t)
	engine := gin.New()
	rg := engine.Group("/network")

	require.NotPanics(t, func() {
		SetupTopologyRouter(rg, gdb, c)
	})

	routes := ci87NetRouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /network/topology/rules/list",
		http.MethodPost + " /network/topology/rules",
		http.MethodPost + " /network/topology/rules/:id",
		http.MethodPost + " /network/topology/rules/:id/update",
		http.MethodPost + " /network/topology/rules/:id/delete",
		http.MethodGet + " /network/topology/rules/effective",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

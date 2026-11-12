package duty

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-3 G1 值班域读/写兄弟拆分新增的 Setup*WriteRouter / Setup*ConfigRouter /
// SetupMyDutyRouter 此前无运行时触达。模式对齐
// internal/api/v1/system/setup_routers_full_test.go：内存 sqlite core +
// gin engine，真实调用 Setup 注册，断言 engine.Routes()。

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

func newCI87SmokeCore(t *testing.T) *core.Core {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}
}

// TestSetupDutyPoolsWriteRouter_RegistersActionRoutes 值班池写三动作 strict 接线。
func TestSetupDutyPoolsWriteRouter_RegistersActionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	root := engine.Group("/duty/pools")

	groups := DutyPoolsWriteRouteGroups{
		Add:    root.Group(""),
		Edit:   root.Group(""),
		Delete: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupDutyPoolsWriteRouter(groups, c)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodPost + " /duty/pools",
		http.MethodPost + " /duty/pools/:id/update",
		http.MethodPost + " /duty/pools/:id/delete",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}

// TestSetupDutySchedulesWriteRouter_RegistersActionRoutes 排班写五动作 strict 接线。
func TestSetupDutySchedulesWriteRouter_RegistersActionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	root := engine.Group("/duty/schedules")

	groups := DutySchedulesWriteRouteGroups{
		Edit:   root.Group(""),
		Delete: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupDutySchedulesWriteRouter(groups, c)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodPost + " /duty/schedules/generate",
		http.MethodPost + " /duty/schedules/swap",
		http.MethodPost + " /duty/schedules/manual",
		http.MethodPost + " /duty/schedules/:id/delete",
		http.MethodPost + " /duty/schedules/batch-delete",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}

// TestSetupDutyConfigAndMyDutyRouters_Register 值班配置读组/写组与我的值班路由。
func TestSetupDutyConfigAndMyDutyRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	cfg := engine.Group("/duty/config")
	my := engine.Group("/duty/my")

	require.NotPanics(t, func() {
		SetupDutyConfigRouter(cfg, c)
		SetupDutyConfigWriteRouter(cfg, c)
		SetupMyDutyRouter(my, c)
	})

	paths := map[string]bool{}
	for _, r := range engine.Routes() {
		paths[r.Method+" "+r.Path] = true
	}
	assert.True(t, paths[http.MethodPost+" /duty/config"])
	assert.True(t, paths[http.MethodPost+" /duty/config/update"])
	assert.True(t, len(engine.Routes()) >= 3, "my-duty routes should be registered")
}

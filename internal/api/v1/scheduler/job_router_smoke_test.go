package scheduler

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// SetupJobRouter 的 monitor 域权限码全链路接线（读 ANY-of / 写 strict
// 端点级中间件）此前无运行时触达。core.Scheduler 传 nil（setup 仅存储
// 不调用），DB 用内存 sqlite。

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

// TestSetupJobRouter_RegistersPermissionWiredRoutes 定时任务 10 端点注册。
func TestSetupJobRouter_RegistersPermissionWiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}

	engine := gin.New()
	rg := engine.Group("/monitor/jobs")

	require.NotPanics(t, func() {
		SetupJobRouter(rg, c)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodPost + " /monitor/jobs/list",
		http.MethodPost + " /monitor/jobs",
		http.MethodPost + " /monitor/jobs/:id",
		http.MethodPost + " /monitor/jobs/:id/update",
		http.MethodPost + " /monitor/jobs/:id/delete",
		http.MethodPost + " /monitor/jobs/:id/status",
		http.MethodPost + " /monitor/jobs/:id/execute",
		http.MethodPost + " /monitor/jobs/logs/list",
		http.MethodPost + " /monitor/jobs/logs/statistics",
		http.MethodPost + " /monitor/jobs/logs/clean",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}

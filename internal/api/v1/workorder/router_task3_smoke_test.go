package workorder

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-3 G1 工单域读/写兄弟拆分新增的 Setup*Router / Setup*WriteRouter
// 此前无运行时触达。复用同包 setupTestDB fixture；模式对齐
// periodic_actions_handler_test.go 的 TestSetupPeriodicWorkOrderRouter_RegistersActionRoutes。

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
)

func ci87WorkOrderCore(t *testing.T) *core.Core {
	t.Helper()
	return &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: setupTestDB(t)}},
		CoreServices: &core.CoreServices{},
	}
}

func ci87RouteSet(engine *gin.Engine) map[string]bool {
	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	return set
}

// TestSetupWorkOrdersRouters_Register 工单读组 + 写八动作 strict 接线。
func TestSetupWorkOrdersRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/orders")

	groups := WorkOrdersWriteRouteGroups{
		Add:    root.Group(""),
		Edit:   root.Group(""),
		Delete: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupWorkOrdersRouter(root, c)
		SetupWorkOrdersWriteRouter(groups, c)
	})

	routes := ci87RouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /workorder/orders/list",
		http.MethodPost + " /workorder/orders",
		http.MethodPost + " /workorder/orders/:id/update",
		http.MethodPost + " /workorder/orders/:id/assign",
		http.MethodPost + " /workorder/orders/:id/assign-duty",
		http.MethodPost + " /workorder/orders/:id/status",
		http.MethodPost + " /workorder/orders/:id/comments",
		http.MethodPost + " /workorder/orders/:id/delete",
		http.MethodPost + " /workorder/orders/batch-delete",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupWorkOrderCategoriesRouters_Register 分类读组 + 写三动作 strict 接线。
func TestSetupWorkOrderCategoriesRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	root := engine.Group("/workorder/categories")

	groups := WorkOrderCategoriesWriteRouteGroups{
		Add:    root.Group(""),
		Edit:   root.Group(""),
		Delete: root.Group(""),
	}
	require.NotPanics(t, func() {
		SetupWorkOrderCategoriesRouter(root, c)
		SetupWorkOrderCategoriesWriteRouter(groups, c)
	})

	routes := ci87RouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /workorder/categories/list",
		http.MethodPost + " /workorder/categories",
		http.MethodPost + " /workorder/categories/:id/update",
		http.MethodPost + " /workorder/categories/:id/delete",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupWorkOrderConfigRouters_Register 配置读组 + 写组 strict 接线。
func TestSetupWorkOrderConfigRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := ci87WorkOrderCore(t)
	engine := gin.New()
	rg := engine.Group("/workorder/config")

	require.NotPanics(t, func() {
		SetupWorkOrderConfigRouter(rg, c)
		SetupWorkOrderConfigWriteRouter(rg, c)
	})

	routes := ci87RouteSet(engine)
	assert.True(t, routes[http.MethodPost+" /workorder/config"])
	assert.True(t, routes[http.MethodPost+" /workorder/config/update"])
}

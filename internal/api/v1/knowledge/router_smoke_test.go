package knowledge

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-3 G1 知识库域读/写兄弟拆分新增的 Setup*WriteRouter 及读组此前
// 无运行时触达。模式对齐 internal/api/v1/system/setup_routers_full_test.go。

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

func ci87RouteSet(engine *gin.Engine) map[string]bool {
	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	return set
}

// TestSetupArticleRouters_Register 文章读组 + 写三动作 strict 接线。
func TestSetupArticleRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	read := engine.Group("/knowledge/articles")
	root := engine.Group("/knowledge/articles")

	groups := ArticleWriteRouteGroups{Add: root.Group(""), Edit: root.Group(""), Delete: root.Group("")}
	require.NotPanics(t, func() {
		SetupArticleRouter(read, c)
		SetupArticleWriteRouter(groups, c)
	})

	routes := ci87RouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /knowledge/articles/list",
		http.MethodPost + " /knowledge/articles",
		http.MethodPost + " /knowledge/articles/:id/update",
		http.MethodPost + " /knowledge/articles/:id/delete",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupCategoryRouters_Register 分类读组 + 写三动作 strict 接线。
func TestSetupCategoryRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	read := engine.Group("/knowledge/categories")
	root := engine.Group("/knowledge/categories")

	groups := CategoryWriteRouteGroups{Add: root.Group(""), Edit: root.Group(""), Delete: root.Group("")}
	require.NotPanics(t, func() {
		SetupCategoryRouter(read, c)
		SetupCategoryWriteRouter(groups, c)
	})

	routes := ci87RouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /knowledge/categories/list",
		http.MethodPost + " /knowledge/categories",
		http.MethodPost + " /knowledge/categories/:id/update",
		http.MethodPost + " /knowledge/categories/:id/delete",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupTagRouters_Register 标签读组 + 写三动作 strict 接线。
func TestSetupTagRouters_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	read := engine.Group("/knowledge/tags")
	root := engine.Group("/knowledge/tags")

	groups := TagWriteRouteGroups{Add: root.Group(""), Edit: root.Group(""), Delete: root.Group("")}
	require.NotPanics(t, func() {
		SetupTagRouter(read, c)
		SetupTagWriteRouter(groups, c)
	})

	routes := ci87RouteSet(engine)
	for _, want := range []string{
		http.MethodPost + " /knowledge/tags/all",
		http.MethodPost + " /knowledge/tags",
		http.MethodPost + " /knowledge/tags/:id/update",
		http.MethodPost + " /knowledge/tags/:id/delete",
	} {
		assert.True(t, routes[want], "route not registered: %s", want)
	}
}

// TestSetupKnowledgeWorkOrderRouter_Register 工单转知识库路由。
func TestSetupKnowledgeWorkOrderRouter_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newCI87SmokeCore(t)
	engine := gin.New()
	rg := engine.Group("/knowledge/workorders")

	require.NotPanics(t, func() {
		SetupWorkOrderRouter(rg, c)
	})
	assert.NotEmpty(t, engine.Routes(), "knowledge workorder routes should be registered")
}

package system

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"gorm.io/gorm"
)

// TestSetupDefaultThemeRouterRoutes 冒烟测试：task-7 默认主题 3 条端点注册到位，
// 路径/方法与前端 src/lib/defaultThemeApi.ts 一一对应（system:config 权限中间件
// 在 router.go 组级挂载，不在此测）。sqlite 内存库仅用于装配 handler 依赖。
func TestSetupDefaultThemeRouterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	// DB / DataCacheService 等字段提升自嵌入的 *CoreInfra / *CoreServices，
	// 须先初始化嵌入指针（参考 ad_account_pool_handler_test）。
	c := &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}}
	c.DB = &db.Database{DB: gdb}

	r := gin.New()
	group := r.Group("/system/settings/config/theme")
	SetupDefaultThemeRouter(group, c)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/system/settings/config/theme/default"},
		{http.MethodPost, "/system/settings/config/theme/default"},
		{http.MethodPost, "/system/settings/config/theme/sync"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// 路由命中即可（依赖精简，命中后进入 handler 逻辑返回非 404 状态；
		// 未注册则是 404）。用 404 判定注册缺失。
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s 未注册（404）", tc.method, tc.path)
		}
	}
}

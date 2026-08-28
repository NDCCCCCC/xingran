package services

// =====================================================================
// Phase 79-03 Task 5: oper_log / api_endpoint / swagger_extractor /
// notice_cron_util 四小文件(前缀 TestOpl7903_ / TestAep7903_ / TestSwe7903_ /
// TestNcu7903_)。
//
// 覆盖目标: oper_log_service.go(56 unc)/ api_endpoint_service.go(46)/
// swagger_extractor.go(18)/ notice_cron_util.go(22)各自 0% → ≥70%。
// 纪律:7903 后缀 helper、sqlite t.TempDir 文件库、禁 t.Parallel、
// RecordAsync/RecordFromGinContext 的 goroutine 用 require.Eventually 轮询
// (禁裸 sleep 长等)、businessType 用 operlog.OperType* 具名常量、
// 敏感词断言只取样 CLAUDE.md 强制清单(真相源在 internal/utils/operlog,
// 不在测试内复制清单常量)。
// =====================================================================

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newOpl7903 装配 sqlite 库 + OperLogService + APIEndpointService。
func newOpl7903(t *testing.T) (*gorm.DB, OperLogService, *APIEndpointService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "opl7903.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.OperLog{},
		&models.User{},
		&models.UserRole{},
		&models.Role{},
		&models.RoleMenu{},
		&models.Menu{},
	), "auto migrate operlog/endpoint chain models")

	svc := NewOperLogService()
	endpointSvc := NewAPIEndpointService(aep7903Metadata(), cache.NewMemoryCache(1000, 5*time.Minute), db)
	return db, svc, endpointSvc
}

// aep7903Metadata 构造测试用端点元数据(两模块,含免权限端点与不可达端点)。
func aep7903Metadata() *config.APIMetadataConfig {
	return &config.APIMetadataConfig{
		Version: "test-7903",
		Metadata: []config.ModuleMetadata{
			{
				Module:   "用户管理",
				Category: "系统管理",
				Icon:     "user",
				Endpoints: []config.EndpointMeta{
					{Route: "/system/users/list", Method: "POST", DisplayName: "用户列表",
						Permissions: []string{"system:user:list"}, DataType: "paginated", DataPath: "data.list"},
					{Route: "/system/users/add", Method: "POST", DisplayName: "创建用户",
						Permissions: []string{"system:user:add"}},
					{Route: "/dashboard/stats", Method: "GET", DisplayName: "仪表盘",
						Permissions: nil, DataType: "single"},
				},
			},
			{
				Module:   "空模块",
				Category: "系统管理",
				Icon:     "empty",
				Endpoints: []config.EndpointMeta{
					{Route: "/empty/no-perm", Method: "GET", Permissions: []string{"nope:missing"}},
				},
			},
		},
	}
}

// opl7903Count 统计 sys_oper_log 行数。
func opl7903Count(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&n).Error)
	return n
}

// svcImpl7903 取出 OperLogService 的私有实现(FilterSensitiveParams /
// getRequestParams / getResponseResult 定义在实现上而非接口上)。
func svcImpl7903(t *testing.T, svc OperLogService) *operLogService {
	t.Helper()
	impl, ok := svc.(*operLogService)
	require.True(t, ok, "OperLogService 实装类型断言")
	return impl
}

// TestOpl7903_RecordOperLog_Direct 直调 RecordOperLog → 落库字段一致。
func TestOpl7903_RecordOperLog_Direct(t *testing.T) {
	db, svc, _ := newOpl7903(t)

	operName := "operator-a"
	operURL := "/system/users/add"
	param := `{"username":"alice"}`
	operTime := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	require.NoError(t, svc.RecordOperLog(context.Background(), db, &models.OperLog{
		Title:         "用户管理",
		BusinessType:  operlog.OperTypeCreate,
		Method:        "CreateUser",
		RequestMethod: "POST",
		OperatorName:  &operName,
		OperUrl:       &operURL,
		OperParam:     &param,
		Status:        int(models.OperLogStatusSuccess),
		CostTime:      42,
		OperTime:      operTime,
	}))

	require.Equal(t, int64(1), opl7903Count(t, db))
	var got models.OperLog
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, "用户管理", got.Title)
	assert.Equal(t, operlog.OperTypeCreate, got.BusinessType, "businessType=新增(operlog 具名常量)")
	assert.Equal(t, "CreateUser", got.Method)
	assert.Equal(t, "POST", got.RequestMethod)
	require.NotNil(t, got.OperatorName)
	assert.Equal(t, "operator-a", *got.OperatorName)
	require.NotNil(t, got.OperUrl)
	assert.Equal(t, operURL, *got.OperUrl)
	require.NotNil(t, got.OperParam)
	assert.Equal(t, param, *got.OperParam)
	assert.Equal(t, int(models.OperLogStatusSuccess), got.Status)
	assert.Equal(t, int64(42), got.CostTime)
	assert.True(t, got.OperTime.Equal(operTime), "OperTime 原样落库(RecordOperLog 为直写,不代填时间)")
}

// TestOpl7903_RecordAsync 异步落库:require.Eventually 轮询(2s 超时,禁裸 sleep 长等)。
func TestOpl7903_RecordAsync(t *testing.T) {
	db, svc, _ := newOpl7903(t)

	operName := "async-operator"
	nickname := "异步操作员"
	deptName := "运维部"
	operIP := "10.0.0.8"
	operParam := `{"id":"1"}`
	jsonResult := `{"code":0}`

	svc.RecordAsync(db, "缓存监控", operlog.OperTypeClean, "CleanCache", "DELETE",
		"/monitor/cache/clean", &operName, &nickname, &deptName, &operIP,
		&operParam, &jsonResult, nil, int(models.OperLogStatusSuccess), 17)

	require.Eventually(t, func() bool {
		return opl7903Count(t, db) == 1
	}, 2*time.Second, 10*time.Millisecond, "异步行应在 2s 内落库")

	var got models.OperLog
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, "缓存监控", got.Title)
	assert.Equal(t, operlog.OperTypeClean, got.BusinessType)
	assert.Equal(t, "CleanCache", got.Method)
	assert.Equal(t, "DELETE", got.RequestMethod)
	require.NotNil(t, got.OperUrl)
	assert.Equal(t, "/monitor/cache/clean", *got.OperUrl)
	require.NotNil(t, got.OperIP)
	assert.Equal(t, "10.0.0.8", *got.OperIP)
	assert.Equal(t, 1, got.OperatorType, "OperatorType 固定 1=后台用户")
	assert.Equal(t, int64(17), got.CostTime)
	require.NotNil(t, got.JsonResult)
}

// TestOpl7903_RecordFromGinContext CreateTestContext 构造 GET/POST 请求 →
// 落库行 method/operUrl/requestMethod/operName 与请求一致;成功/失败两态。
func TestOpl7903_RecordFromGinContext(t *testing.T) {
	db, svc, _ := newOpl7903(t)
	gin.SetMode(gin.TestMode)

	// —— 成功态:POST + claims + query 参数 + response_body + start_time ——
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/system/users/list?name=alice&page=1", nil)
	req.Header.Set("User-Agent", "gtest-7903")
	req.RemoteAddr = "10.1.2.3:5678"
	c.Request = req
	c.Set("claims", map[string]any{"username": "gin-operator"})
	c.Set("response_body", `{"code":0,"message":"success"}`)
	c.Set("start_time", time.Now().Add(-50*time.Millisecond))

	svc.RecordFromGinContext(c, db, "用户管理", operlog.OperTypeOther, "ListUsers")

	require.Eventually(t, func() bool {
		return opl7903Count(t, db) == 1
	}, 2*time.Second, 10*time.Millisecond, "gin 异步行应在 2s 内落库")

	var got models.OperLog
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, "用户管理", got.Title)
	assert.Equal(t, "ListUsers", got.Method)
	assert.Equal(t, "POST", got.RequestMethod, "请求方法透传")
	assert.Equal(t, "/system/users/list?name=alice&page=1", *got.OperUrl, "URL 含 query")
	assert.Equal(t, "10.1.2.3", *got.OperIP, "ClientIP 透传")
	require.NotNil(t, got.OperatorName)
	assert.Equal(t, "gin-operator", *got.OperatorName, "claims.username 覆写默认 unknown")
	assert.Equal(t, int(models.OperLogStatusSuccess), got.Status, "无 c.Errors → 成功态")
	require.NotNil(t, got.JsonResult)
	assert.Equal(t, `{"code":0,"message":"success"}`, *got.JsonResult)
	require.NotNil(t, got.OperParam, "POST 请求记录 query 参数")
	assert.Contains(t, *got.OperParam, "alice")
	assert.GreaterOrEqual(t, got.CostTime, int64(1), "start_time 起算耗时")

	// —— 失败态:c.Errors 非空 → 失败态 + errorMsg 落库 ——
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("GET", "/dashboard/stats", nil)
	req2.RemoteAddr = "10.1.2.4:5678"
	c2.Request = req2
	_ = c2.Error(errors.New("boom-7903"))

	svc.RecordFromGinContext(c2, db, "仪表盘", operlog.OperTypeOther, "Stats")

	require.Eventually(t, func() bool {
		return opl7903Count(t, db) == 2
	}, 2*time.Second, 10*time.Millisecond, "失败行应在 2s 内落库")

	var failed models.OperLog
	require.NoError(t, db.Where("title = ?", "仪表盘").First(&failed).Error)
	assert.Equal(t, int(models.OperLogStatusFailure), failed.Status, "c.Errors 非空 → 失败态")
	require.NotNil(t, failed.ErrorMsg)
	assert.Contains(t, *failed.ErrorMsg, "boom-7903")
	assert.Nil(t, failed.OperParam, "GET 请求不记录参数")
	assert.Nil(t, failed.JsonResult, "未设 response_body → nil")
}

// TestOpl7903_FilterSensitiveParams 表驱动:强制敏感关键词值脱敏为 ******;
// 非敏感字段保留;坏 JSON/非 JSON 输入不 panic(行为按实现锁定)。
func TestOpl7903_FilterSensitiveParams(t *testing.T) {
	_, svc, _ := newOpl7903(t)

	// CLAUDE.md 11 强制关键词抽样全列(真相源 internal/utils/operlog,回归测试锁定;
	// 本测试不断言清单长度,只锁定脱敏行为与 operlog 包一致)。
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"password", `{"password":"hunter2"}`, `{"password":"******"}`},
		{"pwd", `{"pwd":"x"}`, `{"pwd":"******"}`},
		{"secret", `{"secret":"x"}`, `{"secret":"******"}`},
		{"token", `{"token":"x"}`, `{"token":"******"}`},
		{"key", `{"key":"x"}`, `{"key":"******"}`},
		{"salt", `{"salt":"x"}`, `{"salt":"******"}`},
		{"privateKey", `{"privateKey":"x"}`, `{"privateKey":"******"}`},
		{"oldPassword", `{"oldPassword":"x"}`, `{"oldPassword":"******"}`},
		{"macKey", `{"macKey":"x"}`, `{"macKey":"******"}`},
		{"sm4Key", `{"sm4Key":"x"}`, `{"sm4Key":"******"}`},
		{"sm2Key", `{"sm2Key":"x"}`, `{"sm2Key":"******"}`},
		{"case-insensitive", `{"Password":"x"}`, `{"Password":"******"}`},
		{"multi-occurrence", `{"password":"a","note":"n","password":"b"}`,
			`{"password":"******","note":"n","password":"******"}`},
		{"non-sensitive-kept", `{"username":"alice","age":3}`, `{"username":"alice","age":3}`},
		{"mixed", `{"username":"alice","password":"hunter2"}`,
			`{"username":"alice","password":"******"}`},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svcImpl7903(t, svc).FilterSensitiveParams(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}

	// QUIRK-79-03-L(锁定不修):过滤器按 `"<key>":"` 子串定位,非 JSON 形态
	// (如 query 形态 password=abc)不脱敏、原样返回,且不 panic。
	assert.Equal(t, "password=abc&token=zz", svcImpl7903(t, svc).FilterSensitiveParams("password=abc&token=zz"))
	// 坏 JSON 但含 `"key":"` 形态 → 仍按子串规则脱敏(不依赖合法 JSON)
	assert.Equal(t, `{bad json, "key":"******",`, svcImpl7903(t, svc).FilterSensitiveParams(`{bad json, "key":"v",`))
	// 与 operlog 包实现一致性(委托关系锁定)
	assert.Equal(t, operlog.FilterSensitiveParams(`{"password":"p"}`),
		svcImpl7903(t, svc).FilterSensitiveParams(`{"password":"p"}`))

	// getRequestParams/getResponseResult 同包直调(POST 有 query → 参数;GET → nil)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/x?a=1&b=2", nil)
	params := svc.(*operLogService).getRequestParams(c)
	require.NotNil(t, params)
	assert.Contains(t, *params, `"a":"1"`)
	c.Request = httptest.NewRequest("GET", "/x?a=1", nil)
	assert.Nil(t, svc.(*operLogService).getRequestParams(c), "GET 不记录参数")
	assert.Nil(t, svc.(*operLogService).getResponseResult(c), "未设 response_body → nil")
}

// aep7903SeedUser 预置 用户→角色 链(每测试至多调一次)。
func aep7903SeedUser(t *testing.T, db *gorm.DB, userID, roleID string) {
	t.Helper()
	require.NoError(t, db.Create(&models.User{
		BaseModel: models.BaseModel{ID: userID},
		Username:  "user-" + userID,
		Password:  "not-a-real-password",
		Salt:      "salt-7903",
	}).Error, "seed user")
	require.NoError(t, db.Create(&models.UserRole{UserID: userID, RoleID: roleID}).Error)
}

// aep7903SeedMenu 预置菜单(perms/status)并挂到角色。
func aep7903SeedMenu(t *testing.T, db *gorm.DB, roleID, menuID, perms string, status models.MenuStatus) {
	t.Helper()
	require.NoError(t, db.Create(&models.RoleMenu{RoleID: roleID, MenuID: menuID}).Error)
	require.NoError(t, db.Create(&models.Menu{
		BaseModel: models.BaseModel{ID: menuID},
		MenuName:  "菜单-" + menuID,
		Perms:     &perms,
		Status:    status,
	}).Error, "seed menu %s", menuID)
}

// TestAep7903_GetUserAccessibleEndpoints 权限过滤 + 缓存命中 + 缓存失效链。
func TestAep7903_GetUserAccessibleEndpoints(t *testing.T) {
	db, _, endpointSvc := newOpl7903(t)
	ctx := context.Background()

	// u-1 持有 system:user:list 菜单 + 一条停用菜单(不计入权限)+ 一条空 perms 菜单
	aep7903SeedUser(t, db, "u-1", "r-1")
	aep7903SeedMenu(t, db, "r-1", "m-list", "system:user:list", models.MenuStatusNormal)
	aep7903SeedMenu(t, db, "r-1", "m-closed", "system:user:closed", models.MenuStatusStop)
	aep7903SeedMenu(t, db, "r-1", "m-empty", "", models.MenuStatusNormal)

	// 同包直调 getUserPermissions:仅命中正常态非空 perms
	perms, err := endpointSvc.getUserPermissions(ctx, "u-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"system:user:list"}, perms,
		"停用菜单与空 perms 均不计入权限")

	// hasPermission 表驱动:空要求/命中/未命中
	require.True(t, endpointSvc.hasPermission(map[string]bool{"a": true}, nil), "无权限要求 → 放行")
	require.True(t, endpointSvc.hasPermission(map[string]bool{"a": true}, []string{"a", "b"}))
	require.False(t, endpointSvc.hasPermission(map[string]bool{"a": true}, []string{"b"}))
	require.False(t, endpointSvc.hasPermission(map[string]bool{}, []string{"b"}))

	// 可访问端点:模块1 命中 list + 免权限 dashboard;add 与空模块端点不可见
	result, err := endpointSvc.GetUserAccessibleEndpoints(ctx, "u-1")
	require.NoError(t, err)
	require.Len(t, result, 1, "空模块无可见端点 → 整组被剔除")
	assert.Equal(t, "用户管理", result[0].Module)
	assert.Equal(t, "系统管理", result[0].Category)
	assert.Equal(t, "user", result[0].Icon)
	require.Len(t, result[0].Endpoints, 2)
	routes := []string{result[0].Endpoints[0].Route, result[0].Endpoints[1].Route}
	assert.ElementsMatch(t, []string{"/system/users/list", "/dashboard/stats"}, routes)
	for _, ep := range result[0].Endpoints {
		assert.Equal(t, "用户管理", ep.Module)
		assert.Equal(t, "系统管理", ep.Category)
	}
	// 仪表盘(免权限)的 RequiredPerms 为空
	for _, ep := range result[0].Endpoints {
		if ep.Route == "/dashboard/stats" {
			assert.Empty(t, ep.RequiredPerms)
			assert.Equal(t, "single", ep.DataType)
		}
		if ep.Route == "/system/users/list" {
			assert.Equal(t, []string{"system:user:list"}, ep.RequiredPerms)
			assert.Equal(t, "paginated", ep.DataType)
			assert.Equal(t, "data.list", ep.DataPath)
		}
	}

	// 缓存命中证据:删掉授权链后再查 → 仍返回端点(来自缓存)
	require.NoError(t, db.Where("role_id = ?", "r-1").Delete(&models.RoleMenu{}).Error)
	cached, err := endpointSvc.GetUserAccessibleEndpoints(ctx, "u-1")
	require.NoError(t, err)
	require.Len(t, cached, 1, "缓存命中(权限链已删,数据来自缓存)")

	// 缓存失效链:InvalidateUserCache 后重算 → 权限链已删,仅剩免权限的仪表盘端点
	endpointSvc.InvalidateUserCache(ctx, "u-1")
	after, err := endpointSvc.GetUserAccessibleEndpoints(ctx, "u-1")
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Len(t, after[0].Endpoints, 1)
	assert.Equal(t, "/dashboard/stats", after[0].Endpoints[0].Route,
		"失效后重算:仅免权限端点可达(缓存确实失效)")

	// 无任何角色用户 → 仅免权限端点可达(hasPermission 空 required 放行)
	none, err := endpointSvc.GetUserAccessibleEndpoints(ctx, "u-nobody")
	require.NoError(t, err)
	require.Len(t, none, 1)
	require.Len(t, none[0].Endpoints, 1)
	assert.Equal(t, "/dashboard/stats", none[0].Endpoints[0].Route)
}

// TestAep7903_ValidateEndpoint 已知 route+method → EndpointDetail;未知 → 错误。
func TestAep7903_ValidateEndpoint(t *testing.T) {
	_, _, endpointSvc := newOpl7903(t)

	detail, err := endpointSvc.ValidateEndpoint("/system/users/add", "POST")
	require.NoError(t, err)
	assert.Equal(t, "/system/users/add", detail.Route)
	assert.Equal(t, "POST", detail.Method)
	assert.Equal(t, "创建用户", detail.DisplayName)
	assert.Equal(t, "用户管理", detail.Module, "模块名从元数据回填")
	assert.Equal(t, "系统管理", detail.Category)
	assert.Equal(t, "user", detail.Icon)
	assert.Equal(t, []string{"system:user:add"}, detail.RequiredPerms)

	// 未命中 → 明确报错
	_, err = endpointSvc.ValidateEndpoint("/no/such/route", "GET")
	require.ErrorContains(t, err, "endpoint not found")
	// 同路由不同 method → 未命中
	_, err = endpointSvc.ValidateEndpoint("/system/users/add", "DELETE")
	require.ErrorContains(t, err, "endpoint not found")
}

// TestSwe7903_ExtractRoutes 路由抽取(排除 swagger/metrics 等前缀)+ RouteExists + 表驱动。
func TestSwe7903_ExtractRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/users", func(c *gin.Context) {})
	engine.POST("/api/users", func(c *gin.Context) {})
	engine.GET("/swagger/index.html", func(c *gin.Context) {})
	engine.GET("/metrics", func(c *gin.Context) {})

	extractor := NewSwaggerExtractor(engine)

	routes := extractor.ExtractRoutes()
	require.Len(t, routes, 2, "swagger/metrics 前缀被排除")
	paths := map[string]string{}
	for _, r := range routes {
		paths[r.Path+"/"+r.Method] = r.Path
	}
	assert.Contains(t, paths, "/api/users/GET")
	assert.Contains(t, paths, "/api/users/POST")

	// RouteExists 命中/未命中
	assert.True(t, extractor.RouteExists("/api/users", "GET"))
	assert.True(t, extractor.RouteExists("/api/users", "POST"))
	assert.False(t, extractor.RouteExists("/api/users", "DELETE"), "method 不匹配")
	assert.False(t, extractor.RouteExists("/swagger/index.html", "GET"), "被排除路由不可见")
	assert.False(t, extractor.RouteExists("/ghost", "GET"))

	// shouldExcludeRoute 表驱动(同包直调)
	cases := []struct {
		path string
		want bool
	}{
		{"/swagger/index.html", true},
		{"/swagger", true},
		{"/metrics", true},
		{"/favicon.ico", true},
		{"/assets/app.js", true},
		{"/api/users", false},
		{"/health", false},
		{"", false},
		// QUIRK-79-03-M(锁定不修):前缀匹配无边界,"/swaggeriff" 也因
		// strings.HasPrefix("/swagger") 命中被排除。
		{"/swaggeriff", true},
		{"/metricsx", true},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, extractor.shouldExcludeRoute(tc.path), "path=%q", tc.path)
	}
}

// TestNcu7903_CronUtils 四函数:cron 形态/各 recurrence 分支/任务名/常用表达式。
func TestNcu7903_CronUtils(t *testing.T) {
	// CalculateCronExpression:固定时间 → "秒 分 时 日 月 ?" 六段
	at := time.Date(2024, 1, 15, 14, 30, 5, 0, time.UTC)
	assert.Equal(t, "0 30 14 15 1 ?", CalculateCronExpression(at))
	// 12 月补位(int(month) 直出)
	at2 := time.Date(2025, time.December, 3, 7, 5, 0, 0, time.UTC)
	assert.Equal(t, "0 5 7 3 12 ?", CalculateCronExpression(at2))

	t.Run("GenerateCronExpression", func(t *testing.T) {
		// daily:分 时 落位
		got, err := GenerateCronExpression("daily", "09:30", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "0 30 9 * * ?", got)

		// 空执行时间 → 默认上午 9 点
		got, err = GenerateCronExpression("daily", "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "0 0 9 * * ?", got)

		// weekly:weekDay 1-7 校验
		wd := 2
		got, err = GenerateCronExpression("weekly", "08:15", &wd, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "0 15 8 ? * 2", got)
		_, err = GenerateCronExpression("weekly", "08:15", nil, nil, nil)
		require.ErrorContains(t, err, "周几参数错误")
		bad := 0
		_, err = GenerateCronExpression("weekly", "08:15", &bad, nil, nil)
		require.ErrorContains(t, err, "周几参数错误")
		bad = 8
		_, err = GenerateCronExpression("weekly", "08:15", &bad, nil, nil)
		require.ErrorContains(t, err, "周几参数错误")

		// monthly:monthDay 1-31 校验
		md := 15
		got, err = GenerateCronExpression("monthly", "09:00", nil, &md, nil)
		require.NoError(t, err)
		assert.Equal(t, "0 0 9 15 * ?", got)
		_, err = GenerateCronExpression("monthly", "09:00", nil, nil, nil)
		require.ErrorContains(t, err, "月份日期参数错误")
		bad = 32
		_, err = GenerateCronExpression("monthly", "09:00", nil, &bad, nil)
		require.ErrorContains(t, err, "月份日期参数错误")

		// custom:提供表达式原样返回;未提供 → 报错
		expr := "0 0/5 * * * ?"
		got, err = GenerateCronExpression("custom", "09:00", nil, nil, &expr)
		require.NoError(t, err)
		assert.Equal(t, expr, got, "自定义表达式原样透传")
		_, err = GenerateCronExpression("custom", "09:00", nil, nil, nil)
		require.ErrorContains(t, err, "自定义周期需要提供 Cron 表达式")
		empty := ""
		_, err = GenerateCronExpression("custom", "09:00", nil, nil, &empty)
		require.ErrorContains(t, err, "自定义周期需要提供 Cron 表达式")

		// 非法 executeTime:越界小时 / 非数字
		_, err = GenerateCronExpression("daily", "25:00", nil, nil, nil)
		require.ErrorContains(t, err, "执行时间格式错误")
		_, err = GenerateCronExpression("daily", "09:99", nil, nil, nil)
		require.ErrorContains(t, err, "执行时间格式错误")
		_, err = GenerateCronExpression("daily", "abc", nil, nil, nil)
		require.ErrorContains(t, err, "执行时间格式错误")

		// 未知周期类型
		_, err = GenerateCronExpression("hourly", "09:00", nil, nil, nil)
		require.ErrorContains(t, err, "不支持的周期类型")
	})

	assert.Equal(t, "notice_publish_notice-abc-123", GetNoticeJobName("notice-abc-123"),
		"任务名 = notice_publish_<noticeID>")

	// GetCommonCronExpressions:非空、元素形态合法(六段)、元数据非空
	common := GetCommonCronExpressions()
	require.NotEmpty(t, common)
	seen := map[string]bool{}
	for _, ce := range common {
		assert.NotEmpty(t, ce.Name)
		assert.NotEmpty(t, ce.Description)
		segments := strings.Split(ce.Expression, " ")
		assert.Len(t, segments, 6, "表达式 %q 应为六段(秒 分 时 日 月 周)", ce.Expression)
		assert.False(t, seen[ce.Expression], "表达式 %q 不应重复", ce.Expression)
		seen[ce.Expression] = true
	}
}

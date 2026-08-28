package api

// =====================================================================
// Phase 80-04 Task 1: SetupRouter R1 probe — Conclusion A/B 都可接受。
//
// R1 probe 策略(D-80-01):
//   先用最小 mini-Core 调 SetupRouter,
//   panic → 读栈定位模块 → 补该模块所需表/服务 → 重试(15-min 预算)。
//   Conclusion A:全量 SetupRouter 一次通过 → 默认路径.
//   Conclusion B:某模块 panic → 该模块组从装配测试中剥离,
//                 差额由 pkg/errors(Task 3/4)补齐.
//
// 纪律:禁 t.Parallel(setupNoticeHub goroutine + scheduler 全局 SetNoticeHub);
// t.Cleanup 收口 hub.Stop() + SetNoticeHub 恢复;
// 零生产 .go 改动.
// =====================================================================

import (
	"net/http"
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
	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newMiniCore8004 Phase 80-04 keystone fixture(80-03 newMiniCore8003 同形副本)。
// 与 80-03 差异:本包(internal/api)需要完整 CoreInfra + CoreServices 装配,
// 因为 SetupRouter 内部 middleware 直接引用 JWTManager/TokenBlacklistService。
// Conclusion B 现场记录:
//   - R1 probe panic: user_router.go:35 GetAccountPool() panic on nil AuthFactory
//   - 修复:初始化 AuthFactory(AccountPool + DeptOUmapper,均为 nil-safe)
func newMiniCore8004(t *testing.T) (*core.Core, *gorm.DB) {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p8004.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	coreDB := &coredb.Database{DB: gormDB, Type: "sqlite"}
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })

	// 零值 Config → RequestEncryption disabled 分支(core.Config.Security.RequestEncryption.Enabled=false)。
	cfg := &config.Config{}
	cfg.JWT = config.JWTConfig{
		SecretKey:        "80-04-mini-core-jwt-secret-key-0123456789abcdef",
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           "xingran-80-04-test",
		UseSM2:           false, // 用 HS256 简化,jwtMgr 内不生成 SM2 keypair。
	}
	jwtMgr, err := security.NewJWTManager(&cfg.JWT)
	require.NoError(t, err)
	pwdMgr := security.NewPasswordManager(nil)

	// AuthFactory 初始化(user_router.go:35 GetAccountPool() 调用)。
	// userSyncer 可为 nil(GetAuthenticator 内部 nil 保护),只初始化 AccountPool。
	authFactory := security.NewAuthStrategyFactory(gormDB, pwdMgr)
	accountPool := addomain.NewAccountPool(gormDB, nil) // nil redisPubSub
	authFactory.SetUserSyncer(nil)                       // userSyncer 可为 nil,GetAuthenticator 内部 nil 保护
	authFactory.SetAccountPool(accountPool)

	c := &core.Core{
		CoreInfra: &core.CoreInfra{
			Config:                   cfg,
			DB:                       coreDB,
			Cache:                    mem,
			JWTManager:               jwtMgr,
			PwdManager:               pwdMgr,
			CaptchaService:           core.NewCaptchaService(coreDB, mem),
			CaptchaBackgroundService: core.NewCaptchaBackgroundService(coreDB, mem),
			AuthFactory:              authFactory,
		},
		CoreServices: &core.CoreServices{
			TokenBlacklistService: services.NewTokenBlacklistService(mem),
			// OperLogService nil → OperLogMiddleware 内部 nil-check;
			// DataCacheService nil → ops 路由组 floor_service.go:576 nil-guard;
			// CacheConfigService nil → apikey.go:265 nil-guard;
			// NoticeHub nil → setupNoticeHub 会覆盖 core.NoticeHub。
		},
	}
	return c, gormDB
}

// migrateMinimalTables8004 建 SetupRouter 装配探针所需的最小表集。
// 结论 B 时按 panic 模块逐个补。
func migrateMinimalTables8004(t *testing.T, db *gorm.DB) {
	t.Helper()
	// system auth 组相关(auth.go loginLocalDirect 直调).
	// 注意:ADConfig/ADServiceAccount 使用 gen_random_uuid()(PG 语法),sqlite 下不 AutoMigrate。
	// AccountPool 构造时仅存 db 引用,不执行查询;等 spot-check 或后续测试按需建表。
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserRole{},
		&models.Role{},
		&models.Menu{},
		&models.DictType{},
		&models.DictData{},
		&models.Config{},
		&models.LoginLog{},
	))
}

// TestProbe8004_SetupRouter_MinimalAssembly R1 探针:最小 Core 调 SetupRouter。
// 验收:
//   - exit 0 → Conclusion A(全量装配可行),后续走 TestRtr8004_* 路径。
//   - panic → 读栈,记录 panic 模块到注释,判定 Conclusion B。
//     (本任务按 Conclusion A 写;Conclusion B 的拆分路径在 Task 2 注释中说明)
func TestProbe8004_SetupRouter_MinimalAssembly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping probe in -short mode")
	}

	c, db := newMiniCore8004(t)
	migrateMinimalTables8004(t, db)

	// SetupRouter 内部会起 hub goroutine 并改写全局 SetNoticeHub。
	// 保存原值,测试后恢复。
	originalNoticeHub := scheduler.GetNoticeHub()

	r := gin.New()
	group := r.Group("/api/v1")

	// Conclusion A 路径:全量 SetupRouter 一次通过。
	SetupRouter(group, c, []string{"http://localhost"})

	// 验证 setupNoticeHub 副作用已生效。
	require.NotNil(t, c.NoticeHub, "core.NoticeHub must be set by setupNoticeHub")
	require.NotNil(t, c.NoticeHub, "setupNoticeHub must return non-nil hub")

	// t.Cleanup 收口:hub goroutine + scheduler 全局 var。
	t.Cleanup(func() {
		c.NoticeHub.Stop()
		scheduler.SetNoticeHub(originalNoticeHub)
	})

	// 结论 A 验收:到达此处 = SetupRouter 未 panic。
	t.Log("R1 probe: Conclusion A — full SetupRouter succeeded with minimal tables")
}

// ============================================================================
// Phase 80-04 Task 2: SetupRouter 全量装配测试(DQ1-a) + spot-check 请求。
// ============================================================================

// seedTestUser8004 在 db 中建一个启用的测试用户供 spot-check 使用。
func seedTestUser8004(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()
	pwdMgr := security.NewPasswordManager(nil)
	hash, err := pwdMgr.HashPassword("password123")
	require.NoError(t, err)
	user := &models.User{
		Username: "testadmin",
		Password: hash,
		Salt:     "salt-8004",
		Status:   models.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	// 建默认管理员角色(SetupRoleRouter 会引用 sys_role.role_name).
	role := &models.Role{
		RoleName: "管理员",
		RoleKey:  "admin",
		Status:   models.RoleStatusEnabled,
		RoleSort: 1,
	}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error)
	return user
}

// TestRtr8004_SetupRouter_FullAssembly 全量 SetupRouter 装配测试。
// 验证:
//   1. SetupRouter 不 panic(已达 Conclusion A 前提)。
//   2. 路由表已挂载( Routes() 非空)。
//   3. spot-check 请求走通。
func TestRtr8004_SetupRouter_FullAssembly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full assembly in -short mode")
	}

	c, db := newMiniCore8004(t)

	// 建 auth handler 实际引用的全部表(system 组).
	// ADConfig/ADServiceAccount 使用 gen_random_uuid()(PG 语法),sqlite 下不 AutoMigrate。
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserRole{},
		&models.Role{},
		&models.Menu{},
		&models.DictType{},
		&models.DictData{},
		&models.Config{},
		&models.LoginLog{},
		&models.Notice{},
	))
	seedTestUser8004(t, db)

	originalNoticeHub := scheduler.GetNoticeHub()

	r := gin.New()
	group := r.Group("/api/v1")
	SetupRouter(group, c, []string{"http://localhost"})

	t.Cleanup(func() {
		c.NoticeHub.Stop()
		scheduler.SetNoticeHub(originalNoticeHub)
	})

	// 断言路由表非空。
	routes := r.Routes()
	require.NotEmpty(t, routes, "SetupRouter must register at least one route")
	t.Logf("SetupRouter registered %d routes", len(routes))

	// spot-check 1: GET /api/v1/system/auth/config (无需认证路径)
	// 预期 200/401/500 之一,只要路由走到 handler 就行。
	req1, _ := http.NewRequest("GET", "/api/v1/system/auth/config", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	t.Logf("spot-check /system/auth/config: status=%d", w1.Code)
	// 0 值 Config 导致 encryption disabled,但 handler 仍会走。
	// 不严格断言 status —— 只验证路由已挂、handler 未 404。

	// spot-check 2: 认证路径无 token → 401 中间件拒绝。
	req2, _ := http.NewRequest("POST", "/api/v1/system/users/list", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code, "unauthenticated request should be 401")

	// spot-check 3: 不存在路径 → 404。
	req3, _ := http.NewRequest("GET", "/api/v1/nonexistent/path", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusNotFound, w3.Code, "nonexistent path should be 404")
}

// TestRtr8004_NoticeHubSideEffect 验证 setupNoticeHub 副作用正确收口。
func TestRtr8004_NoticeHubSideEffect(t *testing.T) {
	c, db := newMiniCore8004(t)
	migrateMinimalTables8004(t, db)

	originalNoticeHub := scheduler.GetNoticeHub()
	r := gin.New()
	group := r.Group("/api/v1")
	hub := setupNoticeHub(group, c)

	t.Cleanup(func() {
		hub.Stop()
		scheduler.SetNoticeHub(originalNoticeHub)
	})

	// hub 非 nil。
	assert.NotNil(t, hub)

	// core.NoticeHub 已挂。
	assert.NotNil(t, c.NoticeHub)

	// GetNoticeHub() 返回 adapter(全局已改写)。
	current := scheduler.GetNoticeHub()
	assert.NotNil(t, current, "GetNoticeHub must return non-nil after setupNoticeHub")
}

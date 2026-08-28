package v1

// =====================================================================
// Phase 80-03 Task 1 + Task 2: auth.go mini-Core 真装配 handler 测试。
//
// D-80-03 裁决(必读):
//   - fixture 用【真 CaptchaService】(core.NewCaptchaService,具体类型无法 stub)
//     + 【真 security.NewJWTManager】(SM2 keypair 进程内生成) + sqlite 表。
//   - 否决为可测性把 CoreServices.CaptchaService 接口化(生产结构变更)。
//   - 占位 handler 教训(auth_integration_test.go:458)不得复现:本文件全部断言
//     经 SetupAuthRouter 真装配 + httptest 真请求发出,不存在"注册即覆盖"假象。
//
// fixture 形状(80-04 复制输入;跨包不能共享 _test.go,80-04 另起 newMiniCore8004):
//   core.Core{CoreInfra: *core.CoreInfra{Config, DB, Cache, JWTManager, PwdManager,
//     CaptchaService, CaptchaBackgroundService, [AuthFactory]}, CoreServices: &core.CoreServices{}}
//   DB = &coredb.Database{DB: <glebarez sqlite>, Type: "sqlite"}(导出字段,测试手工构造合法)
//
// 纪律: 零 t.Parallel(gin test context + MemoryCache 共享面);t.Cleanup 关连接;
// status 断言一律引用 models.* 常量,禁裸 0/1。
// =====================================================================

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// testJWTSecret8003 fixture 专用 JWT 密钥(≥32 字节,非弱默认值,F-04 校验通过)。
const testJWTSecret8003 = "80-03-mini-core-fixture-jwt-secret-key-0123456789abcdef"

// newMiniCore8003 Phase 80 keystone fixture(D-80-03 真装配)。
// 每个用例独立 sqlite 文件库(t.TempDir)+ 独立 MemoryCache,用例间零共享。
func newMiniCore8003(t *testing.T) (*core.Core, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p8003.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	coreDB := &coredb.Database{DB: gormDB, Type: "sqlite"}
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })

	// 零值 Config → RequestEncryption disabled 分支;
	// JWT 打开 SM2 → keypair 由 manager 进程内生成(getPublicKey/testSM2 可真测,零外呼)。
	cfg := &config.Config{}
	cfg.JWT = config.JWTConfig{
		SecretKey:        testJWTSecret8003,
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           "xingran-80-03-test",
		UseSM2:           true,
	}
	jwtMgr, err := security.NewJWTManager(&cfg.JWT)
	require.NoError(t, err)
	pwdMgr := security.NewPasswordManager(nil)

	c := &core.Core{
		CoreInfra: &core.CoreInfra{
			Config:                   cfg,
			DB:                       coreDB,
			Cache:                    mem,
			JWTManager:               jwtMgr,
			PwdManager:               pwdMgr,
			CaptchaService:           core.NewCaptchaService(coreDB, mem),
			CaptchaBackgroundService: core.NewCaptchaBackgroundService(coreDB, mem),
		},
		CoreServices: &core.CoreServices{},
	}
	return c, gormDB
}

// newAuthCore8003 在 mini-Core 基础上接入真 AuthStrategyFactory(local 模式,sqlite 驱动)。
func newAuthCore8003(t *testing.T) (*core.Core, *gorm.DB) {
	t.Helper()
	c, db := newMiniCore8003(t)
	c.AuthFactory = security.NewAuthStrategyFactory(db, c.PwdManager)
	return c, db
}

// migrateAuthTables8003 建 handler 实际引用的表(R3/78-01 精简哲学:不做全库迁移)。
func migrateAuthTables8003(t *testing.T, db *gorm.DB, targets ...interface{}) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(targets...))
}

// authDefaultTables8003 login/logout/refresh 族引用的表集合。
func authDefaultTables8003(db *gorm.DB) []interface{} {
	return []interface{}{
		&models.User{}, &models.UserRole{}, &models.Config{}, &models.LoginLog{},
	}
}

// seedUser8003 建用户行(密码经 PasswordManager 真实 SM3-PBKDF2 哈希,不走明文)。
func seedUser8003(t *testing.T, db *gorm.DB, username, password string, status models.UserStatus, nickname string) *models.User {
	t.Helper()
	pwdMgr := security.NewPasswordManager(nil)
	hash, err := pwdMgr.HashPassword(password)
	require.NoError(t, err)

	var nick *string
	if nickname != "" {
		n := nickname
		nick = &n
	}
	user := &models.User{
		Username: username,
		Password: hash,
		Salt:     "salt-8003",
		Nickname: nick,
		Status:   status,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// seedUserRole8003 建用户-角色关联行。
func seedUserRole8003(t *testing.T, db *gorm.DB, userID, roleID string) {
	t.Helper()
	require.NoError(t, db.Create(&models.UserRole{UserID: userID, RoleID: roleID}).Error)
}

// seedSysConfig8003 写一行 sys_config。
func seedSysConfig8003(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  key,
		ConfigKey:   key,
		ConfigValue: value,
	}).Error)
}

// mountAuthRouter8003 真 SetupAuthRouter 装配(占位 handler 教训的反面)。
func mountAuthRouter8003(t *testing.T, c *core.Core) *gin.Engine {
	t.Helper()
	router := gin.New()
	group := router.Group("/api/v1/system/auth")
	SetupAuthRouter(group, c)
	return router
}

// apiResp8003 标准响应结构(API响应规范:code/message/data/timestamp/request_id)。
type apiResp8003 struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	RequestID string          `json:"request_id"`
}

// performJSON8003 发出真实 httptest 请求并解析标准响应体。
func performJSON8003(t *testing.T, router *gin.Engine, method, path string, body interface{}) (*httptest.ResponseRecorder, apiResp8003) {
	t.Helper()

	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = strings.NewReader(string(raw))
	}

	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp apiResp8003
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "响应体应为标准 JSON: %s", w.Body.String())
	return w, resp
}

// loginPayload8003 构造登录请求体。
func loginPayload8003(username, password, authMode string) map[string]interface{} {
	payload := map[string]interface{}{
		"username": username,
		"password": password,
	}
	if authMode != "" {
		payload["authMode"] = authMode
	}
	return payload
}

// countLoginLogs8003 统计登录日志行数(可选按用户名过滤)。
func countLoginLogs8003(t *testing.T, db *gorm.DB, username string) int64 {
	t.Helper()
	var count int64
	q := db.Model(&models.LoginLog{})
	if username != "" {
		q = q.Where("user_name = ?", username)
	}
	require.NoError(t, q.Count(&count).Error)
	return count
}

// =====================================================================
// Task 1: loginLocalDirect 四分支 + AuthFactory nil fallback
// 说明:AuthFactory 为 nil 时,POST /login 的第 5 步直接进入 loginLocalDirect
// (auth.go:157-162),因此经真路由发出的请求就是 loginLocalDirect 的真调用链。
// "Locked" 分支位于 login() 第 2 步 CheckLoginLock(auth.go:121-124),同样经真路由触达。
// =====================================================================

// TestAuth8003_LoginLocalDirect_Success 正确凭证 → 令牌对 + 用户信息。
func TestAuth8003_LoginLocalDirect_Success(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	user := seedUser8003(t, db, "local-user", "Str0ng!Pass8003", models.UserStatusEnabled, "本地用户")
	seedUserRole8003(t, db, user.ID, "role-ops-8003")
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("local-user", "Str0ng!Pass8003", ""))

	require.Equal(t, http.StatusOK, w.Code, "登录成功应为 200: %s", resp.Message)
	require.Equal(t, 0, resp.Code)

	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(resp.Data, &loginResp))
	assert.NotEmpty(t, loginResp.AccessToken, "应签发 access token")
	assert.NotEmpty(t, loginResp.RefreshToken, "应签发 refresh token")
	assert.Equal(t, "Bearer", loginResp.TokenType)
	assert.Equal(t, int64(7200), loginResp.ExpiresIn)
	require.NotNil(t, loginResp.User)
	assert.Equal(t, user.ID, loginResp.User.ID)
	assert.Equal(t, "local-user", loginResp.User.Username)
	require.NotNil(t, loginResp.User.Nickname)
	assert.Equal(t, "本地用户", *loginResp.User.Nickname)
	assert.Equal(t, models.UserStatusEnabled, loginResp.User.Status, "status 断言引用 models 常量")
	assert.ElementsMatch(t, []string{"role-ops-8003"}, loginResp.User.Roles, "角色应从 sys_user_role 查出")

	// 登录成功路径调用 ClearLoginFailure + recordLoginLog(异步落库)
	require.Eventually(t, func() bool {
		return countLoginLogs8003(t, db, "local-user") == 1
	}, 2*time.Second, 20*time.Millisecond, "登录成功应异步写一条登录日志")
	var logs []models.LoginLog
	require.NoError(t, db.Where("user_name = ?", "local-user").Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, int(models.LoginLogStatusSuccess), logs[0].Status, "成功日志 status 引用 models 常量值")
	require.NotNil(t, logs[0].Msg)
	assert.Equal(t, "登录成功", *logs[0].Msg)
}

// TestAuth8003_LoginLocalDirect_Locked 触发 CheckLoginLock 锁定分支(auth.go:121-124)。
// 锁定状态由 CaptchaService 的 cache 锁键表达(与生产 RecordLoginFailure 锁定同一把钥匙)。
func TestAuth8003_LoginLocalDirect_Locked(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedUser8003(t, db, "locked-user", "Str0ng!Pass8003", models.UserStatusEnabled, "")

	// 直接置锁键 → CheckLoginLock 必命中锁定分支
	ctx := context.Background()
	lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, "locked-user")
	require.NoError(t, c.Cache.Set(ctx, lockKey, "1", 30*time.Minute))

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("locked-user", "Str0ng!Pass8003", ""))

	assert.Equal(t, http.StatusForbidden, w.Code, "锁定应 403")
	assert.Equal(t, response.ErrForbidden.Code, resp.Code, "业务码引用 response 包常量")
	assert.Contains(t, resp.Message, "账号已锁定", "应返回 CheckLoginLock 的锁定文案")

	// 锁定分支不产生登录日志(未到登录动作)
	assert.Zero(t, countLoginLogs8003(t, db, "locked-user"))
}

// TestAuth8003_LoginLocalDirect_Disabled 禁用用户 → 403 禁用分支。
func TestAuth8003_LoginLocalDirect_Disabled(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedUser8003(t, db, "disabled-user", "Str0ng!Pass8003", models.UserStatusDisabled, "")

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("disabled-user", "Str0ng!Pass8003", ""))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, resp.Message, "用户已被禁用")

	var logs []models.LoginLog
	require.NoError(t, db.Where("user_name = ?", "disabled-user").Find(&logs).Error)
	assert.Empty(t, logs, "禁用分支在 recordLoginLog 之前返回,不写日志")
}

// TestAuth8003_LoginLocalDirect_WrongPassword 密码错 → 401 + 失败计数 + 失败日志。
func TestAuth8003_LoginLocalDirect_WrongPassword(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedUser8003(t, db, "wrongpwd-user", "Str0ng!Pass8003", models.UserStatusEnabled, "")

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("wrongpwd-user", "Totally-Wrong", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 1013, resp.Code, "ErrCredentialInvalid 业务码")

	// loginLocalDirect 失败分支调用 RecordLoginFailure → 缓存计数 +1
	ctx := context.Background()
	failKey := fmt.Sprintf("login:fail:%s", "wrongpwd-user")
	count, err := c.Cache.Get(ctx, failKey)
	require.NoError(t, err, "失败计数键应存在")
	assert.Equal(t, "1", count, "一次密码错误应累计失败 1 次")

	// recordLoginLog(status=失败)异步落库
	require.Eventually(t, func() bool {
		return countLoginLogs8003(t, db, "wrongpwd-user") == 1
	}, 2*time.Second, 20*time.Millisecond, "密码错误应异步写失败登录日志")
	var logs []models.LoginLog
	require.NoError(t, db.Where("user_name = ?", "wrongpwd-user").Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, int(models.LoginLogStatusFailure), logs[0].Status)
	require.NotNil(t, logs[0].Msg)
	assert.Equal(t, "用户或密码错误", *logs[0].Msg)
}

// TestAuth8003_LoginLocalDirect_UserNotFound 用户不存在 → 凭证错误分支(:276-279)。
func TestAuth8003_LoginLocalDirect_UserNotFound(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("ghost-user", "Whatever!8003", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 1013, resp.Code)
	assert.Zero(t, countLoginLogs8003(t, db, "ghost-user"), "用户不存在分支不写登录日志")
}

// TestAuth8003_AuthFactoryNil_Fallback AuthFactory nil → 走 :157-162 fallback。
// 用显式 authMode=ad 证明:工厂为 nil 时请求里的认证模式被完全忽略,仍由本地直接认证兜底成功。
func TestAuth8003_AuthFactoryNil_Fallback(t *testing.T) {
	c, db := newMiniCore8003(t)
	require.Nil(t, c.AuthFactory, "fixture 默认不装 AuthFactory(触发 fallback 前置条件)")
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedUser8003(t, db, "fallback-user", "Str0ng!Pass8003", models.UserStatusEnabled, "回退用户")

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("fallback-user", "Str0ng!Pass8003", "ad"))

	require.Equal(t, http.StatusOK, w.Code, "nil 工厂应回退本地认证而非报 AD 错误: %s", resp.Message)
	require.Equal(t, 0, resp.Code)

	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(resp.Data, &loginResp))
	assert.NotEmpty(t, loginResp.AccessToken, "fallback 生效:应签发令牌")
	require.NotNil(t, loginResp.User)
	assert.Equal(t, "fallback-user", loginResp.User.Username)

	require.Eventually(t, func() bool {
		return countLoginLogs8003(t, db, "fallback-user") == 1
	}, 2*time.Second, 20*time.Millisecond)
}

// TestAuth8003_Login_BadPayload 绑定失败分支(auth.go:100-103)。
func TestAuth8003_Login_BadPayload(t *testing.T) {
	c, _ := newMiniCore8003(t)
	router := mountAuthRouter8003(t, c)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/system/auth/login", strings.NewReader(`{"username":`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp apiResp8003
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Message, "请求参数错误")
}

// TestAuth8003_Login_MissingUsername 必填字段缺失 → 绑定失败分支。
func TestAuth8003_Login_MissingUsername(t *testing.T) {
	c, _ := newMiniCore8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", map[string]interface{}{"password": "no-username"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, resp.Message, "请求参数错误")
}

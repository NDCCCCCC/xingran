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
	"github.com/xingran-next/xingran-go-backend/internal/services"
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
		// TokenBlacklistService 真装配:logout 黑名单路径是真接口调用,
		// 零值 interface 调 AddToBlacklist 会 panic —— 不能省。
		CoreServices: &core.CoreServices{
			TokenBlacklistService: services.NewTokenBlacklistService(mem),
		},
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

// =====================================================================
// Task 2: auth.go 其余 handler 群(全部真装配真请求)
// =====================================================================

// TestAuth8003_GetAuthConfig 表驱动:getAuthConfig 真行为锁定(auth.go:467-490)。
//
// QUIRK-80-03-A(就地锁定,零生产改动):handler 复用同一个 `var config models.Config`
// 承接两次 First() —— GORM 会把 dest 结构体中已非零的字段追加为查询条件,
// 因此第二查询(config_key = sys.auth.default.mode)实际携带了第一行(sys.auth.ad.enabled)
// 的 id/config_value/时间戳等残余条件,必然 Miss → defaultMode 恒回退 "local"。
// 仅当 sys.auth.ad.enabled 行【不存在】时(第一查询未命中,dest 保持零值),
// default.mode 才能被正确读取。
// 正确修法是第二个查询用独立变量,属生产代码变更,超出本 plan 零业务变更纪律
// (plan <notes> + D-80 escape hatch 仅限源码-vs-文档错配),故只锁行为不扩修;
// 建议后续 /gsd-quick 单独修复。
func TestAuth8003_GetAuthConfig(t *testing.T) {
	tests := []struct {
		name            string
		seedEnabled     string
		seedMode        string
		wantADEnabled   bool
		wantDefaultMode string
	}{
		{name: "无配置_双默认local", wantADEnabled: false, wantDefaultMode: "local"},
		// QUIRK-80-03-A:ad.enabled 行先命中 → 第二查询被残余条件污染 → defaultMode 停留 local
		{name: "AD启用_因QUIRK模式读不到", seedEnabled: "true", seedMode: "ad", wantADEnabled: true, wantDefaultMode: "local"},
		{name: "仅配置模式行_正确读出ad", seedMode: "ad", wantADEnabled: false, wantDefaultMode: "ad"},
		{name: "仅配置模式行hybrid", seedMode: "hybrid", wantADEnabled: false, wantDefaultMode: "hybrid"},
		{name: "无效模式值_回退local", seedMode: "bogus-mode", wantADEnabled: false, wantDefaultMode: "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, db := newMiniCore8003(t)
			migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
			if tt.seedEnabled != "" {
				seedSysConfig8003(t, db, "sys.auth.ad.enabled", tt.seedEnabled)
			}
			if tt.seedMode != "" {
				seedSysConfig8003(t, db, "sys.auth.default.mode", tt.seedMode)
			}

			router := mountAuthRouter8003(t, c)
			w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/config", nil)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, 0, resp.Code)
			var data map[string]interface{}
			require.NoError(t, json.Unmarshal(resp.Data, &data))
			assert.Equal(t, tt.wantADEnabled, data["adEnabled"])
			assert.Equal(t, tt.wantDefaultMode, data["defaultMode"])
		})
	}
}

// TestAuth8003_GetPublicKey 真 JWTManager(SM2 keypair 进程内生成)→ 非空公钥。
func TestAuth8003_GetPublicKey(t *testing.T) {
	c, _ := newMiniCore8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/public-key", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, resp.Code, "SM2 已启用时公钥必须可得")
	var data map[string]string
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.NotEmpty(t, data["publicKey"], "公钥字段非空")
}

// TestAuth8003_TestSM2 SM2 加解密自检往返(数据由 handler 内部构造,密钥不出进程)。
func TestAuth8003_TestSM2(t *testing.T) {
	c, _ := newMiniCore8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/test-sm2", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, resp.Code)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, true, data["success"], "SM2 自检应通过")
	assert.NotEmpty(t, data["encrypted"])
	assert.NotEmpty(t, data["decrypted"])
	assert.NotEqual(t, data["testData"], data["encrypted"], "密文不得等于明文(T-80-03-03:只比字段)")
	assert.Contains(t, data["message"], "测试通过")
}

// TestAuth8003_Logout 登出四态:无 token / 空 token / 无 claims / 真令牌拉黑。
// 真令牌路径通过挂载路由先 Set token+claims(生产由 JWT 中间件注入)再调真 logout handler。
func TestAuth8003_Logout(t *testing.T) {
	t.Run("无token与空token与无claims_均幂等成功", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		router := mountAuthRouter8003(t, c)

		for _, path := range []string{"/api/v1/system/auth/logout"} {
			w, resp := performJSON8003(t, router, http.MethodPost, path, nil)
			assert.Equal(t, http.StatusOK, w.Code)
			var data map[string]string
			require.NoError(t, json.Unmarshal(resp.Data, &data))
			assert.Equal(t, "登出成功", data["message"])
		}
	})

	t.Run("真令牌真claims_拉黑生效", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		// 签发真令牌 + 真 claims(与 JWT 中间件注入同构)
		pair, err := c.JWTManager.GenerateTokenPair("user-8003", "logout-user", "昵称", []string{"role-a"})
		require.NoError(t, err)
		claims, err := c.JWTManager.ValidateToken(pair.AccessToken)
		require.NoError(t, err)

		router := gin.New()
		router.POST("/auth/logout", func(gc *gin.Context) {
			gc.Set("token", pair.AccessToken)
			gc.Set("claims", claims)
			logout(c)(gc)
		})

		w, resp := performJSON8003(t, router, http.MethodPost, "/auth/logout", nil)
		require.Equal(t, http.StatusOK, w.Code)
		var data map[string]string
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Equal(t, "登出成功", data["message"])

		// 断言真黑名单行为:token 已入黑名单(IsBlacklisted 查真缓存)
		blacklisted, err := c.TokenBlacklistService.IsBlacklisted(context.Background(), pair.AccessToken)
		require.NoError(t, err)
		assert.True(t, blacklisted, "logout 必须把令牌写入真黑名单")
	})
}

// TestAuth8003_RefreshToken 刷新令牌:合法 refresh / 非法串 / access 误用 三态。
func TestAuth8003_RefreshToken(t *testing.T) {
	t.Run("合法refreshToken_换发新令牌对", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
		user := seedUser8003(t, db, "refresh-user", "Str0ng!Pass8003", models.UserStatusEnabled, "刷新用户")
		seedUserRole8003(t, db, user.ID, "role-refresh-8003")

		pair, err := c.JWTManager.GenerateTokenPair(user.ID, user.Username, "刷新用户", []string{"role-refresh-8003"})
		require.NoError(t, err)

		router := mountAuthRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/refresh", map[string]interface{}{"refreshToken": pair.RefreshToken})

		require.Equal(t, http.StatusOK, w.Code, "合法 refresh 应换发成功: %s", resp.Message)
		require.Equal(t, 0, resp.Code)
		var loginResp LoginResponse
		require.NoError(t, json.Unmarshal(resp.Data, &loginResp))
		assert.NotEmpty(t, loginResp.AccessToken)
		assert.NotEmpty(t, loginResp.RefreshToken)
		require.NotNil(t, loginResp.User)
		assert.Equal(t, user.ID, loginResp.User.ID)
	})

	t.Run("非法token串_拒绝", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		router := mountAuthRouter8003(t, c)
		w, _ := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/refresh", map[string]interface{}{"refreshToken": "not-a-jwt"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("accessToken误用为refresh_角色不匹配拒绝", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		pair, err := c.JWTManager.GenerateTokenPair("uid", "uname", "", []string{"role-x"})
		require.NoError(t, err)

		router := mountAuthRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/refresh", map[string]interface{}{"refreshToken": pair.AccessToken})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, response.ErrTokenInvalid.Code, resp.Code, "access 角色不满足 refresh 校验")
	})

	t.Run("用户已不存在_拒绝", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
		// 不 seed 用户 → 令牌合法但 DB 无行(refresh 令牌 Roles 恒为 ["refresh"],
		// 由 GenerateRefreshTokenWithSM2 硬编码,不经 GenerateTokenPair 的 roles 参数)
		pair, err := c.JWTManager.GenerateTokenPair("ghost-id", "ghost", "", nil)
		require.NoError(t, err)

		router := mountAuthRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/refresh", map[string]interface{}{"refreshToken": pair.RefreshToken})
		assert.Equal(t, http.StatusNotFound, w.Code, "ErrUserNotFound 映射 HTTP 404")
		assert.Equal(t, response.ErrUserNotFound.Code, resp.Code)
	})

	t.Run("缺refreshToken字段_绑定失败", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		router := mountAuthRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/refresh", map[string]interface{}{})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp.Message, "请求参数错误")
	})
}

// TestAuth8003_SyncErrorReasonMessage AD 同步失败原因码映射表(auth.go:648-663)。
func TestAuth8003_SyncErrorReasonMessage(t *testing.T) {
	tests := []struct {
		reason string
		want   string
	}{
		{"admin_dial", "AD 管理员账号配置异常（账号可能被锁定或密码错误），请联系系统管理员"},
		{"admin_bind", "AD 管理员账号配置异常（账号可能被锁定或密码错误），请联系系统管理员"},
		{"user_search", "AD 用户信息搜索失败，请联系系统管理员"},
		{"user_sync", "AD 用户同步到本地失败，请联系系统管理员"},
		{"no_syncer", "AD 用户同步服务未配置，请联系系统管理员"},
		{"unknown-reason", "AD 用户信息同步失败，请联系系统管理员"},
		{"", "AD 用户信息同步失败，请联系系统管理员"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, syncErrorReasonMessage(tt.reason), "reason=%q", tt.reason)
	}
}

// TestAuth8003_GetNicknameOrUsername 昵称回退三态(纯函数直调,无副作用)。
func TestAuth8003_GetNicknameOrUsername(t *testing.T) {
	nick := "显式昵称"
	empty := ""
	assert.Equal(t, "显式昵称", getNicknameOrUsername(&models.User{Username: "u1", Nickname: &nick}))
	assert.Equal(t, "u2", getNicknameOrUsername(&models.User{Username: "u2", Nickname: &empty}),
		"空字符串昵称应回退 username")
	assert.Equal(t, "u3", getNicknameOrUsername(&models.User{Username: "u3", Nickname: nil}),
		"nil 昵称应回退 username")
}

// TestAuth8003_RecordLoginLog 经真请求触达 recordLoginLog,断言行落库(异步,Eventually)。
func TestAuth8003_RecordLoginLog(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	nick := "日志昵称"

	router := gin.New()
	router.POST("/record", func(gc *gin.Context) {
		gc.Request = gc.Request.WithContext(context.Background())
		recordLoginLog(gc, c, "log-user", &nick, int(models.LoginLogStatusSuccess), "记录成功")
		gc.Status(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodPost, "/record", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0")
	req.RemoteAddr = "10.20.30.40:5555"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Eventually(t, func() bool {
		return countLoginLogs8003(t, db, "log-user") == 1
	}, 2*time.Second, 20*time.Millisecond, "recordLoginLog 应异步写登录日志")

	var logs []models.LoginLog
	require.NoError(t, db.Where("user_name = ?", "log-user").Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, int(models.LoginLogStatusSuccess), logs[0].Status)
	assert.Equal(t, "10.20.30.40", logs[0].IPAddr, "客户端 IP 应被记录")
	require.NotNil(t, logs[0].Browser)
	assert.Equal(t, "Chrome", *logs[0].Browser)
	require.NotNil(t, logs[0].OS)
	assert.Equal(t, "Windows", *logs[0].OS)
	require.NotNil(t, logs[0].Nickname)
	assert.Equal(t, "日志昵称", *logs[0].Nickname)
}

// =====================================================================
// Task 2 扩展: AuthFactory 非空路径(login() 第 5-10 步,工厂模式真装配)
// =====================================================================

// TestAuth8003_Login_FactoryLocal_Success 工厂 local 模式完整成功链(步骤 4-10 全走)。
func TestAuth8003_Login_FactoryLocal_Success(t *testing.T) {
	c, db := newAuthCore8003(t)
	require.NotNil(t, c.AuthFactory)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedSysConfig8003(t, db, "sys.auth.default.mode", "local")
	user := seedUser8003(t, db, "factory-user", "Str0ng!Pass8003", models.UserStatusEnabled, "工厂用户")

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("factory-user", "Str0ng!Pass8003", ""))

	require.Equal(t, http.StatusOK, w.Code, "工厂 local 模式应成功: %s", resp.Message)
	require.Equal(t, 0, resp.Code)
	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(resp.Data, &loginResp))
	assert.NotEmpty(t, loginResp.AccessToken)
	require.NotNil(t, loginResp.User)
	assert.Equal(t, user.ID, loginResp.User.ID, "工厂路径响应应携带真实用户 ID")

	// 成功路径 ClearLoginFailure + recordLoginLog("登录成功")
	require.Eventually(t, func() bool {
		return countLoginLogs8003(t, db, "factory-user") == 1
	}, 2*time.Second, 20*time.Millisecond)
	var logs []models.LoginLog
	require.NoError(t, db.Where("user_name = ?", "factory-user").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].Msg)
	assert.Equal(t, "登录成功", *logs[0].Msg)
}

// TestAuth8003_Login_FactoryAuthErrors 表驱动:工厂路径四种认证失败分支。
// 注意:默认分支(非标准错误)对客户端也返回 ErrCredentialInvalid 默认文案
// "用户名或密码错误",差异只在登录日志文案("认证失败")。
func TestAuth8003_Login_FactoryAuthErrors(t *testing.T) {
	tests := []struct {
		name        string
		userTable   string // "empty"=建表不 seed / "seed"=建表+seed / "none"=不建表(触发底层错误)
		userStatus  models.UserStatus
		password    string
		wantMessage string
		wantLogMsg  string
		wantStatus  int
	}{
		{
			name:        "用户不存在_ErrUserNotFound",
			userTable:   "empty",
			password:    "Str0ng!Pass8003",
			wantMessage: "用户名或密码错误",
			wantLogMsg:  "用户名或密码错误",
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "密码错误_ErrInvalidCredentials",
			userTable:   "seed",
			userStatus:  models.UserStatusEnabled,
			password:    "Wrong-Password",
			wantMessage: "用户名或密码错误",
			wantLogMsg:  "用户名或密码错误",
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "用户禁用_ErrUserDisabled",
			userTable:   "seed",
			userStatus:  models.UserStatusDisabled,
			password:    "Str0ng!Pass8003",
			wantMessage: "用户已被禁用",
			wantLogMsg:  "用户已被禁用",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "非标准错误_默认分支认证失败",
			userTable:   "none",
			password:    "Str0ng!Pass8003",
			wantMessage: "用户名或密码错误",
			wantLogMsg:  "认证失败",
			wantStatus:  http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, db := newAuthCore8003(t)
			migrateAuthTables8003(t, db, &models.UserRole{}, &models.Config{}, &models.LoginLog{})
			switch tt.userTable {
			case "seed":
				migrateAuthTables8003(t, db, &models.User{})
				seedUser8003(t, db, "fac-err-user", "Str0ng!Pass8003", tt.userStatus, "")
			case "empty":
				migrateAuthTables8003(t, db, &models.User{})
			}

			router := mountAuthRouter8003(t, c)
			w, resp := performJSON8003(t, router, http.MethodPost,
				"/api/v1/system/auth/login", loginPayload8003("fac-err-user", tt.password, ""))

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)

			if tt.wantLogMsg != "" {
				require.Eventually(t, func() bool {
					return countLoginLogs8003(t, db, "fac-err-user") == 1
				}, 2*time.Second, 20*time.Millisecond, "认证失败分支应记录登录日志")
				var logs []models.LoginLog
				require.NoError(t, db.Where("user_name = ?", "fac-err-user").Find(&logs).Error)
				require.Len(t, logs, 1)
				require.NotNil(t, logs[0].Msg)
				assert.Equal(t, tt.wantLogMsg, *logs[0].Msg)
			}
		})
	}
}

// TestAuth8003_Login_FactoryAuthenticatorUnavailable 默认模式=ad 且无 AD 配置 →
// GetAuthenticator 失败 → "认证服务异常" 分支(auth.go:164-169)。
func TestAuth8003_Login_FactoryAuthenticatorUnavailable(t *testing.T) {
	c, db := newAuthCore8003(t)
	migrateAuthTables8003(t, db, authDefaultTables8003(db)...)
	seedSysConfig8003(t, db, "sys.auth.default.mode", "ad")
	// 不建 sys_ad_config → getADConfigID 找不到可用 AD 配置

	router := mountAuthRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost,
		"/api/v1/system/auth/login", loginPayload8003("any-user", "Str0ng!Pass8003", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "认证服务异常", resp.Message)
}

// TestAuth8003_Login_EncryptedPassword SM2 加密密码路径(:127-135 解密成功/失败)。
func TestAuth8003_Login_EncryptedPassword(t *testing.T) {
	t.Run("SM2解密失败_拒绝", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateAuthTables8003(t, db, authDefaultTables8003(db)...)

		router := mountAuthRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/login", map[string]interface{}{
				"username":          "enc-user",
				"password":          "definitely-not-valid-sm2-ciphertext",
				"encryptedPassword": true,
			})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp.Message, "密码解密失败")
	})
}

// =====================================================================
// Task 2 收尾: SM2-disabled 变体 + logout 早退分支 + 真加密配置 handler
// =====================================================================

// newNoSM2Core8003 SM2 关闭变体(HS256 签名)→ getPublicKey/testSM2 的"不可用"分支。
func newNoSM2Core8003(t *testing.T) *core.Core {
	t.Helper()
	c, _ := newMiniCore8003(t)
	cfg := &config.JWTConfig{
		SecretKey:        testJWTSecret8003,
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           "xingran-80-03-test",
		UseSM2:           false,
	}
	jwtMgr, err := security.NewJWTManager(cfg)
	require.NoError(t, err)
	c.JWTManager = jwtMgr
	return c
}

// TestAuth8003_GetPublicKey_SM2Disabled SM2 关闭 → 公钥不可用错误分支(:503-510)。
func TestAuth8003_GetPublicKey_SM2Disabled(t *testing.T) {
	c := newNoSM2Core8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/public-key", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "SM2 未启用或公钥不可用", resp.Message)
}

// TestAuth8003_TestSM2_SM2Disabled SM2 关闭 → nil keypair 错误分支(:542-546)。
func TestAuth8003_TestSM2_SM2Disabled(t *testing.T) {
	c := newNoSM2Core8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/test-sm2", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "SM2 未启用", resp.Message)
}

// TestAuth8003_Logout_ContextBranches token/claims 上下文早退分支(:336-367)。
func TestAuth8003_Logout_ContextBranches(t *testing.T) {
	tests := []struct {
		name  string
		setup func(gc *gin.Context)
		note  string
	}{
		{name: "token为空字符串", setup: func(gc *gin.Context) { gc.Set("token", "") }},
		{name: "token非字符串类型", setup: func(gc *gin.Context) { gc.Set("token", 12345) }},
		{name: "claims缺失", setup: func(gc *gin.Context) { gc.Set("token", "tok-8003") }},
		{name: "claims类型断言失败", setup: func(gc *gin.Context) {
			gc.Set("token", "tok-8003")
			gc.Set("claims", "not-custom-claims")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newMiniCore8003(t)
			router := gin.New()
			router.POST("/logout", func(gc *gin.Context) {
				tt.setup(gc)
				logout(c)(gc)
			})

			w, resp := performJSON8003(t, router, http.MethodPost, "/logout", nil)
			assert.Equal(t, http.StatusOK, w.Code, tt.note)
			var data map[string]string
			require.NoError(t, json.Unmarshal(resp.Data, &data))
			assert.Equal(t, "登出成功", data["message"])
		})
	}
}

// TestAuth8003_GetEncryptionConfig 真 handler(非 mock 替身):读中间件缓存态。
func TestAuth8003_GetEncryptionConfig(t *testing.T) {
	c, _ := newMiniCore8003(t)
	router := mountAuthRouter8003(t, c)

	w, resp := performJSON8003(t, router, http.MethodGet, "/api/v1/system/auth/encryption-config", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, resp.Code)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Contains(t, data, "enabled", "响应必须含 enabled 字段")
	assert.IsType(t, true, data["enabled"], "enabled 为布尔")
	assert.Equal(t, "cache", data["source"], "handler 固定上报 cache 来源")
}

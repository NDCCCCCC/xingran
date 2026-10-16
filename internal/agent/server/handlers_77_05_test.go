package server

// =====================================================================
// handlers_77_05_test.go — Phase 77 Plan 05 (BLOCK-02 收口)
//
// 覆盖范围: handlers.go 全 9 路由 + middleware.go JWTAuth 端到端 + pty_manager.go
// 五方法 (零 Skipf, 同包白盒直插 sessions) + logger.go InitLogger logPath 分支
// + WithContext/WithRequestID/Fatal 收尾。前缀 TestHdl77_ / TestPty77_ 区分段位。
//
// 复用 77-04 资产 (同包直引):
//   newAgentBackend77(t)              — httptest.NewServer 假后端 + srv.URL 注入
//   newJWT77(t, b)                    — NewJWTAuthenticator(secret, srv.URL, ...)
//   prefillValidLocalToken77(t, auth) — 预置合法 token 绕过真实登录
//   fakeStrategy (config_account_77_05_test.go) — platformStrategy 上层替代
//
// 纪律:
//   - 复用 agent_smoke_test.go init() 设好的 gin.TestMode (P-77-8);
//   - InitLogger 前置 (P-77-5: WithFields.Warn 依赖全局 logger);
//   - 全文件零 time.Sleep, goroutine 同步靠 channel/select (P-77-4);
//   - 测试禁 t.Parallel (全局 logger / seam 共享)。
// =====================================================================

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRouter77 装配 AccountHandler + 假后端 + fakeStrategy, 同时预置合法
// JWT token (prefillValidLocalToken77)。返回的 token 用于客户端在受保护端点
// 请求里设 Authorization 头。fakeStrategy 可为 nil → 默认空 strategy。
func newTestRouter77(t *testing.T, fs *fakeStrategy) (*gin.Engine, *agentBackend77, string) {
	t.Helper()
	require.NoError(t, InitLogger("info", ""), "P-77-5: 全局 logger 必须先初始化")
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	am := NewAccountManager()
	if fs == nil {
		fs = &fakeStrategy{}
	}
	am.strategy = fs
	h := NewAgentHandler(am, auth)
	r := gin.New()
	h.RegisterRoutes(r)
	tok := prefillValidLocalToken77(t, auth)
	return r, b, tok
}

// authedServe 给定 router + token + method/path/body 发送受保护端点请求;
// token 为空时跳过 Authorization 头 (用于公开端点 /auth-fail 断言)。
func authedServe(r *gin.Engine, tok, method, path string, body io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	r.ServeHTTP(w, req)
	return w
}

// =====================================================================
// AccountHandlers — 6 账号 handler × 成功/失败/400 bind 三态
// =====================================================================

// TestHdl77_CreateAccount 三态: invalid JSON → 400, fakeStrategy 成功 → 200,
// fakeStrategy createErr → 500 + sanitizeError 脱敏 (errors.New("password leak"))
func TestHdl77_CreateAccount(t *testing.T) {
	// 400 bind JSON
	fs := &fakeStrategy{}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`not json`))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 200 成功路径
	fs = &fakeStrategy{}
	r, _, tok = newTestRouter77(t, fs)
	w = authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`{"username":"u1","password":"p","is_admin":false}`))
	assert.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, fs.createCalls)
	require.NotNil(t, fs.lastCreate)
	assert.Equal(t, "u1", fs.lastCreate.Username)
	assert.Equal(t, "p", fs.lastCreate.Password)
	assert.False(t, fs.lastCreate.IsAdmin)

	// 500 + sanitize 脱敏 (含 "password" 与 "token" 关键词 → 通用错误消息)
	fs = &fakeStrategy{createErr: errors.New("password verification failed for secret_token")}
	r, _, tok = newTestRouter77(t, fs)
	w = authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`{"username":"u2","password":"p2"}`))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "password",
		"sanitizeError 必须把 password 关键词的错误脱敏为通用消息")
	assert.NotContains(t, body, "secret_token",
		"原始敏感 token 不应泄漏到响应体")
}

func TestHdl77_DeleteAccount(t *testing.T) {
	fs := &fakeStrategy{deleteErr: errors.New("ordinary delete failure")}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodDelete, "/api/v1/accounts/u1", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, 1, fs.deleteCalls)
	assert.Contains(t, w.Body.String(), "ordinary delete failure")
}

func TestHdl77_ResetPassword(t *testing.T) {
	// bind JSON 400
	r, _, tok := newTestRouter77(t, nil)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts/u1/reset",
		strings.NewReader(`not json`))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 成功 200
	fs := &fakeStrategy{}
	r, _, tok = newTestRouter77(t, fs)
	w = authedServe(r, tok, http.MethodPost, "/api/v1/accounts/u1/reset",
		strings.NewReader(`{"new_password":"newpw"}`))
	assert.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, fs.resetCalls)
	require.NotNil(t, fs.lastCreate)
	assert.Equal(t, "u1", fs.lastCreate.Username)
	assert.Equal(t, "newpw", fs.lastCreate.Password)
}

func TestHdl77_EnableAccount(t *testing.T) {
	fs := &fakeStrategy{}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts/u1/enable", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, fs.enableCalls)
}

func TestHdl77_DisableAccount(t *testing.T) {
	fs := &fakeStrategy{disableErr: errors.New("disable failure")}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts/u1/disable", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1, fs.disableCalls)
}

func TestHdl77_ListAccounts(t *testing.T) {
	fs := &fakeStrategy{listResult: []string{"u1", "u2"}}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodGet, "/api/v1/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "u1")
	assert.Contains(t, body, "u2")
	assert.Contains(t, body, `"total":2`)
	assert.Equal(t, 1, fs.listCalls)
}

func TestHdl77_ListAccounts_JSONShape(t *testing.T) {
	fs := &fakeStrategy{listResult: []string{"alpha", "beta"}}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodGet, "/api/v1/accounts", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Accounts []string `json:"accounts"`
		Total    int      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, []string{"alpha", "beta"}, payload.Accounts)
	assert.Equal(t, 2, payload.Total)
}

// =====================================================================
// Register / Heartbeat handler — 走后端 RegisterToBackend / SendHeartbeat
// Register 公开; Heartbeat 受 JWTAuth 保护
// =====================================================================

func TestHdl77_RegisterHandler_BadJSON(t *testing.T) {
	r, _, _ := newTestRouter77(t, nil)
	w := authedServe(r, "", http.MethodPost, "/api/v1/register",
		strings.NewReader(`not json at all`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHdl77_RegisterHandler_Success(t *testing.T) {
	r, _, _ := newTestRouter77(t, nil)
	w := authedServe(r, "", http.MethodPost, "/api/v1/register",
		strings.NewReader(`{"vm_id":"v1","agent_id":"a1"}`))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHdl77_RegisterHandler_Failure(t *testing.T) {
	r, b, _ := newTestRouter77(t, nil)
	b.InstallHook(APIPathRegister, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":1,"message":"register rejected"}`)
	})
	w := authedServe(r, "", http.MethodPost, "/api/v1/register",
		strings.NewReader(`{"vm_id":"v1","agent_id":"a1"}`))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHdl77_HeartbeatHandler_Success(t *testing.T) {
	r, _, tok := newTestRouter77(t, nil)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/heartbeat", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHdl77_HeartbeatHandler_Failure(t *testing.T) {
	r, b, tok := newTestRouter77(t, nil)
	b.InstallHook(APIPathHeartbeat, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	w := authedServe(r, tok, http.MethodPost, "/api/v1/heartbeat", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =====================================================================
// 公开端点: HealthCheck (公开) / WebSocketTerminal (受保护 → 501)
// =====================================================================

func TestHdl77_HealthCheck(t *testing.T) {
	r, _, _ := newTestRouter77(t, nil)
	w := authedServe(r, "", http.MethodPost, "/api/v1/health", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestHdl77_WebSocketTerminal_NotImplemented(t *testing.T) {
	r, _, tok := newTestRouter77(t, nil)
	w := authedServe(r, tok, http.MethodGet, "/api/v1/console/ws", nil)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

// =====================================================================
// sanitizeError 覆盖 — 错误串含 password / token / sql / C:\\ 关键词
// → 通用错误; 非敏感错误 → 原样返回
// =====================================================================

func TestHdl77_SanitizeError_NonSensitive(t *testing.T) {
	fs := &fakeStrategy{createErr: errors.New("ordinary business failure")}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`{"username":"u","password":"p"}`))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "ordinary business failure")
}

func TestHdl77_SanitizeError_DatabaseLeak(t *testing.T) {
	fs := &fakeStrategy{createErr: errors.New("sql: syntax error near SELECT password FROM users")}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`{"username":"u","password":"p"}`))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "SELECT password",
		"sql 错误不应泄漏查询语句原文")
	assert.Contains(t, body, "请求处理失败",
		"sanitize 后的通用错误消息应出现")
}

func TestHdl77_SanitizeError_WindowsPath(t *testing.T) {
	fs := &fakeStrategy{createErr: errors.New(`read file C:\Users\admin\secret.txt failed`)}
	r, _, tok := newTestRouter77(t, fs)
	w := authedServe(r, tok, http.MethodPost, "/api/v1/accounts",
		strings.NewReader(`{"username":"u","password":"p"}`))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "secret.txt",
		"Windows 路径不应出现在响应体")
}

// =====================================================================
// JWTAuth 端到端 — valid token 200 / invalid token 401 / missing 401
// =====================================================================

func TestHdl77_JWTAuth_ValidToken(t *testing.T) {
	r, _, tok := newTestRouter77(t, &fakeStrategy{listResult: []string{"u1"}})
	w := authedServe(r, tok, http.MethodGet, "/api/v1/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "u1")
}

func TestHdl77_JWTAuth_InvalidToken(t *testing.T) {
	r, _, _ := newTestRouter77(t, &fakeStrategy{listResult: []string{"x"}})
	w := authedServe(r, "this-is-not-a-jwt", http.MethodGet, "/api/v1/accounts", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHdl77_JWTAuth_MissingHeader(t *testing.T) {
	r, _, _ := newTestRouter77(t, nil)
	w := authedServe(r, "", http.MethodGet, "/api/v1/accounts", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHdl77_JWTAuth_BadScheme(t *testing.T) {
	r, _, _ := newTestRouter77(t, nil)
	// 用合法 token 之外的 scheme 直接构造请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =====================================================================
// pty_manager — 零 Skipf (ROADMAP Skipf 兜底备注经 RESEARCH 实证可作废)
// =====================================================================

func TestPty77_CreateSession_NotImplemented(t *testing.T) {
	m := NewPTYManager()
	_, err := m.CreateSession(context.Background(), "s1", "/bin/sh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestPty77_CloseSession_NotImplemented(t *testing.T) {
	m := NewPTYManager()
	err := m.CloseSession("s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestPty77_WriteAndReadSession(t *testing.T) {
	m := NewPTYManager()
	// 同包白盒直插 session (非真 pty)。Input 与 Output 共用同一 buffer channel
	// 才能模拟「写读往返」 — 生产 pty 是 Input→pty 输入, pty→Output 输出,
	// 测试不模拟真 pty 进程, 用共享 channel 短路验证 Manager API 行为。
	shared := make(chan string, 4)
	m.sessions["s1"] = &ptySession{
		ID:     "s1",
		Input:  shared,
		Output: shared,
		Done:   make(chan struct{}),
	}

	require.NoError(t, m.WriteToSession("s1", "hello"))
	got, err := m.ReadFromSession("s1")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	assert.Equal(t, []string{"s1"}, m.ListSessions())
}

func TestPty77_SessionNotFound(t *testing.T) {
	m := NewPTYManager()
	err := m.WriteToSession("nope", "data")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")

	_, err = m.ReadFromSession("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestPty77_WriteBufferFull(t *testing.T) {
	m := NewPTYManager()
	m.sessions["full"] = &ptySession{
		ID:     "full",
		Input:  make(chan string), // 无 buffer → 立即 full
		Output: make(chan string),
	}
	err := m.WriteToSession("full", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session input buffer full")
}

func TestPty77_ReadEmpty(t *testing.T) {
	m := NewPTYManager()
	m.sessions["e"] = &ptySession{
		ID:     "e",
		Input:  make(chan string, 1),
		Output: make(chan string),
	}
	_, err := m.ReadFromSession("e")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no data available")
}

func TestPty77_ListSessions_EmptyAndMultiple(t *testing.T) {
	m := NewPTYManager()
	assert.Empty(t, m.ListSessions())
	m.sessions["a"] = &ptySession{ID: "a"}
	m.sessions["b"] = &ptySession{ID: "b"}
	got := m.ListSessions()
	assert.Len(t, got, 2)
	assert.Contains(t, got, "a")
	assert.Contains(t, got, "b")
}

// =====================================================================
// logger 收尾 — InitLogger logPath 分支 / WithContext 取值 / WithRequestID
// 缺省与预设 / Fatal 经 ExitFunc 接管不杀进程
// =====================================================================

func TestHdl77_InitLogger_WithLogPath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, InitLogger("info", dir))
	// 立刻关闭 logFile 句柄 — Windows 上 TempDir RemoveAll 在文件仍被打开
	// 时会失败。logrus.SetOutput 仅替换 writer 不关闭旧 writer, 需显式
	// type-assert 到 *os.File 后 Close 才能让 TempDir 清理成功。
	defer func() {
		if f, ok := logger.Out.(*os.File); ok {
			_ = f.Close()
		}
		logger.SetOutput(io.Discard)
	}()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var found bool
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		if e.Name() == "agent.log" {
			found = true
		}
	}
	assert.True(t, found, "agent.log 应在 logPath 下创建 (实际: %v)", names)
}

func TestHdl77_WithContext_ExtractsValues(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	ctx := context.WithValue(context.Background(), "request_id", "rid-77-05")
	ctx = context.WithValue(ctx, "user_id", "u-99")
	ctx = context.WithValue(ctx, "agent_id", "a-77-05")
	entry := WithContext(ctx)
	require.NotNil(t, entry)
	assert.Equal(t, "rid-77-05", entry.Data["request_id"])
	assert.Equal(t, "u-99", entry.Data["user_id"])
	assert.Equal(t, "a-77-05", entry.Data["agent_id"])
}

func TestHdl77_WithContext_NilContext(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	entry := WithContext(context.TODO())
	require.NotNil(t, entry)
	// 无值时不会 panic, 字段为空
}

func TestHdl77_WithRequestID_EmptyGeneratesUUID(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	e := WithRequestID("")
	require.NotNil(t, e)
	rid, _ := e.Data["request_id"].(string)
	_, err := uuid.Parse(rid)
	assert.NoError(t, err, "空 request_id 应被填为合法 UUID")
}

func TestHdl77_WithRequestID_Preserved(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	e := WithRequestID("preset-rid")
	require.NotNil(t, e)
	assert.Equal(t, "preset-rid", e.Data["request_id"])
}

func TestHdl77_WithFields(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	e := WithFields(logrus.Fields{"k": "v", "n": 1})
	require.NotNil(t, e)
	assert.Equal(t, "v", e.Data["k"])
	assert.Equal(t, 1, e.Data["n"])
}

func TestHdl77_Fatal_NoExitOnExitFuncOverride(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	// logrus.ExitFunc 接管退出钩子: 不杀测试进程
	origExit := logger.ExitFunc
	logger.ExitFunc = func(_ int) {}
	t.Cleanup(func() { logger.ExitFunc = origExit })
	assert.NotPanics(t, func() {
		Fatal("this would normally exit")
	})
}

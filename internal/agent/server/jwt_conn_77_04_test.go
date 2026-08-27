package server

// =====================================================================
// jwt_conn_77_04_test.go — Phase 77 Plan 04 (BLOCK-02 主力)
//
// 覆盖范围: jwt_auth.go 全函数 + connection_manager.go 状态机 (191 stmts)。
//
// 假后端 seam (77-05 可直接复用):
//   NewJWTAuthenticator(secret, backendURL, ...) 的 backendURL 是明文构造参数
//   (jwt_auth.go:61) → newAgentBackend77(t) 建 httptest.NewServer 本地回环假
//   后端, 把 srv.URL 直接喂进去。全部出站请求只可能命中 srv.URL, 仓内零真实
//   外呼 (T-77-04-01)。
//
// 纪律:
//   - 白盒字段 (tokenExpiryAt / reconnectDelay / stopCh) 覆盖前先 t.Cleanup 恢复;
//   - 禁 t.Parallel (全局 logger 与白盒字段共享);
//   - 无裸 sleep — goroutine 断言一律 channel 同步 + 超时护栏 (P-77-4);
//   - 测试凭据仅 "test-secret"/fakeToken 字面量 (T-77-04-03)。
// =====================================================================

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// =====================================================================
// 假后端 (本地回环 httptest.NewServer)
// =====================================================================

// fakeToken77 假后端下发到 currentToken 的字面量 —— 不是合法 JWT,
// 仅用于验证 RegisterToBackend 是否原样提取了 data.token。
const fakeToken77 = "fakeToken-77-04-definitely-not-a-jwt"

// agentBackend77 agent 出站链路的假后端。按 r.URL.Path 分发 APIPath* 常量,
// 默认返回 pkg/response 响应壳 {"code":0,...}; 单测可用 InstallHook 覆写某条
// 路径的响应形态来驱动失败分支。
type agentBackend77 struct {
	t   *testing.T
	srv *httptest.Server

	mu     sync.Mutex
	calls  map[string]int
	auths  map[string][]string // path -> 每次请求的 Authorization 头
	bodies map[string][][]byte // path -> 每次请求的原始 body
	hooks  map[string]http.HandlerFunc
}

// newAgentBackend77 启动本地回环假后端并绑定 t.Cleanup(srv.Close)。
func newAgentBackend77(t *testing.T) *agentBackend77 {
	t.Helper()
	b := &agentBackend77{
		t:      t,
		calls:  make(map[string]int),
		auths:  make(map[string][]string),
		bodies: make(map[string][][]byte),
		hooks:  make(map[string]http.HandlerFunc),
	}
	b.srv = httptest.NewServer(http.HandlerFunc(b.serve))
	t.Cleanup(b.srv.Close)
	return b
}

// serve 先记录请求 (并回填 body 供分发器复读), 再走覆写 hook 或默认成功响应。
// 注意: hook 在服务端 goroutine 里执行, 其内部只允许 t.Errorf (安全),
// 严禁 require./FailNow 族 (跨 goroutine 不安全)。
func (b *agentBackend77) serve(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()

	b.mu.Lock()
	b.calls[r.URL.Path]++
	b.auths[r.URL.Path] = append(b.auths[r.URL.Path], r.Header.Get("Authorization"))
	b.bodies[r.URL.Path] = append(b.bodies[r.URL.Path], raw)
	hook := b.hooks[r.URL.Path]
	b.mu.Unlock()

	r.Body = io.NopCloser(strings.NewReader(string(raw)))

	if hook != nil {
		hook(w, r)
		return
	}

	switch r.URL.Path {
	case APIPathHeartbeat:
		// 出站契约: Authorization: Bearer + 身份四件套 body
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			b.t.Errorf("%s 必须带 Authorization: Bearer <token>, 实际 %q", APIPathHeartbeat, authz)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"code":401,"message":"missing bearer"}`)
			return
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			b.t.Errorf("heartbeat body 不是合法 JSON: %v", err)
		}
		for _, key := range []string{"agent_id", "vm_id", "status", "timestamp"} {
			if _, ok := payload[key]; !ok {
				b.t.Errorf("heartbeat body 缺少身份字段 %q", key)
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"code":0,"message":"success"}`)
	case APIPathStatus:
		_, _ = io.WriteString(w, `{"code":0,"message":"success","data":{"ok":true}}`)
	case APIPathRegister:
		_, _ = fmt.Fprintf(w, `{"code":0,"message":"success","data":{"token":%q}}`, fakeToken77)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, `{"code":404,"message":"unknown path %s"}`, r.URL.Path)
	}
}

// InstallHook 覆写指定路径的默认响应 (单测内调用, 非并发场景)。
func (b *agentBackend77) InstallHook(path string, h http.HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hooks[path] = h
}

func (b *agentBackend77) CallsFor(path string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls[path]
}

func (b *agentBackend77) LastAuth(path string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	seq := b.auths[path]
	if len(seq) == 0 {
		return ""
	}
	return seq[len(seq)-1]
}

// LastBody 解析最后一条请求体为 JSON map (失败即断言报错)。
func (b *agentBackend77) LastBody(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	b.mu.Lock()
	last := b.bodies[path][len(b.bodies[path])-1]
	b.mu.Unlock()
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(last, &payload), "%s 请求体应为合法 JSON", path)
	return payload
}

// newJWT77 以假后端地址构造被测 authenticator (T-77-04-01: 只打本地回环)。
func newJWT77(t *testing.T, b *agentBackend77) *JWTAuthenticator {
	t.Helper()
	require.NoError(t, InitLogger("info", ""), "P-77-5: StartHealthMonitor/Warn 依赖全局 logger")
	return NewJWTAuthenticator("test-secret", b.srv.URL, "agent-1", "vm-1", nil)
}

// =====================================================================
// 状态回调同步工具 (channel, 禁裸 sleep)
// =====================================================================

func newStateRecorder77(capacity int) (chan ConnectionState, func(ConnectionState)) {
	ch := make(chan ConnectionState, capacity)
	return ch, func(s ConnectionState) {
		select {
		case ch <- s:
		default: // 溢出即丢弃, 保证回调永不阻塞生产方
		}
	}
}

// waitStates77 从 channel 定长收取 n 个回调, 超时即失败 (channel 同步等待)。
func waitStates77(t *testing.T, ch chan ConnectionState, n int, timeout time.Duration) []ConnectionState {
	t.Helper()
	out := make([]ConnectionState, 0, n)
	for i := 0; i < n; i++ {
		select {
		case s := <-ch:
			out = append(out, s)
		case <-time.After(timeout):
			t.Fatalf("等待状态回调超时: 已收 %d/%d 个 %v", i, n, out)
		}
	}
	return out
}

func assertStatesEmpty77(t *testing.T, ch chan ConnectionState) {
	t.Helper()
	select {
	case s := <-ch:
		t.Fatalf("不应再收到状态回调, 但收到 %v", s)
	default:
	}
}

// =====================================================================
// jwt_auth.go — CallBackend 及其包装
// =====================================================================

func TestJWT77_CallBackend_HeartbeatSuccessContract(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	// CallBackend 透传调用方给的 body —— 身份注入是 SendHeartbeat 包装层的事,
	// 此处自备完整契约字段验证透传无损。
	resp, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathHeartbeat, map[string]interface{}{
		"agent_id":  "agent-1",
		"vm_id":     "vm-1",
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	assert.Equal(t, 1, b.CallsFor(APIPathHeartbeat))
	assert.True(t, strings.HasPrefix(b.LastAuth(APIPathHeartbeat), "Bearer "), "必须携带 Bearer 授权头")
	payload := b.LastBody(t, APIPathHeartbeat)
	assert.Equal(t, "healthy", payload["status"])
	assert.Equal(t, "agent-1", payload["agent_id"])
	assert.Equal(t, "vm-1", payload["vm_id"])
}

func TestJWT77_CallBackend_StatusPath(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	resp, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathStatus, map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, 1, b.CallsFor(APIPathStatus))
}

func TestJWT77_CallBackend_TrailingSlashTrimmed(t *testing.T) {
	// backendURL 尾部 "/" 被 TrimSuffix 移除, 不得产生双斜杠路径
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, APIPathStatus, r.URL.Path, "路径必须恰好是 APIPath 常量")
		_, _ = io.WriteString(w, `{"code":0,"message":"success"}`)
	}))
	t.Cleanup(srv.Close)

	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", srv.URL+"/", "agent-1", "vm-1", nil)
	resp, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathStatus, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

func TestJWT77_CallBackend_DecodeErrorBranch(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	b.InstallHook(APIPathHeartbeat, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html>definitely not json</html>")
	})

	_, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathHeartbeat, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestJWT77_CallBackend_BackendDownBranch(t *testing.T) {
	b := newAgentBackend77(t)
	url := b.srv.URL
	b.srv.Close() // 幂等: Cleanup 再 Close 安全

	auth := NewJWTAuthenticator("test-secret", url, "agent-1", "vm-1", nil)
	_, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathHeartbeat, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend request failed")
}

func TestJWT77_CallBackend_TLSVerificationErrorBranch(t *testing.T) {
	// 自签 https 假后端 + 默认强校验客户端 (nil tlsConfig → verify on)
	// → 握手报 x509 未知权威 → 命中 CallBackend 的 TLS 错误包装分支
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"message":"success"}`)
	}))
	t.Cleanup(tlsSrv.Close)

	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", tlsSrv.URL, "agent-1", "vm-1", nil)

	_, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathStatus, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS certificate verification failed")
}

func TestJWT77_CallBackend_NilBody_NoTypedNilPanic(t *testing.T) {
	// Q-77-D 回归: CallBackend 声明 var bodyReader *bytes.Reader 而 body==nil 时
	// 不赋值, 传给 http.NewRequestWithContext 的是"接口持 typed-nil 指针",
	// stdlib 按 case *bytes.Reader 调 v.Len() 读字段 → nil 解引用 panic。
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	var gotResp *response.Response
	require.NotPanics(t, func() {
		resp, err := auth.CallBackend(context.Background(), http.MethodPost, APIPathStatus, nil)
		require.NoError(t, err)
		gotResp = resp
	})
	require.NotNil(t, gotResp)
	assert.Equal(t, 0, gotResp.Code)
	assert.Equal(t, 1, b.CallsFor(APIPathStatus))
}

func TestJWT77_SendHeartbeat_ThroughCallBackend(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	require.NoError(t, auth.SendHeartbeat(context.Background()))
	assert.Equal(t, 1, b.CallsFor(APIPathHeartbeat))
	payload := b.LastBody(t, APIPathHeartbeat)
	assert.Equal(t, "healthy", payload["status"])
	assert.Equal(t, "agent-1", payload["agent_id"])
	assert.Equal(t, "vm-1", payload["vm_id"])
	require.Contains(t, payload, "timestamp", "心跳必须带时间戳")
}

func TestJWT77_ReportSystemStatus_InjectsIdentityAndKeepsCustomFields(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	status := map[string]interface{}{"cpu_load": "low"}
	require.NoError(t, auth.ReportSystemStatus(context.Background(), status))

	// 生产实现会就地回填身份字段进调用方 map
	assert.Equal(t, "agent-1", status["agent_id"])
	assert.Equal(t, "vm-1", status["vm_id"])

	payload := b.LastBody(t, APIPathStatus)
	assert.Equal(t, "low", payload["cpu_load"], "自定义字段必须保留")
	assert.Equal(t, "agent-1", payload["agent_id"])
	assert.Equal(t, "vm-1", payload["vm_id"])
	require.Contains(t, payload, "timestamp")
}

// =====================================================================
// jwt_auth.go — RegisterToBackend / Register
// =====================================================================

func TestJWT77_RegisterToBackend_ExtractsTokenIntoCurrentToken(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	regData := map[string]interface{}{"hostname": "win-agent-host"}
	require.NoError(t, auth.RegisterToBackend(context.Background(), regData))

	// token 必须从 data.token 原样提取到 currentToken
	auth.mu.RLock()
	stored := auth.currentToken
	auth.mu.RUnlock()
	assert.Equal(t, fakeToken77, stored)

	// 注册体注入身份三件套
	payload := b.LastBody(t, APIPathRegister)
	assert.Equal(t, "agent-1", payload["agent_id"])
	assert.Equal(t, "vm-1", payload["vm_id"])
	assert.Equal(t, runtime.GOOS, payload["platform"])
	assert.Equal(t, "win-agent-host", payload["hostname"])
}

func TestJWT77_RegisterToBackend_CodeNonZeroFails(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	b.InstallHook(APIPathRegister, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":5001,"message":"重复注册"}`)
	})

	err := auth.RegisterToBackend(context.Background(), map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "注册失败")
	assert.Contains(t, err.Error(), "重复注册")
}

// prefillValidLocalToken77 白盒注入一个仍有效的已知令牌, 隔离隐式刷新副作用。
func prefillValidLocalToken77(t *testing.T, auth *JWTAuthenticator) string {
	t.Helper()
	token, err := auth.generateToken()
	require.NoError(t, err)
	auth.mu.Lock()
	origToken, origExpiry := auth.currentToken, auth.tokenExpiryAt
	auth.currentToken = token
	auth.tokenExpiryAt = time.Now().Add(2 * time.Hour)
	auth.mu.Unlock()
	t.Cleanup(func() {
		auth.mu.Lock()
		auth.currentToken = origToken
		auth.tokenExpiryAt = origExpiry
		auth.mu.Unlock()
	})
	return token
}

func TestJWT77_RegisterToBackend_DataNilKeepsLocalToken(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	b.InstallHook(APIPathRegister, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"message":"success"}`)
	})
	prefilled := prefillValidLocalToken77(t, auth)

	// 无 data 字段时既不该走 data.token 提取也不该破坏现值
	require.NoError(t, auth.RegisterToBackend(context.Background(), map[string]interface{}{}))

	auth.mu.RLock()
	stored := auth.currentToken
	auth.mu.RUnlock()
	assert.Equal(t, prefilled, stored, "响应缺 data 时不得触碰 currentToken")
}

func TestJWT77_RegisterToBackend_EmptyTokenIgnored(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	b.InstallHook(APIPathRegister, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"message":"success","data":{"token":""}}`)
	})
	prefilled := prefillValidLocalToken77(t, auth)

	// 空串 token 过不了 `token != ""` 判定 → 提取分支整体跳过
	require.NoError(t, auth.RegisterToBackend(context.Background(), map[string]interface{}{}))

	auth.mu.RLock()
	stored := auth.currentToken
	auth.mu.RUnlock()
	assert.Equal(t, prefilled, stored, "空 data.token 不提取, 必须保持原值")
}

func TestJWT77_Register_LocalOnlyZeroHTTP(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)

	require.NoError(t, auth.Register(context.Background()))

	auth.mu.RLock()
	token := auth.currentToken
	expiry := auth.tokenExpiryAt
	auth.mu.RUnlock()

	assert.NotEmpty(t, token)
	assert.True(t, time.Now().Add(time.Hour).Before(expiry), "注册后过期时间应落在 24h 默认值附近")

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", claims.AgentID)
	assert.Equal(t, "vm-1", claims.VMID)

	// 零 HTTP: 所有 API 路径零命中
	for _, p := range []string{APIPathHeartbeat, APIPathStatus, APIPathRegister, APIPathTokenRefresh} {
		assert.Zero(t, b.CallsFor(p), "本地 Register 不应触达 %s", p)
	}
}

// =====================================================================
// jwt_auth.go — RefreshToken / GetCurrentToken / ValidateToken / ParseTokenClaims
// =====================================================================

func TestJWT77_RefreshToken_ValidSkipBranch(t *testing.T) {
	auth := newJWT77(t, newAgentBackend77(t))

	auth.mu.Lock()
	origToken := "still-valid-77"
	origExpiry := time.Now().Add(2 * time.Hour)
	auth.currentToken = origToken
	auth.tokenExpiryAt = origExpiry
	auth.mu.Unlock()
	t.Cleanup(func() {
		auth.mu.Lock()
		auth.currentToken = ""
		auth.tokenExpiryAt = time.Time{}
		auth.mu.Unlock()
	})

	require.NoError(t, auth.RefreshToken(context.Background()))

	auth.mu.RLock()
	defer auth.mu.RUnlock()
	assert.Equal(t, origToken, auth.currentToken, "令牌仍有效时 RefreshToken 必须跳过重新生成")
	assert.Equal(t, origExpiry, auth.tokenExpiryAt)
}

func TestJWT77_RefreshToken_ExpiredRegenerates(t *testing.T) {
	auth := newJWT77(t, newAgentBackend77(t))

	auth.mu.Lock()
	auth.currentToken = "expired-77"
	auth.tokenExpiryAt = time.Now().Add(-time.Minute)
	auth.mu.Unlock()
	t.Cleanup(func() {
		auth.mu.Lock()
		auth.currentToken = ""
		auth.tokenExpiryAt = time.Time{}
		auth.mu.Unlock()
	})

	require.NoError(t, auth.RefreshToken(context.Background()))

	auth.mu.RLock()
	token, expiry := auth.currentToken, auth.tokenExpiryAt
	auth.mu.RUnlock()

	assert.NotEqual(t, "expired-77", token, "过期令牌必须重新生成")
	assert.True(t, time.Now().Add(20*time.Hour).Before(expiry), "新令牌有效期应接近 24h 默认值")

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", claims.AgentID)
}

func TestJWT77_GetCurrentToken_FastPathAndFallback(t *testing.T) {
	auth := newJWT77(t, newAgentBackend77(t))

	// 快路径: 仍有效 → 原样返回
	auth.mu.Lock()
	auth.currentToken = "valid-fast-path"
	auth.tokenExpiryAt = time.Now().Add(2 * time.Hour)
	auth.mu.Unlock()
	t.Cleanup(func() {
		auth.mu.Lock()
		auth.currentToken = ""
		auth.tokenExpiryAt = time.Time{}
		auth.mu.Unlock()
	})

	got, err := auth.GetCurrentToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "valid-fast-path", got)

	// 降级: 过期 → 经 RefreshToken 重新生成
	auth.mu.Lock()
	auth.tokenExpiryAt = time.Now().Add(-time.Minute)
	auth.mu.Unlock()

	fallback, err := auth.GetCurrentToken(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, "valid-fast-path", fallback)
	assert.NotEmpty(t, fallback)
}

func TestJWT77_ValidateToken_Matrix(t *testing.T) {
	goodAuth := newJWT77(t, newAgentBackend77(t))

	// 好令牌
	claims, err := goodAuth.generateToken()
	require.NoError(t, err)
	parsed, err := goodAuth.ValidateToken(claims)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", parsed.AgentID)
	assert.Equal(t, "vm-1", parsed.VMID)

	// 错误 secret 签名 → 校验失败
	otherAuth := NewJWTAuthenticator("another-secret", "http://127.0.0.1:1", "agent-1", "vm-1", nil)
	wrongSigned, err := otherAuth.generateToken()
	require.NoError(t, err)
	_, err = goodAuth.ValidateToken(wrongSigned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature")

	// 非 HMAC 签名方法 → 命中 errInvalidSigningMethod 分支
	noneToken := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, jwtClaims{
		AgentID: "agent-1",
		VMID:    "vm-1",
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	noneSigned, err := noneToken.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = goodAuth.ValidateToken(noneSigned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "意外的签名方法")

	// 垃圾串
	_, err = goodAuth.ValidateToken("totally-not-a-jwt")
	require.Error(t, err)
}

func TestJWT77_ParseTokenClaims_Matrix(t *testing.T) {
	newJWT77(t, newAgentBackend77(t)) // 保持包内 logger 一致性

	auth := NewJWTAuthenticator("test-secret", "http://127.0.0.1:1", "agent-1", "vm-1", nil)
	token, err := auth.generateToken()
	require.NoError(t, err)

	result, err := ParseTokenClaims(token)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", result["agent_id"])
	assert.Equal(t, "vm-1", result["vm_id"])
	require.Contains(t, result, "exp")
	require.Contains(t, result, "iat")

	_, err = ParseTokenClaims("garbage")
	require.Error(t, err)

	_, err = ParseTokenClaims("")
	require.Error(t, err)
}

// =====================================================================
// jwt_auth.go — 错误类型与 TLS 配置
// =====================================================================

func TestJWT77_ErrorTypes_Formatting(t *testing.T) {
	authErr := &AuthError{StatusCode: 401, Message: "令牌无效"}
	assert.Contains(t, authErr.Error(), "auth error")
	assert.Contains(t, authErr.Error(), "令牌无效")
	assert.Contains(t, authErr.Error(), "401")

	httpErr := &HTTPError{StatusCode: 503, Message: "后端过载"}
	assert.Contains(t, httpErr.Error(), "http error")
	assert.Contains(t, httpErr.Error(), "后端过载")
	assert.Contains(t, httpErr.Error(), "503")
}

// writeSelfSignedCertPair77 用 stdlib (crypto/ecdsa + crypto/x509) 自签一张
// 同时可用作 CA 与 mTLS 证书对的证书, 写入 t.TempDir 返回三个 PEM 路径。
func writeSelfSignedCertPair77(t *testing.T) (caFile, certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "agent-test-77-04"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	caFile = filepath.Join(dir, "ca.pem")
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	writePEM := func(path, blockType string, blob []byte) {
		t.Helper()
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, pem.Encode(f, &pem.Block{Type: blockType, Bytes: blob}))
		require.NoError(t, f.Close())
	}
	writePEM(caFile, "CERTIFICATE", der)
	writePEM(certFile, "CERTIFICATE", der)
	writePEM(keyFile, "EC PRIVATE KEY", keyDER)
	return caFile, certFile, keyFile
}

func TestJWT77_NewTLSConfigFromConfig_Matrix(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))

	// --- 错误分支 (扩展 agent_smoke_test.go:187-197 既有覆盖) ---
	cfg, err := NewTLSConfigFromConfig("", "", "", true)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "TLS 配置不能全空")

	_, err = NewTLSConfigFromConfig("", "", "/nonexistent/ca-77-04.pem", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CA file")

	// 坏 CA 内容 → parse 失败
	junkCA := filepath.Join(t.TempDir(), "junk-ca.pem")
	require.NoError(t, os.WriteFile(junkCA, []byte("this is not a pem"), 0o600))
	_, err = NewTLSConfigFromConfig("", "", junkCA, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CA certificate")

	// 坏证书对 → LoadX509KeyPair 失败
	badCert := filepath.Join(t.TempDir(), "bad-cert.pem")
	badKey := filepath.Join(t.TempDir(), "bad-key.pem")
	require.NoError(t, os.WriteFile(badCert, []byte("junk"), 0o600))
	require.NoError(t, os.WriteFile(badKey, []byte("junk"), 0o600))
	_, err = NewTLSConfigFromConfig(badCert, badKey, "", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load client certificate")

	// --- Happy path #1: 仅 CA 文件 ---
	caFile, certFile, keyFile := writeSelfSignedCertPair77(t)
	cfg, err = NewTLSConfigFromConfig("", "", caFile, true)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.NotNil(t, cfg.RootCAs)
	assert.False(t, cfg.InsecureSkipVerify, "verifyCertificates=true 时必须校验")
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	assert.Empty(t, cfg.Certificates, "未提供 cert/key 时不应有客户端证书")

	// --- Happy path #2: cert+key mTLS 对 (复用同一张自签证书) ---
	cfg, err = NewTLSConfigFromConfig(certFile, keyFile, "", true)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Certificates, 1)
	assert.False(t, cfg.InsecureSkipVerify)

	// --- Happy path #3: cert+key+ca 三件套 ---
	cfg, err = NewTLSConfigFromConfig(certFile, keyFile, caFile, true)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.RootCAs)

	// --- InsecureSkipVerify 分支: verify=false → 跳过校验 (会 Warn) ---
	cfg, err = NewTLSConfigFromConfig("", "", caFile, false)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.InsecureSkipVerify, "verifyCertificates=false 时应置 InsecureSkipVerify")

	// mTLS 证书对确实能被标准库按对加载 (LoadX509KeyPair 二次自检)
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)
	require.NotEmpty(t, pair.Certificate)
}

// =====================================================================
// connection_manager.go — 状态机
// =====================================================================

func TestCM77_Connect_Success_StateMachineAndCallbacks(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(32)
	cm.SetStateChangeCallback(cb)

	require.NoError(t, cm.Connect(context.Background()))

	// 回调序列必须精确是 [Connecting, Connected]
	seq := waitStates77(t, ch, 2, 2*time.Second)
	assert.Equal(t, []ConnectionState{Connecting, Connected}, seq,
		"连接成功链必须是 Connecting → Register(本地) → SendHeartbeat(HTTP) → Connected")
	assertStatesEmpty77(t, ch)

	require.Equal(t, Connected, cm.GetState())
	require.True(t, cm.IsConnected())

	stats := cm.GetStats()
	assert.Equal(t, "connected", stats["state"])
	assert.Equal(t, 0, stats["reconnect_count"])
	assert.False(t, stats["last_connected"].(time.Time).IsZero())

	// 心跳恰好一次且带完整身份
	assert.Equal(t, 1, b.CallsFor(APIPathHeartbeat))
}

func TestCM77_Connect_HeartbeatFailure_CallbacksAndError(t *testing.T) {
	b := newAgentBackend77(t)
	url := b.srv.URL
	b.srv.Close() // 关停假后端 → 心跳阶段 connection refused (幂等 Cleanup)

	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", url, "agent-1", "vm-1", nil)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(32)
	cm.SetStateChangeCallback(cb)

	err := cm.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "heartbeat failed")

	// 回调序列必须精确是 [Connecting, Disconnected]
	seq := waitStates77(t, ch, 2, 2*time.Second)
	assert.Equal(t, []ConnectionState{Connecting, Disconnected}, seq)
	assertStatesEmpty77(t, ch)

	require.Equal(t, Disconnected, cm.GetState())
	require.False(t, cm.IsConnected())
	stats := cm.GetStats()
	assert.Equal(t, "disconnected", stats["state"])
	assert.False(t, stats["last_disconnect"].(time.Time).IsZero())
	// 注: registration 失败分支 (connection_manager.go:93-101) 在 Register 为
	// 本地实现的现状下不可达 —— 接受不覆盖 (D-03 有据判定, 见 SUMMARY)。
}

func TestCM77_Reconnect_SuccessAfterDelay(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(32)
	cm.SetStateChangeCallback(cb)

	// 白盒缩小时延 (先注册 Cleanup 恢复)
	origDelay := cm.reconnectDelay
	cm.reconnectDelay = time.Millisecond
	t.Cleanup(func() { cm.reconnectDelay = origDelay })

	require.NoError(t, cm.Reconnect(context.Background()))

	// 成功重连链: Reconnecting → (延迟) → Connect → Connecting → Connected
	seq := waitStates77(t, ch, 3, 2*time.Second)
	assert.Equal(t, []ConnectionState{Reconnecting, Connecting, Connected}, seq)
	assertStatesEmpty77(t, ch)
	require.True(t, cm.IsConnected())
	stats := cm.GetStats()
	assert.Equal(t, "connected", stats["state"])
	assert.Equal(t, 0, stats["reconnect_count"], "Connect 成功会把计数清零")
}

func TestCM77_Reconnect_MaxReconnectsExceeded(t *testing.T) {
	// 上限分支不触网, 后端仅占位保包内假后端形态统一
	_ = newAgentBackend77(t)
	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", "http://127.0.0.1:1", "agent-1", "vm-1", nil)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(32)
	cm.SetStateChangeCallback(cb)

	// 白盒把计数顶满上限 (fresh CM, 无需恢复, 但统一守纪律)
	origCount := cm.reconnectCount
	cm.reconnectCount = cm.maxReconnects
	t.Cleanup(func() { cm.reconnectCount = origCount })

	err := cm.Reconnect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max reconnects")
	assert.EqualValues(t, cm.maxReconnects, cm.reconnectCount)
	assertStatesEmpty77(t, ch)
	assert.Equal(t, Disconnected, cm.GetState(), "超限分支不得改变当前状态")
}

func TestCM77_Reconnect_ContextCancelled(t *testing.T) {
	_ = newAgentBackend77(t)
	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", "http://127.0.0.1:1", "agent-1", "vm-1", nil)
	cm := NewConnectionManager(auth)

	// 白盒缩小延迟以加速进入 select 等待
	origDelay := cm.reconnectDelay
	cm.reconnectDelay = time.Millisecond
	t.Cleanup(func() { cm.reconnectDelay = origDelay })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 预取消 → select 中只有 ctx.Done() 就绪, 结果确定

	err := cm.Reconnect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, Reconnecting, cm.GetState(), "取消发生在延迟窗口内, 状态停留在 Reconnecting")
}

func TestCM77_Reconnect_StopChannelCancels(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	// 白盒给足延迟窗口, 保证 goroutine 停在 select 上等 stopCh
	origDelay := cm.reconnectDelay
	cm.reconnectDelay = 100 * time.Millisecond
	t.Cleanup(func() { cm.reconnectDelay = origDelay })
	origStop := cm.stopCh
	t.Cleanup(func() { cm.stopCh = origStop })

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		done <- result{err: cm.Reconnect(context.Background())}
	}()

	// 直写 stopCh (非缓冲 channel 会与接收方握手, 顺序无关)
	cm.stopCh <- struct{}{}

	select {
	case r := <-done:
		require.Error(t, r.err)
		assert.Contains(t, r.err.Error(), "reconnect canceled")
	case <-time.After(2 * time.Second):
		t.Fatal("Reconnect 未在 stopCh 信号后返回")
	}
	assert.Equal(t, Reconnecting, cm.GetState())
}

func TestCM77_StartHealthMonitor_ReconnectsViaChannels(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(512)
	cm.SetStateChangeCallback(cb)

	origDelay := cm.reconnectDelay
	cm.reconnectDelay = time.Millisecond
	t.Cleanup(func() { cm.reconnectDelay = origDelay })

	// 初始处于断连 → 监控器的 else 分支会 goroutine 触发 Reconnect
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	monitorDone := make(chan struct{})
	go func() {
		cm.StartHealthMonitor(ctx, 5*time.Millisecond)
		close(monitorDone)
	}()

	// channel 同步等待首个 Connected (禁裸 sleep, P-77-4)
	found := false
	for !found {
		select {
		case s := <-ch:
			if s == Connected {
				found = true
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("StartHealthMonitor 应在断连状态下通过 Reconnect 达到 Connected")
		}
	}
	require.True(t, cm.IsConnected())

	// 收尾防泄漏 (T-77-04-02): cancel → 等 monitor 退出 → Disconnect 送 stopCh
	cancel()
	select {
	case <-monitorDone:
	case <-time.After(2 * time.Second):
		t.Fatal("StartHealthMonitor 未在 ctx 取消后退出")
	}
	cm.Disconnect()
}

func TestCM77_StartHealthMonitor_ConnectedArmSpawnsReconnectOnHeartbeatFailure(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	ch, cb := newStateRecorder77(512)
	cm.SetStateChangeCallback(cb)

	origDelay := cm.reconnectDelay
	cm.reconnectDelay = time.Millisecond
	t.Cleanup(func() { cm.reconnectDelay = origDelay })

	// 先健康连接成功 → 监控器循环内 IsConnected 为真, 进入心跳真分支
	require.NoError(t, cm.Connect(context.Background()))
	waitStates77(t, ch, 2, 2*time.Second) // 排空 Connecting/Connected

	// 心跳响应改为垃圾字节 → SendHeartbeat 必然 decode 失败,
	// 已连接态监控器只对该失败派生 go Reconnect(不改本函数状态)
	b.InstallHook(APIPathHeartbeat, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "garbage-not-json")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	monitorDone := make(chan struct{})
	go func() {
		cm.StartHealthMonitor(ctx, 5*time.Millisecond)
		close(monitorDone)
	}()

	// channel 同步等待失败臂的指纹事件: 只有 SendHeartbeat 失败才会派生 Reconnect,
	// 故「已连接后再次出现 Reconnecting」即证明走了失败分支 (P-77-4)
	failureFired, chainCompleted := false, false
	for !chainCompleted {
		select {
		case s := <-ch:
			switch s {
			case Reconnecting:
				failureFired = true
			case Disconnected:
				if failureFired {
					chainCompleted = true // 派生链完整走完: 重连尝试已落定
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatal("已连接态的心跳失败未按预期派生 Reconnect(事件指纹缺失)")
		}
	}

	// 收尾防泄漏 (T-77-04-02): cancel → 等 monitor 退出 → Disconnect 清场
	cancel()
	select {
	case <-monitorDone:
	case <-time.After(2 * time.Second):
		t.Fatal("StartHealthMonitor 未在 ctx 取消后退出")
	}
	cm.Disconnect()
}

func TestCM77_StartHealthMonitor_StopChannelExitArm(t *testing.T) {
	newAgentBackend77(t)
	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("test-secret", "http://127.0.0.1:1", "agent-1", "vm-1", nil)
	cm := NewConnectionManager(auth)

	// 巨大间隔让 ticker 永不就绪 + 初始断连无派生 → select 里唯一可能就绪的
	// 就是 stopCh, 退出路径确定性锁定, 且全程零 goroutine 派生
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitorDone := make(chan struct{})
	go func() {
		cm.StartHealthMonitor(ctx, time.Hour)
		close(monitorDone)
	}()

	select {
	case cm.stopCh <- struct{}{}:
	case <-time.After(time.Second):
		t.Fatal("stopCh 发送被阻塞")
	}

	select {
	case <-monitorDone:
	case <-time.After(2 * time.Second):
		t.Fatal("StartHealthMonitor 应经 stopCh 分支退出")
	}
	assert.Equal(t, Disconnected, cm.GetState())
}

func TestCM77_Misc_AccessorsStringsAndDisconnect(t *testing.T) {
	b := newAgentBackend77(t)
	auth := newJWT77(t, b)
	cm := NewConnectionManager(auth)

	// 初始态
	assert.Equal(t, Disconnected, cm.GetState())
	assert.False(t, cm.IsConnected())
	assert.Equal(t, "disconnected", ConnectionState(Disconnected).String())

	// String 四态 + unknown (鸭子类型转换, 避免 map/序依赖)
	assert.Equal(t, "connecting", Connecting.String())
	assert.Equal(t, "connected", Connected.String())
	assert.Equal(t, "reconnecting", Reconnecting.String())
	assert.Equal(t, "unknown", ConnectionState(99).String())

	// 断连态 Disconnect 不发回调、不改状态 (幂等守卫分支)
	seen := make([]ConnectionState, 0, 8)
	cm.SetStateChangeCallback(func(s ConnectionState) { seen = append(seen, s) })
	cm.Disconnect()
	assert.Equal(t, Disconnected, cm.GetState())
	assert.Empty(t, seen)

	// Reconnecting 态 Disconnect → 发 Disconnected 回调
	cm.mu.Lock()
	cm.state = Reconnecting
	cm.mu.Unlock()
	cm.Disconnect()
	assert.Equal(t, Disconnected, cm.GetState())
	assert.Equal(t, []ConnectionState{Disconnected}, seen)

	// StopHealth... stopCh 发送永不阻塞 (select default 分支)
	done := make(chan struct{})
	go func() { cm.Disconnect(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Disconnect 的 stopCh 发送应当是非阻塞的")
	}

	// notifyStateChange 直调 (callback 非空/为空两态都不 panic)
	cm.notifyStateChange(Connected)
	assert.Equal(t, []ConnectionState{Disconnected, Connected}, seen)

	bare := NewConnectionManager(NewJWTAuthenticator("s", "http://127.0.0.1:1", "a", "v", nil))
	bare.SetStateChangeCallback(nil)
	bare.notifyStateChange(Reconnecting) // nil 回调无操作

	stats := cm.GetStats()
	require.Len(t, stats, 4, "统计必须含 state/last_connected/last_disconnect/reconnect_count 四键")
	assert.Equal(t, "disconnected", stats["state"])
	assert.Equal(t, 0, stats["reconnect_count"])
	assert.True(t, stats["last_connected"].(time.Time).IsZero())
	assert.False(t, stats["last_disconnect"].(time.Time).IsZero())
}

package middleware

// DEBT-03（v1.51）：Worker 认证中间件校验矩阵测试。
//
// 矩阵（ProvisioningAuth / WorkerAuth × 宽限期 required=false|true）：
//   - 无 token：宽限期放行（WARN）/ 强制模式 401
//   - 错 token：两种模式均 401（显式提供错误凭证比缺失更可疑）
//   - 有效 token：放行 + 上下文注入；:id 与 claims 不一致 → 403
//
// 用httptest + 真实 gin engine，不 mock gin 层。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

type stubTokenService struct {
	provOK    bool
	claims    *rpa.WorkerClaims
	validateErr error
}

func (s *stubTokenService) ValidateProvisioningToken(token string) bool { return s.provOK }
func (s *stubTokenService) IssueWorkerToken(ctx context.Context, workerID string) (*rpa.WorkerTokenPair, error) {
	return nil, nil
}
func (s *stubTokenService) ValidateWorkerToken(ctx context.Context, token string) (*rpa.WorkerClaims, error) {
	if s.validateErr != nil {
		return nil, s.validateErr
	}
	return s.claims, nil
}
func (s *stubTokenService) RevokeWorkerToken(ctx context.Context, workerID string) error { return nil }

func newAuthTestEngine(t *testing.T, svc rpa.WorkerTokenService, required bool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	gr := r.Group("/api/v1/rpa")
	gr.POST("/workers/register", ProvisioningAuth(svc, required), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	gr.POST("/workers/:id/heartbeat", WorkerAuth(svc, required), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"worker_id": c.GetString("worker_id")})
	})
	gr.POST("/workers/progress", WorkerAuth(svc, required), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func doReq(r *gin.Engine, method, path, headerKey, headerVal string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if headerKey != "" {
		req.Header.Set(headerKey, headerVal)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func setupStubSvc(t *testing.T) rpa.WorkerTokenService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rpamodels.Worker{}))
	_ = db // stub 服务不触库，迁移仅证明模型可建
	return &stubTokenService{
		provOK: true,
		claims: &rpa.WorkerClaims{WorkerID: "w-1", WorkerName: "节点一"},
	}
}

func TestProvisioningAuth_Matrix(t *testing.T) {
	svc := setupStubSvc(t)

	// ---- 宽限期 required=false ----
	r := newAuthTestEngine(t, svc, false)
	if w := doReq(r, "POST", "/api/v1/rpa/workers/register", "", ""); w.Code != http.StatusOK {
		t.Fatalf("宽限期无 token 应放行, got %d", w.Code)
	}
	// 错 token → 401（显式提供错误凭证不宽恕）
	badSvc := &stubTokenService{provOK: false}
	rBad := newAuthTestEngine(t, badSvc, false)
	if w := doReq(rBad, "POST", "/api/v1/rpa/workers/register", HeaderProvisioningToken, "wrong"); w.Code != http.StatusUnauthorized {
		t.Fatalf("宽限期错 token 应 401, got %d", w.Code)
	}
	// 有效 token → 放行
	if w := doReq(r, "POST", "/api/v1/rpa/workers/register", HeaderProvisioningToken, "prov-1"); w.Code != http.StatusOK {
		t.Fatalf("有效 token 应放行, got %d", w.Code)
	}

	// ---- 强制 required=true ----
	// 注意：强制模式错 token 用 provOK=false 的 stub（真实服务按 token 值判定）
	rStrict := newAuthTestEngine(t, svc, true)
	badStrict := newAuthTestEngine(t, badSvc, true)
	if w := doReq(badStrict, "POST", "/api/v1/rpa/workers/register", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("强制模式无 token 应 401, got %d", w.Code)
	}
	if w := doReq(badStrict, "POST", "/api/v1/rpa/workers/register", HeaderProvisioningToken, "wrong"); w.Code != http.StatusUnauthorized {
		t.Fatalf("强制模式错 token 应 401, got %d", w.Code)
	}
	if w := doReq(rStrict, "POST", "/api/v1/rpa/workers/register", HeaderProvisioningToken, "prov-1"); w.Code != http.StatusOK {
		t.Fatalf("强制模式有效 token 应放行, got %d", w.Code)
	}
}

func TestWorkerAuth_Matrix(t *testing.T) {
	svc := setupStubSvc(t)

	// ---- 宽限期 required=false ----
	r := newAuthTestEngine(t, svc, false)
	if w := doReq(r, "POST", "/api/v1/rpa/workers/w-1/heartbeat", "", ""); w.Code != http.StatusOK {
		t.Fatalf("宽限期无 token 心跳应放行, got %d", w.Code)
	}

	// ---- 强制 required=true：401 矩阵 ----
	rStrict := newAuthTestEngine(t, svc, true)
	if w := doReq(rStrict, "POST", "/api/v1/rpa/workers/w-1/heartbeat", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("强制无 token 应 401, got %d", w.Code)
	}
	// 错 token（校验失败）→ 401
	expiredSvc := &stubTokenService{validateErr: context.DeadlineExceeded}
	rExpired := newAuthTestEngine(t, expiredSvc, true)
	if w := doReq(rExpired, "POST", "/api/v1/rpa/workers/w-1/heartbeat", HeaderWorkerToken, "wk_stale"); w.Code != http.StatusUnauthorized {
		t.Fatalf("过期/无效 token 应 401, got %d", w.Code)
	}
	if w := doReq(rStrict, "POST", "/api/v1/rpa/workers/w-1/heartbeat", HeaderWorkerToken, "wk_valid"); w.Code != http.StatusOK {
		t.Fatalf("有效 token 应放行, got %d", w.Code)
	}

	// 凭证归属：持 w-1 凭证伪造 w-2 的心跳 → 403
	if w := doReq(rStrict, "POST", "/api/v1/rpa/workers/w-2/heartbeat", HeaderWorkerToken, "wk_valid"); w.Code != http.StatusForbidden {
		t.Fatalf("凭证与目标节点不匹配应 403, got %d", w.Code)
	}
	// 无 :id 参数的路由不做归属校验
	if w := doReq(rStrict, "POST", "/api/v1/rpa/workers/progress", HeaderWorkerToken, "wk_valid"); w.Code != http.StatusOK {
		t.Fatalf("无 :id 路由应仅校验 token, got %d", w.Code)
	}
	// 上下文注入
	w := doReq(rStrict, "POST", "/api/v1/rpa/workers/w-1/heartbeat", HeaderWorkerToken, "wk_valid")
	if !strings.Contains(w.Body.String(), "w-1") {
		t.Fatalf("上下文应注入 worker_id, got %s", w.Body.String())
	}
}

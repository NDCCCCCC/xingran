# RPA Worker API 认证方案 - 待办任务

## 状态：暂缓（搁置）

**创建日期**: 2026-03-10
**优先级**: P2 - 安全加固
**预计工作量**: 2-3 小时

---

## 一、背景

### 当前状态（不安全）

```go
// 当前路由配置 - 完全公开
func SetupPublicWorkerRouter(r *gin.RouterGroup, core *core.Core) {
    // ❌ 任何人都可以注册 Worker
    r.POST("/workers/register", handler.Register)

    // ❌ 任何人都可以发送心跳
    r.POST("/workers/:id/heartbeat", handler.Heartbeat)

    // ❌ 任何人都可以上报进度
    r.POST("/workers/progress", handler.Progress)
}
```

**安全问题：**
- 攻击者可以注册虚假 Worker
- 攻击者可以伪造心跳保持虚假 Worker 在线
- 攻击者可以伪造执行进度数据
- 无法验证请求是否来自合法的 Worker

---

## 二、方案设计

### 认证流程

```
┌─────────────────────────────────────────────────────────────────┐
│                      1. 初始化部署阶段                          │
├─────────────────────────────────────────────────────────────────┤
│  管理员配置 Provisioning Token                                  │
│  - configs/config.yaml: provisioning_tokens                     │
│  - Worker 配置: worker.provisioning_token                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      2. Worker 注册阶段                          │
├─────────────────────────────────────────────────────────────────┤
│  Worker → 后端                                                    │
│  POST /workers/register                                         │
│  Headers: X-Provisioning-Token: rpa-prod-xxx                     │
│  Body: { workerId, name, ... }                                  │
│                                                                  │
│  后端验证 Provisioning Token                                     │
│  ↓ 生成 Worker Token (含过期时间)                               │
│  ↓ 返回 { workerToken: "wk_xxx", expiresAt: "..." }             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      3. 正常运行阶段                              │
├─────────────────────────────────────────────────────────────────┤
│  Worker → 后端                                                    │
│  POST /workers/:id/heartbeat                                    │
│  Headers: X-Worker-Token: wk_xxx                                │
│                                                                  │
│  后端验证 Worker Token                                           │
│  ↓ 检查有效性、过期时间                                           │
│  ↓ 处理请求                                                      │
└─────────────────────────────────────────────────────────────────┘
```

### Token 结构对比

| 类型 | 用途 | 过期时间 | 示例 |
|------|------|---------|------|
| Provisioning Token | 初始注册 | 长期有效（手动轮换） | `rpa-prod-main-2024` |
| Worker Token | API 调用 | 30-90 天（可刷新） | `wk_abc123xyz789` |

---

## 三、实施计划

### 阶段 1: 数据库层

<!-- VERIFY: `internal/core/db/migrations/113_add_worker_token.sql` -->

```sql
-- 添加 Worker Token 字段
ALTER TABLE sys_rpa_workers
ADD COLUMN token VARCHAR(255) UNIQUE,
ADD COLUMN token_expires_at TIMESTAMP,
ADD COLUMN last_token_rotation TIMESTAMP;

-- 创建索引
CREATE INDEX idx_rpa_workers_token ON sys_rpa_workers(token) WHERE token IS NOT NULL;
```

<!-- VERIFY: `internal/models/rpa/worker.go` -->

```go
// Worker 模型添加字段
type Worker struct {
    models.BaseModel
    // ... 现有字段 ...

    // 新增认证字段
    Token             string     `gorm:"size:255;uniqueIndex" json:"token,omitempty"`
    TokenExpiresAt    *time.Time `json:"tokenExpiresAt,omitempty"`
    LastTokenRotation *time.Time `json:"lastTokenRotation,omitempty"`
}
```

---

### 阶段 2: 服务层

<!-- VERIFY: `internal/services/rpa/worker_token_service.go` -->

```go
package rpa

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/xingran-next/xingran-go-backend/internal/core/security"
    rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
    "gorm.io/gorm"
)

// WorkerTokenService Worker Token 服务
type WorkerTokenService interface {
    // 使用 Provisioning Token 注册并生成 Worker Token
    RegisterWithProvisioningToken(ctx context.Context, req *WorkerRegisterRequest, provToken string) (*rpamodels.Worker, *WorkerTokenPair, error)

    // 验证 Worker Token
    ValidateWorkerToken(ctx context.Context, token string) (*WorkerClaims, error)

    // 刷新 Worker Token
    RefreshWorkerToken(ctx context.Context, token string) (*WorkerTokenPair, error)

    // 撤销 Worker Token
    RevokeWorkerToken(ctx context.Context, workerID string) error
}

type workerTokenServiceImpl struct {
    db                *gorm.DB
    jwtManager        *security.JWTManager
    provTokens        map[string]bool // 预共享令牌白名单
}

type WorkerTokenPair struct {
    Token      string    `json:"token"`
    ExpiresAt  time.Time `json:"expiresAt"`
}

type WorkerClaims struct {
    WorkerID    string    `json:"worker_id"`
    WorkerName  string    `json:"worker_name"`
    Capabilities []string  `json:"capabilities"`
    ExpiresAt   time.Time `json:"exp"`
}
```

---

### 阶段 3: 中间件层

#### 文件: `pkg/middleware/worker_auth.go` (新建)
<!-- VERIFY: `pkg/middleware/worker_auth.go` -->
```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// WorkerAuth Worker Token 认证中间件
func WorkerAuth(tokenService rpa.WorkerTokenService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("X-Worker-Token")
        if token == "" {
            response.Error(c, response.ErrUnauthorized, "缺少 Worker 认证令牌")
            c.Abort()
            return
        }

        claims, err := tokenService.ValidateWorkerToken(c.Request.Context(), token)
        if err != nil {
            response.Error(c, response.ErrUnauthorized, "无效的 Worker 令牌")
            c.Abort()
            return
        }

        // 设置 Worker 信息到上下文
        c.Set("worker_id", claims.WorkerID)
        c.Set("worker_name", claims.WorkerName)
        c.Next()
    }
}

// ProvisioningAuth 预共享令牌认证中间件（仅注册接口使用）
func ProvisioningAuth(validTokens map[string]bool) gin.HandlerFunc {
    return func(c *gin.Context) {
        provToken := c.GetHeader("X-Provisioning-Token")
        if provToken == "" {
            response.Error(c, response.ErrUnauthorized, "缺少 Provisioning 令牌")
            c.Abort()
            return
        }

        if !validTokens[provToken] {
            response.Error(c, response.ErrUnauthorized, "无效的 Provisioning 令牌")
            c.Abort()
            return
        }

        c.Next()
    }
}
```

---

### 阶段 4: 路由层修改

<!-- VERIFY: `internal/api/v1/rpa/rpa_router.go` -->

```go
// SetupPublicWorkerRouter 修改后
func SetupPublicWorkerRouter(r *gin.RouterGroup, core *core.Core) {
    services := rpa.NewServiceGroup(core.GetDB(), core.Config, core.NoticeHub, core.Cache, core.SM4Cipher)

    // 从配置加载 Provisioning Tokens
    provTokens := loadProvisioningTokens(core.Config)

    // 注册接口需要 Provisioning Token 认证
    r.Use(middleware.ProvisioningAuth(provTokens))
    r.POST("/workers/register", handler.Register)
}

// SetupWorkerRouter 添加认证
func SetupWorkerRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
    // 所有 Worker 路由需要 Worker Token 认证
    r.Use(middleware.WorkerAuth(services.WorkerTokenService))

    handler := NewWorkerHandler(services.WorkerService, core)

    r.POST("/list", handler.List)
    r.POST("/:id/heartbeat", handler.Heartbeat)
    r.POST("/progress", handler.Progress)
    // ... 其他路由
}
```

---

### 阶段 5: 配置层

<!-- VERIFY: `configs/config.yaml` -->

```yaml
# 新增 RPA Worker 配置
rpa:
  worker:
    # Provisioning Tokens (预共享令牌白名单)
    provisioning_tokens:
      - "rpa-prod-main-2024"      # 生产环境主令牌
      - "rpa-staging-test-2024"   # 测试环境令牌

    # Worker Token 配置
    token:
      expire_days: 90            # Worker Token 有效期（天）
      rotation_days: 30           # 自动轮换周期（天）
```

<!-- VERIFY: `rpa-worker/configs/config.yaml` -->

```yaml
worker:
  # Worker 的 Provisioning Token（部署时配置）
  provisioning_token: "rpa-prod-main-2024"
```

---

### 阶段 6: Worker 客户端

<!-- VERIFY: `rpa-worker/internal/communication/api_client.go` -->

```go
type APIClient struct {
    baseURL         string
    workerToken      string        // 新增：存储 Worker Token
    provisioningToken string        // 新增：Provisioning Token
    httpClient      *http.Client
    logger           logger.Logger
}

func NewAPIClient(cfg *config.BackendConfig, provToken string, log logger.Logger) *APIClient {
    return &APIClient{
        baseURL:         cfg.BaseURL,
        provisioningToken: provToken,
        httpClient:      &http.Client{Timeout: cfg.Timeout},
        logger:           log,
    }
}

// Register 使用 Provisioning Token 注册
func (c *APIClient) Register(ctx context.Context, req *types.WorkerRegisterRequest) (*types.WorkerRegisterResponse, error) {
    var resp types.APIResponse

    reqBody, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/workers/register", bytes.NewReader(reqBody))

    // 添加 Provisioning Token 头
    httpReq.Header.Set("X-Provisioning-Token", c.provisioningToken)
    httpReq.Header.Set("Content-Type", "application/json")

    // ... 发送请求 ...

    // 解析响应，保存 Worker Token
    var registerResp types.WorkerRegisterResponse
    json.Unmarshal(resp.Data, &registerResp)

    // 存储 Worker Token
    c.workerToken = registerResp.WorkerToken

    return &registerResp, nil
}

// 私有方法：添加 Worker Token 头
func (c *APIClient) addAuthHeaders(req *http.Request) {
    if c.workerToken != "" {
        req.Header.Set("X-Worker-Token", c.workerToken)
    }
}
```

---

## 四、文件清单

### 新建文件 (4个)

| 文件 | 用途 |
|------|------|
| `internal/core/db/migrations/113_add_worker_token.sql` <!-- VERIFY: file does not exist --> | 数据库迁移 |
| `internal/services/rpa/worker_token_service.go` <!-- VERIFY: file does not exist --> | Token 服务 |
| `pkg/middleware/worker_auth.go` <!-- VERIFY: file does not exist --> | 认证中间件 |
| `docs/RPA-Worker-Token管理界面.md` <!-- VERIFY: file does not exist --> | 前端管理文档 |

### 修改文件 (6个)

| 文件 | 修改内容 |
|------|---------|
| `internal/models/rpa/worker.go` <!-- VERIFY: missing token fields --> | 添加 token 字段 |
| `internal/services/rpa/service_group.go` <!-- VERIFY: file does not exist --> | 注册 WorkerTokenService |
| `internal/api/v1/rpa/worker_handler.go` | 修改注册返回 token |
| `internal/api/v1/rpa/rpa_router.go` | 添加认证中间件 |
| `configs/config.yaml` | 添加 provisioning_tokens |
| `rpa-worker/internal/communication/api_client.go` | Token 管理 |
| `rpa-worker/configs/config.yaml` | 添加 provisioning_token |

---

## 五、依赖关系

```
WorkerAuthMiddleware
    │
    ├─→ WorkerTokenService.ValidateWorkerToken()
    │       │
    │       ├─→ 查询数据库验证 token
    │       └─→ 检查过期时间
    │
    └─→ setWorkerContext() 设置上下文

ProvisioningAuthMiddleware
    │
    └─→ 检查配置中的白名单
```

---

## 六、测试计划

### 单元测试

```go
// internal/services/rpa/worker_token_service_test.go
func TestWorkerTokenService_RegisterWithProvisioningToken(t *testing.T)
func TestWorkerTokenService_ValidateWorkerToken(t *testing.T)
func TestWorkerTokenService_RefreshWorkerToken(t *testing.T)
func TestWorkerTokenService_RevokeWorkerToken(t *testing.T)
```

### 集成测试

```bash
# 1. 测试无 Token 访问 - 应 401
curl -X POST http://backend/api/v1/rpa/workers/register

# 2. 测试错误 Provisioning Token - 应 401
curl -X POST http://backend/api/v1/rpa/workers/register \
  -H "X-Provisioning-Token: wrong-token"

# 3. 测试正确注册 - 应成功并返回 token
curl -X POST http://backend/api/v1/rpa/workers/register \
  -H "X-Provisioning-Token: rpa-prod-main-2024" \
  -d '{"workerId":"test-1","name":"Test"}'

# 4. 测试使用 Worker Token 访问
curl -X POST http://backend/api/v1/rpa/workers/test-1/heartbeat \
  -H "X-Worker-Token: wk_abc123"
```

---

## 七、安全考虑

### Token 安全性

| 安全措施 | 实现方式 |
|---------|---------|
| Token 传输 | HTTPS + HTTP 头 |
| Token 存储 | 数据库 + 定期轮换 |
| Token 过期 | 30-90 天自动过期 |
| Token 撤销 | 支持手动撤销 |
| 重放攻击 | Token 包含过期时间 + 可选 nonce |

### 配置安全

```yaml
# 生产环境配置示例
rpa:
  worker:
    provisioning_tokens:
      - "rpa-prod-main-2024"      # 主令牌
      - "rpa-prod-backup-2024"    # 备用令牌（轮换时使用）
    token:
      expire_days: 90
      rotation_days: 30
```

---

## 八、优先级与时机

### 优先级: P2 - 安全加固

**建议实施时机：**

1. ✅ **当前阶段**: 完成 Worker 核心功能测试
2. ⏳ **下一阶段**: 实施此 API Key 认证方案
3. ⏳ **生产前**: 完成安全审计和渗透测试

### 阻塞因素

- [ ] Worker 核心功能稳定运行
- [ ] Redis Stream 消费组问题解决
- [ ] 前端管理界面开发完成

---

## 九、参考资料

### 社区最佳实践

- **GitHub**: API Key + Webhook 认证
- **Stripe**: API Key 发布与轮换机制
- **AWS IAM**: 访问密钥与临时凭证
- **Kubernetes**: Service Account Token

### 相关文档

- `docs/安全和认证设计（国密）.md` - 现有 JWT 认证机制
- `internal/core/security/jwt.go` - JWT 实现
- `pkg/middleware/auth.go` - 认证中间件模式

---

## 十、变更日志

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-03-10 | 1.0 | 初始版本，暂存为待办任务 |

---

**注意**: 本文档为待办任务，实施前需要再次评审技术方案。

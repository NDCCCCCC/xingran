---
milestone: v1.32
status: defined
defined: 2026-09-09
---

# Requirements: XingRan-Next — Milestone v1.32 V132 审计驱动的安全与可靠性收尾 (Audit-Driven Security & Reliability)

**Defined:** 2026-09-09
**Core Value:** 基于 2026-09-09 全量后端审计报告（`.planning/reviews/20260909-backend-audit.md`）的 18 项 P0/P1 风险 + 4 项 Phase 104/107/108 MUST-FIX + 6 项新发现并发风险，按用户决策（范围=P0+P1+回归守护；TLS 选项=环境变量；回归守护=Phase 109 前置）分 5 个 phase 收尾：Phase 109 回归守护前置 → Phase 110 P0 安全 TLS 环境变量化 → Phase 111 P0 并发裸 goroutine 守护 → Phase 112 P1 handler 收敛补丁 → Phase 113 部署文档同步。

**输入来源:**
- `.planning/reviews/20260909-backend-audit.md`（2026-09-09 全量后端审计报告）
- `.planning/PROJECT.md` v1.32 段（D-01~D-06 锁定决策）
- 历史审计：`.planning/reviews/20260612-backend-code-review.md`（v1.31 对比基线）

**锁定决策 (v1.32 init):**

- **D-01 范围**: P0 安全 + P0 并发 + P1 handler 收敛 + 8 项回归守护；P2 清理（panic 改 error / 硬删除改软删除）不在本期
- **D-02 TLS 选项实现**: 全部走环境变量（沿用现有 LDAP_TLS_INSECURE_SKIP_VERIFY 模式，新增 REDIS_TLS_INSECURE_SKIP_VERIFY / WS_ALLOW_ALL_ORIGINS 等），默认 false（严格校验），内网部署可显式置 true 兼容自签证书
- **D-03 回归纪律**: 8 项回归守护作为 Phase 109 前置独立 phase 落地，先测试后修复；每个修复必须有对应回归测试已存在（红→绿 路径）
- **D-04 七 gate 不倒退**: go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage 全程保持绿；新增 8 项 invariants 测试纳入 diff coverage gate
- **D-05 范围外**: P2 清理（panic 改 error / 硬删除改软删除）；agent 裸 c.JSON（已锁定为有意设计）；operlog exclude_paths 继续挂账
- **D-06 Phase 编号**: 从 Phase 109 续编

---

## v1.32 Requirements

### GUARD — 回归守护前置（Phase 109 独立）

> 8 项回归守护测试先于修复落地，作为后续修复 phase 的"安全网"——确保红→绿路径清晰可验证。

- [ ] **GUARD-01**: `pkg/cache/redis_test.go` 新增 `TestRedis_TLSConfig_NotInsecureByDefault` —— 当 `config.TLS=true` 且未设置 `REDIS_TLS_INSECURE_SKIP_VERIFY` 时，`tls.Config.InsecureSkipVerify == false`（Phase 110 红→绿基线）
- [ ] **GUARD-02**: `internal/core/security/ad_authenticator_test.go` 新增 `TestADAAuthenticator_TLSConfig_StrictByDefault` —— `dialConnection` 默认构造的 `tls.Config.InsecureSkipVerify == false`，除非显式设置（Phase 110 红→绿基线）
- [ ] **GUARD-03**: `cmd/main_test.go` 新增 `TestMain_AllowedOrigins_FromConfig_NotWildcard` —— 当 `server.allowed_origins` 已配置时，启动入口不会传入 `[]string{"*"}`；空配置触发 fail-fast（Phase 110 红→绿基线）
- [ ] **GUARD-04**: `pkg/response/handler_helpers_test.go` 新增 `TestHandleGetByID_Returns404_NotBadRequest` —— 当 getter 返回 error 时，HTTP status = 404 而非 400（Phase 112 红→绿基线）
- [ ] **GUARD-05**: `internal/api/v1/monitor/login_log_handler_test.go` 新增 `TestLoginLog_Clean_NilCore_DoesNotPanic` —— 当 `h.core == nil` 或 `h.core.OperLogService == nil` 时，`Clean` 优雅降级不 panic（Phase 112 红→绿基线）
- [ ] **GUARD-06**: `internal/api/v1/operations/{building,floor,workstation}_handler_test.go` 新增 `TestStatistics_ErrorBody_DoesNotLeakSQL` —— 当底层 service 返回 SQL 错误时，响应 body 不包含 "SELECT/UPDATE/INSERT/DELETE" 等关键字（Phase 112 红→绿基线）
- [ ] **GUARD-07**: `internal/services/oper_log_service_test.go` 新增 `TestOperLog_Async_DoesNotPanicOnDBError` —— DB 写入 panic 时 goroutine 自我 recover，进程不崩溃（Phase 111 红→绿基线）
- [ ] **GUARD-08**: `internal/core/captcha_test.go` 新增 `TestCaptcha_Increment_Failure_FailsClosed` —— 当 Redis Increment 失败时，验证码校验 fail-closed（拒绝通过）而非 fail-open（Phase 111 红→绿基线）

### TLS — TLS 选项环境变量化（Phase 110）

> D-02 锁定：所有 TLS 跳过校验走环境变量，默认 false（严格校验），内网可显式置 true 兼容。

- [ ] **TLS-01**: `pkg/cache/redis.go` — `NewRedisCache` 当 `config.TLS=true` 时，读取 `REDIS_TLS_INSECURE_SKIP_VERIFY`（默认 false）；true 时置 `InsecureSkipVerify=true` 并记录一次性 SECURITY warn（与 LDAP 一致模式）
- [ ] **TLS-02**: `internal/core/security/ad_authenticator.go:181-183` — `dialConnection` 移除硬编码 `InsecureSkipVerify: true`，改为读取 `AD_AUTH_TLS_INSECURE_SKIP_VERIFY`（默认 false）；true 时 warn 一次
- [ ] **TLS-03**: `internal/services/email_sender_service.go` — 当前硬编码 `InsecureSkipVerify: false` 已安全，但增加 env 支持（`EMAIL_TLS_INSECURE_SKIP_VERIFY`，默认 false），与 LDAP/Redis 一致模式
- [ ] **TLS-04**: `cmd/main.go` — 移除 `allowedOrigins := []string{"*"}` 硬编码；从 `config.Server.AllowedOrigins` 读取；空配置或仅 `*` 时 fail-fast（生产环境强校验）
- [ ] **TLS-05**: `pkg/middleware/cors.go` — 允许 `*` 通配配置项，但新增 `WS_ALLOW_ALL_ORIGINS` 环境变量覆盖（默认 false = 严格白名单）
- [ ] **TLS-06**: 文档同步：`docs/deployment/secret-management.md` 新增"MUST SET in production" + "内网兼容"两段说明

### GOR — 裸 goroutine 守护（Phase 111）

> D-03 锁定：所有裸 goroutine 必须有 `defer recover()` + detached context（HTTP ctx 启动后即取消，不适合异步任务）。

- [ ] **GOR-01**: `internal/services/oper_log_service.go:67,140` — `RecordAsync` 的 `go func()` 加 `defer recover()` + 错误日志；panic 时记录 SECURITY 级日志而非静默
- [ ] **GOR-02**: `internal/api/v1/system/ad_dept_sync_handler.go:99` — `SyncDeptStructureToAD` 启动异步时使用 `context.WithTimeout(context.Background(), ADSyncTimeout)` + `defer recover()`；不使用 `c.Request.Context()`（HTTP 返回后已取消）
- [ ] **GOR-03**: `internal/agent/server/connection_manager.go:186,196` — `handleReconnect` / `cleanupConnection` 启动的 goroutine 加 `defer recover()` + 状态日志
- [ ] **GOR-04**: `internal/api/v1/auth.go:598` — 登录日志异步写入的 `go func()` 加 `defer recover()`；失败时记录 warn 而非 panic 进程崩溃

### CAP — Captcha Increment Fail-Closed（Phase 111）

- [ ] **CAP-01**: `internal/core/captcha.go:379,439,445` — `s.cache.Increment(ctx, key, 1)` 失败时记录 SECURITY warn 日志；推荐改为同步 DB 兜底计数（无 DB fallback 时强制 fail-closed — 拒绝通过而非放行）

### HANDLER — Phase 104 收敛补丁（Phase 112）

> Phase 104 已统一 14 个 operations handler 的 CRUD 路径，但 Statistics / Search*Options 端点仍泄漏 `err.Error()`。

- [ ] **HANDLER-01**: `internal/api/v1/operations/building_handler.go:40,55` — `Statistics` / `SearchBuildingOptions` 改用 `HandleServiceError`；删除 `response.Error(c, http.StatusInternalServerError, err.Error())` 直接泄漏
- [ ] **HANDLER-02**: `internal/api/v1/operations/floor_handler.go:36,51` — `Statistics` / `SearchFloorOptions` 同样收敛；`List:102` 的 `apperrors.InternalServerErrorWithMsg("查询失败")` 丢弃 err 改为 `HandleServiceError`
- [ ] **HANDLER-03**: `internal/api/v1/operations/workstation_handler.go:58,83,101` — `Statistics` / `GetWorkstationDeptOptions` / `SearchWorkstationOptions` 同样收敛
- [ ] **HANDLER-04**: `internal/api/v1/monitor/login_log_handler.go:106` — `Clean` 加 `h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil` 三重 nil guard（与 OperLogHandler.Clean 对称）
- [ ] **HANDLER-05**: `pkg/response/handler_helpers.go:62` — `HandleGetByID` 改用 `apperrors.New(http.StatusNotFound, ErrNotFound, notFoundMessage)` 或专门的 `ErrNotFound` 常量；当前 int 传入 `toAppError` 把 404 当业务码处理 → HTTP 400

### DOC — 部署文档同步（Phase 113）

- [ ] **DOC-01**: `docs/deployment/secret-management.md` 新增"V132 TLS/Origin 选项"章节：
  - `LDAP_TLS_INSECURE_SKIP_VERIFY`（沿用）
  - `AD_AUTH_TLS_INSECURE_SKIP_VERIFY`（新增）
  - `AD_LEGACY_AES_KEY`（沿用 + MUST SET 提示）
  - `REDIS_TLS_INSECURE_SKIP_VERIFY`（新增）
  - `EMAIL_TLS_INSECURE_SKIP_VERIFY`（新增）
  - `WS_ALLOW_ALL_ORIGINS`（新增）
  - 内网部署兼容性说明 + 生产环境 MUST NOT SET 警告

---

## 范围外（锁定 D-05）

- **P2 清理**: `connection_pool.go:87,105` 负引用计数 panic → 改 error 返回；`ad_ldap_client.go:33` / `vdi/config.go:28` / `column_config_service.go:159,165` panic → error；`addomain/sync.go:618` / `vdi/vm_service_impl.go:322` 硬删除 → 软删除
- **agent 裸 c.JSON**: 已锁定为有意设计，禁改
- **operlog exclude_paths**: 继续挂账
- **菜单类 C 权限继承**: PARTIAL 状态接受，不在本期修复

---

## 回归纪律

每个修复提交必须：
1. 引用对应的 GUARD-N 测试 ID（Phase 109 前置已存在）
2. 修复 commit 中明确说明"红→绿"路径
3. 附七 gate 验证（go build / go test / coverage / lint / type-check）
4. 新增的 invariants 测试纳入 diff coverage gate

---

## 进度追踪

| Phase | 标题 | Requirements | Status |
|-------|------|--------------|--------|
| 109 | 回归守护前置 | GUARD-01..08 | ⏳ PENDING |
| 110 | P0 安全 TLS 环境变量化 | TLS-01..06 | ⏳ PENDING |
| 111 | P0 并发裸 goroutine 守护 | GOR-01..04 + CAP-01 | ⏳ PENDING |
| 112 | P1 handler 收敛补丁 | HANDLER-01..05 | ⏳ PENDING |
| 113 | 部署文档同步 | DOC-01 | ⏳ PENDING |

**Total:** 5 phases / 22 requirements
# 🔍 XingRan-Next 后端代码完整审计报告（v1.31 SHIPPED 后）

**报告日期**：2026-09-09
**审查范围**：`internal/` + `pkg/` 全部 Go 源码（1,068 个文件）
**基线版本**：v1.31 SHIPPED（2026-09-08，commit `4a05ca5`）
**基线对比**：`.planning/reviews/20260612-backend-code-review.md`（2026-06-12）
**审查方式**：6 个并行 Explore agent + 抽样验证 + 全文 grep 排查
**审查维度**：惯用性 / 并发 / 错误处理 / 性能 / 安全 / Phase 104/107/108 新代码质量

---

## 📊 总体评估

| 维度 | v1.31 评分 | 较 06-12 评分 | 趋势 |
|------|-----------|----------------|------|
| **架构清晰度** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ↑ wire contract 收敛 |
| **惯用性** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ↑ handler_helpers 统一 |
| **并发安全** | ⭐⭐⭐⭐ | ⭐⭐ | ↑↑ P0 并发全修复，但新代码有回归 |
| **错误处理** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ↑ apperrors 落地 |
| **性能** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ↑ cache 收敛 |
| **安全性** | ⭐⭐⭐ | ⭐⭐ | ↑↑ P0 安全 6/8 修复 + 2 PARTIAL |
| **测试覆盖** | ⭐⭐⭐ | ⭐⭐ | ↑ skip test 收敛 |
| **新代码质量** | ⭐⭐⭐ | — | 初次评估，4 个 MUST-FIX |

**总体结论**：v1.31 大幅修复了 06-12 报告中的 P0/P1 主要问题（30+ 项中 22 项 FIXED，2 项 PARTIAL）。**但 Phase 104/107/108 引入的新代码和遗留的次要风险需要第二轮收尾**。建议按本报告路线图分批修复 P0 与 P1 残留。

---

## 🆚 06-12 vs v1.31 对比汇总

| 类别 | 总数 | FIXED | PARTIAL | NOT FIXED |
|------|------|-------|---------|-----------|
| P0 安全 | 8 | 6 | 2 | 0 |
| P0 并发 | 7 | 7 | 0 | 0 |
| P0 SQL 注入/越权 | 4 | 4 | 0 | 0 |
| P1 | 8 | 7 | 1 | 0 |
| **小计** | **27** | **24** | **3** | **0** |
| Phase 104/107/108 新发现 | 4 | — | — | 4（MUST-FIX） |
| 并发模式总体审计新发现 | 10 | — | 4 (P0) | 6 (P1/P2) |

---

## 🚨 P0 严重问题 — 现状与残留（按优先级）

### P0-S1: WebSocket CORS `cmd/main.go:104` 硬编码 `*` ⚠️ **回归风险**

**状态**：PARTIAL（应用层 CheckOrigin 严格，但启动入口硬编码允许所有来源）

**位置**：
- `internal/api/v1/ws_notice_handler.go:31-86` — CheckOrigin 有完整的白名单校验
- `cmd/main.go:104` — `allowedOrigins := []string{"*"}`（硬编码）

**残留风险**：生产部署时即使 CheckOrigin 严格，main.go 启动时仍传入 `*`，等效于"开发模式"。**生产事故面**：CSRF 盗用 WebSocket token。

**修复建议**：
```go
// cmd/main.go:104
allowedOrigins := config.Server.AllowedOrigins  // 从配置读取
if len(allowedOrigins) == 0 {
    applogger.Fatalf("server.allowed_origins must be configured in production")
}
```

---

### P0-S2: AD Legacy AES 密钥硬编码 fallback ⚠️ **环境依赖**

**状态**：PARTIAL（env 优先，仍保留 fallback 硬编码）

**位置**：`internal/services/addomain/utils.go:112`
```go
const legacyAESKeyHardcoded = "xingran-ad-domain-key-16"
```

**修复记录**：F-02 实现 `AD_LEGACY_AES_KEY` env 优先；无 env 时回退硬编码并 warn 一次。

**残留风险**：二进制反汇编仍可提取该密钥。**生产部署必须设置 `AD_LEGACY_AES_KEY` 环境变量**，否则数据库所有 AD 密码可被反编译解出。

**修复建议**：
1. 在 `docs/deployment/secret-management.md` 顶部加显眼的"MUST SET"提示
2. 生产部署 checklist 增加此 env 校验项

---

### P0-S3: `ad_authenticator.go:182` `InsecureSkipVerify: true` 仍有遗留 ⚠️ **P0 安全**

**状态**：NOT FIXED（2026-06-12 报告后此分支未迁移到 env 控制）

**位置**：`internal/core/security/ad_authenticator.go:181-183`
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: true, // TODO: 生产环境应配置证书
}
```

**残留风险**：与 `ldap_client.go`（已修复 env 控制）并列的另一条 LDAP 路径仍硬编码跳过 TLS 校验。AD Authenticator 用于 JWT/SSO 登录认证，**影响登录安全**。

**修复建议**：
1. 同步迁移到 env 控制（与 `ldap_client.go:99` 一致）
2. 添加回归测试：`TestADA authenticator_TLSConfig_NotInsecureByDefault`

---

### P0-S4: `pkg/cache/redis.go:35` Redis TLS `InsecureSkipVerify: true` ⚠️ **P0 安全**

**状态**：NOT FIXED（注释提到"待生产化时统一替换为受信 CA 池"）

**位置**：`pkg/cache/redis.go:32-35`
```go
if config.TLS {
    // 托管 Redis (Upstash 等) 强制 TLS;InsecureSkipVerify 与现有 LDAPS 路径一致,
    // 待生产化时统一替换为受信 CA 池 — 单独跟踪,不在本 quick task scope。
    tlsCfg = &tls.Config{InsecureSkipVerify: true}
}
```

**残留风险**：托管 Redis 流量可被 MITM。

**修复建议**：
1. 改为从 config 或 env 读取 CA 证书池
2. 与 LDAP 一致：env 变量 `REDIS_TLS_INSECURE_SKIP_VERIFY`（默认 false）

---

### P0-C1: `OperLogService` 异步 goroutine 无 recover + 静默吞错 ⚠️ **P0 并发/审计可靠性**

**状态**：NOT FIXED（2026-06-12 未识别）

**位置**：
- `internal/services/oper_log_service.go:67` — `_ = db.Create(operLog)` 在 goroutine 中
- `internal/services/oper_log_service.go:140` — 同模式

**残留风险**：
- panic 会导致审计日志丢失，且无监控信号
- DB 写入失败完全静默，运维无法感知

**修复建议**：参考 `user_handler.go:200` 的 AD 同步模式，加 `defer recover()` + 错误日志记录。

---

### P0-C2: AD 部门同步 handler 裸 goroutine + ctx 取消 ⚠️ **P0 并发**

**状态**：NOT FIXED

**位置**：`internal/api/v1/system/ad_dept_sync_handler.go:99-100`
```go
go func() {
    // 缺少 panic recover
    // c.Request.Context() 在 HTTP 返回后已取消，导致同步失败
    h.syncService.SyncDeptStructureToAD(c.Request.Context(), ...)
}()
```

**残留风险**：
1. goroutine panic 会导致进程级崩溃（取决于 GOMAXPROCS）
2. 请求 ctx 在 goroutine 启动后立即取消，AD 同步几乎必然失败

**修复建议**：
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[AD-DEPT-SYNC] panic recovered: %v", r)
        }
    }()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    h.syncService.SyncDeptStructureToAD(ctx, ...)
}()
```

---

### P0-C3: connection_manager 重连裸 goroutine 无 recover ⚠️ **P0 并发**

**状态**：NOT FIXED

**位置**：`internal/agent/server/connection_manager.go:186, 196`
```go
go func() {
    // 缺 panic recover
    cm.handleReconnect(client)
}()
```

**残留风险**：Agent 端连接管理器 Reconnect panic 会导致整个连接管理 goroutine 链断裂。

---

### P0-C4: captcha.go Increment 失败静默吞错 ⚠️ **P0 安全**

**状态**：NOT FIXED

**位置**：`internal/core/captcha.go:379, 439, 445`
```go
_, _ = s.cache.Increment(ctx, key, 1)
```

**残留风险**：验证码次数递增失败时，暴力破解防护静默失效。

**修复建议**：
1. 返回 error，至少记录 warn 日志
2. 或在 Increment 失败时强制走 DB 兜底

---

## 🔥 P1 重要问题 — 新发现

### P1-N1: `login_log_handler.go:106` nil 解引用 ⚠️ **P1**

**位置**：`internal/api/v1/monitor/login_log_handler.go:106`
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "登录日志", operlog.OperTypeClean)
```

**问题**：`h.core.OperLogService` 无 nil guard；`OperLogHandler.Clean`（同包 sister 文件）有 guard。

**修复建议**：加 `if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil` 守卫。

---

### P1-N2: `HandleGetByID` HTTP status 反转（400 vs 404）⚠️ **P1（QUIRK）**

**位置**：`pkg/response/handler_helpers.go:62`
```go
Error(c, http.StatusNotFound, notFoundMessage)
// Error 接 int → toAppError 把 int 视作 Code 而非 HTTPStatus → 实际 HTTPStatus = 400
```

**现状**：当前零调用方（grep 全 internal/ 无 hit），但作为 public API 的一部分会误导后续调用方。

**修复建议**：改用 `apperrors.New(http.StatusNotFound, "ERR_NOT_FOUND", notFoundMessage)` 或专门的 `ErrNotFound` 常量。

---

### P1-N3: operations handlers `err.Error()` 泄漏（Phase 104 收敛不彻底）⚠️ **P1**

**状态**：Phase 104 已收敛 14 个 handler 的 CRUD 部分，但 `Statistics` / `Search*Options` 仍漏

**位置**：
- `internal/api/v1/operations/building_handler.go:40, 55`
- `internal/api/v1/operations/floor_handler.go:36, 51`
- `internal/api/v1/operations/workstation_handler.go:58, 83, 101`

**问题**：均直接 `response.Error(c, http.StatusInternalServerError, err.Error())`，把 SQL/基础设施错误原文返回前端。

**修复建议**：统一替换为 `HandleServiceError(c, err, "查询统计数据")`。

---

### P1-N4: 权限继承（菜单类 C）仍残留 ⚠️ **P1**

**状态**：PARTIAL（已限制为按钮类 F，但继承逻辑本身仍存在）

**位置**：`pkg/middleware/permission.go:162-197`

**修复记录**：已限制为 `menu_type='F'`（按钮）子权限可继承父菜单权限，菜单类（C）已排除。

**残留风险**：按钮类继承 → 父菜单访问。在某些精细化数据权限场景仍构成越权读取。

**修复建议**：v1.32 阶段考虑完全移除继承机制，改为严格精确权限匹配。

---

## 📋 已修复项汇总（v1.31 期间）

### ✅ P0 安全 (6/8 FIXED)

| 编号 | 问题 | 修复 commit / 位置 |
|------|------|---------------------|
| P0-01 | LDAP InsecureSkipVerify 硬编码 | F-01: `ldap_client.go:99` env 控制 |
| P0-03 | decryptPassword 明文回退 | F-03: `utils.go:88-98` 返回空串 |
| P0-04 | JWT 默认密钥硬编码 | F-04: `config.go:494` 默认空 + `jwt.go:52` 强校验 |
| P0-05 | 解密明文写入日志 | F-05: `request_decryption.go:174-193` 仅记录元数据 |
| P0-06 | 错误响应泄露内部细节 | F-06: `request_decryption.go:160` 通用错误消息 |
| P0-08 | InvokeTarget 无白名单 | F-08: `job_service.go:125-136` IsTaskRegistered |

### ✅ P0 并发 (7/7 FIXED)

| 编号 | 问题 | 修复 commit / 位置 |
|------|------|---------------------|
| P0-9 | GetOrSet 裸 goroutine | `data_cache_service.go:119-127` 改为同步 Set |
| P0-10 | AD 同步 goroutine 无保护 | `user_handler.go:200-236` sync.Map dedupe + recover + 重试 |
| P0-11 | generateWorkOrderNo Count+1 竞态 | `workorder_tasks.go:139-144` UUID v4 后缀 |
| P0-12 | TotalGenerated lost update | `periodic.go:419-427` UPDATE...RETURNING |
| P0-13 | Register 无缓冲 channel | `notice_hub.go:77-79,177-180` 缓冲 256 |
| P0-14 | GetConnection 释放重取竞态 | `connection_pool.go:255-262` deviceLock 内原子 +1 |
| P0-18 | InvalidateRoleCache 模式键 | `role_cache_impl.go:173-186` InvalidatePattern 分流 |

### ✅ P0 SQL 注入 / 越权 (4/4 FIXED)

| 编号 | 问题 | 修复 commit / 位置 |
|------|------|---------------------|
| P0-19 | BatchUpdatePositions SQL 注入 | F-19: `workstation_service.go:372-464` GORM 参数化 |
| P0-20 | excel_service 越界 panic | F-20: `excel_service.go:349-353` 循环不变量 |
| P0-21 | dashboard 模板 scope 越权 | F-21: `dashboard_service.go:389-401` 默认 Global |
| P0-22 | Unscoped 软删用户接管 | F-22: `user_ou_service.go:38-49` 拒绝自动恢复 |

### ✅ P1 (7/8 FIXED)

| 编号 | 问题 | 修复 commit / 位置 |
|------|------|---------------------|
| P1-1 | JWT alg confusion | `sm2_jwt.go:232-245` 严格 header.Alg 校验 |
| P1-2 | timestamp 窗口 ±300s | `request_encryption.go:36` 60s |
| P1-3 | nonce cleanup 未调用 | `nonce_storage.go:28-66` ticker 启动清理 |
| P1-5 | PBKDF2 1000 轮 | `password.go:33-36` 600000 轮 |
| P1-6 | 取模密码偏置 | `password.go:168-178` crypto/rand.Int |
| P1-7 | Excel 仅校验后缀 | `excel_handler.go:107-111` magic bytes 校验 |
| P1-8 | 加密配置需重启 | `config_service.go:144-149` 回调热加载 |

---

## 🆕 新发现风险点（需关注）

| 等级 | 位置 | 描述 |
|------|------|------|
| **P0** | `pkg/cache/redis.go:35` | Redis TLS InsecureSkipVerify 硬编码 |
| **P0** | `internal/core/security/ad_authenticator.go:182` | ADA authenticator TLS 硬编码 InsecureSkipVerify |
| **P0** | `internal/services/oper_log_service.go:67,140` | OperLog 异步 goroutine 无 recover + 吞错 |
| **P0** | `internal/api/v1/system/ad_dept_sync_handler.go:99` | AD dept sync 裸 goroutine + ctx 取消 |
| **P0** | `internal/agent/server/connection_manager.go:186,196` | Agent 连接管理重连无 recover |
| **P0** | `internal/core/captcha.go:379,439,445` | 验证码 increment 静默吞错 |
| **P1** | `internal/api/v1/monitor/login_log_handler.go:106` | nil 解引用风险（OperLogService 无 guard） |
| **P1** | `pkg/response/handler_helpers.go:62` | HandleGetByID HTTP 状态 400 vs 404（QUIRK） |
| **P1** | `internal/api/v1/operations/{building,floor,workstation}_handler.go` | Statistics/Search*Options 端点 err.Error() 泄漏 |
| **P1** | `internal/services/portwrite/port_write_service.go:592` | 端口写刷新裸 goroutine 无 recover |
| **P1** | `internal/device/connection_pool.go:87,105` | 负引用计数 panic（应返回 error） |
| **P1** | `internal/api/v1/auth.go:598` | 登录日志裸 goroutine 无 recover |
| **P2** | `internal/services/addomain/sync.go:618` | Unscoped().Delete() 硬删除 AD 组成员 |
| **P2** | `internal/services/vdi/vm_service_impl.go:322` | Unscoped().Delete() 硬删除 VM 记录 |
| **P2** | `internal/services/ad_ldap_client.go:33` | 配置加载 panic（应返回 error） |
| **P2** | `internal/services/vdi/config.go:28` | VDI 配置加载 panic |
| **P2** | `internal/services/system/column_config_service.go:159,165` | 启动 panic（fail-fast 但方式不优雅） |
| **P2** | `internal/core/core.go:767-774` | OperLog fire-and-forget 依赖 100ms sleep |

---

## 📌 待修复的 Skip Tests（Phase 108 残留）

| 文件 | 行 | 原因 | 建议 |
|------|----|------|------|
| `models/rpa/rpa_model_methods_test.go:108` | BUG-TEMPLATE-GET-TAGS | 应修复后取消 skip |
| `pkg/errors/codes_80_04_test.go:189,210,242` | `t.Skip("skipped")` 无说明 | 应补充注释或删除 |
| `pkg/errors/errors_80_04_test.go:189,210,242` | 同上 | 同上 |
| `api/v1/auth_test.go:45` | 需要完整 Core | 应用 Phase 104 创建的 Core init helpers |
| `services/operations/building_service_test.go:10,14` | placeholder | Phase 104 后应能填 mock |

---

## 🎯 推荐修复优先级路线图（v1.32 / v1.33）

### 🔥 紧急修复（1 周内，P0 安全）

1. **`pkg/cache/redis.go:35`** — Redis TLS InsecureSkipVerify 改为 env 控制（与 LDAP 一致）
2. **`internal/core/security/ad_authenticator.go:182`** — 同步迁移到 env 控制；删除 TODO
3. **`cmd/main.go:104`** — WebSocket allowedOrigins 改为配置读取；启动时强校验非空
4. **`internal/core/captcha.go:379,439,445`** — Increment 失败必须记录 warn 或 fail-closed

### 🔧 并发与可靠性修复（2 周内，P0 并发）

5. **`oper_log_service.go:67,140`** — 加 `defer recover()` + 错误日志
6. **`ad_dept_sync_handler.go:99`** — 用 `context.Background()` + timeout + panic recover
7. **`agent/server/connection_manager.go:186,196`** — Reconnect goroutine 加 recover
8. **`api/v1/auth.go:598`** — 登录日志裸 goroutine 加 recover

### 🛠 重构与一致性（1 月内，P1）

9. **`operations/{building,floor,workstation}_handler.go`** — Statistics/Search*Options 端点全部迁到 `HandleServiceError`（Phase 104 收敛补丁）
10. **`login_log_handler.go:106`** — 加 nil guard
11. **`pkg/response/handler_helpers.go:62`** — 修复 HandleGetByID HTTP 状态（QUIRK）
12. **`permission.go:162-197`** — 评估完全移除子→父权限继承

### 🧹 清理与文档（持续，P2）

13. **`ad_ldap_client.go:33` / `vdi/config.go:28` / `column_config_service.go:159,165`** — 配置加载 panic 改返回 error（启动鲁棒性）
14. **`device/connection_pool.go:87,105`** — 负引用计数应返回 error 而非 panic（状态机 bug 信号）
15. **`addomain/sync.go:618` / `vdi/vm_service_impl.go:322`** — 硬删除改为软删除以提高可恢复性

---

## 📈 良好实践（v1.31 期间沉淀）

1. **`base.CacheProvider` / `base.GetOrSetJSON[T]`** — 缓存抽象单一权威（Phase 92）
2. **`pkg/query.NormalizePagination`** — 分页统一入口
3. **`pkg/constants/{pagination,timeouts,ports,protocol,concurrency}.go`** — 常量统一真相源
4. **`pkg/response/handler_helpers.go`** — handler wire 契约（Phase 104）
5. **`pkg/errors/apperrors`** — 业务错误码统一（1001/1500/400001/409001）
6. **`operlog.Record` / `RecordWithBody`** — 操作日志统一落地
7. **`MonitorLogHandler[T]` 泛型** — 14 handler dedup 模式
8. **`base.GORMRepository[T]`** — 通用仓储模式
9. **AD Service Account Pool（Phase 36）** — 多账号熔断机制
10. **deviceLock 内原子 refCount（Phase 92）** — 优雅的连接生命周期管理

---

## 📋 v1.31 代码质量总览

| 模块 | P0 | P1 | P2 | 总计 | 评价 |
|------|----|----|----|------|------|
| AD 域控 | 3 FIXED + 1 NEW | 0 | 1 | 4 | 主要安全路径已修复，但 `ad_authenticator.go` 遗漏 |
| System | 2 FIXED | 1 NEW (err.Error 泄漏) | 0 | 3 | handler 收敛基本完成 |
| Operations | 1 FIXED | 3 NEW (Statistics/Search) | 0 | 4 | CRUD 路径良好，端点遗漏 |
| Network | 2 FIXED | 1 NEW (port_write goroutine) | 0 | 3 | — |
| WebSocket | 1 PARTIAL | 0 | 0 | 1 | main.go 硬编码需修复 |
| Cache | 0 | 1 NEW (Redis TLS) | 0 | 1 | 新发现 |
| Scheduler | 1 FIXED | 0 | 0 | 1 | InvokeTarget 白名单已实现 |
| Crypto | 3 FIXED | 0 | 0 | 3 | 安全增强完整 |
| Monitor | 0 | 1 NEW (login_log nil) | 0 | 1 | 泛型重构后的小问题 |

---

## 🧪 推荐回归守护测试

| 测试目标 | 文件 | 覆盖断言 |
|---------|------|----------|
| **Redis TLS 默认严格** | `pkg/cache/redis_test.go` | `TestRedis_TLSConfig_NotInsecureByDefault` |
| **ADA authenticator TLS** | `internal/core/security/ad_authenticator_test.go` | `TestADA authenticator_TLSConfig_StrictByDefault` |
| **WebSocket allowedOrigins** | `cmd/main_test.go` | `TestMain_AllowedOrigins_FromConfig_NotWildcard` |
| **HandleGetByID 404** | `pkg/response/handler_helpers_test.go` | `TestHandleGetByID_Returns404_NotBadRequest` |
| **LoginLog Clean nil guard** | `internal/api/v1/monitor/login_log_handler_test.go` | `TestLoginLog_Clean_NilCore_DoesNotPanic` |
| **operations handler err.Error 隔离** | `internal/api/v1/operations/*_test.go` | `TestStatistics_ErrorBody_DoesNotLeakSQL` |
| **OperLog 异步 panic recover** | `internal/services/oper_log_service_test.go` | `TestOperLog_Async_DoesNotPanicOnDBError` |
| **Captcha Increment 失败 fail-closed** | `internal/core/captcha_test.go` | `TestCaptcha_Increment_Failure_FailsClosed` |

---

## 📂 详细模块报告索引

本次审计由 6 个并行 Explore agent 输出，按关注点拆分：

1. **P0 安全 8 项** — 密钥/TLS/CORS/权限（6 FIXED + 2 PARTIAL）
2. **P0 并发 7 项** — 裸 goroutine/竞态/channel（7/7 FIXED）
3. **P0 SQL 注入 + 越权** — 4/4 FIXED + 2 新风险点（硬删除）
4. **P1 8 项** — 7 FIXED + 1 PARTIAL（子菜单权限继承）
5. **Phase 104/107/108 新代码** — 4 MUST-FIX + 评分 6.5/10
6. **并发模式总体审计** — 4 P0 新发现 + TOP 10 风险清单

---

**审查者**：Claude Code（golang-pro skill + 6 个 Explore agent 并行）
**下次审计建议**：v1.32 SHIPPED 后（约 2026-10 月初）复查本报告 P0/P1 残留项的修复状态
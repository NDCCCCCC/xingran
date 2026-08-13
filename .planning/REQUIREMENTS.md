# Requirements: XingRan-Next — Milestone v1.21

**Defined:** 2026-08-12
**Milestone:** v1.21 API Key 认证链修复 (API Key Auth Chain Repair)
**Core Value:** End-to-end operational observability and auditability — API Key 作为 JWT 之外的第二条认证通道,其认证链、作用域校验、使用日志必须真实生效且可观测。

> **回归性质:** 本里程碑是对 v1.6「API 密钥管理系统」(Phase 16 / 2026-05-19)的回归修复。调查发现 API Key 认证链在生产路径功能上断开(死代码中间件 + 失败的类型断言 + 前端契约不匹配 + 使用日志失真)。所有需求均为"修复已存在但失效的能力",非新功能。

## v1 Requirements

### AUTH — 认证链核心

- [x] **AUTH-01**: API Key 认证时 `setUserContextForAPIKey` 能正确把 `user_id` / `api_key_id` / `scopes` / `auth_type` 写入 gin context(修复 P0-2 类型断言恒 false — `*models.APIKey` 指针被断言为局部值类型 `apiKeyType`,消除"用 interface{} 避免循环导入"的误判,改为直接 import models 包)
- [x] **AUTH-02**: `MultiAuth` 及其下游 `RequireScope` / `RequireAPIKeyResourcePermission` / `RateLimitByScope` 不再是死代码 — 类型签名、参数传递、作用域匹配逻辑经审查正确,具备被路由挂载的条件(P0-1);`NewUsageLogger` / `NewRateLimiter` 在代码库中有真实实例化路径
- [x] **AUTH-03**: 完成 MultiAuth 路由挂载的启用决策与安全评估 — 通过 phase 内 discuss 明确"是否在生产路由挂载 MultiAuth 使 X-API-Key 认证真正生效",产出决策记录(含作用域继承、IP 白名单、与 JWT 的优先级/回退关系等安全影响)
- [x] **AUTH-04** (ex-FUTURE-APIKEY-01, Phase 61): `RequireAPIKeyResourcePermission(resource, action)` 的 `resource` 参数真实生效 — resource→permission 映射接入,继承权限 (InheritPerms) 下的细粒度资源校验经测试覆盖(仅在 Phase 60 AUTH-03=启用 后执行)

### CONTRACT — 前后端路由契约

- [ ] **CONTRACT-01**: 前端 `getAPIKey` / `updateAPIKey` / `deleteAPIKey` 三个操作不再 404 — 与后端 `apikey_router.go` 注册的路由方法/路径对齐(修复 P1-1;前端 GET/PUT/DELETE vs 后端 POST 的不匹配,由 discuss 决定统一方向:改前端用 POST,或后端补 RESTful 方法)
- [ ] **CONTRACT-02** (Phase 58 discuss 新增,2026-08-13): 前后端**字段命名契约对齐** — 后端 camelCase(`ipWhitelist`/`inheritPerms`/`isActive`/`expiresAt`)与前端 snake_case(`ip_whitelist`/`inherit_perms`/`is_active`/`expires_at`)不一致,且 `api.ts` 无 snake↔camel 转换层。后果:① Create/Update 绑定静默丢弃复合字段(取零值);② List/详情/编辑表单复合字段显示 undefined。修复方向=前端 `types/apikey.ts` + `index.tsx` → camelCase(审计确认后端 camelCase 是全项目约定,FE snake 是孤例),后端零改动。范围限定 API Key 管理 CRUD 类型;`APIKeyUsageLog`/`UsageSummary` 留 Phase 59。

### OBSERV — 可观测性 / 使用日志

- [x] **OBSERV-01**: API Key 使用日志在请求处理完成(`c.Next()` 之后)记录,`StatusCode` / `Duration` / `Success` 取真实值(修复 P1-2 — 当前在 `c.Next()` 前异步记录导致全零值)
- [x] **OBSERV-02**: `GetUsageLogSummary` 的 `successRate` 基于真实的 `Success` 字段聚合,不再恒 ≈ 0%(P1-2 连锁)
- [x] **OBSERV-03**: 使用日志异步 goroutine 使用独立的、不被请求生命周期取消的 context,消除复用 `c.Request.Context()` 的取消竞态(P2)

### SEC — 安全

- [x] **SEC-01**: API Key 存储方式决策与(可选)迁移 — 评估并决定是否将 `Key` 字段从明文改为哈希存储(SM3 或 argon2id),`ValidateAPIKey` 从明文 `WHERE key = ?` 改为哈希比对;若迁移,提供平滑过渡与回滚方案(P2,discuss 决策)
- [x] **SEC-02**: 移除 migration 085 中与 `key` 字段 `uniqueIndex` 重复的冗余索引 `idx_api_keys_key`(P3)

### QUAL — 代码质量与回归防护

- [x] **QUAL-01**: `RateLimitByScope` 的限流响应头 `X-RateLimit-Limit` / `X-RateLimit-Remaining` 用 `strconv.Itoa` 序列化,而非 `string(rune(int))` 编码错误(P2)
- [x] **QUAL-02**: 为 `MultiAuth` / `setUserContextForAPIKey` / `RequireScope` 补充集成测试,覆盖"API Key 认证 → 上下文写入 → 作用域校验"链路,防止 P0-2 类型断言回归(当前 `apikey_test.go` 仅测 3 个纯函数,无集成覆盖)
- [x] **QUAL-03** (ex-FUTURE-APIKEY-02, Phase 61): `RateLimitByScope` 随 MultiAuth 生产挂载全量接入生产路由的配置与调优 — 多 scope key 的限流作用域选择逻辑正确(不再任意只取首个 scope);仅在 Phase 60 AUTH-03=启用 后执行

## v2 Requirements (Future)

随 AUTH-03 启用决策落地后的后续工作,本里程碑不交付:

### API Key 治理扩展

> **FUTURE-APIKEY-01 / FUTURE-APIKEY-02 已于 2026-08-12 升级为 v1 需求 `AUTH-04` / `QUAL-03`,归 Phase 61(资源级权限矩阵 + 限流生产调优),条件:Phase 60 AUTH-03=启用。**

- **FUTURE-APIKEY-03**: 密钥轮换(rotation)与吊销列表机制
- **FUTURE-APIKEY-04**: 使用配额告警与异常调用检测

## Out of Scope

| Feature | Reason |
|---------|--------|
| 新增 API Key 业务功能(轮换策略、批量签发、有效期批量续期) | 本里程碑是回归修复,仅恢复已失效能力,不扩展功能边界 |
| 完整 API Key 治理(配额、异常告警、自动熔断) | 超出回归修复范围,见 FUTURE-APIKEY-04 |
| 重写认证中间件架构(如改为统一 AuthN 中间件链) | 过度工程化,目标是最小化修复使现有设计生效 |

## Traceability

Phase 映射(由 `.planning/ROADMAP.md` v1.21 确认;phase 从 v1.20 末尾 Phase 56 续编):

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 57 | Complete |
| AUTH-02 | Phase 57 | Complete |
| QUAL-02 | Phase 57 | Complete |
| CONTRACT-01 | Phase 58 | Pending |
| CONTRACT-02 | Phase 58 | Pending (discuss 新增 2026-08-13) |
| OBSERV-01 | Phase 59 | Complete |
| OBSERV-02 | Phase 59 | Complete |
| OBSERV-03 | Phase 59 | Complete |
| AUTH-03 | Phase 60 | Complete |
| SEC-01 | Phase 60 | Complete |
| SEC-02 | Phase 60 | Complete |
| QUAL-01 | Phase 60 | Complete |
| AUTH-04 | Phase 61 | Pending (conditional on P60 AUTH-03=启用) |
| QUAL-03 | Phase 61 | Pending (conditional on P60 AUTH-03=启用) |

**Coverage:**
- v1 requirements: 14 total
- Mapped to phases: 14
- Unmapped: 0 ✓
- Orphans: 0 ✓
- Duplicates: 0 ✓ (each requirement mapped to exactly one phase)

**Phase grouping rationale:**
- **Phase 57** (AUTH-01 + AUTH-02 + QUAL-02): P0 修复 + 回归测试同 phase,fix-then-lock 模式
- **Phase 58** (CONTRACT-01 + CONTRACT-02): 前后端契约层独立(路由方法 + 字段命名),与中间件修复解耦;CONTRACT-02 为 2026-08-13 discuss 审计发现的字段命名断裂,与 CONTRACT-01 同属契约层故同 phase 吸收
- **Phase 59** (OBSERV-01/02/03): 三项都触及使用日志观测请求生命周期,自然耦合
- **Phase 60** (AUTH-03 + SEC-01/02 + QUAL-01): 启用/哈希两项 discuss 决策 + 两项直接硬化项(索引 + 限流头)
- **Phase 61** (AUTH-04 + QUAL-03, ex-FUTURE-APIKEY-01/02): MultiAuth 启用后的能力补全 — 资源级权限矩阵 + 限流生产调优;独立 phase 因属实施型工作(区别于 Phase 60 的决策型),且依赖 AUTH-03=启用

---
*Requirements defined: 2026-08-12*
*Last updated: 2026-08-13 — CONTRACT-02 added (字段命名契约对齐,Phase 58 discuss 审计发现并吸收;前端→camelCase,后端零改动;UsageLog/UsageSummary 留 Phase 59). Now 5 phases (57-61) / 14 requirements / 100% coverage.*

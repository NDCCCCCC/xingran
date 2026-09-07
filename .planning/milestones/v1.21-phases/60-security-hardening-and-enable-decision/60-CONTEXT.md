# Phase 60: 安全加固与启用决策 - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

完成 MultiAuth 路由挂载启用与 API Key SM3 哈希存储两项**安全决策**（产出决策记录），落地直接硬化项（限流响应头编码修复、冗余索引手动 SQL），使 API Key 认证链在 `/system/apikeys/*` 管理面具备生产启用条件，**直接**触发 Phase 61（资源级权限矩阵 + 限流生产调优）的执行。

**Requirements:** AUTH-03（启用决策）, SEC-01（SM3 单向哈希迁移）, SEC-02（冗余索引手动 SQL）, QUAL-01（限流头 `strconv.Itoa` 修复）

**本 phase 不做的事**（已显式划出 scope）：getScopeFromContext 多 scope 选择逻辑（QUAL-03 范畴，归 Phase 61）、`RequireAPIKeyResourcePermission` 的 resource 参数接入（AUTH-04 范畴，归 Phase 61）、username 语义修正（Phase 61 资源权限领域）、密钥轮换/吊销（FUTURE-APIKEY-03/04，仍 v2 Future）。

</domain>

<decisions>
## Implementation Decisions

### AUTH-03 MultiAuth 启用决策（生产挂载）

- **D-01:** **启用——全量挂载。** 在 `internal/api/router.go:238-248` 的 `apikeys` 路由组按以下顺序挂载：`RequirePermissions([]string{"system:apikey:*"})` → `MultiAuth(apiKeyService, usageLogger)` → `RateLimitByScope(rateLimiter)`。触发 Phase 61（资源级权限矩阵 + 限流生产调优）立即执行，不再 defer。
- **D-02:** **挂载范围 = 仅 `/system/apikeys/*` 管理面。** 不冲击 JWT 认证的所有其他模块。X-API-Key 认证仅对密钥管理面（admin CRUD + 使用日志查询）生效；运维 / 资产 / 工位等模块继续纯 JWT。
- **D-03:** **认证优先级 = X-API-Key 优先 + JWT 回退。** 与 `apikey.go:27-31` 现有逻辑一致——无 X-API-Key 头时 `MultiAuth` `c.Next()` 跳过，由 JWT 中间件接管；有 X-API-Key 时 `MultiAuth` 完成认证 + 写 context + c.Next() 后记使用日志，JWT 跳过。
- **D-04:** **IP 白名单 = 启用严格拒绝。** 保留 `apikey.go:49-56` 现有 `isIPAllowed` 行为——`IPWhitelist` 非空且客户端 IP 不在白名单（含 CIDR）→ `403 "客户端IP不在白名单中"`。配置默认空白名单即「所有 IP 允许」（与现有语义一致）。

### SEC-01 API Key SM3 单向哈希迁移

- **D-05:** **存储方案 = SM3 单向哈希。** 选用 SM3 单次哈希（`SM3(key + salt)`）而非 SM3-PBKDF2 或 SM4 对称加密。理由：① API Key 是 `crypto/rand` 32 字节随机 = 256 bits 高熵，无字典攻击风险，PBKDF2 拉伸是过度设计；② SM3 哈希不可逆，DB + salt 泄漏后无法还原明文 key；③ 与项目国密栈一致（SM3 已用于密码）；④ 不依赖 SM4_KEY / SM2 私钥的保护强度。
- **D-06:** **Schema 变更 = `Key` 列移除 + `KeyHash` + `Salt` + `KeyPrefix` 三列新增。** `internal/models/api_key.go` `Key string`（gorm uniqueIndex）改为三个字段：`KeyHash string`（`size:64;uniqueIndex;not null`，存 SM3 输出 hex）、`Salt string`（`size:32;not null`，存 hex）、`KeyPrefix string`（`size:12;not null`，存 key 前 12 字符用于 List 搜索）。`ListAPIKeys` 的 `LEFT(key, 12) LIKE` 改为 `KeyPrefix LIKE`（前缀搜索能力保留，参见 D-07）。
- **D-07:** **List 搜索 = KeyPrefix + Name 双字段。** 删除 `apikey_service.go:243/245` 的 `LEFT(key, 12) LIKE` 分支；改为 `Name LIKE ? OR KeyPrefix LIKE ?`（跨 SQLite / PG 一致，不再需要 dialect 兼容分支）。KeyPrefix 在创建时一次性生成并存入。
- **D-08:** **无迁移路径——直接切换。** 用户确认当前生产 DB 中无活跃 API key（admin 自述「现在没有使用中的 api key」），不需要双读期 / 回填脚本 / 渐进步骤。直接改造 schema 与代码：移除 `Key` 列 → 新增 `KeyHash`/`Salt`/`KeyPrefix` 列 → `ValidateAPIKey` 改为 SM3 比对 → `CreateAPIKey` 生成后立即算哈希并存。
- **D-09:** **创建流程 = 一次性返回明文。** `CreateAPIKey` 仍调用 `generateKey()` 生成明文 → 算 `SM3(key+salt)` → 存 `KeyHash`/`Salt`/`KeyPrefix` → CreateAPIKeyResponse 只返回一次明文（与现有契约一致）。后续管理员无法通过 List/GetByID 重新查看到明文。轮换走「重新创建」路径（旧 key 需在新流程下用户重新签发，而非「查回明文」）。

### SEC-02 冗余索引手动 SQL

- **D-10:** **交付形式 = 手动运维 SQL + 文档 + 验证查询，不写 Go migration。** 提供 `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`（`DROP INDEX IF EXISTS idx_api_keys_key;`）+ `.planning/notes/260813-sec02-redundant-index-removal.md`（含为什么移除、如何跑 SQL、验证查询）。**不**新建 `internal/core/db/migrations/` 下的 Go migration。理由：手动运维 SQL 与 Phase 60 决策快路径一致；migration_085 已 `//go:build archive_skip` 不再执行，索引靠历史 DB schema 状态遗留，运维控制更直接。

### QUAL-01 限流响应头编码修复

- **D-11:** **修复方式 = `strconv.Itoa` 替换 `string(rune(int))`。** `internal/middleware/apikey.go:267-268` 改为 `c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))` + `c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))`。Limit=100 → "100"（不再是 "d"）。同步加 `"strconv"` import。
- **D-12:** **测试覆盖 = 单元测试 + 集成测试。** ① 单元测试 `internal/middleware/apikey_test.go`（扩展）：构造 `RateLimitResult{Limit: 100, Remaining: 99}` 调用 `RateLimitByScope` 中间件 → 断言 header 为 `"100"` / `"99"`，可被 `strconv.Atoi` 反解析。② 集成测试复用 Phase 59 sqlite 模式：真实 `gin.Engine` + `httptest` + `NewRateLimiter` + 中间件链路 → 触发限流前导请求 → 断言响应头。
- **D-13:** **范围限定 = 仅 QUAL-01。** `getScopeFromContext` 多 scope 选择逻辑（`apikey.go:285-304`，`scopes[0]` 任意只取首个）是 QUAL-03 范畴，**严格**留 Phase 61 处理。Phase 60 不顺手修，避免与 Phase 61 资源权限设计决策冲突。

### Claude's Discretion

- SM3 哈希格式是否带版本前缀（如 `$sm3$...`，与用户密码格式一致）——planner 可定。**建议**：保持简单，单纯 `KeyHash` 列存 hex，避免与 `HashPassword` 的 `$sm3$iterations$salt$hash` 格式混淆（API Key 单次哈希无 iterations）。
- Salt 长度——16 字节（32 hex 字符）足够，planner 在 16-32 字节间选。
- 新 schema 列顺序在 `models/api_key.go` 中的位置（BaseModel 之后 vs 末尾）。
- 验证查询的具体 SQL 形式（PG `pg_indexes` vs SQLite `sqlite_master`）——planner 在 `.planning/notes/` 文档中按目标环境提供。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与规划
- `.planning/ROADMAP.md` §Phase 60 — Goal / Depends on (Phase 57 + 59) / Requirements (AUTH-03 + SEC-01/02 + QUAL-01) / Success Criteria (SC#1-4)
- `.planning/REQUIREMENTS.md` — AUTH-03 / SEC-01 / SEC-02 / QUAL-01 定义 + Phase 60 Success Criteria 映射
- `.planning/STATE.md` §根因调查结论 — P0-1 (MultiAuth 死代码) / P2-a (限流头编码) / P2-c (密钥明文) / P3 (冗余索引) ground-truth 表

### 既有 Phase 决策（必须遵守 upstream 约束）
- `.planning/phases/57-auth-chain-core-fix-regression-test/57-CONTEXT.md` — D-04 保留全部 7 context 键（user_id/username/nickname/api_key_id/scopes/auth_type/inherit_perms），D-03 4 中间件自洽
- `.planning/phases/58-route-contract-alignment/58-CONTEXT.md` — D-03 字段命名约定（camelCase 是全项目约定）
- `.planning/phases/59-observability-usage-log-fix/59-CONTEXT.md` — D-02 detached context 在 UsageLogger impl 内部（请求生命周期免疫是 UsageLogger 契约）；D-01 Success = 仅 2xx

### 待修复 / 待改造源码
- `internal/middleware/apikey.go` — D-01 挂载点（line 23-87 MultiAuth）+ D-04 IP 白名单（line 49-56）+ D-11 限流头修复（line 267-268）+ D-13 留 Phase 61（line 285-304 getScopeFromContext）
- `internal/api/router.go` — D-01/D-02 挂载位置（line 238-248 apikeys 路由组）
- `internal/services/system/apikey_service.go` — D-08 ValidateAPIKey 改造点（line 128-161，明文 WHERE key = ?）+ D-07 ListAPIKeys 关键词搜索改造（line 243-247 移除 LEFT(key, 12) LIKE）+ D-09 CreateAPIKey 流程（line 164-222）
- `internal/models/api_key.go` — D-06 schema 变更（line 11 Key 字段，改为 KeyHash/Salt/KeyPrefix 三列）
- `internal/core/db/migrations/archive/applied/migration_085_api_keys.go` — D-10 索引创建历史；带 `//go:build archive_skip` 不再编译执行
- `internal/core/security/password.go` — D-05 SM3 哈希参考实现（line 79 sm3.New），但不直接复用 `$sm3$iterations$salt$hash` 格式（D-05+D-09 Claude's Discretion）
- `internal/services/rate_limiter.go` — D-12 RateLimiter 已可用（Phase 57 D-02 验证构造函数可调用）

### 测试基建
- `.planning/codebase/TESTING.md` — 无 gomock、需 DB 用真实连接约定
- `internal/middleware/apikey_test.go` — D-12 既有 3 个纯函数测试（isValidKeyFormat/isIPAllowed/getRequiredScope），扩展点
- `internal/middleware/apikey_integration_test.go` — Phase 57 集成测试基线（fake UsageLogger + httptest）
- Phase 59 sqlite in-memory 模式 — D-12 集成测试可复用

### 国家密码学栈参考
- CLAUDE.md §Security Considerations — SM2（JWT 签名）/ SM3（密码）/ SM4（at-rest 加密）的角色定位
- `pkg/crypto/sm2_jwt.go` — SM2 工具（D-05 SEC-01 决策依据：未选 SM2 加密的理由）
- `pkg/crypto/sm4.go` — SM4 工具（D-05 决策依据：未选 SM4 对称加密的理由）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/middleware/apikey.go` MultiAuth + RequireScope + RequireAPIKeyResourcePermission + RateLimitByScope（D-01 全量已实现，Phase 57 D-03 修复自洽）：4 中间件签名与装配兼容已 Phase 57 D-02 验证
- `internal/core/security/password.go` `sm3.New()` (line 79) — SM3 哈希原语可直接 import 复用
- `internal/services/rate_limiter.go` `NewRateLimiter()` (Phase 57 D-02 已验证可调用)
- `internal/services/usage_logger.go` `NewUsageLogger(db)` (Phase 57 D-02 已验证可调用)
- `internal/core/db.GetDB()` (`core.go`) — D-08 migration 不写但 DB 引擎验证查询可用

### Established Patterns
- 国家密码学栈 = SM2 + SM3 + SM4 三件套（CLAUDE.md 已锁定）；SM3 已用于密码哈希，本次 SEC-01 与密码哈希一致用 SM3 是国密栈一致性的体现
- 标准 Router Pattern: `r.Group(prefix)` → `r.Use(middleware)` → `r.POST(...)`；当前 apikeys 组 (router.go:238-248) 只挂 `RequirePermissions`，新增 MultiAuth + RateLimitByScope 是「追加」非「替换」
- migration 命名约定: `migration_NNN_*.go` 顺序递增；SEC-02 选择「不写 migration」是显式偏离此约定（用户决策 D-10）
- GORM tag 约定: `uniqueIndex;not null`（KeyHash 沿用 Key 的索引模式）

### Integration Points
- `internal/api/router.go:238-248` (apikeys 路由组) — D-01 挂载位置；新增 middleware 顺序：`RequirePermissions` → `MultiAuth` → `RateLimitByScope`
- `internal/models/api_key.go` (APIKey struct) — D-06 schema 变更；需同步 `internal/services/system/apikey_request.go` 字段命名（Key 不出现在 request struct 中）
- `internal/services/system/apikey_service.go` ValidateAPIKey (line 128-161) — D-08 改造点
- `internal/services/system/apikey_service.go` CreateAPIKey (line 164-222) — D-09 一次性明文返回
- `internal/services/system/apikey_service.go` ListAPIKeys (line 224-) — D-07 KeyPrefix LIKE 改造
- DB schema — D-10 手动 SQL 移除 `idx_api_keys_key`

</code_context>

<specifics>
## Specific Ideas

- 用户取向（来自 Phase 57-59 CONTEXT specifics 一贯体现）：让 API Key 系统「真正可用 + 真正安全」。Phase 60 把代码就绪推到生产可用，并把「最弱的安全环节」（明文存储）升级为「单向不可逆」哈希。
- 用户决策偏离推荐：SEC-01 选了 SM3 单向哈希（我推荐 SM4 对称加密）——理由是「不可逆 = 更高安全等级」，且不依赖 SM4_KEY 保护强度；SEC-02 选了手动 SQL（我推荐 Go migration）——理由是「决策快路径 + 运维可控」。下游 planner 必须理解这些偏离是有意识的产品决策，不要「自动修正」回推荐方案。
- 用户偏好最小爆炸半径：AUTH-03 挂载仅 `/system/apikeys/*` 而非全 JWT 认证路由；SEC-02 手动 SQL 而非 Go migration；QUAL-01 不顺手修 QUAL-03 范畴的多 scope 选择。
- 「无活跃 API key」是用户提供的关键事实——简化了 SEC-01 的迁移路径（无需双读期 / 回填脚本）；planner 不要过度设计兼容层。

</specifics>

<deferred>
## Deferred Ideas

- **资源级细粒度权限矩阵**（`RequireAPIKeyResourcePermission` 的 resource→permission 映射 + InheritPerms 资源校验）→ **Phase 61 / AUTH-04**（ex-FUTURE-APIKEY-01，now unconditional since Phase 60 AUTH-03=启用）。2026-08-13 discuss 期间 D-13 明确 QUAL-03 范畴也归 Phase 61。
- **限流生产接入与调优**（`RateLimitByScope` 多 scope 选择逻辑 getScopeFromContext line 285-304、RateLimitByScope 生产路由全量接入 / 配置调优）→ **Phase 61 / QUAL-03**（ex-FUTURE-APIKEY-02）。
- **username 语义修正**（`username=ak.Name` → 取关联 User 真名）→ Phase 61 资源权限领域（需加载 User）。
- **密钥轮换/吊销、配额告警** → FUTURE-APIKEY-03/04（仍 v2 Future，未升级为 v1）。
- **SEC-01 哈希后管理界面支持轮换** → 本 phase 不做（D-09 走「重新创建」路径而非「查回明文」）。如需「查回明文」能力，需切换到 SM4 对称加密（属 v2 Future 决策）。

</deferred>

---

*Phase: 60-安全加固与启用决策*
*Context gathered: 2026-08-13*
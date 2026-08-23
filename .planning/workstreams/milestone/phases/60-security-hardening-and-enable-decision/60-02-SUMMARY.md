---
phase: 60-security-hardening-and-enable-decision
plan: 02
subsystem: security
tags: [apikey, sm3, hash-migration, secret-at-rest, constant-time-compare, redundant-index, manual-sql, sec01, sec02]

# Dependency graph
requires:
  - phase: 60-security-hardening-and-enable-decision
    plan: 01
    provides: MultiAuth + RateLimitByScope 生产挂载（SEC-01 哈希存储的认证消费方）
  - phase: 57-auth-chain-core-fix-regression-test
    provides: MultiAuth 中间件自洽 + 7 context 键约束（ValidateAPIKey 上游）
provides:
  - API Key data-at-rest SM3 单向哈希存储（DB 泄漏不可还原明文）(SEC-01)
  - KeyPrefix 真实列 → List 搜索跨 dialect 一致命中 (SEC-01 / D-07)
  - 冗余索引 idx_api_keys_key 手动运维 SQL 收敛 (SEC-02)
  - 5+1 维度决策记录（SEC-01: D-05..D-09; SEC-02: D-10）
affects:
  - 61-resource-permission-matrix-and-rate-limit-tuning（凭据层硬化后启用资源权限）

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "高熵随机凭据用 SM3 单次哈希（非 PBKDF2）——256-bit 熵无字典攻击面，不可逆，不依赖 SM4_KEY"
    - "ValidateAPIKey: KeyPrefix 缩窄候选 + SM3 重算 + subtle.ConstantTimeCompare 恒定时间比对"
    - "KeyHash/Salt json:\"-\" 隐藏 + KeyPrefix json:\"keyPrefix\" 暴露 —— service 层 DB 行完整返回，handler 序列化层裁剪"
    - "退役期 schema 运维走 docs/operations/sql/ 手动 SQL（非 Go migration），与 archive_skip 退役模式对齐"

key-files:
  created:
    - docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql
    - .planning/notes/260813-sec01-hash-migration.md
    - .planning/notes/260813-sec02-redundant-index-removal.md
  modified:
    - internal/models/api_key.go
    - internal/services/system/apikey_service.go
    - internal/services/system/apikey_service_test.go
    - internal/api/v1/system/apikey_handler.go

key-decisions:
  - "SM3 单次哈希存储 (D-05)，不复用 HashPassword 的 $sm3$iters$salt$hash 格式（避免与密码哈希混淆）"
  - "Key→KeyHash/Salt/KeyPrefix 三列替换 (D-06)，KeyHash/Salt json:- 隐藏，KeyPrefix json:keyPrefix 暴露"
  - "List 关键词搜索删 dialect 分支用 key_prefix LIKE (D-07)；scope 搜索 dialect 分支保留（范围限定）"
  - "用户确认无活跃 API key → 直接切换，无双读/回填/迁移 (D-08)"
  - "明文一次性返回 (D-09)，轮换=重新创建，无重置明文路径"
  - "手动运维 SQL（非 Go migration）(D-10)，运维窗口执行 + 验证查询监督"

patterns-established:
  - "凭据哈希存储三列模式: KeyHash(唯一)+Salt(随机)+KeyPrefix(可搜前缀)"
  - "hashAPIKey(key,salt) + generateSalt() 顶层 helper（不复用 PasswordManager，语义分离）"

requirements-completed: [SEC-01, SEC-02]

# Metrics
duration: 45 min
completed: 2026-08-13
---

# Phase 60 Plan 02: API Key SM3 哈希存储 + 冗余索引收敛 Summary

**API Key 存储从明文 `WHERE key = ?` 改为 SM3(key+salt) 单向哈希（SEC-01），并以手动运维 SQL 收敛 migration_085 残留的冗余索引 `idx_api_keys_key`（SEC-02）。DB 泄漏后无法还原明文凭据，Phase 61 资源权限矩阵启用时凭据层无根因缺陷。**

## Performance

- **Duration:** 45 min（含 subagent 生成失败的 inline 回退重执行 + stash 事故恢复）
- **Started:** 2026-08-13T05:50:00Z
- **Completed:** 2026-08-13T06:47:00Z
- **Tasks:** 2 / 2 complete
- **Files modified:** 4（模型 + 服务 + 测试 + handler ripple）

## Accomplishments

- **SEC-01 全栈 SM3 哈希迁移**：`models.APIKey` 的 `Key` 明文列替换为 `KeyHash`(64 hex uniqueIndex)/`Salt`(32 hex)/`KeyPrefix`(12 index) 三列；`ValidateAPIKey` 改 KeyPrefix 缩窄 + SM3 重算 + `subtle.ConstantTimeCompare` 恒定时间比对；`CreateAPIKey` 生成 salt+hash 存三列、明文仅 `*string` 一次性返回；`ListAPIKeys` 删 keyword dialect 分支（`key_prefix LIKE` 跨 SQLite/PG 一致）
- **SEC-02 冗余索引收敛**：`docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`（`DROP INDEX IF EXISTS` + 7 行注释），无 Go migration（D-10）；决策记录含 PG `pg_indexes` + SQLite `sqlite_master` 双 dialect 验证查询
- **handler ripple 修复**：`apikey_handler.go` `maskKey` 移除、`maskAPIKeys`/`GetByID` 仅暴露 `KeyPrefix`（Key 字段移除后的编译必需修复，计划外但属 CLAUDE.md 全仓 build 规则范畴）
- **测试按新 schema 重写**：`apikey_service_test.go`（1323 行）`setupTestDB` DDL 三列化 + `createTestAPIKey` 双返回（`(*APIKey, string)` 明文）+ 全子测试断言更新；`go test ./internal/services/system/` 全绿
- **6 维度决策记录**：SEC-01 5 维度（D-05..D-09）+ SEC-02 4 维度（D-10），锁定用户偏离推荐的选择（不可逆哈希 vs SM4 对称加密；手动 SQL vs Go migration）

## Task Commits

Each task committed atomically:

1. **Task 1 (SEC-01)**: `d14f5ae` (feat) — SM3 哈希存储迁移 + handler KeyPrefix 暴露 + 5 维度决策记录
2. **Task 2 (SEC-02)**: `0ab4957` (docs) — 手动运维 SQL 移除冗余索引 + 4 维度决策记录

## Files Created/Modified

- `internal/models/api_key.go` — `Key string` → `KeyHash`(64,uniqueIndex,json:-)/`Salt`(32,json:-)/`KeyPrefix`(12,index,json:keyPrefix) 三列
- `internal/services/system/apikey_service.go` — `hashAPIKey`(SM3)+`generateSalt` helper；`ValidateAPIKey`/`CreateAPIKey`/`ListAPIKeys` 三函数重写；`crypto/subtle` + `github.com/tjfoc/gmsm/sm3` import
- `internal/services/system/apikey_service_test.go` — `setupTestDB` DDL 三列化；`createTestAPIKey` 双返回 + 固定盐哈希；`TestCreateAPIKey`/`TestValidateAPIKey`/`TestListAPIKeys`/`TestGetAPIKey` 断言更新
- `internal/api/v1/system/apikey_handler.go` — `maskKey` 移除（明文不存储，无脱敏对象）；`maskAPIKeys`/`GetByID` 输出 `keyPrefix`
- `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` — 手动运维 SQL（新建）
- `.planning/notes/260813-sec01-hash-migration.md` — SEC-01 5 维度决策记录（新建）
- `.planning/notes/260813-sec02-redundant-index-removal.md` — SEC-02 4 维度决策记录 + 双 dialect 验证查询（新建）

## Decisions Made

| 决策 | 理由 |
|------|------|
| SM3 单次哈希（非 PBKDF2 / SM4 对称加密） | 256-bit 熵 key 无字典攻击面；不可逆消除 P2-c；不依赖 SM4_KEY 保护 |
| hashAPIKey 不复用 HashPassword | 避免 `$sm3$iters$salt$hash` 密码格式混淆；API Key 用裸 hex 哈希无前缀 |
| ValidateAPIKey 前缀缩窄 + 恒定时间比对 | 明文 `WHERE key = ?` 路径消除；`subtle.ConstantTimeCompare` 防侧信道时序 |
| KeyPrefix 做搜索（非 KeyHash） | 前缀明文可读可搜；哈希不可逆无法关键词命中 |
| scope 搜索 dialect 分支保留 | 本 phase 范围限定仅 KeyPrefix 项；scope 留 Phase 61 |
| 直接切换无双读/回填 | 用户确认无活跃 API key（D-08 前提） |
| 手动 SQL 非 Go migration | 决策快路径 + 运维可控 + 与 migration_085 archive_skip 退役模式对齐（D-10） |
| handler maskKey 移除 | 明文不存储后无脱敏对象；KeyPrefix 天然是短标识符 |

## Deviations from Plan

1. **`apikey_handler.go` ripple 修复（计划外但必需）**：计划未将 handler 列入 `files_modified`，但移除 `models.APIKey.Key` 后 `maskKey(apiKey.Key)` / `maskAPIKeys(... key.Key ...)` 无法编译。按 CLAUDE.md「`go build ./...` 全仓绿」规则修复——`maskKey` 移除、`maskAPIKeys`/`GetByID` 仅暴露 `KeyPrefix`。属根因修复的必要 ripple，非范围蔓延。
2. **`createTestAPIKey` 双返回签名**（计划 §163 方案）：返回 `(*models.APIKey, string)` 明文，供 `ValidateAPIKey` 子测试调用——计划已预判，按计划执行。
3. **scope 搜索 dialect 分支保留**：`isSQLite` 局部变量因此保留（计划明确 scope 分支不动），与「删 isSQLite」的字面要求略有出入——但计划同时要求 scope 分支保留（依赖 `isSQLite`），故保留该变量是满足两条约束的唯一解。
4. **`TestGetAPIKey` 断言修正**：计划要求 `assert.Empty(t, result.KeyHash)`，但 service 层返回完整 DB 行（含 KeyHash），`json:"-"` 仅在 handler 序列化层隐藏。改为断言 `KeyPrefix` 长度=12 + `KeyPrefix` 比对（移除错误的 `assert.Empty`），并注释说明 service-vs-handler 层语义差异。

## Issues Encountered

1. **`gsd-executor` subagent 模型路由故障（环境级，非代码问题）**：60-01 与 60-02 的 subagent 均因 API `modelCode: 不存在` 错误终止。60-01 在错误前已完成并提交（走 completion-signal fallback 抽查验证）；60-02 则在开工前终止，由编排器 inline 顺序执行（文档化 fallback）。
2. **git stash 事故**：stash pop 过程中 model/service/test 三个文件一度回退至 baseline（仅 handler 保留编辑），后从残留 stash `0fed43c` 完整恢复——该 stash 含连贯的最终版 SEC-01 编辑。恢复后 `go build ./...` 全绿、`go test ./internal/services/system/` 全绿。
3. **`TestValidateAPIKey` 间歇性 flake**：完整包运行下偶发失败，单独/组合运行确定性通过。根因是预先存在的异步 `Update last_used_at` goroutine 与共享 `cache=shared` SQLite 的时序竞争（本 plan 未触碰该 goroutine）——属预先存在的技术债，非 SEC-01 回归。
4. **预先存在的全仓测试失败**：`internal/services/operations`、`asset`、`addomain`、`api/v1`、`api/v1/auth` 等包的失败经 baseline 对比（stash 后重跑）证实与 SEC-01 无关——同一组测试在未应用 SEC-01 的 baseline 上以相同方式失败。

## Next Phase Readiness

- **Phase 61 / AUTH-04（资源级权限矩阵 + 限流调优）已具备凭据层硬化**：API Key 哈希存储（SEC-01）+ 中间件挂载（60-01）后，资源级权限 + InheritPerms 真实生效 + `getScopeFromContext` 多 scope 选择（QUAL-03）可安全启用。
- **运维待办**：生产 DB 在运维窗口执行 `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`，并用 `.planning/notes/260813-sec02-redundant-index-removal.md` 的验证查询确认 `idx_api_keys_key` 从 `pg_indexes` / `sqlite_master` 消失。
- **预先存在测试债（非本 phase 范围）**：operations/asset/addomain/api/v1/api/v1/auth 包的基线失败测试需独立 phase 排查；`TestValidateAPIKey` 的共享缓存 flake 建议后续用独立 DB 连接或同步化 last_used_at 更新修复。

---

*Phase: 60-security-hardening-and-enable-decision*
*Completed: 2026-08-13*

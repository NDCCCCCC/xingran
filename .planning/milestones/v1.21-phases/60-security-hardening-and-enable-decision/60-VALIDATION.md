---
phase: 60
slug: security-hardening-and-enable-decision
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 60 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 内容源自 `60-RESEARCH.md` §Validation Architecture（Nyquist Dimension 8 验证架构）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1（assert / require） |
| **Config file** | none — 测试紧邻源码 `*_test.go`（`.planning/codebase/TESTING.md` 约定） |
| **Quick run command** | `go test -v -run "TestRateLimit|TestMultiAuth|TestValidateAPIKey" ./internal/middleware/ ./internal/services/system/` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~数秒（sqlite 文件 DB，无外部 PG/Redis 进程） |

**真实 DB（SEC-01 SM3 哈希路径需 DB 行实证）：** sqlite 文件 DB —— 复用 Phase 59 `setupUsageLoggerTestDB`（per-test 独立文件 `os.TempDir()` + `UnixNano()` + `pid` 唯一名 + `busy_timeout=5000`，裸 `CREATE TABLE` DDL 绕过 `gen_random_uuid()` PG 专有陷阱）。

**SEC-02 手动 SQL：** 单独运维文件（`docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`），不进 Go binary；由运维人员手动跑 + notes 验证查询。

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/middleware/... ./internal/services/system/`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green；SC#1-4 必须有自动化或手动验证锚点
- **Max feedback latency:** ~10 秒（自动化测试）；SEC-02 手动 SQL 依赖运维窗口

---

## Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| **AUTH-03** / SC#1 | MultiAuth 挂载到 `/system/apikeys/*` 路由组 | 集成 | `go test -v -run TestMultiAuthProductionMount ./internal/api/...` | ❌ Wave 0 新增 |
| **AUTH-03** | MultiAuth 优先级 + JWT 回退 | 集成 | Phase 57 `TestMultiAuthIntegration` 三路径全跑 | ✅ Phase 57 既有 |
| **SEC-01** / SC#2 | `CreateAPIKey` 生成 SM3 哈希 + Salt + KeyPrefix | 单元 | `go test -v -run TestCreateAPIKeySM3Hash ./internal/services/system/` | ❌ Wave 0 新增 |
| **SEC-01** | `ValidateAPIKey` 哈希比对（恒定时间） | 单元 | `go test -v -run TestValidateAPIKeySM3Hash ./internal/services/system/` | ❌ Wave 0 新增 |
| **SEC-01** | `ListAPIKeys` 关键词搜索 `KeyPrefix LIKE` | 单元 | `go test -v -run TestListAPIKeysKeyPrefixLike ./internal/services/system/` | ❌ Wave 0 新增 |
| **SEC-02** / SC#3 | idx_api_keys_key 冗余索引被移除 | 手动 | 跑 `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` + notes 验证查询 | N/A (manual) |
| **QUAL-01** / SC#4 | 限流头 `strconv.Itoa` 编码（单测） | 单元 | `go test -v -run TestRateLimitHeaderEncoding ./internal/middleware/` | ❌ Wave 0 新增 |
| **QUAL-01** | 限流头集成测试（跨 gin.Engine + 中间件链路） | 集成 | `go test -v -run TestRateLimitHeadersInResponse ./internal/middleware/` | ❌ Wave 0 新增 |

> 任务 ID（`{N}-01-TASK` 形式）在 planner 创建 PLAN.md 后回填。上表已锁定每条 Requirement/SC 的验证机制、测试类型与命令，planner 不得偏离。

---

## Success Criteria → 验证机制（断言形式）

| SC | 验证方法 | 断言形式 |
|----|----------|----------|
| **SC#1** AUTH-03 决策记录 | `.planning/notes/260813-auth03-enable-decision.md` 存在 + 含 4 维度 | Notes 含「挂载范围 / 认证优先级 / IP 白名单 / JWT 回退」4 段（D-01..D-04 决策落地） |
| **SC#2** SEC-01 决策记录 + 哈希迁移 | `.planning/notes/260813-sec01-hash-migration.md` 存在 + 自动化测试覆盖 CreateAPIKey/ValidateAPIKey/ListAPIKeys | Notes 含「存储方案 / Schema / List 搜索 / 迁移路径 / 创建流程」5 段；`apikey_service_test.go` 三测试全绿 |
| **SC#3** SEC-02 索引收敛 | `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 存在 + notes 验证查询可执行 | `ls docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 退出码 0；PG `pg_indexes` 不再含 `idx_api_keys_key`；SQLite `sqlite_master` 不再含 `idx_api_keys_key` |
| **SC#4** QUAL-01 限流头 | 集成测试断言 + curl 可解析 | `strconv.Atoi(limitHeader)` 返回整数（>0），err=nil；header 字面量不再是单字符 |

---

## Wave 0 Requirements

- [ ] `internal/services/system/apikey_service_test.go` — 新建文件，覆盖 SEC-01 三函数（CreateAPIKey SM3 哈希 / ValidateAPIKey 哈希比对 / ListAPIKeys KeyPrefix LIKE）
- [ ] `internal/middleware/apikey_test.go` — 扩展 `TestRateLimitHeaderEncoding`（D-12 QUAL-01 单测，断言 `strconv.Atoi` 反解析成功）
- [ ] `internal/middleware/apikey_integration_test.go` — 扩展 `TestRateLimitHeadersInResponse`（D-12 QUAL-01 集成测试，跨真实 gin.Engine + httptest）
- [ ] `internal/api/router_test.go` — 新建（可选），验证 `/system/apikeys/*` middleware 链装配（gin.Engine + 触发，验证 MultiAuth 在 RequirePermissions 与 RateLimitByScope 之间生效）

*Wave 0 4 项，全部由 planner 在 plan 1 起步时一次性交付。*

---

## Hermetic 性保证（sqlite 测试 DB）

- **Per-test 独立文件 DB:** `os.TempDir()` + `UnixNano()` + `pid` 唯一名 → 测试间零共享状态。
- **`busy_timeout=5000`:** 写锁排队而非立即报错，消除并发 goroutine 撞锁。
- **不用 `t.TempDir()`:** fire-and-forget goroutine 测试结束后仍可能写文件，自动 cleanup 删占用文件 mark fail。
- **裸 `CREATE TABLE` DDL:** 绕过 `gen_random_uuid()` PG 专有陷阱（`60-RESEARCH.md` §Pitfall 1）。
- **零外部进程:** 无需启动 PostgreSQL/Redis，CI 友好。
- **SEC-02 SQL hermetic 性:** SQL 文件单独存放，运维窗口跑，不参与 Go 测试套件。

---

## Manual-Only Verifications

- **SEC-02 / SC#3:** idx_api_keys_key 索引移除是手动运维 SQL 操作，需运维窗口执行 + 跑 PG/SQLite 验证查询确认索引收敛。预期在 production-like DB 上跑，避免 CI 假阳。
- **AUTH-03 决策记录:** `.planning/notes/260813-auth03-enable-decision.md` 决策记录是人工编写的 markdown 文档，非自动化产物。验证方式 = 文件存在 + 4 维度段落齐全。

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s（自动化）
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
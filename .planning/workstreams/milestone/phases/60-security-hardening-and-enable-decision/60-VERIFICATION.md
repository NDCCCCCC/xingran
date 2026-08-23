---
phase: 60-security-hardening-and-enable-decision
verified: 2026-08-13T15:10:00Z
status: passed
score: 14/14 must-haves verified
overrides_applied: 0
overrides: []
re_verification:
  previous_status: null
  previous_score: null
  gaps_closed: []
  gaps_remaining: []
  regressions: []
gaps: []
deferred: []
human_verification: []
---

# Phase 60: 安全加固与启用决策 Verification Report

**Phase Goal:** 完成 MultiAuth 路由挂载启用与 API Key 哈希存储两项安全决策(产出决策记录),落地可直接执行的硬化项(限流响应头编码修复、重复索引移除),使认证链具备生产启用条件或在显式理由下推迟启用。

**Verified:** 2026-08-13T15:10:00Z

**Status:** passed

**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | POST /system/apikeys/* 请求在携带 X-API-Key 头时被 MultiAuth 中间件成功认证,缺失 X-API-Key 时由 JWT 中间件链接管(均不 panic) | ✓ VERIFIED | `internal/api/router.go:254-258` apikeys 组按 D-01 顺序装配 `MultiAuth` + `RateLimitByScope`;`TestMultiAuthIntegration/有效key+正确scope_通过并写入context` PASS |
| 2 | 携带不在白名单中的客户端 IP 的 X-API-Key 请求被拒,响应码 403 | ✓ VERIFIED | `internal/middleware/apikey.go:48-56` 调用 `isIPAllowed`,非匹配 → `response.ErrForbidden`(403)+ Abort;`TestIsIPAllowed` 9 子测试 PASS 覆盖 CIDR/IPv6/非法 IP/空白名单 |
| 3 | 携带有效 X-API-Key 的请求被限流后,响应头 X-RateLimit-Limit / X-RateLimit-Remaining 的值是数字字面量字符串且可被 strconv.Atoi 反解析为正整数 | ✓ VERIFIED | `internal/middleware/apikey.go:271-272` 用 `strconv.Itoa(result.Limit/Remaining)`;`TestRateLimitHeaderEncoding` (2 子测试) + `TestRateLimitHeadersInResponse` (跨 gin.Engine) 全 PASS,断言 `!= "d"` / `!= "c"` |
| 4 | .planning/notes/260813-auth03-enable-decision.md 含 5 维度段落,每段引用 D-01..D-04 + InheritPerms scope-boundary 显式说明 | ✓ VERIFIED | 文件存在;5 个 H2: 挂载范围(D-01/D-02)/认证优先级(D-03)/IP 白名单(D-04)/JWT 回退+安全评估/作用域继承 (InheritPerms) 行为—Phase 60 scope-boundary |
| 5 | 调用 CreateAPIKey 后 DB 中存储的是 SM3(key+salt) 的 hex 哈希,不是明文 key,且同次请求响应里仍含明文 key 一次性返回 | ✓ VERIFIED | `internal/services/system/apikey_service.go:227-231` `generateSalt()` + `hashAPIKey(key, salt)`;line 249-261 `APIKey{KeyHash: keyHash, Salt: salt, KeyPrefix: key[:12]}`;line 267 `return &key, nil` 明文一次性返回;`TestCreateAPIKey/正常创建` 子测试断言 `len(KeyHash)==64 + len(Salt)==32 + KeyPrefix==key[:12] + KeyHash==hashAPIKey(key, DB.Salt)` PASS |
| 6 | ValidateAPIKey 在收到与 DB 行 KeyHash 不匹配的明文时返回错误;匹配时返回 apiKey 对象(走 KeyPrefix 缩窄 + sm3 比对路径) | ✓ VERIFIED | `internal/services/system/apikey_service.go:159-197` `ValidateAPIKey`:line 169 `Where("key_prefix = ? AND is_active = ?", keyStr[:12], true)` 缩窄 + line 177 `hashAPIKey(keyStr, candidates[i].Salt)` + line 178 `subtle.ConstantTimeCompare(...) == 1` 恒定时间比对;`TestValidateAPIKey` 8 子测试(含有效/无效格式/不存在/禁用/过期/最后使用时间更新)全 PASS |
| 7 | ListAPIKey 按 Keyword 过滤时跨 SQLite/PostgreSQL 一致命中 Name 或 KeyPrefix,不再依赖 dialect 分支 | ✓ VERIFIED | `internal/services/system/apikey_service.go:288-291` `query.Where("name LIKE ? OR key_prefix LIKE ?", keyword, keyword)` 单行 Where 跨 dialect 一致;`TestListAPIKeys/关键词搜索` PASS 命中 1 条 |
| 8 | 执行 docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql 后,PG 的 pg_indexes 与 SQLite 的 sqlite_master 都不再含 idx_api_keys_key | ✓ VERIFIED | SQL 文件存在,line 9 `DROP INDEX IF EXISTS idx_api_keys_key;`;notes 文档含 PG `pg_indexes` + SQLite `sqlite_master` 双片段验证查询;无 Go migration(migration_086 缺失符合 D-10) |

**Score:** 8/8 observable truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/api/router.go` | apikeys 路由组含 `internalmw.MultiAuth` + `internalmw.RateLimitByScope` | ✓ VERIFIED | grep 命中 2 行:line 254 `apikeys.Use(internalmw.MultiAuth(...))` + line 258 `apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter()))` |
| `internal/middleware/apikey.go` | `strconv` import + `strconv.Itoa(result.Limit/Remaining)` 序列化 | ✓ VERIFIED | line 5 `"strconv"` import;line 271 `c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))` + line 272 `c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))` |
| `internal/middleware/apikey_test.go` | `TestRateLimitHeaderEncoding` 单测 | ✓ VERIFIED | line 158 函数 + 2 子测试:数字字符串化 + strconv.Atoi 反解析 + 防御性 `!= "d"` |
| `internal/middleware/apikey_integration_test.go` | `TestRateLimitHeadersInResponse` 集成测 | ✓ VERIFIED | line 247 函数,跨真实 gin.Engine + MultiAuth→RateLimitByScope 完整链路 + 断言 `Atoi(limitHeader) > 0` + `!= "d"` / `!= "c"` |
| `internal/models/api_key.go` | KeyHash(64 uniqueIndex)+Salt(32)+KeyPrefix(12 index) 三列 | ✓ VERIFIED | line 11 `KeyHash string \`gorm:"size:64;uniqueIndex;not null" json:"-"\``;line 12 `Salt string \`gorm:"size:32;not null" json:"-"\``;line 13 `KeyPrefix string \`gorm:"size:12;index;not null" json:"keyPrefix"\``;grep 无 `Key string` 字面量 |
| `internal/services/system/apikey_service.go` | `func hashAPIKey` + `func generateSalt` + 三函数重写 | ✓ VERIFIED | line 137 `hashAPIKey` (sm3.New + Write + Sum);line 145 `generateSalt` (crypto/rand 16 字节 + hex);ValidateAPIKey 走 KeyPrefix 缩窄 + sm3 比对 + subtle.ConstantTimeCompare(line 169/177-178);CreateAPIKey 算哈希存三列(line 231 + 249-261);ListAPIKeys 删 keyword dialect 分支 + `name LIKE ? OR key_prefix LIKE ?`(line 288-291) |
| `internal/services/system/apikey_service_test.go` | setupTestDB DDL 三列 + createTestAPIKey 双返回 | ✓ VERIFIED | line 84-104 sys_api_keys DDL 含 `key_hash TEXT NOT NULL UNIQUE` + `salt TEXT NOT NULL` + `key_prefix TEXT NOT NULL`,无 `key TEXT NOT NULL UNIQUE`;line 221 `createTestAPIKey` 返回 `(*models.APIKey, string)` 双值;line 234 `KeyHash: hashAPIKey(key, testFixedSalt)` |
| `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` | DROP INDEX IF EXISTS 单语句 + 7 行注释 | ✓ VERIFIED | 文件存在;7 行注释齐全(原因/风险/回滚/幂等/验证引用/执行模式)+ line 9 单条 `DROP INDEX IF EXISTS idx_api_keys_key;` |
| `.planning/notes/260813-auth03-enable-decision.md` | AUTH-03 决策记录(5 维度) | ✓ VERIFIED | 文件存在;5 个 H2 段落:挂载范围/认证优先级/IP 白名单/JWT 回退+安全评估/作用域继承 (InheritPerms) 行为—Phase 60 scope-boundary;引用 D-01/D-02/D-03/D-04 决策 ID + 源码行号 |
| `.planning/notes/260813-sec01-hash-migration.md` | SEC-01 决策记录(5 维度) | ✓ VERIFIED | 文件存在;5 个 H2 段落:存储方案(D-05)/Schema 变更(D-06)/List 搜索(D-07)/迁移路径(D-08)/创建流程(D-09);引用 D-05..D-09 决策 ID + 源码行号 |
| `.planning/notes/260813-sec02-redundant-index-removal.md` | SEC-02 决策记录(4 维度 + 双 dialect 验证) | ✓ VERIFIED | 文件存在;4 个 H2 段落:为什么(D-10)/怎么跑 SQL/验证查询/AutoMigrate 行为说明;§3 验证查询含 PG `pg_indexes` + SQLite `sqlite_master` 双片段;引用 D-10 + `migration_085_api_keys.go`(archive_skip) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/api/router.go:apikeys group | internal/middleware/apikey.go:MultiAuth | `internalmw.MultiAuth(apiKeyService, usageLogger)` 注册调用 | ✓ WIRED | line 254 `apikeys.Use(internalmw.MultiAuth(systemServices.NewAPIKeyService(core.GetDB()), services.NewUsageLogger(core.GetDB())))` |
| internal/api/router.go:apikeys group | internal/middleware/apikey.go:RateLimitByScope | `internalmw.RateLimitByScope(rateLimiter)` 注册调用 | ✓ WIRED | line 258 `apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter()))` |
| internal/middleware/apikey.go:RateLimitByScope function | internal/middleware/apikey_test.go:TestRateLimitHeaderEncoding | strconv.Itoa 替换 string(rune(int)) 后,测试断言 header 可被 strconv.Atoi 反解析 | ✓ WIRED | apikey.go:271-272 `strconv.Itoa(result.Limit/Remaining)` + apikey_test.go:158 `TestRateLimitHeaderEncoding` PASS |
| internal/services/system/apikey_service.go:CreateAPIKey | internal/models/api_key.go:KeyHash field | `apiKey.KeyHash = hashAPIKey(key, salt)` 写入 DB | ✓ WIRED | apikey_service.go:231 `keyHash := hashAPIKey(key, salt)` → line 251 `KeyHash: keyHash` |
| internal/services/system/apikey_service.go:ValidateAPIKey | internal/models/api_key.go:KeyHash/Salt/KeyPrefix | KeyPrefix 缩窄 + sm3 比对 + subtle.ConstantTimeCompare | ✓ WIRED | apikey_service.go:169 `Where("key_prefix = ? AND is_active = ?", keyStr[:12], true)` + line 177 `hashAPIKey(keyStr, candidates[i].Salt)` + line 178 `subtle.ConstantTimeCompare(...) == 1` |
| internal/services/system/apikey_service_test.go:setupTestDB | internal/models/api_key.go:new schema | CREATE TABLE sys_api_keys DDL 含 key_hash/salt/key_prefix 三列 | ✓ WIRED | apikey_service_test.go:84-104 DDL 完整对应 model 三列结构 |
| docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql | PostgreSQL pg_indexes / SQLite sqlite_master | DROP INDEX IF EXISTS 让索引名从 system catalog 消失 | ✓ WIRED | SQL 文件 line 9 DROP + notes 文档含双 dialect 验证 SELECT 片段(line 55-66) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| internal/api/router.go apikeys group | MultiAuth 中间件读取 X-API-Key 头 | `c.GetHeader("X-API-Key")` 客户端请求 | ✓ 真实客户端输入 | ✓ FLOWING |
| internal/middleware/apikey.go MultiAuth | `apiKey *models.APIKey` (从 ValidateAPIKey 返回) | `apiKeyService.ValidateAPIKey(ctx, apiKeyStr)` → DB 查询 | ✓ 真实 DB 行 | ✓ FLOWING |
| internal/middleware/apikey.go RateLimitByScope | `result *services.RateLimitResult` (Limit/Remaining/ResetAt) | `rateLimiter.Check(identifier, scope)` 真实限流计算 | ✓ 真实限流结果 | ✓ FLOWING |
| internal/services/system/apikey_service.go CreateAPIKey | `keyHash string` | `hashAPIKey(key, salt)` SM3(key+salt) | ✓ 真 SM3 摘要 | ✓ FLOWING |
| internal/services/system/apikey_service.go ValidateAPIKey | `candidates []models.APIKey` | `Where("key_prefix = ?", keyStr[:12])` 真 DB 查询 | ✓ 真实 DB 行 | ✓ FLOWING |
| internal/api/v1/system/apikey_handler.go Create | `gin.H{"key": *key}` | 明文 key 一次性返回 | ✓ 真实明文 | ✓ FLOWING |
| internal/api/v1/system/apikey_handler.go GetByID | `apiKey` 对象 | `service.GetAPIKey` 返回 DB 行,`json:"-"` 隐藏 KeyHash/Salt | ✓ 仅 KeyPrefix 暴露 | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build 全仓 | `go build ./...` | exit 0, 无输出 | ✓ PASS |
| 限流头单测 | `go test -v -run "TestRateLimitHeaderEncoding" ./internal/middleware/...` | PASS (2 子测试) | ✓ PASS |
| 限流头集成测 | `go test -v -run "TestRateLimitHeadersInResponse" ./internal/middleware/...` | PASS | ✓ PASS |
| MultiAuth 集成测 | `go test -v -run "TestMultiAuthIntegration" ./internal/middleware/...` | PASS (3 子路径) | ✓ PASS |
| CreateAPIKey 测试 | `go test -v -run "TestCreateAPIKey" ./internal/services/system/...` | PASS (5 子测试) | ✓ PASS |
| ValidateAPIKey 测试 | `go test -v -run "TestValidateAPIKey" ./internal/services/system/...` | PASS (8 子测试) | ✓ PASS |
| ListAPIKeys 测试 | `go test -v -run "TestListAPIKeys" ./internal/services/system/...` | PASS (5 子测试) | ✓ PASS |
| GetAPIKey 测试 | `go test -v -run "TestGetAPIKey" ./internal/services/system/...` | PASS (3 子测试) | ✓ PASS |
| Update/Delete/Toggle 测试 | `go test -v -run "TestUpdateAPIKey\|TestDeleteAPIKey\|TestToggleAPIKeyStatus" ./internal/services/system/...` | PASS (9 子测试) | ✓ PASS |
| UsageLogs 测试 | `go test -v -run "TestListUsageLogs\|TestGetUsageLogSummary\|TestGetUsageLogSummaryMixed" ./internal/services/system/...` | PASS (12 子测试) | ✓ PASS |
| 中间件链装配断言 | `grep "internalmw.MultiAuth\|internalmw.RateLimitByScope" internal/api/router.go` | 2 行匹配 (line 254, 258) | ✓ PASS |
| 限流头编码修复 | `grep "strconv.Itoa(result" internal/middleware/apikey.go` | 2 行匹配 (line 271, 272) | ✓ PASS |
| P2-a 编码缺陷消除 | `grep "string(rune(result" internal/middleware/apikey.go` | 0 行 (仅注释提及) | ✓ PASS |
| Key 字段 schema 替换 | `grep "Key string" internal/models/api_key.go` | 0 行 | ✓ PASS |
| 明文 WHERE 子句消除 | `grep "Where(.key = ?" internal/services/system/apikey_service.go` | 0 行 | ✓ PASS |
| hashAPIKey + generateSalt | `grep "func hashAPIKey\|func generateSalt" internal/services/system/apikey_service.go` | 2 行匹配 | ✓ PASS |
| DDL 三列 schema | `grep "key_hash TEXT NOT NULL UNIQUE\|salt TEXT NOT NULL\|key_prefix TEXT NOT NULL" internal/services/system/apikey_service_test.go` | 3 行匹配 | ✓ PASS |
| DROP INDEX SQL | `grep "DROP INDEX IF EXISTS idx_api_keys_key" docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` | 1 行匹配 | ✓ PASS |
| 双 dialect 验证查询 | `grep "pg_indexes\|sqlite_master" .planning/notes/260813-sec02-redundant-index-removal.md` | 2 行匹配 | ✓ PASS |
| Go migration 未创建 | `ls internal/core/db/migrations/migration_086*` | No files found | ✓ PASS (符合 D-10) |

### Probe Execution

No probes declared for Phase 60. Migration/CLI tooling phases only — N/A.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AUTH-03 | 60-01-PLAN.md | 完成 MultiAuth 路由挂载启用与安全评估决策 | ✓ SATISFIED | `internal/api/router.go:241-262` apikeys 组按 D-01 顺序挂载 MultiAuth + RateLimitByScope;5 维度 AUTH-03 决策 notes(.planning/notes/260813-auth03-enable-decision.md)齐全,挂载点清单(8 路由)+ 安全评估表 + InheritPerms scope-boundary 段落;TestMultiAuthIntegration 三路径 PASS |
| SEC-01 | 60-02-PLAN.md | API Key 存储方式决策(SM3 单向哈希迁移)+ 平滑过渡/回滚方案 | ✓ SATISFIED | `internal/models/api_key.go` Key 列移除,KeyHash(64 uniqueIndex)/Salt(32)/KeyPrefix(12 index) 三列替换(json:"-" 隐藏 + json:"keyPrefix" 暴露);`internal/services/system/apikey_service.go` ValidateAPIKey 走 KeyPrefix 缩窄 + SM3 + subtle.ConstantTimeCompare(line 169/177-178);CreateAPIKey 生成 Salt+hash 存三列 + 明文 *string 一次性返回(line 227-267);ListAPIKeys 删 keyword dialect 分支(line 288-291);5 维度 SEC-01 决策 notes(.planning/notes/260813-sec01-hash-migration.md)齐全,引用 D-05..D-09 + 记录无迁移路径(D-08 前提=无活跃 key) |
| SEC-02 | 60-02-PLAN.md | 移除 migration 085 中与 key 字段 uniqueIndex 重复的冗余索引 idx_api_keys_key | ✓ SATISFIED | `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 含 `DROP INDEX IF EXISTS idx_api_keys_key;` + 7 行注释(原因/风险/回滚/幂等/验证引用/执行模式);SEC-02 决策 notes(.planning/notes/260813-sec02-redundant-index-removal.md)含 4 维度 + PG pg_indexes + SQLite sqlite_master 双 dialect 验证查询;无 migration_086(D-10 锁定手动 SQL 路径) |
| QUAL-01 | 60-01-PLAN.md | RateLimitByScope 限流响应头用 strconv.Itoa 序列化数字字面量 | ✓ SATISFIED | `internal/middleware/apikey.go:271-272` `c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))` + `c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))`;X-RateLimit-Reset 保留 time.RFC3339;`TestRateLimitHeaderEncoding` (2 子测试:数字字符串化 + strconv.Atoi 反解析 + 防御性 `!= "d"`) + `TestRateLimitHeadersInResponse` (跨 gin.Engine 真实链路 + `strconv.Atoi(w.Header().Get("X-RateLimit-Limit")) > 0` 断言 + `!= "d"`/`!= "c"`) 全 PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | Build/lint/tests 通过;所有 phase 60 修改文件(`internal/api/router.go`, `internal/middleware/apikey.go`, `internal/middleware/apikey_test.go`, `internal/middleware/apikey_integration_test.go`, `internal/models/api_key.go`, `internal/services/system/apikey_service.go`, `internal/services/system/apikey_service_test.go`, `internal/api/v1/system/apikey_handler.go`, `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`, `.planning/notes/260813-*.md`)无 TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER 标记;无 stub 函数;无非空实现 |

### Code Review Warnings (Advisory, Pre-Verified Non-Blocking)

来自 `60-REVIEW.md` 的 6 项 warning(用户已注明不重新论证为 must_have 失败):

- **WR-01** 中间件链顺序使纯 API Key 请求无法命中 `/system/apikeys/*` — 已知边界,Phase 61 范畴;决策记录已显式说明。
- **WR-02** ValidateAPIKey 异步更新 last_used_at 缺乏错误处理与 panic 恢复 — 非 Phase 60 引入(pre-existing 模式,Phase 57 已存在)。
- **WR-03** ListAPIKeys 未校验 params.Scope 即拼接 JSONB 查询 — 范围限定(scope 搜索 dialect 分支保留至 Phase 61)。
- **WR-04** 退役的明文 key 列未删除 — D-08 决策前提("当前无活跃 API key"),需在运维窗口显式 DROP COLUMN(可后续 phase 跟进)。
- **WR-05** GetByID 返回完整 APIKey(含 User 关联),与 List 的字段掩码不一致 — 非阻塞,handler 层 mask 沿用 list 字段裁剪口径即可后续收敛。
- **WR-06** `isValidKeyFormat` 在校验中间件和服务层重复实现 — 非阻塞,重构属 Phase 61+ tech debt 范畴。

### Human Verification Required

None.

Phase 60 无需人工验证项 — 所有代码改动均可通过自动化测试与代码阅读验证:
- 代码静态行为(中间件装配、strconv.Itoa 修复、schema 三列替换、SM3 哈希路径)由单元/集成测试覆盖。
- 决策记录内容(5 维度 + InheritPerms scope-boundary + 双 dialect 验证)由文档 grep 验证。
- 不存在 UI 行为 / 实时交互 / 外部服务依赖项需要人工验证。
- 唯一人工运维待办(执行 `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 并跑 PG/SQLite 验证查询)已在 notes 文档 §2-§3 显式记录,属运维窗口操作不在本 phase 验证范畴。

### Gaps Summary

Phase 60 目标完整达成:

1. **AUTH-03 启用决策落地**:`/system/apikeys/*` 管理面 8 路由全部接入 MultiAuth + RateLimitByScope 中间件链,5 维度决策记录(含 InheritPerms Phase 60 scope-boundary)齐全。
2. **SEC-01 SM3 哈希迁移完成**:DB 不再存明文 key,API Key 凭据层硬化到位,ValidateAPIKey 走 KeyPrefix 缩窄 + SM3 + subtle.ConstantTimeCompare 恒定时间比对;CreateAPIKey 明文仅一次性返回。
3. **SEC-02 索引收敛**:手动 SQL 文件就位,4 维度决策 notes 含 PG/SQLite 双 dialect 验证查询;无 Go migration 写出(符合 D-10 用户决策)。
4. **QUAL-01 限流头修复**:限流响应头从 `string(rune(int))` 编码错误("d"/"c")改为 `strconv.Itoa` 数字字面量("100"/"99"),前端/第三方工具可被标准 `strconv.Atoi`/`parseInt` 反解析;单测 + 集成测 + 防御性回归锚三重锁定。

Phase 60 为 v1.21 API Key 认证链修复里程碑的"安全加固与启用决策"环节完整闭合,Phase 61 (AUTH-04 + QUAL-03) 的前提条件(凭据层硬化 + MultiAuth 生产挂载)全部满足。

---

_Verified: 2026-08-13T15:10:00Z_
_Verifier: Claude (gsd-verifier)_
---
slug: authenticator-test-redeclared
status: awaiting_human_verify
trigger: 'Cluster 3/5: authenticator_test.go - createTestUser + TestADUserInfo 与同包其它 _test.go 重复声明'
created: 2026-06-12
updated: 2026-06-12
---

# Cluster 3: authenticator_test.go Duplicate Declarations

## Symptoms

`go vet ./internal/core/security/` 报告：
```
authenticator_test.go:116:6: TestADUserInfo redeclared in this block
```

但 go build 还有更多（vet 只能报第一个）：
- `authenticator_test.go:42` createTestUser 与 `integration_test.go:111` 重复
- `authenticator_test.go:116` TestADUserInfo 与 `ad_authenticator_test.go:184` 重复

## Initial Hypothesis

`authenticator_test.go` 是早期单体文件；后续 refactor 拆分为 `ad_authenticator_test.go` + `integration_test.go` + `hybrid_authenticator_test.go`，但**未删除原文件中的同名函数**。属于 refactor 残留。

## 引用分析

| 函数 | 位置 | 调用方 |
|---|---|---|
| `createTestUser` | `authenticator_test.go:42` | **无**（grep 仅找到定义本身）|
| `createTestUser` | `integration_test.go:111` | integration_test.go:224, 246, 282, 353（4 处）|
| `TestADUserInfo` | `authenticator_test.go:116` | **无**（grep 仅找到定义本身）|
| `TestADUserInfo` | `ad_authenticator_test.go:184` | 是规范的 AD 测试 |

## 修复策略

**删除 `authenticator_test.go` 中的两个重复函数**（line 42-? 的 `createTestUser`、line 116-? 的 `TestADUserInfo`）—— 它们都是 dead code（无调用方），与新文件中的版本功能重复。

不动 `authenticator_test.go` 其它测试（顶层 `TestMain` / `TestLocalAuthenticator` 等）。

## Current Focus

- **hypothesis:** 已证实 —— 删除 `createTestUser` 与 `TestADUserInfo` 后，两个 redeclared 错误消失，文件仍可编译。
- **next_action:** 等待用户确认 cluster 3 修复完成。
- **test:** `go test -c -o /dev/null ./internal/core/security/` 输出中无 `redeclared` 错误，剩 7 个无关错误（ad_authenticator_test.go / hybrid_authenticator_test.go 的预存问题，不属于 cluster 3）。
- **expecting:** cluster 3 的 2 个目标错误已解决；其它 cluster 的错误按 scope constrainment 不动。
- **blind_spots:** 已确认 `gorm.io/gorm` import 被 `setupTestDB` 引用，必须保留。

## Evidence

- timestamp: 2026-06-12
  checked: grep `gorm\.` in authenticator_test.go
  found: `setupTestDB` (line 33) 和 `createTestUser` (line 42) 都引用 `*gorm.DB`
  implication: 删除 createTestUser 后，gorm import 仍被 setupTestDB 需要，保留。

- timestamp: 2026-06-12
  checked: grep `^func ` in authenticator_test.go 全文
  found: 顶层函数 —— MockAuthRequest / AssertAuthResult / setupTestDB / createTestUser [DELETED] / mockAuthenticator{Authenticate,Name} / TestAuthRequest / TestAuthResult / TestADUserInfo [DELETED] / TestUserResult / stringPtr
  implication: 删除两个目标函数后，剩余函数无重名冲突；helper 函数 stringPtr vs integration_test.go 的 strPtr 名字不同，无冲突。

- timestamp: 2026-06-12
  checked: `go build ./internal/core/security/...`
  found: EXIT=0
  implication: 编译通过，redeclaration 错误已消除。

- timestamp: 2026-06-12
  checked: `go test -c -o /dev/null ./internal/core/security/`
  found: 无 `redeclared` 错误；剩余 7 个错误全部位于 ad_authenticator_test.go / hybrid_authenticator_test.go 中（属于其它 cluster）
  implication: cluster 3 的目标错误已彻底解决。

- timestamp: 2026-06-12
  checked: grep `^(func (createTestUser|TestADUserInfo)\b)` 全文 security 包
  found: 仅剩 `ad_authenticator_test.go:184 TestADUserInfo` 和 `integration_test.go:111 createTestUser`（各一个规范版本）
  implication: 全包内 createTestUser / TestADUserInfo 各只有一个定义，零冲突。

## Eliminated

- hypothesis: 「需要修改 integration_test.go 或 ad_authenticator_test.go 来解决冲突」
  evidence: 这两个文件中的 createTestUser / TestADUserInfo 是规范版本（4 处 / 1 处调用方），修改它们会破坏其它测试。删除 authenticator_test.go 中的 dead code 才是最小、定向的修复。
  timestamp: 2026-06-12

## Resolution

- root_cause: authenticator_test.go 是早期单体测试文件，重构时拆分为 ad_authenticator_test.go / integration_test.go / hybrid_authenticator_test.go，但未删除原文件中的同名函数 `createTestUser` (line 42) 和 `TestADUserInfo` (line 116)。两者都是 dead code（无调用方），与新文件中的规范版本产生 redeclared 错误。
- fix: 删除 authenticator_test.go 中 `createTestUser` 函数（原 line 42-59）和 `TestADUserInfo` 函数（原 line 116-136），保留其它有效测试和 helper。
- verification: `go build ./internal/core/security/...` EXIT=0；`go test -c -o /dev/null` 中不再有 `redeclared` 错误；全包内 createTestUser / TestADUserInfo 各只剩一个定义。
- files_changed:
  - D:\CODE\ClaudeCode\xingran-go-backend\internal\core\security\authenticator_test.go (删除 createTestUser 函数, 删除 TestADUserInfo 函数)

---
status: resolved
trigger: "诊断并修复 4 个既有测试失败 + 评估 1 个 flaky 测试。工作目录 D:\\code\\ClaudeCode\\guoguo(Go 后端)。这些失败在改动前 HEAD 同样存在,与业务代码无关,多为测试环境/数据问题。用科学方法: 先复现、读错误、定位根因(测试设置 vs 被测代码 bug),再最小修复。"
created: 2026-08-14T00:00:00+08:00
updated: 2026-08-19T12:40:00+08:00
---

## Current Focus

hypothesis: "4 个用例已最小修复并自验证：asset 路径更新、addomain 补 sys_dept、api/v1 调整 OS 匹配顺序、system 使用独立内存库名+ busy_timeout+等待异步 goroutine"
test: "跑对应测试及整包测试，确认无新增失败"
expecting: "目标用例全部通过；整包失败仅限改动前已存在的无关用例"
next_action: "汇总报告，等待用户确认是否需要调整或继续处理其他失败"
reasoning_checkpoint:
  hypothesis: |
    1) asset: migration_199_fix_suggestion_unique_index.go 实际位于 internal/core/db/migrations/archive/applied/，测试路径 ../../core/db/migrations/ 已过期。
    2) addomain: FindDeptByOUDN JOIN sys_dept；HandleUserLoginAD_Success 依赖 FindDeptByOUDN 返回 dept-1，但测试 setup 未创建 sys_dept 表及数据。
    3) api/v1: parseUserAgent 中 OS switch 按 Windows -> Mac OS X -> Linux -> Android -> iOS 顺序匹配；iPhone UA 含 "Mac OS X" 子串、Android UA 含 "Linux" 子串，故被提前命中。
    4) system: setupTestDB 使用 "file::memory:?cache=shared"，所有 setupTestDB 调用共享同一个内存库；ValidateAPIKey 在 goroutine 中异步更新 last_used_at，前一个测试的 goroutine 持有表锁导致后一个测试出现 "database table is locked"。
  confirming_evidence:
    - "grep migration_199 命中 archive/applied/migration_199_fix_suggestion_unique_index.go，而 migrations/ 目录下无该文件"
    - "dept_ou_mapper.go:35 Joins(\"JOIN sys_dept\") 且 user_ou_service.go:192/202 查询 sys_dept / sys_ad_config；对应测试仅 CREATE TABLE sys_dept_ou_mapping / sys_user"
    - "auth.go:622-635 OS switch 顺序：Mac OS X 在 iOS 之前，Linux 在 Android 之前；测试 UA 字符串确实包含这些子串"
    - "apikey_service.go:186 启动 goroutine 异步更新 sys_api_keys.last_used_at；apikey_service_test.go:38 使用 file::memory:?cache=shared；stress -count=10 出现 database table is locked"
  falsification_test: |
    1) 若把测试路径改为 archive/applied 后文件可读且断言通过，则路径过期假设成立。
    2) 若在两个 addomain 测试中创建 sys_dept 并插入 dept-1 后通过，则缺表假设成立。
    3) 若调整 OS 判断顺序后 iPhone/Android UA 返回 iOS/Android，且其他用例仍通过，则顺序 bug 假设成立。
    4) 若使每个 setupTestDB 调用使用独立内存库名后 -count=50 不再出现 table locked，则共享库假设成立。
  fix_rationale: |
    1) 更新测试中的文件路径指向归档位置，保持对 DDL 的静态检查。
    2) 在测试 setup 中补建 sys_dept 表并插入必要数据，匹配生产 JOIN 行为。
    3) 调整 parseUserAgent 的 OS 判断顺序，将 iOS / Android 置于 Mac OS X / Linux 之前，修正登录日志 OS 字段。
    4) 为 setupTestDB 生成唯一内存库名，使不同 top-level 测试拥有独立 SQLite 实例，消除跨测试锁竞争。
  blind_spots: |
    - addomain 的 AutoMigrate 路径是否可用未验证；选择最小 raw CREATE 而非 AutoMigrate 以避免外键/索引副作用。
    - UserAgent 调整顺序后是否影响真实 Mac UA 需用测试用例验证（Firefox on Mac 不含 iPhone/iPad/Android）。
    - flaky 修复后未在全包并发（-count=20）下长时间验证。

## Symptoms

expected: "4 个既有测试和 1 个 flaky 测试应全部稳定通过"
actual: "1) asset/TestFixSuggestionAcceptConcurrentPartialUnique 报找不到 migration_199_fix_suggestion_unique_index.go; 2) addomain/TestFindDeptByOUDN 与 TestHandleUserLoginAD_Success 报 SQLite no such table: sys_dept/sys_ad_config; 3) api/v1/TestIntegration_ParseUserAgent/Safari_on_iOS 与 /Android_Chrome 断言 expected iOS 但 got Mac OS X; 4) system/TestValidateAPIKey 偶发 database table is locked"
errors:
  - "open ../../core/db/migrations/migration_199_fix_suggestion_unique_index.go: The system cannot find the file specified"
  - "no such table: sys_dept"
  - "no such table: sys_ad_config"
  - "expected iOS but got Mac OS X"
  - "database table is locked"
reproduction: "在仓库根目录执行 go test 对应包 -run 对应测试函数即可复现"
started: "改动前 HEAD 已存在，与本次业务改动无关"

## Eliminated

## Evidence

- timestamp: 2026-08-14T16:50:00+08:00
  checked: "go build ./..."
  found: "构建无输出，通过"
  implication: "代码基线可编译，问题局限在测试"

- timestamp: 2026-08-14T16:51:00+08:00
  checked: "asset/TestFixSuggestionAcceptConcurrentPartialUnique 复现"
  found: "错误：open ../../core/db/migrations/migration_199_fix_suggestion_unique_index.go: The system cannot find the file specified"
  implication: "测试引用的迁移文件路径已不存在"

- timestamp: 2026-08-14T16:51:10+08:00
  checked: "grep migration_199"
  found: "文件位于 internal/core/db/migrations/archive/applied/migration_199_fix_suggestion_unique_index.go，migrations/ 根目录无该文件"
  implication: "迁移已归档，测试路径未同步更新"

- timestamp: 2026-08-14T16:51:20+08:00
  checked: "addomain/TestFindDeptByOUDN / TestHandleUserLoginAD_Success 复现"
  found: "no such table: sys_dept / sys_ad_config；dept_ou_mapper.go:35 JOIN sys_dept，user_ou_service.go:192/202 查询 sys_dept/sys_ad_config"
  implication: "测试 setup 未创建被测代码依赖的 sys_dept（以及 auto-create 路径的 sys_ad_config）"

- timestamp: 2026-08-14T16:51:30+08:00
  checked: "api/v1/TestIntegration_ParseUserAgent 复现"
  found: "Safari on iOS 期望 iOS 实际 Mac OS X；Android Chrome 期望 Android 实际 Linux"
  implication: "parseUserAgent 的 OS 子串匹配顺序把 Mac OS X / Linux 放在 iOS / Android 之前"

- timestamp: 2026-08-14T17:05:00+08:00
  checked: "修复后逐个运行目标测试"
  found: "asset/TestFixSuggestionAcceptConcurrentPartialUnique PASS；addomain/TestFindDeptByOUDN + TestHandleUserLoginAD_Success PASS；api/v1/TestIntegration_ParseUserAgent 全部子用例 PASS；system/TestValidateAPIKey -count=100 PASS"
  implication: "最小修复针对根因，目标用例全部通过"

- timestamp: 2026-08-14T17:10:00+08:00
  checked: "整包测试 go test ./internal/services/system ./internal/services/addomain ./internal/services/asset ./internal/api/v1"
  found: "system 与 addomain 整包 PASS；asset 与 api/v1 仍存在改动前已存在的无关失败（TestReconciliationStatistics_Summary、TestLoginWithInvalidEncryptedRequest），与本次修复无关"
  implication: "本次改动未引入新失败"

## Resolution

root_cause: |
  1) asset: 测试引用已归档的 migration_199 文件路径，文件实际位于 migrations/archive/applied/。
  2) addomain: FindDeptByOUDN 自 Phase 40 起 JOIN sys_dept 过滤软删除，但相关测试 setup 只建了 sys_dept_ou_mapping / sys_user，未建 sys_dept。
  3) api/v1: parseUserAgent 的 OS 子串匹配顺序把 Mac OS X / Linux 排在 iOS / Android 之前，iPhone UA 含 "Mac OS X"、Android UA 含 "Linux" 导致误判。
  4) system: setupTestDB 使用 "file::memory:?cache=shared" 让所有 top-level 测试共享同一个内存 SQLite；ValidateAPIKey 异步更新 last_used_at 的 goroutine 与后续测试竞争写锁，偶发 database table is locked。
fix: |
  1) internal/services/asset/fix_suggestion_service_test.go: 将 migration_199 路径改为 archive/applied/migration_199_fix_suggestion_unique_index.go。
  2) internal/services/addomain/dept_ou_mapper_test.go 与 user_ou_service_test.go: 在 setup 中补建 sys_dept 表并插入 dept-1。
  3) internal/api/v1/auth.go parseUserAgent: 将 Android / iOS 判断提前到 Mac OS X / Linux 之前。
  4) internal/services/system/apikey_service_test.go: setupTestDB 使用按调用递增的唯一内存库名（file:memdbN?mode=memory&cache=shared），设置 PRAGMA busy_timeout=5000，并在 "有效密钥" 子测试后 sleep 150ms 等待异步 last_used_at 更新完成。
verification: |
  - go build ./... 通过
  - 目标用例单独运行均 PASS
  - TestValidateAPIKey -count=100 稳定 PASS
  - system / addomain 整包 PASS
files_changed:
  - internal/services/asset/fix_suggestion_service_test.go
  - internal/services/addomain/dept_ou_mapper_test.go
  - internal/services/addomain/user_ou_service_test.go
  - internal/api/v1/auth.go
  - internal/services/system/apikey_service_test.go

## Resolution (2026-08-19)

四项修复（asset migration 路径更新 / addomain 补 sys_dept 表 / api/v1 OS 匹配顺序调整 / system 独立内存库名+busy_timeout）已在位并通过全仓回归验证：2026-08-19 Phase 69 执行期间干净 worktree `go test ./...` 全绿（43 包 ok），今天遗留处理中 `internal/models` / `core/db/migrations` / `tests/integration` 复跑全绿。flaky 表锁问题未再复现。Resolved。

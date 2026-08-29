---
phase: 75-quirk-behavior-fixes
verified: 2026-08-23T04:35:00Z
status: gaps_found
score: 16/16
overrides_applied: 0
re_verification:
  previous_status: ""
  previous_score: ""
  gaps_closed: []
  gaps_remaining:
    - "Stale QUIRK lock comment asserting old MetricsCacheManager.Stop double-close panic remains in internal/core/core_74_08_test.go:379"
  regressions: []
gaps:
  - truth: "No lingering QUIRK lock comments asserting old broken behavior remain in Phase-75-affected test files"
    status: failed
    reason: "internal/core/core_74_08_test.go:379 still contains a comment stating 'Stop 双重调用会 close 已关闭 channel(QUIRK D-12);defer 一次即可'. After Q-7 (MetricsCacheManager.Stop idempotent via sync.Once), double Stop is safe; this comment asserts the old broken behavior and should be removed or rewritten."
    artifacts:
      - path: "internal/core/core_74_08_test.go"
        issue: "Line 379 retains a QUIRK lock comment describing the pre-fix Stop double-close panic."
    missing:
      - "Delete or rewrite line 379 comment to reflect the Q-7 fix (e.g. 'Stop is idempotent after Q-7; double defer is safe')."
---

# Phase 75: QUIRK 行为修正 — Verification Report

**Phase Goal:** 按“修复 + 同 commit 翻转断言 + 回归用例 + 原子 commit”五步法关闭全部 15 项 QUIRK（Q-1..Q-15）及 M-2，并确保相关测试断言与代码行为一致。

**Verified:** 2026-08-23
**Status:** `gaps_found` — 所有 16 项 QUIRK/M-2 行为已验证通过，但存在 1 处未清理的旧行为 QUIRK 注释。
**Re-verification:** No

---

## Per-QUIRK / M-2 Verification

| # | Item | Status | Evidence (file:line) |
|---|------|--------|----------------------|
| Q-1 | `MemoryCache.IncrementBy` 缺 key 时按 0 起算并新建项（返回 1），对齐 Redis INCR | PASS | `pkg/cache/memory.go:187-224`；回归 `pkg/cache/cache_74_08_test.go:131-157`；core captcha 三处 workaround 已删除，`internal/core/core_74_08_test.go:127/155/210` 均直测 `GenerateCaptcha` / `RecordLoginFailure` 真实链路 |
| Q-2 | `IncrementBy` 遇到非数字字符串返回 `ErrNotInteger` | PASS | `pkg/cache/memory.go:203-206`；回归 `pkg/cache/cache_74_08_test.go:162-168` |
| Q-3 | `ModelExtractor` 行首/空格分隔 `sysDescr` 能命中型号 | PASS | `internal/device/model_extractor.go:55` 使用 `(?:^|[\s\r\n])` 锚定；回归 `internal/device/device_74_08_test.go:39-45` 及双路径 `device_74_08_test.go:67-79` |
| Q-4 | `sm2.DecryptWithSM2` 对非法/过短密文返回 error，不 panic | PASS | `pkg/crypto/sm2_jwt.go:393-396`（`<96` 拒绝，bare decrypt 仅在 `>=97`）；回归 `pkg/crypto/crypto_74_08_test.go:324-335` 含 96 字节两种形态 |
| Q-5 | `validateFile` 无扩展名返回“不支持的文件格式”，不 panic | PASS | `internal/core/captcha_background.go:358-382`；回归 `internal/core/core_74_08_test.go:409-412` |
| Q-6 | `GetRandomEnabled` sqlite 分支不走 PG-only `@>`/`jsonb_array_length` | PASS | `internal/core/captcha_background.go:166` 以 `s.db.Type == "postgres"` 守卫 fallback 查询 |
| Q-7 | `MetricsCacheManager.Stop` 幂等（`sync.Once`） | PASS | `internal/pkg/cache/manager.go:402-408`；回归 `internal/pkg/cache/manager_stop_75_03_test.go` |
| Q-8 | USG6000E 保留尾字母 | PASS | `internal/device/model_extractor.go:59` 正则含 `[A-Z]{0,2}`；回归 `internal/device/device_74_08_test.go:45-47` |
| Q-9 | `nextIP(255.255.255.255)` 返回 nil，`ScanIPRange` 循环 nil 短路 | PASS | `internal/device/snmp_client.go:727-748` 与 `710-715`；回归 `internal/device/device_74_08_test.go:162-174` |
| Q-10 | `retry.containsIgnoreCase` 精确大小写不敏感匹配 | PASS | `internal/agent/pkg/retry/retry.go:127` 使用 `strings.Contains(strings.ToLower(...))`；回归 `internal/agent/pkg/retry/retry_74_12_test.go:42-45,105-112,138-145` |
| Q-11 | `normalizeParentID` 统一塌缩 nil/""/"0"；迁移 210 归一存量 `parent_id='0'` | PASS | `internal/services/system/menu_service.go:391-396`、`internal/models/system/requests/menu_requests.go:62-67`、迁移 `internal/core/db/migrations/migration_210_normalize_menu_parent_id.go`；注册于 `internal/core/db/database.go:859-862`（PG）与 `:879-882`（sqlite）；回归 `internal/services/system/menu_service_test.go:139-225` 与 `internal/core/db/migrations/migration_210_normalize_menu_parent_id_test.go` |
| Q-12 | `InitLogger` 非法 level 返回 error，但 logger 已初始化 | PASS | `internal/agent/server/logger.go:15-49`；回归 `internal/agent/server/agent_smoke_test.go:50-65` |
| Q-13 | `NewTLSConfigFromConfig` 全空参数报错；`cmd/agent/main.go` TLS 禁用 guard | PASS | `internal/agent/server/jwt_auth.go:88-91`、guard `cmd/agent/main.go:51-66`；回归 `internal/agent/server/agent_smoke_test.go:187-197` |
| Q-14 | `GetUnifiedDiff` 相同配置返回 `--- old\n+++ new\n` | PASS | `internal/utils/config_diff.go:249-252`；回归 `internal/utils/config_diff_test.go:33-36` |
| Q-15 | `GetDiskInfoDetailed` 去递归；平台文件存在 | PASS | `internal/pkg/system/disk_info.go:14-25` 仅委托 `getDiskInfoByPlatform`；平台文件 `disk_info_windows.go`、`disk_info_unix.go` 存在；回归 `internal/pkg/system/system_test.go:50-57` |
| M-2 | `cpu_linux.go` 包级状态加 `sync.Mutex`，无自递归 | PASS | `internal/pkg/system/cpu_linux.go:27-31`、`cpuStatsMu` 保护 `GetCPUUsage:35-36`；函数调用图为 `GetCPUUsage` → `readCPUStats`，无递归 |

**Score:** 16/16 QUIRK/M-2 items verified.

---

## Gate Results

| Gate | Command | Result | Notes |
|------|---------|--------|-------|
| Build | `go build ./...` | PASS | No compilation errors |
| Full test suite | `go test -count=1 ./...` | PASS | Exit 0 on clean run; all packages `ok` |
| Race tests (Q-1/Q-7) | `go test -race ./pkg/cache/... ./internal/core/...` | SKIP/ENV | Failed at build because `gcc` is not installed on this Windows host (`cgo: C compiler "gcc" not found`); not a code defect |
| Working tree cleanliness | `git status --short` | PASS | Only planning/session artifacts changed/untracked: `.planning/workstreams/frontend-coverage/STATE.md` (modified) and `.planning/active-workstream` (untracked). No business-code changes pending |

---

## Anti-Patterns / Cleanup

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/core/core_74_08_test.go` | 379 | Stale `QUIRK` lock comment asserting pre-Q-7 `MetricsCacheManager.Stop` double-close panic | Warning | Documents old broken behavior after the behavior has been fixed; could mislead future maintainers. Does not affect runtime. |

The other `QUIRK`/`quirk` comments found in affected test files (`device_74_08_test.go:571,617`, `system_test.go:13`) explicitly reference **D-12** decisions not to fix unrelated issues and are outside Phase 75 scope.

---

## Gaps Summary

1. **Stale QUIRK comment in `internal/core/core_74_08_test.go:379`**
   - The comment says `Stop 双重调用会 close 已关闭 channel(QUIRK D-12);defer 一次即可`.
   - Q-7 made `MetricsCacheManager.Stop` idempotent via `sync.Once`, so the described panic no longer occurs.
   - **Fix:** Remove the comment or rewrite it to document the post-Q-7 idempotency.

All 15 QUIRKs and M-2 are observably fixed in the tree; the only remaining issue is this documentation cleanup.

---

_Verified: 2026-08-23_
_Verifier: Claude (gsd-verifier)_

# Phase 102 Plan 03 Summary: CACHE-02 调用点替换（47 处）+ 扫描守护

**Plan:** 102-03
**Phase:** 102-mechanical-constants
**Completed:** 2026-09-07
**Commits:** 3

## One-liner

10 模块 47 个缓存键调用点机械替换为 cache_keys.go 常量/helper 引用，交付内联键扫描守护（12 文件窄扫硬失败）。

## Objective

CACHE-02 第二步：10 模块 47 个调用点按 Plan 02 注册表机械替换为常量/helper 引用，交付内联键扫描守护。行为等价——键值/TTL/缓存调用形态三不变。

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | system 同包 3 文件替换（notice/settings/widget）+ 新增 GetNoticeMyNoticesPattern helper | 6ef2e26 | notice_cache_impl.go, settings_cache_impl.go, widget_data_fetcher.go, cache_keys.go |
| 2 | 子包 5 文件替换（duty/workorder/knowledge/network/rpa）含 knowledge :134 条件后缀位 | 8a03ef5 | duty_cache_impl.go, workorder_cache_impl.go, knowledge_cache_impl.go, network/cache_impl.go, selector_learner.go |
| 3 | 根包 2 文件替换（api_endpoint/mac_history）+ TestCacheKeyInlineResidue 扫描守护 | 74d7eeb | api_endpoint_service.go, mac_history_query_service.go, cache_keys_102_test.go |

## Commits

- **6ef2e26** `refactor(102-03): replace cache key literals with helper calls in system package` — Task 1
- **8a03ef5** `refactor(102-03): replace cache key literals with helpers in 5 sub-packages` — Task 2
- **74d7eeb** `feat(102-03): root package 2 files replace + scan guard TestCacheKeyInlineResidue` — Task 3

## Key Files Created/Modified

| File | Change |
|------|--------|
| `internal/services/system/notice_cache_impl.go` | buildMyNoticesKey 改 helper 调用；失效位改 pattern helper；移除 fmt import |
| `internal/services/system/settings_cache_impl.go` | 3 处 settings:user 改 GetSettingsUserKey；移除 fmt import |
| `internal/services/system/widget_data_fetcher.go` | buildWidgetCacheKey 改 GetWidgetDataKey/GetWidgetDataParamsKey（%x 保留） |
| `internal/services/system/cache_keys.go` | 新增 GetNoticeMyNoticesPattern helper（Plan 02 未注册，扫描守护收敛必需） |
| `internal/services/duty/duty_cache_impl.go` | 8 位点替换；移除 fmt import |
| `internal/services/workorder/workorder_cache_impl.go` | 6 位点替换；移除 fmt import |
| `internal/services/knowledge/knowledge_cache_impl.go` | 9 位点替换含 :134 条件后缀位（GetKbCategoryStatusKey）；移除 fmt import |
| `internal/services/network/cache_impl.go` | 8 位点替换；移除 fmt import |
| `internal/services/rpa/selector_learner.go` | getCacheKey 改 GetRpaSelectorBestKey；新增 systemServices import；闭包位点未动 |
| `internal/services/api_endpoint_service.go` | 2 处 user_endpoints:%s 改 constants.UserEndpointsKeyFormat；新增 constants import |
| `internal/services/mac_history_query_service.go` | mac:vendor:%s 改 constants.MacVendorKeyFormat（import 已在位） |
| `internal/services/system/cache_keys_102_test.go` | 追加 TestCacheKeyInlineResidue 扫描守护（12 文件窄扫） |

## Verification Results

- `go build ./...` — 退出码 0
- `go test ./internal/services/system/ -count=1` — ok（TestCacheKeyEquivalence + TestCacheKeyInlineResidue 绿）
- `go test ./internal/services/duty/ ./internal/services/workorder/ ./internal/services/knowledge/ ./internal/services/network/ ./internal/services/rpa/ -count=1` — 5 包全绿
- `go test ./internal/services/ -count=1` — ok（根包 2 文件覆盖）
- grep 内联键字面量：10 文件零残留
- rpa selector_learner.go 闭包位点（:169-174/:226）未改动（Phase 103 前置保持）

## D-102-1 修订披露

api_endpoint_service.go 和 mac_history_query_service.go（根包）因 import 环无法引用 cache_keys.go（system→root 反向依赖）。修订为根包 2 格式落 `pkg/constants/cache.go`（同 D-102-2 captcha 先例，Plan 02 已注册）。

## 内联扫描守护设计（D-102-5② / D-102-7）

- **12 文件窄扫**（captcha 2 + system 3 + 子包 4 + rpa 1 + 根包 2）
- **匹配口径**：fmt.Sprintf 首参字符串字面量，剥离 % 动词后含 `:`，按 `:` 分段各非空段匹配 `^[a-z0-9_]*$`（末段允许独立 `*`）
- **白名单初始零条目**：任何豁免必须显式登记
- **定位方式**：go.mod 搜索向上推导仓库根（Windows 兼容）
- **当前结果**：12 文件 0 违规（captcha.go 已用常量；selector_learner 闭包天然不命中）

## Deviations from Plan

**Rule 2 - Auto-add missing critical functionality：**
- 新增 `GetNoticeMyNoticesPattern` helper（cache_keys.go 未注册，扫描守护上线前必须收敛）
- 新增 `systemServices` import 到 selector_learner.go（RESEARCH §import 环实证，零成本）

## Threat Flags

无新增安全面。键值逐字符等价由 TestCacheKeyEquivalence 快照守护。

## Self-Check

- [x] 47 调用点全部替换（10 模块）
- [x] knowledge :134 条件后缀位已改 GetKbCategoryStatusKey
- [x] rpa 闭包位点未动（Phase 103 前置）
- [x] 零内联键字面量残留（grep 验证）
- [x] TTL/getExpiration 行零改动
- [x] TestCacheKeyInlineResidue 绿（12 文件 0 违规）
- [x] go build ./... 退出码 0
- [x] 3 个 commit 存在

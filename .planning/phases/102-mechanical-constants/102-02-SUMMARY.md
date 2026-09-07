# Phase 102 Plan 02 Summary: CACHE-02 注册表（前8模块 + 根包2格式）

**Plan:** 102-02
**Phase:** 102-mechanical-constants
**Completed:** 2026-09-07
**Commits:** 2

## One-liner

为 notice/settings/duty/workorder/knowledge/network/widget/rpa 8模块及根包2格式注册缓存键常量与helper，前缀常量+GetXxxKey形态，等价快照测试绿。

## Objective

CACHE-02 第一步：将10模块全部键格式注册进两个宿主并交付等价快照测试（守护①）。本plan只注册、不替换调用点（调用点替换是Plan 102-03）。

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | RED→GREEN: cache_keys.go注册8模块20常量+27helper+等价快照测试 | 4433499 | internal/services/system/cache_keys.go, internal/services/system/cache_keys_102_test.go |
| 2 | pkg/constants追加根包2格式+快照测试 | f3648c6 | pkg/constants/cache.go, pkg/constants/cache_102_test.go |

## Commits

- **4433499** `test(102-02): add RED test and register 20 cache key constants + 27 helpers` — TDD RED+GREEN
- **f3648c6** `feat(102-02): register user_endpoints and mac:vendor key formats in pkg/constants` — 根包2格式+测试

## Key Files Created/Modified

| File | Change |
|------|--------|
| `internal/services/system/cache_keys.go` | 新增20常量（notice×3/settings×1/duty×3/workorder×3/knowledge×4/network×4/widget×1/rpa×1）+ 27个GetXxxKey/GetXxxPattern helper |
| `internal/services/system/cache_keys_102_test.go` | 新建TestCacheKeyEquivalence快照表+helper输出断言（20常量快照+27helper输出） |
| `pkg/constants/cache.go` | 新增UserEndpointsKeyFormat/MacVendorKeyFormat（根包2格式，import环豁免落点） |
| `pkg/constants/cache_102_test.go` | 追加TestRootServiceCacheKeyEquivalence（2格式快照） |

## Verification Results

- `go build ./...` — 退出码0
- `go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1` — ok（20常量快照+27helper输出断言）
- `go test ./pkg/constants/ -count=1` — ok（含TestCaptchaCacheKeyEquivalence 6例+TestRootServiceCacheKeyEquivalence 2例）
- `go vet ./internal/services/system/` — 无警告
- 零独立Pattern常量（grep `CacheKey\w+Pattern\s*=` 计数=0）
- 所有plan interfaces块命名/签名/键值逐字一致

## D-102-1修订披露

D-102-1字面「全集中注册进cache_keys.go」因import环不可行：api_endpoint_service.go和mac_history_query_service.go（根包）物理上不能引用system/cache_keys.go（system→root反向依赖）。修订为：8模块→cache_keys.go，根包2格式→pkg/constants/cache.go（同D-102-2 captcha先例，落点机理一致）。

## TTL口径披露

本plan注册的常量/helper均不承载TTL（TTL仍由调用点getExpiration解析）。Plan 102-01同口径披露延续有效。

## Threat Flags

无新增安全面。

## Self-Check

- [x] cache_keys.go含CacheKeyNoticeMyNotices/CacheKeyRpaSelectorBest等20常量（值同interfaces块）
- [x] cache_keys_102_test.go含20常量快照+27helper输出断言
- [x] 零独立Pattern常量
- [x] pkg/constants/cache.go含UserEndpointsKeyFormat/MacVendorKeyFormat
- [x] go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1绿
- [x] go test ./pkg/constants/ -count=1绿（含追加的TestRootServiceCacheKeyEquivalence）
- [x] go build ./...退出码0
- [x] git log 4433499/f3648c6存在

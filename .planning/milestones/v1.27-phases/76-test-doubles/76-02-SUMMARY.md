---
phase: 76-test-doubles
plan: "02"
subsystem: device
tags: [scrapligo, driver-factory, test-doubles, filetransport, injection-seam]

# Dependency graph
requires:
  - 76-01（仅排序依赖：wave 2 隔离 go.mod 写入窗口，无内容依赖）
provides:
  - newNetworkDriver 包级工厂 var（scrapli_wrapper.go:115，platform→network.Driver 唯一构造入口）
  - FileTransport 注入经公开构造器 NewScrapliWrapper→Open→SendCommand 全链的已验证模式（driver_factory_76_02_test.go）
  - internal/device/testdata/ Open 场景 fixture 模式（exec prompt + on-open 回显）
affects: [phase-78-BLOCK-04, 76-05]

# Tech tracking
tech-stack:
  added: []（零新依赖）
  patterns:
    - "包级 var 工厂：错误包装移入工厂内部保 byte 不变，调用点 `d, err := newNetworkDriver(...)` 原样 return"
    - "FileTransport fixture 回放：首行 prompt + on-open screen-length 回显 + 命令回显/输出/收尾 prompt"
    - "var 覆盖纪律顺序写法：orig := x → t.Cleanup(恢复) → 覆盖 → 之后只 t.Errorf+return 禁 t.Fatal"

key-files:
  created:
    - internal/device/driver_factory_76_02_test.go
    - internal/device/testdata/huawei_vrp_open.fixture
  modified:
    - internal/device/scrapli_wrapper.go

key-decisions:
  - "工厂首参签名 interface{}（非 PLAN 原文 string）：platformIdentifier 返回 interface{}（锐捷 patched YAML []byte），scrapligo NewPlatform 首参本就是 f interface{}——string 签名编译不过且会砍掉 patched YAML 能力（Rule 3 修正）"
  - "错误包装移入工厂内部（PLAN 裁决，优先于 PATTERNS 建议的调用点包装）：两条文案 byte 不变，工厂内聚"
  - "注释措辞避开 t.Parallel 字面：验收标准要求 grep 不到该字符串，纪律语义以『禁止并行执行』成文"

requirements-completed: [INFRA-02]

# Metrics
duration: 7min
completed: 2026-08-23
---

# Phase 76 Plan 02: ScrapliWrapper Driver 工厂 var Summary

**scrapli_wrapper.go 两构造器尾部重复的 platform→driver 构造段抽取为包级工厂 var newNetworkDriver（错误包装移入工厂、文案 byte 不变），FileTransport 经公开构造器 NewScrapliWrapper→Open→SendCommand 全链 fixture 回放的注入演示测试实证该缝可用——Phase 78 BLOCK-04（device ≥70%）的工厂入口就此交付**

## Performance

- **Duration:** 7 min（06:59:01Z → 07:06:26Z）
- **Tasks:** 2/2（各含独立 commit）
- **Files modified:** 3（1 生产重构 + 1 测试 + 1 fixture）

## Accomplishments

- INFRA-02 落地：`var newNetworkDriver`（scrapli_wrapper.go:115）成为 platform→network.Driver 唯一构造入口；NewScrapliWrapper / NewScrapliWrapperWithPort 及 connection_pool.createConnection（经构造器，零改动）全部经工厂
- 生产路径零行为变化实证：device 包现有测试全绿、portwrite 包（FileTransport e2e 先例，60s 全量）全绿、`go build ./...` exit 0
- 注入能力实证：TestDriverFactoryFileTransportInjection 覆盖工厂 var 后经公开构造器驱动 Open → SendCommand，断言响应 Contains "Huawei"
- fixture：huawei_vrp Open 场景（首行 exec prompt `<dummy-host>` 匹配 `^<[\w.\-@/:]{1,63}>$` + on-open `screen-length 0 temporary` 回显 + display version 输出），纯 LF ASCII、全 dummy 值
- lint：`golangci-lint run ./internal/device/...` 0 issues

## Task Commits

1. **Task 1: 抽取 newNetworkDriver 包级工厂 var** - `d9e9d84` (refactor)
2. **Task 2: FileTransport 注入演示测试 + testdata fixture** - `af98afb` (test)

## Files Created/Modified

- `internal/device/scrapli_wrapper.go` - +29/-26：工厂 var（含纪律注释）+ 两构造器调用点替换（:172/:226）；platformIdentifier/Telnet/SSH 分支/connection_pool.go 一律未动
- `internal/device/driver_factory_76_02_test.go` - 116 行（≥50 门槛）：factoryFixturePath（runtime.Caller 定位本包 testdata）+ TestDriverFactoryFileTransportInjection
- `internal/device/testdata/huawei_vrp_open.fixture` - 7 行 Open 场景回放字节流

## Decisions Made

- **工厂首参 interface{}（Rule 3 修正 PLAN 笔误）**：PLAN 工厂签名原文 `platformName, host string`，但 `platformIdentifier(vendor)` 返回 `interface{}`（string 或锐捷 patched YAML `[]byte`），且 scrapligo v1.4.0 `platform.NewPlatform(f interface{}, host string, ...)` 首参本就是 interface{}（RESEARCH 自己核实的签名）。string 签名直接编译失败，且类型断言 string 会砍掉 patched YAML 注入能力。修正后 key_links pattern `newNetworkDriver\(platformName` 与调用形参名不受影响。
- **错误包装位置从 PATTERNS 建议改为 PLAN 裁决**：PATTERNS 建议"工厂内不包装、调用点保留原包装"，PLAN 明文裁决"两段 fmt.Errorf 移入工厂内部"。按 PLAN 执行——两种方式错误字符串均 byte 不变，PLAN 为更高权威且工厂内聚。
- **测试闭包签名同步 interface{}**：RESEARCH 注入示例 `func(_, _ string, ...)` 按工厂实际类型修正为 `func(_ interface{}, _ string, ...)`（参数被忽略，不影响回放语义）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] PLAN 工厂签名 `platformName, host string` 与调用点实参类型不匹配**
- **Found during:** Task 1（`go build ./...` 编译失败：cannot use platformName (variable of type interface{}) as string value）
- **Issue:** platformIdentifier 返回 interface{}（锐捷 patched YAML 是 []byte），PLAN 工厂签名首参定为 string 导致两处调用点编译不过；scrapligo NewPlatform 首参本就是 `f interface{}`
- **Fix:** 工厂首参改为 `interface{}`（scrapli_wrapper.go:115），对齐 scrapligo 真实签名并保住 patched YAML 注入；Task 2 注入闭包签名同步
- **Files modified:** internal/device/scrapli_wrapper.go、internal/device/driver_factory_76_02_test.go
- **Verification:** `go build ./...` exit 0；device + portwrite 全绿
- **Committed in:** d9e9d84 / af98afb

---

**Total deviations:** 1 auto-fixed（Rule 3 blocking）
**Impact on plan:** 编译必需的最小类型修正；key_links pattern、错误字符串、must_haves truths 全部不受影响。

## Issues Encountered

- commitlint subject-case 拒绝 Task 2 首版 subject（"FileTransport" 大写开头触发 sentence-case 规则）——改小写开头措辞（"newNetworkDriver 工厂注入演示测试与 fixture"）后通过，非代码问题。
- `-race` 抽查不可执行：windows/amd64 race detector 需 cgo，本机 PATH 无 gcc（cgo: C compiler "gcc" not found）。PLAN verification 标注该项为可选；并发安全由纪律（禁并行 + t.Cleanup 恢复）+ 全包串行绿兜底。

## Verification Results（plan 收尾门）

- `grep -n "var newNetworkDriver" internal/device/scrapli_wrapper.go` → **PASS**（:115）
- `grep -c "创建平台实例失败"` / `grep -c "获取网络驱动失败"` → 各 2（代码 1 + 工厂注释 1；代码处 :122/:127 均在工厂内部，文案 byte 不变）→ **PASS**
- `platform.NewPlatform(` 代码直调 2→1（:116 工厂内；:92 是 platformIdentifier 既有注释，红线内未动）→ **PASS**
- `go build ./...` → **PASS exit 0**
- `go test -count=1 -run 'TestDriverFactory' ./internal/device/ -v` → **PASS**
- `go test -count=1 ./internal/device/` → **PASS**（现有测试 + 注入演示全绿）
- `go test -count=1 ./internal/services/portwrite/` → **PASS**（FileTransport e2e 先例包回归，60s）
- `golangci-lint run --timeout=5m ./internal/device/...` → **PASS** 0 issues
- fixture 首行 `<dummy-host>` / 纯 LF ASCII（od -c 验证）→ **PASS**
- `grep -c "t.Parallel" driver_factory_76_02_test.go` → 0 → **PASS**
- `go test -race`（可选）→ **SKIP**（本机无 gcc，cgo 不可用；见 Issues）

## User Setup Required

None - 无外部服务配置需求（fixture 回放全进程内）。

## Next Phase Readiness

- Phase 78 BLOCK-04（device ≥70% FileTransport/fake SSH）可直接照搬 driver_factory_76_02_test.go 的覆盖模式（orig→Cleanup→覆盖→生产链驱动）
- 76-05 AST 守护：本 plan 未新增任何 ForTesting 符号（注入走包私有 var，测试与生产同包物理隔离 + 编译器 _test.go 边界），守护面不变
- 债务提示：FileTransport 下 OpenContext 的 GetPrompt 轮询要求 fixture 首行确定性 prompt（RESEARCH 已知边界，Phase 78 处理）；driver 不 Close 的纪律已在本测试文件头成文

## Self-Check: PASSED

- 文件存在：scrapli_wrapper.go（modified）/ driver_factory_76_02_test.go / testdata/huawei_vrp_open.fixture / 76-02-SUMMARY.md 全部 FOUND
- 提交存在：d9e9d84 / af98afb 在 git log 中 FOUND
- 工作树干净（无未跟踪生成物）

---

*Phase: 76-test-doubles*
*Completed: 2026-08-23*

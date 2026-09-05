---
phase: 92-p2
plan: 01
subsystem: backend-cache
tags: [go-generics, type-alias, cache, miniredis, dependency-inversion, refactoring]

# Dependency graph
requires:
  - phase: 91-crud-base-repository-t-p1
    provides: internal/services/base 零依赖纯抽象包基线（gorm + pkg/errors + pkg/logger only）与"有意设计"注释风格先例
  - phase: 79 (v1.27 INFRA-01)
    provides: miniredis/v2.38.0 test-only 测试基建 + data_cache_service_79_01_test.go 双装配测试模板
provides:
  - internal/services/base 缓存抽象全套：TTLResolver 接口 + CacheServiceBase 薄基类 + CacheProvider 9 方法接口 + NoOpCacheProvider + CacheStats/CacheEntry + 泛型函数族 GetOrSetJSON[T]/SetJSON[T]/Invalidate/InvalidatePattern（单一权威位置）
  - system 侧 5 个 type alias 门面（CacheProvider/NoOpCacheProvider/CacheStats/CacheEntry/CacheServiceBase = base 同名类型），20+ 外部消费文件零改动
  - CacheConfigService.GetDurationWithDefault nil-receiver 防护（typed-nil interface → default，Pitfall 1）
  - TestBase92_* 系列 10 用例双装配测试（MemoryCache + miniredis，FastForward 纪律）
affects: [92-02 (system 29 处样板迁移靶点), 92-03 (operations + D-04 失效调用面 19 处 + root 定性 + monitor rename), 92-04 (CLAUDE.md Cache Service Convention + invariants 扫描 + LOC 双口径收口)]

# Tech tracking
tech-stack:
  added: 零新增依赖（go 1.24 泛型/type alias 语言特性 + 既有 miniredis v2.38.0）
  patterns: [consumer-defined interface（TTLResolver 隐式满足）, type alias 原位翻转（零引用面改动迁移）, 泛型包级函数族（薄委托红线）, miniredis mr.FastForward TTL 推进]

key-files:
  created:
    - internal/services/base/cache_service_base.go
    - internal/services/base/cache_provider.go
    - internal/services/base/cache_functions.go
    - internal/services/base/cache_service_base_test.go
  modified:
    - internal/services/system/cache_provider.go
    - internal/services/system/cache_utils.go
    - internal/services/system/cache_adapter.go
    - internal/services/cache_config_service.go
    - internal/services/system/cache_infra_test.go
    - internal/services/system/user_cache_impl_test.go

key-decisions:
  - "D-01 落地形态：TTLResolver 为 base 包内 consumer-defined 最小接口，*services.CacheConfigService 隐式满足，base 保持零依赖（cache 三文件仅 context/time/reflect/pkg/logger）"
  - "D-02 落地形态：system/cache_provider.go 129 行定义删除、原位留 5 alias + var _ 编译期断言，api/core/duty/knowledge/network/workorder 全部消费者零改动（git diff 证实）"
  - "私有 setValue 导出为 base.SetValue（唯一迁移偏差）：同包 CacheAdapter（core.go/excel_handler.go 生产装配在用）与 2 个同包测试文件依赖它，导出保持单一来源不复制两份"
  - "GetOrSetJSON/SetJSON 薄委托红线执行：仅类型包装委托 p.GetOrSet，P0 #9 同步写/JSON 往返/错误透传零重写"
  - "Invalidate/InvalidatePattern void + warn 日志（pattern=%s 信息量大的格式），nil 防护内聚底层"

patterns-established:
  - "Pattern: base.GetOrSetJSON[T] 单 return 形态——92-02/92-03 共 32 处样板的迁移靶点签名（ctx, p, key, ttl, query func() (T, error)）"
  - "Pattern: type alias 原位翻转迁移——定义删、别名留、引用零改动（alias=同一类型），后续可删"
  - "Pattern: nil-receiver 防护——接口化字段前的 typed-nil panic 排除，一行 if s == nil return default"

requirements-completed: [CACHE-UNIFY-01, CACHE-UNIFY-05]

# Metrics
duration: 31min
completed: 2026-09-05
---

# Phase 92 Plan 01: base 缓存抽象包抽取 + alias 翻转 Summary

**`internal/services/base` 缓存抽象全套落地（TTLResolver 薄基类 + CacheProvider 全家 + 泛型函数族 GetOrSetJSON[T]/SetJSON[T]/Invalidate/InvalidatePattern），system 侧 type alias 翻转实现 20+ 消费文件零改动迁移，nil-receiver 防护排除 typed-nil panic，TestBase92 十用例 miniredis 双装配锁定行为，`go test ./internal/services/...` 20 包全绿零回归**

## Performance

- **Duration:** 31 min
- **Started:** 2026-09-05T03:09:08Z
- **Completed:** 2026-09-05T03:40:00Z
- **Tasks:** 3/3
- **Files modified:** 10（新建 4 + 修改 6）

## Accomplishments

- base 包三生产文件：TTLResolver 接口 + CacheServiceBase{Config TTLResolver} 薄基类 + CacheProvider 9 方法接口 + NoOpCacheProvider + setValue(导出为 SetValue) + CacheStats/CacheEntry + 泛型函数族四件，零反向依赖（grep 证实无 `internal/services` import）
- system/cache_provider.go 129 行定义删除 → 原位 5 个 type alias + `var _ CacheProvider = (*NoOpCacheProvider)(nil)` 编译期断言；api/router.go、6 处 NoOp fallback、duty/knowledge/network/workorder 测试 var _ 断言、core.go 装配全部零改动编译通过（D-02 承诺，git diff 证实保护范围零文件触碰）
- CacheConfigService.GetDurationWithDefault 顶部 nil-receiver 防护：typed-nil interface 场景（~100 处 nil-config 测试构造 + 92-03 将至的 DataCacheService.GetExpiration 委托路径）返回 default 不 panic（Pitfall 1）
- TestBase92_* 十用例：命中/穿透/错误透传（双装配表驱动）、mr.FastForward TTL 过期、Invalidate/InvalidatePattern 失效 + nil provider 静默、NoOp 透传 + 指针类型回填（Pitfall 5 单向改善断言）、GetStats、GetExpiration 零值/typed-nil 端到端——CACHE-UNIFY-05 base 侧测试面进 CI gate
- 回归零污染：`go build ./...` 0 错误、`go test ./internal/services/...` 20 包全绿、operlog/status AST 锁值防线保持绿、InvalidateCacheByPattern/ByKey 双轨期原样保留（仅追加 92-03 待删除注释）

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建 base 缓存抽象包三文件（D-01/D-02/D-03/D-04）** - `5ae394c` (feat)
2. **Task 2: alias 翻转 + CacheServiceBase 迁出 + nil-receiver 防护（D-02 + Pitfall 1）** - `756df90` (feat)
3. **Task 3: base/cache_service_base_test.go 双装配测试（CACHE-UNIFY-05 / D-09）** - `b54a685` (test)
4. **style: gofmt 注释规范** - `2865a3a` (style，零语义变更)

## Files Created/Modified

- `internal/services/base/cache_service_base.go` - TTLResolver 接口 + CacheServiceBase 薄基类 + GetExpiration（48 行，D-01）
- `internal/services/base/cache_provider.go` - CacheProvider 9 方法 + NoOpCacheProvider + SetValue + CacheStats/CacheEntry（146 行，D-02 逐字迁入，唯一偏差 setValue→SetValue）
- `internal/services/base/cache_functions.go` - GetOrSetJSON[T]/SetJSON[T]/Invalidate/InvalidatePattern（95 行，D-03/D-04）
- `internal/services/base/cache_service_base_test.go` - TestBase92 系列十用例双装配测试（326 行，CACHE-UNIFY-05）
- `internal/services/system/cache_provider.go` - 129 行定义 → 23 行 alias 门面（5 alias + 编译期断言）
- `internal/services/system/cache_utils.go` - CacheServiceBase/GetExpiration 定义迁出；InvalidateCache* 保留 + 92-03 待删除注释；filterSlice/contains/paginate 不动
- `internal/services/system/cache_adapter.go` - setValue 调用点 2 处 → base.SetValue（Rule 3）
- `internal/services/cache_config_service.go` - GetDurationWithDefault nil-receiver 防护 +3 行
- `internal/services/system/cache_infra_test.go` / `user_cache_impl_test.go` - 同包测试 setValue 调用点 → base.SetValue（Rule 3）

## Decisions Made

- **LOC 双口径（诚实报告）**：生产代码 +323/-135——base 包投资单列 +289（3 新文件）；system/root 侧净减 -101（cache_provider.go 141→23、cache_utils.go -10，含 +3 防护与 +1 偏差注释）；测试 +326。D-05 的"样板毛减 ≥200"锚点在 92-02/92-03 完成 32 处迁移后于 92-04 收口判定，本 plan 是投资 plan。
- **InvalidateCacheByPattern/ByKey 本 plan 不删**（plan 原文锁定）：42 处调用面由 92-03 与删除动作同 commit 改写，双轨期无失效真空窗口（T-92-02 缓解达成）。
- **键构造零触碰**（T-92-01 缓解达成）：本 plan 仅建抽象未迁任何调用点，缓存键构造零变更；零新依赖（T-92-SC accept 达成）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 私有 setValue 导出为 base.SetValue**
- **Found during:** Task 2（alias 翻转后全量编译）
- **Issue:** plan 未识别到 setValue 的同包消费者——`system/cache_adapter.go`（CacheAdapter，core.go:381 / excel_handler.go:54 生产装配在用）及 cache_infra_test.go / user_cache_impl_test.go 调用它；迁移至 base 后私有符号不可达，`go build` 报 undefined
- **Fix:** base 侧导出为 SetValue（保持单一来源，不复制两份），system 同包 3 文件 4 处调用点机械替换 + 注释更新；导出注释注明"新代码应优先 GetOrSetJSON[T]"
- **Files modified:** internal/services/base/cache_provider.go, internal/services/system/cache_adapter.go, internal/services/system/cache_infra_test.go, internal/services/system/user_cache_impl_test.go
- **Verification:** go build ./... 0 错误；不破坏 D-02 零改动承诺（承诺范围 = api/core/duty/knowledge/network/workorder 外部消费者，git diff 证实零触碰；cache_adapter.go 是 system 同包文件）
- **Committed in:** 756df90（Task 2 commit 内）

### Plan 校准记录（非规则偏差，验收判据与 plan 原文的小口径差）

1. Task 1 验收 `grep -c "GetOrSet(" cache_provider.go >= 3` 实测 **2**：逐字搬迁源（原 system/cache_provider.go）本身仅含 2 处（接口声明 + NoOp 实现）；逐字搬迁红线（D-02）优先，未注水凑数。"接口 + NoOp 均在"的判据实质达成。
2. Task 3 验收 `time.Sleep`/`t.Parallel` grep==0 与"纪律头照抄 79_01"冲突：79_01 模板头部纪律注释本身含这两个字样。措辞改写为"禁并行子测试/禁裸 sleep"——纪律含义完整保留且满足字面验收（实际代码零 sleep、零 parallel）。

---

**Total deviations:** 1 auto-fixed（Rule 3 blocking）+ 2 plan 校准记录
**Impact on plan:** SetValue 导出为编译必需的最小修复，语义零变更（行为与迁移前逐字一致）；无范围 creep，D-02 零改动承诺在保护文件集上完整达成。

## TDD Gate Compliance

Task 3 标记 tdd="true"，但 plan 任务序（Task 1/2 先实现、Task 3 后测试）使字面 RED gate 不可达成——测试对已交付实现首次运行即 PASS（预期行为，非 fail-fast 违例）。plan frontmatter type: execute（非 tdd），plan 级 gate 不适用。Task 3 以 `test(92-01)` 单 commit 交付，作为 CACHE-UNIFY-05 行为锁；无 test→feat RED/GREEN commit 序列，如实记录。

## Issues Encountered

- SetJSON 初稿按"组合写"直觉写成 `_, err := p.GetOrSet(...)`——GetOrSet 仅返回 error，编译器立即纠正为直接 return（Task 1 commit 内修复）
- 全量 `go test ./internal/services/...` 超 600s 前台超时 → 转后台跑完：20 包 ok，exit 0（首次运行零非 ok 行即全绿，第二次为包计数复核）

## User Setup Required

None - no external service configuration required.（零新依赖，miniredis 为既有 test-only 依赖）

## Next Phase Readiness

- **92-02（system 29 处样板迁移）**：靶点签名已冻结——`base.GetOrSetJSON(ctx, p, key, ttl, query func() (T, error))`；30 个嵌入点经 CacheServiceBase alias 零改动保留，`s.GetExpiration(...)` 结果作参传入即达单 return 形态；notice 逃兵归队（删私有 getExpiration + 嵌入基类）
- **92-03（operations + D-04 收尾）**：base.Invalidate/InvalidatePattern 已就绪（nil 防护 + pattern=%s 日志格式已统一）；InvalidateCache* 双轨注释已标注删除归属；DataCacheService.GetExpiration 委托 base 时 nil-receiver 防护已就位
- **92-04（收口）**：D-05 LOC 双口径的"样板毛减"计数以本 plan 后基线为起点；monitor rename（D-08 CacheOperator）与本 plan 无耦合
- 无 blocker；本 plan 未触碰 monitor/operations/root 缓存文件（范围锁定）

## Self-Check: PASSED

- 文件存在性：base 4 新文件 + system/root 6 修改文件全部在盘（build/test 通过即证）
- Commit 存在性：`5ae394c` / `756df90` / `b54a685` / `2865a3a` 经 git log 证实
- 验证项：go build ./... 0 错误；go test ./internal/services/... 20 包全绿；go test ./internal/utils/operlog/ ./internal/models/ AST 防线绿；quick 三包（base/system/operations）绿；git diff 保护范围（api/core/duty/knowledge/network/workorder）零触碰；无意外文件删除（diff-filter=D 空）

---
*Phase: 92-p2*
*Completed: 2026-09-05*

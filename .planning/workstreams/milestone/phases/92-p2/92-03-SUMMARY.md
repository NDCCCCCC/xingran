---
phase: 92-p2
plan: 03
subsystem: backend-cache
tags: [go-generics, cache, refactoring, zero-behavior-change, compiler-driven-deletion]

# Dependency graph
requires:
  - phase: 92-01
    provides: base.GetOrSetJSON[T]/Invalidate/InvalidatePattern 泛型函数族签名（迁移靶点）+ CacheServiceBase alias 门面 + nil-receiver 防护（typed-nil 委托路径）
  - phase: 92-02
    provides: system 侧 21 处失效调用已改写 base 底层（双轨期——InvalidateCache* 删除归属本 plan）
provides:
  - CACHE-UNIFY-03：operations 3 处真实 GetOrSet 样板收敛 base.GetOrSetJSON（:168 注释行假阳性未动）
  - CACHE-UNIFY-04：DataCacheService 原地定性（GetExpiration 委托 base + D-07 双定位注释，不标 @Deprecated，装配链零改动）
  - D-04 完全达成：base.Invalidate/InvalidatePattern 为唯一失效底层，system.InvalidateCacheByPattern/ByKey 已删除（42 处调用全部收敛 base，编译器证明零遗漏）
  - CacheInvalidator 分发器保留 + 底层委托（Excel 管道 entityType 分发语义与 ExcelConfig.CachePatterns 读取链零改动）
affects: [92-04 (monitor rename + invariants 扫描 + CLAUDE.md/REQUIREMENTS/ROADMAP 措辞同步 + D-05 LOC 双口径收口判定)]

# Tech tracking
tech-stack:
  added: 零新增依赖
  patterns: [编译器当 checklist 的原子删除（删除与最后一批调用点改写同 commit）, 组合字面量取地址委托（&base.CacheServiceBase{...}）, 分发器保留 + 底层委托的双层结构]

key-files:
  created: []
  modified:
    - internal/services/operations/floor_cache_impl.go
    - internal/services/operations/cache_invalidator.go
    - internal/core/core.go
    - internal/services/duty/duty_cache_impl.go
    - internal/services/knowledge/knowledge_cache_impl.go
    - internal/services/network/cache_impl.go
    - internal/services/workorder/workorder_cache_impl.go
    - internal/services/system/cache_utils.go
    - internal/services/data_cache_service.go

key-decisions:
  - "D-04 原子性执行：19 处外围失效调用改写与 system.InvalidateCacheByPattern/ByKey 两函数删除落同一 commit（dbc987b），go build ./... 零错误即编译器证明生产代码零遗漏调用者"
  - "A5 范围纪律：duty/knowledge/network/workorder 4 文件的 GetOrSet 样板不迁（12 处留 v1.30+ 候选），只改编译器强制的失效调用行——scope constrainment"
  - "core.go:366 装配链语义保留：system.NewCacheProvider(c.DataCacheService) provider 实参原样搬移，仅失效函数名改写（T-92-03 缓解）；system import 因 CacheKeyMenuTree 等常量引用保留"
  - "floor GetFloorsByBuildingID 闭包内直查 db 查询体逐字搬入 query 闭包（非委托 floorService 的特殊站点）；floor:tree / floor:building:%s 内联键 diff 零字符变更（T-92-01 缓解）"
  - "DataCacheService.GetExpiration 委托用 (&base.CacheServiceBase{Config: s.cacheConfig})——组合字面量不可寻址，指针方法必须显式取地址（plan 片段缺 &，Rule 3 编译必需修复）"

patterns-established:
  - "Pattern: 失效调用改写模板——systemServices.InvalidateCacheByKey(ctx, s.cache, keys, MOD) → base.Invalidate(同实参)，ByPattern → base.InvalidatePattern（1 行机械替换 + import base）"
  - "Pattern: 平行 TTL 逻辑消除模板——保留方法签名，内部委托 base 基类，typed-nil 经 nil-receiver 防护等价兜底"
  - "Pattern: 文件定位注释（D-07）——双定位 + 新代码路径指引，不标 @Deprecated"

requirements-completed: [CACHE-UNIFY-03, CACHE-UNIFY-04]

# Metrics
duration: 20min
completed: 2026-09-05
---

# Phase 92 Plan 03: operations 迁移 + D-04 失效底层归一 + root 定性 Summary

**operations floor 3 处样板收敛 base.GetOrSetJSON + CacheInvalidator 底层委托保留分发语义，D-04 涟漪 19 处外围失效调用与 system.InvalidateCacheByPattern/ByKey 删除同 commit 落地（编译器证明零遗漏），DataCacheService 原地定性（GetExpiration 委托 base + D-07 定位注释不标 @Deprecated），键构造与失效实参逐字保留，`go test ./internal/services/...` 20 包全绿零回归**

## Performance

- **Duration:** 20 min
- **Started:** 2026-09-05T04:12:27Z
- **Completed:** 2026-09-05T04:32:48Z
- **Tasks:** 3/3
- **Files modified:** 9（全部生产代码，零测试文件改动）

## Accomplishments

- floor_cache_impl.go 3 处真实 GetOrSet 样板（GetTree / GetFloorsByBuildingID / SearchFloorOptions :179）迁移 base.GetOrSetJSON 单 return；:167-168 注释行假阳性未被误迁（plan 识别的 grep 假阳性锚点完整保留）；跨包嵌入 struct 与 NewFloorServiceWithCache 构造函数零改动（alias 兼容实证达成）
- CacheInvalidator：struct/构造函数零改动（Excel 导入管道 entityType 分发语义保留），InvalidateByEntityType/InvalidateByPatterns 底层 for 循环委托 base.InvalidatePattern（nil 防护已内聚 base）；循环内成功 Debugf（"清除缓存成功"）随循环消失——日志面变化，无测试断言旧格式
- D-04 收尾：duty 5 / knowledge 4 / network 5 / workorder 4 / core.go 1 共 19 处外围失效调用机械改写 base.Invalidate/base.InvalidatePattern（module 常量、pattern/key 实参、provider 装配逐字保留）+ system/cache_utils.go 删除 InvalidateCacheByPattern/ByKey 两函数，同 commit（dbc987b）；go build ./... 零错误 = 编译器证明零遗漏（42 处 = 92-02 的 21 + 本 plan 的 21 全部收敛 base 唯一底层）
- DataCacheService 原地定性（D-06/D-07）：GetExpiration 签名保留、内部委托 base 基类统一 TTL 解析（typed-nil 经 92-01 nil-receiver 防护返回 default，与原判空语义完全等价）；文件头新增 D-07 双定位注释块（pkg/cache.Cache 业务封装层 + base.CacheProvider 实现底座/Adaptee；新代码走 base.GetOrSetJSON + base.CacheProvider；不标 @Deprecated）；P0 #9 同步写、Get/Set/GetOrSet 主体、CacheKeyBuilder、core 装配链零改动
- 回归零污染：go build ./... 0 错误；go test ./internal/services/... 20 包全绿（SC-4 全量收尾口径）；go test ./internal/core/... 绿（T-92-03 装配链回归）；operlog/status AST 锁值防线绿

## Task Commits

Each task was committed atomically:

1. **Task 1: floor_cache_impl 迁移 + CacheInvalidator 底层委托（CACHE-UNIFY-03 / D-04）** - `f2b59f9` (feat)
2. **Task 2: D-04 涟漪外围 19 处改写 + 删除 InvalidateCacheByPattern/ByKey（编译器驱动收尾）** - `dbc987b` (feat)
3. **Task 3: DataCacheService 原地定性（D-06 GetExpiration 委托 + D-07 定位注释）** - `724dc45` (feat)

## Files Created/Modified

- `internal/services/operations/floor_cache_impl.go` - 3 处 base.GetOrSetJSON + 2 处 base.Invalidate*（+35/-71；键构造零字符变更）
- `internal/services/operations/cache_invalidator.go` - 两方法底层委托 base.InvalidatePattern（+14/-29；struct/构造零改动）
- `internal/core/core.go` - :366 失效调用改写 base.InvalidatePattern（provider 实参原样；+2/-1）
- `internal/services/duty/duty_cache_impl.go` - 5 处失效改写 + import base（+6/-5）
- `internal/services/knowledge/knowledge_cache_impl.go` - 4 处失效改写 + import base（+5/-4）
- `internal/services/network/cache_impl.go` - 5 处失效改写 + import base（+6/-5）
- `internal/services/workorder/workorder_cache_impl.go` - 4 处失效改写 + import base（+5/-4）
- `internal/services/system/cache_utils.go` - InvalidateCacheByPattern/ByKey 删除（D-04 达成标记注释留下）（+3/-24）
- `internal/services/data_cache_service.go` - GetExpiration 委托 + D-07 定位注释块（+24/-4；P0 #9 段零 diff）

## Decisions Made

- **LOC 双口径（诚实报告）**：本 plan 生产代码净减 **-31**（9 文件 +86/-117：样板迁移 -36、失效涟漪删除 -15、data_cache_service 注释投资 +20）。Phase 92 累计：92-01 投资 +289（base 包）+ root/system 侧 -101、92-02 净减 -171、92-03 净减 -31——D-05 "样板毛减 ≥200" 锚点按 plan 归 92-04 收口判定。
- **CacheInvalidator 日志面变化（plan 预告项）**：底层循环委托 base 后，每 pattern 的成功 Debugf（`[%s] 清除缓存成功: pattern=%s`）消失（base.InvalidatePattern 仅失败时 Warnf）；失败 Warnf 格式与 base 统一（同款 pattern=%s 格式，零字符差）。无测试断言旧格式串，Excel 管道失效覆盖面零收窄（T-92-02 缓解达成）。
- **base.GetOrSetJSON 泛型推断**：floor 3 处均由 query 闭包返回类型推断 T（[]FloorTreeNode / []operations.OpsFloor / []DropdownOption），无需显式类型实参——比 plan 写法 base.GetOrSetJSON[[]FloorTreeNode](...) 更简洁且编译等价。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] GetExpiration 委托片段缺取地址——组合字面量不可寻址**
- **Found during:** Task 3（首次编译）
- **Issue:** plan action/key_links 给出的委托形态 `base.CacheServiceBase{Config: s.cacheConfig}.GetExpiration(...)` 无法编译——GetExpiration 是指针接收者方法（`func (b *CacheServiceBase) ...`），组合字面量值不可寻址，Go 报 `cannot call pointer method GetExpiration on base.CacheServiceBase`
- **Fix:** 修正为 `(&base.CacheServiceBase{Config: s.cacheConfig}).GetExpiration(configKey, defaultExpiration)`（对组合字面量取地址合法）；AC 要求的 `base.CacheServiceBase{Config: s.cacheConfig}` 子串在 diff 中完整保留，语义与 plan 意图逐字一致
- **Files modified:** internal/services/data_cache_service.go（仅方法体一行）
- **Verification:** go build ./... 0 错误；TestDcs7901_*/TestCcs7901_*（含 TestCcs7901_GetExpiration_Wired）全绿
- **Committed in:** 724dc45（Task 3 commit 内）

### Plan 校准记录（非规则偏差，验收判据与 plan 原文的小口径差）

1. Task 2 commit subject 因 commitlint subject-case 规则（禁止大写开头）改写为小写起句 "外围 19 处失效调用改写 base 底层并删除 invalidateCacheBy* 旧函数"——旧函数名精确拼写（InvalidateCacheByPattern/InvalidateCacheByKey）在 commit body 与代码注释中完整保留。
2. Task 1 验收 `grep -c "base.InvalidatePattern" cache_invalidator.go == 2` 首轮实测 3（代码调用 2 + 注释说明 1）——注释措辞改写为 "委托 base 泛型函数" 后达成 == 2；调用面与 plan 意图（2 处委托）自始一致。
3. Task 1 验收 "floor:tree 与 floor:building:%s 键构造 git diff 零字符变更" 按字面理解为**键字符串字面量零字符变更**（diff 中该字符串的 -/+ 行内字符串部分逐字符相同，仅所在行位置移动）——已达成本意（Redis 现存键继续命中，T-92-01）。

---

**Total deviations:** 1 auto-fixed（Rule 3 编译必需取地址）+ 3 plan 校准记录
**Impact on plan:** 取地址为编译器强制的最小语法修复，运行语义与 plan 意图逐字一致；全部键构造/失效实参/module 常量/TTL 值逐字保留，零业务行为变更底线（v1.29 D-05）达成。

## Issues Encountered

- 无阻塞问题；commitlint body-max-line-length 与 subject-case 两条 hook 规则要求 commit message 换行与改写（两次重提，零代码影响）
- go test ./internal/services/... 全量 ~7.5 min，按 92-01 先例转后台执行（20 包全 ok，EXIT=0）

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **92-04（收口）**：CACHE-UNIFY-01..05 代码面全部达成；monitor CacheOperator rename（D-08）、invariants 扫描锁（D-10②）、CLAUDE.md/REQUIREMENTS/ROADMAP 措辞同步（D-10①/D-06——CACHE-UNIFY-03/04 的复选框已按 92-01 先例勾选并附实际形态注记，完整措辞修订仍归 92-04）、D-05 LOC 双口径收口判定（本 plan 已提供 92-01/02/03 三段累计数据）。
- **v1.30+ 候选（本 plan 确认的范围外残留）**：duty/knowledge/network/workorder 4 文件 12 处 GetOrSet 样板迁移（A5）；mac_history 接口化后 DataCacheService 字面迁移再评估。
- 无 blocker；monitor/ 与 root 另两个缓存文件（mac_history_cache_decorator/template_cache）本 plan 零触碰（范围锁定）。

## Self-Check: PASSED

- 文件存在性：9 个修改文件全部在盘（build/test 通过即证）
- Commit 存在性：`f2b59f9` / `dbc987b` / `724dc45` 经 git log 证实
- 验证项：go build ./... 0 错误；go test ./internal/services/... 20 包全绿（EXIT=0）；go test ./internal/core/... 绿；go test ./internal/utils/operlog/ ./internal/models/ AST 防线绿；InvalidateCacheByPattern/ByKey 生产代码引用 0（仅注释/文档行）；base.Invalidate 外围 5 文件合计 19（core 1 + duty 5 + knowledge 4 + network 5 + workorder 4）；floor GetOrSet(ctx) 残留 0、base.GetOrSetJSON == 3；cache_utils.go 两函数定义 0；GetOrSet(ctx) system/operations 生产残留 0（adapter.go:32 实现端除外，92-02 校准先例）；无意外文件删除（3 commits 全为修改，无 D 状态）

---
*Phase: 92-p2*
*Completed: 2026-09-05*

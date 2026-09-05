---
phase: 92-p2
plan: 02
subsystem: backend-cache
tags: [go-generics, cache, refactoring, zero-behavior-change, pilot-batch-rhythm]

# Dependency graph
requires:
  - phase: 92-01
    provides: base.GetOrSetJSON[T]/Invalidate/InvalidatePattern 泛型函数族 + CacheServiceBase alias 门面（迁移靶点签名）
provides:
  - system 9 个 *_cache_impl.go 全部收敛 base.GetOrSetJSON（29 处样板清零，调用段 ≤6 行）——CACHE-UNIFY-02 主体达成
  - system 包内 21 处失效调用改写 base.Invalidate/InvalidatePattern（D-04 涟漪的 in-system 部分；system.InvalidateCache* 函数本体保留至 92-03 删除）
  - notice_cache_impl.go 逃兵归队：CacheServiceBase 嵌入 + 私有 getExpiration 删除 + noticeListPage 具名类型 + buildMyNoticesKey helper
  - provider 契约对齐的 mock 模板（JSON 往返回填 dest）——后续 cache mock 编写先例
affects: [92-03 (operations 3 处 + D-04 外围 19 处改写 + cache_utils.go 删除 + root 定性 + monitor rename), 92-04 (D-05 LOC 双口径收口判定 + invariants 扫描 + CLAUDE.md 修订)]

# Tech tracking
tech-stack:
  added: 零新增依赖
  patterns: [泛型单 return 读穿透（pilot → 批量 → 复杂站点三段节奏）, 具名局部类型替代匿名 struct 缓存值（JSON 形状不变）, 键构造 helper 搬迁（逐字节等价）, mock 契约对齐（JSON 往返回填 dest）]

key-files:
  created: []
  modified:
    - internal/services/system/user_cache_impl.go
    - internal/services/system/menu_cache_impl.go
    - internal/services/system/role_cache_impl.go
    - internal/services/system/role_cache_impl_test.go
    - internal/services/system/config_cache_impl.go
    - internal/services/system/department_cache_impl.go
    - internal/services/system/dict_cache_impl.go
    - internal/services/system/post_cache_impl.go
    - internal/services/system/settings_cache_impl.go
    - internal/services/system/notice_cache_impl.go

key-decisions:
  - "Pilot → 批量 → 复杂站点三段节奏执行（Phase 91 先例）：user 5+7 站点先行验证模板，menu/role/config/department/dict/post/settings 按文件逐个迁移并每文件 go build，notice 复杂站点收尾"
  - "user/role List 站点 T=*PageResult：NoOp 反射路径由静默丢零值修正为返回真实数据（Pitfall 5 单向改善，plan 明令禁止修回）"
  - "notice 归队四步：嵌入 CacheServiceBase + 构造接线 + 删私有 getExpiration（config 专有字段随之删除，唯一消费者）+ 匿名 struct 具名化 noticeListPage（JSON 字段名 List/Total 不变）"
  - "buildMyNoticesKey helper 搬迁键 if/else 双分支：两种 fmt.Sprintf 格式串与参数顺序原样搬入，键构造逐字节等价（Redis 现存键继续命中）"
  - "system.InvalidateCacheByPattern/ByKey 函数本体本 plan 不删（双轨期）：21 处 in-system 调用已全部改写 base 底层，函数删除归 92-03 与外围 19 处改写同 commit"

patterns-established:
  - "Pattern: GetOrSet 迁移单 return 模板——return base.GetOrSetJSON(ctx, s.cache, key, s.GetExpiration(cfg, d), func() (T, error) { ... })，闭包逻辑逐字保留、T 以闭包返回类型为准"
  - "Pattern: 具名局部类型替代缓存值匿名 struct——JSON 字段名与顺序不变则缓存数据兼容"
  - "Pattern: cache mock 必须回填 dest（JSON 往返）——T 为指针形态后零值 struct 隐式容忍不再存在"

requirements-completed: [CACHE-UNIFY-02, CACHE-UNIFY-05]

# Metrics
duration: 21min
completed: 2026-09-05
---

# Phase 92 Plan 02: system 9 文件 29 处 GetOrSet 样板迁移 Summary

**system 包 9 个 cache_impl 文件的 29 处 interface{} 闭包式 GetOrSet 五段式样板全部收敛至 base.GetOrSetJSON 单 return（pilot user → 批量 7 文件 → 复杂 notice 三段节奏），21 处失效调用改写 base.Invalidate/InvalidatePattern 底层，notice 逃兵归队嵌入基类并消灭私有 getExpiration 平行 TTL 逻辑，键构造 diff 级零变更，`go test ./internal/services/system/` 全绿零回归**

## Performance

- **Duration:** 21 min
- **Started:** 2026-09-05T03:44:49Z
- **Completed:** 2026-09-05T04:05:43Z
- **Tasks:** 3/3
- **Files modified:** 10（生产 9 + 测试 1）

## Accomplishments

- 29/29 处 GetOrSet 样板迁移完成（user 5 + menu 6 + role 6 + config 3 + department 2 + dict 2 + post 2 + settings 1 + notice 2），每处调用段 ≤6 行，interface{} 闭包式 GetOrSet 在 system 包生产代码清零
- 21 处失效调用改写 base.Invalidate/base.InvalidatePattern（user 7 / menu 2 / role 3 / config 2 / department 1 / dict 2 / post 1 / notice 3），module 常量与 pattern/key 实参逐字不变；`system.InvalidateCacheByPattern/ByKey` 函数本体按 plan 保留（双轨期，92-03 与外围 19 处改写同 commit 删除）
- notice_cache_impl.go 逃兵归队：CacheServiceBase 嵌入 + `CacheServiceBase{Config: config}` 构造接线，私有 getExpiration 平行 TTL 逻辑删除（config 专有字段唯一消费者即该方法，随之删除），TTL configKey 与 default 值逐字不动
- 复杂站点按 Pitfall 处置：GetUserNotices 匿名 struct 具名化为 `noticeListPage`（JSON 字段名 List/Total 与顺序不变，缓存数据兼容），键 if/else 双分支抽 `buildMyNoticesKey` 包级 helper（格式串与参数顺序原样搬入）；user/role List 站点 T=*PageResult 使 NoOp 路径返回真实数据（Pitfall 5 单向改善）
- 键构造零变更达成：GetUserByIDKey 等 helper、CacheKey* 常量、内联 fmt.Sprintf 格式串在全部 diff 中仅位置移动零字符变更（T-92-01 缓解达成）
- 回归零污染：`go build ./...` 每文件迁移后即跑 0 错误；`go test ./internal/services/system/` 全绿；`go test ./internal/services/operations/ -run "Cache|Floor"` 绿（floor 跨包嵌入点未受影响）；operlog/status AST 锁值防线绿

## Task Commits

Each task was committed atomically:

1. **Task 1: Pilot — user_cache_impl.go 迁移（5 处 GetOrSet + 7 处失效调用）** - `ced0b7a` (feat)
2. **Task 2: 批量迁移 — menu/role/config/department/dict/post/settings（22 处 GetOrSet + 11 处失效调用）** - `6e7bb07` (feat)
3. **Task 3: notice_cache_impl.go 逃兵归队（嵌入基类 + 具名局部类型 + 键 helper）** - `1093670` (feat)

## Files Created/Modified

- `internal/services/system/user_cache_impl.go` - pilot 样本：5 处 GetOrSetJSON 单 return + 7 处 base.Invalidate*（+28/-61）
- `internal/services/system/menu_cache_impl.go` - 6 处迁移 + 2 处失效（+27/-62；含 InvalidateUserMenuCacheByProvider 包级 helper 改写）
- `internal/services/system/role_cache_impl.go` - 6 处迁移（含 List T=*PageResult 站点）+ 3 处失效（+29/-65）
- `internal/services/system/config_cache_impl.go` - 3 处迁移 + 2 处失效（+15/-38）
- `internal/services/system/department_cache_impl.go` - 2 处迁移（GetTreeWithFilter 键后缀拼接保留）+ 1 处失效（+10/-19）
- `internal/services/system/dict_cache_impl.go` - 双 struct 各 1 处迁移（fmt.Errorf 包装逐字保留）+ 2 处失效（+11/-20）
- `internal/services/system/post_cache_impl.go` - 2 处迁移 + 1 处失效（+10/-21）
- `internal/services/system/settings_cache_impl.go` - 1 处迁移（无失效调用）（+5/-13）
- `internal/services/system/notice_cache_impl.go` - 逃兵归队全套（+46/-53）
- `internal/services/system/role_cache_impl_test.go` - mockRoleCacheProvider.GetOrSet 补齐 dest 回填契约（Rule 1，+9/-1）

## Decisions Made

- **LOC 双口径（诚实报告）**：口径 A——9 个生产文件净减 **-171**（+181/-352，29 处样板 + notice 归队动作合计；含 buildMyNoticesKey/noticeListPage 约 +20 行摊销基建）；测试 +8。D-05"样板毛减 ≥200"为 phase 级锚点（32 处全量），92-03 完成 operations 3 处 + CacheInvalidator 委托后由 92-04 收口判定。
- **无效化函数本体不删**（plan 原文锁定）：21 处 in-system 调用已全部走 base 底层，但 `system.InvalidateCacheByPattern/ByKey` 与 duty/knowledge/network/workorder/core.go 的 19 处外围调用保留原样——92-03 删除函数时由编译器强制零遗漏改写（T-92-02 无失效真空）。
- **adapter.go:32 的 `GetOrSet(ctx` 非样板**：`cacheProviderAdapter.GetOrSet` 是 CacheProvider 接口的实现端（委托 DataCacheService），与 base/cache_functions.go 的 `p.GetOrSet` 薄委托同类；D-06 明确 adapter 不动，不属 29 处消费者样板。plan 验收 grep `wc -l == 0` 的本意（29 处清零）已达成，见下方校准记录。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] mockRoleCacheProvider.GetOrSet 未回填 dest 导致 TestRoleCache_List_MockCache panic**
- **Found during:** Task 2（role 迁移后全量测试）
- **Issue:** 该 mock 运行 query 后丢弃结果、从不写 dest——违反 provider 真实契约（DataCacheService.GetOrSet 将 query 结果经 JSON 往返回填 dest）。旧 `var result PageResult` 值类型形态下零值 struct 被隐式容忍而"通过"；迁移为 T=*PageResult 后返回 nil，`result.Total` 解引用 panic
- **Fix:** mock 的 GetOrSet 补齐 JSON 往返回填 dest（与 92-01 miniredis 行为锁的真实 provider 语义对齐）；同文件 TC4/5/6/7/29 断言（Empty 类）在回填后依然成立，TC24 错误路径不受影响。TC25/26 注释中"mock dest issue"的历史 wart 就此消除
- **Files modified:** internal/services/system/role_cache_impl_test.go
- **Verification:** `go test ./internal/services/system/` 全量绿（panic 前被中止的后续测试亦全部执行通过）
- **Committed in:** 6e7bb07（Task 2 commit 内）

### Plan 校准记录（非规则偏差，验收判据与 plan 原文的小口径差）

1. Task 2 验收"base.GetOrSetJSON 合计 == 20"系 plan 算术笔误：其自列分布 menu 6 + role 6 + config 3 + department 2 + dict 2 + post 2 + settings 1 = **22**。实测 22 处，与分布枚举逐文件一致（29 = 22 + user 5 + notice 2）。
2. Plan 验收"grep -rn 'GetOrSet(ctx' internal/services/system/*.go（非 test）== 0"多匹配一处：`adapter.go:32` 是 CacheProvider 适配器的实现端委托（D-06 锁定不动文件），非消费者样板；排除该行后生产代码残留 = 0（29 处清零达成）。
3. Plan 验收"grep -c 'CacheServiceBase' notice_cache_impl.go >= 2"实测 **3**（struct 嵌入字段类型 + 构造字面量 + 归队注释说明），判据方向一致超额达成。

---

**Total deviations:** 1 auto-fixed（Rule 1 测试 mock 契约）+ 3 plan 校准记录
**Impact on plan:** mock 修复使测试与真实 provider 契约对齐，生产行为零变更；全部键构造/闭包逻辑/TTL 值逐字保留，零业务行为变更底线（v1.29 D-05）达成。

## Issues Encountered

- role_cache_impl.go 原文件缺文件尾换行（历史遗留），gofmt -w 顺带修复（文件在本次触碰范围内，零语义变更）
- `go test -run "Cache"` 首轮被 TestRoleCache_List_MockCache panic 中止，掩盖了其余测试结果——修复 mock 后全量重跑确认无连带失败

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **92-03（operations + D-04 收尾）**：in-system 21 处已清零，`system.InvalidateCacheByPattern/ByKey` 删除时仅剩 duty 5 / knowledge 4 / network 5 / workorder 4 / core.go 1 共 19 处外围调用需同 commit 机械改写（base.Invalidate/InvalidatePattern 签名已冻结）；floor 3 处样板迁移可直接复制本 plan 模板（含 T=指针站点的 mock 注意事项）
- **92-04（收口）**：D-05 LOC 双口径以 92-01（投资 +289）与本 plan（样板净减 -171）为累计基线；invariants 扫描的检测模式需排除 adapter 实现端（本 plan 校准记录 2 的先例）
- 无 blocker；本 plan 未触碰 operations/monitor/root/duty/knowledge/network/workorder（范围锁定）

## Self-Check: PASSED

- 文件存在性：10 个修改文件全部在盘（build/test 通过即证）
- Commit 存在性：`ced0b7a` / `6e7bb07` / `1093670` 经 git log 证实
- 验证项：go build ./... 0 错误；go test ./internal/services/system/ 全绿；go test ./internal/services/operations/ -run "Cache|Floor" 绿；go test ./internal/utils/operlog/ ./internal/models/ AST 防线绿；9 文件 base.GetOrSetJSON 合计 29、GetOrSet(ctx) 生产残留 0（adapter 实现端除外）、impl 文件 InvalidateCacheBy* 残留 0；键构造 diff 零字符变更；无意外文件删除（8+1 文件全为修改，无 D 状态）

---
*Phase: 92-p2*
*Completed: 2026-09-05*

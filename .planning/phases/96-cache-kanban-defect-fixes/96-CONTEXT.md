# Phase 96: 确定性缓存/看板缺陷修复 - Context

**Gathered:** 2026-09-06
**Status:** Ready for planning

<domain>
## Phase Boundary

修复 CACHEDEF-01..05 五项缓存确定性缺陷（失效不命中 / 键污染 / 前缀剥离恒等），每项附行为级回归测试；另将 JOBSTAT-01 重定性为**死代码删除处置**——删除 `internal/api/v1/job_utils.go` 整文件（GetJobStatistics + FormatDuration，均无生产调用方）连同对应测试，原时区日界缺陷随删除消解。无新功能，无 D-03 设计决策（v1.30 的设计决策项全在 Phase 97/99）。

Phase 96↔98 边界（ROADMAP 已锁）：duty/workorder cache_impl 中的 interface{} 闭包 GetOrSet 迁 `base.GetOrSetJSON[T]` 归 Phase 98；本相只修缺陷行，不顺手迁移。

</domain>

<decisions>
## Implementation Decisions

### CACHEDEF-01 — 部门下拉缓存失效方向
- **D-01: 写键侧独立键。** `GetSelectDataWithCache`（department_cache_impl.go:75-76）写键从裸键 `CacheKeyDeptTree`（`dept:tree`）改为 `BuildDeptCacheKey("tree:select")`（→ `cache:tree:select`），与既有失效模式 `BuildDeptCacheKey("tree:select")+"*"`（同文件 :99）精确匹配——恢复原始设计意图。与生产路径 `GetTreeWithFilter` 的 `cache:dept:tree`（:42）互不干扰。

### CACHEDEF-02 — config 详情缓存失效补齐
- **D-02: `InvalidateConfigCache` 签名加 id 参数。** 签名改 `(ctx, id, configKey)`，keys 列表补 `fmt.Sprintf("config:id:%s", id)`。该方法无接口暴露、唯一调用方是 Delete（config 对象已在手），签名变更零接口影响。

### CACHEDEF-03 — duty 月份解析修复
- **D-03: 修复 `parseInt` 语义为解析全部数字位**（删除 `len(s) >= 4` 前置；惯用实现 `strconv.Atoi` + 失败回退 0）。修复后 `parseInt("07")=7` 直接命中写键——已侦察确认写键（duty_cache_impl.go:153）与失效键（:307）均为 `duty:monthly:%d:%d` 同格式、双方消费 int，无二次失配。

### CACHEDEF-04 — workorder 待办缓存键补 Limit 维度
- **D-04: 显式恒定格式。** 键改 `workorder:my_pending:<userID>:limit:<N>`，Limit=0（含 nil req，按 0 处理）也显式写 0，无格式分支。TTL 仅 2 分钟，无旧键兼容负担。

### CACHEDEF-05 — 缓存监控前缀剥离
- **D-05: 修 `normalizeCacheKeyForService`（monitor/cache_service.go:766-771）**：`len(key) > 6 && key[:6] == "xingran:"`（6 字节切片比 8 字节字面量恒 false）改 `strings.HasPrefix` + `strings.TrimPrefix`（幂等单次剥离，无双重剥离风险）。
- **D-06: 调用面核查已完成（discuss 阶段）**：全部 6 个调用点（:295 列表显示 / :397 键详情 / :470 del / :478 exists / :489 expire / :536 batch del）均对齐修复意图，无调用方依赖恒等行为；不带前缀键经 HasPrefix 不匹配原样透传，无回归。research 只需补一项：确认 :295 `displayKey` 完整数据流不回流 provider 调用。

### JOBSTAT-01 — 重定性：死代码删除处置（原修复方案作废）
- **D-07: 删除整个 `internal/api/v1/job_utils.go`**（GetJobStatistics + FormatDuration）+ `api_v1_tail_80_03_test.go` 中对应两组测试（`TestJbu8003_FormatDuration` / `TestJbu8003_GetJobStatistics`），文件头注释同步修订。
- **D-08: 证据链（support v1.30 D-01 账目修订 18→17）**：全仓无生产调用方（仅定义 + 测试引用）；源自初始脚手架 ea528c6（2026-08-12）从未接线；生产看板实际走 `/monitor/jobs/logs/statistics` → `jobLogService.Statistics`（job_log_service.go:135，全时段统计无「今日」语义），「生产看板缺陷」前提不成立。该文件内 ws_notice 回归测试组（WSNOTICE-01 防线）与 router 装配测试组**必须保留**。
- REQUIREMENTS.md / ROADMAP.md 账目已同步（workstream live + 根摘要四处），执行时 JOBSTAT-01 完成态 = 文件删除 + build 通过。

### 回归测试纪律（v1.30 D-02 细化）
- **D-09: 行为级 + 边界负例。** 每 CACHEDEF 修复断言修复后完整行为链（真 cache 引擎 + sqlite 内存库，如：写键→变更→失效→读回新数据），另加边界/负例：CACHEDEF-05 不带前缀键原样透传、CACHEDEF-04 不同 limit 键互异、CACHEDEF-01/03 失效后读回新数据。测试同时喂养七 gate 的 coverage ≥77.5 指标。JOBSTAT-01 因删除处置不附修复测试（删除即回归防线：build 失败即报警）。

### Claude's Discretion
- parseInt 具体实现（Atoi 回退 vs 手写循环去前置）——语义锁定为「解析全部数字位」即可
- 测试文件命名与放置（沿项目 `*_test.go` 同包惯例）
- JOBSTAT 删除提交的粒度（可独立 commit 便于回溯）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 缺陷登记与来源链
- `.planning/workstreams/milestone/REQUIREMENTS.md` — CACHEDEF-01..05 + JOBSTAT-01 需求定义（含 JOBSTAT-01 删除处置重定性）与 v1.30 锁定决策 D-01..D-05
- `.planning/workstreams/milestone/ROADMAP.md` §Phase 96 — Goal / Success Criteria / Notes（已同步删除处置）
- `.planning/milestones/v1.29-REQUIREMENTS.md` §V130-CANDIDATES — 缺陷登记原文（含「不得顺手修复」的 v1.29 约束背景）
- `.planning/milestones/v1.29-phases/92-p2/92-REVIEW.md` §WR-01..WR-05 — 五项缓存缺陷的发现审查记录

### 项目约定（CLAUDE.md 相关章节）
- `CLAUDE.md` §Cache Service Convention — base.GetOrSetJSON[T] 唯一权威、键构造单源 `internal/services/system/cache_keys.go`、xingran: 前缀自动添加语义
- `CLAUDE.md` §Compilation & Build Verification — 每步改码后 `go build ./...`

### 代码参照（修复现场）
- `internal/services/system/department_cache_impl.go` — CACHEDEF-01 现场（:42 生产写键范式 / :75 缺陷写键 / :95 失效模式）
- `internal/services/system/config_cache_impl.go` — CACHEDEF-02 现场（:39 读键 / :71 失效 / :105 Delete）
- `internal/services/duty/duty_cache_impl.go` — CACHEDEF-03 现场（:123-124/:187 parseInt 调用 / :153 写键 / :307 失效键 / :333 parseInt）
- `internal/services/workorder/workorder_cache_impl.go` — CACHEDEF-04 现场（:210 GetMyPending）
- `internal/services/monitor/cache_service.go` — CACHEDEF-05 现场（:766 normalize + 6 调用点）
- `internal/api/v1/job_utils.go` + `internal/api/v1/api_v1_tail_80_03_test.go` — JOBSTAT-01 删除对象

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/services/system/cache_keys.go` — 键常量 + Build*CacheKey 构造器 + :253 TTL 归类 switch（CACHEDEF-01 修复直接复用 `BuildDeptCacheKey`）
- `internal/services/base` 泛型函数族 `GetOrSetJSON[T]` — 全部写读路径已在该抽象上，修复只动键/失效参数
- `pkg/cache` memory 引擎 — 回归测试可用的真 cache 实现（无需 Redis）
- `internal/services/scheduler/job_log_statistics_test.go` — sqlite 内存库 + 建表 seed 的测试范式参照

### Established Patterns
- Handler-Service + `*_cache_impl.go` 装饰器模式：失效方法内聚在 cache impl，修复不触 handler
- 回归测试同包 `*_test.go` 惯例（operlog/regression_test.go、status_constants_test.go 同款纪律）
- `base.Invalidate` / `base.InvalidatePattern` 自带 nil-guard + 统一日志，失效修复只需改键列表

### Integration Points
- CACHEDEF-02 签名变更仅 :117 一个调用方（已核实无接口暴露）
- Phase 98 将再触 duty/workorder cache_impl——本相修复行越少，98 迁移冲突面越小（ROADMAP 先后序已锁）

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

- **RPA 统计前端契约**：`rpaApi.ts:605` 期望 `/rpa/statistics/tasks` 返回 `todayExecuted/todayFailed`，后端 `ExecutionStatisticsResult` 无今日字段——前端契约问题，移交 **Phase 100（FEFIX 前端契约修复）** discuss 时决定是否纳入 V130R-10..12 范围（本次 discuss 已裁定：记 deferred，不动 v1.30 REQUIREMENTS 账目）
- **生产看板「今日成功/失败」统计能力**：当前 `jobLogService.Statistics` 为全时段口径、无今日语义——如需今日统计属**新功能**（非缺陷治理），不属 v1.30，记 backlog 候选

</deferred>

---

*Phase: 96-cache-kanban-defect-fixes*
*Context gathered: 2026-09-06*

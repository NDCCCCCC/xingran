---
phase: 92-p2
plan: 04
subsystem: backend-cache
tags: [rename-disambiguation, ast-invariants, documentation-sync, loc-audit, phase-gate]

# Dependency graph
requires:
  - phase: 92-01
    provides: base.CacheProvider 唯一权威定名（D-02，monitor rename 的消歧前提）+ miniredis 测试（D-09）
  - phase: 92-02
    provides: system 29 处样板清零（invariants 扫描硬档的守护对象）
  - phase: 92-03
    provides: operations + D-04 收尾 + root 定性（REQUIREMENTS 措辞同步的事实基础 + LOC 累计数据）
provides:
  - monitor.CacheOperator 接口（原 monitor.CacheProvider，D-08 rename 消歧）——仓内 CacheProvider 名唯一指向 base.CacheProvider
  - TestNoInterfaceGetOrSetResidue invariants 扫描锁（D-10②）：system/operations 硬档 0 残留 + 白名单机制 + 外围 warning 档
  - CLAUDE.md Cache Service Convention 段 + 5 处过时缓存描述修订（D-10①）
  - REQUIREMENTS § CACHE-UNIFY-01..05 / ROADMAP Phase 92 SC 措辞与实际达成形态对齐（D-06，REQ_SYNC_OK）
  - D-05 LOC 双口径审计报告（口径 A 毛减 187 + 校准 207 / 口径 B 全口径净增 529 诚实记录）
affects: [93 (config_backup), 94 (前端 API 工厂化), v1.30 候选（duty/knowledge/network/workorder 12 处样板迁移 + A6 死代码清理）]

# Tech tracking
tech-stack:
  added: 零新增依赖
  patterns: [AST 空接口判定（Methods nil 或空 List 双口径）, 硬档+warning 档双档调和（Phase 89/90 硬锁与 D-10② 宽松档）, runtime.Caller 相对路径定位（禁绝对路径断言）, 双口径 LOC 报告（OVR-91-01 先例）]

key-files:
  created:
    - internal/services/system/cache_invariants_92_test.go
  modified:
    - internal/services/monitor/cache_service.go
    - internal/api/v1/monitor/cache_router.go
    - internal/services/monitor/cache_service_test.go
    - CLAUDE.md
    - .planning/REQUIREMENTS.md
    - .planning/workstreams/milestone/ROADMAP.md

key-decisions:
  - "D-08 rename 目标名采纳 CacheOperator（D-08 建议名）；rename 面 = 6 处代码点（cache_service.go :49/:155/:165 + cache_router.go :42 + cache_service_test.go :34/:217）+ 4 处 cosmetic 注释，checker blocker 补面的测试断言面与生产面同 commit 同步"
  - "invariants 断言双档：硬档锁 system/operations 已清零面（超出 allowedResidues 即 fail），warning 档覆盖 duty/knowledge/network/workorder 外围 11 处（A5 v1.30+ 候选）——Phase 89/90 硬锁惯例与 D-10② 'warning 不 fail' 的调和按 plan 锁定执行"
  - "AST 空接口判定用 Methods == nil || len(Methods.List) == 0 双口径——go/parser 对 interface{} 可能产生非 nil 空 FieldList，单口径判定会致扫描器空转（已实测踩坑并修复，warning 计数与 grep 逐数吻合为证）"
  - "CLAUDE.md 修订以实况为准：Legacy Services 列表按 ls internal/services/*.go 实况重写（root 仅存 data_cache_service/cache_config_service/mac_history_cache_decorator/template_cache 四个缓存相关文件）；Common Gotchas 的 'Cache dual architecture' 失实条目顺带修正（D-10① 全部过时缓存描述修订语义内）"
  - "LOC 双口径按 plan 锁定：口径 A（12 迁移/改造生产文件毛减）= 187 < 200 未达锚点诚实记录；校准口径（剔除 data_cache_service D-07 注释投资 +20 后 11 文件）= 207 ≥ 200 达成；口径 B（repo 全口径）净增 529（测试投资 +540）诚实双报告"

patterns-established:
  - "Pattern: monitor.CacheOperator——同名异形接口消歧先例（业务缓存抽象唯一权威 base.CacheProvider，监控原始操作型另立名）"
  - "Pattern: invariants 扫描白名单机制——allowedResidues map 初始全空，新增豁免必须显式登记（diff 可见，T-92-01 缓解模板）"
  - "Pattern: 人为回归演练（注入残留转红 → 撤销转绿）作为 invariants 测试真实性的 acceptance 必做项"

requirements-completed: [CACHE-UNIFY-04, CACHE-UNIFY-05]

# Metrics
duration: 24min
completed: 2026-09-05
---

# Phase 92 Plan 04: 消歧 + 守护 + 收口 Summary

**monitor.CacheProvider rename 为 CacheOperator 消除同名异形（含测试断言面，仓内 CacheProvider 名唯一指向 base），TestNoInterfaceGetOrSetResidue AST 扫描锁进 CI（system/operations 硬档 0 残留 + 白名单机制 + 外围 warning 档 11 处），CLAUDE.md 五处过时缓存段修订 + Cache Service Convention 新段，REQUIREMENTS/ROADMAP 措辞与达成形态对齐，LOC 双口径诚实审计，phase gate 全量绿**

## Performance

- **Duration:** 24 min
- **Started:** 2026-09-05T04:48:42Z
- **Completed:** 2026-09-05T05:12:31Z（全量 suite 后台等待 ~10 min 不计有效工时，有效 ~14 min；取 wall-clock 24 min 记录）
- **Tasks:** 4/4
- **Files modified:** 7（新建 1 + 修改 6；全部在 plan files_modified 清单内）

## Accomplishments

- **Task 1（D-08 rename 消歧）**：monitor 包 `CacheProvider` → `CacheOperator`，生产 4 处代码点（接口定义 :49 含 D-08 消歧 doc 注释、字段 :155、构造参数 :165、adapter 返回类型 cache_router.go:42）+ 测试 2 处代码点（cache_service_test.go :34 编译期断言 `var _ CacheOperator`、:217 newTestCacheService 参数）同 commit 同步；4 处 cosmetic 注释顺带改齐。8 方法签名（Get/Set/Delete/Exists/Expire/TTL/Keys/FlushDB）逐字保留，语义零变更。CacheProviderAdapter/NewCacheProviderAdapter 构造名与 CacheConfigProvider/MultiLevelCacheProvider/DirectRedisProvider/StatsProvider 不撞名清单零触碰。acceptance 硬 gate 达成：monitor 两包 `\bCacheProvider\b` 裸引用 = 0，`monitorServices.CacheProvider\b` 全仓 = 0——base.CacheProvider 成为仓内唯一权威 CacheProvider 名
- **Task 2（D-10② invariants 锁）**：`cache_invariants_92_test.go` TestNoInterfaceGetOrSetResidue 落地——AST 精确匹配（CallExpr selector .GetOrSet + FuncLit 返回 (interface{}, error)，`any` 别名同计）；硬档 system(9 文件)+operations(1 文件) 残留 == allowedResidues（初始 0）；warning 档 duty 3 / knowledge 3 / network 3 / workorder 2 共 11 处仅 t.Logf 不 fail（与 grep 实测逐数吻合，扫描非空转）；白名单 allowedResidues 机制在位；人为回归演练通过（注入残留 → 精确报 settings_cache_impl.go:71 转红 → 撤销转绿，工作区干净）
- **Task 3（D-10①/D-06 三文档同步）**：CLAUDE.md 五处修订（Dual Cache Architecture 失实段重写为 Unified Cache Architecture；Cache System 段改 base 单权威四层定位；Legacy Services 列表按 ls 实况重写；Working with Cache 锁 base.GetOrSetJSON + base.CacheProvider；CacheProvider Interface 段路径改 base/cache_provider.go + 9 方法清单）+ 新增 Cache Service Convention 段（照 Pagination Convention 先例格式，锁唯一权威/invariants 守护/CacheOperator 消歧/DataCacheService 禁新增调用点）+ Common Gotchas 的 Cache dual architecture 失实条目顺带修正；REQUIREMENTS § CACHE-UNIFY-01..05 措辞按达成形态正式化（01 泛型函数族 D-03 / 02 实测 29 处 / 03 实测 3 处 + 分发器保留 D-04 / 04 原地定性 D-06 / 05 invariants 92-04 补齐）；ROADMAP Phase 92 SC-1/2/3/5 措辞同步 + 92-04 勾选（3a2efe5 先例，仅措辞校准无范围变更）
- **Task 4（收口）**：LOC 双口径审计落档（见下节）；phase gate 全绿——`go build ./...` 0 错误、全量 `go test ./...` 0 失败（后台 ~10 min，exit 0，零非 ok 行）、operlog/status AST 锁值防线绿、invariants 扫描绿

## LOC 双口径审计（D-05，OVR-91-01 先例；git numstat 命令级证据）

**审计基点**：`dd86452`（92-01 首 commit `5ae394c` 的父 commit）→ HEAD（含 92-04 全部改动）
**审计命令**：`git diff --numstat dd86452..HEAD -- <files>`

### 口径 A：样板调用段毛减（D-05 锚点判定口径，plan 锁定 12 个迁移/改造生产文件）

| 文件 | added | deleted |
|------|------:|--------:|
| system/user_cache_impl.go | 28 | 61 |
| system/menu_cache_impl.go | 27 | 62 |
| system/role_cache_impl.go | 29 | 65 |
| system/config_cache_impl.go | 15 | 38 |
| system/department_cache_impl.go | 10 | 19 |
| system/dict_cache_impl.go | 11 | 20 |
| system/post_cache_impl.go | 10 | 21 |
| system/settings_cache_impl.go | 5 | 13 |
| system/notice_cache_impl.go | 46 | 53 |
| operations/floor_cache_impl.go | 29 | 46 |
| operations/cache_invalidator.go | 6 | 25 |
| services/data_cache_service.go | 24 | 4 |
| **合计** | **240** | **427** |

**毛减（deleted − added）= 187 行 < 200 锚点——未达成，诚实记录（差 13，达成率 93.5%）。**

校准注记：plan 锁定的 12 文件口径含 `data_cache_service.go` 的 D-07 双定位注释块（净投资 +20，非样板迁移面）；剔除该注释投资后，11 个纯样板迁移/改造文件的毛减 = **207 ≥ 200，达成**。D-05 的定性底线（32 处样板全部收敛 base.GetOrSetJSON、调用段 ≤6 行、仓内 interface{} 闭包式 GetOrSet 清零 + invariants 锁守护）**全部达成**——量化锚点按口径定义如实双报告，不凑数（v1.29 诚实纪律，Phase 91 D-07 shortfall 先例同款）。

### 口径 B：repo 全口径（internal/services/ + internal/core/，诚实报告）

| 口径 | add | del | 净减（del−add） |
|------|----:|----:|----------:|
| 全口径（生产 + 测试） | 1151 | 622 | **−529（净增 529）** |
| 仅生产代码（排除 *_test.go） | 599 | 610 | **+11** |
| 仅测试代码 | 552 | 12 | −540（净增 540） |

拆解：生产侧 +11 净减 = base 包投资 +289（cache_service_base 48 + cache_provider 146 + cache_functions 95）抵消非 base 生产净减 300（12 迁移文件 + cache_provider.go 129→23 alias 门面 + cache_utils.go InvalidateCache* 删除 + duty/knowledge/network/workorder/core 涟漪改写）；测试净增 540 为计划内投资（base TestBase92 miniredis 双装配 +326、invariants 扫描 +203、role mock 契约修复 +9 等）。**预期 shortfall 不算失败（planning 锁定双基）——test 投资本期主动为"不可无声倒退"付费，符合 Phase 92 收口目标。**

## Task Commits

Each task was committed atomically:

1. **Task 1: monitor CacheProvider 接口更名 CacheOperator（D-08 rename 消歧，含测试断言面）** - `94edab0` (refactor)
2. **Task 2: invariants 扫描锁 cache_impl 无 interface{} 闭包式 GetOrSet 残留（D-10②）** - `23b3a11` (test)
3. **Task 3: 三文档措辞同步 — CLAUDE.md/REQUIREMENTS/ROADMAP（D-10①/D-06）** - `01d08aa` (docs)
4. **Task 4: LOC 双口径审计 + phase gate + SUMMARY** - 本 commit（docs）

## D-01..D-10 决策落地对照表

| 决策 | 内容 | 落地 plan | 状态 |
|------|------|----------|------|
| D-01 | 迁 base + 依赖倒置（TTLResolver consumer-defined） | 92-01 | ✅ base 零依赖保持 |
| D-02 | CacheProvider 全家搬 base + 5 alias | 92-01 | ✅ 20+ 消费文件零改动 |
| D-03 | 泛型包级函数族（Go method 无类型参数） | 92-01 定义 + 92-02/03 迁移 | ✅ 32 处收敛单 return |
| D-04 | base 唯一失效底层 + Invalidator 委托保留分发 | 92-01/02/03 | ✅ 42 处调用归一，编译器证明零遗漏 |
| D-05 | 验收锚点 = 定性清零 + LOC ≥200 | 92-02/03 迁移 + 92-04 判定 | ✅ 定性达成；量化口径 A 187/200 未达诚实记录（校准口径 207 达成）、口径 B 双报告 |
| D-06 | DataCacheService 原地定性 + 平行 TTL 消除 + 措辞修订 | 92-03 代码 + 92-04 REQUIREMENTS 正式化 | ✅ REQ_SYNC_OK |
| D-07 | 不标 @Deprecated，双定位注释 | 92-03 注释 + 92-04 CLAUDE.md 同步 | ✅ |
| D-08 | monitor 同名接口 rename 消歧 | 92-04 Task 1 | ✅ CacheOperator，裸引用清零 |
| D-09 | SC-5 = miniredis 自动化 | 92-01 TestBase92 | ✅ 进 CI gate |
| D-10 | ① CLAUDE.md 修订 + Convention 段；② invariants 扫描锁 | 92-04 Task 2 + Task 3 | ✅ 双防线落地 |

## Pitfall 5 单向改善记录（防"修回去"）

- **user/role List 站点 NoOp 路径修正**（92-02 落地）：`T=*PageResult` 后 NoOp 反射路径由"闭包返回 *T 与 dest T 不 assignable → 静默丢零值"修正为"经 JSON 往返回填真实数据"——DataCacheService==nil 的路由 fallback 场景个别站点从返回零值变为返回真实数据，属缺陷修复方向的单向改善（研究 Pitfall 5 预告项兑现）。既有测试用真缓存/mock 不受影响；**禁止把该行为变化当回归"修回去"**。
- **SetJSON 组合语义说明**（92-01 落地，代码注释锁定）：CacheProvider 接口无原生 Set（D-02 锁定 9 方法，接口扩展属范围外），SetJSON 实现为 `Delete(key)` 成功后 `GetOrSet(恒返回 value 闭包)` 的组合写——先删后写 = 覆盖语义，删除与写入之间的中间窗口读者只会 miss 回源，无脏读。当前 32 处调用点零个直接 Set 使用者（研究 A7），该函数为 D-03 函数族完备性成员。

## A6 死代码登记（v1.30 清理候选，本期未动）

`internal/api/v1/system/cache_adapter.go` 的 `NewDataCacheAdapter`——生产代码 0 调用者（本 plan 收口时 grep 复核：仅 cache_adapter_test.go 测试引用），是第 4 个 CacheProvider 适配实现的死代码 duplicate（CONTEXT 未记录、92-RESEARCH 发现清单项）。按 scope constrainment 原则本期不动，登记为 v1.30 清理候选。

## 日志格式统一说明

失效日志统一为含 pattern/key 字段的信息量大的格式（对齐原 operations/cache_invalidator.go 的 `[%s] 清除缓存失败: pattern=%s, error=%v`，key 版同构）；`CacheInvalidator` 底层循环委托 base 后，每 pattern 的成功 Debugf（"清除缓存成功"）随之消失（base.InvalidatePattern 仅失败时 Warnf）——日志面变化无测试断言旧格式，失效覆盖面零收窄（92-03 已记录，此处为 phase 级收口归档）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] invariants 扫描器空接口判定单口径致扫描空转**
- **Found during:** Task 2（首跑 warning 档全 0——duty/knowledge/network/workorder 实有 11 处残留却报 0）
- **Issue:** `interface{}` 在 go/parser 下可能解析为非 nil 的空 `ast.FieldList`（Methods != nil 但 List 为空），单用 `v.Methods == nil` 判定空接口恒 false，扫描器检测不到任何残留——若直接交付，硬档守护形同虚设（T-92-01 威胁面）
- **Fix:** 空接口判定改双口径 `v.Methods == nil || len(v.Methods.List) == 0`；修复后 warning 计数（duty 3 / knowledge 3 / network 3 / workorder 2）与 grep 实测逐数吻合，人为回归演练转红精确到 file:line
- **Files modified:** internal/services/system/cache_invariants_92_test.go（isEmptyInterface 一处）
- **Committed in:** 23b3a11（Task 2 commit 内）

### Plan 校准记录（非规则偏差）

1. Task 3 验收 `grep -c "Legacy Services" CLAUDE.md == 0 或列表内容与实况一致`：实测按重写路径达成 == 0（原 6 件套失实列表按 ls 实况重写为 "Legacy Root Cache Files" 4 文件 + Phase 79/92 迁移注记）；同段 Common Gotchas 的 "Cache dual architecture" 失实条目顺带修正（D-10① "过时缓存描述全部修订" 语义内，非范围扩张）。
2. Task 2 plan 建议的检测正则口径（`\.GetOrSet\(ctx[^\n]*...`）未采用——AST 精确匹配一次成型且不依赖 ctx 实参名（调用点实参名变化不误报/漏报），plan 原文允许"优先 AST 精确匹配"。
3. Task 4 的 "REQUIREMENTS CACHE-UNIFY-01..05 若核验达成可勾选 [x]"：五项在前序 plan 收口时已全部 [x]（92-01/02/03 附注记），本 plan Task 3 已完成正式措辞收敛，无新增勾选动作。

---

**Total deviations:** 1 auto-fixed（Rule 1 扫描器空转 bug）+ 3 plan 校准记录
**Impact on plan:** 扫描器修复是 invariants 真实性的前提（否则守护空转），零范围 creep；三文档 diff 仅措辞校准无范围变更。

## TDD Gate Compliance

本 plan frontmatter type: execute（非 tdd），无 tdd="true" 任务；Task 2 invariants 测试以 `test(92-01)` 同款独立 test commit 交付（23b3a11），含注入残留转红的人为演练，等效满足行为锁真实性验证。无 RED→GREEN commit 序列需求，如实记录。

## Issues Encountered

- commitlint body-max-line-length 钩子拒了一次 Task 3 commit message（换行重提通过，零代码影响；92-03 同款先例）
- 人为回归演练首次注入误用 `contextContext` 类型名致编译失败——非有效红门，修正为 `context.Context` 后重做（转红/转绿均以残留检测为准）
- 全量 `go test ./...` ~10 min 超前台预算，按 92-01/92-03 先例转后台跑完：exit 0，零非 ok 行

## User Setup Required

None - no external service configuration required.（零新依赖）

## Next Phase Readiness

- **Phase 92 SHIPPED**：CACHE-UNIFY-01..05 全部达成且措辞同步；phase gate 全绿，可进入 `/gsd:verify-work`（92 VALIDATION Phase Gate 四项全过：全量绿 / AST 防线绿 / LOC 双口径报告落档 / invariants 残留 = 0）
- **下一 phase**：93（config_backup 三处 TODO 闭环）或 94（前端 API 工厂化，完全独立）可并行启动；Phase 95（v1.28 SHIP 收口 + v1.29 closeout）必须最后
- **v1.30+ 候选登记**：duty/knowledge/network/workorder 12 处 GetOrSet 样板迁移（invariants warning 档可见可追踪）；A6 死代码 NewDataCacheAdapter 清理；mac_history 接口化后 DataCacheService 字面迁移再评估；system cache_provider.go 5 alias 删除（全部消费者改用 base 直引后）
- 无 blocker

## Self-Check: PASSED

- 文件存在性：cache_invariants_92_test.go（新建）+ 6 个修改文件全部在盘（build/test 通过即证）
- Commit 存在性：`94edab0` / `23b3a11` / `01d08aa` 经 git log 证实；phase 四 plan 原子 commit 链完整（92-01: 5ae394c/756df90/b54a685/2865a3a；92-02: ced0b7a/6e7bb07/1093670；92-03: f2b59f9/dbc987b/724dc45；92-04: 94edab0/23b3a11/01d08aa）
- 验证项：go build ./... 0 错误；go test ./... 全量绿（exit 0）；go test ./internal/utils/operlog/ ./internal/models/ 防线绿；monitor 两包裸 CacheProvider = 0；var _ CacheOperator = 1；adapter 命名 2 处不动；invariants 硬档 PASS + warning 档 11 处可见；人为回归演练红→绿闭环；三文档 grep 验收全过；无意外文件删除（本 plan 4 commit 无 D 状态文件）

---
*Phase: 92-p2*
*Completed: 2026-09-05*

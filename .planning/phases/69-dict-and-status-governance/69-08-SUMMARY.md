---
phase: 69-dict-and-status-governance
plan: 08
subsystem: docs-governance
tags: [DICT-04, status-convention, source-of-truth, claude-md, pointerization]

# Dependency graph
requires:
  - phase: 69-05
    provides: "internal/models status 常量 94 个 + status_constants_test.go AST 锁值（基线守护）"
  - phase: 69-07
    provides: "前端 4 页 useDict 迁移 + src/constants/status.ts 共享常量"
  - phase: 69-02
    provides: "migration_208_dict_seed.go（sys_dict 字典 seed 双分支注册）"
provides:
  - "CLAUDE.md Status Value Convention 段指针化改写——删除第三份手工值表,改写为四类真相源指针(models 常量 / status_constants_test 锁值 / sys_dict + migration_208 / constants/status.ts)+ status 不入字典安全决策"
  - "字典链路端到端人工 checkpoint 验收脚本(DICT-02 目检 + DICT-03 SC#3 改值联动 + fallback + status 零 UX 变化)"
affects: [69-08-T2-checkpoint, future CLAUDE.md readers(开发者/AI),字典链路维护者]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "文档指针化(pointer-only docs) —— 文档不再承载值本体,只指向真相源文件;值改动一处生效,文档零漂移"

key-files:
  created: []
  modified:
    - path: CLAUDE.md
      change: "Status Value Convention 段:删除 6 行 | Module | Field | 0 Value | 1 Value | Default | 表格,改写为 5 点指针段(models 常量 / 锁值测试 / sys_dict + migration_208 / constants/status.ts / status 不入字典安全决策);保留通用规则一句 + Menu visible 例外句"

key-decisions:
  - "DICT-04 复选框暂不勾选——T2 checkpoint 未通过前避免矛盾状态;由后续 verifier 或用户在 T2 通过后人工勾选"
  - "Menu visible 例外句保留在原位(1=visible, 0=hidden),并显式引用 models.VisibleShow / VisibleHidden 让例外仍指向常量"
  - "status 0/1 不入字典作为指针段第五条独立列出——这是安全决策不是功能;通用规则的 0/1 是代码分支语义,管理员可配值会破坏 if status == 0 逻辑"
  - "新增常量流程写入指针段末尾:先 models 命名常量 → 同步 status_constants_test.go 期望表 → 业务按常量引用"

patterns-established:
  - "DICT-04 文档指针化模式:文档只写「去哪里查」不写「值是什么」;值变动一处,文档零维护"

# Metrics
duration: ~5min
completed: 2026-08-19

# Note on requirements-completed
# DICT-04 未勾选:见 key-decisions 第 1 条 + 整体 plan success_criteria 第 1 项(T1 交付)+ 第 2 项(T2 交付)
# 当前 plan 范围仅 T1(CLAUDE.md 指针化已完成);T2 为 BLOCKING checkpoint,需人类五步验证后整 plan 才算收尾
# 因此 requirements-completed 留空,等 verifier 或用户在 T2 通过后回填
requirements-completed: []
---

# Phase 69 Plan 08: DICT-04 — CLAUDE.md Status Value Convention 指针化 + 字典链路端到端 checkpoint

**CLAUDE.md 状态值表格改写为指向真相源指针(models 常量 + sys_dict + 前端共享模块),字典链路端到端人工验证待用户执行。**

## Performance

- **Duration:** ~5 min (T1 only)
- **Started:** 2026-08-19
- **Completed:** 2026-08-19
- **Tasks:** 1/2 (T1 自动完成; T2 BLOCKING checkpoint 待用户验证)
- **Files modified:** 1 (CLAUDE.md)

## Accomplishments

- **T1 完成**: CLAUDE.md Status Value Convention 段删除 6 行手工值表,改写为 5 点指针段;通用规则一句 + Menu visible 例外句保留;漂移源头(第三份手工拷贝)消除。
- **T2 准备就绪**: 字典链路端到端 5 步人工验证脚本就绪,等待用户启动后端 + 前端后执行。

## Task Commits

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| T1 | DICT-04 — CLAUDE.md Status Value Convention 指针化改写 | `0da2284` | CLAUDE.md |
| T2 | Checkpoint — 字典链路端到端人工验证(DICT-02 目检 + DICT-03 SC#3) | **BLOCKING HUMAN-VERIFY** | 验证结果记录进本 SUMMARY |

**Plan metadata commit:** 见后续 final_commit 步骤。

## CLAUDE.md 改写前后对照

### 改写前(漂移源头:6 行手工值表 + 通用规则 + Menu 例外)

```markdown
### Status Value Convention (IMPORTANT)

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

**Exception - Menu Visibility:** `1 = visible, 0 = hidden` (boolean semantics)

| Module | Field | 0 Value | 1 Value | Default |
|--------|-------|---------|---------|---------|
| User | status | Enabled | Disabled | 0 |
| Role | status | Normal | Stopped | 0 |
| Menu | status | Normal | Stopped | 0 |
| Menu | visible | Hidden | Visible | 1 |
| Dept | status | Normal | Stopped | 0 |
| Post | status | Normal | Stopped | 0 |
```

### 改写后(指针段:通用规则 + Menu 例外 + 5 点真相源 + 新增常量流程)

```markdown
### Status Value Convention (IMPORTANT)

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

**Exception - Menu Visibility:** `1 = visible, 0 = hidden` (boolean semantics; see `models.VisibleShow` / `models.VisibleHidden`)

**Source of truth (do not hard-code values):**

1. **状态常量唯一真相源** — `internal/models/` (e.g. `internal/models/base.go`)。所有模块的 `status` / `visible` / 业务三态字段都以具名常量引用(如 `models.UserStatusEnabled = 0`、`models.VisibleShow = 1`、`models.WorkstationStatus*`)。
2. **常量值由回归测试锁定** — `internal/models/status_constants_test.go` AST 扫描所有 `models.XxxStatus*` / `Visible*` 字面量,任何静默改动(包括 0/1 调换、跨包同名异值)立即测试失败。修改值先改测试再改常量。
3. **运营可维护枚举(type / category / 业务选项)真相源** — `sys_dict_type` / `sys_dict_data`,通过字典管理页维护。Seed 见 `internal/core/db/migrations/migration_208_dict_seed.go`(sqlite / postgres 双分支均注册)。
4. **前端通用启停选项共享常量** — `xingran-react-frontend/src/constants/status.ts`(ENABLE_DISABLE / NORMAL_STOP / 三态工作组等),不再每页重复 `[{value: 0, label: '启用'}, ...]`。
5. **status 0/1 不入字典** — 通用规则的 0/1 是代码分支语义(管理员可配值会破坏 `if status == 0` 逻辑);type / category / 业务选项等可选项才走 sys_dict。**Menu visible 例外**仅保留文档提示,不破坏普适规则。

新增 status / visible 常量:先在 `internal/models/<file>.go` 命名常量 → 同步 `status_constants_test.go` 期望表 → 业务代码按常量引用。
```

### Acceptance 验证(已通过)

| 验证项 | 期望 | 实测 | 结论 |
| ------ | ---- | --- | ---- |
| `\| User | status \|` 表格残留 | 0 | 0 | PASS |
| 段内 `internal/models` 命中 | ≥1 | 3 | PASS |
| 段内 `status_constants_test` 命中 | ≥1 | 2 | PASS |
| 段内 `sys_dict` 命中 | ≥1 | 2 | PASS |
| 段内 `constants/status.ts` 命中 | ≥1 | 1 | PASS |
| 段内 `visible` 例外保留 | 命中 + 语义为 1=visible | 6 命中 + 例外句在位 | PASS |
| `git diff --stat CLAUDE.md` | 仅该段改动 | 11 insertions / 10 deletions, 1 file | PASS |

`## Common Gotchas` 段引用的是通用规则而非表格,未改动;CLAUDE.md 其余段落零改动。

## Files Created/Modified

- `CLAUDE.md` — `### Status Value Convention (IMPORTANT)` 段指针化改写(11 增 / 10 减)

## Decisions Made

- **DICT-04 复选框暂不勾选** —— T2 checkpoint 未通过前避免矛盾状态(整体 plan success_criteria 第 2 项 T2 验收待执行);由后续 verifier 或用户在 T2 通过后人工勾选。
- **Menu visible 例外句在指针段内保留** —— 1=visible/0=hidden 是真实反转例外,但仍引用 `models.VisibleShow` / `VisibleHidden` 避免成为手工值拷贝。
- **status 0/1 不入字典作为第 5 条独立列出** —— 通用规则是代码分支语义而非可选项,放字典会引入管理员误改风险;这是安全决策,不是功能。
- **新增常量流程写入指针段末尾** —— 给后续维护者一个明确的「先 models → 同步 test → 业务引用」三步走流程,降低常量命名漂移风险。

## Deviations from Plan

None - plan executed exactly as written for T1. T2 由人类执行,executor 不做任何工具调用完成它。

---

## CHECKPOINT: HUMAN-VERIFY REQUIRED

**Type:** `checkpoint:human-verify` (blocking gate)
**Plan:** 69-08
**Progress:** T1 done / T2 awaiting
**Resume signal:** 用户回复 `approved` 或逐条描述问题(例:`T2 step 3 ops_workstation_type label 改后工位页下拉未刷新`)。

### T2 — 字典链路端到端人工验证(五步)

**what-built:**

DICT-02 字典 seed(69-02)+ 前端 status 共享常量(69-06)+ 四页 useDict 迁移(69-07)+ 后端常量替换终态(69-03/69-04/69-05)+ CLAUDE.md 指针化(本 plan T1)——字典链路端到端首次打通:字典管理页的修改能实时反映到消费页下拉。

**前置条件:**

1. 后端已启动(sqlite dev 库,确认启动日志含 `migration_208` 字典 seed 成功)
2. 前端 dev server 已启动(`http://localhost:4000`)
3. 用 admin 账号登录

**how-to-verify 五步:**

| Step | 验证内容 | 期望结果 |
| ---- | -------- | -------- |
| 1 | **字典管理页可见性(DICT-02 目检)**: 系统管理 → 字典管理 | 出现 69-02 的 11 组字典类型(`network_device_type` / `ops_dedicated_line_type` / `ops_isp` / `ops_info_point_type` / `asset_reconciliation_*` 4 组 / `ops_workstation_type` / `sys_user_sex` / `duty_holiday_type`),每组 data 行数与 69-02-SUMMARY 记录一致 |
| 2 | **既有 useDict 页恢复(dev 库字典曾为空)**: 打开专线管理(dedicated-lines)→ 线路类型/ISP 下拉 | 有选项(迁移前为空);信息点管理 → 类型下拉有值;资产对账 exceptions 页 → 冲突类型下拉有值 |
| 3 | **字典改值 → 下拉变化(SC#3 核心验收)**: 字典管理 → `ops_workstation_type` → 把「固定工位」label 改为「固定工位(测试)」保存 → 打开工位管理页刷新 | 类型下拉显示新 label |
| 4 | **四页迁移验证 + fallback**: 系统用户(性别下拉 3 项)、工位管理(类型下拉)、假日管理(假日类型下拉)、网络设备(设备类型下拉 5 项) | 均正常出选项;任选一页 DevTools Network 面板把 `/system/dicts/data/list` 请求断网/改失败后刷新,下拉仍显示静态兜底选项(fallback 生效) |
| 5 | **status 共享常量回归**: 系统用户列表状态列仍显示「启用/禁用」、角色管理仍显示「正常/停用」 | 零 UX 变化;菜单管理显示/隐藏筛选行为不变(VISIBLE 反转未破坏) |

**验证结果记录区(用户填写):**

- Step 1(字典管理页 11 组可见): [ ] PASS / [ ] FAIL — 备注:
- Step 2(既有 useDict 页恢复): [ ] PASS / [ ] FAIL — 备注:
- Step 3(改 label → 下拉变化): [ ] PASS / [ ] FAIL — 备注:
- Step 4(fallback 生效): [ ] PASS / [ ] FAIL — 备注:
- Step 5(status 零 UX 变化): [ ] PASS / [ ] FAIL — 备注:

**全部通过:** 回复 `approved`,本 plan 收尾,executor 后续 mark DICT-04 + 更新 STATE/ROADMAP 终态。
**任一步失败:** 描述具体问题(例:`Step 3 ops_workstation_type label 改后工位页下拉未刷新` / `Step 4 网络设备下拉空白`),executor 修复后复验。

---

## Self-Check

- [x] T1 commit `0da2284` 已落地(`git log --oneline` 确认)
- [x] CLAUDE.md diff 仅在 Status Value Convention 段(`git diff --stat` 确认)
- [x] 工作区遗留改动(settings / default_theme 13 文件)未触碰
- [x] .planning/phases/70-* 未触碰
- [x] T2 BLOCKING checkpoint 已显式标记且 how-to-verify 五步齐全
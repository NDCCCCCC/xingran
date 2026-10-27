---
gsd_state_version: "1.0"
milestone: v1.23
milestone_name: 前端契约全量收尾与遗留清偿 (Frontend Contract Full Rollout & Legacy Clearance)
current_phase: 69
current_phase_name: roadmap defined, ready to plan
status: Roadmap created (Phases 69-75, 22/22 REQ mapped)
stopped_at: Phase 69 context gathered
last_updated: "2026-10-27T07:55:48.502Z"
last_activity: 2026-10-27
last_activity_desc: v1.23 roadmap created (Phases 69-75)
state_head: febde25341ccf72f61ee98a8687d3aea4be78421
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.23 milestone planning (roadmap created 2026-10-27, Phases 69-75)
**Last activity**: 2026-10-27
**Prior activity**: 2026-10-27 — v1.23 roadmap CREATED（Phases 69-75, 22/22 REQ mapped）
**Prior activity (2)**: 2026-10-27 — v1.22 前端公用组件整治 SHIPPED & ARCHIVED（Phases 63-68.1, 7 phases, 20/20 REQ satisfied）

## Current Milestone: v1.23 前端契约全量收尾与遗留清偿

**Core value**: 把 v1.22 建立但未落地的契约真正推到全量——修复用户主诉的「文本输入框无法一键删除」，并清偿 v1.22 收档结转的 debug / UAT / 流程积压，让里程碑以「无债、无盲区」收档。

**事实基线**: `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS,SUMMARY}.md`（4 维并行研究, commit b5b3aeaf）+ `.planning/v1.23-seeds.md`（A/B/C/D 四类结转项）+ `.planning/debug/` 7 份 debug 文档。

**用户主诉根因**: TextArea 74 处 0 allowClear 覆盖——`AppInput` 契约只包单行 `Input`, `eslint.config.js:114` 注释又明确 `Input.TextArea`「有意不拦」, 形成零约束盲区。

## Current Position

Phase: 69 — Not started (roadmap defined, ready to plan)
Plan: —
Status: Roadmap created (Phases 69-75, 22/22 REQ mapped)
Last activity: 2026-10-27 — v1.23 roadmap created (Phases 69-75)

### Phase 依赖速查（v1.23）

| Phase | 主题 | 硬依赖 | 需求 |
|-------|------|--------|------|
| 69 | 后端 debug 重验 spike + 信息收集下发 | — | DBG-01 |
| 70 | TextArea 契约基建 (W0) | — | TEXT-01, TEXT-03, TEXT-04, HYG-05 |
| 71 | TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6) | 70 | TEXT-02, MIG-01, MIG-02, MIG-04 |
| 72 | 后端 debug 修复 wave (#3→#2→#1-b) | 69 | DBG-02, DBG-03, DBG-04 |
| 73 | DB 实测日 + cron.go 双执行调查 | 72 + 69 | DBG-05, DBG-06, DBG-07 |
| 74 | COMP-05 收口 + lint 清偿第一波 + 流程积压归档 | 71 | MIG-03, HYG-01, HYG-02, HYG-03, HYG-04 |
| 75 | 浏览器实机 UAT + 收档 | 74 + 72/73 | UAT-01, UAT-02 |

**并行区间**：Phase 69/70/71（后端线 + 前端线无共享文件）∥ Phase 72/73；Phase 74/75 必须等前端线（71）完成。

## Accumulated Context

### v1.23 关键决策（init + roadmap 期, locked）

- `AppTextArea` **独立具名导出**，不挂 `AppInput.TextArea` 静态别名——成员表达式 selector 无法区分合法别名用法与非法裸用法，挂别名会永久放弃成员表达式拦截面、74 处收口假绿
- 排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估；每个 phase 的成功标准以「权威计数落盘/对账」表述，不硬编码某个 grep 数
- DBG-03 走 **Go 侧防线**（写入方枚举 + 显式 normalize + 回归测试），**不用 DB TRIGGER**（维持 v1.19 决策）
- 零新依赖：不引 jscodeshift / Playwright；UAT 用环境已有的 agent-browser skill
- 排序硬约束：迁移 → 收口 → lint → UAT（UAT 必须测迁移后的页面；lint 排在迁移后因迁移波次自身产生新 warn 峰值）
- 后端 debug 线与前端线**全程无共享文件、可并行**；#3 先于 #2（同文件 mac_collection_service.go）；cron.go 改动独立隔离（回归闸门 = `go test ./internal/scheduler/` 全套）
- DB 实测日集中 #4/#5/#7（一次环境准备换三个清单）

### v1.22 关键决策（init + roadmap 期, locked）

- 抽契约不抽业务：资产选择器、部门树选择器等业务语义组件已够用，不抽
- `useDebouncedSearch` 并入 RemoteSelect，不单独抽
- 巨型文件拆分不单独立项，随 COMP-05 迁移顺带
- 默认值必须下沉封装层，禁止逐页补 allowClear（横跨 73 文件，不可维护）
- ConfirmDelete 是危险操作唯一入口；危险操作必须二次确认
- COMP-01/02（AppInput/AppSelect/ConfirmDelete）先于 COMP-03/04（PageTable/FormModal）落地，后者依赖前者
- COMP-05 迁移顺序：C1 试点 discoveries 最小页 → backups → devices/cache 巨型页；每页一个 PR，旧实现保留一个迭代
- FIX-02 Modal 脏检查在 Phase 64 存量修补、Phase 66 契约化到 FormModal（两段式）
- 本里程碑纯前端，不动后端接口；预期收益：消除约 7,000~8,000 行样板

### 继承自 v1.21 的环境约束

- `gsd_run` CLI 在本环境不可用（`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>`）——STATE/ROADMAP/REQUIREMENTS 用手工编辑（Phase 57-62 先例）
- 提交落在 main 分支（`branching_strategy: none`）
- DB migration 编号已用到 218（Phase 59）；v1.23 **可能**新增 migration（#4 若确认 `sys_job` 双行根因 → 唯一索引；#3 走 Go 侧防线**不需要** DB TRIGGER migration），落地时从 219 起编

### Decisions

- [v1.21 遗留] 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）仍待人工执行；v1.18/1.19/1.20 的 site-visit UAT 债同样挂账

### Blockers

- 无

### Todos

- Phase 69 / 70 无前置依赖，可立即 `/gsd-plan-phase 69` 与 `/gsd-plan-phase 70` 并行起草
- Phase 71 等 Phase 70 计划落地；Phase 72 等 Phase 69 的 spike 产出；Phase 73 等 Phase 72 落地 + 用户 DB 实测回填
- Phase 74 等 Phase 71 迁移完成（收口硬依赖）；Phase 75 等 Phase 74 + 72/73
- vendor-react gzip 776 kB 基线沿用为 Phase 71/74 零回归守卫
- 权威计数基线（eslint 命中数 + 豁免清单条目数）由 Phase 70 落盘，Phase 71/74 以其为对账分母

### Roadmap Evolution

- Phases 69-75 created for v1.23「前端契约全量收尾与遗留清偿」: 69 后端 debug 重验 spike（DBG-01）→ 70 TextArea 契约基建 W0（TEXT-01/03/04 + HYG-05）→ 71 TextArea 存量 + COMP-05 分域迁移 W1-W6（TEXT-02 + MIG-01/02/04）→ 72 后端 debug 修复 wave（DBG-02/03/04，#3 先于 #2）→ 73 DB 实测日 + cron.go 双执行调查（DBG-05/06/07）→ 74 COMP-05 收口 + lint 清偿 + quick 目录归档（MIG-03 + HYG-01~04）→ 75 浏览器实机 UAT + 收档（UAT-01/02）。22/22 需求映射，无孤儿无重复。 (CRITICAL)
- Phase 68.1 inserted after Phase 68: v1.22 收尾清偿 phase：危险操作收口 + 契约 minor 补课 + 豁免收缩 + 存量迁移收尾（DEBT-04） (URGENT)

## Session

**Last session:** 2026-10-27T07:55:48.462Z
**Stopped at:** Phase 69 context gathered
**Resume file:** .planning/phases/69-debug-spike/69-CONTEXT.md

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| (v1.23 尚无 plan 执行记录) | — | — | — |

## Operator Next Steps

- `/gsd-plan-phase 69` 与 `/gsd-plan-phase 70` 可并行起草（两者均无前置硬依赖）
- `/gsd-plan-phase 71` 需等 Phase 70 的契约组件与 selector 落地
- Phase 72 需先消费 Phase 69 的 spike 三分类清单与用户 SQL/命令回填

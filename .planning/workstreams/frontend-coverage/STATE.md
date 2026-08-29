---
gsd_state_version: 1.0
milestone: v1.28
milestone_name: 前端测试覆盖率优秀
status: executing
stopped_at: "Phase 88 batch 推进中 — batch47 完成 (GLOBAL 45.87%), active workstream 已切回 frontend-coverage, 用户指令持续 batch 至 70% 目标"
last_updated: "2026-08-29T12:30:00.000Z"
last_activity: 2026-08-29
progress:
  total_phases: 7
  completed_phases: 6
  total_plans: 20
  completed_plans: 20
  percent: 86
---

# Project State (v1.28 — frontend-coverage workstream)

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-23) — v1.28 Current Milestone 段

**Core value:** 前端全量口径测试覆盖率达到优秀（≥70%），CI 防倒退 gate 对称后端 v1.26 治理，使前端覆盖率从此不可无声倒退
**Current focus:** Phase 88 收口 — batch 模式持续推进至 70% (batch47 完成, GLOBAL 45.87%)

## Current Position

Phase: 88
Plan: 88-PROGRESS(batch 模式迭代中 — batch24..47 已完成)
Status: Executing Phase 88 (batch 循环: scout → 测试 → coverage → gate → ratchet → commit → push → CI)
Last activity: 2026-08-29

### Batch 推进记录 (batch41 起,详见 .planning/frontend-coverage-baseline.md ratchet 表)

| Batch | 目标 | GLOBAL | Floor bump | Commit |
|-------|------|--------|-----------|--------|
| 41 | workorder useTemplateActions | 45.12→45.13 | workorder 44.2 | 19d7815 |
| 42 | system menu useMenuActions | 45.36 | sys 34.1→36.9 | c600aa6 |
| 43 | network useExecutionModals | 45.55 | net 36.8→38.9 | 347ed16 |
| 44 | shared ExcelExport | 45.69 | shared 27.4→30.8 | 0abf31d |
| 45 | shared ExcelImport | 45.78 | shared 30.8→33.0 | 3709c04 |
| 46 | shared ImageGallery | 45.86 | shared 33.0→34.9 | 046a730 |
| 47 | shared ActionButtons | 45.87 | shared 34.9→35.4 | 26ae79b |

Progress: [████████░░] 80%（milestone 内 8/10 plans；Phase 83 内 3/5 plans）

## Performance Metrics

**Velocity:**

- Total plans completed: 30（Phase 82 全部完成：82-01 口径切换 / 82-02 主 gate / 82-03 diff gate / 82-04 ci.yml 接线 / 82-05 真实 CI 验证）

## Accumulated Context

### Decisions

v1.28 init 锁定决策（详见 PROJECT.md v1.28 段，不可违反）:

- D-01 目标线: 语句 ≥70%（全量口径，statements 3.67% → ≥70%）
- D-02 范围: 全 src + 白名单 ≤5%（候选 cad-editor 804 + cad-elements 224 ≈ 4.5%，终版 Phase 82 登记）
- D-03 CI gate: 4 层对称后端 v1.26（全局阈值 + per-dir floor + baseline ratchet + PR diff ≥80%）
- D-04 Phase 编号: 82 起（**75-81 属并行 milestone workstream v1.27，严禁使用**）

关键对标: 后端 v1.26 模式（`.github/scripts/check-coverage.sh` / `.planning/coverage-baseline.md` ratchet tracking / 74-10 diff coverage bash+awk 自实现）

- [Phase 82]: 82-01: QUAL-02 仅交付配置侧（exclude 真相源 D-10），登记文档侧由 82-04 落地后才整体 complete
- [Phase 82]: 82-01: 两步口径断言零偏差——584/571/22602/21574/830/3.85 全部实测命中研究基点
- [Phase 82]: 82-02: pages/login floor 61.6 为真相源（62.11−0.5 一位小数），RESEARCH 表 61.7 属研究期舍入笔误（Open Question 3：gate 可复算输出优先）
- [Phase 82]: 82-02: --init GLOBAL 向下截断而非四舍五入——实测落在 (3.7,3.8) 区间时舍入会生成高于实测的阈值让 gate 自锁，82-05 CI 校准直接受益
- [Phase 82]: 82-03: diff gate merge-base 回退——git 三点 diff 对无共同祖先基线 fatal 'no merge base'（plan 空树合成基线技术假设盲点），merge-base 为空时回退两树 diff，只放宽不收紧，CI PR 主路径不变
- [Phase 82]: 82-03: D-15 span 口径 DIFF 参照基线 7376/77481/9.52%（空树全量 join）——与 statements 3.85%/istanbul-Lines 3.88% 三口径对账，9.52% 由 span 包含 + 空行/注释分布不均两机制解释，非 join bug；82-04/82-05 对照用
- [Phase 82]: 82-04: components 细分参考 block 取 json 复算而非 RESEARCH 研究期数字——RESEARCH 其余 91 与具名合计 3981 不等于 gate 口径 3958, 按以脚本输出为准纪律 13 行合计恰为 3958/215/118
- [Phase 82]: 82-04: 基线文档双口径并存有意为之——文档头 3.67%/22602/584 白名单前口径 vs ratchet 表与快照 3.85%/21574/571 gate 口径, 防 verify-work 对 SC-3 字面误读
- [Phase 82]: 82-04: frontend-coverage-diff 的 download-artifact 必须带 with.path 还原 upload v4 LCA 剥离——无 path 则 json 落 workspace 根、gate 全 PR 静默软跳过 exit 0 (GOV-04 失效), 82-05 以 diff job 日志无软跳过提示复核
- [Phase 82]: 82-05: D-14 校准未触发——CI 实测 3.85% (run 32642143749) == 本地 3.85% 零漂移, GLOBAL 3.8 维持, .coverage-fe-floors 零改动; ratchet 起点 SHA 回填 bddb2fc (82-04 落盘 commit 携带 floors 最终态), CI 读数行记 8c7b69f
- [Phase 82]: 82-05: PR-only job 的 push run 断言口径 = conclusion==skipped (job 仍列于 run 而非消失); diff gate 实读 profile 的证据 = 日志无软跳过提示 + 有 diff gate 输出行, job 绿本身不能证明 gate 真跑
- [Phase 83]: 83-01: CR-01 闭环——三类路径(src/test/、*.d.ts、cad-editor 白名单)零行进 diff 分母经真实 CI 确认(PR #7 关闭不 merge, run 32675492512 四 job 全绿), Phase 83 测试/harness PR 不再踩红
- [Phase 83]: 83-01: GOV-04 profile 主路径首次真实触发(补 82-REVIEW IN-06)——diff gate 日志无 json 缺失软跳过提示且有 diff gate 输出行; ci.yml 注释已与 fail-closed exit 2 行为对齐(55389ae)
- [Phase 83]: 83-01: .coverage-fe-floors 零变更——验证性 plan 无覆盖率变化不触发 D-11 ratchet; 本地空树合成基线前后对照(修复前 145 行进分母 0.00% FAIL → 修复后 PASS)与真实 CI 结论一致
- [Phase 83]: 83-02: utils floor=89.7——gate json 复算 90.21−0.5 一位小数（v8 text 88.28% 不作数，D-12 gate 输出为真相源）
- [Phase 83]: 83-02: QUAL-01 不在本 plan 标记 complete——REQUIREMENTS 映射 Phase 88 里程碑收口，本 plan 仅贡献 399 pass / 0 fail 证据
- [Phase 83]: 83-02: 篡改密文实测 sm-crypto 抛 padding is invalid（D-08 直测可行）；filterExternalOrgDepts docstring 与实现不一致按 actual 锁定，不修业务代码
- [Phase 83]: 83-03: lib gate 实测 86.56% (902/1042) 为真相源,v8 text 93.15% 不采信(重申 82-02 json 复算优先);floor 86.56-0.5=86.06 一位小数向下截断为 86.0,余量 0.56pp
- [Phase 83]: 83-03: D-07 双轨 mock 范式落地——vi.mock(axios) 工厂捕获双实例+拦截器 handler 提取直驱,真模块链覆盖 401 刷新队列/400 解密重放/login 401 短路,后续 P0/P1 plan 可复用
- [Phase 83]: 83-03: IN-06 (vitest 版本统一) 条件未触发不执行;adDomainApi.deleteMapping URL 多余 } + 后端缺 /ad-domain/mappings 路由,范围外登记 deferred-items.md
- [Phase 83]: hooks floor 91.7 / store floor 95.0 (83-04 实测 92.29%/95.59% 各减 0.5pp 截断)
- [Phase 83]: noticeStore unreadCount 为服务器侧计数,不随本地 50 条列表淘汰回退 (83-04)

### Pending Todos

None yet.

### Blockers/Concerns

- 并行 workstream 纪律: 本 workstream 可写产物仅在 `.planning/workstreams/frontend-coverage/` 下；PROJECT.md / MILESTONES.md / config.json 只读
- `.github/workflows/ci.yml` 为两 workstream 共享编辑区——Phase 82 加前端 gate 步骤时严禁触碰后端 job（v1.27 同时在改后端覆盖率），建议只增不改
- roadmap 口径备注: 全局 ≥70% 由"白名单外所有目录 per-dir ≥70%"数学保证（加权平均）；design-system 194 stmts 已归入 Phase 84 / COMP-05 零散桶，不留无主面积

## Session Continuity

Last session: 2026-08-29
Stopped at: Phase 88 batch47 完成 (GLOBAL 45.87%), 持续 batch 推进中
Resume file: .planning/workstreams/frontend-coverage/phases/88-final-70/88-PROGRESS.md

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 82 P01 | 6min | 2 tasks | 2 files |
| Phase 82 P02 | 12min | 2 tasks | 2 files |
| Phase 82 P3 | 11min | 2 tasks | 1 files |
| Phase 82 P4 | 6min | 2 tasks tasks | 2 files files |
| Phase 82 P5 | 66min (含确认/merge 等待) | 1 task + 1 checkpoint | 1 file |
| Phase 83 P01 | 90min | 4 tasks | 2 files |
| Phase 83 P02 | 22min | 3 tasks | 23 files |
| Phase 83 P83-03 | 26m | 3 tasks | 20 files |
| Phase 83 P83-04 | 38m | 3 tasks | 18 files |

# Coverage Baseline: v1.28 前端测试覆盖率优秀

**起点来源:** 2026-08-23 全量口径本地实测（Phase 82 口径修正首跑产物 `xingran-react-frontend/coverage/coverage-final.json`；旧口径 24.58% 只计被 import 文件，失真已由 Phase 82 修正——Vitest 4 移除 `coverage.all` 后旧数字不再可比）
**测量口径:** statements，vitest `coverage.include` 全量圈定 `src/**/*.{ts,tsx}`（Vitest 4 无 `coverage.all`，include 是全量口径唯一开关），白名单排除 cad-editor / cad-elements 两目录——白名单前 3.67% / 22602 stmts / 584 文件（全量含 cad 两目录），白名单后（gate 口径）**3.85% / 21574 stmts / 571 文件**
**生成方式:** `bash .github/scripts/check-frontend-coverage.sh --init xingran-react-frontend/coverage/coverage-final.json` + 同脚本 awk 聚合（本地与 CI 同公式）

---

## 起点 (v1.28 启动前)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-23 | 起点 | 3.85 | 21574 | 830 | 15 | bddb2fc | n/a | n/a | n/a |
| 2026-08-23 | Phase 82 CI 实测读数 | 3.85 | 21574 | 830 | 15 | 8c7b69f | n/a | n/a | n/a |
| 2026-08-24 | Phase 83-02 utils 全清 | 7.46 | 21574 | 1609 | 15 | d9e52e6 | 83-02 | utils 7.7 | utils 89.7 |
| 2026-08-24 | Phase 83-03 lib 全清 | 11.11 | 21574 | 2397 | 15 | 3d57a8e | 83-03 | lib 10.4 | lib 86.0 |
| 2026-08-24 | Phase 83-04 hooks+store 全清 | 18.03 | 21574 | 3890 | 15 | 7b2d22a | 83-04 | hooks 7.6 / store 4.3 | hooks 91.7 / store 95.0 |
| 2026-08-27 | Phase 84-01a shared 组件全清 | 18.91 | 21574 | 4081 | 15 | c76946a | 84-01a | shared 0.0 | shared 21.9 |
| 2026-08-27 | Phase 84-01b dashboard 测试落地 | 19.08 | 21574 | 4116 | 15 | c8dd140 | 84-01b | dashboard 0.0 | dashboard 2.1 |
| 2026-08-27 | Phase 84-02a layout 组件测试 | 19.39 | 21574 | 4186 | 15 | $(git rev-parse --short HEAD) | 84-02a | layout 0.0 | layout 14.1 |
| 2026-08-27 | Phase 84-02b cron+captcha+ops | 19.61 | 21574 | 4230 | | | 84-02b | cr/cp/op 0.0 | cr 7.0/cp 10.5/op 0.8 |
| 2026-08-27 | Phase 84-03a network+recon 测试 | 19.64 | 21574 | 4237 | | | 84-03a | network/recon 0.0 | network 50.1 / recon 21.0 |
| 2026-08-27 | Phase 84-03b design-system+components 聚合收口 | 20.06 | 21574 | 4328 | 9 | $(git rev-parse --short HEAD) | 84-03b | ds 15.0/components 4.9 | ds 52.5/components 13.4 |
| 2026-08-27 | Phase 85-W1 floors+workstations | 20.21 | 21574 | 4377 | | | 85-W1 | ops 0.0 | ops 0.7 |
| 2026-08-27 | Phase 85-W2 3d+rpa | 20.51 | 21574 | 4426 | | | 85-W2 | ops 0.7 | ops 2.2 |
| 2026-08-27 | Phase 85-W3 buildings hooks + bs types | 20.67 | 21574 | 4460 | | | 85-W3 | ops 2.2 | ops 3.1 |
| 2026-08-27 | Phase 85-W4 零散 5 目录 | 20.75 | 21574 | 4478 | | | 85-W4 | ops 3.1 | ops 3.6 |
| 2026-08-27 | Phase 86-W1 system dept/role/notice | 20.90 | 21574 | 4511 | | | 86-W1 | sys 2.2 | sys 3.7 |
| 2026-08-27 | Phase 86-W2 menu/dict/user/零散 | 21.21 | 21574 | 4576 | | | 86-W2 | sys 3.7 | sys 6.4 |
| 2026-08-27 | Phase 86-W3 net discoveries/exec/templates | 21.34 | 21574 | 4605 | | | 86-W3 | net 2.6 | net 4.2 |
| 2026-08-27 | Phase 86-W4 command/backups/devices/cred | 21.60 | 21574 | 4662 | | | 86-W4 | net 4.2 | net 7.1 |
| 2026-08-27 | Phase 87-W1 duty+ad-domain | 21.70 | 21574 | 4682 | | | 87-W1 | duty/ad 0.0 | duty 0.5/ad 0.2 |
| 2026-08-27 | Phase 87-W2 monitor+vdi | 21.97 | 21574 | 4740 | | | 87-W2 | mon/vdi 0.0 | mon 7.7/vdi 0.2 |
| 2026-08-27 | Phase 87-W3 workorder+asset | 21.98 | 21574 | 4744 | | | 87-W3 | wo/as 0.0 | wo 0.05/as 0.05 |
| 2026-08-27 | Phase 87-W4 五小目录 | 22.11 | 21574 | 4771 | | | 87-W4 | 0.0 | kn 1.7/ds 2.4/mn 2.6/ad 2.2 |
| 2026-08-27 | Phase 88 页面渲染模式落地 | 23.83 | 21574 | 5142 | | | 88-R1 | sys 6.4 | sys 20.8 |
| 2026-08-27 | Phase 88 批量页面渲染 R2 | 29.03 | 21574 | 6263 | | | 88-R2 | 多目录低值 | duty 16.5/net 20.7/ops 12.1/ad-dom 8.5/vdi 16.7/wo 16.5/kn 29.6 |
| 2026-08-28 | Phase 88 批量页面渲染 R3 | 30.77 | 21574 | 6638 | | | 88-R3 | mon 7.7/ad-dom 8.5 | mon 38.4/ad-dom 17.8 |
| 2026-08-28 | Phase 88 批量页面渲染 R4 | 33.42 | 21574 | 7210 | | | 88-R4 | sys 20.8/net 20.7/duty 16.5 | sys 29.3/net 32.5/duty 28.9 |
| 2026-08-28 | Phase 88 批量渲染 R5 components+widgets | 34.02 | 21574 | 7340 | | | 88-R5 | comps 13.4 | comps 18.6 |
| 2026-08-28 | Phase 88 R6 dashboard+reconciliation 深测 | 35.27 | 21574 | 7610 | | | 88-R6 | dash 4.1/recon 21.0 | dash 25.4/recon 49.5 |
| 2026-08-28 | Phase 88 R7-9 ops/vdi/wo/ad-dom/port-write | 36.19 | 21574 | 7808 | | | 88-R7 | ops 12.1 等 | ops 17.0/vdi 21.3/wo 25.0/ad 20.0 |
| 2026-08-28 | Phase 88 R10-12 子页零散渲染 + maxWorkers | 36.86 | 21574 | 7954 | | | 88-R10 | 0.0 | table 24.5/IconSel 37.0/duty 35.0/kn 46.4 |
| 2026-08-28 | Phase 88 R13-14 workstations子模块+operations子组件 | 37.18 | 21574 | 8023 | | | 88-R13 | ops 12.1/comp-ops 3.5 | ops 17.5/comp-ops 9.0 |
| 2026-08-28 | Phase 88 R15 design-system ConfigProvider 链路 + knowledge view/modals | 37.47 | 21574 | 8084 | | | 88-R15 | ds 52.5/kn 46.4 | ds 78.9/kn 50.3 |
| 2026-08-28 | Phase 88 R16 asset对账/my-notices/dashboard-system | 38.83 | 21574 | 8379 | | | 88-R16 | as 0.05/ds-sys 2.4/mn 2.6 | as 40.15/ds-sys 20.1/mn 40.4 |
| 2026-08-28 | Phase 88 R17 router安全/ad同步/profile/asset组件 | 39.63 | 21574 | 8551 | | | 88-R17 | router 0/ad 2.2 | router 32.9/ad 72.4/profile 36.2/comp-asset 42.7 |
| 2026-08-28 | Phase 88 R18 workorder 三页深测 | 39.73 | 21574 | 8572 | | | 88-R18 | wo 25.6 | wo 28.9 |
| 2026-08-28 | Phase 88 R19 vdi 双列表+shared子组件 | 39.88 | 21574 | 8605 | | | 88-R19 | vdi 21.9/comp-shared 24.2 | vdi 26.3/comp-shared 24.1 |
| 2026-08-28 | Phase 88 R20 fix-suggestion子组件+duty子组件 | 40.02 | 21574 | 8636 | | | 88-R20 | duty 35.5 | duty 37.5 |
| 2026-08-28 | Phase 88 R21 notice/dept/role/post 子组件 | 40.24 | 21574 | 8683 | | | 88-R21 | sys 29.9 | sys 30.8 |
| 2026-08-28 | Phase 88 R22 network/reconciliation子组件 | 40.31 | 21574 | 8698 | | | 88-R22 | recon 49.5 | recon 60.6 |
| 2026-08-28 | Phase 88 R23 network discoveries/templates modals | 40.32 | 21574 | 8699 | | | 88-R23 | 0.0 | 0.0 |
| 2026-08-28 | Phase 88 R24 workstations 钩子+HybridLayout | 41.28 | 21574 | 8905 | | | 88-R24 | ops 17.0/comp-ops 9.0 | ops 19.9/comp-ops 35.7 |
| 2026-08-28 | Phase 88 R25 floors 子组件+geocoding 钩子 | 41.48 | 21574 | 8950 | | | 88-R25 | ops 19.9 | ops 21.2 |
| 2026-08-28 | Phase 88 R26 monitor utils+columns+vdi 按钮表 | 41.69 | 21574 | 8996 | | | 88-R26 | mon 38.4 | mon 45.5 |
| 2026-08-28 | Phase 88 R27 duty holidays/schedules utils+WeeklyView | 41.85 | 21574 | 9029 | | | 88-R27 | duty 37.5 | duty 40.3 |
| 2026-08-28 | Phase 88 R28 ad-domain logs+AccountPoolTab 渲染 | 42.07 | 21574 | 9077 | | | 88-R28 | ad-dom 20.0 | ad-dom 24.4 |
| 2026-08-28 | Phase 88 R29 network command hooks+mac heatmap | 42.46 | 21574 | 9161 | | | 88-R29 | net 32.5 | net 36.8 |
| 2026-08-28 | Phase 88 R30 dashboard edit/view+NotificationBell | 42.62 | 21574 | 9196 | | | 88-R30 | ds-sys 20.1 | ds-sys 34.9 |
| 2026-08-28 | Phase 88 R31 captcha+services API 封装 | 43.05 | 21574 | 9289 | | | 88-R31 | captcha 21.5/services 3.3 | captcha 44.9/services 42.3 |
| 2026-08-28 | Phase 88 R32 components network 子组件+MACEventsTimeline | 43.21 | 21574 | 9323 | | | 88-R32 | comp-net 51.9 | comp-net 63.3 |
| 2026-08-28 | Phase 88 R33 components widgetRegistry+VDI/AssetRow | 43.36 | 21574 | 9355 | | | 88-R33 | comp-table 24.5/comp-dash 25.4 | comp-table 88.7/comp-dash 27.2 |
| 2026-08-28 | Phase 88 R34 router routeConfigManager+componentLoader | 43.83 | 21574 | 9457 | | | 88-R34 | router 32.9 | router 74.5 |
| 2026-08-28 | Phase 88 R35 dashboard widgets BaseWidget | 43.98 | 21574 | 9489 | | | 88-R35 | comp-dash 27.2 | comp-dash 29.2 |
| 2026-08-28 | Phase 88 R36 system apikeys LogsModal | 44.22 | 21574 | 9541 | | | 88-R36 | sys 30.8 | sys 32.7 |
| 2026-08-28 | Phase 88 R37 system role useRoleActions | 44.32 | 21574 | 9563 | | | 88-R37 | sys 32.7 | sys 34.1 |
| 2026-08-28 | Phase 88 R38 dashboard Stat/Progress widget | 44.54 | 21574 | 9610 | | | 88-R38 | comp-dash 29.2 | comp-dash 33.7 |
| 2026-08-28 | Phase 88 R39 workorder useWorkOrderActions | 44.80 | 21574 | 9667 | | | 88-R39 | workorder 28.9 | workorder 38.9 |
| 2026-08-28 | Phase 88 R40 components shared ColumnConfigModal | 44.99 | 21574 | 9707 | | | 88-R40 | comp-shared 24.1 | comp-shared 27.4 |
| 2026-08-28 | Phase 88 R41 workorder useTemplateActions | 45.12 | 21574 | 9736 | | | 88-R41 | workorder 38.9 | workorder 44.2 |
| 2026-08-29 | Phase 88 R42 system menu useMenuActions | 45.36 | 21574 | 9786 | | | 88-R42 | sys 34.1 | sys 36.9 |
| 2026-08-29 | Phase 88 R43 network executions useExecutionModals | 45.55 | 21574 | 9827 | | | 88-R43 | net 36.8 | net 38.9 |
| 2026-08-29 | Phase 88 R44 components shared ExcelExport | 45.69 | 21574 | 9857 | | | 88-R44 | comp-shared 27.4 | comp-shared 30.8 |
| 2026-08-29 | Phase 88 R45 components shared ExcelImport | 45.78 | 21574 | 9877 | | | 88-R45 | comp-shared 30.8 | comp-shared 33.0 |
| 2026-08-29 | Phase 88 R46 components shared ImageGallery | 45.86 | 21574 | 9894 | | | 88-R46 | comp-shared 33.0 | comp-shared 34.9 |
| 2026-08-29 | Phase 88 R47 components shared ActionButtons | 45.87 | 21574 | 9897 | | | 88-R47 | comp-shared 34.9 | comp-shared 35.4 |

### Per-directory (起点, D-05 粒度 = src 一级目录 + pages 二级拆分 + `(src root)`/`api` 显式条目)

```
目录                          stmts   covered      pct  文件数
components                      3958       215    5.43%    118
pages/operations                3611         0    0.00%     76
pages/system                    2203        60    2.72%     56
pages/network                   1962        61    3.11%     61
pages/duty                      1190         0    0.00%     40
pages/ad-domain                 1082         0    0.00%      8
hooks                           1050        85    8.10%     27
lib                             1042       114   10.94%     20
utils                            950        78    8.21%     23
pages/monitor                    627         0    0.00%     26
store                            589        28    4.75%      9
pages/vdi                        567         0    0.00%      4
pages/workorder                  551         0    0.00%     14
pages/asset                      475         0    0.00%      7
router                           272         0    0.00%      5
pages/knowledge                  262         0    0.00%      7
services                         238         9    3.78%     13
pages/dashboard-system           203         0    0.00%      7
design-system                    194        30   15.46%     13
pages/my-notices                 127         0    0.00%      2
pages/login                       95        59   62.11%      1
pages/profile                     87         0    0.00%      1
constants                         84        33   39.29%      6
pages/settings                    65        54   83.08%      1
pages/ad                          37         0    0.00%      1
types                             32         4   12.50%     22
(src root)                        13         0    0.00%      2
api                                8         0    0.00%      1
TOTAL                          21574       830    3.85%    571
```

Notes (起点快照):

- 数据源 = coverage-final.json 复算（与 gate 脚本 awk 聚合同公式、逐值一致）；per-dir floor 初值（D-06 实测 −0.5pp，下限 0）见 `.coverage-fe-floors`，不在此重复维护。
- 0pct_pkg_count = 15 = 上表 pct 为 0.00% 的目录数（pages/operations、pages/duty、pages/ad-domain、pages/monitor、pages/vdi、pages/workorder、pages/asset、pages/knowledge、pages/dashboard-system、pages/my-notices、pages/profile、pages/ad、router、(src root)、api）。
- pages/login floor 为 61.6（62.11 − 0.5 按一位小数为 61.6；82-RESEARCH 表 61.7 属研究期舍入笔误——82-02 决策：以 gate 可复算的脚本输出为真相源）。
- components 文件数 118 为 json 实测（RESEARCH 表 152 属研究期 CLI 覆盖运行计数口径；stmts/covered 逐位一致，gate 只消费 stmts/covered——82-02 已决策无需处理）。

### components 细分参考 (非 gate 粒度, 供 Phase 84 用)

```
子目录                          stmts   covered      pct  文件数
dashboard                        1068         0    0.00%     29
shared                            892        25    2.80%     21
layout                            507         0    0.00%     16
network                           324       164   50.62%      9
CronSelector                      316         0    0.00%     10
captcha                           154         0    0.00%      4
operations                        149         0    0.00%      5
reconciliation                    144        26   18.06%     10
(components 顶层散件)             135         0    0.00%      3
asset                              81         0    0.00%      2
DeptTree                           64         0    0.00%      1
IconSelect                         56         0    0.00%      1
其余零散 table/three/charts/NoticeDetail/markdown/modal
                                   68         0    0.00%      7
TOTAL                            3958       215    5.43%    118
```

（components 在 gate 中是**一个整体 floor**——D-05 粒度；上表仅作 Phase 84 拆解参考。白名单 cad-editor 804 / cad-elements 224 stmts 已在 coverage json 生成前排除，不在 components 总数内。）

---

## 白名单登记 (QUAL-02)

| 目录 | 排除理由 | 面积 (stmts/文件) | 占总语句 | 复审条件 |
|------|---------|------------------|---------|---------|
| src/components/cad-editor | 重画布 CAD/3D 交互 UI，vitest jsdom 单测性价比极低（对称后端 v1.26「辅助包不强制覆盖」先例） | 804 / 8 | 3.56% | 自身 statements ≥70% 可启动移除（D-11 定量触发）；每个 milestone 收口强制重审 |
| src/components/cad-elements | 同上（CAD 画布元素库，重渲染低确定性 UI） | 224 / 5 | 0.99% | 同上 |
| **合计** | | **1028 / 13** | **4.55% ≤ 5%** | |

- **D-12 锁死**：白名单仅此两项；未来任何新增必须走 milestone 级显式决策，并重新核算总面积 ≤5%。
- **D-10 同源**：排除的单一真相源是 `xingran-react-frontend/vitest.config.ts` 的 `coverage.exclude` 数组；gate 脚本做漂移检测（cad 目录出现在 coverage json 即 exit 6），配置漂移使 CI 变红而非静默放行。

### 倒退检查 (本行)

- [x] 无新增白名单项（仅 cad-editor + cad-elements 两项，D-12 锁死）
- [x] 白名单面积 ≤5%（1028 / 22602 = 4.55%）
- [x] 全局与 per-dir 数字可由 gate 脚本复算（`bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` → GLOBAL PASS + 28/28 目录 PASS）

> **Ratchet note（P-ratchet，后端 D-04 前端对称）:** 起点 row 的 `commit` 列已由
> plan 82-05（真实 CI 验证，2026-08-23）回填为 82-04 落盘 commit 短 SHA——CI 实测
> 3.85% ≥ 阈值 3.8，未触发 D-14 校准，`.coverage-fe-floors` 未变更（起点 commit
> 携带其最终态）。`.coverage-fe-floors` 的每次变更必须与本文件的追加落在同一
> commit；floor 只升不降（D-06 初值 = 实测 −0.5pp 的噪声余量纪律见
> `.coverage-fe-floors` 头注释与 gate 脚本头注释摘录）。

## CI 验证记录 (Phase 82 · 82-05, 2026-08-23)

- **验证 PR**: https://github.com/NDCCCCCC/xingran/pull/6 — docs-only 单 commit `4d1361b`（基线文档 +2 行，不触 `xingran-react-frontend/src`），squash merge = `8c7b69f`
- **证据一（PR run 32642143749）**: frontend job pass（`Coverage gate`: `TOTAL 21574 830 3.85%` / `PASS: weighted avg 3.85% >= threshold 3.80%` / `PASS: per-dir floor gate — 28/28 directories >= floor`）；frontend-coverage-diff job pass 且日志**无** json 缺失软跳过提示（`Test step skipped` / `skipping gate` 均 0 次），空 diff 软通过 1 次（`diff-coverage: no testable .ts/.tsx lines changed vs origin/main — PASS (nothing to gate)`）——diff gate 实读 artifact 还原后的 profile，非静默空转（T-82-04-05 mitigation 真实 CI 生效）
- **证据二（main push run 32643452003，head=8c7b69f）**: frontend = success；frontend-coverage-diff = **skipped**（PR-only job 在 push run 的既定表现——job 仍列于 run 但标记 skipped）；backend = success / coverage-diff = skipped（main 常态）
- **D-14 校准判定**: 无需校准——CI 实测 3.85%（run 32642143749）≥ 阈值 3.8，与本地读数零漂移；`.coverage-fe-floors` GLOBAL 3.8 维持
- **D-04 观察项**: `Test (coverage)` 步骤 CI 实际耗时 41 秒（13:23:20→13:24:01；15 分钟 timeout 余量充足，不调整）
- **先行佐证（作废 run 32630491947，PR #5）**: head 被并行 workstream 共享工作树事故污染后废弃重开（处置链见 82-05-SUMMARY）；其读数与正式证据完全一致（3.85% / 28/28 目录 / 无软跳过），佐证 gate 行为可复现

## CI 验证记录 (Phase 83 · 83-01 CR-01 试验 PR 验证, 2026-08-24)

- **试验 PR**: https://github.com/NDCCCCCC/xingran/pull/7 "[DO NOT MERGE] Phase 83 CR-01 trial PR" — 分支 `phase-83-trial-cr01`，head `7d481f9`，状态 CLOSED（关闭不 merge，2026-08-24T00:13:33Z）
- **三类路径变更清单**: `xingran-react-frontend/src/test/utils/trial-test.ts`（src/test/ 占位测试）、`xingran-react-frontend/src/types/global.d.ts`（.d.ts 追加注释）、`xingran-react-frontend/src/components/cad-editor/theme.ts`（cad-editor 白名单目录追加注释）——每类至少一个文件，均为无害变更，不触业务逻辑
- **CI run**: https://github.com/NDCCCCCC/xingran/actions/runs/32675492512 — 四 job 全绿：backend / frontend / frontend-coverage-diff / coverage-diff 均 success
- **diff gate 结论**: frontend-coverage-diff 日志输出 `diff-coverage: no testable .ts/.tsx lines changed vs origin/main — PASS`——三类路径（src/test/、`*.d.ts`、cad-editor 白名单）零行进入 diff 分母，CR-01 pathspec 镜像在真实 CI 生效，无误报失败
- **frontend job**: Tests 159 passed（存量无回归）+ Coverage gate PASS（3.85% >= 3.80%；GLOBAL 与 28/28 目录 floor 均不变）
- **GOV-04 profile 主路径首次真实触发（补 82-REVIEW IN-06）**: 本 run 是 CR-01/WR-01~03 四项修复合入后，diff gate 首次在真实 PR 上以"非空三类路径变更 + profile 实读"主路径运行——日志无 json 缺失软跳过提示；fail-closed exit 2 行为已由 ci.yml 注释修正（commit 55389ae）与脚本实际对齐
- **本地前后对照（Task 2 空树合成基线）**: 修复前同 diff 有 145 行（三类路径）进入分母、diff 覆盖率 0.00% FAIL；修复后同 diff 输出 "no testable .ts/.tsx lines changed ... PASS"。合成基线前后对照与真实 CI 结论一致，CR-01 修复闭环
- **floors 不变**: 本 plan 为验证性变更，无覆盖率变化，`.coverage-fe-floors` 未修改、不触发 D-11 ratchet

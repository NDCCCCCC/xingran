# Coverage Baseline: v1.28 前端测试覆盖率优秀

**起点来源:** 2026-08-23 全量口径本地实测（Phase 82 口径修正首跑产物 `xingran-react-frontend/coverage/coverage-final.json`；旧口径 24.58% 只计被 import 文件，失真已由 Phase 82 修正——Vitest 4 移除 `coverage.all` 后旧数字不再可比）
**测量口径:** statements，vitest `coverage.include` 全量圈定 `src/**/*.{ts,tsx}`（Vitest 4 无 `coverage.all`，include 是全量口径唯一开关），白名单排除 cad-editor / cad-elements 两目录——白名单前 3.67% / 22602 stmts / 584 文件（全量含 cad 两目录），白名单后（gate 口径）**3.85% / 21574 stmts / 571 文件**
**生成方式:** `bash .github/scripts/check-frontend-coverage.sh --init xingran-react-frontend/coverage/coverage-final.json` + 同脚本 awk 聚合（本地与 CI 同公式）

---

## 起点 (v1.28 启动前)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-23 | 起点 | 3.85 | 21574 | 830 | 15 | TBD (atomic ratchet) | n/a | n/a | n/a |

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

> **Ratchet note（P-ratchet，后端 D-04 前端对称）:** The `commit` column on the 起点
> row reads `TBD (atomic ratchet)` until plan 82-05 (真实 CI 验证 + 校准定稿)
> amends this file with the actual short SHA. `.coverage-fe-floors` 的每次变更
> 必须与本文件的追加落在同一 commit；floor 只升不降（D-06 初值 = 实测 −0.5pp
> 的噪声余量纪律见 `.coverage-fe-floors` 头注释与 gate 脚本头注释摘录）。

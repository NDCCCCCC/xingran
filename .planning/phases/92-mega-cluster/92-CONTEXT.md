---
phase: "92"
slug: "mega-cluster"
status: complete
created: "2026-10-30"
mode: discuss
---

# Phase 92 — CONTEXT：暗色 mega-cluster 合并

> 本文档锁定 Phase 92 的实施决策。下游 researcher / planner / executor 以此为准，无需再次询问用户。

---

## 已锁定决策（USER DECISIONS）

### D-01: 等价性确认采用「AST 脚本自动分类 + 例外表人工补」

- **决策**：编写一次性 Node 扫描脚本（存放于 `.planning/phases/92-mega-cluster/tools/`），把 R6/R7 区间内全部暗色规则逐条解析为（选择器 + 属性 + 值）结构化清单；自动与 `AntdThemeBridge.tsx` 已配置的 token（`token.*` + `components.*`）做匹配分类：
  - **covered**：已有 token 等价物 → 直接删；
  - **extendable**：antd 存在该 token 但 Bridge 未配置 → 扩 Bridge 后删；
  - **exception**：antd 无法表达（图表 / 3D / 登录页插画等）→ 进例外登记表，逐条写理由。
- **验收口径**：零未确认项 = 每条命中三类之一，例外表逐条有理由。
- **Reversibility**: reversible（脚本是纯只读分析，删除按 wave 独立 commit 可回滚）。

### D-02: 暗色残留策略「全量 token 化 + 少量例外分区」

- **决策**：能 token 化的暗色样式全部交还 `AntdThemeBridge` 的 `components.*`（扩表现有 `board = isDark ? ruoyiBrand.dark : ruoyiBrand.cream` 映射）；仅 antd token 无法表达的少数场景保留 `[data-color-mode="dark"]` 手写分区，终态 ≤12 块、每块登记保留理由。
- **边界**：不采用「优先 darkAlgorithm 派生、少扩 components 表」路线 —— Phase 89 已实证 dark 派生对饱和品牌绿基底会产出错误色相（89-RESEARCH §1.1.2），Map 层显式给值的纪律延续。
- **Reversibility**: costly（影响全站暗色视觉，但每波独立 commit）。

### D-03: 视觉验收分工「92 机器验收，93 人工巡检」

- **决策**：Phase 92 以 computed-style 自动断言 + 21 形态矩阵脚本（复用 `.planning/phases/90-input-variant/90-visual-evidence/` 工具链改造为 92 口径）完成机器验收，差异清单逐条登记；需要 AD 登录的 14 页人工巡检截图**移交 Phase 93 终验一次完成**，Phase 92 登记为显式移交项（precedent：Phase 90 → 93 移交模式）。
- **Reversibility**: reversible（验收排期调整，不改代码）。

### D-04: 执行波次「3 波：扫描 → R6 → R7」

- **决策**：
  - **Wave 1（91-01 对应 92-01）**：AST 等价性扫描脚本 + 三类分类清单 + 例外登记表 + `AntdThemeBridge` 扩表清单草案；
  - **Wave 2（92-02）**：删 R6 残块（约 `:1967-2742` 区间，行号以 Wave 1 实测为准）+ 扩 Bridge token + 机器验证；
  - **Wave 3（92-03）**：删 R7 残块 + 量化终态达标（M1/M2/M4/M5/M9/M7）+ 矩阵机器验收 + 移交登记。
- **纪律**：删除为**块级操作**（按 cluster 边界整段删），不逐条手改（ROADMAP 已锁）；每波独立 commit 可回滚。
- **Reversibility**: costly（每波回滚点独立）。

---

## 灰区已决定 / 沿用既有决策（不再讨论）

| 事项 | 结论 | 来源 |
|---|---|---|
| 删除方式 | 块级整段删，不逐条手改 | ROADMAP Phase 92 成功标准 1 |
| Map 层 token 纪律 | 显式给值，不交 darkAlgorithm 派生 | Phase 89 D-02 / 89-RESEARCH §1.1.2 |
| `ruoyiBrand` 单一色板真相源 | 所有新 token 值从 `colors.ts` 读取，禁新增裸 hex | Phase 89 D-03 |
| C11/C13 ratchet | 只降不升，Phase 92 终值 M1 ≤480 / M7 ≤60 | ROADMAP 量化终态 |
| 移交项模式 | 人工巡检移交 93，登记显式移交表 | Phase 90 先例 |

## 边界与红线

- **零修改**：`src/constants/permissionCodes.ts`、`scripts/gating-exemptions.json`、`check-gating-coverage.mjs`（invariance 25+51=76）。
- **零新增**：`.css` 文件数、裸 hex（新增 hex 须走 `layer:"CSS"` 豁免登记，且终态 M7 ≤60 使豁免空间趋零）。
- **既有契约全绿**：C1-C16c（含 VIS-01/02/03/05、Input 域 8px 圆角、autofill 修复）在任何 wave 结束时都必须保持 PASS。
- **暗色禁止回退**：删暗色副本前必须先确认 token 等价物已生效（Wave 2/3 均为先扩表后删除，同 commit 内完成），否则暗色组件瞬间回 antd 默认色。

## 移交 Phase 93 的登记项

1. 14 页人工巡检截图（AD 登录墙）；
2. 暗色 `--input-bg` 对比度例外（X-2，描边判据在 92 内完成提亮后由 93 终验）。

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

- **D-01:** 等价性确认采用「AST 脚本自动分类 + 例外表人工补」——编写一次性 Node 扫描脚本（`.planning/phases/92-mega-cluster/tools/scan-dark-rules.cjs`），把 R6/R7 全部暗色规则解析为（选择器+属性+值）清单，与 AntdThemeBridge 已配置 token 匹配分三类：covered（有等价物→删）/ extendable（antd 有 token 但 Bridge 未配→扩表后删）/ exception（无法表达→登记理由）。验收口径：零未确认项，每条命中三类之一。Reversibility: reversible（纯只读分析，删除按 wave 独立 commit）。
- **D-02:** 暗色残留策略「全量 token 化 + 少量例外分区」——能 token 化的全部交还 AntdThemeBridge `components.*`（扩表现有 `board` 映射）；仅 antd 无法表达的少数场景（图表/3D/Scrollbar/Focus Visible 等）保留 `[data-color-mode="dark"]` 手写分区，终态 ≤12 块逐块登记理由。不采用「优先 darkAlgorithm 派生」路线（Phase 89 已实证派生错误色相）。Reversibility: costly。
- **D-03:** 视觉验收分工「92 机器验收，93 人工巡检」——Phase 92 以 computed-style 断言 + 21 形态矩阵脚本完成机器验收；需 AD 登录的 14 页人工巡检截图移交 Phase 93 终验，登记显式移交表（Phase 90 先例）。Reversibility: reversible。
- **D-04:** 执行波次「3 波：扫描 → R6 → R7」——Wave 1 扫描与登记表；Wave 2 删 R6 主体+扩 token+验证；Wave 3 删 R7 残块+量化终态达标+矩阵机器验收+移交登记。删除为块级操作（ROADMAP 红线），每波独立 commit 可回滚，先扩 token 后删块同 commit。Reversibility: costly。

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

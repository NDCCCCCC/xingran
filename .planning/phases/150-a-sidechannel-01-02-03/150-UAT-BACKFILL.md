# Phase 150 · 132/133 backfill UAT 作废与重跑登记

> 依据：`150-PLAN.md` 裁决 D-5、ROADMAP :159/:186/:200、用户 2026-11-09 DECISION-①（允许推翻「范围快照回显」产品语义）。
> 本文件为 **新建登记件**，只登记作废 / 改写 / 重跑窗口 / 边界，不勾选任何完成、不写生产数据、AI 不擅自代替 QA 验收。
> **`.planning/milestones/` 下所有归档件本期只读**（`132-SUMMARY.md`、`133-UAT.md` 等一字不改）。

---

## 1. 作废清单（因 DECISION-① 推翻「范围快照回显」语义）

| # | 归档位置（只读） | 作废内容 | 作废原因 |
|---|------------------|----------|----------|
| 1 | `132-SUMMARY.md:65` | 「`BackfillDeptResult` 回显范围快照（档位/部门名/总数/FailClosed），预览与执行两次请求各自从同一 `ScopeResult` 派生」及「COUNT 保持全库口径 = 请求行数」双口径 | Phase 150 删除 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal/SkippedNoUser/SkippedBadUserDept` 六字段并整段删除全库 `countQ`（D-1/D-3）；响应体不再回显任何全库 / 越界规模 |
| 2 | `132-SUMMARY.md:74` | 四类「本次范围：…（共 N 条）」确认框文案：All `全库（共 N 条）` / Custom·Dept·DeptChild `{部门名}及子部门（共 N 条）`（含「等 M 个部门」）/ Self `仅本人关联工位（共 N 条）` / FailClosed | `useBackfillDept.tsx` 的 `rangeText()` 改为不含任何数值的可见集内表述；「共 N 条」「等 M 个部门」全部作废（T3 / D-1） |
| 3 | `132-SUMMARY.md:103` | 「P1-5（审查）已知残余风险、接受；本 phase 只做到不逐条回显明细 + 聚合计数入 operlog，**不假装已解决**」裁定 | 该「接受残余」前提已被 DECISION-① 推翻：① 纳入 A+B 根治，A(150) 已完成响应体去规模化；本登记件替代其「未解决」结论作为 UAT 依据 |

> 说明：以上均为**只读归档件**的语义作废登记，不修改归档原文；作废后相关 UAT 结论须按第 3 节重跑窗口重新裁定。

---

## 2. 改写登记（133-UAT.md A6 段）

| 归档位置（只读） | 原结论 | 改写后的验收口径 |
|------------------|--------|------------------|
| `133-UAT.md` A6「越界计数入 operlog、不整批 403」 | 越界行不写入、不阻断整批，`ImportResult.Errors`（含行号）与 `ScopeOutOfScope` 回显越界计数，聚合入 operlog | **越界计数仅入 operlog、HTTP 响应体零回显、仍不整批 403**：越界行静默跳过（不 `Failed++`、不 append `Errors`、不回显行号），`ImportResult.ScopeOutOfScope` 改 `json:"-"` 仅供 `excel_handler.go` 的 `operlog.WithOperParam` 读取；`gin.H` 形状 `inserted/updated/failed/errors` 不变，但 `failed`/`errors` 不再随越界行数变化（D-6 / F-4） |

---

## 3. 重跑窗口与验收人（QA 裁定，AI 不擅自勾选）

- **重跑范围**：132/133 backfill 相关的预览 / 执行确认框文案、Excel 导入越界行提示、admin（All 档）业务数值零回归、差分攻击（Dept / DeptChild 两相邻 scope、全库 N 不同但可见 M 相同）。
- **重跑窗口与验收人**：**由 QA 裁定重跑窗口与验收人，AI 不擅自勾选完成**；79 条人工验证跟踪表（`.planning/notes/261108-human-verification-tracker.md`）保持 **0 勾选**，按 `261108-uat-writeback-policy.md` 由人工执行后回写。
- **本期状态**：仅登记作废与改写，**不勾选任何完成项、不写生产数据、不伪造 UAT 已重跑完成**。

---

## 4. 边界声明（并行 phase 隔离）

- `133-UAT.md:79-91` 的 **D1 / D2 段归 Phase 149，本期一字不动**（其 D1/D2「确认未收口」结论作废与重跑窗口由 149 登记）。
- `.planning/milestones/` 归档件本期**只读**（含 `132-SUMMARY.md`、`133-UAT.md`）；ROADMAP :186 硬约束不改归档件。
- 本 phase 的 UAT 载体即本新建文件 `150-UAT-BACKFILL.md`，落在 SDK 已确定的 `.planning/phases/150-a-sidechannel-01-02-03/` 目录。
- 任务书点名的「只改 backfill 段，不动 149 的 D1/D2 段」已落实。

---

## 5. All 档零回归复核项（供 QA 重跑时对照）

- admin（All 档）的 `candidates` / `toFill` / `toCorrect` / `skippedAlreadyValid` / `samples` 与 130 前口径一致；`samples` 长度 ≤ 20。
- 前端可见「全部工位」措辞（`rangeText()` All 分支），但**不含「共 N 条」等任何数值**。
- **全档位统一删除 6 个规模字段**（D-1）：无 `if isAll` 档位分支回显；All 档响应体同样不含 `scopeTotal/scopeVisible/scopeOutOfScope/scopeDeptTotal/skippedNoUser/skippedBadUserDept`。
- `datascope` 未注入 ctx → All 兜底短路不变。

---

## 6. 不伪造闭环声明

- **B（Phase 151 统一 ScopedDB）未落地前不得宣称 ① 已根治**；本 phase 仅完成 A（响应体去规模化），不防回归。
- 本文不出现「UAT 已重跑完成 / 已验收 / 统一架构已落地 / PG 双跑通过」等不实闭环措辞。

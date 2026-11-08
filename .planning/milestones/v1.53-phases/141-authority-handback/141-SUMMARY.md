---
one-liner: 全仓清理临时清单权威引用、归档 OPEN-ITEMS.md 与历史 UAT 重建件、用 GSD milestone complete 把权威交回自动生成的 ROADMAP/REQUIREMENTS/STATE/MILESTONES
---

# Phase 141 — 登记接受残余 + 权威交回 GSD（v1.53 / ACK-01 + ACK-02 + GOV-01）

**执行**：2026-11-08，Lead 亲自收口（必须最后执行）。

## 背景裁定（用户 2026-11-08）
`OPEN-ITEMS.md` 只是本次里程碑的**临时输入清单**，不是长期权威；项目处理干净后须归档它、清理全部「权威 / 唯一入口」引用，**把权威交回 GSD 自动生成体系**（`ROADMAP.md` / `REQUIREMENTS.md` / `STATE.md` / `MILESTONES.md` + GSD SDK `milestone complete` 归档产物）。因此 GOV-01 口径为「临时清单归档 + 权威交回 + 一致性复核」，**不再**维护 A/B/C 节、「移入 C 节」或「唯一入口」。

## 交付清单

| 项 | 内容 |
|---|---|
| ACK-01/02 | `.planning/notes/261108-v153-locked-residuals.md`：v1.52 四项 LOCKED 残余 + B13 两项已知约束，逐条含裁定 / 残余风险 / 不改代码理由 / 触发复核条件 |
| Phase 141a | 独立智能体 cleanup：21 个文件 92 组精确替换，a 类 42 处权威表述 + b 类登记说明全部中性化；`REQUIREMENTS.md` 的 GOV-01 / VAL 回写 / Out of Scope / Core Value 冲突口径**在收档前改完**（避免被 GSD 原样固化进归档副本后 `git rm`） |
| 独立审计 | 智能体 auditor：全仓 120 处 OPEN-ITEMS 引用分类（a 42 / b 78）、`audit-open` 9 类检查与 suppression 机制、`milestone complete` 归档行为与 5 大遗漏风险 |
| 代码注释 | Lead 集成时清理最后 2 处（B8 / B13），中性化为「2026-11-08 开放项盘点 #Bxx」 |
| 旧权威降级 | `MILESTONE-v1.52-SUMMARY.md:2`、`MILESTONES.md:1`、`ARCHIVE-INDEX.md:3`、`HANDOFF.json`、`.audit-open-snapshot.json` 等由「权威 / 为准」降级为「收档时点快照」，当前状态收敛到 STATE / ROADMAP / MILESTONES |
| Phase 目录 | 4 份历史 UAT 重建件移入 `v1.52-phases/historical-uat/`（避免被误归档为 v1.53 产物），映射写入 `ARCHIVE-INDEX.md` §2.1；134-141 各建目录 + `NN-SUMMARY.md`（含 `one-liner` frontmatter 供 GSD 提取 accomplishments） |
| 收档动作 | GSD SDK `milestone complete v1.53 --name "开放项全量清偿" --confirm`：归档 ROADMAP/REQUIREMENTS 到 `milestones/v1.53-*`、移动 8 个 phase 目录、重写 STATE.md、append MILESTONES.md 条目；随后 ROADMAP.md 重组为分组摘要、PROJECT.md evolution review、`git rm REQUIREMENTS.md`（为下一里程碑清空） |
| 临时清单归档 | `OPEN-ITEMS.md` 移入 `.planning/milestones/v1.53-OPEN-ITEMS.md`（历史快照，其标题与自述权威声明随归档转为历史记录，不再是活跃权威） |

## 一致性复核
- 收档前全仓 `OPEN-ITEMS` 命中从 120+ 降至 35（全部为允许保留的中性表述 / 待收档删除的裁定段 / 临时清单本体），代码目录 0 命中。
- `ROADMAP.md:25/27` 的「权威回收裁定」与「ACK 共同约定」段在收档时删除（供其它智能体读取的临时指令，归档后无意义）。
- VAL-* 结果写入 GSD 产物（STATE.md Deferred Items / MILESTONES.md / 未来里程碑 UAT 清单），不回写临时清单。

## 回归闸门（Lead 集成期统一跑）
- `go build ./...` ✅ / `go test ./...` 全仓 ✅
- `npx tsc --noEmit` ✅ / `npm run lint:ci` ✅（eslint 0 + 全部契约脚本 ALL PASS + 路由注册表 708/314/139 OK）
- 集成期修复 2 个真实回归：`WorkOrderRatingModal.tsx` 的 `<Input.TextArea>` 违反 AppTextArea 契约、Phase 138 新路由未登记 `route-registry.json`。

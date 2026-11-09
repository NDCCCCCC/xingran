# 归档路径映射索引（2026-11-08 GSD 归档）

> 本文件是 **2026-11-08 `/gsd-complete-milestone v1.52`** 归档的路径换算对照表（历史快照，非当前状态权威）。
> 当前状态以 `.planning/STATE.md` frontmatter / `.planning/ROADMAP.md` / `.planning/MILESTONES.md`（GSD 自动生成体系）为准。
> 归档后 `.planning/phases/` 与 `.planning/quick/` 下的 v1.51/v1.52 产物已物理移动，
> 历史文档与代码注释里凡引用**旧路径**者，一律按下表换算；**旧路径不再代表文件不在**，请到新位置查。
> 本索引不改开放项登记，只解决路径失效；当前状态以 `.planning/STATE.md` / `.planning/ROADMAP.md` 为准。

## 1. 顶层归档产物

| 旧路径 | 新路径 | 说明 |
|---|---|---|
| `.planning/ROADMAP.md`（归档副本） | `.planning/milestones/v1.52-ROADMAP.md` | GSD 复制归档；**当前 `.planning/ROADMAP.md` 仍保留**（v1.52 SHIPPED 状态，未删除） |
| `.planning/REQUIREMENTS.md`（归档副本） | `.planning/milestones/v1.52-REQUIREMENTS.md` | GSD 复制归档；**当前 `.planning/REQUIREMENTS.md` 仍保留**（DS-01~07），因本次选择「只归档不提交」未执行 `git rm` |
| `.planning/milestones/v1.51-REQUIREMENTS.md` | 不变 | v1.51 既有归档 |

## 2. Phase 目录（`.planning/phases/*` → `.planning/milestones/v1.52-phases/*`）

| 旧路径 | 新路径 | 主要引用方（需换算） |
|---|---|---|
| `.planning/phases/129-real-device-uat/` | `.planning/milestones/v1.52-phases/129-real-device-uat/` | 2026-11-08 开放项盘点 A2/A3/A5（129-UAT-BACKFILL.md / 129-RESULTS.md / 129-HUMAN-UAT.md）、`PROJECT.md:217`、`phases/129-real-device-uat/129-*` 自身 |
| `.planning/phases/130-device-evidence/` | `.planning/milestones/v1.52-phases/130-device-evidence/` | `STATE.md:67-69`（历史目录编号消歧）、`internal/services/portcollection/device_capability.go:119` 注释、`130-RUIIJIE-TRUNK-ALLOWED.md:191` |
| `.planning/phases/130-scope-provider/` | `.planning/milestones/v1.52-phases/130-scope-provider/` | `MILESTONE-v1.52-SUMMARY.md:87`、`STATE.md:69` |
| `.planning/phases/131-read-scope/` | `.planning/milestones/v1.52-phases/131-read-scope/` | `MILESTONE-v1.52-SUMMARY.md:91`（131-CACHE-INVENTORY.md） |
| `.planning/phases/132-write-scope/` | `.planning/milestones/v1.52-phases/132-write-scope/` | `MILESTONE-v1.52-SUMMARY.md:87` |
| `.planning/phases/133-scope-final/` | `.planning/milestones/v1.52-phases/133-scope-final/` | 2026-11-08 开放项盘点 A8（133-UAT.md）、`MILESTONE-v1.52-SUMMARY.md:87,90`、`internal/services/operations/scope_entities.go:8` 注释 |

索引：`.planning/milestones/v1.52-phases/README.md`

## 2.1 历史 UAT 重建件（v1.53 Phase 140 产出，2026-11-08 补登记）

> Phase 140 按 2026-11-08 开放项盘点 A4/A5 重建了 4 份磁盘缺失的原始 UAT 清单；因属 v1.18/1.19/1.20/1.21 历史清单，为避免被 `milestone complete v1.53` 误归入 v1.53 产物，统一移入 v1.52 归档区。历史正文与本次产物凡引用旧路径者，按此表换算。

| 旧路径（重建时） | 新路径 | 归属 |
|---|---|---|
| `.planning/phases/48-component-serials/48-HUMAN-UAT.md` | `.planning/milestones/v1.52-phases/historical-uat/48-component-serials/48-HUMAN-UAT.md` | v1.18 3 项真机 SNMP/CLI |
| `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` | `.planning/milestones/v1.52-phases/historical-uat/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` | v1.19 7 项 SSH |
| `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md` | `.planning/milestones/v1.52-phases/historical-uat/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md` | v1.20 12 项 VLAN/绑定 |
| `.planning/phases/57-ad-outbox-lite/57-HUMAN-UAT.md` | `.planning/milestones/v1.52-phases/historical-uat/57-ad-outbox-lite/57-HUMAN-UAT.md` | v1.21 3 项 |

索引：`.planning/milestones/v1.52-phases/historical-uat/`。

## 3. Quick 任务目录（`.planning/quick/*` → `.planning/milestones/v1.52-quick/*`）

| 旧路径 | 新路径 |
|---|---|
| `.planning/quick/261102-m94-theme-style-chk-theme-style/` | `.planning/milestones/v1.52-quick/261102-m94-theme-style-chk-theme-style/` |
| `.planning/quick/261106-wsb-workstation-backfill-dept-button/` | `.planning/milestones/v1.52-quick/261106-wsb-workstation-backfill-dept-button/` |
| `.planning/quick/261106-wsd-workstation-dept-infer-from-user/` | `.planning/milestones/v1.52-quick/261106-wsd-workstation-dept-infer-from-user/` |
| `.planning/quick/261106-wso-workstation-floor-org-override/` | `.planning/milestones/v1.52-quick/261106-wso-workstation-floor-org-override/` |

索引：`.planning/milestones/v1.52-quick/README.md`（GSD 自动生成，含 4 条 SUMMARY 链接）。
`STATE.md` 的 `### Quick Tasks Completed` 表已按 GSD 归档语义清空数据行（表结构保留）。

## 4. 调试信息（`.planning/debug/`）

**不做物理移动**，理由（`.planning/notes/261102-planning-sync-strategy.md:36-38`）：`debug/**` 含 1 处真实 DB 明文口令与内网 IP 密集，既有策略为「留本地、不入库、入库前须脱敏」。

- 现状：223 个文件（含 `debug/resolved/` 37 个），绝大多数 `status: resolved/closed`，由 v1.23 Phases 69-75 全终结（`MILESTONES.md:52-56`）。
- GSD `audit-open` 收档审计：`debug_sessions` 计数 **0**（3 项为历史 acknowledged suppression），无未关闭调试项。
- 归档性质：全部为**历史调试记录/取证快照**，不驱动修复；当前状态见 `.planning/STATE.md` / `.planning/ROADMAP.md`。
- 唯一本次修过的文件：`debug/workstation-export-context-cancel.md`（去掉重复 `status:` 字段，避免 `audit-uat` 误报，见 `RETROSPECTIVE.md:383`）。
- 如需对 debug 目录做二次归档，须先按同步策略脱敏并单独批次评估。

## 5. 其它历史目录（未移动，仅登记）

| 目录 | 状态 | 说明 |
|---|---|---|
| `.planning/notes/` | 保留 | v1.17~v1.52 裁定/审计/基线，含 2026-11-08 开放项盘点引用的 4 份 `261107-v152-*-decision.md` |
| `.planning/seeds/` | 保留 | 3 份：`SEED-ad-push-enable-direction.md`（真实待办，2026-11-08 开放项盘点 A1）、另 2 份已标 SUPERSEDED/历史归档 |
| `.planning/todos/` | 保留 | `done/` + `completed/`，均已标历史归档 |
| `.planning/reports/` / `reviews/` / `spikes/` / `research/` / `codebase/` | 保留 | 历史审查/研究快照，时点记录不驱动修复 |
| （归档前临时备份目录） | 已清理 | 备份目录已清理，无残留 |

## 6. 历史文档里指向「更早 phase 目录」的引用

`MILESTONES.md:140,174`、`PROJECT.md:693`、`notes/260628-gorm-schema-drift-register.md:4`、`notes/2026-10-29-pg-data-scope-verification.md:4-5,175`、`reports/MILESTONE_SUMMARY-v1.18.md:132,175,185`、`debug/recon-uniq-index-23505.md:48`、`docs/deployment/login-encryption-deployment.md:746-747`、`docs/plans/2026-07-09-v1.20.1-design.md:303,368`、`internal/services/asset/fix_suggestion_monitor.go:20`、`fix_suggestion_service.go:27`、`internal/core/db/migrations/migration_167_workstation_backfill_status.go:20` 等，指向的是 **v1.16~v1.25 期间早已归档/清理的旧 phase 目录**（24-uuid / 34-oper-log-full-coverage / 42-r1 / 46-r5 / 48 / 54 / 56 / 84-pmodel-gaps / 57-ad-outbox-lite 等），**不是本次归档造成的失效**。这些是历史时点引用，按各文件既有的「历史归档」注记对待，勿据此新建修复任务。

---

**维护约定**：新增归档批次时在本文件追加映射段；旧路径引用不要逐个改写历史正文（会破坏可追溯性），以本索引换算为准。

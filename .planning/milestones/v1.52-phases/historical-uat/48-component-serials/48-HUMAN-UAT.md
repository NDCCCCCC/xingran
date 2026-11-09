---
rebuild: true
rebuild_date: 2026-11-08
rebuild_by: Phase 140 / VAL-04
original_file_missing_on_disk: true
original_refs:
  - .planning/OPEN-ITEMS.md #A5（本次里程碑的临时输入清单，取数用）
  - .planning/REQUIREMENTS.md VAL-04
  - .planning/MILESTONES.md:196-218（v1.18 收档：3 项 site-visit UAT deferred）
  - .planning/reports/MILESTONE_SUMMARY-v1.18.md:126-132,169-175（D-08/D-11/D-10 三项 + owner）
  - git 历史原件：.planning/milestones/phases-archive-pre-v1.21/48-device-component-serials-planned/48-HUMAN-UAT.md
  - 史实目录名：48-device-component-serials-planned（见 MILESTONE_SUMMARY-v1.18.md:132,175 / MILESTONES.md:218 / .planning/ARCHIVE-INDEX.md:63 旧 phase 目录为历史归档引用）
status: 重建件 / 原始文件已不在磁盘
milestone: v1.18
phase: 48-component-serials（史实目录名 48-device-component-serials-planned）
---

# Phase 48 人工 UAT 清单（重建件 / 原始文件已不在磁盘）

> ⚠️ **本文件为 2026-11-08 Phase 140（VAL-04）重建件。** 原始 `48-HUMAN-UAT.md` 已不在磁盘；
> 依据 `.planning/OPEN-ITEMS.md` #A5（本次里程碑的临时输入清单，取数用）、
> `.planning/MILESTONES.md` v1.18 收档段与 `.planning/reports/MILESTONE_SUMMARY-v1.18.md` 记载的
> 3 项 site-visit UAT，以及 git 历史中归档的原文件（提交 `5f7cf0b6f` 归档、`d3063a09f` 清理前）
> 逐项重建，条目内容与状态不改。
>
> **路径选择说明**：史实目录全名为 `48-device-component-serials-planned`
> （`MILESTONE_SUMMARY-v1.18.md:132,175`、`MILESTONES.md:218`、git 归档路径）；
> 本里程碑按 Phase 140 交付物清单统一使用 `48-component-serials/`。二者按
> `.planning/ARCHIVE-INDEX.md` §6「指向更早 phase 目录的引用属 v1.16~v1.25 已归档旧目录、按历史归档注记对待」
> 换算，对应同一 Phase 48，不另建并行清单。
>
> **口径**：3 项统一标「待执行（不标已完成）」；本文件不写生产数据、不切换生产配置。
> 实际执行由现场运维同事按 `.planning/notes/261108-uat-writeback-policy.md` 的口径完成；
> 执行完成后在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选，
> **未执行不得勾选**。回写与跟踪见 `.planning/notes/261108-human-verification-tracker.md`。

## 自动化闸门（史实已 PASS，本环境 2026-07-04 跑过；仅作背景，不代替人工项）

- `go build ./...` clean；`go test ./internal/services/component_collector/` 21 tests
- `go test ./internal/utils/operlog/` 5 regression tests（25 OperType + 11 sensitive keyword + Record 5 参签名不回归）
- `go test ./internal/services/operations/ -run TestAsset` 4 List/Statistics 过滤测试
- `go test ./internal/services/ -run TestCollectComponentInfo` 5 D-10 两步流水线测试
- `go test ./internal/api/v1/operations/ -run TestListComponents` 3 handler 测试
- 前端 `tsc --noEmit` clean / `npm run build`（1m 40s，vendor-react 775 kB gzip 基线零回归）
- psql schema introspection：`ops_asset` 4 新列 + `recon_category` + 索引 `uniq_recon_asset_type_cat_open` + 字典 seed OK
- Task 3 automated gate：`grep -q "Phase 48 真机 UAT" .planning/STATE.md` PASSED

## Tests（site-visit，3 项 deferred → 本跟踪口径统一「待执行」）

- [ ] **48-UAT-01**（D-08）真机 Huawei S8700 SNMP ENTITY-MIB 单 GET 路径
  - **验证内容 / 预期**：`EntityCollector.Collect` 在真实 S8700 上拿到 chassis / fan / engine 序列号；
    Huawei S8700 拒绝 GetBulk，改单 GET loop（D-08）。
  - **前置授权 / 环境**：现场访问 + S8700 真机接入（本环境无 S8700，仅 fixture-driven 单测）。
  - **执行人**：现场运维同事（owner，携带 S8700 接入）。
  - **证据位**：真机采集输出 + `ComponentListTab` 前端实测记录。
  - **回写状态**：待执行（不标已完成）；执行完成后按回写口径在 GSD 产物 / 未来里程碑 UAT 清单登记证据并勾选。

- [ ] **48-UAT-02**（D-11）真机 Ruijie RS8607E SNMP ENTITY-MIB 627 行含 352 `temprature*` 噪声过滤
  - **验证内容 / 预期**：D-11 filter（`Class==8 && strings.HasPrefix(Name, "temprature")`）
    把 Components 削减到 ≤20 真实组件，real PSU 保留。
  - **前置授权 / 环境**：现场访问 + RS8607E 真机接入（本环境无真机；synthetic-fixture test 已断言过滤规则，
    不验证 352 实际数字）。
  - **执行人**：现场运维同事。
  - **证据位**：真机 627 行原始输出 + 过滤后组件数 / PSU 保留结论。
  - **回写状态**：待执行（不标已完成）；执行完成后按回写口径登记证据并勾选。

- [ ] **48-UAT-03**（D-10）真机两步流水线（Huawei S8700 `10GE5/0/4` actually-up fiber interface）
  - **验证内容 / 预期**：`display interface status` 返回 up；`display interface transceiver` 返回 Vendor SN；
    down ports 跳过（D-10 enforcement 在 `cmd_dispatcher` + `runTwoStepTransceiverPipeline` 编排）。
  - **前置授权 / 环境**：现场访问 + S8700 光口可实测 CLI 两步流水线（本环境无 fiber port）。
  - **执行人**：现场运维同事。
  - **证据位**：两步命令输出 + `runTwoStepTransceiverPipeline` 实测记录。
  - **回写状态**：待执行（不标已完成）；执行完成后按回写口径登记证据并勾选。

## Summary（重建口径）

| Metric | Count |
|--------|-------|
| total | 3 |
| 待执行（不标已完成） | 3 |
| 已勾选 / 已完成（伪造） | 0 |

## Owner（史实）

现场访问时由运维同事携带 S8700 / RS8607E 接入，跑 `collectComponentInfo` cron + 前端 `ComponentListTab` 实测。
史实 owner 段原写「Site visit 完成后回写本文件」；按本里程碑权威回收裁定，回写目标改为
**GSD 自动生成的产物 / 未来里程碑的 REQUIREMENTS / UAT 清单**（见回写口径文档），不在本重建件上预置勾选。

## 关联声明（史实）

- `.planning/STATE.md` §Phase 48 真机 UAT deferred 声明（2026-07-04）
- `.planning/reports/MILESTONE_SUMMARY-v1.18.md` §4 / §6（3 项 site-visit + owner）
- `.planning/MILESTONES.md` v1.18 段（`Known deferred items at close`: 3 项 site-visit UAT, status `partial`）
- `.planning/ARCHIVE-INDEX.md` §6（历史 phase 目录引用按既有「历史归档」注记对待）

# Phase 48: Validation Map

**Generated:** 2026-07-04 (revision 1 — addresses gsd-plan-checker WARNING 1)
**Source:** `48-RESEARCH.md` §Validation Architecture (line 614-657) reconciled against `48-01/02/03-PLAN.md` actual task structure
**Purpose:** Map every REQ-48-XX → concrete test file + automated command. Track Wave 0 gaps.

---

## Test Framework

| Property | Value |
|----------|-------|
| Backend framework | Go `testing` (stdlib) |
| Frontend framework | vitest |
| Backend quick run | `go test ./internal/services/component_collector/...` |
| Backend full suite | `go test ./...` |
| Frontend type-check | `cd xingran-react-frontend && npm run type-check` |
| Frontend lint | `cd xingran-react-frontend && npm run lint` |

## Sampling Rate

- **Per task commit:** `go test ./internal/services/component_collector/...`(component_collector 包)
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green + frontend `npm run type-check` + `npm run lint` + manual UAT site visit deferred(RESEARCH §Environment Availability)

---

## REQ → Test Map (11 requirements)

| Req ID | Behavior | Test File | Test Names | Wave 0 Status |
|--------|----------|-----------|------------|---------------|
| REQ-48-01 | ops_asset 4 new cols + sys_data_reconciliation.recon_category col exist;new partial unique `uniq_recon_asset_type_cat_open` | `internal/services/operations/asset_listfilter_test.go` (48-01 T2) + migration smoke test(由 48-01 T1 build/vet 兜底,DB introspection 在 UAT 步骤 2) | `TestAssetListExcludesComponents`, `TestAssetStatisticsExcludesComponents`, `TestAssetListFilterDoesNotBreakExistingFilters` | Inline as RED-first in 48-01 T2(no separate 48-00) |
| REQ-48-02 | SNMP ENTITY-MIB single-GET parses Huawei/Ruijie sample ENTITY-MIB output correctly | `internal/services/component_collector/snmp_entity_collector_test.go` (48-02 T2a) | `TestEntityCollectorRuijieFiltersTemprature`, `TestEntityCollectorHuaweiDualClassDedup`, `TestEntityCollectorPreservesRealPowerSupply` + `internal/device/...` `TestCountPhysicalEntities`, `TestGetEntityAttrs` | Inline as RED-first in 48-02 T2a |
| REQ-48-03 | TextFSM templates parse all vendor CLI sample files(Huawei display device esn + transceiver;Ruijie show version modules + transceiver DDM) | `internal/services/component_collector/cli_huawei_collector_test.go` + `cli_ruijie_collector_test.go` (48-02 T2b/T2c) | `TestHuaweiCliParseDisplayDeviceEsn`, `TestHuaweiCliParseInterfaceStatus`, `TestHuaweiCliParseInterfaceTransceiver`, `TestHuaweiTransceiverFiltersDownPorts`, `TestRuijieCliParseShowVersionModules`, `TestRuijieCliParseInterfacesStatus`, `TestRuijieCliParseTransceiverDDM`, `TestRuijieTransceiverFiltersDownPorts` | Inline as RED-first in 48-02 T2b/T2c |
| REQ-48-04 | Ruijie `temprature*` filter drops 352 noise nodes under powerSupply(8) class(D-11) | `internal/services/component_collector/snmp_entity_collector_test.go` (48-02 T2a) | `TestEntityCollectorRuijieFiltersTemprature`, `TestEntityCollectorPreservesRealPowerSupply` | Inline as RED-first in 48-02 T2a |
| REQ-48-05 | Component lookup + UPDATE-only behaviour (never INSERT);on hit UPDATE 4 cols;on miss emit anomaly | `internal/services/component_collector/ops_asset_writer_test.go` (48-03 T1) | `TestOpsAssetWriterHitUpdatesFourColumns`, `TestOpsAssetWriterMissEmitsAnomaly`, `TestOpsAssetWriterParentMissingDoesNotEmitSwitchAnomaly` | Inline as RED-first in 48-03 T1 |
| REQ-48-06 | Reconciliation anomaly row appears with `recon_category='component_serial'`, conflict_type=F, idempotent on repeat emit | `internal/services/component_collector/reconciliation_emitter_test.go` (48-03 T1) | `TestReconciliationEmitterIdempotent` (+ asserted indirectly by `TestOpsAssetWriterMissEmitsAnomaly`) | Inline as RED-first in 48-03 T1 |
| REQ-48-07 | assetService.List()/Statistics() exclude component_type IS NOT NULL rows | `internal/services/operations/asset_listfilter_test.go` (48-01 T2) | `TestAssetListExcludesComponents`, `TestAssetStatisticsExcludesComponents`, `TestAssetListFilterDoesNotBreakExistingFilters` | Inline as RED-first in 48-01 T2 |
| REQ-48-08 | Parent-switch absent → parent_asset_id NULL, source_device_id still written, no anomaly | `internal/services/component_collector/ops_asset_writer_test.go` (48-03 T1) | `TestOpsAssetWriterParentMissingDoesNotEmitSwitchAnomaly` | Inline as RED-first in 48-03 T1 |
| REQ-48-09 | Stack-mode entPhysicalContainedIn tree resolution hook(D-12) | `internal/services/component_collector/owner_resolver_test.go` (48-02 T1) | `TestOwnerResolverSingleChassis`, `TestOwnerResolverStackMode`, `TestOwnerResolverOrphanComponent` | Inline as RED-first in 48-02 T1 |
| REQ-48-10 | operlog entry written per component UPDATE (module="资产管理", operType=14/OperTypeSync) | `internal/services/component_collector/ops_asset_writer_test.go` (48-03 T1) + `internal/utils/operlog/regression_test.go`(回归保护) | `TestOpsAssetWriterRecordsOperlog` | Inline as RED-first in 48-03 T1;regression_test.go 不回归 |
| REQ-48-11 | Existing cron hooks collect components (does not break chassis update path;D-14) + D-10 two-step pipeline ordering | `internal/services/device_info_collection_service_test.go` 或新 `component_pipeline_hook_test.go` (48-03 T2) | `TestCollectComponentInfoHuaweiTwoStepPipeline`, `TestCollectComponentInfoRuijieTwoStepPipeline`, `TestCollectComponentInfoFailureDoesNotBlock`, `TestListComponents` | Inline as RED-first in 48-03 T2 |

---

## Wave 0 Gap Handling

RESEARCH.md §Validation Architecture 声明"Wave 0 is 11 tests + 1 fixture file"且建议 `48-00-PLAN.md` for test scaffold。**修订决定(gsd-plan-checker WARNING 2)**:把 Wave 0 测试脚手架作为各 plan 内的 RED-first TDD 流程内联,不单独建 48-00-PLAN.md。

**理由:**
1. 各 plan 任务已是 `tdd="true"` + `<behavior>` 块 + RED-first 流程,等价于 Wave 0
2. 单独建 48-00-PLAN.md 会让测试文件先于实现文件 commit(空骨架),增加 review 噪声
3. 测试文件路径在各任务的 `<files>` 字段已明确,executor 不需要 48-00 指引

**Wave 0 gaps(updated):**
- [x] `internal/services/component_collector/component_set.go` + `owner_resolver.go` + `owner_resolver_test.go` — 内联在 48-02 T1
- [x] `internal/services/component_collector/snmp_entity_collector_test.go` — 内联在 48-02 T2a
- [x] `internal/services/component_collector/cli_huawei_collector_test.go` — 内联在 48-02 T2b
- [x] `internal/services/component_collector/cli_ruijie_collector_test.go` — 内联在 48-02 T2c
- [x] `internal/services/component_collector/fixtures_loader.go` — 内联在 48-02 T2a
- [x] `internal/services/component_collector/ops_asset_writer_test.go` — 内联在 48-03 T1
- [x] `internal/services/component_collector/reconciliation_emitter_test.go` — 内联在 48-03 T1
- [x] `internal/services/component_collector/pipeline_test.go` — 内联在 48-03 T1
- [x] `internal/services/operations/asset_listfilter_test.go` — 内联在 48-01 T2
- [x] collectComponentInfo / ListComponents 测试 — 内联在 48-03 T2
- [x] fixtures 来源:`templates/samples/*.txt`(35 个真机样本:29 huawei + 6 ruijie,直接读;**INFO 1 修订** — 不复制到 `templates/parser/samples_*.txt`,RESEARCH 第 656 行的"copies"条目从 Wave 0 列表移除)

---

## Manual / UAT Coverage (cannot be automated)

| Item | Verification Path | Deferred To |
|------|-------------------|-------------|
| Migration 201 实际在 PostgreSQL 应用成功 | UAT 步骤 1(启动 backend 观察日志)+ 步骤 2(`\d ops_asset` / `\d sys_data_reconciliation` / `pg_indexes` 查询) | User site visit |
| 真机 SNMP ENTITY-MIB(S8700 / RS8607E)采集路径 | 真机 UAT | User site visit(RESEARCH §Environment Availability 声明) |
| 真机 CLI(display device esn / show version / display interface transceiver / show interface transceiver)解析 | 真机 UAT | User site visit |
| D-10 真机两步流水线在 Huawei S8700 10GE5/0/4 实际 up 接口下走通 | 真机 UAT | User site visit |
| 前端「从属组件清单」Tab 实际渲染 | UAT 步骤 3(前端资产列表展开行) | Local 可做(不需真机) |
| 现有对账看板 recon_category=component_serial 单独筛选 | UAT 步骤 4 | Local 可做 |

---

## Notes

- **Fixture 计数策略(WARNING 3):**`fixtures_loader.go` 用 `filepath.Glob("templates/samples/*.txt")` 动态计数,断言 `CountFixtures() > 0` 而非写死具体数。当前实际 35 个(29 huawei + 6 ruijie;`_run.log` 非 .txt 不计入),未来样本增减不需改测试。
- **D-10 强制测试覆盖:**REQ-48-03 含两个硬性 D-10 测试(`TestHuaweiTransceiverFiltersDownPorts` + `TestRuijieTransceiverFiltersDownPorts`),分别在华为/锐捷 CLI 采集器侧断言 down 接口 transceiver 行被丢弃。
- **DDM 死代码消除(WARNING 4):**`TestRuijieCliParseTransceiverDDM` 显式断言 `RuijieCliCollector.ParseTransceiverDDM` 调用 `_ddm.textfsm` 解析 Vendor Serial Number + DDM Bias/Tx/Rx Power,消除模板未被引用的死代码风险。

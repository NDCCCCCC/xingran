---
phase: 49-v1-18-gap-closure
verified: 2026-07-06T08:30:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: none
  notes: "Initial verification — no prior VERIFICATION.md existed (Step 0 confirmed)"
---

# Phase 49: v1.18 组件采集 gap closure — 验证报告

**Phase Goal:** 修复 v1.18 组件采集链路 3 个连锁缺口(Gap 3 chassis SN 写回 → Gap 2 parent_asset_id 关联 → Gap 1 collectComponentInfo 接入解析器),使 chassis 资产展开「从属组件清单」Tab 能正确显示板卡组件。端到端验证目标:RS8607E-03 (devicesn=G1M9140000175) 展开显示 ≥6 条板卡。

**Verified:** 2026-07-06T08:30:00Z
**Status:** passed
**Re-verification:** No — 初始验证(Step 0 确认 `.planning/phases/49-v1-18-gap-closure/` 下无前置 VERIFICATION.md)

---

## Goal Achievement

### Observable Truths

合并来源:ROADMAP Phase 49 success criterion(端到端 ≥6 条板卡)+ 49-01-PLAN frontmatter(4 truths)+ 49-02-PLAN frontmatter(7 truths,与 49-01 有重叠 chassis SN 真值,去重后保留 49-02 措辞)。所有 truths 必须全部通过 — 验证策略对每条做 3 层(存在/实质/接线)+ 数据流追溯。

| # | Truth | Status | Evidence(读码 + 测试 + E2E) |
|---|-------|--------|------------------------------|
| 1 | Ruijie show version 经 ParseShowVersionModules 解析后 chassis SN 写入 `sys_network_device.serial_number` | ✓ VERIFIED | `device_info_collection_service.go:301` CollectDeviceInfo 命令循环调 `enrichChassisSerial`;`:348-384` 实现,`:364` 调 `NewRuijieCliCollector().ParseShowVersionModules`;only-if-empty guard 在 `:353`(enrichChassisSerial)+ `:394`(updateDeviceInfo)。`TestCollectDeviceInfo_RuijieChassisSN` 用真机 fixture `ruijie_10_62_63_21_show_version.txt` 断言 SN=`G1HLC0R000096` PASS。E2E(49-02-SUMMARY):RS8607E-03 serial_number=`G1M9140000175` 已落库,14 台在线设备 serial_number 已填充。 |
| 2 | Huawei display device esn 经 ParseDisplayDeviceEsn 解析后 chassis ESN 写入 serial_number | ✓ VERIFIED | `:366` 调 `NewHuaweiCliCollector().ParseDisplayDeviceEsn`;`:313` getCommandsByVendor huawei 分支含 `"display device esn"`。`TestCollectDeviceInfo_HuaweiChassisESN` 断言 ESN=`102599861597` PASS;`TestCollectDeviceInfo_HuaweiEsn_UnrecognizedCommand` 验证退役命令语义(空 SN + nil err)PASS。 |
| 3 | 现有 Model/SoftwareVersion/Uptime 字段不被破坏,继续走 only-if-empty 语义 | ✓ VERIFIED | `:298` `parseDeviceInfo`(legacy 字符串解析)与 `:301` `enrichChassisSerial` 在循环内**串行叠加**,非替换;`:391-402` updateDeviceInfo 双层 only-if-empty guard。`TestCollectDeviceInfo_LegacyParseStillRuns` 断言 ruijie legacy parser 仍工作 PASS;`TestCollectDeviceInfo_ChassisSN_DoesNotOverwriteExisting` 断言 enrichChassisSerial 不覆盖已有 SN PASS。 |
| 4 | collectComponentInfo 在 transceiver pipeline 之前调用 ParseShowVersionModules 提取板卡(engine/card) | ✓ VERIFIED | `:713` collectComponentInfo Step 1 调 `collectBoardsInto`;`:794` collectBoardsInto 调 `NewRuijieCliCollector().ParseShowVersionModules(raw)`。`TestCollectBoardsInto_RuijieBoards` 用真机 fixture 断言 ≥6 板卡(engine/card)PASS(实读 fixture M1/1-5/M2)。 |
| 5 | 提取的板卡组件(丢弃 chassis 行)合并进 ComponentSet.Components,通过 Pipeline.Run 写入 ops_asset;set.Chassis 保持 nil | ✓ VERIFIED | `:807-809` collectBoardsInto 显式 `if c.ComponentType == ComponentTypeChassis { continue }`;`:708` set 初始化时 Chassis 不赋值;`:729` 仅 append xceiverSet.Components;`pipeline.go:108` `writer.Write(ctx, device.ID, parentAssetID, set.Components)` 不消费 `*set.Chassis`(grep 验证 pipeline.go 中无 Chassis 引用)。`TestCollectBoardsInto_ChassisRowDropped` 双重断言 set.Chassis==nil + set.Components 不含 chassis 类型 PASS。 |
| 6 | parent_asset_id 通过 `ops_asset.devicesn = sys_network_device.serial_number`(由 49-01 填充)匹配 chassis 资产 | ✓ VERIFIED | Gap 2 无代码改动(符合计划)。`device_info_collection_service.go:749` devRef.SerialNumber = device.SerialNumber;`pipeline.go:95` Pipeline.Run 调 `assetSvc.GetByDeviceSN`;`:855` cronAssetLookup.GetByDeviceSN 用 `WHERE devicesn = ? AND deleted_at IS NULL`;`ops_asset_writer.go:126` 写 `parent_asset_id`。E2E(49-02-SUMMARY):9 条组件全部 parent_asset_id=`0c3ac91f-...`(RS8607E-03 chassis asset id),关联链通畅。 |
| 7 | RS8607E-03 部署后展开「从属组件清单」Tab 显示 ≥6 条板卡组件 | ✓ VERIFIED | E2E checkpoint(49-02-PLAN Task 2,blocking gate)由运维同事现场执行,2026-07-05 PASSED。生产 PG 实测:RS8607E-03 (devicesn=G1M9140000175) 下 7 条 engine/card(Slot M1/M2 + Slot 1-5)+ 2 条 transceiver = 9 条,超过 ≥6 门槛。49-HUMAN-UAT.md frontmatter `status: passed`,4 项 UAT 中 3 项 passed + 1 项 deferred(浏览器渲染 — 数据层已证实)。 |

**Score:** 7/7 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/services/device_info_collection_service.go` | collectDeviceInfo/collectComponentInfo/enrichChassisSerial/collectBoardsInto/getCommandsByVendor 接入 chassis+board 解析器并合并 ComponentSet | ✓ VERIFIED | 903 行(改动前 ~700 行);`ParseShowVersionModules` 引用 2 处(:364 chassis SN, :794 boards);`ParseDisplayDeviceEsn` 引用 1 处(:366);错误注释 `already collected by the chassis collector` 已替换为历史错误说明(:680)。Level 4 数据流追溯:真机 fixture → parser → set.Components → Pipeline.Run → OpsAssetWriter.Write → ops_asset。 |
| `internal/services/device_info_collection_service_test.go` | 5 个 TestCollectDeviceInfo* 单测覆盖 chassis SN 提取+写回 | ✓ VERIFIED | 126 行,5 测试全部 PASS(RuijieChassisSN/HuaweiChassisESN/HuaweiEsn_UnrecognizedCommand/ChassisSN_DoesNotOverwriteExisting/LegacyParseStillRuns)。非 stub — 实读 `templates/samples/` 真机 fixture,断言具体 SN 值。 |
| `internal/services/device_info_collection_service_boards_test.go` | 5 个 TestCollectBoardsInto* 单测覆盖板卡采集 + chassis 行丢弃 + 容错 | ✓ VERIFIED | 151 行,5 测试全部 PASS(RuijieBoards ≥6/ChassisRowDropped/HuaweiNoBoards/OutOfScopeVendor/CommandErrorTolerated)。非 stub — 实读真机 fixture,断言 ≥6 板卡 + chassis 行不进 set.Components。 |
| `internal/services/component_collector/cli_ruijie_collector.go` | ParseShowVersionModules 解析 show version 输出为 chassis + slot 组件 | ✓ VERIFIED | 未修改(Phase 48 已实现);`:51-84` 实现,`:60-67` chassis 行,`:75-81` slot 行(M1/M2→engine,数字→card);既有测试不回归。 |
| `internal/services/component_collector/cli_huawei_collector.go` | ParseDisplayDeviceEsn 解析 display device esn 为 chassis 组件 | ✓ VERIFIED | 未修改;`:60-79` 实现,退役命令返回空切片+nil err(Pitfall 3);既有测试不回归。 |
| `internal/services/component_collector/reconciliation_emitter.go` | dedup query 单子句 `recon_category = ?`(跨方言修复) | ✓ VERIFIED | `:68` `Where("asset_id = ? AND conflict_type = ? AND recon_category = ?", ...)`;原 `IS ?` 已移除(grep `recon_category IS \?` 仅在注释 :59 历史说明出现,无活跃代码);commit d9a59468。E2E 验证 component_serial anomaly 已落 sys_data_reconciliation。 |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| CollectDeviceInfo(cmd loop) | enrichChassisSerial | 函数调用 | ✓ WIRED | `:301` 每条命令后调;`isChassisSNCommand` 过滤避免无关命令重复解析 |
| enrichChassisSerial(ruijie) | NewRuijieCliCollector().ParseShowVersionModules | 函数调用 | ✓ WIRED | `:364`,结果按 ComponentTypeChassis 过滤写入 info.SerialNumber |
| enrichChassisSerial(huawei) | NewHuaweiCliCollector().ParseDisplayDeviceEsn | 函数调用 | ✓ WIRED | `:366`,同上 |
| collectComponentInfo | collectBoardsInto | 函数调用 | ✓ WIRED | `:713` Step 1,失败容忍 |
| collectBoardsInto | ParseShowVersionModules | 函数调用 + chassis 行丢弃 | ✓ WIRED | `:794`,chassis 行 `continue`(:807-809) |
| collectComponentInfo | runTwoStepTransceiverPipeline | 函数调用 | ✓ WIRED | `:719` Step 2,只 append Components,忽略 Chassis 字段 |
| collectComponentInfo | Pipeline.Run | 函数调用 | ✓ WIRED | `:752`,传 devRef + set |
| Pipeline.Run | cronAssetLookup.GetByDeviceSN(devicesn 列) | 函数调用 | ✓ WIRED | `pipeline.go:95` + `:855` |
| Pipeline.Run | OpsAssetWriter.Write | 函数调用 | ✓ WIRED | `pipeline.go:108`,写 parent_asset_id (`ops_asset_writer.go:126`) |
| updateDeviceInfo | db.Updates(serial_number=info.SerialNumber) | only-if-empty UPDATE | ✓ WIRED | `:394-396` 双层 guard |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| enrichChassisSerial | info.SerialNumber | ParseShowVersionModules/ParseDisplayDeviceEsn → CHASSIS_SN/ESN template 字段 | ✓ 真机 fixture G1HLC0R000096 / 102599861597 | ✓ FLOWING |
| collectBoardsInto | set.Components | ParseShowVersionModules → engine/card 行 | ✓ 真机 fixture M1/1-5/M2 7 行 | ✓ FLOWING |
| Pipeline.Run | parentAssetID | cronAssetLookup.GetByDeviceSN → ops_asset.id (devicesn 列) | ✓ E2E: RS8607E-03 → 0c3ac91f-... | ✓ FLOWING |
| OpsAssetWriter.Write | ops_asset 行(parent_asset_id, component_type, ...) | set.Components per-SN UPSERT | ✓ E2E: 9 行落表(7 engine/card + 2 transceiver) | ✓ FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| TestCollectDeviceInfo*(5 测试) | `go test ./internal/services/ -run TestCollectDeviceInfo -v -count=1` | 5/5 PASS | ✓ PASS |
| TestCollectBoardsInto*(5 测试) | `go test ./internal/services/ -run TestCollectBoardsInto -v -count=1` | 5/5 PASS | ✓ PASS |
| collector 回归 | `go test ./internal/services/component_collector/ -count=1` | ok 1.549s | ✓ PASS |
| 全量构建 | `go build ./...` | exit 0 | ✓ PASS |
| emitter 单子句确认 | grep `recon_category IS \?` reconciliation_emitter.go(活跃代码) | 0 匹配(仅注释 :59 提及历史) | ✓ PASS |
| 错误注释清除确认 | grep `already collected by the chassis collector` device_info_collection_service.go | 1 匹配,位于 :680 历史说明段(非活跃误导注释) | ✓ PASS |

---

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| (无 probe 脚本) | — | — | SKIP — 本阶段未声明 probe,Phase 49 PLAN/SUMMARY 用 go test + E2E SQL 校验,无 `scripts/*/tests/probe-*.sh` |

---

### Requirements Coverage

无 `.planning/REQUIREMENTS.md`(本仓库未维护此文件)。Phase 49 source-of-truth 是 `.planning/ROADMAP.md:215` Phase 49 段(Requirements: TBD,源自 49-CONTEXT.md 的 3 个连锁缺口 + 端到端验证目标)。Plan frontmatter 用 informal ID `GAP-1`/`GAP-2`/`GAP-3`/`E2E`。

| ID | Source Plan | Description | Status | Evidence |
|----|-------------|-------------|--------|----------|
| GAP-3 | 49-01-PLAN | chassis SN 解析器接入,回填 sys_network_device.serial_number | ✓ SATISFIED | Truth 1+2+3,enrichChassisSerial + getCommandsByVendor huawei esn 命令,E2E 14 台填充 |
| GAP-1 | 49-02-PLAN | collectComponentInfo 接入 ParseShowVersionModules(板卡)+ chassis 行丢弃 | ✓ SATISFIED | Truth 4+5,collectBoardsInto + Pipeline.Run,E2E RS8607E-03 = 7 板卡 |
| GAP-2 | 49-02-PLAN | parent_asset_id 通过 devicesn=sn 关联(无代码改动) | ✓ SATISFIED | Truth 6,cronAssetLookup.GetByDeviceSN(devicesn 列) + OpsAssetWriter.Write(parent_asset_id),E2E 9 行 parent_asset_id 正确 |
| E2E | 49-02-PLAN Task 2 | RS8607E-03 展开显示 ≥6 条板卡(blocking checkpoint) | ✓ SATISFIED | Truth 7,生产 PG 实测 9 条组件(≥6 门槛),运维同事 2026-07-05 现场确认 |

无 orphaned requirements。

---

### Anti-Patterns Found

对 phase 改动的 6 个文件(device_info_collection_service.go + 2 测试文件 + cli_ruijie_collector.go + cli_huawei_collector.go + reconciliation_emitter.go)扫描 TBD/FIXME/XXX/PLACEHOLDER/return null/return []/console.log/HACK:

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (无) | — | — | — | 目标文件无 debt markers、无 stub return、无 placeholder 字符串。`device_info_collection_service.go` 仅 `:680` 注释中提及历史错误注释文本 `already collected by the chassis collector`(作为修正说明,非活跃代码)。 |

零 blocker 反模式。零 warning 反模式。

---

### Deferred Items(非 gaps,已在 SUMMARY/UAT 文档化)

这些条目在 49-02-SUMMARY.md `Deferred items` 段 + 49-HUMAN-UAT.md `Gaps` 段已明确登记,均不在 Phase 49 scope 内:

| # | Item | 文档化位置 | 处理建议 |
|---|------|-----------|---------|
| 1 | Huawei S5735 固定形态设备 `display device esn` 返回 `ESN of slot 1`(非 chassis)→ textfsm 模板未覆盖 | 49-02-SUMMARY L174 | 后续 phase: textfsm 加 slot fallback 匹配 |
| 2 | 连接池 `当前=23, 最大=20` TOCTOU 竞态 + 偶发卡死(`connection_pool.go:266-273`) | 49-02-SUMMARY L175 | 后续 follow-up: poolLock 或调高 MaxConnections |
| 3 | `net_device_enrichment_task` running 僵尸任务(31 条)阻塞 Enqueue dedup,recoverPendingTasks 仅恢复 pending | 49-02-SUMMARY L176 + L161 | 后续 follow-up: recoverPendingTasks 同时清理过期 running |
| 4 | 浏览器 UI「从属组件清单」Tab 渲染最终确认(数据层已证实 9 条) | 49-HUMAN-UAT.md L91-93 | 下次现场访问最终确认;非阻塞,SQL 数据层达标 ≥6 |

按 Step 9b 检查后续 milestone phase 是否覆盖:本 phase 是 v1.18 gap closure,后续 milestone v1.19 ROADMAP 尚未规划(`.planning/milestones/v1.18-ROADMAP.md` 已归档,v1.19 未建)。这些 deferred items 不影响 Phase 49 goal(RS8607E-03 ≥6 板卡已达成 9 条),归为 follow-up backlog 而非本阶段 gaps。

---

### Human Verification Required

无新增 human verification 需求。E2E checkpoint(49-02-PLAN Task 2,`type="checkpoint:human-verify" gate="blocking"`)已在 2026-07-05 由运维同事现场执行通过并记录于 49-02-SUMMARY.md + 49-HUMAN-UAT.md(`status: passed`,4 项中 3 passed + 1 deferred 浏览器渲染)。本验证为代码层确认,production E2E 已完成。

---

### Gaps Summary

无 gaps。所有 7 个 must-have truths VERIFIED,6 个 artifacts 全部 VERIFIED(存在 + 实质 + 接线 + 数据流追溯),10 个 key links 全部 WIRED,行为学验证 6 项全 PASS,emitter IS-string SQL bug 修复确认,错误注释清除确认。E2E 结果(9 条组件,超 ≥6 门槛)与代码实现一致。Phase goal 完整达成。

---

## 验证结论

**PASSED** — Phase 49 v1.18 gap closure 三个连锁缺口(Gap 3 → Gap 2 → Gap 1)在代码层完整修复,production E2E(RS8607E-03 = 9 条组件)超过 ≥6 门槛。可推进下一阶段。

---

_Verified: 2026-07-06T08:30:00Z_
_Verifier: Claude (gsd-verifier)_

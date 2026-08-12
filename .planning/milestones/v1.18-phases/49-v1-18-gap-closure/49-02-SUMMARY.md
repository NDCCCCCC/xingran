---
phase: 49-v1-18-gap-closure
plan: 02
subsystem: device-info-collection
tags: [board-collection, gap-1, gap-2, ruijie, component-collector, dead-code-fix]
requires:
  - "component_collector.ParseShowVersionModules (Phase 48 Wave 2)"
  - "component_collector.ComponentSet / Component / ComponentType constants"
  - "49-01 enrichChassisSerial (serial_number fill unlocks Gap 2 association)"
provides:
  - "device_info_collection_service.collectBoardsInto — board (engine/card) collection helper"
  - "collectComponentInfo 不再是 dead code: ruijie 板卡(M1/1-5/M2)通过 ParseShowVersionModules 提取并合并到 ComponentSet.Components"
  - "ops_asset 出现 component_type IN ('engine','card') 的板卡行(下一 cron 周期生效,E2E 待人工验证)"
affects:
  - "Phase 48 组件清单 Tab: RS8607E-03 展开「从属组件清单」应显示 ≥6 条板卡(待 E2E 部署验证)"
  - "前端资产管理 component_type 过滤: chassis 行被显式丢弃,不会污染前端 Tab"
tech-stack:
  added: []
  patterns:
    - "Additive composition: collectBoardsInto 单独负责板卡采集, runTwoStepTransceiverPipeline 单独负责光模块, collectComponentInfo 合并两者到 set.Components"
    - "Chassis row dropping: BLOCKER-2 设计 — Pipeline.Run 只消费 set.Components,从不消费 *set.Chassis; chassis 行在此处被显式丢弃避免污染 ops_asset"
    - "D-14 fault tolerance 分层: 板卡采集失败 → 记录后继续 transceiver 流程; transceiver 失败但有板卡 → 继续 Pipeline.Run; 两者全失败 → return nil(不阻塞 chassis 更新)"
key-files:
  created:
    - internal/services/device_info_collection_service_boards_test.go
  modified:
    - internal/services/device_info_collection_service.go
decisions:
  - "Chassis 行在 collectBoardsInto 内显式丢弃(选项 A),保持 set.Chassis == nil — Pipeline.Run(pipeline.go:108)只消费 set.Components,Pipeline 从不引用 *set.Chassis"
  - "M1 主控板卡(SN 在真机上等于 chassis SN)仍作为 engine 类型保留在 set.Components 中,因 M1 的 ComponentType 是 ComponentTypeEngine,与被丢弃的 chassis 行独立"
  - "Huawei 板卡走 ENTITY-MIB SNMP 路径(D-08 deferred),collectBoardsInto 对 huawei 直接 no-op —— 49-01 已处理 huawei chassis SN 写回 sys_network_device,不在此重复"
  - "Gap 2(parent_asset_id 关联)无需代码改动 —— 49-01 填充 sys_network_device.serial_number 后,既有 cronAssetLookup.GetByDeviceSN(line 766-767)用 devicesn 列匹配自然打通"
  - "Transceiver 失败但板卡有数据时不再 return err,而是记录后继续 Pipeline.Run(原 dead-code bug 修复)"
metrics:
  duration: ~20 min (code) + E2E validated 2026-07-05
  completed: 2026-07-05
  tasks: 2/2 (Task 1 code + Task 2 E2E checkpoint PASSED)
  files: 2
---

# Phase 49 Plan 02: 板卡组件采集接入 Summary

**One-liner:** 修复 collectComponentInfo dead-code 根因 —— 接入 ParseShowVersionModules 提取 ruijie 板卡(M1/1-5/M2)合并进 ComponentSet.Components 喂给 Pipeline.Run,显式丢弃 chassis 行避免污染前端 Tab。Gap 2 关联通过 49-01 serial_number 填充 + 既有 GetByDeviceSN 自然打通,无需额外代码改动。

## What Was Built

### 问题根因(Gap 1,诊断已确认)

`collectComponentInfo`(device_info_collection_service.go:687)是 dead code:
- 只调用了光模块 transceiver pipeline(runTwoStepTransceiverPipeline)
- **从未调用**现成的、已通过单测验证的 `ParseShowVersionModules` / `ParseDisplayDeviceEsn`
- 自欺欺人的注释 `"already collected by the chassis collector"` 引用了一个**根本不存在**的 collector
- 结果:`ops_asset WHERE component_type IS NOT NULL` = 0 条,所有 ruijie/huawei 设备前端「从属组件清单」Tab 空

### 实现

1. **新增 `collectBoardsInto(device, runner, set)` 辅助函数**(line 755-810):
   - **Ruijie**: 通过 `runner("show version")` 取输出 → `NewRuijieCliCollector().ParseShowVersionModules(raw)`。遍历返回的 `[]Component`:**`ComponentType==ComponentTypeChassis` 的元素跳过**(continue,不 append)—— chassis 资产已存在(parent 本身),Pipeline.Run 也只消费 set.Components。其余(engine/card)append 到 set.Components。
   - **Huawei**: 直接 return nil(no-op)。板卡走 ENTITY-MIB SNMP 路径(D-08 deferred),49-01 已处理 huawei chassis SN 写回,不在此重复。
   - **H3C/Maipu/unknown**: 直接 return nil。沿用既有 line 691-694 短路语义。
   - **D-14 fault tolerance**: 命令执行 error / parser error → applogger.Infof 记录后 return nil(不阻塞 transceiver pipeline)。

2. **重构 `collectComponentInfo`**(line 687-753):
   - 显式初始化 `set := &component_collector.ComponentSet{Components: []component_collector.Component{}}`
   - Step 1: 调用 `collectBoardsInto` 填板卡(失败容忍,记日志)
   - Step 2: 调用 `runTwoStepTransceiverPipeline` 取光模块,把它的 Components append 到 set.Components(**忽略**它返回的 Chassis 字段,保持 set.Chassis == nil)
   - Step 3 (修复 line 705 dead-code bug): `if len(set.Components) == 0 { return nil }` —— 只要板卡有结果(即便 transceiver 空)就继续 Pipeline.Run
   - 调用 `pipeline.Run(ctx, devRef, set)`(已存在)

3. **删除/修正错误注释**: 原 `"already collected by the chassis collector"` 注释被替换为正确的设计说明,解释 49-02 Gap-1 修复以及 SNMP ENTITY-MIB deferred 状态(D-08)。

### Gap 2 — 无代码改动

板卡组件 `parent_asset_id` 关联通过既有链路自然打通:
- 49-01 填充 `sys_network_device.serial_number`(ruijie fixture G1HLC0R000096)
- `collectComponentInfo` 构造 `devRef.SerialNumber = device.SerialNumber`(line 750 已传,未改)
- `Pipeline.Run`(pipeline.go:92-104)调 `assetSvc.GetByDeviceSN(device.SerialNumber)`
- `cronAssetLookup.GetByDeviceSN`(line 766-767)用 `WHERE devicesn = ? AND deleted_at IS NULL` 匹配 ops_asset
- 匹配到的 ops_asset.id 作为 `parentAssetID` 传给 `writer.Write`

→ 板卡组件写入 ops_asset 时 `parent_asset_id` 自动指向 chassis 资产。Gap 2 无需额外代码改动,符合计划预期。

## Verification

| Check | Command | Result |
|-------|---------|--------|
| Target tests (RED→GREEN) | `go test ./internal/services/ -run TestCollectBoardsInto -v -count=1` | 5/5 PASS(RuijieBoards / ChassisRowDropped / HuaweiNoBoards / OutOfScopeVendor / CommandErrorTolerated) |
| 49-01 regression guard | `go test ./internal/services/ -run TestCollectDeviceInfo -v -count=1` | 5/5 PASS(chassis SN 解析未回归) |
| Collector regression | `go test ./internal/services/component_collector/ -count=1` | PASS(Ruijie/Huawei fixture + GetCollectorCommands 不回归) |
| Build | `go build ./...` | exit 0 |

### Acceptance Criteria

- [x] SOURCE: `collectComponentInfo` 函数体内 `ParseShowVersionModules` 被调用(line 794,通过 `collectBoardsInto` helper)
- [x] SOURCE: 错误注释 `"already collected by the chassis collector"` 已被替换为正确说明(line 680 现作为历史错误的描述)
- [x] BEHAVIOR: TestCollectBoardsInto_RuijieBoards — mock runner 返回 ruijie fixture,断言 set.Components 含 ≥6 条 ComponentType 为 engine/card 的元素(M1/1-5/M2)
- [x] BEHAVIOR: TestCollectBoardsInto_ChassisRowDropped — 断言 set.Components 不含 chassis 类型元素,set.Chassis == nil(BLOCKER-2 invariant)
- [x] BEHAVIOR: TestCollectBoardsInto_HuaweiNoBoards — huawei vendor 不调用 runner(ENTITY-MIB deferred D-08)
- [x] BEHAVIOR: TestCollectBoardsInto_OutOfScopeVendor — H3C vendor 直接 return nil(沿用既有短路)
- [x] BEHAVIOR: TestCollectBoardsInto_CommandErrorTolerated — SSH 命令失败返回 nil(D-14 容错)
- [x] TEST: `go test ./internal/services/ -run TestCollectBoardsInto -count=1` exit 0
- [x] TEST: `go test ./internal/services/component_collector/ -count=1` exit 0(既有测试不回归)
- [x] BUILD: `go build ./...` exit 0

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED  | 7764874f `test(49-02): add failing tests for board component collection` | 5 测试编译失败(`collectBoardsInto undefined`) |
| GREEN | b787256e `feat(49-02): wire board collection into collectComponentInfo` | 5/5 PASS |
| REFACTOR | — | 无需(实现已最小,collectBoardsInto 单一职责) |

## Deviations from Plan

None — plan 执行与设计一致。

### Plan 范围内但未触碰(scope 守护,符合 CLAUDE.md Scope Constrainment)

- **runTwoStepTransceiverPipeline 函数内部未改**:plan 明确指示保持原样(D-10 transceiver 专注路径)
- **Pipeline.Run / OpsAssetWriter 未改**:plan 明确指示不动(Gap 2 通过既有 GetByDeviceSN 自然打通)
- **cronAssetLookup.GetByDeviceSN 未改**:已正确用 devicesn 列匹配,Gap 2 关联自然成立
- **49-01 enrichChassisSerial / isChassisSNCommand / getCommandsByVendor 未触碰**:49-01 已合并,无回归
- **既有 services 包 TestRateLimiter_Cleanup 超时(180s sleep-based)**:预先存在的测试慢问题,与本次改动无关,不在 scope

## Task 2: E2E 端到端验证 — ✅ PASSED (2026-07-05)

Task 2 是 `type="checkpoint:human-verify" gate="blocking"` 任务,经运维同事现场执行通过。

### E2E 验证结果(生产 PG 实测)

**Step A — chassis SN 已写回 sys_network_device(49-01 前置):**

在线 ruijie/huawei 24 台中 **14 台 serial_number 已填充**(部署前 0/24):
- RS8607E-03 (目标设备) = `G1M9140000175` ✅(与 ops_asset.devicesn 一致,Gap 2 关联通畅)
- RS8607E-01/02, S5750, S5750C, S2910, S2952×2, RSR10, S8700×2 等全部命中
- 未填充的 10 台:huawei S5735 系列(返回 `ESN of slot 1` 而非 `ESN of chassis`,textfsm 模板未覆盖固定形态设备 — parser 增强 deferred,非本阶段回归)
- 未填充的 3 台 ruijie (RG50X×2, RG8607E-1):离线/不可达(updated_at 停留在 2026-05-08)

**Step C — RS8607E-03 板卡组件已写入 ops_asset(49-02 核心验证,9 条):**

| devicesn | component_type | component_slot | parent_asset_id |
|----------|----------------|----------------|------------------|
| G1MA11N00053A | card | Slot 1 | 0c3ac91f-... |
| G1N20TZ000052 | card | Slot 2 | 0c3ac91f-... |
| G1N20TZ00013B | card | Slot 3 | 0c3ac91f-... |
| G1N20TZ000010 | card | Slot 4 | 0c3ac91f-... |
| G1MRC5K000051 | card | Slot 5 | 0c3ac91f-... |
| G1M9140000175 | engine | Slot M1 | 0c3ac91f-... |
| G1MA1H9000847 | engine | Slot M2 | 0c3ac91f-... |
| G1PT54938427C | transceiver | TenGigabitEthernet 1/47 | 0c3ac91f-... |
| G1PT549429142 | transceiver | TenGigabitEthernet 1/48 | 0c3ac91f-... |

**7 条 engine/card + 2 条 transceiver = 9 条组件**,超过计划 ≥6 的验收门槛,且与 49-CONTEXT.md §验证目标 表格逐行匹配(M1 SN = chassis SN = G1M9140000175,Ruijie 主控复用 chassis SN 的已知行为)。

**Step B — component_serial anomaly 已写入 sys_data_reconciliation(emitter 修复验证):**

asset_id=a8bc5970-..., conflict_type=F, recon_category=component_serial, severity=medium — emitter 的 PG `IS ?` SQLSTATE 42601 bug 修复后,MISS 路径 anomaly 正常落表。

### E2E 期间发现并处理的 3 个阻塞项(非 49-02 代码 bug)

| # | 问题 | 根因 | 处理 |
|---|------|------|------|
| 1 | net_device_enrichment_task 31 条 `running` 僵尸任务阻塞 Enqueue dedup | 06-13 worker crash,`recoverPendingTasks` 仅恢复 pending 不恢复 running | DB 清理:`UPDATE ... SET status='failed' WHERE status='running' AND completed_at IS NULL`(运维执行) |
| 2 | ReconciliationEmitter dedup `recon_category IS ?` 在 PG 报 SQLSTATE 42601 | SQLite 把 IS 当 nullable 等价,PG 拒绝 `IS <string>`;Phase 48 遗留跨方言 bug | 修复 `reconciliation_emitter.go:61` → 单子句 `recon_category = ?`(commit d9a59468) |
| 3 | 连接池 `当前=23, 最大=20` TOCTOU 竞态 + 偶发卡死 | `connection_pool.go:266-273` count check 与 create 之间释放锁 | 重启清掉 stuck 连接(临时);根治是 pool 层的 follow-up,非本阶段 scope |

## Plan Status: ✅ COMPLETE — Task 1 + Task 2 全部通过

- Task 1(代码改动): **COMPLETE** — commits `7764874f`(RED) + `b787256e`(GREEN)
- Task 2(E2E checkpoint): **PASSED** — 生产 SQL 实测 RS8607E-03 展开 9 条组件(7 板卡 + 2 光模块),超过 ≥6 门槛

### Deferred items(非本阶段 scope,记录后续)

- **Huawei S5735 固定形态设备 chassis SN**:`display device esn` 返回 `ESN of slot 1`(非 `ESN of chassis`),textfsm 模板未覆盖。parser 增强:加 `ESN of slot 1` fallback 匹配 → 后续 phase 处理
- **连接池 TOCTOU 竞态根治**:在 `createConnection` 外加 `poolLock`,或调高 `MaxConnections` → 后续 follow-up
- **`net_device_enrichment_task` 僵尸恢复**:`recoverPendingTasks` 应同时清理过期 `running` 任务 → 后续 follow-up
- **前端 Tab 浏览器确认**:SQL 数据层已证实(9 条组件 + parent_asset_id 正确),前端 ComponentListTab 查询即此 SQL;UI 渲染留作下次现场访问最终确认

## Self-Check: PASSED

| Item | Status |
|------|--------|
| `internal/services/device_info_collection_service.go` | FOUND |
| `internal/services/device_info_collection_service_boards_test.go` | FOUND |
| `.planning/phases/49-v1-18-gap-closure/49-02-SUMMARY.md` | FOUND |
| commit `7764874f` (RED gate) | FOUND |
| commit `b787256e` (GREEN gate) | FOUND |
| AC grep: `ParseShowVersionModules` 在 collectComponentInfo scope 内引用 | 6 (≥1 PASS) |
| AC grep: 错误注释 `"already collected by the chassis collector"` 已替换 | line 680 改为历史错误说明 PASS |

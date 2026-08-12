---
phase: 49-v1-18-gap-closure
plan: 01
subsystem: device-info-collection
tags: [chassis-sn, gap-3, ruijie, huawei, component-collector]
requires:
  - "component_collector.ParseShowVersionModules (Phase 48 Wave 2)"
  - "component_collector.ParseDisplayDeviceEsn (Phase 48 Wave 2)"
provides:
  - "device_info_collection_service.enrichChassisSerial — chassis SN 回填路径"
  - "sys_network_device.serial_number 在 ruijie/huawei 在线设备上由空变为非空(下一 cron 周期生效)"
affects:
  - "Phase 49-02: 组件采集接入(parseDeviceInfo 完成后,collectComponentInfo 才能靠 device.SerialNumber 关联 chassis 资产)"
  - "Phase 48 组件清单 Tab: chassis SN 解锁后,parent_asset_id 匹配率从 0% 上升"
tech-stack:
  added: []
  patterns:
    - "Additive enrichment: 在不替换 legacy 字符串解析的前提下,叠加已验证的解析器作为 chassis SN 的真实来源"
    - "Only-if-empty 语义贯穿 enrichChassisSerial → updateDeviceInfo(不覆盖 SNMP 探测值)"
key-files:
  created:
    - internal/services/device_info_collection_service_test.go
  modified:
    - internal/services/device_info_collection_service.go
    - templates/huawei_vrp_display_device_esn.textfsm
decisions:
  - "Chassis SN 走 component_collector 已验证解析器(ParseShowVersionModules/ParseDisplayDeviceEsn),不复用 legacy 字符串解析"
  - "Huawei getCommandsByVendor 新增 'display device esn' 命令(V600R024C00 部分机型退役此命令 → 解析器返回空切片+nil err,非错误)"
  - "enrichChassisSerial 仅在 chassis SN 命令上触发,避免对 display version / display device 等无关输出重复跑模板解析"
  - "Huawei esn 模板规则 ': ' → '::' 容忍(从 \\s+ 改 \\s*)以匹配真机无空格输出"
metrics:
  duration: ~25 min
  completed: 2026-07-05
  tasks: 1
  files: 3
---

# Phase 49 Plan 01: chassis SN 采集接入 Summary

**One-liner:** 接入已验证的 component_collector chassis 解析器,在 CollectDeviceInfo 命令循环中回填 info.SerialNumber,修复 sys_network_device.serial_number 在 ruijie/huawei 在线设备上 100% 为空的 Gap 3 根因。

## What Was Built

### 问题根因(诊断确认)
`sys_network_device.serial_number` 在 ruijie(12/12)/huawei(12/12)在线设备上 100% 为空 → `ops_asset.devicesn = sys_network_device.serial_number` 关联失败 → 板卡组件 `parent_asset_id` 无 chassis 锚点 → 组件清单 Tab 空(Phase 48 用户报告)。

链路断点:`device_info_collection_service.collectDeviceInfo` 只调用 legacy `parseRuijieDeviceInfo`/`parseHuaweiDeviceInfo` 关键字字符串解析,无法匹配生产 `show version` / `display device esn` 输出格式。项目已有的、经过单测验证(对真机 fixture 稳定)的 `component_collector.ParseShowVersionModules` / `ParseDisplayDeviceEsn` 从未在采集链路被调用。

### 实现(叠加,非替换)
1. **`enrichChassisSerial(cmd, vendor, raw, info)`** 新方法:`isChassisSNCommand` 识别 chassis SN 命令(ruijie/maipu=`show version`,huawei/h3c=`display device esn`),调对应解析器,在返回的 `[]Component` 中找 `ComponentType==ComponentTypeChassis` 的元素,写入 `info.SerialNumber`。
2. **`CollectDeviceInfo` 命令循环**:每条命令执行后,先调 legacy `parseDeviceInfo`(负责 Model/SoftwareVersion/Uptime),再调 `enrichChassisSerial`。
3. **`getCommandsByVendor` huawei 分支**:从 `["display version", "display device"]` → `["display version", "display device", "display device esn"]`。
4. **only-if-empty 语义保留**:`enrichChassisSerial` 内部 + `updateDeviceInfo`(line 320-322 原有逻辑)双层 guard,不覆盖 SNMP 探测或更早命令已采的值。
5. **D-14 容错**:解析器返回 error → `applogger.Infof` 记录并返回(不阻塞);`huawei "Unrecognized command"` → 解析器返回空切片+nil err(非错误)。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Huawei esn 模板规则无法匹配真机无空格输出**
- **Found during:** Task 1 GREEN phase(测试 TestCollectDeviceInfo_HuaweiChassisESN 第一次失败)
- **Issue:** `templates/huawei_vrp_display_device_esn.textfsm` 规则 `^ESN\s+of\s+chassis\s+${ChassisID}\s*:\s+${ESN}\s*$$` 在冒号后用 `\s+`(至少 1 个空格),而真机 fixture `huawei_10_62_25_253_display_device_esn.txt` 内容为 `ESN of chassis 1:102599861597`(冒号后无空格) → 模板返回 0 条记录,SN 静默丢失。
- **Verification:** 直接 `regexp.MustCompile` 验证 — `(\d+)\s*:\s+(\S+)` 对 `1:102599861597` 不匹配,改成 `(\d+)\s*:\s*(\S+)` 后匹配。
- **Fix:** 模板规则冒号两侧均改为 `\s*`(零或多个空格),既匹配真机无空格形式,也兼容既有合成 fixture(冒号后带空格)。
- **Files modified:** `templates/huawei_vrp_display_device_esn.textfsm`(规则注释同步更新示例)
- **Regression guard:** 既有 `TestHuaweiCliParseDisplayDeviceEsn`(success-path 用 `: ` 带空格)仍通过。
- **Commit:** c102bdfe

### Plan 范围内但未触碰(scope 守护,符合 CLAUDE.md Scope Constrainment)

- **`parseRuijieDeviceInfo` / `parseHuaweiDeviceInfo` 字符串解析未改**:plan 明确指示保持原样作为 model/version/uptime 的辅助路径,model/version/uptime 脆弱性不在本阶段范围。
- **`updateDeviceInfo`(line 320-340 区域)未改**:其 only-if-empty 写回逻辑正符合本计划语义。
- **`collectComponentInfo` 未改**:属 plan 49-02(Wave 2,依赖本 plan 填好的 serial_number 做 chassis 资产关联)。

## Verification

| Check | Command | Result |
|-------|---------|--------|
| Target tests | `go test ./internal/services/ -run TestCollectDeviceInfo -v -count=1` | 5/5 PASS(RuijieChassisSN / HuaweiChassisESN / HuaweiEsn_UnrecognizedCommand / ChassisSN_DoesNotOverwriteExisting / LegacyParseStillRuns) |
| Collector regression | `go test ./internal/services/component_collector/ -count=1` | PASS(既有 Ruijie/Huawei fixture 测试 + GetCollectorCommands 不回归) |
| Build | `go build ./...` | exit 0 |

### Acceptance Criteria
- [x] SOURCE: `ParseShowVersionModules` 和 `ParseDisplayDeviceEsn` 在 `device_info_collection_service.go` 中被调用(line 361/363)
- [x] SOURCE: huawei 分支 `getCommandsByVendor` 返回值包含 `"display device esn"`(line 310)
- [x] BEHAVIOR: ruijie fixture → `info.SerialNumber == "G1HLC0R000096"`(与 fixture line 7 + 既有 `TestRuijieCliParseShowVersionModules` 一致)
- [x] BEHAVIOR: huawei fixture → `info.SerialNumber == "102599861597"`(从 fixture 实读)
- [x] BEHAVIOR: huawei "Unrecognized command" → `info.SerialNumber == ""` 且无 error
- [x] TEST: services 目标测试 exit 0
- [x] TEST: component_collector 测试 exit 0(回归保护)
- [x] BUILD: `go build ./...` exit 0

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED  | d32aeed6 `test(49-01): add failing tests for chassis SN enrichment` | 5 测试编译失败(`enrichChassisSerial undefined`) |
| GREEN | c102bdfe `feat(49-01): wire chassis SN parsers into device info collection` | 5/5 PASS |
| REFACTOR | — | 无需(实现已最小) |

## Deployment Verification(下一 cron 周期后可校验)

```sql
-- 部署后等下次 device_info_update cron(每小时)跑完,期望从 24 降到接近 0
SELECT count(*) FROM sys_network_device
WHERE vendor IN ('ruijie','huawei') AND status=0 AND serial_number='';
```

## Self-Check: PASSED

| Item | Status |
|------|--------|
| `internal/services/device_info_collection_service.go` | FOUND |
| `internal/services/device_info_collection_service_test.go` | FOUND |
| `templates/huawei_vrp_display_device_esn.textfsm` | FOUND |
| `.planning/phases/49-v1-18-gap-closure/49-01-SUMMARY.md` | FOUND |
| commit `d32aeed6` (RED gate) | FOUND |
| commit `c102bdfe` (GREEN gate) | FOUND |
| AC grep: `ParseShowVersionModules`/`ParseDisplayDeviceEsn` 引用数 | 3 (≥2 PASS) |

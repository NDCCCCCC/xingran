# Phase 49 CONTEXT — v1.18 组件采集 gap closure

> **来源**: 用户报告"CX-WH-WH-04F-FL-RS8607E-03 交换机展开后从属组件为空",真机 `show version` 显示 6 个 Slot 独立 SN 但前端 Tab 空。完整诊断于 2026-07-05。

## 症状

- 资产列表 → 选 chassis 资产 → 展开「从属组件清单」Tab → **空**
- DB 验证: `ops_asset WHERE component_type IS NOT NULL` = **0 条**(全表无任何组件记录)
- 影响: 所有 ruijie/huawei 设备(各 12 台在线),不仅 RS8607E-03

## 3 个连锁缺口(诊断已确认,plan-phase 无需重新调查)

### Gap 1: 板卡采集器是 dead code(核心 bug)

**位置**: `internal/services/device_info_collection_service.go:616-654` `collectComponentInfo`

**现状**:
```go
// line 630: 只调用了光模块 pipeline
set, err := runTwoStepTransceiverPipeline(ctx, device.Vendor, runner)
// line 634-637: 没 up 的 SFP 口时光模块 set 为空 → 直接 return nil,啥都不写
```

**未调用的现成代码**(已写完整实现 + 单测,但运行时无人调用):
- `component_collector.NewRuijieCliCollector().ParseShowVersionModules(raw)` — 解析 `show version` 的 Slot M1/1-5/M2 + chassis SN(正是用户贴的输出格式,fixture `ruijie_10_62_63_21_show_version.txt` 验证过)
- `component_collector.NewHuaweiCliCollector().ParseDisplayDeviceEsn(raw)` — 解析 `display device esn` 的 chassis ESN

**自欺欺人的注释**(line 605-609):
> "Ruijie business-board SNs come from show version, **already collected by the chassis collector**"

→ 那个 "chassis collector" **根本不存在**。错误假设导致采集器代码写了不接入。

### Gap 2: 资产关联断裂(parent_asset_id 匹配不上)

**现状**: 板卡组件 `parent_asset_id` 在 `pipeline.Run` 里靠 `device.SerialNumber` 匹配 chassis 资产(DeviceRef.SerialNumber)。

**问题**: `sys_network_device.serial_number` 全空(见 Gap 3),匹配不上任何 `ops_asset.devicesn`。

**关联覆盖度实测**(2026-07-05):
```
ops_asset (component_type IS NULL, 主设备) 6688 条
  ↕ JOIN ON sys_network_device.serial_number = ops_asset.devicesn
匹配成功: 0 条
匹配失败: 6688 条
```

### Gap 3: chassis SN 全空(前置根因)

**实测**(2026-07-05):
| vendor | status=0(在线) | serial_number 空的 |
|--------|---------------|-------------------|
| ruijie | 12 | **12 (100%)** |
| huawei | 12 | **12 (100%)** |

**链路问题**: chassis SN 采集(show version 的 "System serial number" 行 / display device esn)→ 写回 `sys_network_device.serial_number` 链路有更基础问题。需排查:
- 现有 chassis 采集器是否调用了 `show version` / `display device esn`?
- 解析了 SN 但没写回 `sys_network_device.serial_number`?
- 还是根本没采集 chassis SN(只采集端口/MAC)?

## 用户补充(2026-07-05):devicesn vs sequenceno 列

> "资产列表中有设备序列号和序列号两列,**以设备序列号为准**,序列号这列有很多空的"

**实测确认**:
| 列 | 字段 | 充实度 | 角色 |
|----|------|--------|------|
| 设备序列号 | `ops_asset.devicesn` | **6688/6688 (100%)** | ✅ 权威(用户确认) |
| 序列号 | `ops_asset.sequenceno` | **0/6688 (0%)** | ❌ 全空(废弃) |

**含义**:
- 关联策略必须以 `ops_asset.devicesn` 为准(不用 sequenceno)
- 板卡组件 `parent_asset_id` 解析: `devicesn = sys_network_device.serial_number`(Gap 3 修复后)

## 修复顺序(关键:有依赖)

```
Gap 3 (chassis SN 采集写回 sys_network_device)
   ↓ 解锁关联
Gap 2 (板卡组件 parent_asset_id 通过 devicesn=sn 关联 chassis 资产)
   ↓ 解锁展示
Gap 1 (collectComponentInfo 接入 ParseShowVersionModules / ParseDisplayDeviceEsn)
   ↓ 数据回填
验证目标达成
```

**不能先修 Gap 1**:即使接入板卡采集,组件 `parent_asset_id` 靠 `device.SerialNumber`(空)匹配 → 还是写不进 chassis 资产名下。

## 验证目标(端到端)

部署新二进制 → 等下次 device_info_update cron(每小时)跑完 → 在 web 验证:

资产 **CX-WH-WH-04F-FL-RS8607E-03**(devicesn=G1M9140000175)展开「从属组件清单」应显示 **6 条**(实际 7 条,含 chassis 自身;前端只展板卡 6 条):

| Slot | 类型 | SN | 来源命令 |
|------|------|-----|---------|
| M1 | engine(主控) | G1MA11N00053A? 或 G1M9140000175 | `show version` |
| 1 | card(业务板) | G1MA11N00053A | `show version` |
| 2 | card | G1N20TZ000052 | `show version` |
| 3 | card | G1N20TZ00013B | `show version` |
| 4 | card | G1N20TZ000010 | `show version` |
| 5 | card | G1MRC5K000051 | `show version` |
| M2 | engine(主控) | G1MA1H9000847 | `show version` |

(M1 SN 用户贴的输出里也是 G1M9140000175 — 与 chassis 同 SN,锐捷主控复用 chassis SN 的已知行为,parser 需保留去重逻辑)

## 已排除的方向

- ❌ 前端 Tab 实现完好(`ComponentListTab.tsx` + `index.tsx:536-544` expandable + `/ops/asset/components` 端点 + IS NULL 过滤) — 纯数据缺失问题
- ❌ Schema 已就绪(migration_201 已建 4 列 + 索引)
- ❌ 字典已 seed(`asset_reconciliation_recon_category` 含 `component_serial`)
- ❌ 不是 SNMP ENTITY-MIB 路径问题(那是有意 deferred 的 D-08,本环境用 CLI 路径足够)

## 参考

- Phase 48 ROADMAP: `.planning/milestones/v1.18-ROADMAP.md` Phase 48 段
- 诊断会话: 2026-07-05 本对话
- 相关代码:
  - `internal/services/device_info_collection_service.go:600-654` (collectComponentInfo)
  - `internal/services/component_collector/cli_ruijie_collector.go:48-84` (ParseShowVersionModules)
  - `internal/services/component_collector/cli_huawei_collector.go:55-79` (ParseDisplayDeviceEsn)
  - `internal/services/component_collector/pipeline.go` (DeviceRef.SerialNumber 匹配点)

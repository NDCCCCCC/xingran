# 网络设备组件序列号采集：设计决策与调研结论

**探索日期**: 2026-07-03（含外网调研成功回补）
**来源**: `/gsd-explore` session — 网络设备检测逻辑 + 一机多序列号
**关联阶段**: Phase 48（见 ROADMAP，v1.18）
**关联 seed**: `.planning/seeds/snmp-entity-mib-acceleration.md`
**关联 research**: `.planning/research/questions.md`（RQ-001 真机输出格式）

---

## 1. 背景与缺口

一个网络设备（尤其框式交换机）有**多个序列号**：机箱、多块板卡/引擎卡（每槽一个）、多个电源、多个风扇、以及在用接口上的光模块——各自独立序列号。

**方案转向（2026-07-03 用户决策）**：不再用独立子表，而是**每个有序列号的组件都作为一台资产设备进入资产系统 `ops_asset`**（"资产系统里就是这样"），并保存组件对交换机/路由的**从属关系**，前端展示。

**现状发现（代码级）**：

| 项 | 位置 | 结论 |
|----|------|------|
| 资产模型 | `internal/models/asset.go` (`ops_asset`) | **以 `DeviceSN` 为唯一键**（`uniqueIndex`）→ 组件序列号天然是一条资产记录的主键 |
| 资产父指针 | — | **不存在**，需新增自引用 `parent_asset_id` |
| 采集来源桥接 | — | **不存在**，需新增 `source_device_id` → `sys_network_device` |
| 主设备序列号 | `internal/models/network_device.go:53` `SerialNumber string` | 单字段，仅存机箱 SN |
| 板卡解析模板 | `templates/huawei_vrp_display_device.textfsm` | 已能解析板卡，但数据只在内存、从未落库 |
| 资产采集器 | `internal/collectors/asset_collector.go` | 已跑 `display version`/`display device`/`show version`，只取单序列号 |
| 采集服务 | `internal/services/device_info_collection_service.go` | 后台异步 worker 池（默认 5），`getCommandsByVendor` 按厂商分发 |
| 连接层 | `internal/device/scrapli_wrapper.go` | Scrapli SSH，平台映射：华为`huawei_vrp`/H3C`hp_comware`/锐捷`ruijie_rjos`/迈普`cisco_iosxe` |
| TextFSM 引擎 | `internal/templates/textfsm.go` | 自研解析器，`Clone()` 并发安全 + 模板缓存 |
| SNMP 通道 | `internal/device/snmp_client.go` | gosnmp 已封装，可作 ENTITY-MIB 加速入口 |

## 2. 锁定的设计决策

| 维度 | 决策 |
|------|------|
| **用途** | 完整硬件清单——设备档案 + 资产盘点/审计 + 备件管理 + 对账/合规上报（四用途全选） |
| **组件范围** | 板卡/引擎卡（必选）+ 电源 + 风扇 + 光模块 |
| **光模块粒度** | 只采在用/up 接口 |
| **建模方式** | **组件 = `ops_asset` 行**（`DeviceSN` = 组件序列号），非独立子表 |
| **从属关系** | 新增 `parent_asset_id`（自引用 → `ops_asset.id`）指向交换机资产行 |
| **采集桥接** | 新增 `source_device_id`（→ `sys_network_device.id`）标记采集来源；**双向桥接** |
| **组件元数据** | 需 `component_type`(chassis/card/engine/power/fan/transceiver) + `slot/position`（存哪列待定，可复用 `AttributeValue` 或新增列） |
| **采集路线** | v1 纯 CLI + TextFSM 单一路径（光模块 SNMP 拿不到，必走 CLI）；SNMP 作后续加速 |
| **对账价值** | 序列号是跨系统比对唯一键，天然对接 v1.17 资产对账 |

## 3. 外网调研结论（2026-07-03 成功，均来自厂商官方文档）

**四厂商组件序列号采集命令**：

| 厂商 | 板卡/电源/风扇 | 光模块 |
|------|---------------|--------|
| 华为 VRP | `display device`（各板卡/风扇/电源序列号）+ `display elabel`（电子标签：单板 BarCode、电源 SN） | `display transceiver interface X`（`Manu. Serial Number`）/ `display interface X transceiver verbose` |
| H3C Comware | `display device manuinfo` / `display device manuinfo power`（电源电子标签）/ **`display asset-info`**（机框/单板/风扇框/电源资产信息） | `display transceiver manuinfo interface X`（`Manu. Serial Number`） |
| 锐捷 RGOS | `show version slots` / `show version` | `show interface X transceiver`（类型+序列号）/ `... transceiver diagnosis`（DDM） |
| 迈普 MyPower | `show hardware` / `show version` | `show transceiver interface X` |

**两个关键发现（命中"组件作为资产 + 从属关系"方案）**：

1. **H3C `display asset-info`** 直接输出机框/单板/风扇框/电源模块的**资产信息**——厂商自身即把组件当资产列，与本方案一致。
2. **SNMP ENTITY-MIB `entPhysicalContainedIn`（OID `.1.3.6.1.2.1.47.1.1.1.1.4`）是"从属关系"的权威来源**——记录每个物理实体被谁包含（机框含板卡、板卡含光模块），华为/H3C 均支持。从属关系是标准 MIB 里本就有的父子容器树，非硬凑。

**来源**：
- 华为查看电源序列号：https://support.huawei.com/enterprise/zh/doc/EDOC1100365044/b9908d8c
- 华为查看光模块序列号：https://support.huawei.com/enterprise/zh/doc/EDOC1100212311/50f328b9
- 华为 display device 查各板卡/风扇/电源序列号：https://support.huawei.com/enterprise/zh/doc/EDOC1100368577/c6801428
- H3C display asset-info：https://www.h3c.com/cn/d_202109/1465444_30005_0.htm
- H3C display transceiver manuinfo：https://www.h3c.com/cn/d_202206/1630901_30005_0.htm
- RFC 6933 ENTITY-MIB (Version 4)：https://datatracker.ietf.org/doc/rfc6933/

## 4. 已知坑（写代码/模板前必读）

- **机框与单板各有独立序列号**（华为官方明确）；`display device serial-number` 只返机框，需 `display elabel`/`display device` 拿单板。
- **低端机 SN / `entPhysicalSerialNum` 返回空字符串** → 空 SN 不入库或标记，避免污染 `ops_asset.DeviceSN`（唯一键，空值会冲突）。
- **堆叠场景（华为 iStack / H3C IRF）序列号按成员漂移** → 用 slot 前缀或 `entPhysicalContainedIn` 重建归属。
- **光模块必走 CLI**，不依赖 SNMP。
- **TextFSM 模板脆弱**：厂商新版本输出格式变更会破解析，按 vendor×command 维护，参考 `internal/services/portcollection/parser.go` `getInterfaceCommand` 分发模式。
- **组件入 `ops_asset` 的唯一键冲突**：组件 SN 与主设备 SN 同表，需确保 `DeviceSN` 全局唯一，且区分资产层级（设备 vs 组件）。

## 5. 建议实现骨架

```go
// internal/models/asset.go 扩展 Asset（ops_asset）
type Asset struct {
    // ... 现有字段 ...
    ParentAssetID  *string `gorm:"size:64;index;column:parent_asset_id" json:"parentAssetId,omitempty"`   // 自引用 → ops_asset.id（从属关系）
    SourceDeviceID *string `gorm:"size:64;index;column:source_device_id" json:"sourceDeviceId,omitempty"` // → sys_network_device.id（采集来源桥接）
    ComponentType  *string `gorm:"size:32;column:component_type" json:"componentType,omitempty"`          // chassis/card/engine/power/fan/transceiver
    ComponentSlot  *string `gorm:"size:64;column:component_slot" json:"componentSlot,omitempty"`          // 槽位/接口位置
}
```

**采集→落库流程**：SSH 采集组件序列号 → 按 vendor 分发 TextFSM 解析 → 组装 `[]Asset`（设 ComponentType/Slot/SourceDeviceID/ParentAssetID）→ 按 `DeviceSN` upsert 到 `ops_asset`（参考端口采集 BatchUpsert + CacheInvalidator）→ 对账：与外部资产同步来的组件资产按 SN 匹配。

**待定问题**（见 RQ-001）：真机输出格式；component_type/slot 用新列还是复用 `AttributeValue`；父交换机若尚未是 `ops_asset` 行时如何创建/链接。

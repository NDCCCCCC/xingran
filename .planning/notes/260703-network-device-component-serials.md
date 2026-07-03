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
| **组件资产来源** | **外部资产系统 Excel 已导入**，采集**只匹配、不新建**（不引入 ops_asset 自动写入路径） |
| **采集核心价值** | 发现**从属关系**——外部 Excel 只有扁平资产行，"哪块板卡插在哪台交换机哪个槽"靠 SSH 采集才知道 |
| **从属关系** | 采集在匹配到的组件 `ops_asset` 行上写 `parent_asset_id`（自引用 → 交换机资产行） |
| **采集桥接** | 写 `source_device_id`（→ `sys_network_device.id`）标记采集来源；**双向桥接** |
| **SN 匹配不上** | **生成对账异常**（接入 v1.17 `sys_data_reconciliation`）：实物有账无=盘盈/未登记，账有实物无=盘亏/已拆 |
| **对账定位** | Phase 48 = 对账引擎新增"组件序列号"维度，**本质是 v1.17 R2 物理层对账的具体形态**；依赖对账底座（Phase 42-46）先就位 |
| **组件元数据** | `component_type`(chassis/card/engine/power/fan/transceiver) + `slot/position`，采集写到匹配行（新列 or 复用 `AttributeValue`，plan 时定） |
| **采集路线** | **v1 已定为双路径**（基于 2026-07-03 真机 ENTITY-MIB 探针突破）：<br>• **主路径**: SNMP ENTITY-MIB 单 GET 拉 `entPhysicalSerialNum` / `entPhysicalClass` / `entPhysicalContainedIn` 覆盖 chassis/module/fan（实测华为 S8700 上 100% 拿到 SN）<br>• **CLI 补充**: 电源 SN（华为实现漏洞，ENTITIY-MIB 也是空）+ 锐捷 `show version`（模块 SN 完美一段）+ 华为 `display interface transceiver`（光模块 SN + DDM 唯一来源） |
| **踩坑注记** | gosnmp `WalkFunc` 收 `SnmpPDU`，OID 在 `dataUnit.Name`；HW 拒绝 BulkWalk → v1 走单 GET；勿用 `public` 默认 community，从 `sys_auth_credential.snmp_communities` 拿真值 |
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

**现状约束（Explore 复核，2026-07-03）**：三张表完全独立、无同步——`sys_network_device`（唯一键 `ip_address`，SSH 采集）、`ops_asset`（唯一键 `devicesn`，**纯外部 Excel 导入**，采集器至今从不写它）、`sys_device_asset`（采集快照）。v1.17 对账以 `ops_asset` 为单一源，网络侧数据尚未进对账（R2 预留）。

```go
// internal/models/asset.go 扩展 Asset（ops_asset）— 仅加关系/元数据列，不改 DeviceSN 语义
type Asset struct {
    // ... 现有字段 ...
    ParentAssetID  *string `gorm:"size:64;index;column:parent_asset_id" json:"parentAssetId,omitempty"`   // 自引用 → ops_asset.id（从属关系）
    SourceDeviceID *string `gorm:"size:64;index;column:source_device_id" json:"sourceDeviceId,omitempty"` // → sys_network_device.id（采集来源桥接）
    ComponentType  *string `gorm:"size:32;column:component_type" json:"componentType,omitempty"`          // chassis/card/engine/power/fan/transceiver
    ComponentSlot  *string `gorm:"size:64;column:component_slot" json:"componentSlot,omitempty"`          // 槽位/接口位置
}
```

**采集→匹配→对账流程（不新建资产行）**：
1. SSH 采集组件序列号 + 所属交换机（按 vendor 分发 TextFSM 解析）。
2. 采集侧父交换机以 `sys_network_device` 锚定；组件/交换机在 `ops_asset` 里按 `DeviceSN` **查找已有行**。
3. 匹配到组件行 → UPDATE `parent_asset_id`（指向交换机资产行）+ `source_device_id` + `component_type/slot`。
4. **匹配不上 → 生成 `sys_data_reconciliation` 对账异常**（实物有账无 / 账有实物无）。
5. 前端资产/设备详情页按 `parent_asset_id` 聚合展示"从属组件清单"。

**关键约束**：采集对 `ops_asset` **只 UPDATE 不 INSERT**，保住"实物层 vs 声明层"分层不被污染。父交换机若不在 `ops_asset`，`parent_asset_id` 留空、仍写 `source_device_id`（见 RQ-002）。

**待定问题**（见 RQ-002）：真机输出格式；父交换机不在 ops_asset 时的降级；component_type/slot 用新列还是 `AttributeValue`；组件对账异常用哪个 category（新增 vs 复用 v1.17 D 异常）。

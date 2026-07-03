# 网络设备组件序列号采集：设计决策与调研结论

**探索日期**: 2026-07-03
**来源**: `/gsd-explore` session — 网络设备检测逻辑 + 一机多序列号
**关联阶段**: 拟新增 Phase 48（见 ROADMAP）
**关联 seed**: `.planning/seeds/snmp-entity-mib-acceleration.md`
**关联 research**: `.planning/research/questions.md`（四厂商真机输出格式）

---

## 1. 背景与缺口

一个网络设备（尤其框式交换机）有**多个序列号**：机箱、多块板卡（每槽一个）、多个电源、多个风扇、以及在用接口上的光模块——各自独立序列号。现有数据模型装不下。

**现状发现（代码级）**：

| 项 | 位置 | 结论 |
|----|------|------|
| 主设备序列号 | `internal/models/network_device.go:53` `SerialNumber string` | **单字段**，无法一对多 |
| 资产快照序列号 | `internal/models/device_asset.go` `SerialNumber` | 同样单字段 |
| 组件表 | — | **不存在**（无 DeviceComponent/Card/PowerSupply/Module/Fan） |
| 板卡解析模板 | `templates/huawei_vrp_display_device.textfsm` | **已能解析板卡，但数据只在内存、从未落库** |
| 资产采集器 | `internal/collectors/asset_collector.go` | 已在跑 `display version`/`display device`/`show version`，只取单序列号（字符串解析，非 TextFSM） |
| 采集服务 | `internal/services/device_info_collection_service.go` | 后台异步 worker 池（默认 5），`getCommandsByVendor` 按厂商分发命令 |
| 连接层 | `internal/device/scrapli_wrapper.go` | Scrapli SSH，平台映射：华为`huawei_vrp`/H3C`hp_comware`/锐捷`ruijie_rjos`/迈普`cisco_iosxe` |
| TextFSM 引擎 | `internal/templates/textfsm.go` | 自研解析器，`Clone()` 并发安全 + 模板缓存 |
| SNMP 通道 | `internal/device/snmp_client.go` | gosnmp 已封装，可作后续 ENTITY-MIB 加速入口 |

**结论**：这不是"从零采集"，而是**扩展数据模型 + 持久化已经能拿到的组件数据**。

## 2. 锁定的设计决策

| 维度 | 决策 |
|------|------|
| **用途** | 完整硬件清单——设备档案 + 资产盘点/审计 + 备件管理 + 对账/合规上报（四用途全选） |
| **组件范围** | 板卡（必选）+ 电源 + 风扇 + 光模块 |
| **光模块粒度** | **只采在用/up 接口**（大多数空口无模块，平衡覆盖与耗时） |
| **数据模型** | 新建 `DeviceComponent` 一对多组件表：`device_id` + `component_type`(chassis/card/power/fan/transceiver) + `slot/position` + `serial_number` + `model` + `status` |
| **采集路线** | **v1 纯 CLI + TextFSM 单一路径**，复用现有 Scrapli 链路；SNMP 作后续加速（见 seed） |
| **对账价值** | 序列号是跨系统比对唯一键，天然对接 v1.17 资产对账里程碑 |

## 3. 调研结论（外网受限，基于厂商通用实现，生产需真机验证）

1. **光模块序列号 SNMP 基本拿不到**——ENTITY-MIB 里光模块常不在实体树，**必须走 CLI**（`display transceiver` / `show transceiver` 的 DDM vendor serial）。这是排除"纯 SNMP 方案"的决定性理由。
2. **板卡/电源/风扇**：华为/H3C 的 ENTITY-MIB（`entPhysicalSerialNum` OID `.1.3.6.1.2.1.47.1.1.1.1.11`）较完整；锐捷/迈普参差。但 CLI `display device manuinfo` 系命令能一次拿全，覆盖率最高。
3. **四厂商 CLI 命令速查**（待真机验证）：

| 厂商 | 板卡/电源/风扇 | 光模块 |
|------|---------------|--------|
| 华为 VRP | `display device manuinfo`（含 slot/PSU/FAN 段） | `display transceiver interface X` / `display transceiver diagnosis interface X`（DDM） |
| H3C Comware | `display device manuinfo` / `display device serial-number` | `display transceiver interface X` / `display transceiver manuinfo` |
| 锐捷 RGOS | `show version`（含 slot/power/fan 段，部分型号 `show slot`） | `show transceiver detail interface X` |
| 迈普 MyPower | `show hardware` / `show version` | `show transceiver interface X` / `show optic-module` |

## 4. 已知坑（写代码/模板前必读）

- **低端机 `entPhysicalSerialNum` / CLI SN 返回空字符串** → GORM 写入需容错，空 SN 不入库或标记。
- **堆叠场景（华为 iStack / H3C IRF）序列号按成员漂移** → 成员设备单独显示，需用 `entPhysicalContainedIn` 或 slot 前缀重建归属。
- **SN 长度为 0** 时解析/写库容错。
- **光模块必走 CLI**，不依赖 SNMP。
- **TextFSM 模板脆弱**：厂商新版本输出格式变更会破解析，需按 vendor×command 维度维护，参考 `internal/services/portcollection/parser.go` 的 `getInterfaceCommand` 按厂商分发模式。

## 5. 建议实现骨架

```go
// internal/models/device_component.go（新建）
type DeviceComponent struct {
    ID            string `gorm:"type:uuid;primary_key"`
    DeviceID      string `gorm:"type:uuid;not null;index"`
    ComponentType string // chassis/card/power/fan/transceiver
    Slot          string // 槽位/位置，如 "Slot 1" / "GE1/0/24"
    SerialNumber  string `gorm:"index"` // 组件序列号（对账比对键）
    Model         string // 组件型号
    HardwareVer   string
    Status        string // Online/Offline
    // created_at / updated_at / deleted_at
}
```

采集侧建议在 `internal/services/portcollection/` 或采集服务下按 vendor 分发 resolver，统一返回 `[]DeviceComponent`，批量 upsert（参考端口采集的 `BatchUpsert` + `CacheInvalidator` 模式）。

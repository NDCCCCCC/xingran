# Research Questions

开放研究问题追踪。每条含背景、待验证项、阻塞对象。

---

## RQ-001: 四厂商组件命令真机输出格式验证

**提出日期**: 2026-07-03
**来源**: gsd:explore — 网络设备组件序列号采集
**优先级**: high（阻塞 Phase 48 TextFSM 模板编写）
**关联**: `.planning/notes/260703-network-device-component-serials.md`

### 进展状态: 部分已验证（2026-07-03 真机采样）

**已采样**：华为 S8700 (V600R024C00, 10.62.25.253) + 锐捷 RS8607E (10.62.63.21) 各 1 台框式，36 个原始回显存 `templates/samples/`（一键采样工具 `cmd/collect_component_samples/` 已保留作参考痕迹，可删除）。

### 真机验证结果

#### 锐捷 RS8607E RGOS — ✅ `show version` 一条全搞定

整机 SN + 全部模块 SN 同段输出，TextFSM **极简模板**：

```
System serial number    : G1HLC0R000096
Module information:
  Slot M1 : M8600E-CM
    Serial number       : G1HLC0R000096
  Slot 1 : M8600E-24GT20SFP4XS-ED
    Serial number       : G1P7286000129
  Slot 2 : M8600E-48GT-P-ED
    Serial number       : G1N20TZ00011A
  ...
```

实战字段：`ModuleType(M8600E-CM / -48GT-P-ED)`、`Slot(M1/1/2/...)`、`SerialNumber(G1...)`、`Role(Master/Backup, 仅 M1/M2)`。
**模板开写前置条件已具备。**

`show version slots` 也能补一套槽位 → 模块型号 + 在线状态，用于校验。但 SN 来源仍是 `show version`。

#### 华为 CloudEngine S8700 V600R024C00 — ⚠️ elabel 被官方弃用

| 命令 | 结果 |
|------|------|
| `display version` | OK：整机版本 + 整机 SN 在某段，但格式需复查样本 |
| `display device esn` | ✅ 整机 SN 单独一行：`ESN of chassis 1:102599861597` |
| `display device` | ✅ 完整硬件清单（10 板卡/4 电源/2 风扇，含 Slot/Power/Alarm） |
| `display device board` | 同上 |
| `display device fan` | ✅ 风扇实时状态（转速%/模式/风向，**无 SN**） |
| `display device power` | ✅ 电源实时状态（电压/电流/功率，**无 SN**） |
| `display device elabel` | ❌ V600R024C00 该命令被弃用（仅返 "command is being executed..." 假成功） |
| `display elabel slot N` | ❌ 整命令不存在（"Unrecognized command"） |
| `display device slot N` | ⚠️ 命中但 BarCode 字段**全空** |
| `display device manufacture-info` / `... slot N` | ❌ "Too many parameters"（命令语法对带 slot 不认） |
| **`display interface transceiver`** | ✅ **光模块 SN 完整拿到**：`Manu. Serial Number: 8000012000082`，含厂家/PN/DDM |

#### 华为光模块 SN 命令总结

**唯一可靠**：`display interface transceiver`（每个接口一段，含厂商 + PN + Manu. Serial Number）。
V600R024C00 的 `transceiver` 子命令在不同 VRP 版本间可能不一样；本机 `display transceiver ?` 返回 "Unrecognized"，说明这版直接走 `display interface transceiver`。建议 Phase 48 模板覆盖该命令。

### 剩余缺口

1. **华为板卡/电源/风扇的 SN**：V600R024C00 elabel 路径完全失效。三个备选：
   - **走 SNMP ENTITY-MIB** 拉板卡/电源/风扇 SN（`entPhysicalSerialNum`），华为/H3C CE 系基本支持（此时 seed `snmp-entity-mib-acceleration` 应**提前到 v1 实现**而非留 v2）；
   - **接受现状**：仅采集光模块 SN + 整机 SN，板卡/电源/风扇作为采集盲区留人工盘点；
   - **走 ELabel 历史命令**的替代：试 `display device-info`/`display device asset-info`/`display board-info`，本机尚未测。
2. **锐捷光模块样本缺**：这台 RS8607E 所有光口都没插模块（`transceiver is absent`），无法拿真实 DDM 输出。建议换一台有 SFP 的设备补一次。
3. **SNMP ENTITY-MIB 真机覆盖度**：两家设备是否真返回非空 `entPhysicalSerialNum`？建议接入阶段先抓 `snmpwalk` 输出验证。
4. **H3C / 迈普完全无采样**：此环境无 H3C/迈普设备，模板必须留待其他环境落地。

### 建议方式

跑过的工具（已删）：`cmd/collect_component_samples/` 用现有 `DeviceConnectionPool` + `SendCommand` 复用生产代码路径，对接本环境在线设备批量跑命令。建议**重写为常驻 ad-hoc 工具**，方便以后再补样本。

模板落地优先顺序：
1. ✅ **锐捷 `show version`** — 立刻可写 (data 已有)
2. ✅ **华为 `display interface transceiver`** — 立刻可写 (data 已有)
3. ⏳ 华为 `display device esn` / `display device` — 立刻可写 (用于整机 SN 与硬件清单骨架)
4. ⚠️ 华为板卡/电源/风扇 SN — 待技术路线拍板 (SNMP / 接受盲区 / 其他命令)
5. ⚠️ 锐捷光模块 DDM TextFSM — 待另一台设备采样
6. ⚠️ H3C/迈普模板 — 待另一环境

---

### 待验证项

1. ~~每条命令的实际回显样例~~ ✅ 部分完成 (36 文件在 templates/samples/)
2. ~~命令是否需要特权模式 / 分页关闭~~ — 未测；v1 建议先尝试默认，失败再加 `screen-length 0 temporary`
3. SNMP ENTITY-MIB 真机覆盖度：四厂商 `entPhysicalSerialNum` 对板卡/电源/风扇是否返非空？（决定 seed 路线）— **未测**
4. 堆叠设备（iStack/IRF）输出里 slot 编号如何区分成员归属。 — 未测，本环境无堆叠
5. 空 SN / 异常回显样本（低端机场景）— 未测


---

## RQ-002: 组件入资产系统的建模细节

**提出日期**: 2026-07-03
**来源**: gsd:explore — 方案转向"组件作为资产 + 双向桥接"
**优先级**: medium（阻塞 Phase 48 R1 migration 设计）
**关联**: `.planning/notes/260703-network-device-component-serials.md`

### 背景

已定：组件作为 `ops_asset` 行（`DeviceSN`=组件序列号），`parent_asset_id` 自引用指向交换机资产行，`source_device_id` 桥接 `sys_network_device`。剩余建模细节需确认。

### 待确认项

已定（2026-07-03 discuss）：
- **组件资产来源** = 外部资产系统 Excel 已导入，采集**只匹配不新建**（不引入 ops_asset 自动写入路径）。
- **SN 匹配不上** = 生成 `sys_data_reconciliation` 对账异常（实物有账无=盘盈、账有实物无=盘亏）。Phase 48 依赖并扩展 v1.17 对账引擎，是 R2 物理层对账的组件形态。
- **父交换机不在 `ops_asset`** = `parent_asset_id` 留空、仍写 `source_device_id` 降级；**不**为此额外报异常（交换机整机缺失是另一层问题）。
- **`component_type` / `component_slot`** = 新增专用列（非复用 `AttributeValue`），因需按类型筛选。
- **组件对账异常 category** = 新增"组件序列号"专属类别（便于对账页单独筛出）；**前提**：写码前确认 `reconciliation_detection.go` 允许扩展 category，否则回退复用现有类型。
- **资产层级区分** = 设备列表/统计（`asset_service.go` `List()`/`Statistics()`）默认按 `component_type IS NULL` 排除组件行（避免 1 台交换机+6 板卡=7 台设备）；组件另开视图。

**均已定，RQ-002 关闭。剩余仅 RQ-001 真机输出格式待验证。**


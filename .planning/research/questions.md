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

### 实测数据：SNMP ENTITY-MIB 关键突破 (2026-07-03)

参考真机探针 `cmd/snmp_entity_probe/`（已删）的产物，结论：

- 华为 S8700 V600R024C00 **启用 ENTITY-MIB 且覆盖完整**：`entPhysicalSerialNum` 板卡/引擎/风扇**实测有非空值**；`entPhysicalContainedIn` 给完整父子容器树。
- **唯一缺口：电源 SN**——S8700 在 entPhysicalSerialNum 对电源也是空（HW 实现漏洞），需走 CLI `display power` 兜底该子集。

**华为 S8700 实测 ENTITY-MIB 摘录** (community=`Jt0916ct@M1`, 设备 IP `10.62.25.253`):

| Class | Name | Model | SerialNum |
|-------|------|-------|-----------|
| chassis(3) | CloudEngine S8700-6 | — | **102599861597** |
| fan(9)* | LSG7G48VX1E0 1 | LSG7G48VX1E0 | **102599806093** |
| module(5) | LPU slot 1 | — | _empty_ |
| fan(9)* | LSG7G48VX1E0 2 | LSG7G48VX1E0 | **102599806089** |
| fan(9)* | LSG7G48VX1E0 3 | LSG7G48VX1E0 | **102599806092** |
| fan(9)* | LSG7G48VX1E0 4 | LSG7G48VX1E0 | **102599806088** |
| fan(9)* | LSG7SRUEX1C0 5 | LSG7SRUEX1C0 | **102599806030** (主控) |
| fan(9)* | LSG7SRUEX1C0 6 | LSG7SRUEX1C0 | **102599806031** (备控) |
| module(5) | POWER slot 1-6 | — | _empty_ (HW 漏洞) |
| stack(7) | FAN 1 | — | **5B2599001652** |
| stack(7) | FAN 2 | — | **5B2599001751** |

注：华为 LPU/MPU 业务板与引擎在 ENTITY-MIB 中**实体化了两次**（module 占位 + class=fan 实体带 SN），取后者即可；电源 module 占位但 SN 空。

**结论**：**SNMP 必须成为 Phase 48 v1 的主采集路径**（CLI 退为 CLI 端 CLI 补充；光模块 CLI 仍为主）。原 seed `snmp-entity-mib-acceleration` 已 superseded 并入 Phase 48 R2。

**踩坑记录**：
- gosnmp v1.35.0 `WalkFunc` 只收 `SnmpPDU`，OID 在 `dataUnit.Name`
- 华为设备对 GETBULK/walk 大概率限制/拒绝（基础 OID 单 GET 立即返回，wALK 类全 timeout），**v1 实现需走单 GET 而非 BulkWalk**
- 测试 community 时**避免公开默认值 `public`** ——真实 community 从 `sys_auth_credential.snmp_communities` 取，格式如 `Jt0916ct@M1`

### 剩余缺口

1. ~~华为板卡/电源/风扇 SN~~ ✅ **完成**：SNMP 拿到板卡/引擎/风扇；电源仍缺，需 CLI 兜底
2. ~~锐捷 ENTITY-MIB 测~~ ✅ **完成 (2026-07-03)**：覆盖甚至超过华为；详见下表
3. **锐捷光模块样本缺**：这台 RS8607E 所有光口都没插模块（`transceiver is absent`），无法拿真实 DDM 输出。建议换一台有 SFP 的设备补一次。
4. **H3C / 迈普完全无采样**：此环境无 H3C/迈普设备，模板必须留待其他环境落地。

### 锐捷 ENTITY-MIB 实测数据 (2026-07-03, RS8607E 10.62.63.21, community=`cpic321`)

锐捷实现比华为**更完整**——电源 SN 也在 SNMP 返回（华为电源 SN 是空的）。**M1/M2 SN（`G1HLC0R000096`/`G1HLB1R000196`）与 CLI `show version` 完全一致**，双向交叉验证。

| Class | Name | Model | SN | ContainedIn |
|-------|------|-------|----|-----|
| chassis(3) | RG-S8607E | RG-S8607E | **G1J40D100022A** | 0 |
| backplane(4) | BACKPLANE_CA05 | — | — | 1 (passive) |
| module(5) | slot 1-5, M1, M2 | — | — | 2 (实槽占位) |
| **stack(7)** | fan1 | M07-FAN | **3530J40CZ0012** | 1 |
| **port(6)** | power0 | RG-PA600I | **A82603150300065** | 1 |
| port(6) | power1 | RG-PA600I | **A82603150300758** | 1 |
| port(6) | power2 | RG-PA600I | **A82603150300534** | 1 |
| fan(9) | mboard M1 | M8600E-CM | **G1HLC0R000096** | 8 (Master) |
| fan(9) | mboard M2 | M8600E-CM | **G1HLB1R000196** | 9 (Slave) |

锐捷全表 627 项，**非空 SN: 12**（其中 4 电源 + 1 风扇 + 1 chassis + 2 引擎 M1/M2 + 2x SensorBoard + 2x CN6130 都是噪声），核心 12 个全对得上。

**锐捷唯一 SNP 缺口**：slot 1-5 的 module 占位（"slot 1"-仅占位，无 SN）。**这部分正好是 CLI `show version` 拿到的 5 个单板 SN**（G1P7286000129 / G1N20TZ00011A / G1MV41U000047 / G1NRBA000001B / G1MV41U00001C）。所以 **CLI `show version` 是锐捷单板 SN 的唯一来源**。

**过滤建议**：锐捷 ENTITY-MIB 实质 import 时需 **从 SN 维度去重**+丢掉 `Class=8/powerSupply` 且 Name 以 `temprature`/`温度` 开头的传感器节点（typo `temprature` 出现 352 次，是锐捷私有扩展）。

### 建议方式

跑过的工具（已删）：`cmd/snmp_entity_probe/` 用 gosnmp v1.35.0 单 GET 模式探针。建议**重写为常驻 ad-hoc 工具**，便于以后再扩展采样。

模板落地优先顺序：
1. ✅ **锐捷 `show version`** — 立刻可写 (data 已有) — 补 SNMP 缺的 5 个业务板 SN
2. ✅ **华为 `display interface transceiver`** — 立刻可写 (data 已有)
3. ✅ **华为 ENTITY-MIB** — 立刻可写 (data 已有) — chassis/module/fan
4. ✅ **锐捷 ENTITY-MIB** — 立刻可写 (data 已有) — chassis/fan/powerSupply
5. ⏳ 华为 `display device esn` 互验 + `display power` 兜底缺漏的电源 SN
6. ⚠️ 锐捷光模块 DDM TextFSM — 待另一台设备采样
7. ⚠️ H3C/迈普模板 — 待另一环境

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


# Research Questions

开放研究问题追踪。每条含背景、待验证项、阻塞对象。

---

## RQ-001: 四厂商组件命令真机输出格式验证

**提出日期**: 2026-07-03
**来源**: gsd:explore — 网络设备组件序列号采集
**优先级**: high（阻塞 Phase 48 TextFSM 模板编写）
**关联**: `.planning/notes/260703-network-device-component-serials.md`

### 背景

外网调研已确认四厂商**命令名**（见下），但 TextFSM 模板必须按**真实回显格式**编写，格式猜错则解析全废。命令已定，缺的是每条命令的实际输出样例。

**已确认命令**（厂商官方文档，2026-07-03）：
- 华为：`display device`（板卡/风扇/电源序列号）、`display elabel`（单板 BarCode / 电源 SN）、`display transceiver interface X`（光模块 `Manu. Serial Number`）
- H3C：`display device manuinfo` / `display device manuinfo power`、`display asset-info`（机框/单板/风扇/电源资产信息）、`display transceiver manuinfo interface X`
- 锐捷：`show version slots` / `show version`、`show interface X transceiver`（+ `transceiver diagnosis`）
- 迈普：`show hardware` / `show version`、`show transceiver interface X`

### 待验证项

1. **每条命令的实际回显样例**（抓真机，用于写/对齐 TextFSM 列定义）。
2. 命令是否需特权模式 / 分页关闭（`screen-length 0 temporary` 等）。
3. **SNMP ENTITY-MIB 真机覆盖度**：四厂商 `entPhysicalSerialNum` 对板卡/电源/风扇是否返非空？`entPhysicalContainedIn` 是否可靠重建从属树？（决定 seed `snmp-entity-mib-acceleration` 可行性）
4. 堆叠设备（iStack/IRF）输出里 slot 编号如何区分成员归属。
5. 空 SN / 异常回显样本收集（低端机场景），用于容错设计。

### 建议方式

用现有 `internal/device/executor.go` 的 `ExecuteOnDevice` 对现网各厂商代表型号跑命令，保存回显样本到 `templates/samples/`，再据此编写模板。半成品 `templates/huawei_vrp_display_device.textfsm` 可作华为板卡起点。

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

仍待确认：
1. **父交换机若不在 `ops_asset`**（外部只导了组件、没导交换机本体）→ `parent_asset_id` 留空、只写 `source_device_id` 降级，还是也报一条对账异常？
2. **`component_type` / `component_slot` 存哪**：新增专用列，还是复用现有 `AttributeValue`(size:500) JSON？
3. **组件对账异常用哪个 category**：v1.17 现有异常类型（如 D 异常）能否复用，还是需新增"组件序列号"异常类别？需读 `models/reconciliation.go` + `reconciliation_detection.go` 定。
4. **资产层级区分**：组件行与设备行在 `ops_asset` 同表，前端设备列表/统计（`asset_service.go` `List()`/`Statistics()`）是否需按 `component_type IS NOT NULL` 过滤组件行，避免把板卡当独立设备统计。


# Research Questions

开放研究问题追踪。每条含背景、待验证项、阻塞对象。

---

## RQ-001: 四厂商组件命令真机输出格式验证

**提出日期**: 2026-07-03
**来源**: gsd:explore — 网络设备组件序列号采集
**优先级**: high（阻塞 Phase 48 TextFSM 模板编写）
**关联**: `.planning/notes/260703-network-device-component-serials.md`

### 背景

本次探索的技术调研外网受限，四厂商组件采集命令和 SNMP 覆盖度结论基于厂商通用实现，**未经真机验证**。TextFSM 模板必须按真实输出格式编写，格式猜错则解析全废。

### 待验证项

1. **CLI 实际输出格式**（每家抓真机回显，用于写/对齐 TextFSM 模板）：
   - 华为 VRP：`display device manuinfo`、`display transceiver interface X`、`display transceiver diagnosis interface X`
   - H3C Comware：`display device manuinfo` / `display device serial-number`、`display transceiver interface X`
   - 锐捷 RGOS：`show version` / `show slot`、`show transceiver detail interface X`
   - 迈普 MyPower：`show hardware` / `show version`、`show transceiver interface X` / `show optic-module`
2. **命令是否需要特权模式 / 分页关闭**（`screen-length 0 temporary` 等）。
3. **SNMP ENTITY-MIB 真机覆盖度**：四厂商 `entPhysicalSerialNum` 对板卡/电源/风扇是否真返回非空？（决定 seed `snmp-entity-mib-acceleration` 可行性）
4. **堆叠设备（iStack/IRF）** 输出里 slot 编号如何区分成员归属。
5. **空 SN / 异常回显** 样本收集（低端机场景），用于容错设计。

### 建议方式

用现有 `internal/device/executor.go` 的 `ExecuteOnDevice` 对现网各厂商代表型号跑一遍命令，保存回显样本到 `templates/samples/`，再据此编写模板。半成品 `templates/huawei_vrp_display_device.textfsm` 可作华为板卡起点。

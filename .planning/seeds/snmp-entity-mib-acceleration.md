---
title: SNMP ENTITY-MIB 组件采集加速
trigger_condition: CLI 逐设备采集成性能瓶颈，或需跨厂商结构化差异比对时
planted_date: 2026-07-03
origin: gsd:explore session — 网络设备组件序列号采集
related_note: .planning/notes/260703-network-device-component-serials.md
priority: low
status: pending
---

# SNMP ENTITY-MIB 组件采集加速

## 背景

Phase 48（网络设备组件序列号采集）v1 采用**纯 CLI + TextFSM** 单一路径。原因：光模块序列号 SNMP 拿不到，无论如何都得走 CLI，纯 CLI 省掉引入 SNMP 新代码路径的复杂度。

但 CLI 逐设备采集慢、模板脆弱。板卡/电源/风扇这三类**在 SNMP ENTITY-MIB 里是标准化、结构化、可快速批量拉取**的，适合作为加速通道。

## 触发条件（满足任一考虑启动）

1. CLI 逐设备采集耗时成为瓶颈（设备数增长 / 采集窗口不够）
2. 需要跨厂商**结构化差异比对**（历史快照 diff、组件增减告警）
3. 华为/H3C 设备占比高（这两家 ENTITY-MIB 实现较完整）

## 范围摘要

- 引入 SNMP 拉 `entPhysicalTable`（板卡/电源/风扇），复用 `internal/device/snmp_client.go`
- 混合分层兜底：先 SNMP 拉板卡/电源/风扇 → `entPhysicalSerialNum` 为空的项 fallback 到 CLI `display device manuinfo`
- **光模块仍强制走 CLI**，不纳入本 seed
- 目标 OID：`entPhysicalClass`(.5)/`entPhysicalName`(.7)/`entPhysicalSerialNum`(.11)/`entPhysicalModelName`(.13)/`entPhysicalContainedIn`（重建堆叠归属）

## 已知坑

- 低端机 `entPhysicalSerialNum` 返空 → 必须 fallback CLI
- 锐捷/迈普 ENTITY-MIB 实现参差 → 覆盖度需真机验证
- 堆叠 iStack/IRF 序列号漂移 → 用 `entPhysicalContainedIn` 重建拓扑

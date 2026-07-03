---
title: 电源/风扇 SNMP 多 community fallback + 可选 v2/v3
trigger_condition: ENTITY-MIB 拿不全某些组件 SN，或需要适配仅 v3 的设备
planted_date: 2026-07-03
origin: gsd:explore — 网络设备组件序列号采集，2026-07-03 SNMP 真机探针结果
related_note: .planning/notes/260703-network-device-component-serials.md
related_research: .planning/research/questions.md#rq-001-四厂商组件命令真机输出格式验证
status: SUPERSEDED — 已并入 Phase 48 v1 实现
priority: high
---

# 电源/风扇 SNMP 路径补充 + v3/多 community 处理

## 状态变更 (2026-07-03)

原 seed 设想把 SNMP 留作 v2 加速层。**真机探针结果证明必须提前到 v1 实施**：

- **板卡/引擎/风扇** 实测可经 ENTITY-MIB 直接拿到 SN（含 `entPhysicalSerialNum`）
- **电源 SN** —— ENTITY-MIB 也是空，是 HW 实现漏洞
- 因此 SNMP 必须成为 Phase 48 v1 的**主采集路径**（CLI 退居补充）

如果还有更多想做的（不在 Phase 48 范围内）：
- 适配 SNMPv3（auth/priv）凭据
- 多 community fallback
- 低速设备更慢的 retry

### 真机数据样本（华为 S8700 V600R024C00, 10.62.25.253, community=Jt0916ct@M1, 2026-07-03）

| Class | Name | ModelName | SerialNum |
|-------|------|-----------|-----------|
| chassis(3) | CloudEngine S8700-6 | — | **102599861597** |
| fan(9)* | LSG7G48VX1E0 1 | LSG7G48VX1E0 | **102599806093** |
| module(5) | LPU slot 1 | — | _empty_ |
| fan(9)* | LSG7G48VX1E0 2 | LSG7G48VX1E0 | **102599806089** |
| … | … | … | … |
| fan(9)* | LSG7SRUEX1C0 5 | LSG7SRUEX1C0 | **102599806030** |
| module(5) | POWER slot 1..6 | — | _empty_（华为此版漏洞） |
| stack(7) | FAN 1 | — | **5B2599001652** |
| stack(7) | FAN 2 | — | **5B2599001751** |

注：业务板与引擎在 HW ENTITY-MIB 中**实体化了两次**（module 占位 + fan class 实体带 SN），取后者即可；电源 module 占位但 SN 空 → Phase 48 走 CLI `display power` 兜底。

## 原 scope 摘要（已并入 Phase 48 v1）

- 引入 SNMP 拉 `entPhysicalTable`（板卡/电源/风扇），复用 `internal/device/snmp_client.go`
- 混合分层兜底：SNMP 拿不到 SN 的项 fallback 到 CLI 对应命令（华为 `display power`）
- **光模块仍走 CLI** `display interface transceiver`，不依赖 SNMP
- 目标 OID：`entPhysicalClass(.5)` / `entPhysicalSerialNum(.11)` / `entPhysicalModelName(.13)` / `entPhysicalContainedIn(.4)`（重建父子树）

## 实测坑

- gosnmp v1.35.0 的 `WalkFunc` 只收 `SnmpPDU`，OID 在 `dataUnit.Name`
- HW `MaxOids` 设 1 时也是 GETBULK-rejected，但基础 OID 单 GET 可成功 → 走单 GET 即可避免 false timeout
- ENTITY-MIB 子树 walk 全 timeout，但精确 OID GET 立即返回 → HW 对 GETBULK 大概率有限速/拒绝，v1 实现需按单 GET

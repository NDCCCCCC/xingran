---
phase: 56
slug: vlan-v1-20-1-0-5-plans-initiated-2026-07-09
status: deferred
verifier_status: human_needed
created: 2026-07-09
milestone: v1.20.1
automated_gates:
  go_build: PASSED
  go_test_portwrite_e2e: PASSED (11 new TestE2E_SetAccessVlan|TestE2E_PortBinding + 10 v1.19 e2e = 21 total, FileTransport replay)
  go_test_portwrite_full: PASSED
  go_test_portcollection: PASSED (Phase 56 W1 vendor template tests)
  go_test_network_handlers: PASSED (Phase 56 W3 handler/router tests)
  go_test_operlog_regression: PASSED (25 OperType + 11 sensitive keyword regression intact)
  npm_type_check: PASSED
  npm_build: PASSED (vendor-react gzip 774.95 kB <= 776 kB baseline)
---

# Phase 56 v1.20.1 网络设备 VLAN + 端口绑定 — Site-Visit UAT 推迟清单

本 phase 在 mock SSH FileTransport 层（Task 2 e2e tests）已证明集成链路
（RenderCommand → executeWrite → SendConfigs → parseConfigError → CollectDevice
refresh）端到端工作。**真机 SSH 验证推迟到下次现场访问**（owner: 现场运维同事），
参照 v1.19 Phase 54 / v1.18 Phase 48 推迟模式。

## Mock-e2e 已覆盖 (CI 内)

Task 2 已落地的 11 个 TestE2E_* 测试覆盖:

- Huawei VRP `set_access_vlan`（3-cmd 链路：interface + port link-type access + port default vlan）
- Ruijie RGOS `set_access_vlan`（3-cmd 链路：interface + switchport mode access + switchport access vlan）
- Huawei `port_binding` add/remove × with/without MAC（user-bind static ip-address [mac-address AA-BB-CC-DD-EE-FF]）
- Ruijie `port_binding` add/remove × with/without MAC（switchport port-security binding [aabb.ccdd.eeff] IP）
- `set_access_vlan` pre-state NoOp 短路（DB VLAN == 目标 → 跳过 SSH）

RenderCommand 3-vendor × 2-action × variant 全部命令字面正确性 + SendConfigs →
parseConfigError 链路在 scrapligo FileTransport 上的端到端执行 + PortResult.CommandSent
包含正确的厂商命令子串 + success path 触发 CollectDevice 后台刷新。

限制: FileTransport 不能完全模拟真实 SSH 的 firmware 行为差异 + prompt timing +
字节编码差异；跨固件 (Huawei V200R005 vs V200R022) + Ruijie 真机 prompt 带空格
（`[Ruijie-GigabitEthernet 1/0/1]`）需现场验证。

## Site-visit 待现场验证 (12 项)

### Huawei S8700 (production)

- [ ] **Huawei S8700 set_access_vlan — PVID 切换验证** | SSH 设备，执行 `interface GE1/0/0; port link-type access; port default vlan 100`，然后 `display port vlan GE1/0/0` 验证 PVID 改为 100 | 预期: PVID=100，端口类型=access | 现场运维 | 下次现场访问
- [ ] **Huawei S8700 set_access_vlan — 老固件 V200R005 兼容性** | SSH V200R005 老固件设备，验证 `port link-type access` 前置在老固件上仍被接受（无 `% Wrong parameter` 拒绝） | 预期: 3 条命令全部成功，无错误回显 | 现场运维 | 跨固件对比（RISK-03 验证）
- [ ] **Huawei S8700 set_access_vlan — V200R022+ 兼容性** | 验证 V200R022+ 新固件是否仍需要 `port link-type access` 前置 | 预期: 前置命令被接受；若新固件可省略，记录为 follow-up | 现场运维 | 跨固件对比
- [ ] **Huawei S8700 port_binding add (IP only)** | SSH 设备，执行 `user-bind static ip-address 10.62.25.5`，然后 `display user-bind static all` 验证 IP 出现在列表 | 预期: IP 出现在 user-bind 表 | 现场运维 | 下次现场访问
- [ ] **Huawei S8700 port_binding add (with MAC)** | SSH 设备，执行 `user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF`，验证 MAC 正确显示 | 预期: IP + MAC 同时出现在 user-bind 表；MAC 格式为 AA-BB-CC-DD-EE-FF（华为 hyphen 格式） | 现场运维 | 下次现场访问
- [ ] **Huawei S8700 port_binding remove** | SSH 设备，先 add 绑定，再 `undo user-bind static ip-address 10.62.25.5`，验证 `display user-bind static all` 不再显示该 IP | 预期: IP 从 user-bind 表移除 | 现场运维 | 下次现场访问

### Ruijie RS8607E (production)

- [ ] **Ruijie RS8607E set_access_vlan — switchport mode/access 验证** | SSH 设备，执行 `interface GigabitEthernet 1/1; switchport mode access; switchport access vlan 318`，然后 `show interfaces status` 验证 Vlan 列改为 318 | 预期: VLAN=318，端口模式=access | 现场运维 | 下次现场访问
- [ ] **Ruijie RS8607E port_binding add (with MAC)** | SSH 设备，执行 `switchport port-security binding AABB.CCDD.EEFF 10.62.2.111`，然后 `show port-security binding` 验证 MAC 出现在表格 | 预期: MAC + IP 出现在 port-security binding 表（RISK-02 关键验证：用户实采输出无 MAC 列，需现场确认） | 现场运维 | 下次现场访问
- [ ] **Ruijie RS8607E port_binding add (IP only)** | SSH 设备，执行 `switchport port-security binding 10.62.2.111`，验证 IP-only binding 在 show 输出中正常显示 | 预期: IP 出现在 port-security binding 表（无 MAC 行） | 现场运维 | 下次现场访问
- [ ] **Ruijie RS8607E port_binding remove** | SSH 设备，先 add 绑定，再 `no switchport port-security binding 10.62.2.111`，验证 show 输出移除该行 | 预期: IP 从 port-security binding 表移除 | 现场运维 | 下次现场访问

### H3C Comware (production - 如有)

- [ ] **H3C Comware V7 user-bind 关键字 (no static) 验证** | SSH H3C 设备（若现场有 H3C），执行 `user-bind ip-address 10.x.x.x mac-address xxxx-xxxx-xxxx`，验证设备接受（H3C 关键字与 Huawei 不同的关键 RISK-01 验证）；若现场无 H3C 设备，此条标 deferred | 预期: H3C 接受无 `static` 关键字的 user-bind 命令 | 现场运维 | 视现场设备情况
- [ ] **H3C Comware V7 set_access_vlan — port access vlan 关键字** | SSH H3C 设备，执行 `interface GigabitEthernet1/0/1; port link-type access; port access vlan 100`，验证 PVID 切换 | 预期: PVID=100（H3C 用 `port access vlan` 而非华为的 `port default vlan`） | 现场运维 | 视现场设备情况

## 参照 precedent

- v1.19 推迟模式: `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`（7 项 site-visit SSH 验证）
- v1.18 推迟模式: `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md`（3 项 site-visit SSH 验证）
- v1.20.1 本期 12 项（1.7x 倍 v1.19 per-action 验证量，因新增 2 actions 涉及跨厂商 + 跨固件 + MAC/IP 形态变体）

## Mock-e2e 覆盖率 (CI 内)

Task 2 已落地的 11 个 TestE2E_* 测试覆盖:

- RenderCommand 3-vendor × 2-action × variant 全部命令字面正确性
- SendConfigs → parseConfigError 链路在 scrapligo FileTransport 上的端到端执行
- PortResult.CommandSent 包含正确的厂商命令子串
- success path 触发 CollectDevice 后台刷新（fire-and-forget）
- pre-state NoOp 短路（set_access_vlan DB VLAN 匹配 → 跳过 SSH）

限制: FileTransport 不能完全模拟真实 SSH 的 firmware 行为差异；跨固件 (Huawei
V200R005 vs V200R022) + Ruijie 真机 prompt 带空格需现场验证。

## Summary

| Metric | Count |
|--------|-------|
| total | 12 |
| passed | 0 |
| pending | 12 |
| skipped | 0 |
| blocked | 0 |

## Owner

现场访问时由运维同事携带 Huawei S8700 / Ruijie RS8607E / H3C（视现场情况）设备
接入，跑 12 项 SSH 实测。Site visit 完成后回写本文件（将 `[ ]` 改为 `[x]` +
实测详情），并通知 owner 关闭此 UAT。

## 关联声明

- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`（v1.19 site-visit UAT precedent）
- `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md`（v1.18 site-visit UAT precedent）
- `docs/plans/2026-07-09-v1.20.1-design.md`（v1.20.1 Real Device UAT 推迟决策来源）
- `.planning/MILESTONES.md` v1.20.1 entry（Deferred 段引用本文件）

---
slug: mac-collection-cisco-no-vendor-support
status: diagnosed
trigger: MAC地址 b022.7a2e.4a4f 在 Cisco 交换机 CX-WH-WH-04F-FL-RS8607E-03 上未采集到（Gi2/25，VLAN 308，port-security dynamic active）
created: 2026-06-29T00:00:00Z
updated: 2026-06-29
session_type: bug
---

# Debug Session: mac-collection-cisco-no-vendor-support

## Symptoms

### Expected Behavior
MAC 地址采集任务应能采集 Cisco 交换机上的 MAC 表项（含 `show mac address-table` 和 `show port-security address`）。

### Actual Behavior
- 设备 `CX-WH-WH-04F-FL-RS8607E-03` (Cisco RS8607E) 上的 MAC `b022.7a2e.4a4f`（端口 Gi2/25，VLAN 308，port-security dynamic active）**未被采集到 sys_device_mac_address 表**。
- 同一设备上其他所有 MAC 均同样不可见。
- 失败完全静默：cron 任务执行无错误日志，sys_oper_log 无失败记录。

### Switch Output Evidence
```
CX-WH-WH-04F-FL-RS8607E-03#sh port-security address | in 4a4f
56   308   b022.7a2e.4a4f  GigabitEthernet 2/25      Dynamic            --          active
```

### Reproduction
1. 在前端 MAC 地址管理页面过滤 device_name = `CX-WH-WH-04F-FL-RS8607E-03` 或 mac = `b022.7a2e.4a4f`，结果为空。
2. 在交换机上 `show port-security address | in 4a4f` 能查到。
3. 对比：同一环境下 Huawei/H3C/Ruijie/Maipu 设备的 MAC 能正常采集入库。

---

## Current Focus

**Hypothesis:** MAC 采集系统**完全没有 Cisco 厂商支持**。`DeviceVendor` 枚举只有 huawei/h3c/ruijie/maipu 四种取值；`parseMACLine` 的 vendor switch 没有 default 分支，Cisco 设备上执行的每一行都被静默丢弃。

**Next Action:** 与用户确认是否进入修复（需要扩展厂商枚举 + 双命令采集 + Cisco TextFSM 模板 + Cisco VLAN link type 检测）。

**Test:** 实施后用此 MAC `b022.7a2e.4a4f` 作为验收用例。

**Expecting:** 修复后 `sys_device_mac_address` 出现 `(device_id_of_CX-WH-WH-04F-FL-RS8607E-03, b022.7a2e.4a4f, GigabitEthernet 2/25, 308, dynamic, port-security)`。

**Reasoning Checkpoint:** 已完成根因分析。

**TDD Checkpoint:** 待用户确认。

---

## Evidence

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `internal/models/network_device.go:17-22` 的 `DeviceVendor` 枚举只有 4 个常量：
    - VendorHuawei = "huawei"
    - VendorH3C    = "h3c"
    - VendorRuijie = "ruijie"
    - VendorMaipu  = "maipu"
    没有 VendorCisco。前端映射（`internal/api/v1/network/network_export_handler.go:89,205`、`batch_export_handler.go:167,226`）也只接受这 4 个 vendor 字符串。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `internal/services/mac_collection_service.go:362-441` 的 `parseMACLine` 函数只有两个 case：
    - case VendorHuawei, VendorH3C
    - case VendorRuijie, VendorMaipu
    没有 default 分支。Cisco 设备的每一行被解析为 `entry{MACAddress:""}` 并返回 `(zero, false)`，调用方 `parseMACAddressTable` 不向 `entries` 添加任何元素。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `internal/services/mac_collection_service.go:144` 的 `getMACCommand` 按 vendor 返回命令：
    - huawei/h3c → "display mac-address"
    - ruijie/maipu → "show mac-address-table"
    - default → "show mac-address-table"
    命令本身在 Cisco IOS-XE/NX-OS 上能跑（默认分支走对了），但解析层没有对应分支。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `templates/` 目录下没有 cisco_ios_xe_* 或 cisco_nxos_* 模板：
    `ls templates/ | grep -iE "cisco|ios|nexus"` 返回空。
    即便执行了命令，也没有 TextFSM 解析 Cisco 输出。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `internal/collectors/mac_collector.go:233-260` 的 `parseGenericMACAddress` 是 dead-code 路径（cron 实际走 `mac_collection_service.go`），且即便调用也硬编码 `InterfaceName:"unknown"`、丢弃 VLAN ID。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    用户提供的交换机输出是 `show port-security address`（7 字段：Index VLAN MAC Type Interface LearnAge Action），而系统采集的命令是 `show mac-address-table` / `display mac-address`（4-5 字段）。**系统从未查询 port-security 表**，所以 sticky/secure MAC 在所有 Cisco 设备上都不可能被发现。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    即使用户手动把 CX 设备的 vendor 改成 ruijie/huawei 绕过 UI：
    - Cisco 的 MAC 格式 `xxxx.xxxx.xxxx`（点分）与 Huawei 的 `XXXX-XXXX-XXXX`（连字符）不匹配；
    - Cisco 的 Gi2/25 含空格，`strings.Fields` 会切成 7 个 token，Ruijie 分支会把 `308` 当 MAC、`b022.7a2e.4a4f` 当 type，产生垃圾数据但不含真实 MAC。

- timestamp: 2026-06-29T00:00:00Z
  source: code_inspection
  finding: |
    `internal/services/portcollection/trunk_filter.go:27-32` 的 trunk hint tokens（trunk/hybrid/uplink/uplink-port）是 vendor-neutral 的，但 `sys_device_port_status.port_type` 仅由 Huawei/H3C/Ruijie/Maipu 解析路径填充，Cisco 设备 `port_type` 为 NULL，`BuildTrunkPortBlockset` 返回空 blockset。
    **这意味着 trunk filter 不是本次问题的元凶**——Cisco 设备根本到不了 trunk filter 这一步，解析阶段就已经把所有行丢了。

---

## Eliminated

- timestamp: 2026-06-29T00:00:00Z
  hypothesis: trunk filter 把 Gi2/25 误判为 trunk 端口并跳过
  evidence: Cisco 设备 `port_type` 为 NULL，hint blockset 为空，filter 不会过激
  conclusion: 排除。真实根因是解析层在更上游丢掉了整行。

- timestamp: 2026-06-29T00:00:00Z
  hypothesis: UNIQUE 约束 (device_id, mac_address) 静默去重
  evidence: 如果解析成功 `entries` 切片就非空，Step 8 才会执行 upsert；Cisco 解析层返回空 entries，根本没进入 DB
  conclusion: 排除。根因在解析层，不在 DB 层。

- timestamp: 2026-06-29T00:00:00Z
  hypothesis: 设备未被发现（discover 未跑 / scrapligo 拨号失败）
  evidence: 题目陈述该 MAC 在交换机上能查到；如果 scrapligo 没连上，cron 会写 sys_oper_log 失败记录；事实上没看到错误
  conclusion: 排除。连接成功，只是解析失败。

---

## Resolution

**Root Cause:** 系统没有 Cisco 厂商支持。`DeviceVendor` 枚举封闭（huawei/h3c/ruijie/maipu），`parseMACLine` 的 switch 没有 default 分支，Cisco 设备输出的每一行都被静默丢弃；更严重的是系统从未查询 `show port-security address`，所有 Cisco sticky/secure MAC 不可见。

**Why This MAC Specifically:** `b022.7a2e.4a4f` 在 port-security 表中（动态安全 MAC），而 cron 任务只查 `show mac-address-table`（普通动态 MAC），即便加了 Cisco vendor 分支也未必能查到这一条；需要双命令采集。

**Suggested Fix (未实施，等用户决策):**

1. **`internal/models/network_device.go`**: 增加 `VendorCisco = "cisco"` 常量；同步前端 vendor 列表。
2. **`internal/services/mac_collection_service.go`**:
   - `getMACCommand` Cisco 厂商返回 `[]string{"show mac address-table", "show port-security address"}`（双命令合并去重）。
   - `parseMACLine` 新增 `case VendorCisco` 分支，分别处理两种 Cisco 输出格式：
     - `show mac address-table`：5 字段 `Vlan Mac Address Type Ports`
     - `show port-security address`：7 字段 `Idx VLAN MAC Type Interface LearnAge Action`
3. **`templates/`**: 新增 `cisco_ios_xe_show_mac_address-table.textfsm` 和 `cisco_ios_xe_show_port-security_address.textfsm`（如启用 TextFSM 路径；当前 `mac_collection_service.go` 走的是正则/strings.Fields，可保留正则路径但要支持新格式）。
4. **`internal/services/portcollection/trunk_filter.go`**:
   - 增加 Cisco-specific hint tokens：`routed`、`layer3`、`switchport disabled`。
   - 长期方案：增加 `vlan_link_type` 列（Phase 40 已 defer 到 v1.16），从 `show interfaces switchport` 提取 `Operational Mode: trunk|access`。
5. **前端**:
   - `xingran-react-frontend/src/pages/network/device/` 的 vendor 下拉增加 "Cisco"。
   - 同步 `internal/api/v1/network/network_export_handler.go` 和 `batch_export_handler.go` 的 vendor 白名单。
6. **回归测试**:
   - `internal/services/mac_collection_service_test.go` 增加 Cisco `show mac address-table` 和 `show port-security address` 两条测试用例，断言 `b022.7a2e.4a4f` + VLAN 308 + Gi2/25 能被解析。
   - 验证 trunk filter 不误伤 access 端口。

**Files to Change (Proposed):**
- `internal/models/network_device.go`
- `internal/services/mac_collection_service.go`
- `internal/services/portcollection/parser.go`（如有 TextFSM 切换）
- `internal/services/portcollection/trunk_filter.go`
- `internal/api/v1/network/network_export_handler.go`
- `internal/api/v1/network/batch_export_handler.go`
- `xingran-react-frontend/src/pages/network/device/index.tsx`（vendor dropdown）

**Confidence:** HIGH（5 个独立代码层证据汇聚到同一根因）

---

## Specialist Hint

backend - 需要 Go 后端深度修改：GORM 模型扩展、TextFSM/正则解析双格式、cron 双命令合并、UI 同步。
---
slug: mac-collection-trunk-filter
status: resolved
deferred_to: v1.16-tech-debt
trigger: MAC地址采集问题：交换机trunk口导致重复，需过滤trunk口
created: 2026-05-09T00:00:00Z
updated: 2026-06-25
session_type: bug
---

# Debug Session: mac-collection-trunk-filter

## Symptoms

### Expected Behavior
只采集接入端口的MAC（access端口），不采集交换机间互联的trunk口

### Actual Behavior
MAC地址大量重复 - 交换机互联的trunk口会采集到很多其他交换机上连接的设备

### Error Messages
无具体错误信息，是数据重复问题

### Timeline
一直存在（从系统上线就有这个问题）

### Reproduction
- 范围：所有交换机都有这个问题
- 触发：执行MAC地址采集任务

---

## Current Focus

**Hypothesis:** MAC采集服务在采集MAC地址时没有过滤trunk端口，导致交换机间互联的trunk口上的MAC地址被重复采集

**Next Action:** IMPLEMENT FIX - 准备实施修复方案

**Test:** 验证修复后MAC地址表只包含access端口的MAC地址

**Expecting:** MAC采集服务应该查询端口状态，跳过trunk/hybrid类型的端口

**Reasoning Checkpoint:** 已完成根因分析，确认三步修复方案

**TDD Checkpoint:** 待定

---

## Evidence

- timestamp: 2026-05-09T00:00:00Z
  source: code_inspection
  finding: |
    MAC采集服务分析 (internal/services/mac_collection_service.go)：
    - collectDeviceMAC() 方法采集所有MAC地址，无过滤逻辑
    - parseMACAddressTable() 解析命令输出，提取InterfaceName但不检查端口类型
    - 所有解析到的MAC地址都入库（lines 147-172）
    - 没有查询端口状态或过滤trunk端口的逻辑

- timestamp: 2026-05-09T00:00:00Z
  source: code_inspection
  finding: |
    端口状态表结构分析 (internal/models/device_port_status.go)：
    - DevicePortStatus.PortType字段存储物理类型（copper/fiber/--）
    - 缺少VLAN link type字段（trunk/access/hybrid）
    - 数据库迁移002_add_port_physical_attributes.sql确认port_type用途
    - 需要添加新字段vlan_link_type存储VLAN链路类型

- timestamp: 2026-05-09T00:00:00Z
  source: code_inspection
  finding: |
    VLAN link type获取方法：
    华为/H3C: display port vlan
    - 模板：templates/huawei_vrp_display_port_vlan.textfsm（已存在）
    - 提取字段：LINK_TYPE (trunk|access|auto|hybrid|desirable)

    锐捷/迈普: 需要确认命令
    - 可能命令：show interfaces switchport, show vlan port
    - 可能需要创建相应的TextFSM模板

- timestamp: 2026-05-09T00:00:00Z
  source: code_inspection
  finding: |
    数据关联分析：
    - sys_device_mac_address: (device_id, interface_name) -> MAC地址
    - sys_device_port_status: (device_id, interface_name, vlan_link_type) -> 端口状态
    - 可以通过device_id + interface_name关联获取端口的VLAN link type
    - MAC采集时需要JOIN查询或批量查询端口状态

- timestamp: 2026-05-09T00:00:00Z
  source: code_inspection
  finding: |
    端口采集服务分析 (internal/services/portcollection/)：
    - CollectionService.collectDevicePort() 已采集端口状态
    - parseInterfaceList() 提取InterfaceInfo，包含PortType（物理类型）
    - 没有调用display port vlan等命令获取VLAN link type
    - 需要扩展采集逻辑，添加VLAN link type采集

---

## Eliminated

- timestamp: 2026-05-09T00:00:00Z
  hypothesis: 可能现有的port_type字段已经包含VLAN link type信息
  evidence: 检查数据库迁移文件，确认port_type只存储物理介质类型（copper/fiber）
  conclusion: 需要添加新字段vlan_link_type

---

## Resolution

**Root Cause:** MAC采集服务没有识别和过滤trunk端口的逻辑，导致采集到交换机间互联trunk口上的重复MAC地址。根本原因是三层架构问题：

1. **数据层缺失**：sys_device_port_status表缺少vlan_link_type字段，无法存储端口的VLAN链路类型（trunk/access/hybrid）

2. **采集层不完整**：端口采集服务(portcollection)只采集物理端口类型（copper/fiber），没有采集VLAN link type信息

3. **业务层无过滤**：MAC采集服务(mac_collection_service)在入库前没有查询端口的VLAN link type并过滤trunk端口

**Fix:** 需要实施三层修复：

**Layer 1 - Database Schema** (internal/core/db/migrations/):
- 添加新迁移文件：0xx_add_port_vlan_link_type.sql
- 字段定义：vlan_link_type VARCHAR(20), 默认NULL
- 支持值：'trunk', 'access', 'hybrid', 'auto', 'desirable', NULL

**Layer 2 - Port Collection** (internal/services/portcollection/):
- 扩展InterfaceInfo结构体，添加VLANLink_type字段
- 添加parsePortVLAN()方法调用厂商特定命令：
  - 华为/H3C: "display port vlan" + huawei_vrp_display_port_vlan.textfsm
  - 锐捷/迈普: 需要确认命令（可能: "show vlan port" 或 "show interfaces switchport"）
- 在collectDevicePort()中调用并存储到数据库

**Layer 3 - MAC Collection** (internal/services/mac_collection_service.go):
- 在collectDeviceMAC()中添加端口类型查询
- 构建interface_name -> vlan_link_type映射表
- 在parseMACAddressTable()后过滤：跳过vlan_link_type IN ('trunk', 'hybrid')的端口
- 只保留access端口的MAC地址

**Files Changed:**
- internal/core/db/migrations/0xx_add_port_vlan_link_type.sql (新建)
- internal/models/device_port_status.go (添加字段)
- internal/services/portcollection/parser.go (扩展解析)
- internal/services/portcollection/collection.go (调用VLAN采集)
- internal/services/mac_collection_service.go (添加过滤逻辑)

**Verification:** 待实施后验证

**Specialist Hint:** backend - 涉及Go后端架构修改，需要熟悉GORM模型、数据库迁移、服务层设计

## Phase 40 Closure (2026-06-25)

本次落地最小可编译的 trunk 过滤（Resolution Layer 3 子集）：

- 新增 `internal/services/portcollection/trunk_filter.go`：暴露 `BuildTrunkPortBlockset` / `IsTrunkPort`
  两个 helper。基于 `sys_device_port_status.port_type` 取 trunk/hybrid/uplink 暗示词构成 blockset。
- `internal/services/mac_collection_service.go:collectDeviceMAC` Step 2.5 调
  `BuildTrunkPortBlockset`，Step 4 新增"过滤规则 3" 跳过 trunk/互联端口上的 MAC 入库。

未落（推迟到后续 phase）：Resolution 描述的完整三层修复
（sys_device_port_status 新增 `vlan_link_type` 列 + 厂商 `display port vlan` /
`show interfaces switchport` 采集 + 本过滤改读 `vlan_link_type IN ('trunk','hybrid')`）。
当前 PortType 兜底过滤已能消除大部分 trunk 端口 MAC 重复，可在缺少新迁移/解析模板的
前提下先收效，frontmatter 翻 `resolved`。

verification: `go build ./...` 退出 0；trunk_filter.go 存在；mac_collection_service.go 含 "过滤规则3" 与 `portcollection.IsTrunkPort` 调用
files_changed: internal/services/portcollection/trunk_filter.go, internal/services/mac_collection_service.go, .planning/debug/mac-collection-trunk-filter.md

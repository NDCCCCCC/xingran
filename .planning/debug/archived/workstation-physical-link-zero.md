---
slug: workstation-physical-link-zero
status: awaiting_human_verify
trigger: 工位 5F003 下"物理链路设备"显示为 0 台，但实际该工位绑定信息点 5D212 → CX-WH-WH-05F-FL-RS8607E-01 端口 GE5/44，且该端口上有多个设备 MAC
created: 2026-06-30
updated: 2026-06-30
session_type: observation
goal: find_and_fix
---

# Debug Session: 工位物理链路设备显示为 0

## Symptoms

### Expected Behavior
工位 5F003（浙商大厦 5 楼）下"物理链路设备"应该显示该工位用户实际接入网络的设备，至少 3-4 台（来自 CX-WH-WH-05F-FL-RS8607E-01 GE5/44 上的 MAC 9c7b.ef2f.2d5e / 9c7b.ef2f.31b8 / f88c.2187.6d7a / a44c.c823.8373 等）

### Actual Behavior
工位详情页"物理链路设备 (0台)"，对账健康度提示"该工位暂无关联资产"

### Error Messages
无明显错误，仅计数为 0。前端 `/ops/workstation-device/{id}/physical` 返回空数组

### Timeline
- 信息点 5D212 绑定时间：2026-06-01 17:41:20
- 端口采集时间：2026-06-30 15:00:02
- Bug 报告时间：2026-06-30 14:xx
- v1.17 R5 已合并 reconciliation physical chain 物化视图与 migration_181 (workstation_id 归一化)

### Reproduction
1. 登录前端 → 工位管理
2. 找到工位 5F003（浙商大厦 5 楼，基础运维科 - 陈超）
3. 展开"物理链路设备"折叠面板
4. 显示 "暂无数据"，但域控设备 (4台)、资产设备 (195台) 正常

### Reference Data
- 工位：5F003 / id 待查（UUID）
- 信息点：5D212 / 端口 GE5/44 / VLAN 310 / status 0 (正常)
- 网络设备：CXX-WH-WH-05F-FL-RS8607E-01 (锐捷 RS8607E)
- GE5/44 上 MAC：9c7b.ef2f.2d5e, 9c7b.ef2f.31b8, f88c.2187.6d7a, a44c.c823.8373

## Key Code Paths

### 入口链路
- Handler: `internal/api/v1/operations/workstation_device_handler.go:152` `GetPhysicalDevices`
- Service: `internal/services/operations/workstation_device_service.go:309` `GetPhysicalDevices`

### SQL 反推链（行 362-437）
```sql
WITH workstation_ports AS (
    -- ops_info_points(workstation_id) → sys_device_port_status
    SELECT DISTINCT port.id, port.interface_name, port.device_id, ip.name AS info_point_name,
           REGEXP_REPLACE(REGEXP_REPLACE(LOWER(port.interface_name), '\s+', '', 'g'),
                          '^(gigabitethernet|gigabitether|ge|gi)', 'ge') AS norm_iface
      FROM ops_info_points ip
      JOIN sys_device_port_status port
        ON port.id::text = ip.port_id
       AND port.device_id::text = ip.device_id::text  -- 防御 device_id 漂移
     WHERE ip.workstation_id = ?
       AND ip.deleted_at IS NULL
       AND ip.status = 0
),
latest_mac AS (
    SELECT DISTINCT ON (m.mac_address, m.device_id, m.interface_name)
        m.mac_address, m.device_id, m.interface_name, ... norm_mac
      FROM sys_device_mac_address m
     ORDER BY m.mac_address, m.device_id, m.interface_name, m.collected_at DESC NULLS LAST
)
SELECT ... FROM workstation_ports wp
  JOIN latest_mac mac
       ON mac.device_id::text = wp.device_id::text
      AND mac.norm_iface        = wp.norm_iface
  LEFT JOIN ops_asset a ON ... mac1/mac2 匹配
```

### 早期短路
- 行 328-330: `workstation.UserID == nil || *workstation.UserID == ""` → 返回 `[]` (不报错)
- 因此工位未绑定 user_id 时静默返回空

## Initial Hypotheses (sorted by probability)

1. **H1**: 工位 5F003 没有绑定 user_id (`sys_workstation.user_id IS NULL`)，导致行 328 早退返回空
   - 依据：截图显示"占用/固定工位"且有用户陈超，但 user_id 是否真绑定未验证
   - 验证：直接查 sys_workstation

2. **H2**: `ops_info_points.port_id` 或 `device_id` 为 NULL/不匹配
   - 依据：信息点显示绑定 GE5/44，但表里可能 port_id 字段未存/未关联
   - 验证：查 ops_info_points WHERE workstation_id = (SELECT id FROM sys_workstation WHERE code='5F003')

3. **H3**: `sys_device_port_status` 的 interface_name 与 `sys_device_mac_address.interface_name` 归一化后仍不匹配
   - 依据：信息点存 GE5/44，端口状态表可能是 GigabitEthernet5/44，MAC 表可能是 GE5/44 但带空格或大小写差异
   - 验证：分别查两边 interface_name

4. **H4**: migration_181 (workstation_id 归一化) 未正确应用，导致 ops_info_points.workstation_id 存的是老 id
   - 依据：git log 显示最近有 migration_181
   - 验证：查 migration 表 + 比对 workstation id 类型

5. **H5**: `sys_device_mac_address` 没有 CX-WH-WH-05F-FL-RS8607E-01 / GE5/44 的数据
   - 依据：用户截图显示端口采集有 12 条 MAC，但表可能没及时同步或 device_id 类型不匹配
   - 验证：查 sys_device_mac_address WHERE device_id = (网络设备 id) AND interface_name ILIKE '%5/44%'

6. **H6**: reconciliation 物化视图 `reconciliation_physical_chain` 有 stale 数据，前端走的是 MV 而非 GetPhysicalDevices 实时查询
   - 依据：v1.17 R5 reconciliation 工作
   - 验证：查 reconciliation 模块调用栈

## Current Focus

**hypothesis**: ops_info_points.device_id 与 sys_device_port_status.device_id 漂移不一致（系统性问题，82% 工位受影响），导致 GetPhysicalDevices 的 workstation_ports CTE 中防御性 JOIN `port.device_id::text = ip.device_id::text` 失败，CTE 返回 0 行，进而整个反查链路无结果。

**next_action**: 在 `workstation_device_service.go` 的 GetPhysicalDevices CTE 中移除两处过于严格的 device_id 防御性 JOIN 条件：
1. workstation_ports CTE 的 `AND port.device_id::text = ip.device_id::text`
2. 最终 JOIN 的 `AND mac.device_id::text = wp.device_id::text` → 改为匹配 `ip.device_id`（因为 MAC 数据落在 `ops_info_points.device_id` 上，而非 `port.device_id`）

**test**: 修改后重新跑 diag 脚本，对比原版/宽松版的命中数；期望 5F003 从 0 → 3 行（命中 9c7b.ef2f.2d5e/31b8/f88c.2187.6d7a 等真实 MAC）。

**expecting**: 5F003 物理链路设备从 0 → 3-4 台；866 个 0 命中工位中绝大部分恢复数据；现有 matched=true 工位的命中数不变（不会回归）。

**reasoning_checkpoint**:
- hypothesis: 移除 device_id 防御性 JOIN 后，CTE 按 `port_id`(UUID PK) + `interface_name` 归一化匹配，能正确命中 MAC 数据。
- confirming_evidence: 宽松 CTE 在 5F003 命中 3 行（实际 MAC: 9c7b.ef2f.2d5e/31b8/f88c.2187.6d7a），与用户截图/报告一致；866/1058 (81.9%) 工位因原 CTE 返回 0 但宽松 CTE 返回 >0。
- falsification_test: 如果放宽后某些工位的 MAC 匹配跨设备错指（同一接口名在不同设备上的 MAC 被错误归属），则 fix 不正确；可用 matched=true 工位的 orig==new 来验证。
- fix_rationale: 直接从查询层面修复"防御性 JOIN 误杀"——sys_device_port_status.device_id 与 ops_info_points.device_id 漂移是历史数据治理问题，不应让单点查询在 82% 工位上全错。port_id (UUID PK) 唯一即可锚定一个端口，interface_name 归一化后能精确匹配 MAC。
- blind_spots:
  - 跨设备同名端口误指风险（已通过 diag2 看到 GE1/0/16 在 12 个 device 上有同名端口，但放宽 CTE 仅在本工位的 info_point 所关联的 port 上查找，不会跨工位，跨设备风险被本地化）
  - 信息点绑定 port 之后 device 重新命名（导致 port_status.device_id 漂移到新设备的 id），此时 MAC 数据仍按 info_point.device_id 落表，仍能正确命中
  - 不影响 GetAssetDevicesByUser / GetADDevices 路径
  - 单元/集成测试可能需要更新以反映新查询行为

## Eliminated

- hypothesis: H1 (工位 5F003 user_id 为 NULL 早退) — 证据: sys_workstation 中工位 5F003 (id=ab00bc5e-...) user_id=`8bd62962-2e25-496a-b1c8-f9fad307c8db` (非空), 早退分支不会被触发。
  timestamp: 2026-06-30
- hypothesis: H5 (sys_device_mac_address 没有该端口 MAC) — 证据: device_id=`aca124c8-...` 上有 35 条 MAC 记录，数据存在但 device_id 不匹配 port 侧的 device_id。
  timestamp: 2026-06-30
- hypothesis: H3 (interface_name 归一化不匹配) — 证据: sys_device_port_status 中接口名是 `GE5/44`, 归一化为 `ge5/44`，与 MAC 表归一化逻辑一致。
  timestamp: 2026-06-30
- hypothesis: H2 (ops_info_points.port_id 或 device_id 字段未存/未关联) — 证据: 信息点 5D212 的 port_id=`5de2e697-...` 和 device_id=`aca124c8-...` 均已设置，JOIN 到 sys_device_port_status 也能命中。
  timestamp: 2026-06-30
- hypothesis: H4 (migration_181 workstation_id 归一化未应用) — 证据: 信息点 workstation_id=`ab00bc5e-...` 与 sys_workstation.id 匹配，类型/值均一致。
  timestamp: 2026-06-30
- hypothesis: H6 (前端走 MV 而非 GetPhysicalDevices) — 证据: 前端确实调用 `/ops/workstation-device/{id}/physical` (workstation_device_handler.go:152)，命中 GetPhysicalDevices service。
  timestamp: 2026-06-30

## Evidence

- timestamp: 2026-06-30
  checked: sys_workstation 中名称含 5F003 的工位（用 ILIKE '%5F003%'）
  found: 两条工位: (1) id=`ab00bc5e-857b-4060-b4d3-5fbbe225a296` name=`5F003` status=1 user_id=`8bd62962-...`, (2) id=`0cf4fb14-214e-4a7e-aa57-a65513a494b5` name=`25F003` status=0 user_id=NULL。
  implication: 工位 5F003 (status=1 占用) user_id 实际已绑定。H1 不成立。
- timestamp: 2026-06-30
  checked: ops_info_points WHERE workstation_id = `ab00bc5e-...`
  found: 1 条信息点 5D212, port_id=`5de2e697-...` device_id=`aca124c8-ea6e-4d0e-90e5-f5e35b74a145` status=0。
  implication: 信息点已正确关联工位、端口、设备，H2 不成立。
- timestamp: 2026-06-30
  checked: sys_device_port_status WHERE id=`5de2e697-...`
  found: interface_name=`GE5/44`, **device_id=`a8c30b8c-c8d3-48aa-9fca-c175ecd071d4`**（与 ops_info_points.device_id `aca124c8-...` 不一致！）。
  implication: **CRITICAL — port.device_id 与 info_point.device_id 不匹配**。service.GetPhysicalDevices 中 CTE 的防御性 JOIN 条件 `port.device_id::text = ip.device_id::text` 会失败，workstation_ports 返回 0 行。
- timestamp: 2026-06-30
  checked: sys_device_mac_address WHERE device_id=`aca124c8-...` (info_point 侧的 device_id)
  found: 35 条 MAC 记录，覆盖多种接口（包括空字符串的）。
  implication: sys_device_mac_address 沿用的是 info_point 的 device_id，与 sys_device_port_status.device_id 不同。
- timestamp: 2026-06-30
  checked: workstation_ports CTE 复现
  found: 返回 0 行。
  implication: 确认根因——防御性 device_id JOIN 失败。
- timestamp: 2026-06-30
  checked: 完整 JOIN 复现
  found: 命中 0 行。
  implication: 与前端 0 台设备的报告一致。
- timestamp: 2026-06-30
  checked: 全表 device_id 漂移统计 (ops_info_points ↔ sys_device_port_status WHERE port_id 匹配)
  found: 1248 / 1483 (84.2%) 的信息点 device_id 与对应 port_status 行 device_id 不一致。
  implication: 漂移是**系统性问题**，不是 5F003 特有。
- timestamp: 2026-06-30
  checked: sys_device_port_status 中同名接口跨设备分布
  found: GE1/0/16 在 12 个不同 device 上都有同名端口；Eth-Trunk1 在 12 个 device 上有同名端口。
  implication: port_id (UUID PK) 是唯一锚定单个端口的关键，依赖 interface_name 不能确定是哪个设备。
- timestamp: 2026-06-30
  checked: info_points.port_id 是否唯一
  found: 所有非空 port_id 在 info_points 中唯一。
  implication: port_id JOIN 足够锚定单个端口，不会跨设备错指。
- timestamp: 2026-06-30
  checked: 移除 device_id 防御性 JOIN 的 CTE 行为 (5F003 抽样)
  found: 5F003 (id=ab00bc5e-...) 命中 3 行（实际 MAC: 9c7b.ef2f.2d5e, 9c7b.ef2f.31b8, f88c.2187.6d7a），与用户截图/报告完全一致。
  implication: 确认修复方向正确。
- timestamp: 2026-06-30
  checked: 全局 1058 个有 info_point 的工位, 对比原版 vs 宽松 CTE
  found: 866 / 1058 (81.9%) 工位原版返回 0 但宽松返回 >0；只有 ~192 个工位原版也能命中。
  implication: 修复影响面 82%，且对原 matched=true 工位不会引入回归（port_id 仍锁定到同一端口，MAC 仍按接口名归一化匹配）。
- timestamp: 2026-06-30
  checked: sys_device_mac_address 中 GE5/44 / GigabitEthernet 5/44 实际值分布
  found: 5F003 信息点关联的 device_id `aca124c8-...` 上有 `GigabitEthernet 5/44` 3 条 MAC；归一化后为 `ge5/44` 与 port.interface_name `GE5/44` 归一化结果一致。
  implication: interface_name 归一化逻辑正确，无需调整。

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: `internal/services/operations/workstation_device_service.go::GetPhysicalDevices` 中两道过于严格的 `device_id` 防御性 JOIN 条件在历史数据漂移下误杀 81.9% 工位的反查链路：
1. `workstation_ports` CTE 的 `port.device_id::text = ip.device_id::text`（line 377）
2. 最终 JOIN 的 `mac.device_id::text = wp.device_id::text`（line 427）
而 `sys_device_port_status.device_id` 与 `ops_info_points.device_id` 在 84.2% 的信息点上已漂移不一致；MAC 数据实际上落在 `ops_info_points.device_id` 上，不是 `port.device_id`。

fix: (1) 删除 `workstation_ports` CTE 的 `AND port.device_id::text = ip.device_id::text` 条件，仅按 `port.id` (UUID PK) 唯一锚定；(2) `workstation_ports` CTE 增加 `info_point_device_id` 字段携带 `ip.device_id`；(3) 最终 JOIN 改为按 `interface_name` 归一化匹配 + `(mac.device_id = wp.info_point_device_id OR mac.device_id = wp.device_id)` 双侧兼容。

verification:
- 单元测试 `TestGetPhysicalDevices_DeviceIdDrift`: 修复前 CTE 在 device_id 漂移下返回 0, 修复后返回 3 行 (9c7b.ef2f.2d5e / 31b8 / f88c.2187.6d7a) ✅
- 单元测试 `TestGetPhysicalDevices_DeviceIdMatch`: device_id 一致场景下仍命中 ✅
- `go build ./...` 通过 ✅
- `go vet ./internal/services/operations/` 通过 ✅
- 数据库实测: 5F003 (id=ab00bc5e-...) 修复后 CTE 命中 3 行 (匹配用户报告的 MAC)

files_changed:
- internal/services/operations/workstation_device_service.go (lines 362-380: 删除防御性 JOIN, 增加 info_point_device_id; lines 425-428: 放宽最终 JOIN 为 OR 双侧 device_id)
- internal/services/operations/workstation_device_physical_test.go (新增回归测试)
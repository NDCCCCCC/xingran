---
slug: port-collection-ownership-clash
status: resolved
goal: find_and_fix
trigger: |
  端口采集代码功能更新后，已存在的端口不再更新状态。日志大量 WARN:
  "[PortCollection] <device>: skip iface GE4/0/23 (already owned by device <other>, ownership clash)"
  最终 "all ports have ownership clashes, skip batch save"。前端显示端口采集时间停在几天前。
  附带: 网络设备统计页 GORM 错误 (Scan 参数不匹配 + sys_department 表不存在)。
created: 2026-07-08
updated: 2026-07-08
---

# Debug: 端口采集不更新 (ownership clash 误判)

## Symptoms
- **Expected**: 端口采集应刷新已存在端口的 admin/oper_status、collected_at。
- **Actual**: 除首个写入设备外，其余设备的端口全部 skip，collected_at 停在首次写入时间。
- **Errors**:
  - `WARN [PortCollection] <dev>: skip iface GE4/0/23 (already owned by device <uuid>, ownership clash)`
  - `WARN [PortCollection] <dev>: all ports have ownership clashes, skip batch save`
  - (统计页) `sql: expected 2 destination arguments in Scan, not 1`
  - (统计页) `relation "sys_department" does not exist (SQLSTATE 42P01)`
- **Timeline**: 自 `0b95834a` (feat 45-r5, 2026-06-30) 引入 ownership clash 检查后开始。
- **Reproduction**: 对任意两台都有 GE4/0/23/Vlanif26/NULL0/MEth0/0/0 等通用接口名的设备触发采集。

## Current Focus
- **hypothesis**: CONFIRMED — [C-fix] ownership clash 逻辑基于错误前提 (interface_name 跨设备唯一)。
- **next_action**: 移除错误块 + 修测试 + 修统计页 GORM + go build/test 验证。

## Root Cause (CONFIRMED, 三方证据互证)

`internal/services/portcollection/collection.go:260-309` 的 **[C-fix] 防同名异物冲突** 块:

```go
Where("interface_name IN ? AND device_id <> ?", ifaceNames, device.ID)
// 命中即 skip 该端口,认定 ownership clash
```

它假设 `interface_name` 跨设备唯一,任何"别的设备也有同名 interface"都是冲突。

**证据 1 — DB 唯一约束是复合键 (device_id, interface_name),多设备同名本就合法**:
- `models/device_port_status.go:33-34`: `uniqueIndex:uniq_device_interface,priority:1/2` (DeviceID + InterfaceName 复合)
- `migration_177`: `ALTER TABLE ... ADD CONSTRAINT uniq_device_interface UNIQUE (device_id, interface_name)`

**证据 2 — UPSERT 本身正确 (collection.go:311-314)**:
`clause.OnConflict{Columns:[device_id, interface_name]}` — 同设备更新、跨设备各自插入,语义正确。

**证据 3 — [C-fix] 与上述直接矛盾**: GE4/0/23、Vlanif26、NULL0、MEth0/0/0 在交换机间普遍重复 (每台华为 S8700 都有 GE4/0/1~48),该查询几乎对所有非首采设备返回"冲突",全 skip。

**结论**: [C-fix] 块多余且有害,复合唯一键 + OnConflict 已正确处理。整块移除即可。

## Fix Plan
1. `collection.go`: 移除 260-309 整块 (注释 + 匿名块)。
2. `collection_test.go`: 删除 `TestOwnershipClashCheck_*` (锁定错误行为); 新增 `TestMultiDeviceSameInterfaceNameCoexist` 守护复合唯一键语义。
3. `network_device_service.go` GetDeviceStatistics:
   - `Scan(&map[string]int64)` (双列) → `Scan(&[]struct{Key,Count})` 再转 map (device_type/vendor/dept 三处)。
   - `LEFT JOIN sys_department` → `LEFT JOIN sys_dept`。
4. `go build ./...` + `go test ./internal/services/portcollection/`。

## Evidence
- 2026-07-08: 读取 collection.go:260-309 确认 [C-fix] 块逻辑与 OnConflict 复合键矛盾。
- 2026-07-08: `git log -S "ownership clash"` → 由 0b95834a 引入。
- 2026-07-08: models/device_port_status.go + migration_177 确认复合唯一键。
- 2026-07-08: network_device_service.go:521-539 定位统计页两个 GORM bug。
- 2026-07-08: models/dept.go:27 确认正确表名 sys_dept。

## Resolution
- root_cause: [C-fix] ownership clash 误把跨设备同名通用接口名 (GE4/0/23 等) 当冲突 skip。
- fix: 移除 collection.go 的 [C-fix] 块 (替换为防回归注释); 重写 collection_test.go 为正向回归守护 (TestMultiDeviceSameInterfaceNameCoexist); 修 GetDeviceStatistics 三处 Scan(&map)→Scan(&[]struct)+转 map, 并 sys_department→sys_dept。
- files_changed: internal/services/portcollection/collection.go, internal/services/portcollection/collection_test.go, internal/services/network_device_service.go
- verification: `go build ./...` EXIT 0; `go test ./internal/services/portcollection/` PASS — TestMultiDeviceSameInterfaceNameCoexist 验证两台设备同名 GE0/0/1 共存 + 再采集时 OnConflict 各自 UPSERT 不串扰、不新增行。
- deploy_note: 代码层修复需重新编译并部署后端二进制才会对生产生效 (当前生产跑的是改动前二进制); 部署后已存在端口的 collected_at 会在下一次采集周期恢复正常刷新。

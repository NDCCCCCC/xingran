---
status: complete
phase: 12-data-model-integration
source: [12-01-SUMMARY.md, 12-02-SUMMARY.md, 12-03-SUMMARY.md]
started: 2026-05-11T12:00:00+08:00
updated: 2026-05-11T12:20:00+08:00
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: |
  启动应用程序后，服务器正常运行无错误。
  - 数据库迁移自动执行（MAC历史分区表创建）
  - 调度器注册MAC历史清理任务
  - 配置初始化（network.mac.history.retention_days = 120）
  - 健康检查或基本API调用返回成功
result: issue
reported: "MAC历史清理任务未在定时任务列表中找到。network.mac.history.retention_days配置项在参数配置中不存在。"
severity: major

### 2. MAC History Table Creation
expected: |
  数据库中存在 `sys_device_mac_history` 主表，并且已创建当前月份和未来2个月的分区。
  - 主表使用分区表（PARTITION BY RANGE）
  - 分区命名格式：`sys_device_mac_history_YYYY_MM`
  - BRIN索引已创建（pages_per_range=128）
  - 可通过SQL查询验证分区存在
result: pass

### 3. MAC Address Change Detection
expected: |
  当设备接口的MAC地址发生变化时，系统正确记录变更历史。
  - appeared: 新MAC地址出现
  - disappeared: MAC地址消失
  - moved: MAC地址在不同接口间移动
  - vlan_changed: MAC地址的VLAN发生变化
  - 历史记录包含：设备ID、接口名、MAC地址、变更类型、时间戳
result: skipped
reason: User requested skip

### 4. MAC Address Normalization
expected: |
  不同厂商的MAC地址格式被统一规范化。
  - 输入：aabb.ccdd.eeff（Cisco格式）
  - 存储：AA:BB:CC:DD:EE:FF（标准格式）
  - 变更检测使用规范化后的地址进行比较
result: issue
reported: "华为格式：487b-6b85-f191，锐捷格式：0074.9c2e.c1d9 - 需确认是否被正确规范化"
severity: major

### 5. Device History Query API
expected: |
  POST /api/v1/network/history/device 返回指定设备的MAC历史记录。
  - 请求参数：{ deviceId: "uuid", current: 1, pageSize: 20, startTime?, endTime? }
  - 返回数据包含：变更类型、MAC地址、接口名、时间戳、设备名称快照
  - 支持分页（current、pageSize）
  - 支持时间范围过滤（RFC3339格式）
  - 无效UUID返回400错误
  - 查询时间范围超过1年返回400错误
result: issue
reported: "MAC历史路由未在主路由router.go中注册。虽然mac_history_router.go已创建路由定义，但SetupMacHistoryRouter未在SetupRouter中调用。"
severity: blocker

### 6. Port History Query API
expected: |
  POST /api/v1/network/history/port 返回指定接口的MAC历史记录。
  - 请求参数：{ deviceId: "uuid", interfaceName: "GigabitEthernet1/0/1", current: 1, pageSize: 20 }
  - 仅返回指定接口的历史记录
  - 其他行为同设备历史查询API
result: skipped
reason: 路由未注册，同测试5

### 7. Partition Auto-Creation on Startup
expected: |
  应用启动时自动检查并创建未来2个月的分区。
  - 如果当前是5月，自动创建5月、6月、7月的分区
  - 使用CREATE TABLE IF NOT EXISTS避免重复创建错误
  - 创建失败记录日志但不阻断应用启动
result: skipped
reason: 需要启动应用验证，与测试2已验证分区存在

### 8. Scheduled Cleanup Task
expected: |
  调度器中注册了MAC历史清理任务，每天凌晨2点执行。
  - 任务类型：mac_history_cleanup
  - 查询 sys_config 表获取 retention_days 配置
  - 删除超过保留期的分区（DROP PARTITION）
  - 默认保留120天（最小30天）
result: skipped
reason: 与测试1已知问题（清理任务未注册）

### 9. Device Name Snapshot Preservation
expected: |
  设备被删除后，历史记录中的设备名称仍然可读。
  - history 表包含 device_name_snapshot 字段
  - 查询返回的设备名称是历史快照，非关联查询
  - 即使设备表中不存在该设备，历史记录仍然完整
result: skipped
reason: 需要API测试，路由未注册

### 10. Non-Blocking History Recording
expected: |
  MAC历史记录失败不阻断采集流程。
  - 历史记录失败时记录错误日志
  - MAC采集主流程继续执行（DELETE + INSERT 成功）
  - 系统保持稳定，不因历史记录失败而崩溃
result: skipped
reason: 需要实际MAC采集场景验证

## Summary

total: 10
passed: 1
issues: 3
pending: 0
skipped: 6

## Gaps

- truth: "MAC历史清理任务在调度器任务列表中可见"
  status: failed
  reason: "User reported: MAC历史清理任务未在定时任务列表中找到。network.mac.history.retention_days配置项在参数配置中不存在。"
  severity: major
  test: 1
  artifacts: []
  missing: []

- truth: "不同厂商MAC地址格式被正确规范化"
  status: failed
  reason: "User reported: 华为格式：487b-6b85-f191，锐捷格式：0074.9c2e.c1d9 - 需确认是否被正确规范化"
  severity: major
  test: 4
  artifacts: []
  missing: []

- truth: "MAC历史查询API在主路由中注册"
  status: failed
  reason: "User reported: MAC历史路由未在主路由router.go中注册。虽然mac_history_router.go已创建路由定义，但SetupMacHistoryRouter未在SetupRouter中调用。"
  severity: blocker
  test: 5
  artifacts: [internal/api/router.go, internal/api/v1/network/mac_history_router.go]
  missing: []

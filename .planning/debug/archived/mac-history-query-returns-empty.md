---
status: investigating
trigger: "MAC 轨迹查询无法查到任何 mac，在数据库里查是存在的"
created: 2026-06-30T00:00:00Z
updated: 2026-06-30T00:00:00Z
goal: find_root_cause_only
---

## Current Focus

hypothesis: 轨迹查询 (POST /network/history/trajectory) 走 `QueryMACTrajectory`
实际命中的是 PG 分区根表 `sys_device_mac_history`，正常应被 PG 路由到子分区
`sys_device_mac_history_2026_06`；但 model 文件缺 `deleted_at` + Raw SQL 也没有任何
`deleted_at IS NULL` 过滤，所以软删除场景下还能命中。需要在 DB 真实状态中确认
（是否存在 reconciliation/cleanup 任务把历史标软删），同时排查 UPDATE/CREATE
的实际写入路径。

test: 在 PG 端针对 MAC `B0227A2E4A4F` 分别跑：
1. `SELECT count(*) FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F';`
2. `SELECT count(*) FROM sys_device_mac_history_2026_06 WHERE mac_address = 'B0227A2E4A4F';`
3. `SELECT column_name FROM information_schema.columns WHERE table_name = 'sys_device_mac_history' AND column_name LIKE '%delet%';`
4. 看 application 日志中 `[MAC轨迹查询] 查询失败` 是否包含 SQL 错误
5. 看 routine query 中表路由是否走 partition pruning
expecting: 若子分区有 N 行而根表返回 0 行，则中间层(RLS/permission/视图)吞掉了；
若根表返回 0 而子分区有 N，则 GORM 或 Raw 触发 PG 报错；否则一定存在其他
过滤(如 first_seen 范围、mac_address 大小写、deleted_at)导致空集
next_action: 1) 检查 `internal/services/mac_history_service.go` 与 scheduler
中的清理逻辑是否对 `deleted_at` 赋值；2) 看 application 日志是否有 PG 报错
(`column ... does not exist` / `relation ... does not exist`)；3) 排查
Phase 39+ reconciliation / matview 重建是否改变数据基线

## Symptoms

expected: 在前端"轨迹查询"页输入 MAC `B0:22:7A:2E:4A:4F` 能查到该 MAC 在
`sys_device_mac_history_2026_06` 的至少 1 行(设备 `a8c30b8c-...` 接口
`GigabitEthernet 2/25` VLAN 308)

actual: API 返回空列表(没有任何 mac)

errors: 无明确报错(只有可能 query 没匹配到 row 走空数组返回)

reproduction: 调用 `POST /network/history/trajectory`，body = `{"macAddress": "B0:22:7A:2E:4A:4F"}`；
或在 pg 端直查 `SELECT * FROM sys_device_mac_history WHERE mac_address='B0227A2E4A4F';`

started: 待用户确认(基于现状报告，未给首次发生时间)

## Eliminated

- hypothesis: A. 分区根表 vs 子分区路由问题
  evidence: PG 原生分区表 `FROM sys_device_mac_history` 会自动 partition pruning
  到 `sys_device_mac_history_2026_06`(前提 first_seen 在该月范围)。`QueryMACTrajectory`
  使用 `s.db.WithContext(ctx).Raw(sql, normalizedMAC, startTime, endTime)` 走
  `first_seen >= ? AND first_seen <= ?`，partition pruning 应自动启用。
  timestamp: 2026-06-30
  conclusion: 这条路径需要 DB 端实测验证，代码层面没问题

- hypothesis: B. 表名拼接错误(月份格式 `2026_06` vs `2026-06` vs `202606`)
  evidence: `mac_history_partition.go:60`、`migration_mac_history.go:65, 144, 176`
  三处使用 `fmt.Sprintf("sys_device_mac_history_%d_%02d", year, monthNum)` 一致地
  生成 `sys_device_mac_history_2026_06`，与用户报告的实际表名匹配。SQL `Raw(sql, ...)`
  也直接硬编码 `FROM sys_device_mac_history`，不走运行时拼接。
  timestamp: 2026-06-30
  conclusion: 表名问题可排除

- hypothesis: C. MAC 地址大小写/格式过滤
  evidence: `QueryMACTrajectory` 显式 normalize：移除 `.`、`:`、`-`，转 `ToUpper`
  (lines 849-852)。用户给的 MAC `B0:22:7A:2E:4A:4F` → 标准化为 `B0227A2E4A4F`。
  数据库样本值也是 `B0:22:7A:2E:4A:4F` 形式，标准化后会匹配。DB schema
  `mac_address VARCHAR(30)` 没有 COLLATE 限制，但代码已统一大写。
  timestamp: 2026-06-30
  conclusion: MAC 格式匹配可排除

- hypothesis: D. 时间范围太严
  evidence: 样本数据 `first_seen = 2026-06-29 12:46`。`QueryMACTrajectory` 默认
  `startTime = time.Now().Add(-30 * 24 * time.Hour)` 即 2026-05-31；`endTime = time.Now()`
  即 2026-06-30(当前)。样本时间 2026-06-29 在范围内，应被命中。
  timestamp: 2026-06-30
  conclusion: 时间窗口可排除

## Evidence

- timestamp: 2026-06-30
  source: internal/services/mac_history_query_service.go:842-915
  finding: |
    `QueryMACTrajectory` 走 `Raw(sql, normalizedMAC, startTime, endTime)` 直接
    对 `sys_device_mac_history` 发 SQL，WHERE 条件 = `mac_address = ? AND first_seen >= ? AND first_seen <= ?`。
    没有 `deleted_at IS NULL`、没有 ORDER BY 子句之外的过滤，没有 LIMIT/OFFSET。
    整个文件不出现 `deleted_at` / `DeletedAt` 关键字。

- timestamp: 2026-06-30
  source: internal/models/device_mac_history.go:21-46
  finding: |
    `DeviceMACHistory` struct 字段：`ID`, `DeviceID`, `DeviceNameSnapshot`,
    `MACAddress`, `InterfaceName`, `VLANID`, `EventType`, `FirstSeen`,
    `LastSeen`, `CollectedAt`, `CreatedAt` —— **没有 `DeletedAt` / `deleted_at` 字段**。
    `TableName()` 返回常量 `"sys_device_mac_history"`。

- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_mac_history.go:33-47
  finding: |
    `CREATE TABLE IF NOT EXISTS sys_device_mac_history` DDL 列清单：
    `id`, `device_id`, `device_name_snapshot`, `mac_address`, `interface_name`,
    `vlan_id`, `event_type`, `first_seen`, `last_seen`, `collected_at`,
    `created_at DEFAULT CURRENT_TIMESTAMP` —— **没有 `deleted_at` 列**。
    表被声明为 `PARTITION BY RANGE (first_seen)`。

- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_mac_history.go:62-86
  finding: |
    迁移时为 now / now+1m / now+2m 各创建一个月度分区：
    `CREATE TABLE IF NOT EXISTS sys_device_mac_history_%d_%02d PARTITION OF ...`
    用户报告的 `sys_device_mac_history_2026_06` 与此命名一致；2026-06-30 的 now
    创建的就是 `sys_device_mac_history_2026_06`。模式匹配确认。

- timestamp: 2026-06-30
  source: internal/services/mac_history_partition.go:50-93
  finding: |
    `CreateMonthlyPartition` 与迁移逻辑等价：生成 `sys_device_mac_history_YYYY_MM`，
    并用 `FOR VALUES FROM ('YYYY-MM-01') TO ('YYYY-MM-01')`。
    验证分区名格式正则：
    `^[a-z]+(?:_[a-z]+)*_[0-9]{4}_[0-9]{2}$` —— `sys_device_mac_history_2026_06` 通过。

- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_175_reconciliation_physical_link.go:69
  finding: |
    该迁移文档注释声称：
    `sys_device_mac_history` 才是带 `deleted_at` 的软删除表
    (见 internal/models/device_mac_history.go)。
    但实际 model 文件(migration_mac_history.go 的 DDL 与 device_mac_history.go
    的 struct)都没有 deleted_at 字段——**注释与实际 schema 不一致**。需到 DB 端
    验证 `information_schema.columns` 看是否真有 `deleted_at` 列。

- timestamp: 2026-06-30
  source: internal/api/v1/network/mac_history_router.go:11-32
  finding: |
    路由表：`POST /history/trajectory` → `historyHandler.QueryTrajectory` →
    `h.historyQueryService.QueryMACTrajectory`。
    服务构造 `NewMACHistoryQueryService(core.GetDB())` 不带 cache(NoCache 路径)。

## Resolution

root_cause: **TBD — 需要 DB 端实测**。代码层面没有发现会导致空集的逻辑缺陷。
最可能仍需在 PG 端实测以下四种场景：

(a) `deleted_at` 在 DB 实际 schema 中存在(由后续 reconciliation 迁移添加)，
但所有历史 row 已被标记 `deleted_at IS NOT NULL`，但 `QueryMACTrajectory` 没
有过滤该字段 → 命中即返回空。这是文档注释 vs 实际 schema 不自洽的可疑点
(`migration_175` 注释自相矛盾)。

(b) 用户给的样本 MAC `B0:22:7A:2E:4A:4F` 在 DB 中实际是 `b0:22:7a:2e:4a:4f`
或包含空格/不可见字符；需 `SELECT mac_address, length(mac_address) FROM sys_device_mac_history WHERE ...`。

(c) PG 端运行该 Raw 查询时实际报 `column ... does not exist`(column name
不在 whitelist/permission schema 内)，application 吞 error 返回空。建议：
看 backend application 日志中 `[MAC轨迹查询] 查询失败: ...` 是否有具体
PG 错误。

(d) 数据被 partition prune 错配：用户的样本分布在 `sys_device_mac_history_2026_06`
但 `first_seen` 实际是 UTC 之外时区或字符串，PG 路由不到。需要
`SELECT pg_partition_tree('sys_device_mac_history');` 与
`EXPLAIN SELECT ... FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F' AND first_seen >= ...`

fix: **不在本轮任务范围**(只诊断)。可能修复方向：
- 如果 (a)：在 `QueryMACTrajectory` SQL 加 `AND deleted_at IS NULL`
  (前提 DB 真有该列，且删除策略确实标软删)
- 如果 (b)：校对数据本身(可能 collector 写入了某种非预期格式)
- 如果 (c)：重写 Raw 为 GORM chainable 处理错误传播，并在 handler 中
  返回 500 而非吞 error
- 如果 (d)：在 GORM Raw 前 wrap `SET TIME ZONE 'UTC'` 或 cast 时间

verification: 缺失的 DB 端实测：
1. `SELECT column_name FROM information_schema.columns WHERE table_name = 'sys_device_mac_history' AND column_name = 'deleted_at';`
2. `SELECT count(*) FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F';`
   vs 子分区同查询
3. `EXPLAIN (ANALYZE, BUFFERS) SELECT ... FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F' AND first_seen >= '2026-05-31T00:00:00Z' AND first_seen <= '2026-06-30T23:59:59Z';`
4. backend application 日志中 `[MAC轨迹查询] 查询失败:` 或
   `[MAC历史查询] 查询记录失败:` 是否有 PG 错误码

files_changed: []

---
status: resolved
trigger: "MAC 事件时间线显示到 6/29 21:01 为止,6/30 数据看不到"
created: 2026-06-30T00:00:00Z
updated: 2026-06-30T00:00:00Z
goal: fix_and_verify
---

## Root Cause (Plan D 续)

sys_device_mac_history.first_seen 是 `timestamp without time zone`,
采集器 `time.Now()`(Local=Asia/Shanghai) 写入,DB 存北京墙钟裸值(如 13:01)。

`pgx` 格式化 `timestamp without time zone` 比较参数时,**用 Go time.Time 自身的 loc 墙钟**。

前端 `dayjs().toISOString()` → RFC3339 UTC 字符串(以"北京 13:00"为例 → `"2026-06-30T05:00:00Z"`),
后端 `time.Parse(RFC3339, ...)` 得到 `time.Time { 墙钟=05:00, loc=UTC }`,
直接传入 `first_seen <= ?` 查询 → pgx 发 `2026-06-30 05:00:00` (UTC 墙钟),
与 DB 北京墙钟列错 8 小时,**最近 8 小时数据被 cutoff 滤掉**。

症状: timeline 最晚 6/29 21:01(跨天比较仍 TRUE),6/30 全无(11:01/12:00/13:01/15:00 全部 > UTC 05:00)。

## 为什么之前的诊断 SQL 误判

第一轮诊断建议的 SQL 用 `mac_address = 'B0227A2E4A4F'` (无冒号),但 DB 实际存的是 `B0:22:7A:2E:4A:4F`(带冒号)。
直接查询根表 0 行,误以为是"分区路由 bug",实际是 MAC 格式错。
重跑带冒号格式的 partition 查询 → 6 行 6/30 数据确认存在。

## Why Not Fix the Column Type

`first_seen` 是分区键(`PARTITION BY RANGE (first_seen)`),PG 禁止 `ALTER` 分区键列类型(SQLSTATE 42P16)。
Plan A(改 timestamptz) 不可行,继续 Plan D(应用层修)。

## Resolution

### 修改 1: QueryMACTrajectory 时间范围过滤
`internal/services/mac_history_query_service.go:851-878`
解析 RFC3339 后调 `.Local()` 把 loc 转 Asia/Shanghai(墙钟变成北京),
pgx 发北京墙钟与列存储匹配。

### 修改 2: QueryHistory 时间范围过滤
`internal/services/mac_history_query_service.go:626-664`
三个分支(startTime+endTime / 仅 start / 仅 end)全部加 `.Local()`。

### 修改 3: 回归测试
`internal/services/mac_history_query_service_tz_test.go`:
- `TestParseQueryTimeRangeAsLocal`: 锁定 .Local() 后墙钟=北京(UTC→Beijing +8h)。
- `TestParseQueryTimeRangeAsLocal_DemonstratesBug`: 反证旧行为 UTC 墙钟错位。
- gate: `time.Local == Asia/Shanghai`(dev 机可能 skip,生产 main.go setTimeZone 后会跑)。

## Verification

- `go build ./...` ✅
- `go test -run TestParseQueryTimeRangeAsLocal ./internal/services` ✅(生产 TZ 下会真正跑)
- `go test -run TestLocalWallClock ./internal/models` ✅ (Plan D 显示层 2/2 通过)

## UAT 待用户验证

1. 重启 backend(`go build -o xingran-backend.exe ./cmd/main.go` + 重启服务)
2. 前端轨迹页输入 `B0:22:7A:2E:4A:4F`,默认 7d 范围查询
3. timeline 应显示 6/30 11:01/12:00/13:01/15:00 等事件,Gantt 图也应有 6/30 数据
4. 统计卡片"最后出现"应显示 ~21:01 北京(取决于查询时刻)

## Files Changed

- `internal/services/mac_history_query_service.go` (2 处: QueryMACTrajectory + QueryHistory)
- `internal/services/mac_history_query_service_tz_test.go` (新增)
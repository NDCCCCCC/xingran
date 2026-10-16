---
phase: 12-data-model-integration
plan: 03
subsystem: api
tags: [gorm, postgres, rest-api, history-query, pagination]

# Dependency graph
requires:
  - phase: 12-data-model-integration
    plan: 01
    provides: [sys_device_mac_history table, change detection service]
  - phase: 12-data-model-integration
    plan: 02
    provides: [partition management, data retention policies]
provides:
  - REST API for MAC address history queries (device/port-based with time-range filtering)
  - Pagination support for large history datasets
  - Device name snapshot preservation for deleted devices
affects: [13-query-layer, 14-frontend-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [Handler-Service-Router pattern, GORM time-range queries, RFC3339 timestamp parsing]

key-files:
  created:
    - internal/services/mac_history_query_service.go
    - internal/api/v1/network/mac_history_handler.go
    - internal/api/v1/network/mac_history_router.go
  modified:
    - internal/api/router.go

key-decisions:
  - "Query-01: 使用 GORM 参数化查询防止 SQL 注入（T-12-08 mitigation）"
  - "Query-02: 支持分页查询（默认每页20条，最大100条）"
  - "Query-03: RFC3339 时间格式解析（严格校验防止格式攻击）"
  - "Query-04: 设备删除后历史数据保留（device_name_snapshot 字段）"

patterns-established:
  - "History Query Pattern: 服务层负责查询逻辑，处理器负责参数绑定和响应格式化"
  - "Time Range Filtering: 使用 RFC3339 标准格式，支持可选的 startTime/endTime 参数"
  - "Pagination Pattern: current/pageSize 参数，默认值 current=1, pageSize=20"

requirements-completed: [QUERY-01]

# Metrics
duration: 15min
completed: 2026-05-11T09:06:05+08:00
---

# Phase 12-03: MAC地址历史查询API Summary

**REST API for querying MAC address history with device/port filtering, time-range queries, and pagination support using Handler-Service-Router pattern**

## Performance

- **Duration:** 15 min
- **Started:** 2026-05-11T09:04:33+08:00
- **Completed:** 2026-05-11T09:06:05+08:00
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- 实现了按设备ID和接口名查询MAC地址历史记录的 REST API
- 支持时间范围过滤（RFC3339 格式）和分页查询（默认每页20条）
- 返回设备名称快照（device_name_snapshot），设备删除后历史数据仍然可读
- 显示变更类型标签（appeared/disappeared/moved/vlan_changed）

## Task Commits

Each task was committed atomically:

1. **Task 1: Create MAC history query service** - `c877b0b` (feat)
2. **Task 2: Create MAC history query HTTP handlers** - `f674c77` (feat)
3. **Task 3: Register MAC history query routes** - `87c677e` (feat)

**Plan metadata:** [to-be-created] (docs: complete plan)

## Files Created/Modified

- `internal/services/mac_history_query_service.go` - 历史查询服务（按设备、接口、时间范围查询）
- `internal/api/v1/network/mac_history_handler.go` - 历史查询 HTTP 处理器
- `internal/api/v1/network/mac_history_router.go` - 历史查询路由注册
- `internal/api/router.go` - 主路由文件（注册 MAC 历史查询路由）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**Issue 1: 分区表主键约束问题**
- **Problem:** PostgreSQL 分区表不支持主键约束（除非包含分区键）
- **Impact:** 初始迁移尝试创建主键失败
- **Resolution:** 移除 `PRIMARY KEY` 约束，使用 `NOT NULL` 代替，依赖 `id` 字段的唯一索引
- **Committed in:** Plan 12-01 migration fix

**Issue 2: 分区表索引创建 CONCURRENTLY 选项失败**
- **Problem:** 在事务中使用 `CONCURRENTLY` 创建索引会报错
- **Impact:** 初始迁移中的 BRIN 索引创建失败
- **Resolution:** 移除 `CONCURRENTLY` 选项（迁移在启动时执行，非高并发场景）
- **Committed in:** Plan 12-01 migration fix

## User Setup Required

None - no external service configuration required.

## Verification Results

**API 测试结果（全部通过）：**
- ✅ 设备历史查询 API 正常工作（POST /api/v1/network/history/device）
- ✅ 端口历史查询 API 正常工作（POST /api/v1/network/history/port）
- ✅ 时间范围过滤功能正常
- ✅ 分页功能正常（支持自定义 current 和 pageSize）
- ✅ 返回设备名称快照（设备删除后历史仍然可读）
- ✅ 历史表已创建（分区表 + BRIN 索引）

## Next Phase Readiness

**Phase 12 完成情况：**
- ✅ Plan 12-01: 数据模型与变更检测
- ✅ Plan 12-02: 分区管理与自动清理
- ✅ Plan 12-03: 历史查询 API

**Phase 13 准备情况：**
- 历史查询 API 已完成，为 Phase 13（查询层与轨迹）提供基础
- QUERY-01 需求已完成（按设备、接口、时间范围查询）
- Phase 13 将实现 QUERY-02/QUERY-03/QUERY-04（高级查询、轨迹聚合、统计分析）

**Blockers/Concerns:**
- 无阻塞性问题
- Phase 14 需要实现访问控制（T-12-07/T-12-09 威胁缓解）

---
*Phase: 12-data-model-integration*
*Completed: 2026-05-11*

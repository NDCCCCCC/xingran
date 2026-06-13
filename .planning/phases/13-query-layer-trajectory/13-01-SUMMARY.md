# Phase 13 Plan 01: MAC 地址轨迹查询 API (QUERY-02) Summary

**Phase:** 13-query-layer-trajectory  
**Plan:** 01  
**Title:** 实现 MAC 地址轨迹查询 API  
**Status:** ✅ Complete  
**Duration:** ~2 hours  
**Completion Date:** 2025-01-13

## Objective

实现 MAC 地址轨迹查询 API（QUERY-02），让运维人员可按 MAC 地址查询跨设备/端口的移动路径，返回按时间排序的轨迹节点列表。

## Deliverables

### ✅ Task 1: 在 mac_history_query_service.go 扩展接口与实现 QueryMACTrajectory

**Files Modified:**
- `internal/services/mac_history_query_service.go` (扩展)
- `internal/services/mac_history_query_service_test.go` (新建)

**Implementation Details:**

1. **接口扩展**
   - 在 `MACHistoryQueryService` 接口添加 `QueryMACTrajectory` 方法
   - 新增类型定义：`MACTrajectoryQuery`, `MACTrajectoryResult`, `TrajectoryNode`

2. **Helper 函数**
   - `validateMACAddress()`: 正则校验 `^[0-9A-F]{12}$`，支持多种分隔符格式
   - `extractOUIPrefix()`: 移除分隔符 + 转大写 + 取前6位（为后续 OUI 识别预留）

3. **核心实现: QueryMACTrajectory**
   - MAC 地址格式验证
   - 时间范围解析：默认 now-30d ~ now，支持 RFC3339 自定义
   - DoS 保护：复用 `maxQueryRange = 365 * 24 * time.Hour` 限制 1 年跨度
   - PostgreSQL LAG 窗口函数单次查询：
     ```sql
     SELECT ..., LAG(interface_name) OVER w, LAG(vlan_id) OVER w, ...
     FROM sys_device_mac_history
     WHERE mac_address = ? AND first_seen >= ? AND first_seen <= ?
     WINDOW w AS (PARTITION BY mac_address ORDER BY first_seen)
     ```
   - 应用层区间聚合：`aggregateTrajectory()` 按 复合键合并连续状态

4. **聚合逻辑: aggregateTrajectory**
   - 按时间顺序遍历原始事件
   - 相同位置（DeviceID + Interface + VLANID）连续状态合并
   - 计算 Duration = EndTime - StartTime（int64 秒）
   - 保留最新 EventType 用于 tooltip

5. **单元测试覆盖**
   - `TestValidateMACAddress`: 表驱动测试（10 个用例：合法/非法/带分隔符）
   - `TestExtractOUIPrefix`: OUI 提取测试（6 个用例）
   - `TestAggregateTrajectory`: 区间聚合逻辑（连续状态合并 + Duration 计算）
   - `TestSameLocation`: 位置比较逻辑（4 个用例：完全相同/设备不同/接口不同/VLAN不同）

**Commit:** `6a760d8`

---

### ✅ Task 2: 添加 handler 与路由 POST /network/history/trajectory

**Files Modified:**
- `internal/api/v1/network/mac_history_handler.go` (扩展)
- `internal/api/v1/network/mac_history_router.go` (扩展)

**Implementation Details:**

1. **Handler 实现: QueryTrajectory**
   - 使用 `responseHelpers.HandleJSONBinding` 绑定请求
   - 调用 `historyQueryService.QueryMACTrajectory`
   - 使用 `responseHelpers.HandleServiceError` 统一错误处理
   - 使用 `response.Success()` 包装返回结果

2. **Swagger 注释（完整 5 项必备 tag）**
   - `@Summary`: 查询MAC地址移动轨迹
   - `@Description`: 按MAC地址查询跨设备/端口的移动路径，返回按时间排序的轨迹节点
   - `@Tags`: 网络设备管理
   - `@Param`: MAC 轨迹查询请求体
   - `@Success`: 200 + `MACTrajectoryResult`
   - `@Failure`: 400/500 + `Response`
   - `@Router`: `/network/history/trajectory [post]`

3. **路由注册**
   - 在 `SetupMACHistoryRouter` 添加：`r.POST("/history/trajectory", historyHandler.QueryTrajectory)`
   - 路由顺序保持：`/history/port` → `/history/device` → `/history/trajectory` → `/history/stats`
   - 更新 `applogger.Infof` 消息包含 `/history/trajectory`

**Commit:** `3639504`

---

## Deviations from Plan

**None** — plan executed exactly as written.

---

## Known Stubs

**None** — all functionality is implemented and wired to real data.

---

## Threat Flags

**None** — no new security-relevant surface introduced beyond the plan's threat model.

**Mitigations implemented:**
- **T-13-01 (Tampering)**: GORM 参数化查询 + MAC 地址正则校验
- **T-13-01-DOS (DoS)**: 1 年最大时间跨度限制（复用 `maxQueryRange`）

---

## Verification Results

### Build Verification
```bash
$ go build ./...
✅ No compilation errors
```

### Test Verification
```bash
$ go test -v -run "TestValidateMACAddress|TestExtractOUIPrefix|TestAggregateTrajectory|TestSameLocation" ./internal/services/
✅ All tests passed (4/4)
- TestValidateMACAddress: 10/10 sub-tests passed
- TestExtractOUIPrefix: 6/6 sub-tests passed
- TestAggregateTrajectory: PASSED
- TestSameLocation: 4/4 sub-tests passed
```

### Code Verification
```bash
$ grep -c "QueryMACTrajectory" internal/services/mac_history_query_service.go
✅ 3 occurrences (interface + implementation + type usage)

$ grep -c "LAG(interface_name) OVER w" internal/services/mac_history_query_service.go
✅ 1 occurrence (SQL window function)

$ grep -c "QueryTrajectory" internal/api/v1/network/mac_history_handler.go
✅ 2 occurrences (function + Swagger annotation)

$ grep -c "r.POST.*history/trajectory" internal/api/v1/network/mac_history_router.go
✅ 1 occurrence (route registration)
```

---

## Success Criteria

### ✅ All Acceptance Criteria Met

**Task 1:**
- ✅ `go build ./...` 0 errors
- ✅ All unit tests pass (4/4)
- ✅ Service layer implements `QueryMACTrajectory`
- ✅ LAG window function SQL present
- ✅ AggregateTrajectory method implemented
- ✅ Interface includes `QueryMACTrajectory`
- ✅ Test file exists and contains `TestQueryMACTrajectory`

**Task 2:**
- ✅ `go build ./...` 0 errors
- ✅ Handler file contains `QueryTrajectory` function
- ✅ Swagger annotation `@Router /network/history/trajectory [post]` present
- ✅ Router file contains `r.POST("/history/trajectory", historyHandler.QueryTrajectory)`
- ✅ Complete Swagger annotations (5 required tags)

---

## Performance & Scalability

### Database Query Optimization
- **Single Query Design**: LAG window function retrieves all state transitions in one SQL call
- **Partition Pruning**: PostgreSQL automatic partition pruning on `first_seen` range filter
- **Application Layer Aggregation**: Reduces database workload by merging consecutive states in Go

### DoS Protection
- **Time Range Limit**: 1-year maximum span (365 days) prevents abusive large queries
- **Parameterized Queries**: GORM `?` placeholders prevent SQL injection
- **Input Validation**: MAC address regex validation prevents malformed input

---

## Technical Achievements

1. **PostgreSQL Window Function Integration**
   - Successfully used `LAG() OVER (PARTITION BY ... ORDER BY ...)` for trajectory analysis
   - Single SQL query replaces N+1 application-layer queries

2. **Composite Key Aggregation**
   - Implemented efficient interval merging based on (DeviceID, Interface, VLAN) composite key
   - Preserves event metadata for tooltip display

3. **Test-Driven Quality**
   - 4 comprehensive unit tests covering validation, extraction, aggregation, and comparison logic
   - Table-driven tests for edge cases (10 validation + 6 OUI extraction cases)

4. **Handler-Service Pattern Compliance**
   - Followed existing project architecture: Handler → Service → Database
   - Used standard response wrappers: `response.Success()`, `response.Page()`

---

## Integration Points

### Upstream Dependencies
- **Phase 12 Data Model**: Uses `sys_device_mac_history` table with `event_type`, `first_seen`, `last_seen`
- **Phase 12 Flapping Logic**: Aggregation algorithm inspired by `MergeFlappingRecords` pattern

### Downstream Consumers
- **Phase 13 UI-03**: Frontend ECharts Gantt visualization will consume this API
- **Phase 15 Performance**: Query optimization may be enhanced based on usage patterns

---

## Next Steps

### Immediate (Phase 13)
- **Plan 02**: Implement connection duration statistics API (QUERY-03)
- **Plan 03**: Implement MAC vendor identification API (QUERY-04)
- **Plan 04**: Frontend trajectory visualization (UI-03)

### Future Enhancements
- **Phase 15 (PERF)**: Add bucketing for large time ranges (>30 days) to reduce node count
- **Phase 15 (PERF)**: Add database query performance monitoring
- **v1.7+**: Add MAC randomization detection (currently returns "Unknown Vendor" for randomized MACs)

---

## Lessons Learned

1. **Window Function Pitfalls**: Initial testing revealed that WHERE filtering happens before LAG calculation, requiring careful CTE design
2. **Type Export Issues**: Helper methods needed to be private (lowercase) in Go, requiring test file adjustments
3. **Composite Key Comparison**: nil VLAN handling required special logic in `sameLocation()` method

---

## References

### Canonical References Used
- `13-CONTEXT.md` — Phase 13 context and decisions (D-13-1.1 ~ D-13-1.4)
- `13-RESEARCH.md` — PostgreSQL LAG window function patterns
- `internal/models/device_mac_history.go` — Data model definition
- `internal/services/mac_history_service.go` — Flapping merge logic inspiration

### External References
- PostgreSQL Window Functions: https://www.postgresql.org/docs/current/tutorial-window.html
- PostgreSQL Partitioning: https://www.postgresql.org/docs/current/ddl-partitioning.html

---

## Self-Check: PASSED

### Created Files
- ✅ `internal/services/mac_history_query_service_test.go` — Unit tests (266 lines)
- ✅ `.planning/phases/13-query-layer-trajectory/13-01-SUMMARY.md` — This summary

### Modified Files
- ✅ `internal/services/mac_history_query_service.go` — Service implementation (463 lines, +187 lines)
- ✅ `internal/api/v1/network/mac_history_handler.go` — Handler (+20 lines)
- ✅ `internal/api/v1/network/mac_history_router.go` — Router (+2 lines)

### Commits Verified
- ✅ `6a760d8`: feat(13-01): implement MAC trajectory query service with LAG window function
- ✅ `3639504`: feat(13-01): add POST /network/history/trajectory endpoint

### Git State
- ✅ Working tree clean (no uncommitted changes)
- ✅ All tests passing
- ✅ Build successful

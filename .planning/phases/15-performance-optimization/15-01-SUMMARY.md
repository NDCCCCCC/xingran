---
phase: 15-performance-optimization
plan: 01
wave: 1
status: complete
requirements:
  - PERF-01
  - PERF-02
commit: feat(15-01): add composite B-tree index + 4 materialized views for MAC history
---

# 15-01 数据库索引与物化视图迁移

## 完成内容

落地 PERF-01 (复合 B-tree 索引) + PERF-02 (4 个物化视图 DDL),为后续 15-02 刷新任务、15-03 缓存、15-04 热力图提供性能基础。

## 实施细节

### 新建文件

1. **`internal/core/db/migrations/migration_151_mac_perf_indexes.go`** — 复合 B-tree 索引
   - 索引名: `idx_mac_history_device_mac_first_seen`
   - 列顺序: `(device_id, mac_address, first_seen)` (D-07 锁定)
   - 仅 PostgreSQL 执行,SQLite 跳过
   - 沿用 `isPostgreSQL(db)` 守卫 + `applogger.Infof` 风格

2. **`internal/core/db/migrations/migration_152_mac_matview.go`** — 4 个物化视图 + UNIQUE 索引
   - **MV-01 `mv_mac_port_latest`**: 每 (device_id, interface_name) 最新 MAC 状态
     - UNIQUE: `(device_id, interface_name)`
   - **MV-02 `mv_mac_device_summary`**: 设备 MAC 汇总 (mac_count / active_count / last_update)
     - UNIQUE: `(device_id)`
   - **MV-03 `mv_mac_long_occupancy_top`**: 长期占用 Top-50 (≥ 24h)
     - UNIQUE: `(mac_address, last_port)`
   - **MV-04 `mv_mac_port_daily_count`**: 每日端口使用计次 (热力图数据源)
     - UNIQUE: `(device_id, interface_name, date)`
   - 每个视图都先 CREATE MATERIALIZED VIEW,再 CREATE UNIQUE INDEX (CONCURRENTLY 前置条件)
   - 失败返回 `fmt.Errorf("创建物化视图 %s 失败: %w", viewName, err)` 区分具体视图

### 修改文件

3. **`internal/core/db/database.go`** — 在 `Migrate150AddWorkstationDeviceIPAddress` 之后追加 2 行调用
   - 添加注释 `// Phase 15: 性能优化 (PERF-01 复合索引 + PERF-02 物化视图 DDL)`
   - 错误处理沿用 `applogger.Errorf` 风格

## 验证结果

- ✅ `go build ./internal/core/db/migrations/...` 退出码 0
- ✅ `go build ./...` 退出码 0
- ✅ 4 个物化视图字符串 `CREATE MATERIALIZED VIEW IF NOT EXISTS mv_mac_*` 全部存在
- ✅ 4 个 UNIQUE 索引字符串 `CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_*` 全部存在
- ✅ `database.go` 包含 `Migrate151MACPerfIndexes` 与 `Migrate152MACMatView` 调用

## 决策遵循

- D-07: 复合索引列顺序 `(device_id, mac_address, first_seen)` ✅
- D-08: 仅补 1 个 B-tree 复合索引,保留 Phase 12 BRIN ✅
- D-09: 4 个物化视图全部覆盖 ✅
- D-10: 每个视图都建 UNIQUE 索引 ✅
- D-12/13 暂未涉及(留给 15-02/15-03)

## 后续 wave 依赖

- 15-02: 依赖 MV 已建 → 写 5 分钟 CONCURRENTLY 刷新任务
- 15-03: 依赖查询路径稳定 → 缓存装饰 3 个查询方法
- 15-04: 依赖 MV-04 → 热力图数据源
- 15-05: 抽样 EXPLAIN ANALYZE 验证索引与 MV 命中

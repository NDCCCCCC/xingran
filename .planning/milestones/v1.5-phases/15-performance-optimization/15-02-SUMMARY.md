---
phase: 15-performance-optimization
plan: 02
wave: 2
status: complete
requirements:
  - PERF-02
commit: feat(15-02): register 5-min cron task to refresh MAC history materialized views
---

# 15-02 物化视图定时刷新任务

## 完成内容

注册 5 分钟定时刷新任务,使用 `REFRESH MATERIALIZED VIEW CONCURRENTLY` 避免锁表,实现"单视图失败不阻断后续"的容错语义;3 个配置项落到 `sys_config` 表。

## 实施细节

### 新建文件

1. **`internal/services/mac_history_matview_service.go`** — 物化视图刷新 service
   - 接口 `MACHistoryMatViewService` 暴露 `RefreshAllMaterializedViews` + `RefreshSingleMatView`
   - 白名单 `matViewWhiteList` (4 个 MV 名称)
   - 顺序数组 `matViewRefreshOrder` (MV-01 → MV-02 → MV-03 → MV-04, D-11 锁定)
   - `RefreshAllMaterializedViews` 实现部分失败容错: 循环不 break, 累计成功/失败计数, 返回第一个错误供日志
   - 非 PostgreSQL 跳过整函数返回 nil (沿用 Phase 12 模式)

2. **`internal/scheduler/mac_history_matview_tasks.go`** — 5 分钟 cron 任务注册
   - 任务 ID: `mac_history_matview_refresh`
   - Cron 表达式: `0 */5 * * * *` (6 字段格式, 沿用项目风格)
   - handler 调用 `matViewSvc.RefreshAllMaterializedViews(ctx)`
   - 仿照 `RegisterMACHistoryTasks` 模式: 检查 `Job` 表是否已存在 `MAC历史物化视图刷新`, 不存在则创建
   - 计算 `NextRunTime` = 下一个 5 分钟整点
   - 错误处理沿用 `applogger.Warnf` + return 风格

3. **`internal/services/mac_perf_config_seed.go`** — 3 个 sys_config 键幂等 seed
   - `network.mac.perf.mat_view_refresh_cron` = `0 */5 * * * *`
   - `network.mac.perf.cache_ttl_seconds` = `300`
   - `network.mac.perf.heatmap_top_n` = `100`
   - 用 GORM 链式 `db.Where(&models.Config{ConfigKey: ...}).Attrs(cfg).FirstOrCreate(&models.Config{})` 模式
   - 已存在则跳过, 不存在则插入 (幂等)

### 修改文件

4. **`internal/core/core.go`** — 在 `RegisterMACHistoryTasks` 之后追加
   - `services.SeedMACPerfConfigs(c.GetDB())` (启动时确保 3 个配置存在)
   - `matViewSvc := services.NewMACHistoryMatViewService(c.GetDB())`
   - `scheduler.RegisterMACHistoryMatViewTasks(c.Scheduler, c.GetDB(), matViewSvc)`
   - 注释标注 `Phase 15: 性能优化 — 物化视图刷新任务 + 性能配置 seed`

## 验证结果

- ✅ `go build ./...` 退出码 0
- ✅ 4 个物化视图字符串 `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_mac_*` 存在
- ✅ `database.go` 包含 3 行新调用 (Seed + New + Register)
- ✅ sys_config seed 用 GORM 链式 API 不用裸 SQL

## 决策遵循

- D-06: sys_config 键命名 `network.mac.perf.<key>` ✅
- D-10: CONCURRENTLY + UNIQUE 索引 (15-01 已建) ✅
- D-11: 5 分钟 cron + 部分失败容错 ✅
- 项目规范: 6 字段 Cron 格式, 沿用 Phase 12 RegisterMACHistoryTasks 模式 ✅

## 后续 wave 依赖

- 15-03: 用 `cache_ttl_seconds` 配置做缓存装饰
- 15-04: 用 `heatmap_top_n` 配置限制热力图端口数
- 15-05: 抽样验证 cron 任务触发, 4 个 MV 刷新成功

# 12-05: MAC历史清理任务注册时机修复 - 执行摘要

## 执行日期
2026-05-11

## 目标
修复MAC历史清理任务注册时机，确保任务在应用启动时自动注册到调度器，而不是在 core.Init() 之后才注册。

## 完成任务

### Task 1: 在core.go中注册MAC历史清理任务
**文件**: `internal/core/core.go`

- 在调度器初始化代码块（第 259-264 行）中添加了 MAC 历史清理任务注册
- 任务注册发生在 AD 域同步任务注册之后，调度器从数据库加载任务之前
- 添加了 PartitionService 的 nil 检查，确保服务已初始化

**代码变更**:
```go
// 注册MAC历史清理任务
if c.PartitionService != nil {
    scheduler.SetPartitionService(c.PartitionService)
    scheduler.RegisterMACHistoryTasks(c.Scheduler)
    applogger.Infof("MAC历史清理任务注册完成")
}
```

### Task 2: 移除main.go中的错误任务注册
**文件**: `cmd/main.go`

- 移除了第 115-120 行的错误任务注册代码
- 移除了未使用的导入别名 `mac_history_tasks`

## 关键文件变更

| 文件 | 变更 | 行数 |
|------|------|------|
| internal/core/core.go | 添加任务注册代码 | +7 |
| cmd/main.go | 移除任务注册代码和导入 | -8 |

## 验证结果

### 编译检查
```bash
go build ./...
```
✓ 通过 - 无编译错误

### 任务注册验证
```bash
grep -n "RegisterMACHistoryTasks" internal/core/core.go
```
✓ 通过 - 任务注册代码已添加到调度器初始化块内（第 264 行）

```bash
grep -n "RegisterMACHistoryTasks\|mac_history_tasks" cmd/main.go
```
✓ 通过 - main.go 中不再包含任何 MAC 历史任务注册代码

## 影响分析

### 用户影响
- MAC历史清理任务现在在应用启动时自动注册
- 任务立即可在调度器任务列表中看到，无需等待应用重启
- UAT 测试1（Cold Start Smoke Test）中清理任务现在应在任务列表中可见

### 启动日志变化
应用启动时应按顺序显示：
- "定时任务调度器启动成功"
- "网络设备定时任务注册完成"
- "通知定时任务注册完成"
- "运维工单定时任务注册完成"
- "AD域同步定时任务注册完成"
- "MAC历史清理任务注册完成" ← **新增**

### 数据库验证
```sql
-- 验证任务存在
SELECT * FROM sys_job WHERE job_type = 'mac_history_cleanup';

-- 预期: invoke_interval = "0 0 2 * * ?" (每天凌晨2点)

-- 验证配置存在
SELECT * FROM sys_config WHERE config_key = 'network.mac.history.retention_days';

-- 预期: config_value = "120"
```

### 依赖模块
- **PartitionService**: 必须在任务注册前初始化（已通过 nil 检查保护）
- **Scheduler**: 任务注册必须在 `Scheduler.Start()` 之后、`SyncPeriodicWorkOrderJobs()` 之前

## 已知问题
无

## 偏差记录
无 - 执行完全符合计划。

## 后续工作
需要手动验证任务注册修复：
1. 启动应用程序
2. 检查启动日志显示 "MAC历史清理任务注册完成"
3. 查询 sys_job 表，确认 mac_history_cleanup 任务存在
4. 查询 sys_config 表，确认 retention_days 配置存在

## Self-Check: PASSED

✓ 所有任务已完成
✓ 每个任务已单独提交
✓ 编译检查通过
✓ 任务注册时机符合预期（在调度器初始化块内）

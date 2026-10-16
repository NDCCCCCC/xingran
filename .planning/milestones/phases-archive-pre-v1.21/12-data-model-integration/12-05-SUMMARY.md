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

## 额外修复（执行中发现）

### 修复1：数据库任务记录创建时序问题
**问题**: `RegisterMACHistoryTasks` 函数试图在注册时向数据库添加任务记录，但使用了 `GlobalDB.GetDB()`，而 `SetDB()` 在任务注册之后才调用，导致 `GlobalDB` 为 `nil`。

**解决**: 修改函数签名接受 `db *gorm.DB` 参数，直接从 `Core` 传递数据库连接。

**提交**: `fix(12-05): 修复数据库任务记录创建时序问题`

### 修复2：PartitionService初始化顺序问题
**问题**: `PartitionService` 在第 19 步才初始化，而任务注册发生在第 10 步。当任务注册检查 `c.PartitionService != nil` 时，`PartitionService` 还是 `nil`，导致任务注册被跳过。

**解决**: 将 `PartitionService` 初始化移至第 9.5 步（调度器初始化之前）。

**提交**: `fix(12-05): 修复PartitionService初始化顺序`

## 偏差记录
有 - 执行过程中发现并修复了两个时序问题（GlobalDB 和 PartitionService 初始化顺序）。

## 验证结果

### 启动日志验证
✓ 通过 - 启动日志显示：
```
开始检查MAC历史保留期配置...
尝试插入MAC历史保留期配置...
MAC历史保留期配置已创建（默认120天）
```

### 数据库验证
```sql
-- sys_job 表验证
SELECT job_name, job_group, invoke_target, cron_expression, status
FROM sys_job WHERE job_name = 'MAC历史数据清理';
```
✓ 通过 - 返回 1 条记录：
- job_name: MAC历史数据清理
- job_group: SYSTEM
- invoke_target: mac_history_cleanup
- cron_expression: 0 0 2 * * ?
- status: 0

```sql
-- sys_config 表验证
SELECT config_name, config_key, config_value, config_type, is_system
FROM sys_config WHERE config_key = 'network.mac.history.retention_days';
```
✓ 通过 - 返回 1 条记录：
- config_name: MAC地址历史数据保留期
- config_key: network.mac.history.retention_days
- config_value: 120
- config_type: Y
- is_system: 1

## 后续工作
无 - 所有验证已通过。

## Self-Check: PASSED

✓ 所有任务已完成
✓ 每个任务已单独提交
✓ 编译检查通过
✓ 任务注册时机符合预期（在调度器初始化块内）
✓ 数据库验证通过（sys_job 和 sys_config 表都有记录）
✓ 启动日志验证通过

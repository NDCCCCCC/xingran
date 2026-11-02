# 启用三个失败的调度任务

## 背景

从调试报告 `.planning/debug/scheduler-task-handler-not-found.md` 中发现三个调度任务执行失败：

1. **vdi_vm_sync** - VDI虚拟机数据同步
2. **ad_group_member_sync** - AD域组成员同步
3. **dept_to_ad_sync** - 部门到AD同步

## 任务目标

修复调度器初始化和任务处理器注册问题，使这三个任务能够正常执行。

## 修复计划

### 1. 修复 VDI 任务注册
- **文件**: `internal/core/core.go`
- **操作**: 在 `Init()` 方法中添加 `scheduler.RegisterVDISyncTasks(c.Scheduler)` 调用
- **位置**: 第 285 行后（`RegisterADSyncTasks` 调用之后）

### 2. 修复 AD 调度器初始化
- **文件**: `internal/core/core.go`
- **操作**: 在 `Init()` 方法中添加 `scheduler.StartADSyncScheduler(c.GetDB())` 调用
- **位置**: 第 285 行后

### 3. 实现部门到AD同步任务
- **文件**: 创建 `internal/scheduler/dept_sync_tasks.go`
- **操作**: 参考现有的 AD 同步任务模式实现 `dept_to_ad_sync` 处理器
- **注册**: 在 `core.go` 中调用注册函数

## 实施步骤

1. 读取相关文件了解现有模式
2. 修改 `internal/core/core.go` 添加 VDI 注册和 AD 调度器启动
3. 创建 `internal/scheduler/dept_sync_tasks.go` 实现部门同步处理器
4. 在 `core.go` 中注册部门同步任务
5. 验证修改：运行 `go build ./...` 确保编译通过
6. 提交更改

## 验证标准

- ✅ 代码成功编译无错误
- ✅ 三个任务处理器都已注册
- ✅ AD 调度器在启动时初始化

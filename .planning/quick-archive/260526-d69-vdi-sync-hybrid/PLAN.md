# VDI 虚拟机混合数据同步方案

## 目标
实现 VDI 虚拟机数据的混合同步机制，确保数据始终是最新的。

## 方案概述
实施方案4（混合方案）：
1. **首次访问自动同步**（已实现）- 数据库为空时自动拉取
2. **手动同步按钮** - 用户可随时触发同步
3. **后台定期同步** - 复用现有的定时任务模块自动同步

## 实施步骤

### 1. 添加手动同步 API 端点
**文件**: `internal/api/v1/vdi/vm_handler.go`
- 添加 `SyncAll` 方法处理 POST `/api/v1/vdi/vm/sync-all` 请求
- 调用服务层的 `SyncAllVMs` 方法
- 返回同步结果（成功数量、失败数量等）

**文件**: `internal/api/v1/vdi/vm_router.go`
- 注册新路由: `r.POST("/sync-all", vmHandler.SyncAll)`

### 2. 创建 VDI 同步定时任务
**文件**: `internal/scheduler/vdi_sync_job.go`（新建）
- 实现 `VDISyncJob` 结构体，包含 `Run()` 方法
- 调用 VM 服务的 `SyncAllVMs` 方法
- 记录同步日志和统计信息

**文件**: `internal/core/core.go`
- 在初始化时注册 VDI 同步任务到调度器
- Cron 表达式: `0 */10 * * * *`（每10分钟执行一次）

### 3. 数据库初始化任务记录
**SQL**: 在 `sys_job` 表中插入 VDI 同步任务记录
```sql
INSERT INTO sys_job (job_id, job_name, job_group, invoke_target, cron_expression, misfire_policy, concurrent, status, create_by, create_time, remark)
VALUES (gen_random_uuid(), 'VDI虚拟机数据同步', 'DEFAULT', 'vdiSyncJob.Run', '0 */10 * * * *', 'DO_NOTHING', 0, 0, 'admin', NOW(), '定期从VDI服务器同步虚拟机数据');
```

### 4. 前端同步按钮（可选，后续实现）
**文件**: `xingran-react-frontend/src/pages/vdi/VirtualMachine/index.tsx`
- 在工具栏添加"同步"按钮
- 调用 `/api/v1/vdi/vm/sync-all` 端点
- 显示同步进度和结果

## 验证标准
- [ ] 手动同步 API 端点可正常调用
- [ ] 定时任务正确注册到调度器
- [ ] 同步功能正常工作，能从 VDI 服务器拉取数据
- [ ] 代码编译通过
- [ ] 功能测试通过

## 技术细节
- 复用现有的 `internal/scheduler` 模块
- 使用已有的 `job_service` 管理任务
- 同步逻辑复用 `vm_service_impl.go` 中的 `syncVMsFromVDI` 方法

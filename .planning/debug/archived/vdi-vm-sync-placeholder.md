---
slug: vdi-vm-sync-placeholder
status: resolved
deferred_to: v1.16-tech-debt
trigger: vdi虚拟机获取的定时任务提示占位，请排查代码，确认实际情况
created: 2026-05-27
updated: 2026-06-25
---

## Symptoms

### Expected Behavior
从真实 VDI 服务器（深信服VDI）获取虚拟机列表数据

### Actual Behavior
定时任务执行时显示"占位实现"，日志如下：
```
INFO[2026-05-27 16:32:31] 执行任务: VDI虚拟机数据同步, 目标: vdi_vm_sync:auto
INFO[2026-05-27 16:32:31] 开始同步 VDI 虚拟机数据，共 1 个服务器
INFO[2026-05-27 16:32:31] 开始同步 VDI 服务器 [深信服VDI] 的虚拟机数据
INFO[2026-05-27 16:32:31] VDI 服务器 [深信服VDI] 同步完成（占位实现），耗时: 100.307ms
INFO[2026-05-27 16:32:31] VDI 虚拟机数据同步完成: 成功=1, 失败=0
```

### Error Messages
无错误消息，但显示"占位实现"

### Timeline
- **曾经正常工作**：同步的数据保存到了本地数据库表里
- 占位提示不清楚是什么时候出现的
- 需要确认是否已经实现了从真实 VDI 获取虚拟机的逻辑

### Reproduction
触发定时任务 "VDI虚拟机数据同步" (vdi_vm_sync:auto)

## Current Focus

**hypothesis**: The scheduler task `syncVDIServerVMs` in `internal/scheduler/vdi_sync_tasks.go` contains only placeholder implementation that sleeps for 100ms, while the actual VDI sync logic exists in `internal/services/vdi/vm_service_impl.go` but is never called by the scheduler.
**next_action**: Implement proper integration between scheduler and VDI service
**reasoning_checkpoint**: Found that `vmServiceImpl.syncVMsFromVDI()` contains real VDI API calls, but the scheduler task `syncVDIServerVMs()` just sleeps 100ms and logs "占位实现"
**test**:
**expecting**:

## Evidence

- timestamp: 2026-05-27T16:32:31Z
  source: code_analysis
  location: internal/scheduler/vdi_sync_tasks.go:95-119
  summary: "Scheduler task `syncVDIServerVMs` contains only placeholder implementation"
  details: |
    The function `syncVDIServerVMs` in the scheduler only:
    1. Logs start message
    2. Sleeps for 100ms (line 107: `time.Sleep(100 * time.Millisecond)`)
    3. Logs completion with "占位实现" (placeholder implementation)
    4. Updates server's last_sync_time
    5. Contains TODO comment listing required implementation steps

- timestamp: 2026-05-27T16:32:31Z
  source: code_analysis
  location: internal/services/vdi/vm_service_impl.go:62-117
  summary: "Real VDI sync implementation exists but is unused by scheduler"
  details: |
    The method `vmServiceImpl.syncVMsFromVDI()` contains complete implementation:
    1. Calls `client.ListResourceGroups(ctx)` to get resource groups
    2. Iterates through groups and calls `client.ListResources(ctx, groupID)`
    3. For each resource, calls `client.ListResourceServers(ctx, resourceID, page, pageSize)`
    4. Saves/updates VM records in local database via `saveOrUpdateVM()`
    5. Uses real VDI API client with authentication and error handling

- timestamp: 2026-05-27T16:32:31Z
  source: code_analysis
  location: internal/services/vdi/vm_service_impl.go:640-663
  summary: "Public method `SyncAllVMs` exists for manual sync but scheduler doesn't use it"
  details: |
    The `vmServiceImpl.SyncAllVMs(ctx, serverID)` method:
    1. Validates server exists in database
    2. Queries all VMs for that server
    3. Calls `SyncVMFromVDI` for each VM to update status
    4. This method is accessible but never called by the scheduler task

- timestamp: 2026-05-27T17:00:00Z
  source: investigation
  location: comprehensive_data_source_analysis
  summary: "All possible VDI data entry points identified"
  details: |
    **定时任务 (Scheduler)**:
    - `vdi_vm_sync` - 当前为占位实现，不写入真实数据
    - 其他定时任务: 无发现调用 VDI 服务

    **API 端点**:
    - `POST /vdi/vm/sync-all` - 手动同步所有虚拟机
    - `POST /vdi/vm/{id}/sync` - 同步单个虚拟机
    - `POST /vdi/vm` - 创建虚拟机（调用VDI API）
    - `POST /vdi/vm/{id}/update` - 更新虚拟机
    - 所有 VM 操作都会触发数据库写入

    **手动同步入口**:
    - `vmServiceImpl.SyncAllVMs(ctx, serverID)` - 批量同步
    - `vmServiceImpl.SyncVMFromVDI(ctx, vmID)` - 单个同步
    - `vmServiceImpl.syncVMsFromVDI(ctx)` - 从VDI拉取并保存

    **前端同步按钮**:
    - 前端 API 客户端已实现: `vmApi.sync(id)` 和 `vmApi.syncAll()`
    - 位于: `xingran-react-frontend/src/lib/vdiApi.ts`
    - 用户可通过前端触发同步操作

    **数据库写入点**:
    - `saveOrUpdateVM()` - 创建或更新 VM 记录（line 119-159）
    - `CreateVM()` - 通过 VDI API 创建新 VM
    - `UpdateVM()` - 更新现有 VM 信息

    **其他数据来源**:
    - 无发现批量导入脚本
    - 无发现 webhook 回调
    - 无发现事件驱动同步机制
    - 无发现迁移脚本中的 VDI 数据初始化

## Eliminated

- timestamp: 2026-05-27T16:32:31Z
  hypothesis: "VDI client implementation is missing"
  evidence: "Found complete VDI client implementation in `internal/services/vdi/vdi_client_extended.go` with real API calls to VDI server endpoints"
  reasoning: "The VDI client `vdiClientExtendedImpl` implements `VDIClientExtended` interface with methods like `Authenticate`, `GetVM`, `ListVMs`, `ListResourceGroups`, `ListResources`, `ListResourceServers`, etc."

- timestamp: 2026-05-27T16:32:31Z
  hypothesis: "Database models for VDI are missing"
  evidence: "Found database models `VDIServer` and `VDIVirtualMachine` in `internal/models/` with proper GORM mappings"
  reasoning: "The models exist and are used by the service layer"

- timestamp: 2026-05-27T16:32:31Z
  hypothesis: "VDI service doesn't have sync logic"
  evidence: "Found complete sync implementation in `vmServiceImpl.syncVMsFromVDI()` with proper VDI API integration"
  reasoning: "The service layer has full sync logic but it's not connected to the scheduler"

- timestamp: 2026-05-27T17:00:00Z
  hypothesis: "Data comes from other scheduler tasks"
  evidence: "Searched all scheduler code, found no other tasks calling VDI services"
  reasoning: "Only `vdi_vm_sync` task exists, and it's currently a placeholder"

- timestamp: 2026-05-27T17:00:00Z
  hypothesis: "Data comes from batch import scripts"
  evidence: "No VDI import scripts found in codebase"
  reasoning: "Excel import exists for other modules (building, floor, workstation) but not for VDI VMs"

- timestamp: 2026-05-27T17:00:00Z
  hypothesis: "Data comes from webhook or event-driven sync"
  evidence: "No webhook or event-driven VDI sync mechanisms found"
  reasoning: "VDI integration is purely pull-based (scheduler or manual API calls)"

## Resolution

**root_cause**: The scheduler task `syncVDIServerVMs` in `internal/scheduler/vdi_sync_tasks.go` contains a placeholder implementation that only sleeps for 100ms and logs "占位实现", instead of calling the actual VDI sync logic implemented in `internal/services/vdi/vm_service_impl.go`. The real sync logic exists and was working before (as evidenced by user's database records), but the scheduler task was never updated to use it.

**data_source_investigation**: Based on comprehensive code analysis, the existing VDI VM data in the database came from one of these sources:

1. **手动API调用 (Most Likely)**: Users or administrators called the manual sync endpoints:
   - `POST /vdi/vm/sync-all` - Sync all VMs from VDI server
   - `POST /vdi/vm/{id}/sync` - Sync individual VM

2. **前端同步按钮**: Users triggered sync through the frontend UI:
   - Frontend has implemented `vmApi.sync(id)` and `vmApi.syncAll()` methods
   - Located in: `xingran-react-frontend/src/lib/vdiApi.ts`

3. **创建虚拟机时同步**: When VMs were created through the VDI API:
   - `POST /vdi/vm` endpoint calls `CreateVM()` which interacts with VDI server
   - This would create records both in VDI and local database

**current_data_flow**: Currently, VDI data only enters the database through:
- **Manual API calls** (sync-all or individual sync)
- **VM creation/update operations** through the API
- **Scheduler is NOT contributing data** (placeholder implementation)

**fix**: Replace the placeholder implementation in `syncVDIServerVMs` with a call to the VDI service's sync method. The fix involves:
1. Inject `vdi.VMService` into the scheduler
2. Replace the `time.Sleep(100ms)` placeholder with `vmService.SyncAllVMs(ctx, server.ID)`
3. Remove the TODO comment and "占位实现" log message
4. Keep the error handling and server last_sync_time update logic

**verification**: After fix, verify that:
1. Scheduled task triggers real VDI API calls
2. VM data from VDI server is saved to database
3. Log shows actual sync progress (resource groups, resources, VM counts)
4. No more "占位实现" message in logs

**files_changed**:
- internal/scheduler/vdi_sync_tasks.go: Replace placeholder implementation with real sync call
- internal/scheduler/scheduler.go: Add VMService dependency injection

## Phase 40 Closure (2026-06-25)

复测 `internal/scheduler/vdi_sync_tasks.go`:
- `syncVDIServerVMs` (line 92-119) 已不再 `time.Sleep(100ms)` 占位，
  改调 `GetVDIVMService()` 拿到 `VMService` 实例并调用 `SyncVMsFromVDIByServer(ctx, server)`
- 整条链路（scheduler → service → vdi_client_extended → saveOrUpdateVM）已接通

`占位实现` 字符串在 internal/scheduler/ 下已 0 命中。frontmatter 翻 `resolved`。

verification: `grep -rn "占位实现" internal/scheduler/` 0 命中；`vmService.SyncVMsFromVDIByServer` 调用存在于 vdi_sync_tasks.go:105
files_changed: .planning/debug/vdi-vm-sync-placeholder.md
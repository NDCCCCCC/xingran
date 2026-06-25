---
slug: resource-group-dropdown-empty
status: resolved
trigger: 创建虚拟机时资源组下拉框没有数据
created: 2026-05-29
updated: 2026-05-29
type: bug
---

## Symptoms

**Expected behavior:**
下拉框应该显示所有可用的资源组列表，当选择VDI服务器后应该显示该服务器对应的资源组。

**Actual behavior:**
资源组下拉框完全空白，没有任何选项可以显示。

**Error messages:**
无错误信息，没有控制台错误，没有页面错误提示。

**Timeline:**
这是刚完成的功能（提交 371aa50），首次测试时发现的问题。

**Reproduction:**
1. 打开虚拟机列表页面
2. 点击"创建虚拟机"按钮
3. 选择一个VDI服务器
4. 观察资源组下拉框 - 完全空白

## Current Focus

**hypothesis:** 已确认 - 同步流程缺少资源组持久化步骤

**next_action:** 已完成修复

**test:**
1. `go build ./...` 编译通过
2. 前端调用 `/vdi/vm/resource-groups` 查询本地数据库的 `sys_vdi_resource_group` 表
3. VDI同步时新增 `syncResourceGroups` 方法将资源组写入该表

**expecting:** 同步后下拉框显示资源组数据

## Evidence

- 2026-05-29: 前端正确调用 `vmApi.listResourceGroups(selectedServerId)` -> `/vdi/vm/resource-groups` API
- 2026-05-29: 后端路由 `internal/api/v1/vdi/vm_router.go:18` 正确注册 `ListResourceGroups` handler
- 2026-05-29: `ListResourceGroups` 服务方法 (vm_service_impl.go:288-312) 查询 `sys_vdi_resource_group` 表，按 `status=0` 和 `vdi_server_id` 过滤
- 2026-05-29: **根因**: `syncVMsFromVDI` 方法从 VDI API 获取资源组（line 70）但从未将它们写入 `sys_vdi_resource_group` 表，导致该表始终为空
- 2026-05-29: 整个代码库中除了 `ListResourceGroups` 查询外，没有其他地方写入 `models.VDIResourceGroup`

## Resolution

**root_cause:** `syncVMsFromVDI()` 从 VDI API 获取资源组数据后仅用于遍历获取虚拟机，从未将资源组持久化到 `sys_vdi_resource_group` 数据库表。`ListResourceGroups()` 查询该空表，返回空结果，导致前端下拉框无数据。

**fix:** 在 `syncVMsFromVDI` 中获取资源组后调用新增的 `syncResourceGroups` 方法，将 VDI API 返回的资源组 upsert 到 `sys_vdi_resource_group` 表。同时添加孤立资源组清理逻辑。修改文件: `internal/services/vdi/vm_service_impl.go`

## Specialist Review

specialist_hint: go
Review: LOOKS_GOOD - 根因清晰，修复方向正确。使用 upsert 模式与现有 saveOrUpdateVM 一致，enable->status 映射符合项目惯例。

## Debug Log

- Cycle 1 (investigation): 读取前端组件、API客户端、后端路由、handler、service实现，追踪完整数据流，定位到同步缺失环节
- Cycle 1 (fix): 添加 syncResourceGroups + saveOrUpdateResourceGroup 方法，go build ./... 通过

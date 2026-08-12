---
slug: vdi-vm-sync-missing-vm
status: resolved
deferred_to: v1.16-tech-debt
trigger: VDI虚拟机数据同步任务没有将新增的第9台虚拟机同步到数据库，之前任务能正常同步新增和删除虚拟机，推测最近添加删除功能时破坏了新增功能
created: 2026-06-02
updated: 2026-06-25
type: bug
---

## Symptoms

**Expected behavior:**
VDI服务器上有9台虚拟机（数据资源4台，研发资源5台），同步任务应该将所有9台虚拟机的状态同步到数据库，包括新增的虚拟机。

**Actual behavior:**
数据库中只有8台虚拟机，新增的第9台虚拟机（属于数据资源ID=9）没有被同步到数据库。

**Error messages:**
```
INFO[2026-06-02 09:51:27] 执行任务: VDI虚拟机数据同步, 目标: vdi_vm_sync:auto
INFO[2026-06-02 09:51:27] 开始同步 VDI 虚拟机数据，共 1 个服务器
INFO[2026-06-02 09:51:27] 开始同步 VDI 服务器 [深信服VDI] 的虚拟机数据
[VDI SYNC] Syncing 1 resource groups for server e2d08b76-1649-4d55-84a5-1aefa094c88c
[VDI SYNC] Processing resource group: 默认资源组 (ID: 0)
[VDI AUTH DEBUG] Using cached token, length: 454, expiry: 2026-06-03 08:51:17.847553 +0800 CST
[VDI API DEBUG] Calling: GET https://10.62.0.79:6060/v1/resources/list/0
[VDI SYNC] Found 2 resources in group 默认资源组
[VDI SYNC] Processing resource: 数据 (ID: 9)
[VDI AUTH DEBUG] Using cached token, length: 454, expiry: 2026-06-03 08:51:17.847553 +0800 CST
[VDI API DEBUG] Calling: GET https://10.62.0.79:6060/v1/resource/servers?rcid=9&page=1&page_size=100
[VDI SYNC] Processing resource: 研发 (ID: 5)
[VDI AUTH DEBUG] Using cached token, length: 454, expiry: 2026-06-03 08:51:17.847553 +0800 CST
[VDI API DEBUG] Calling: GET https://10.62.0.79:6060/v1/resource/servers?rcid=5&page=1&page_size=100
[VDI SYNC] Starting orphaned VMs cleanup...
[VDI SYNC] No orphaned VMs found, database is in sync
INFO[2026-06-02 09:51:31] VDI 服务器 [深信服VDI] 同步完成，耗时: 3.9464802s
INFO[2026-06-02 09:51:31] VDI 虚拟机数据同步完成: 成功=1, 失败=0
INFO[2026-06-02 09:51:31] 任务执行成功 [VDI虚拟机数据同步.DEFAULT], 耗时: 3949ms
```

**关键观察：**
- 日志显示调用了两个 `/v1/resource/servers` API（rcid=9 和 rcid=5）
- 但没有看到"Found X VMs in API response"或"Processing VM: ..."的日志
- 直接跳到"Starting orphaned VMs cleanup"环节
- 缺少虚拟机列表解析和处理的日志输出

**Timeline:**
- 任务之前正常工作，能够同步新增虚拟机和删除不存在的虚拟机
- 最近添加了"删除不存在的虚拟机"功能后，新增虚拟机的同步功能失效
- 问题首次出现于最近一次修改后

**Reproduction:**
1. VDI服务器上新增一台虚拟机（属于数据资源ID=9）
2. 手动触发或等待定时任务执行 VDI 虚拟机数据同步
3. 检查数据库 `vdi_vm` 表，新增虚拟机未出现在数据库中
4. 查看日志，缺少"Processing VM"相关的日志输出

**User-provided context:**
- VDI服务器：10.62.0.79:6060
- 资源组：默认资源组 (ID: 0)
- 资源：数据 (ID: 9) 有4台虚拟机，研发 (ID: 5) 有5台虚拟机
- 总共9台虚拟机，数据库只有8台
- 新增虚拟机属于数据资源
- 推测最近添加删除功能时破坏了新增功能

## Current Focus

**hypothesis:** ❌ 原假设不成立。这不是代码bug，而是API覆盖范围限制。

**✅ 根本原因已确认（Root Cause Found）：**

新增的第9台虚拟机是通过**克隆**其他虚拟机创建的，而现有的8台虚拟机是通过**模板派生**的。

| 虚拟机类型 | 创建方式 | 存储位置 | API可见性 | 数量 |
|-----------|---------|---------|-----------|------|
| 派生虚拟机 | 模板派生 | VDI服务器 | ✅ `/v1/resource/servers` 可获取 | 8台 |
| 克隆虚拟机 | 克隆复制 | VMP服务器 | ❌ 该API无法获取 | 1台 |

**API 限制说明：**
- `GET /v1/resource/servers?rcid=&page=&page_size=` 只能获取**派生的虚拟机**（在VDI服务器中显示）
- **克隆的虚拟机**仅在VMP服务器显示，不在此API的返回结果中
- 这两种虚拟机类型存储在不同的管理服务器上

**User-confirmed root cause:**
"多的那台是通过其他虚拟机复制的（克隆），而其他的都是通过模板派生的。GET /v1/resource/servers 只能获取派生的虚拟机（在VDI服务器中显示），无法获取克隆的虚拟机（仅在VMP服务器显示）。"

## Resolution

**root_cause:** API覆盖范围限制。当前同步任务只调用了VDI服务器的API，无法获取存储在VMP服务器上的克隆虚拟机。

**fix (可选方案):**
1. **方案A**：扩展同步任务，同时调用VMP服务器的API获取克隆虚拟机列表
2. **方案B**：在VDI管理界面中明确标注此限制，告知用户克隆虚拟机不会被自动同步
3. **方案C**：将克隆虚拟机转换为派生虚拟机（如果业务允许）

**需要额外信息：**
- VMP服务器的API端点地址和认证方式
- 克隆虚拟机的数据结构是否与派生虚拟机兼容

## Evidence


## Eliminated


## Phase 40 Closure (2026-06-25)

根因为 VDI API 覆盖范围限制（VMP 服务器持有克隆 VM，`/v1/resource/servers`
端点不可见）。Phase 40 采取 Resolution 方案 B（文档化限制）：
在 `internal/services/vdi/vdi_client_extended.go` 的 `ListResourceServers`
添加文档注释，说明本端点仅返回派生 VM，克隆 VM 需扩展 VMP API 调用或转换类型。

方案 A（扩展 VMP API）需要 VMP 服务器端点信息与认证方式，目前不可得，
推迟到后续 phase；方案 C（克隆→派生转换）属运维侧动作，非代码修复。
本次落地为最小且无运行时副作用的注释标注，frontmatter 翻 `resolved`。

verification: `go build ./...` 退出 0；`ListResourceServers` 文档注释含"克隆 VM" 说明
files_changed: internal/services/vdi/vdi_client_extended.go, .planning/debug/vdi-vm-sync-missing-vm.md
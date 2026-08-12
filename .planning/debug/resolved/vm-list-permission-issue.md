---
slug: vm-list-permission-issue
status: resolved
trigger: 虚拟机列表页面权限控制问题：无权限时按钮仍可见且预加载请求报403
created: 2025-06-04T16:46:00+08:00
updated: 2025-06-04T17:30:00+08:00
---

# Debug Session: VM List Permission Issue

## Symptoms

### Expected Behavior
在没有创建虚拟机、绑定用户、删除虚拟机权限时：
- 相关操作按钮应该**隐藏**
- 不应该发送创建VM所需的预加载API请求

### Actual Behavior
- **按钮仍然可见**
- 点击按钮后报403权限错误
- 页面加载时发送预加载请求并返回403错误：
  - `POST /v1/vdi/vms/vtp-platforms` -> 403 Forbidden
  - `POST /v1/vdi/vms/run-positions` -> 403 Forbidden
  - `POST /v1/vdi/vms/storages` -> 403 Forbidden
  - `POST /v1/vdi/vms/networks` -> 403 Forbidden

### Error Messages
```
[VDI Preload] 预加载失败: Error: 没有访问权限
POST http://10.62.10.33:9000/api/v1/vdi/vms/vtp-platforms 403 (Forbidden)
POST http://10.62.10.33:9000/api/v1/vdi/vms/run-positions 403 (Forbidden)
POST http://10.62.10.33:9000/api/v1/vdi/vms/storages 403 (Forbidden)
POST http://10.62.10.33:9000/api/v1/vdi/vms/networks 403 (Forbidden)
```

### Timeline
- 首次发现：2025-06-04
- 角色配置：test角色无创建VM、绑定用户、删除VM权限

### Reproduction Steps
1. 使用test角色登录（无创建VM、绑定用户、删除VM权限）
2. 访问虚拟机列表页面
3. 观察到相关按钮仍可见
4. 控制台显示预加载请求403错误

## Current Focus

### Hypothesis
已确认：前端权限控制存在5个缺陷

### Next Action
已完成修复

### Test
TypeScript编译通过

### Expecting
无权限用户不应看到操作按钮，不应触发403请求

### Reasoning Checkpoint
5个独立的root cause，全部为前端代码问题

### TDD Checkpoint
不适用（前端UI修复）

## Evidence

- 2025-06-04T17:00: 权限来源错误 - index.tsx第43行从 `user?.permissions` (authStore) 获取权限，但登录API不返回permissions字段。正确来源是 menuStore.permissions (通过 /system/my-menus/permissions 加载)
- 2025-06-04T17:05: 列渲染函数无权限检查 - columns定义中操作列(render函数)硬编码所有按钮(开机/关机/重启/同步/绑定用户/删除)，不检查权限。已有的renderOperationButtons函数正确过滤权限但未被使用
- 2025-06-04T17:08: 工具栏按钮无权限检查 - "创建虚拟机"和"快速创建"按钮始终显示，不检查 vdi:vm:add 权限
- 2025-06-04T17:10: 预加载无权限检查 - preloadVDIData 在useEffect中无条件触发，调用需要 vdi:vm:add 权限的API，导致403
- 2025-06-04T17:12: 权限标识不匹配 - vmOperationButtons.tsx使用 `vdi:vm:delete` 但后端路由使用 `vdi:vm:remove`
- 2025-06-04T17:20: 代码结构损坏 - handleQuickCreate函数的finally块未正确闭合，与renderOperationButtons函数定义交错

## Eliminated

- 后端权限中间件工作正常（正确返回403）
- menuStore权限加载逻辑正常（/system/my-menus/permissions返回正确权限列表）

## Resolution

### Root Cause
前端VM列表页面存在5个权限控制缺陷：(1) 从authStore读取permissions但登录API不返回该字段，应从menuStore读取；(2) 表格操作列硬编码所有按钮不做权限过滤；(3) 创建/快速创建按钮始终可见不检查权限；(4) 预加载VDI配置数据不检查权限导致403；(5) 删除按钮权限标识 `vdi:vm:delete` 与后端 `vdi:vm:remove` 不匹配。

### Fix Applied
1. **权限来源修复** - 导入useMenuStore，从menuStore.permissions获取权限列表（而非user?.permissions）
2. **操作列权限过滤** - 将columns操作列的render函数改为调用renderOperationButtons(record)，该函数按权限过滤按钮
3. **工具栏权限控制** - "创建虚拟机"和"快速创建"按钮仅在canCreateVM(vdi:vm:add)为true时显示
4. **预加载权限守卫** - preloadVDIData函数开头检查canCreateVM，无权限时跳过预加载，消除403请求
5. **权限标识修复** - vmOperationButtons.tsx中将 `vdi:vm:delete` 改为 `vdi:vm:remove` 匹配后端路由
6. **批量操作权限** - 批量开机/关机/重启按钮添加对应权限检查
7. **代码结构修复** - 修正handleQuickCreate函数未闭合的finally块

### Verification
TypeScript编译通过（npx tsc --noEmit 无错误）

### Files Changed
- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` - 权限来源修复、操作列权限过滤、工具栏权限控制、预加载守卫、批量操作权限
- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/vmOperationButtons.tsx` - 权限标识 vdi:vm:delete -> vdi:vm:remove

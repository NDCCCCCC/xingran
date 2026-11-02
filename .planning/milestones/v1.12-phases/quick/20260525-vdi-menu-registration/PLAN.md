---
task: 添加 VDI 菜单到 sys_menu 表
created: 2026-05-25
slug: vdi-menu-registration
---

# Quick Task: 添加 VDI 菜单到 sys_menu 表

## 目标

在 Phase 22A 完成后，需要添加 VDI 菜单到 `sys_menu` 表，以注册前端路由并使 VDI 功能在系统中可用。

## 背景信息

Phase 22A 已完成：
- 后端 API 路由已注册（`/api/v1/vdi/*`）
- 前端组件已创建（VirtualMachineList, VirtualMachineDetail, VDIServerConfig）
- 前端路由需要对应的菜单记录才能访问

## 任务内容

### 需要插入的菜单记录

根据 `xingran-react-frontend/src/router/index.tsx` 中的路由定义：

```
/vdi/vm         → 虚拟机列表
/vdi/vm/:id     → 虚拟机详情
/vdi/servers    → VDI服务器配置
```

### 菜单结构设计

```
虚拟机管理 (一级菜单, menu_type=M)
├── 虚拟机管理 (二级菜单, menu_type=C, /vdi/vm)
├── 虚拟机详情 (隐藏菜单, menu_type=C, /vdi/vm/:id)
└── VDI服务器配置 (二级菜单, menu_type=C, /vdi/servers)
```

## 执行步骤

1. 创建数据库迁移脚本（`internal/core/db/migrations/129_add_vdi_menus.sql`）
2. 插入菜单记录到 `sys_menu` 表
3. 验证菜单记录正确插入
4. 提交变更

## 菜单权限标识符

使用 Phase 22 CONTEXT.md 中定义的权限：
- `vdi:visit` — 访问虚拟机管理模块
- `vdi:vm:list` — 查看虚拟机列表
- `vdi:vm:view` — 查看虚拟机详情
- `vdi:admin` — VDI系统管理（服务器配置）

## 预期结果

- `sys_menu` 表中新增 4 条 VDI 菜单记录
- 前端路由可通过菜单访问
- 权限系统正确控制 VDI 功能访问

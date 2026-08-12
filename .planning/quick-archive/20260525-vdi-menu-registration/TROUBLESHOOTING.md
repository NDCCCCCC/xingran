# VDI 菜单问题排查和修复

## 问题描述

迁移脚本执行后，VDI 菜单没有在前端显示。

## 根本原因

1. **组件路径不匹配**: 原迁移脚本中的组件路径格式不正确
2. **可能的权限问题**: 用户角色没有分配 VDI 权限

## 解决方案

### 1. 执行修复后的迁移脚本

```bash
# 方法 1: 在数据库中直接执行
psql -U xingran -d xingran_next -f internal/core/db/migrations/130_fix_vdi_menu_component_paths.sql

# 方法 2: 重启应用（自动执行迁移）
# 迁移脚本会在应用启动时自动运行
```

### 2. 为管理员角色分配 VDI 权限

```sql
-- 查询管理员角色的 menu_id（通常是 role_id = 1）
SELECT role_id, role_name, menu_id FROM sys_role_menu 
WHERE role_id = 1 AND menu_id LIKE '770e8400%';

-- 如果没有结果，执行以下插入
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 1, id FROM sys_menu WHERE id LIKE '770e8400%';
```

### 3. 清除缓存并重新登录

```bash
# 前端清除缓存
# 1. 打开浏览器开发者工具 (F12)
# 2. 在控制台执行:
localStorage.clear();
sessionStorage.clear();

# 3. 刷新页面并重新登录
```

## 验证步骤

### 1. 检查数据库中的菜单记录

```sql
-- 检查 VDI 菜单是否存在
SELECT id, menu_name, path, component, visible, status
FROM sys_menu
WHERE id LIKE '770e8400%'
ORDER BY order_num;

-- 预期结果: 23 条记录
```

### 2. 检查角色权限

```sql
-- 检查当前登录用户的角色是否有 VDI 菜单权限
SELECT 
    m.menu_name, 
    m.path, 
    m.component
FROM sys_menu m
INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
INNER JOIN sys_user u ON ur.user_id = u.id
WHERE u.user_name = 'admin'  -- 替换为您的用户名
  AND m.id LIKE '770e8400%'
  AND m.status = '0'  -- 正常状态
  AND m.visible = '1';  -- 可见
```

### 3. 检查前端路由

```javascript
// 在浏览器控制台执行
// 检查菜单是否被加载
const menuStore = useMenuStore();
console.log('VDI 菜单:', menuStore.allMenus.filter(m => m.path.includes('vdi')));
```

## 组件路径说明

| 菜单名称 | 数据库组件路径 | 实际文件路径 |
|---------|--------------|-------------|
| 虚拟机列表 | `vdi/VirtualMachineList/index` | `src/pages/vdi/VirtualMachineList/index.tsx` |
| 虚拟机详情 | `vdi/VirtualMachineDetail/index` | `src/pages/vdi/VirtualMachineDetail/index.tsx` |
| VDI服务器配置 | `vdi/VDIServerConfig/index` | `src/pages/vdi/VDIServerConfig/index.tsx` |

## 菜单结构

```
虚拟机管理 (一级菜单, visible=1, status=0)
├── 虚拟机列表 (二级菜单, visible=1, status=0)
│   └── 9 个按钮权限 (F类型)
├── 虚拟机详情 (二级菜单, visible=0, status=0) ← 隐藏菜单
│   └── 5 个按钮权限 (F类型)
└── VDI服务器配置 (二级菜单, visible=1, status=0)
    └── 5 个按钮权限 (F类型)
```

## 常见问题

### Q1: 菜单仍然不显示

**A**: 检查以下几点：
1. 确认数据库中有 23 条 VDI 菜单记录
2. 确认用户角色有 `vdi:visit` 权限
3. 清除浏览器缓存和 localStorage
4. 重新登录

### Q2: 点击菜单提示"页面加载失败"

**A**: 检查组件文件是否存在：
```bash
ls -la xingran-react-frontend/src/pages/vdi/
# 应该看到 3 个目录: VDIServerConfig, VirtualMachineDetail, VirtualMachineList
```

### Q3: 如何给特定用户分配 VDI 权限

**A**: 有两种方法：
1. **通过角色**: 在"角色管理"页面，将 VDI 权限分配给角色
2. **直接SQL**:
```sql
-- 查询用户ID
SELECT user_id, user_name FROM sys_user WHERE user_name = 'your_username';

-- 分配菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT ur.role_id, m.id
FROM sys_user_role ur
CROSS JOIN sys_menu m
WHERE ur.user_id = 'your_user_id'
  AND m.id LIKE '770e8400%';
```

## 相关文件

- 迁移脚本: `internal/core/db/migrations/130_fix_vdi_menu_component_paths.sql`
- 前端组件: `xingran-react-frontend/src/pages/vdi/`
- 路由配置: `xingran-react-frontend/src/router/DynamicRoutes.tsx`
- 组件加载器: `xingran-react-frontend/src/router/componentLoader.tsx`

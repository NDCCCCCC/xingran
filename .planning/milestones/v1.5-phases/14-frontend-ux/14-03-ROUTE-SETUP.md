# Route Registration for MAC 历史查询 Page

**Plan:** 14-03 — 菜单/权限/路由注册
**Date:** 2026-06-14
**Status:** Ready for UAT (frontend pages由 14-01/14-02/14-04/14-05 实施)

## Frontend Implementation Status

本 plan 仅完成 **后端 sys_menu 注册** 与 **路由注册说明**。前端页面 (`/network/mac/history`) 由 Phase 14 其他 plan 实施:

| Plan  | 交付物                                                                                                            |
| ----- | ----------------------------------------------------------------------------------------------------------------- |
| 14-01 | `xingran-react-frontend/src/pages/network/mac/history/index.tsx`(主列表页 + `MACEventsTimeline` 组件)             |
| 14-02 | `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` UX 增强(已有 Phase 13 基础)                            |
| 14-04 | `history.tsx` 工具栏"导出当前查询/导出全量"按钮 + Excel 下载逻辑(`network:mac:export` 权限点消费)                  |
| 14-05 | 移动端响应式 + 空/加载/错误三态打磨                                                                              |

## Backend Menu Registration Required

执行本目录的 `14-menu-registration.sql` 脚本即可完成 sys_menu 注册。脚本包含 5 步:

1. **验证父菜单存在**: 查询 `sys_menu` 中 `id = 0013f129-3ec0-4e55-8ffc-25d97b20c37b` 是否存在(MAC地址父菜单)。
2. **注册 MAC 历史查询主菜单**: `path = 'mac/history'`, `component = 'pages/network/mac/history'`, `perms = 'network:mac:list'`, `order_num = 11`。
3. **注册"查询"按钮权限点**: `menu_type = 'F'`, `perms = 'network:mac:query'`, `visible = 0`(不在菜单显示)。
4. **注册"导出"按钮权限点**: `menu_type = 'F'`, `perms = 'network:mac:export'`, `visible = 0`(由 14-04 实施按钮渲染时消费)。
5. **验证注册结果**: 查询 `WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export')`,预期返回 3 行。

### SQL 执行步骤

```bash
# 1. 连接 PostgreSQL 数据库
psql -h <DB_HOST> -U <DB_USER> -d <DB_NAME>

# 2. 执行脚本
\i .planning/phases/14-frontend-ux/14-menu-registration.sql

# 3. (可选)手动重跑:三个 INSERT 均带 WHERE NOT EXISTS,幂等安全
\i .planning/phases/14-frontend-ux/14-menu-registration.sql
```

**重要:** 执行完成后请将步骤 5 的 SELECT 输出截图或粘贴到本次会话记录中,作为 UAT 证据。

## Route Configuration

| 字段              | 值                                              | 说明                                                            |
| ----------------- | ----------------------------------------------- | --------------------------------------------------------------- |
| **Path**          | `/network/mac/history`                          | 浏览器 URL                                                       |
| **Component**     | `pages/network/mac/history`                      | 前端组件路径(由 `routeGenerator` 自动解析)                       |
| **Parent Menu**   | MAC 地址 (id: `0013f129-...`, path: `mac`)       | Phase 13 已注册的父菜单                                          |
| **Order**         | 11                                              | 父菜单下排序(MAC轨迹查询 = 10,本菜单 = 11)                      |
| **Icon**          | `history`                                       | AntD 图标名,与 `meta.icon` 一致                                  |
| **Visible**       | 1                                               | 侧边栏显示                                                        |
| **Status**        | 0                                               | 启用(遵循 0=正常 1=停用 约定)                                    |
| **Perms(主)**     | `network:mac:list`                              | 菜单可见性权限点                                                  |
| **Perms(操作)**   | `network:mac:query`, `network:mac:export`        | 按钮级权限点(不显示在菜单,仅用于按钮控制)                       |
| **Meta**          | `{"icon":"history","title":"MAC 历史查询", ...}` | 前端动态路由元数据                                                |

## Access Pattern (DynamicRoutes 自动发现)

注册完成后,前端 `DynamicRoutes` 组件(`xingran-react-frontend/src/router/DynamicRoutes.tsx`)会:

1. 用户登录成功后,从 `/getInfo` 接口拉取角色绑定的菜单列表(含 `path`/`component`/`meta`)。
2. `routeGenerator` 根据 `component = 'pages/network/mac/history'` 自动解析为动态导入的 React 组件(`React.lazy`)。
3. 路由自动添加至 `react-router-dom` 路由表,包裹 `Suspense` 与 `Layout`(沿用 Phase 30 D-08 懒加载规范)。
4. 侧边栏根据 `meta.icon` + `meta.title` 渲染菜单项。
5. 访问 `/network/mac/history` 触发组件懒加载,显示页面。

**不需要手动修改任何 React Router 配置**(这是 XingRan-Next 动态路由模式的核心特性,参考 `docs/项目概述和架构设计.md`)。

## 权限分配 (Role Binding)

`network:mac:list/query/export` 三个权限点必须绑定到运维相关角色,否则普通用户看不到菜单。可通过 `sys_role_menu` 表关联:

```sql
-- 绑定到 admin 和 ops 两个角色(占位,具体角色 key 以实际项目为准)
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.role_id, m.id
FROM sys_role r, sys_menu m
WHERE r.role_key IN ('admin', 'ops')
  AND m.perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export')
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu srm
    WHERE srm.role_id = r.role_id AND srm.menu_id = m.id
  );
```

### 三个权限点的职责分工

| 权限点                | 菜单层级 | 消费方       | 用途                                                                 | 实施 plan |
| --------------------- | -------- | ------------ | -------------------------------------------------------------------- | --------- |
| `network:mac:list`    | 菜单项   | DynamicRoutes | 菜单可见性判断:有权限的用户才能在侧边栏看到 "MAC 历史查询" 菜单项   | 14-03     |
| `network:mac:query`   | 按钮权限 | history.tsx  | 工具栏"查询"按钮可见性(以及后端 `/network/history/list` 权限拦截)    | 14-01     |
| `network:mac:export`  | 按钮权限 | history.tsx  | 工具栏"导出当前查询"/"导出全量"按钮可见性                            | 14-04     |

**职责边界:** 本 plan (14-03) 仅注册三个权限点到 `sys_menu` 表。**实际的按钮渲染、权限拦截、API 调用全部由 14-01 / 14-04 实施**。14-04 在历史文件中应通过 `useAuth().permissions.includes('network:mac:export')` 或等价机制读取权限点。

## Verification Checklist (UAT 验收清单)

执行人完成 SQL 脚本运行后,逐项勾选:

- [ ] **(a) SQL 成功执行**:步骤 5 的 `SELECT` 返回 3 行(无报错)。
- [ ] **(b) sys_menu 有 3 行新记录**:菜单 1 行 + 按钮 2 行,按 `order_num` 排序依次为 11/12/13。
- [ ] **(c) 角色绑定**:admin/ops 角色在 `sys_role_menu` 中能找到对应 menu_id(执行上面的 INSERT 之后)。
- [ ] **(d) 登录后侧边栏可见**:使用 admin/ops 账号登录,在"网络管理 > MAC 地址"下能看到 "MAC 历史查询" 子菜单。
- [ ] **(e) 路由可达**:点击 "MAC 历史查询" 菜单后,浏览器跳转到 `/network/mac/history`,页面不报 404。
- [ ] **(f) DynamicRoutes 自动注册生效**:DevTools Network 面板能看到 `pages/network/mac/history` 的 chunk 动态加载(`React.lazy`)。
- [ ] **(g) network:mac:query 权限控制查询按钮**:无该权限的用户(例如纯查看角色)进入页面后,"查询"按钮不显示或被 disabled(由 14-01 实施,本 plan 仅占位)。
- [ ] **(h) network:mac:export 权限控制导出按钮**:无该权限的用户进入页面后,"导出"按钮不可见或被 disabled(由 14-04 实施,本 plan 仅占位)。

## 回滚 (Rollback)

如需撤销本 plan 的注册:

```sql
-- 1. 解除角色绑定
DELETE FROM sys_role_menu
WHERE menu_id IN (
  SELECT id FROM sys_menu
  WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export')
);

-- 2. 删除菜单条目(注意:同时删除 1 个菜单项 + 2 个按钮权限点)
DELETE FROM sys_menu
WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export');

-- 3. 验证清理结果(预期返回 0 行)
SELECT COUNT(*) FROM sys_menu
WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export');
```

## 关联文档

- **上游规范**: `.planning/phases/14-frontend-ux/14-CONTEXT.md` §D-04 菜单与权限规范(锁定 `network:mac:list/query/export`)
- **SQL 模板**: `.planning/phases/13-query-layer-trajectory/13-06-menu-registration-v4.sql` (Phase 13 已交付的 MAC 轨迹查询菜单注册)
- **需求清单**: `.planning/REQUIREMENTS.md` §UI-01 (历史查询页面)、§UI-04 (时间线组件);UI-02 归属 14-04
- **CLAUDE.md**: `0=正常/启用,1=停用/隐藏` 状态约定;`status` 字段值 0 表示启用
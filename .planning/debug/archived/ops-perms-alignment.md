---
slug: ops-perms-alignment
status: resolved
created: 2026-06-17
updated: 2026-06-17
title: ops 模块 perms 命名系统性对齐方案
trigger: "空间管理角色勾全所有空间菜单(含增删改查),但楼宇/楼层/工位/机房/专线/信息点/网络设备/字典等接口全部 403"
root_cause: "菜单 seed (database.go) 用复数+连字符+remove/query 命名 perms, 路由代码用单数+无连字符+delete, 两边全面不一致 → 精确匹配失败 → 读写全 403"
---

# ops 模块 perms 命名系统性对齐方案

## 根因 (诊断 SQL 已证实)

用户"空间管理"角色持有 perms (来自 sys_menu seed):
```
ops:buildings:list/add/edit/remove + query   (复数 s)
ops:floors:...                                (复数)
ops:workstations:...                          (复数)
ops:info-points:...                           (连字符)
ops:server-rooms:...                          (连字符)
ops:dedicated-lines:...                       (连字符)
ops:room-devices:...                          (连字符)
```

路由 RequirePermissions 要求:
```
ops:building:list/add/edit/delete   (单数, delete)
ops:floor:...
ops:workstation:...
ops:infopoint:...                    (合并小写)
ops:serverroom:...
ops:line:...                         (⚠️ 路由用了 line, 与菜单 dedicated-lines 完全不同)
ops:roomdevice:...
```

**两边对不上 → checkUserPermission 精确匹配失败 → 读写全 403。**
building/floor/workstation 的 list 之前能 200, 只是因为 RequirePermissionsWithQuery 放行了 ops:building:spaces:list, 不是因为 buildings 匹配上了 building。

## 统一规范 (建议)

**单数、无连字符、动作词 list/add/edit/delete** —— 与 system (user/role/dept/menu/dict)、network (device/credential/template)、asset (Migrate146 已用 ops:asset:list/add/edit/delete) 模块完全一致。ops 的 buildings/floors/... 这批 seed 是唯一异类。

## 对齐矩阵 (菜单/DB 错 → 目标对)

| 模块 | 当前 perms (错) | 目标 perms (对) |
|------|----------------|----------------|
| 楼宇 | `ops:buildings:*` | `ops:building:*` |
| 楼层 | `ops:floors:*` | `ops:floor:*` |
| 工位 | `ops:workstations:*` | `ops:workstation:*` |
| 信息点 | `ops:info-points:*` | `ops:infopoint:*` |
| 机房 | `ops:server-rooms:*` | `ops:serverroom:*` |
| 专线 | `ops:dedicated-lines:*` (菜单) / `ops:line:*` (路由⚠️) | `ops:dedicatedline:*` (两边统一) |
| 机房设备 | `ops:room-devices:*` | `ops:roomdevice:*` |
| 删除动作 | `*:remove` | `*:delete` |
| 查询动作 | `*:query` (F按钮) | 保留 `*:query` (仅 resource 名改单数; 路由不查 query, 靠继承无害) |
| 楼宇空间 | `ops:building:spaces:*` | **不变** (正确, 只读可视化 perms) |

## 改动清单 (4 处, 前端不改)

### 1. `internal/core/db/database.go` (seed 源, 影响新部署)
- 页面菜单 perms (1317-1323): buildings/floors/workstations/info-points/server-rooms/dedicated-lines/room-devices → 单数/合并
- 按钮 perms (1371-1404): resource 名改单数; `:remove` → `:delete`
- 注: seed 是"不存在才创建"(非幂等更新), 改它只为新环境正确

### 2. `internal/api/router.go` (dedicatedLine 路由异常项)
- 第 718-723 行: `ops:line:list/add/edit/delete` → `ops:dedicatedline:list/add/edit/delete`
- 其他 ops 路由 (building/floor/workstation/serverroom/infopoint/roomdevice/asset) 已是单数, 不动

### 3. `internal/core/db/migrations/migration_159_align_ops_perms.go` (修现有 DB, 必须)
- 新建 Go migration 函数 `Migrate159AlignOpsPerms(db)`, UPDATE sys_menu perms:
  - 连字符模块先替换 (dedicated-lines/server-rooms/info-points/room-devices)
  - 复数模块 (buildings/floors/workstations)
  - `:remove` → `:delete` (仅 ops 范围)
- 幂等 (WHERE perms LIKE 'ops:...%'), 重复执行无副作用
- 注册到 database.go (Migrate158 之后)

### 4. 前端
- **不改**。前端按钮权限从 `/my-menus` 动态获取 (grep 证实 xingran-react-frontend 零硬编码 ops perms)。

## 执行顺序
1. 写 migration_159 (UPDATE DB)
2. 改 database.go seed 源
3. 改 router.go dedicatedLine
4. 注册 migration_159 到 database.go
5. `go build ./...` + `go test ./pkg/middleware/`
6. 重启后端 (migration 自动跑, DB perms 更新; 缓存失效)
7. 用户重新登录 (刷新菜单/perms 缓存)

## 风险与回滚
- sys_role_menu 关联 menu_id (不改 menu_id), 只改 sys_menu.perms 值 → 角色已勾的菜单关联不变, perms 值更新后自动匹配路由 → 读写都通
- 回滚: 反向 UPDATE sys_menu perms (单数→复数) 即可恢复
- 不影响 building:spaces 只读页 (perms 不变)

## 已执行 (2026-06-17)

files_changed:
  - internal/api/router.go (dedicatedLine 组 ops:line:* → ops:dedicatedline:*, 4 处)
  - internal/core/db/database.go (seed 源 7 页面菜单 + 28 按钮 perms 改单数/无连字符/remove→delete; 注册 Migrate159)
  - internal/core/db/migrations/migration_159_align_ops_perms.go (新增, UPDATE sys_menu perms, 幂等)
verification:
  - go build ./... exit 0
  - go test ./pkg/middleware/ 通过
deploy:
  - 重启 xingran-backend.exe → migration_159 自动 UPDATE sys_menu.perms
  - 用户重新登录 (刷新菜单/perms 缓存)
  - 验证: 重跑诊断 SQL, perms 应已变为 ops:building:* / ops:serverroom:* / ops:dedicatedline:* (单数)

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测三处确认修复落地 — (1) `internal/core/db/migrations/migration_159_align_ops_perms.go` 文件存在，是新格式的 Go 迁移（非 archive SQL）；(2) `internal/core/db/database.go:382` 注册调用 `migrations.Migrate159AlignOpsPerms(d.DB)` 启动时自动执行；(3) `internal/api/router.go:753-759` 专线模块已用 `ops:dedicatedline:list/add/edit/delete`（grep 命中 4 处）。原 .md 述及的"路由端对齐 + seed 改单数 + migration 159 UPDATE DB"三链路完整保留在当前代码树，可被 auto-migrate 调用。
files_changed: internal/api/router.go (dedicatedLine 改为 ops:dedicatedline:*) + internal/core/db/database.go (seed 单数化 + 注册 migration_159) + internal/core/db/migrations/migration_159_align_ops_perms.go (新增)
action: re-verify-then-flip (D-01)

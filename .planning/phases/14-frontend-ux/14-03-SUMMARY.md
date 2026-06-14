---
phase: 14-frontend-ux
plan: 03
subsystem: menu-permission-registration
tags: [sys_menu, permission, route-setup, mac-history]
dependency_graph:
  requires:
    - 13-query-layer-trajectory/13-06-menu-registration-v4.sql (parent menu & column structure)
    - 13-query-layer-trajectory/13-04-ROUTE-SETUP.md (doc format template)
    - 14-frontend-ux/14-CONTEXT.md (D-04 permission specs)
  provides:
    - sys_menu entry for MAC 历史查询 page
    - 3 permission points: network:mac:list/query/export
    - Rollback SQL and 8-item UAT verification checklist
  affects:
    - 14-01 (history list page consumes network:mac:list/query)
    - 14-02 (trajectory page — unrelated, no impact)
    - 14-04 (export buttons consume network:mac:export)
    - 14-05 (mobile responsive — unrelated, no impact)
tech-stack:
  added: []
  patterns:
    - sys_menu INSERT with WHERE NOT EXISTS idempotency guard
    - DynamicRoutes auto-discovery via menu metadata
    - menu_type='F' button permission points (visible=0)
key-files:
  created:
    - .planning/phases/14-frontend-ux/14-menu-registration.sql
    - .planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md
  modified: []
decisions:
  - "沿用 13-06 v4 SQL 模板的列序 (id, created_at, updated_at, created_by, updated_by, version, menu_name, ...)" 
  - "parent_id = 0013f129-3ec0-4e55-8ffc-25d97b20c37b 引用 Phase 13 已注册的 MAC 地址父菜单"
  - "order_num = 11 排在 MAC 轨迹查询 = 10 之后"
  - "component = 'pages/network/mac/history' 供 14-01 创建页面后由 DynamicRoutes 自动解析"
  - "network:mac:list 作为主菜单可见性权限点;query/export 作为按钮级权限点 (menu_type='F')"
  - "network:mac:export 权限点注册后由 14-04 在 history.tsx 工具栏消费"
  - "三个 INSERT 全部带 WHERE NOT EXISTS 保护,确保脚本可重复执行不产生重复条目"
metrics:
  duration: ~5 min
  completed_date: 2026-06-14
  task_count: 2
  file_count: 2
  commit_count: 2
---

# Phase 14 Plan 03: Menu/Permission/Route Registration Summary

## One-Liner

注册 `MAC 历史查询` 菜单及 `network:mac:list/query/export` 三个权限点,生成可重复执行的 sys_menu SQL 与完整 UAT 验收文档,作为 Phase 14 Wave 2 (14-01/14-02/14-04/14-05) 路由可达的前置门。

## Tasks Completed

| Task | Name                                       | Commit  | Files Created                                              |
| ---- | ------------------------------------------ | ------- | ---------------------------------------------------------- |
| 1    | 编写 sys_menu 注册 SQL 脚本                | 7a91a9f | `.planning/phases/14-frontend-ux/14-menu-registration.sql` |
| 2    | 编写路由注册说明文档                        | fd106cd | `.planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md`     |

## What Was Built

### Task 1: SQL 注册脚本

`.planning/phases/14-frontend-ux/14-menu-registration.sql` (194 行) 包含 5 个段:

1. **步骤 1 — 父菜单校验**:SELECT 查询 `id = 0013f129-3ec0-4e55-8ffc-25d97b20c37b` 是否存在。
2. **步骤 2 — 主菜单 INSERT** (幂等):`menu_type='C'`, `path='mac/history'`, `component='pages/network/mac/history'`, `perms='network:mac:list'`, `order_num=11`, `icon='history'`。`WHERE NOT EXISTS` 守卫保证可重复执行。
3. **步骤 3 — "查询"按钮权限 INSERT** (幂等):`menu_type='F'`, `perms='network:mac:query'`, `visible=0`, 由 14-01 history.tsx 工具栏"查询"按钮消费。
4. **步骤 4 — "导出"按钮权限 INSERT** (幂等):`menu_type='F'`, `perms='network:mac:export'`, `visible=0`, 由 14-04 history.tsx 工具栏"导出"按钮消费。
5. **步骤 5 — 验证查询**:SELECT 三个权限点,预期返回 3 行。

底部包含回滚 SQL (DELETE FROM sys_menu WHERE perms IN ...) 已注释,供运维查阅。

### Task 2: 路由注册说明文档

`.planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md` (136 行) 沿用 13-04 ROUTE-SETUP.md 的同构格式,包含:

- **前置依赖表**:列出 14-01/14-02/14-04/14-05 的交付物路径
- **SQL 执行步骤**:`psql` 连接 + `\i` 执行 + 重跑幂等说明
- **Route Configuration 表格**:11 个字段的最终值与说明
- **Access Pattern**:DynamicRoutes 自动发现 + routeGenerator 解析 + React.lazy 懒加载的完整流程
- **权限分配 SQL**:admin/ops 角色绑定 (INSERT INTO sys_role_menu)
- **三个权限点职责分工表**:list/query/export 分别归属哪个 plan 消费
- **Verification Checklist (8 项)**:从 SQL 执行到权限拦截的全链路验收
- **回滚 SQL**:解除角色绑定 + 删除菜单 + 验证清理
- **关联文档引用**:14-CONTEXT.md §D-04、13-06 v4 SQL、REQUIREMENTS.md §UI-01/UI-04

## Acceptance Criteria Verification

### Task 1
- [x] 包含 3 个 INSERT INTO sys_menu (1 主菜单 + 2 按钮权限) — `grep -c "INSERT INTO sys_menu" = 3`
- [x] parent_id 字面量 = `0013f129-3ec0-4e55-8ffc-25d97b20c37b` — 已验证
- [x] path = `mac/history`, component = `pages/network/mac/history` — 已验证
- [x] perms = `network:mac:list`, order_num = 11 — 已验证
- [x] 按钮权限 menu_type = `F`, perms 分别为 `network:mac:query` 和 `network:mac:export`, visible = 0 — 已验证
- [x] 包含 WHERE NOT EXISTS 形式的幂等保护 (3 处) — `grep -c "WHERE NOT EXISTS" = 3`
- [x] 包含验证 INSERT 结果的 SELECT 语句 (步骤 5) — 已验证

### Task 2
- [x] 文件存在 — `.planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md`
- [x] 包含 8 个 Verification Checklist 项 (≥ 6) — `grep -c "^- \[ \]" = 8`
- [x] 文档引用 `14-menu-registration.sql` (3 处) — 已验证
- [x] 文档说明三个权限点用途 (19 处 `network:mac:` 引用) — 已验证
- [x] 文档包含回滚 SQL — `DELETE FROM sys_menu WHERE perms IN (...)` 已验证

### Phase Success Criteria
- [x] SQL 包含 1 个主菜单 + 2 个按钮权限 INSERT
- [x] 主菜单字段值与 14-CONTEXT.md §D-04 完全一致
- [x] ROUTE-SETUP.md 提供 8 项验收 checklist 与回滚 SQL
- [x] 文件位于 `.planning/phases/14-frontend-ux/`,文件名符合命名规范
- [x] `network:mac:export` 权限点已注册,归属 14-04 实施
- [x] UI-02 不在 plan requirements 字段中 (归属 14-04)

## Deviations from Plan

**None — plan executed exactly as written.**

## Integration Points

- **前置依赖**:
  - Phase 13-06 v4 SQL 提供列序与父菜单 ID (硬性引用)
  - Phase 13-04 ROUTE-SETUP.md 提供文档结构模板
- **后续消费者**:
  - **14-01** 创建 `pages/network/mac/history/index.tsx` → DynamicRoutes 自动从 sys_menu 读取 component 字段挂载
  - **14-01** 创建 `MACEventsTimeline` 组件 → 消费 `network:mac:query` 权限点
  - **14-04** 创建导出按钮 → 消费 `network:mac:export` 权限点
  - **14-05** 移动端适配 → 不依赖本 plan 的菜单结构(独立 UX 优化)

## Risk Notes

1. **父菜单 ID 硬编码**:SQL 中 `0013f129-3ec0-4e55-8ffc-25d97b20c37b` 是 Phase 13-06 实际写入的 ID。**执行前需先运行步骤 1 的 SELECT 验证**;若父菜单不存在(Phase 13 未执行),INSERT 会因外键约束失败。
2. **菜单排序冲突**:order_num = 11 假设父菜单下没有其他 order_num = 11 的菜单。**若 13-06 之外的其他脚本也插入到 11**,需手动调整 order_num 避免排序并列。
3. **角色绑定是占位**:ROUTE-SETUP.md 中的 `INSERT INTO sys_role_menu` 使用 `('admin', 'ops')` 作为示例,**实际项目的运维角色 key 可能不同**(需 UAT 阶段由运维确认)。
4. **UI-02 严格隔离**:本 plan 不涉及导出按钮实现,仅注册权限点;若 14-04 提前发现权限点未注册,可使用本 SQL 单独补跑步骤 4(幂等安全)。

## Self-Check

```bash
# Files exist
[ -f ".planning/phases/14-frontend-ux/14-menu-registration.sql" ] && echo "FOUND: 14-menu-registration.sql" || echo "MISSING: 14-menu-registration.sql"
[ -f ".planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md" ] && echo "FOUND: 14-03-ROUTE-SETUP.md" || echo "MISSING: 14-03-ROUTE-SETUP.md"

# Commits exist
git log --oneline | grep -q "7a91a9f" && echo "FOUND: 7a91a9f (Task 1)" || echo "MISSING: 7a91a9f"
git log --oneline | grep -q "fd106cd" && echo "FOUND: fd106cd (Task 2)" || echo "MISSING: fd106cd"
```

All checks passed.

## Self-Check: PASSED
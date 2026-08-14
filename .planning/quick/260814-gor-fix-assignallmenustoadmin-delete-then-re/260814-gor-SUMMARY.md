---
phase: quick-260814-gor
plan: 01
subsystem: auth
tags: [permission, gorm, startup, admin, role-menu, idempotent, transaction]

# Dependency graph
requires: []
provides:
  - "assignAllMenusToAdmin 增量幂等差集补全实现（Pluck id + 差集 + 事务批量 INSERT，无 DELETE）"
affects: [startup-bootstrapping, admin-permissions, sys_role_menu]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "diff-completion（预查已有 + 差集 + 批量 INSERT）替代 delete-then-reinsert"
    - "Pluck 单列查询替代 SELECT * 全表，降低 dev 慢库启动负载"
    - "写操作包单事务，失败回滚不丢已有权限"

key-files:
  created: []
  modified:
    - "pkg/permission/service.go"

key-decisions:
  - "RoleMenu 无 (role_id,menu_id) 唯一约束/无主键 → 不能 ON CONFLICT，采用预查已有 + 差集 + INSERT（差集天然去重）"
  - "不再清理指向已软删菜单的陈旧 role_menu 关联（读取层 JOIN deleted_at IS NULL 天然屏蔽，无害）"

patterns-established:
  - "增量幂等补全：差集为空秒过、差集非空事务批量补全、永不 DELETE 已有关联"

requirements-completed:
  - BUGFIX-ASSIGNALLMENUS-INCREMENTAL-IDEMPOTENT

# Metrics
duration: ~7min
completed: 2026-08-14
---

# Quick 260814-gor: assignAllMenusToAdmin 增量幂等差集补全 Summary

**改写 assignAllMenusToAdmin 为增量幂等差集补全（Pluck id + 差集 + 单事务 CreateInBatches），删除原"先删后全量重插"的丢权限定时炸弹**

## Performance

- **Duration:** ~7 min
- **Completed:** 2026-08-14T04:07:39Z
- **Tasks:** 1
- **Files modified:** 1（`pkg/permission/service.go`）

## Accomplishments

- 消除 admin 权限丢失风险：删除 `db.Where("role_id = ?", adminRole.ID).Delete(&models.RoleMenu{})` 整行，不再有"已删未插"的中间窗口。
- 启动加速：admin 已齐备时（差集为空）函数直接 `return nil`，不触碰 `sys_role_menu` 表。
- 查询降载：菜单查询从 `SELECT * FROM sys_menu`（全字段全表，dev 慢库 239 条曾耗时 2 分 27 秒触发 `unexpected EOF`）改为 `Pluck("id")` 仅取 id 列。
- 数据完整性：补全过程（差集非空时）包在单一 `db.Transaction` 内，`CreateInBatches(missing, 100)` 失败回滚不丢已有权限。

## Task Commits

1. **Task 1: 重写 assignAllMenusToAdmin 为增量幂等差集补全** - `a0ea57b` (fix)

## Files Created/Modified

- `pkg/permission/service.go` - `assignAllMenusToAdmin`（70-96 行旧实现）改为 Pluck id + 差集 + 事务批量 INSERT 的增量幂等补全；签名 `func (s *service) assignAllMenusToAdmin(db *gorm.DB) error` 不变；`InitDefaultRolesAndMenus` 调用形态不变；其余方法（`createDefaultAdminRole`/`UpdateRoleMenus`/`UpdateRoleDepts`/`GetUserMenus`/`GetUserPermissions`）未改。

## 新逻辑（已落地）

```
1. db.Where("role_key = ?", "admin").First(&adminRole)                       // 查 admin role（保持）
2. db.Model(&models.Menu{}).Pluck("id", &allMenuIDs)                         // 全部现存菜单 id（GORM 自动过滤软删）
3. db.Model(&models.RoleMenu{}).Where("role_id = ?", adminRole.ID).
   Pluck("menu_id", &existingIDs)                                            // admin 已有
4. existingSet = set(existingIDs); missing = [{adminRole.ID, mid} for mid in allMenuIDs if mid not in existingSet]
5. if len(missing) == 0 { return nil }                                       // 幂等快速路径
6. return db.Transaction(func(tx) error { return tx.CreateInBatches(missing, 100).Error })
```

## 行为等价性说明（重要差异）

- **原语义**："admin 拥有全部菜单"——靠每次启动 `delete-all + re-insert-all`（无事务）达成。
- **新语义**："admin 拥有全部菜单"——靠 `diff-completion`（只补缺失、永不删除）达成。
- **可接受差异**：不再清理"指向已软删菜单的陈旧 role_menu 关联"。这是无害的：
  - 这些 `menu_id` 在 `sys_menu` 已被软删，`GetUserMenus`（`service.go:183-233`）/`GetUserPermissions`（`service.go:272-286`）的 JOIN 均带 `m.deleted_at IS NULL`，陈旧关联在读取层被天然屏蔽，不会授予任何权限。
  - admin 的目标本就是"拥有全部**现存**菜单"，陈旧关联不影响此目标。
  - 对应威胁 T-quick-gor-03（Information Disclosure / 权限残留）= accept，已在 PLAN `<threat_model>` 记录。

## 数据完整性保证

- 写操作（`CreateInBatches`）包在单一 `db.Transaction` 内：要么完整补全缺失项、要么整体回滚。任何中断（网络抖动、连接断开）都不会把 admin 留在"已删未插"的丢权限中间状态——因为根本不再有 DELETE 步骤。

## 启动性能改善

- **查询降载**：`db.Find(&allMenus)`（`SELECT *`，239 条全字段）→ `db.Model(&models.Menu{}).Pluck("id", &allMenuIDs)`（仅取 id 列）。数据量与耗时大幅下降，规避 Supabase pooler `unexpected EOF`。
- **幂等快速路径**：admin 已齐备时差集为空，`return nil` 秒过，不触碰 `sys_role_menu` 表。

## Decisions Made

- RoleMenu 无 (role_id, menu_id) 唯一约束/无主键（`internal/models/role.go:22-25` 已核实）→ 不能 `ON CONFLICT`，必须"预查已有 + 差集 + INSERT"；差集构造天然去重。
- `Menu` 嵌入 `BaseModel`（`internal/models/menu.go:48` + `internal/models/base.go`）→ GORM 对 `Pluck("id")` 自动追加 `deleted_at IS NULL`，无需手写过滤。
- 不强加新测试：`pkg/permission/` 无 `service_test.go`，PLAN 约定不新增（既有测试为 `config_v1201_test.go` 与 `resource_action_map_test.go`，均不覆盖本函数）。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- `go build ./...` → exit 0（全量编译通过，强制门）。
- `go test ./pkg/permission/...` → `ok github.com/xingran-next/xingran-go-backend/pkg/permission 0.653s`（既有测试 config_v1201_test / resource_action_map_test 不回归）。
- 静态 grep 确认（`pkg/permission/service.go`）：
  - `assignAllMenusToAdmin` 函数体内含 `Pluck("id", &allMenuIDs)`（L89）、`Pluck("menu_id", &existingIDs)`（L97）、`db.Transaction(`（L123）、`CreateInBatches(missing`（L124）。
  - 函数体内**不再**含 `Delete(&models.RoleMenu{})` 与 `Find(&allMenus`（L142 的 `Delete(&models.RoleMenu{})` 属 `UpdateRoleMenus`，未触碰）。
  - 函数签名仍为 `func (s *service) assignAllMenusToAdmin(db *gorm.DB) error`（L81）。
- 可选运行验证（dev 环境可用时，本次未执行）：连续二次启动后查 `SELECT count(*) FROM sys_role_menu WHERE role_id = <admin>` 与 `SELECT count(DISTINCT menu_id) FROM sys_role_menu WHERE role_id = <admin>` 应相等且等于 `sys_menu` 现存菜单数（无重复、无丢失）。

## Next Phase Readiness

- 单文件、零接口变更、向后兼容；`go build ./...` 与 `go test ./pkg/permission/...` 全绿。
- `internal/core/core.go:308-313` 调用点未改，`InitDefaultRolesAndMenus` 行为对调用方透明（仍是"为 admin 分配全部菜单"，仅实现策略变更）。

## Self-Check: PASSED

- FOUND: `pkg/permission/service.go`
- FOUND: `.planning/quick/260814-gor-fix-assignallmenustoadmin-delete-then-re/260814-gor-SUMMARY.md`
- FOUND: commit `a0ea57b`

---
*Quick: 260814-gor*
*Completed: 2026-08-14*

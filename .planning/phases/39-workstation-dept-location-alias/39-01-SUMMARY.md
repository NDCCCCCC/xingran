---
phase: 39-workstation-dept-location-alias
plan: 01
subsystem: database
tags: [gorm, postgresql, migration, permissions, rbac, sys_dept]

# Dependency graph
requires: []
provides:
  - sys_dept_location_alias 表结构(承载逻辑部门→物理部门映射)
  - 4 条 ops:location:alias:* 按钮权限(sys_menu seed)
  - Migration 165 注册入口(GORM AutoMigrate + partial unique index)
affects:
  - 39-02 (alias CRUD API 读取此表)
  - 39-03 (工位部门下拉 union 注入此表)
  - future role authorization (管理员需手动授予 4 个新按钮权限)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - GORM AutoMigrate 用于 PG/SQLite 双 DB 兼容建表
    - partial unique index 通过原生 SQL `CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL` 兜底(GORM tag 不支持 partial 语法)
    - 按钮权限 seed 模式: parent menu lookup → idempotent count check → create with menu_type='F' visible=0

key-files:
  created:
    - internal/models/sys_dept_location_alias.go
    - internal/core/db/migrations/migration_165_sys_dept_location_alias.go
  modified:
    - internal/core/db/database.go

key-decisions:
  - "GORM AutoMigrate 替代原生 CREATE TABLE SQL — 兼容 SQLite 双 DB,绕开 UUID/gen_random_uuid 语法差异"
  - "varchar(64) 字段类型对齐 sys_workstation.dept_id — 满足 CLAUDE.md UUID Foreign Keys 约定"
  - "Scope 字段默认 'workstation' 预留扩展 — D-04 决策"
  - "partial unique index 在 PG 上跑(migration 165 注释中标注 SQLite 跳过) — 本项目生产 DB 为 PG"
  - "sys_role_menu 不写入任何角色 — D-04 锁定: 谁也不给,管理员手动授权"

patterns-established:
  - "按钮权限 seed 三段式: 查父菜单(失败 WARN 跳过) → 幂等 count 检查 → 单条 db.Create;绝不批量 INSERT 也不授权 sys_role_menu"
  - "D-04/D-05 决策在 migration 注释中显式标注 (/* D-04 锁定 */) 防止后续维护者误以为漏了关联逻辑"

requirements-completed: [REQ-39-01, REQ-39-04]

# Metrics
duration: 5min
completed: 2026-06-25
---

# Phase 39 Plan 01: SysDeptLocationAlias 数据层 + 按钮权限 Seed

**sys_dept_location_alias 表结构 + 4 条 ops:location:alias:* 按钮权限,符合 D-04 不授权任何角色 + D-05 不修改 sys_workstation 双锁定决策**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-25T03:40:19Z
- **Completed:** 2026-06-25T03:45:40Z
- **Tasks:** 3
- **Files modified:** 3 (2 created + 1 modified)

## Accomplishments

- 创建 `SysDeptLocationAlias` GORM 模型(嵌入 BaseModel,4 业务字段: dept_id/location_id/scope/remark),`idx_location_id` 复合索引支持反向查询
- 编写 Migration 165:`sys_dept_location_alias` 表通过 GORM AutoMigrate 创建(PG/SQLite 双 DB 兼容);partial unique index on `(dept_id, scope) WHERE deleted_at IS NULL` 防软删除场景下重复映射
- Seed 4 条 `sys_menu` 按钮权限(menu_type='F', visible=0 隐藏):`ops:location:alias:{list,add,edit,delete}`,挂在"工位管理"菜单下,幂等(perms 已存在 skip)
- 严格遵守 D-04: **不** INSERT `sys_role_menu` 任何记录,管理员需手动授权
- 严格遵守 D-05: **不** UPDATE `sys_workstation` 任何记录,alias union 由 Plan 39-03 应用层实现
- `database.go` 在 Migration 164 之后、Migrate117 之前注册 Migration 165,日志风格一致

## Task Commits

Each task was committed atomically:

1. **Task 1: SysDeptLocationAlias GORM 模型** - `cc7c628` (feat)
2. **Task 2: Migration 165 — 建表 + 4 条 sys_menu 权限 seed** - `ba90ec4` (feat)
3. **Task 3: database.go 注册 migration 165** - `23457ae` (feat)

## Files Created/Modified

- `internal/models/sys_dept_location_alias.go` - GORM 模型,BaseModel 嵌入 + 4 字段,TableName() 返回 "sys_dept_location_alias"
- `internal/core/db/migrations/migration_165_sys_dept_location_alias.go` - 建表 + partial unique index + 4 按钮权限 seed,D-04/D-05 决策在注释中显式标注
- `internal/core/db/database.go` - 注册 Migrate165SysDeptLocationAlias 调用,位置在 Migrate164 之后、Migrate117 之前

## Decisions Made

- **AutoMigrate 而非原生 SQL 建表**: SQLite 不支持 PG 的 `gen_random_uuid()` + `UUID` 类型,使用 GORM `db.AutoMigrate(&models.SysDeptLocationAlias{})` 兼容双 DB,与 migration_162 同款模式
- **partial unique index 用原生 SQL 兜底**: GORM tag 不支持 `WHERE deleted_at IS NULL` 的 partial unique 语法,用 `db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS ...")` 实现,SQLite 不支持会 log 跳过(本项目生产 DB 为 PG)
- **`parent_menu` 缺失时 WARN + 跳过而非阻断**: 找不到"工位管理"菜单时记录 warning 并 continue,不 panic、不 fail-fast,允许迁移在初始化早期安全执行
- **按钮权限 visible=0**: 隐藏 — 不出现在菜单树中,但可被前端 `getAuthButtons` 查询和角色授权使用,与 migration_163 同款约定

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## Next Phase Readiness

- 数据层基础就绪,Plan 39-02 (alias CRUD API) 可直接基于 `SysDeptLocationAlias` 模型开发 5 个 POST 端点 (`/list`, `/add`, `/:id/update`, `/:id/delete`, `/export`)
- Plan 39-03 (工位部门下拉 union 注入) 可在工位导入/查询 service 中 LEFT JOIN `sys_dept_location_alias` 表
- 4 条新按钮权限 (`ops:location:alias:*`) 已 seed 但**未授权任何角色**,管理员需在"角色管理 → 菜单权限"页手动勾选授予目标角色
- 启动应用后,首次执行会自动创建 `sys_dept_location_alias` 表 + 写入 4 条 sys_menu 记录;幂等

---

*Phase: 39-workstation-dept-location-alias*
*Completed: 2026-06-25*

## Self-Check: PASSED

- Files created: `internal/models/sys_dept_location_alias.go`, `internal/core/db/migrations/migration_165_sys_dept_location_alias.go` (both exist)
- File modified: `internal/core/db/database.go` (registered Migrate165SysDeptLocationAlias)
- Commits verified: `cc7c628` (Task 1), `ba90ec4` (Task 2), `23457ae` (Task 3) — all present in `git log --oneline`
- `go build ./...` exit 0, `go vet ./...` 0 warnings
- D-04 compliance: 0 INSERT into sys_role_menu (grep verified — all 4 matches in comments)
- D-05 compliance: 0 UPDATE sys_workstation (grep verified)
- 4 perms present: `ops:location:alias:{list,add,edit,delete}`
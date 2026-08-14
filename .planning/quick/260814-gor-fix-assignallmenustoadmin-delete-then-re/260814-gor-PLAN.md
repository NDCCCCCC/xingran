---
phase: quick-260814-gor
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/permission/service.go
autonomous: true
requirements:
  - BUGFIX-ASSIGNALLMENUS-INCREMENTAL-IDEMPOTENT

must_haves:
  truths:
    - "每次启动（SkipSetup=false）调用 assignAllMenusToAdmin 时，不再执行 DELETE sys_role_menu WHERE role_id = admin 的全量删除"
    - "admin 已拥有全部现存菜单时（差集为空），assignAllMenusToAdmin 直接 return nil，不再触碰 sys_role_menu 表"
    - "admin 缺失部分菜单时，仅 INSERT 缺失项（差集），且整个补全过程包在单一事务中，失败回滚不丢已有权限"
    - "assignAllMenusToAdmin 的函数签名 (s *service, db *gorm.DB) error 不变，InitDefaultRolesAndMenus 调用点不变"
  artifacts:
    - path: "pkg/permission/service.go"
      provides: "assignAllMenusToAdmin 的增量幂等差集补全实现（Pluck id + 差集 + 事务批量 INSERT，无 DELETE）"
      contains: "CreateInBatches"
  key_links:
    - from: "pkg/permission/service.go (assignAllMenusToAdmin)"
      to: "sys_role_menu"
      via: "Pluck menu_id 查已有 → 差集 → CreateInBatches 仅补缺失（事务包裹）"
      pattern: "CreateInBatches"
    - from: "internal/core/core.go:302-315 (InitDefaultRolesAndMenus 调用点)"
      to: "pkg/permission/service.go (assignAllMenusToAdmin)"
      via: "permissionSvc.InitDefaultRolesAndMenus → s.assignAllMenusToAdmin（签名不变，调用形态不变）"
      pattern: "assignAllMenusToAdmin"
---

<objective>
修复 `pkg/permission/service.go:70-96 assignAllMenusToAdmin` 的"先删后全量重插"数据丢失风险，改为**增量幂等差集补全**。

**根因（已诊断，无需重新调查）**：当前实现每次启动（SkipSetup=false 时）都执行：
1. `db.Find(&allMenus)` —— `SELECT * FROM sys_menu` 全字段全表（dev 慢库 239 条曾耗时 2 分 27 秒触发 `unexpected EOF`，Supabase pooler 连接断开）
2. `db.Where("role_id = ?", adminRole.ID).Delete(&models.RoleMenu{})` —— **先删 admin 全部菜单关联**
3. 逐条 `db.Create(&roleMenu)` —— **再插回，无事务**

这是定时炸弹：下次若 SELECT 碰巧成功 → 先删 admin 全部菜单 → 逐条 INSERT；若 INSERT 中途网络抖动 = admin 丢权限。菜单导入（36→239）放大暴露面。

**修复**：只补缺失、不删除、加事务。

Purpose: 消除 admin 权限丢失风险 + 启动加速（admin 已齐备时秒过）。
Output: 1 个文件修改，零接口变更，向后兼容。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@D:/code/ClaudeCode/guoguo/CLAUDE.md
@D:/code/ClaudeCode/guoguo/.planning/STATE.md

# 修复现场（已诊断，无需重新调查）
@D:/code/ClaudeCode/guoguo/pkg/permission/service.go
@D:/code/ClaudeCode/guoguo/internal/models/role.go
@D:/code/ClaudeCode/guoguo/internal/models/menu.go
@D:/code/ClaudeCode/guoguo/internal/models/base.go

# 调用点（不改）
@D:/code/ClaudeCode/guoguo/internal/core/core.go

<interfaces>
<!-- 执行者直接使用以下契约，无需再探索代码库 -->

## 目标函数（当前实现，待替换）
`pkg/permission/service.go:69-96`
```go
// assignAllMenusToAdmin 为管理员分配所有菜单权限
func (s *service) assignAllMenusToAdmin(db *gorm.DB) error {
    var adminRole models.Role
    if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
        return err
    }
    var allMenus []models.Menu
    if err := db.Find(&allMenus).Error; err != nil {
        return err
    }
    // 清除现有的角色菜单关联
    db.Where("role_id = ?", adminRole.ID).Delete(&models.RoleMenu{})
    // 创建新的角色菜单关联
    for _, menu := range allMenus {
        roleMenu := models.RoleMenu{
            RoleID: adminRole.ID,
            MenuID: menu.ID,
        }
        if err := db.Create(&roleMenu).Error; err != nil {
            return err
        }
    }
    return nil
}
```

## 调用点（签名必须保持兼容）
- `pkg/permission/service.go:37` —— `InitDefaultRolesAndMenus` 内 `s.assignAllMenusToAdmin(db)`，签名 `(s *service, db *gorm.DB) error`。
- `internal/core/core.go:308-313` —— `permissionSvc.InitDefaultRolesAndMenus(c.GetDB())`，失败仅 WARN 不阻断启动（保持不变）。

## 关键约束事实（已核实）
- `internal/models/role.go:22-25` —— `RoleMenu{ RoleID string gorm:"type:uuid;not null"; MenuID string gorm:"type:uuid;not null" }`，**无 (role_id, menu_id) 唯一约束、无主键** → 不能用 `ON CONFLICT`，必须"预查已有 + 差集 + INSERT"。
- `internal/models/base.go:11-15` —— `BaseModel` 含 `DeletedAt gorm.DeletedAt gorm:"index"`，`Menu` 嵌入 BaseModel → 用 `db.Model(&models.Menu{}).Pluck("id", ...)` 时 GORM 自动追加 `deleted_at IS NULL`，**无需手写**。
- `internal/models/role.go:39-41` —— `RoleMenu` 表名 `sys_role_menu`。
- `pkg/permission/` 目录无 `service_test.go`（现有测试为 `config_v1201_test.go` 与 `resource_action_map_test.go`，均不覆盖 `assignAllMenusToAdmin`）→ **无既有测试需适配，不强加新测试**。

## 新逻辑伪代码
```
1. db.Where("role_key = ?", "admin").First(&adminRole)           // 查 admin role（保持）
2. db.Model(&models.Menu{}).Pluck("id", &allMenuIDs)             // 查全部菜单 id（只取 id 列，GORM 自动过滤软删）
3. db.Model(&models.RoleMenu{}).Where("role_id = ?", adminRole.ID).Pluck("menu_id", &existingIDs)  // admin 已有
4. existingSet = set(existingIDs)
   missing = [ {adminRole.ID, mid} for mid in allMenuIDs if mid not in existingSet ]
5. if len(missing) == 0 { return nil }                           // 幂等快速路径（启动加速）
6. return db.Transaction(func(tx) error {                        // 事务包裹补全
       return tx.CreateInBatches(missing, 100).Error             // 批量插，失败回滚不丢已有
   })
```

## 行为等价性说明（必写入 SUMMARY）
- 原语义："admin 拥有全部菜单"（靠每次启动 delete-all + re-insert-all 达成）。
- 新语义："admin 拥有全部菜单"（靠 diff-completion：只补缺失、永不删除）。
- **可接受差异**：不再清理"指向已删菜单的陈旧 role_menu 关联"。这是无害的：
  - 这些 menu_id 在 `sys_menu` 已被软删，`GetUserMenus` / `GetUserPermissions` 在 JOIN 时带 `m.deleted_at IS NULL` 过滤，陈旧关联在读取层被天然屏蔽，不会授予任何权限。
  - admin 的目标本就是"拥有全部现存菜单"，陈旧关联不影响此目标。
</interfaces>
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: 重写 assignAllMenusToAdmin 为增量幂等差集补全（Pluck id + 差集 + 事务批量 INSERT，无 DELETE）</name>
  <files>pkg/permission/service.go</files>
  <behavior>
    - admin 已拥有全部现存菜单时，函数直接 return nil，不执行任何 DELETE、不执行任何 INSERT（幂等快速路径）。
    - admin 缺失 N 条菜单时，仅 INSERT 这 N 条缺失项；已有 role_menu 行不动。
    - 整个补全（Pluck + 构造差集 + CreateInBatches）中，写操作包在单一 `db.Transaction` 内，失败回滚不丢已有权限。
    - 不再调用 `db.Where("role_id = ?", ...).Delete(&models.RoleMenu{})`。
    - 不再用 `db.Find(&allMenus)` 拉全字段菜单表，改用 `db.Model(&models.Menu{}).Pluck("id", &allMenuIDs)` 只取 id 列。
    - 函数签名 `func (s *service) assignAllMenusToAdmin(db *gorm.DB) error` 不变；`InitDefaultRolesAndMenus` 调用形态不变。
    - 不修改 `createDefaultAdminRole`、不改 `service.go` 其他方法、不改 `core.go` 调用点。
  </behavior>
  <action>
    替换 `pkg/permission/service.go` 中 `assignAllMenusToAdmin` 的函数体（当前 70-96 行），新实现严格遵循 `<interfaces>` 中的"新逻辑伪代码"：

    1. 保留查 admin role 的两行不动（`db.Where("role_key = ?", "admin").First(&adminRole)`）。
    2. 将 `var allMenus []models.Menu; db.Find(&allMenus)` 改为 `var allMenuIDs []string; db.Model(&models.Menu{}).Pluck("id", &allMenuIDs)` —— 仅取 id 列，GORM 自动追加 `deleted_at IS NULL`，无需手写过滤。
    3. 新增查 admin 已有关联：`var existingIDs []string; db.Model(&models.RoleMenu{}).Where("role_id = ?", adminRole.ID).Pluck("menu_id", &existingIDs)`。
    4. 在内存构造差集：把 `existingIDs` 放入 `map[string]struct{}`（预分配 `make(map[string]struct{}, len(existingIDs))`），遍历 `allMenuIDs`，凡不在 map 中的，追加到 `missing []models.RoleMenu`（每项 `{RoleID: adminRole.ID, MenuID: mid}`）。
    5. 幂等快速路径：`if len(missing) == 0 { return nil }`。
    6. 差集非空：`return db.Transaction(func(tx *gorm.DB) error { return tx.CreateInBatches(missing, 100).Error })`（批量大小 100，适合 239 条规模；事务保证失败回滚）。
    7. **删除**原实现中的 `db.Where("role_id = ?", adminRole.ID).Delete(&models.RoleMenu{})` 整行——这是本次修复的核心，消除丢权限风险。
    8. 函数注释可更新为说明新语义："增量幂等补全 admin 缺失菜单，不删除已有；差集为空秒过；陈旧关联（指向已删菜单）不清理，读取层天然屏蔽（见 SUMMARY 行为等价性说明）"。

    范围红线：不碰 `createDefaultAdminRole`、`InitDefaultRolesAndMenus`、`UpdateRoleMenus`、`GetUserMenus`、`core.go`；不改 imports（新逻辑只用到已有的 `gorm` 与 `models`）；不新增/删除测试文件（`pkg/permission/` 无 `service_test.go`，按约定不强加）。
  </action>
  <verify>
    <automated>cd D:/code/ClaudeCode/guoguo && go build ./... && go test ./pkg/permission/...</automated>
  </verify>
  <done>
    - `go build ./...` exit 0
    - `go test ./pkg/permission/...` 既有测试（config_v1201_test、resource_action_map_test）不回归
    - `pkg/permission/service.go` 中 `assignAllMenusToAdmin` 函数体：
      - 出现 `Pluck("id", &allMenuIDs)` 与 `Pluck("menu_id", &existingIDs)`（grep 命中两条 Pluck）
      - 出现 `CreateInBatches(missing`（grep 命中）
      - 出现 `db.Transaction(`（grep 命中）
      - **不再出现** `.Delete(&models.RoleMenu{})`（grep `Delete\(&models.RoleMenu{}` 在本函数范围内不命中）
      - **不再出现** `db.Find(&allMenus)`（grep `Find\(&allMenus` 不命中）
    - 函数签名仍为 `func (s *service) assignAllMenusToAdmin(db *gorm.DB) error`
    - `InitDefaultRolesAndMenus` 内 `s.assignAllMenusToAdmin(db)` 调用形态未变
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 启动引导 → sys_role_menu | 后端启动（SkipSetup=false）时由 `InitDefaultRolesAndMenus` 触发；本任务只改写操作策略（delete+reinsert → diff-only insert），不改鉴权与触发条件 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-gor-01 | Tampering / Elevation of Privilege | assignAllMenusToAdmin（旧：无事务先删后插） | mitigate | 改为增量差集补全 + 单事务包裹；任何中断要么完整补全、要么回滚，admin 已有权限不再被暴露在"已删未插"的中间窗口 |
| T-quick-gor-02 | Denial of Service | 启动流程（原 SELECT * FROM sys_menu 全表） | mitigate | 改用 `Pluck("id")` 仅取 id 列，大幅减少数据量与耗时；admin 已齐备时差集为空秒过，启动加速 |
| T-quick-gor-03 | Information Disclosure / 权限残留 | 陈旧 role_menu 关联（指向已软删菜单） | accept | 新逻辑不清理陈旧关联；这些 menu_id 在 sys_menu 已软删，`GetUserMenus`/`GetUserPermissions` 的 JOIN 带 `deleted_at IS NULL` 在读取层天然屏蔽，不授予任何权限。属可接受 tradeoff，已在 `<interfaces>` 行为等价性说明中记录 |

注：本计划无 npm/pip/cargo 包安装，T-{phase}-SC（供应链）不适用。
</threat_model>

<verification>
- `go build ./...` 全量编译通过（强制门）
- `go test ./pkg/permission/...` 既有测试不回归
- 静态确认（grep）：`assignAllMenusToAdmin` 函数体内不再有 `Delete(&models.RoleMenu{})` 与 `Find(&allMenus)`，含 `Pluck`、`CreateInBatches`、`db.Transaction`
- 可选运行验证（dev 环境可用时）：启动后端，日志出现"默认角色和菜单初始化成功"；连续二次启动后查 `SELECT count(*) FROM sys_role_menu WHERE role_id = <admin>` 与 `SELECT count(DISTINCT menu_id) FROM sys_role_menu WHERE role_id = <admin>` 应相等且等于 sys_menu 现存菜单数（无重复、无丢失）
</verification>

<success_criteria>
- `assignAllMenusToAdmin` 改为增量幂等差集补全：差集为空秒过、差集非空事务批量补全、永不 DELETE 已有 admin 菜单关联。
- admin 权限丢失的定时炸弹（先删后插无事务）被消除。
- 启动加速：admin 已齐备时不再触碰 sys_role_menu 表；菜单查询从 `SELECT *` 降为 `Pluck id`。
- 单文件、零接口变更、向后兼容；`go build ./...` 与 `go test ./pkg/permission/...` 全绿。
- 未触碰范围外代码：`createDefaultAdminRole`、`InitDefaultRolesAndMenus`、`UpdateRoleMenus`、`GetUserMenus`、`core.go` 调用点均未改。
- 行为等价性差异（陈旧关联不清理）显式记入 SUMMARY。
</success_criteria>

<output>
Create `.planning/quick/260814-gor-fix-assignallmenustoadmin-delete-then-re/260814-gor-SUMMARY.md` when done.

**必写入 SUMMARY 的项**：
1. 行为等价性说明（见 `<interfaces>` 末尾）：新逻辑不再清理指向已软删菜单的陈旧 role_menu 关联，读取层天然屏蔽，无害。
2. 启动性能改善：admin 已齐备时秒过；菜单查询从 `SELECT *` 降为 `Pluck id`。
3. 数据完整性保证：写操作包在单事务内，失败回滚不丢已有权限。
4. 验证记录：`go build` / `go test ./pkg/permission/...` 结果；若做了 dev 库二次启动验证，附 count 对比。
</output>

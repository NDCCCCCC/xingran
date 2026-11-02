# GORM Schema 漂移登记 — sys_role_menu / sys_user_role（Phase 84 PMODEL-04）

> 状态：**显式债务登记，只登记不补列**（Phase 84 裁定：不给这两张表补模型列）。
> 来源：`.planning/phases/84-pmodel-gaps/84-RESEARCH.md` §D2。
> 前提说明：库表实际列集**待实机 `\d sys_role_menu` / `\d sys_user_role` 确认**（研究置信度 LOW→HIGH）。
> 机制背景（RESEARCH §D2 HIGH 推断）：GORM AutoMigrate 按**模型**补列——模型没声明的列它
> 不会建也不会删；模型未映射的既有列会与模型「静默并存」，不报错，但 `SELECT *`→struct
> 扫描、审计导出、未来加字段都可能踩坑。

---

## 条目 1：`sys_role_menu`

- **模型**：`internal/models/role.go:24-27`（`RoleMenu`，`TableName()` 在 :41-43）
- **现状**：模型仅 `RoleID` + `MenuID` 两列（复合索引 tag `idx_sys_role_menu_role_id_menu_id`）。
  - **缺 `id` 主键映射**：模型无 ID 字段。
  - **缺 `created_at` 映射**：模型无 CreatedAt 字段。
- **库表多出列（待实机确认）**：若表由 SQL 迁移/手工建出 `id`、`created_at` 列，则模型
  未映射这两列；若表完全由该模型 AutoMigrate 建出，则表里根本没有 `id`/`created_at`。
- **现状靠 DB default**：`id` 若存在依赖 DB default / 序列；`created_at` 若存在依赖 DB
  default（如 `now()`）。GORM 写入路径不提供这两列的值。
- **风险**：`SELECT *`（含 `db.Raw("SELECT * FROM sys_role_menu")`、部分 ORM 调试路径）
  时多余列被 Scan 忽略——目前无实际报错；未来给 RoleMenu 加字段或改主键策略时，
  `id` 的存在与否会导致 AutoMigrate 行为分叉；审计/导出工具若假设每表有 `id`/`created_at`
  会拿到零值。
- **建议动作**：**显式声明不动**（登记为已知漂移）。理由：关联表行集永远通过
  `(role_id, menu_id)` 访问，无逐行主键访问路径；补 `id` 列属于 schema 变更，超出
  Phase 84 边界。若未来实机确认表确有 `id`/`created_at` 且有业务需要，再单独立项补映射。

## 条目 2：`sys_user_role`

- **模型**：`internal/models/user.go:51-55`（`UserRole`，`TableName()` 见同文件）
- **现状**：模型有 `UserID` + `RoleID` + `CreatedAt time.Time`——
  - **有 CreatedAt 但无 gorm tag**：无 `autoCreateTime`/`default:now()` tag，AutoMigrate
    会按 Go 类型建列，但 GORM 写入路径（`db.Create`）是否填值取决于 GORM 对 `CreatedAt`
    约定字段的自动填充（约定名命中时会填；显式 SQL INSERT 绕过时靠 DB default）。
  - **缺 `id` 主键映射**：模型无 ID 字段。
- **库表多出列（待实机确认）**：`id`（若表由 SQL 手建）；`created_at` 已由模型映射，
  但**写路径覆盖不全**。
- **现状靠 DB default**：`id` 依赖 DB default；`created_at` 在非 GORM 写入路径（如
  `db.Exec("INSERT ...")`）依赖 DB default，可能落零值/NULL。
- **风险**：与条目 1 同类——静默并存，无启动报错；`id` 缺映射导致未来主键策略变更时
  AutoMigrate 行为分叉；`CreatedAt` 无 tag 导致跨写路径时间戳不一致（部分行有时间、
  部分行为零值），影响按时间审计关联授权变更的可信度。
- **建议动作**：**显式声明不动**（登记为已知漂移）。理由同上：关联表按
  `(user_id, role_id)` 访问；补 `id` 列与 CreatedAt tag 语义固化属 schema/model 变更，
  超 Phase 84 边界，需单独立项。

---

## 登记汇总

| 表 | 模型位置 | 缺失映射 | 处置 |
|----|----------|----------|------|
| `sys_role_menu` | `internal/models/role.go:24-27` | `id`、`created_at` | 显式不动，登记债务 |
| `sys_user_role` | `internal/models/user.go:51-55` | `id`（`created_at` 已映射但无 tag、跨写路径覆盖不全） | 显式不动，登记债务 |

- 库表实际列集确认动作：实机 `\d sys_role_menu` / `\d sys_user_role`（PG），确认后把
  本 note 置信度 LOW→HIGH。
- 明确不做：补列、补映射、修 sqlite ON CONFLICT 方言（Future 条目）。

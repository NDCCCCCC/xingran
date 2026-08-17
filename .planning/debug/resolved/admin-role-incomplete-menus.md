---
status: resolved
trigger: |
  admin 只有运维管理菜单权限，之前在 Supabase(PG) 上也是这样。admin 作为超级用户应该拥有所有权限。用户要求彻底修复。
created: 2026-08-17
updated: 2026-08-17
resolved: 2026-08-17
---

# Debug: admin-role-incomplete-menus

## Symptoms

- **Expected**: admin（超级管理员）登录后拥有系统全部菜单与按钮权限
- **Actual**: admin 仅有"运维管理"菜单（楼宇/楼层/工位/信息点/机房/专线/机房设备及其按钮）
- **Timeline**: 非本次 sqlite 切换引入 — Supabase(PG) 时期即如此，属长期存在的数据/种子缺陷
- **Reproduction**: admin 登录（任意库），观察左侧菜单只有运维管理一组
- **线索**: sqlite 全新库启动日志显示菜单种子只创建了"运维管理菜单"一族（internal/core/db/init_data.go 或迁移）；需排查：
  1. 菜单种子的单一事实源在哪（init_data.go？归档 SQL？迁移？）——为何只种子了运维管理一族，系统管理（用户/角色/部门/菜单/字典…）等菜单是否存在于某处但未种子/未授权
  2. admin 角色的菜单授权逻辑：是显式 role-menu 关联还是超级管理员隐式全量？若是显式关联，种子时给 admin 授了哪些
  3. "彻底修复"语义：修复后应有机制保证 admin（或超级管理员角色）始终拥有全部菜单（如种子时全量授权 + 新菜单种子时自动授权 admin），而非一次性补数据
  4. 注意 PG 生产库兼容：若修复涉及种子/迁移逻辑变更，需确保对已存在数据的 PG 库幂等（已存在则跳过/补齐缺失授权），不能只在 sqlite 分支生效
  5. 前端菜单由 /system/menu/router 或类似接口按角色下发，确认修复后下发链路无需改动

## Current Focus

resolved: |
  人工验证通过（2026-08-17）：用户重启后端 + admin 重新登录，确认菜单全量下发。
  会话归档。

## Evidence (fix phase)

- timestamp: 2026-08-17 (fix)
  checked: Migrate207 单元测试 (sqlite 内存库, 4 用例)
  found: 全新库种子 239 条/10 顶级根/无悬空 parent/幂等重跑; 尊重用户删除(不复活);
         遗留 36 条子树软删归并+授权清理; 授权闭环 admin=239
  implication: 修复逻辑正确且幂等

- timestamp: 2026-08-17 (fix)
  checked: 测试暴露 MenuMeta.Scan 在 sqlite 上 "type assertion to []byte failed"
  found: glebarez/modernc 对 jsonb(TEXT) 列返回 string 而非 []byte; Go 种子从不写 meta 故缺陷潜伏至今
  implication: 潜伏的双方言缺陷, 已修复 (Scan 兼容 string/[]byte/空)

- timestamp: 2026-08-17 (verify)
  checked: 真实库 data/xingran.db 启动后端实测
  found: 日志 "软删归并遗留运维管理子树 36 条" + "规范菜单目录种子完成 (239 条 INSERT, 幂等)";
         sys_menu 存活=239, 顶级 10 目录齐全; admin sys_role_menu=239; 遗留 36 条已软删(可恢复)
  implication: 真实环境修复生效

- timestamp: 2026-08-17 (verify)
  checked: GetUserMenus 等价 SQL (admin 用户→角色→授权→可见正常菜单)
  found: admin 可见 191 条, 顶级可见 9 族 (用户中心 visible=0 隐藏为生产数据原样, 设计如此)
  implication: 读取链路无需改动, 数据就位后自动下发全量菜单

- timestamp: 2026-08-17 (verify)
  checked: go build ./... + go test ./internal/core/db/... ./internal/models/... ./pkg/permission/... + 全量 go test ./...
  found: 全部通过; 唯一失败 TestADLoginWithOUProcessing 经 stash 对照实验证明为预存失败(与本次改动无关)
  implication: 无回归

- timestamp: 2026-08-17 (human-verify)
  checked: 用户重启后端 + admin 重新登录前端
  found: 菜单全量下发, 用户确认修复生效
  implication: RESOLVED

## Resolution

root_cause: |
  全新安装的库中 sys_menu 只种子了「运维管理」一族(init_data.go createOperationsManagementMenus, 36 条)。
  完整 239 条菜单目录只存在于 archive SQL(不执行)与一次性手工导入产物 xingran_menus_dedup.sql;
  历史 Go 菜单迁移(018/029/017/013 等)按 database.go:585-606 设计不再启动期调用。
  因此任何新建库(PG 或 sqlite)都只有运维管理菜单可授 —— admin 授权机制
  (assignAllMenusToAdmin, 每次启动增量补全全部现存菜单)本身无缺陷, 是目录缺失。
  次生缺陷: models.MenuMeta.Scan 只断言 []byte, sqlite 驱动返回 string,
  一旦菜单 meta 非空(规范目录全部带 meta)读取即 Scan error —— 已一并修复。

fix: |
  1) 新增 migration_207_menu_catalog_seed.go + menu_catalog_seed.sql (go:embed):
     239 条规范菜单目录(源自 quick-260814-ehg 生产去重保留集, dev 五项复核 PASS 的同一数据),
     启动期幂等种子, ON CONFLICT DO NOTHING, 双方言(PG+sqlite, 时间戳剥离 +00);
     规范根存在即整体跳过(尊重用户删除); 慢速路径先软删归并 init_data.go 遗留的
     非规范「运维管理」子树并清理其 sys_role_menu 关联。
  2) database.go AutoMigrate 挂钩: PG 走 advisory-lock 块, sqlite 走 else 分支 —
     均不受 SkipSetup 门控, 且先于 initData 的 createOperationsManagementMenus
     (规范根就位后 Go 种子按 (name,parent) 查重自然全跳过)。
  3) models/menu.go MenuMeta.Scan 兼容 string/[]byte/空 (sqlite 双方言修复)。
  机制闭环: 目录种子(207) + assignAllMenusToAdmin(每次启动增量补全) =
  admin 始终拥有全部现存菜单, 未来新菜单下次启动自动授予 admin。

verification: |
  - 4 个 sqlite 单元测试全 PASS (新鲜种子/幂等/尊重删除/遗留归并/授权闭环)
  - 真实 sqlite 库启动实测: 239 存活菜单, 10 顶级目录, admin 授权 239, 遗留 36 软删
  - GetUserMenus 等价 SQL: admin 可见 191 条, 9 个可见顶级族
  - go build ./... 通过; 相关包测试全过; 全量测试唯一失败为预存问题(stash 对照证明)
  - PG 路径: 代码审查 — 规范根已存在的库零影响跳过; 缺失的库走同一修复路径
  - 人工验证: 重启后端 + admin 重新登录, 用户确认菜单全量下发 (2026-08-17)

files_changed:
  - internal/core/db/migrations/migration_207_menu_catalog_seed.go (新增)
  - internal/core/db/migrations/menu_catalog_seed.sql (新增, go:embed 内嵌)
  - internal/core/db/migrations/migration_207_menu_catalog_seed_test.go (新增)
  - internal/core/db/database.go (AutoMigrate 双方言挂钩 Migrate207)
  - internal/models/menu.go (MenuMeta.Scan sqlite 兼容修复)
  - data/xingran.db (已迁移; 备份 data/xingran.db.bak-260817-menus)

## Evidence

- timestamp: 2026-08-17 (investigation)
  checked: sqlite 实库 data/xingran.db 直接 SQL 查询
  found: sys_menu 存活=36, 顶级目录仅「运维管理」(M); 软删=0; admin(sys_role role_key='admin') 的 sys_role_menu=36
  implication: 授权机制与现存菜单一一对应, 缺陷在菜单目录本身缺失而非授权

- timestamp: 2026-08-17 (investigation)
  checked: internal/core/db/init_data.go 全文
  found: initData() 种子序列 = 部门/用户/角色/用户角色/网络参数/定时任务/验证码参数/运维管理菜单/加密开关/AD配置 — 菜单仅此一族
  implication: 全新库菜单种子的 Go 侧唯一事实源只覆盖运维管理

- timestamp: 2026-08-17 (investigation)
  checked: internal/core/db/database.go:585-606 + AutoMigrate 迁移块(710-744)
  found: 注释明确"所有迁移(Migrate033/117/143~201)都是 schema 演进期一次性脚本...仅不再启动期调用"; 启动期仅跑 175/176/202-206 且包在 `if d.Type=="postgres"`  advisory-lock 块内 (sqlite 永不执行)
  implication: 历史菜单迁移(018 duty/029 building spaces/017 AD/013 workorder 等)对新库不生效

- timestamp: 2026-08-17 (investigation)
  checked: internal/core/db/migrations/archive/legacy-2026-06-15/init_data.sql + xingran_menus_dedup.sql
  found: 完整菜单目录只存在于归档 SQL(不执行)与一次性导入产物 xingran_menus_dedup.sql(239 条 ON CONFLICT DO NOTHING, 含规范 id: 系统管理=d67f4240-f887-481a-b345-94fb36782500, 运维管理=c50b5b01-fdbc-4821-a3e8-2e2178339b20)
  implication: 权威目录未固化进启动链路, 是长期缺陷的单一根源(PG 时期同样如此)

- timestamp: 2026-08-17 (investigation)
  checked: pkg/permission/service.go assignAllMenusToAdmin + core.go:314-327 调用点
  found: 每次启动(SkipSetup=false 时)增量幂全补全 admin 全部现存菜单(差集事务 INSERT); quick-260814-gor 已修复其 delete-then-reinsert 缺陷
  implication: "admin 始终拥有全部权限"的授权机制已存在, 缺的是"全部菜单存在于库中"

- timestamp: 2026-08-17 (investigation)
  checked: 菜单读取链路 menu_router.go:65 → menu_handler.GetUserMenus → menuService.GetUserMenus (menu_service.go:397)
  found: 按 user→role→role_menu→menu 关联查询, 无 admin 短路, 无其它过滤缺陷
  implication: 读取链路无需改动; 菜单存在+授权后自动下发

- timestamp: 2026-08-17 (investigation)
  checked: sqlite schema (sys_menu 19 列齐全可空, PK id; sys_role_menu 两列) + go.mod (glebarez/sqlite v1.11)
  found: dedup SQL 列集与 sqlite schema 完全匹配; ON CONFLICT DO NOTHING 双方言支持; 时间戳 '+00' 后缀对 glebarez 解析有风险需规范化
  implication: SQL 嵌入方案可行, 需剥离时区后缀

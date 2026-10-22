---
status: complete
phase: 59-ad-login-guard-reconciliation
source: [59-VERIFICATION.md]
started: 2026-10-20T01:20:00Z
updated: 2026-10-21T01:44:00.000Z
---

## Current Test

[testing complete]

## Tests

### 1. 真实 AD 登录守卫拦截（GUARD-01 端到端）
expected: 管理员编辑用户 dept → 等队列入队（pending/failed）→ 该用户 AD 登录 → 日志出现『部门回写被活跃队列守卫拦截』WARN（含 userID/username/本地 dept UUID/AD 解析 dept UUID/队列行 ID/action_type/attempts），且 sys_user.dept_id 未被 AD 旧值覆盖
result: [pass]

> **2026-10-22 生产端到端实测（chenchao-076，登录域 .39 实例，GUARD-01 真实登录同步路径）：**
> - 布现场：本地 dept_id 临时改 分公司本部（≠AD 解析值=基础运维科，制造漂移）+ 插入 pending update_user 行 aaaa1111-…ba11。
> - **守卫半段**：用户真实登录 → 登录同步执行（sys_user.updated_at 10:41:06 刷新）→ dept_id **保持分公司本部**未被 AD 解析值覆盖，行仍 pending → GUARD-01 拦截生效（代码路径 user_sync_service.go:476-494 findActiveUserUpdateQueue 命中分支）。
> - **对照半段**：行置 pushed（活跃谓词不命中）→ 再次真实登录 → dept_id 10:47:01 被 AD 解析值正常改回基础运维科 → 无活跃行时走 D-08 正常覆盖分支。
> - 清理：测试行已删，chenchao-076 dept 与 AD 实际位置一致，环境零残留。
reason: 用户跳过

### 2. 真实 PG 启动冒烟：migration_217 视图 + migration_218 权限 seed + 二次重启幂等
expected: 目标 PG 环境启动服务：AutoMigrate 前 DROP 视图、Migrate217 无错、SELECT count(*) FROM v_dept_ou_mismatch 可执行；再次重启同库无 SQLSTATE 0A000『cannot alter type』且视图重建成功；启动日志无『对账查询按钮权限 seed 失败』
result: pass
verified: 在 WSL PostgreSQL 14 隔离实例（全新库 ruoyi_next）上真实启动 embed 二进制，共 6 轮启动验证。run3/run4：migration_217 建视图成功、11 列齐全、SELECT count(*) 可执行、二次重启无 SQLSTATE 0A000 无「cannot alter type」、视图重建成功。run5/run6：migration_218 输出「Migration 218 completed: reconciliation button permission ready」（不再降级跳过）、F 型「对账查询」行 perms=ops:ad:config:reconciliation:list 恰好 1 条、精准授权 1 个角色、重跑无重复。另验证 HTTP 层：/ → 302 /xingran/、/xingran/ → 200、/xingran/api/v1/ad-domain/reconciliation/mismatch → 401（命中路由非 404）。
issues_found: 2（见 Gaps G-59-2 / G-59-3，均已在本轮修复并复验）

### 3. 对账 Tab 浏览器权限渲染（授权与未授权两态）
expected: 管理员为角色授予 ops:ad:config:reconciliation:list → 该角色登录后对账 Tab 出数据；未授权角色登录后 Tab 内渲染『无对账查看权限』403 且不发对账数据请求；直接 POST /api/v1/ad-domain/reconciliation/mismatch 被后端 RequirePermissions 拒为 403
result: pass
verified: 用户实测确认（2026-10-20 11:42 部署 11:17 二进制后）—— 授权角色登录后对账 Tab 出数据（真实生产数据渲染成功）；权限树「域控配置」下出现「对账查询」按钮（Test 4 同轮确认）

### 4. 生产库父菜单『AD域控配置』（menu_type='C'）存在性确认
expected: 生产 sys_menu 存在该父菜单 → migration_218 自动建 F 行并精准授权；若缺失 → 管理员需手建按钮权限 ops:ad:config:reconciliation:list 并为相关角色授权（迁移已按 warning + nil 降级，不会启动失败）
result: pass
verified: 生产库只读探针（2026-10-20）证实父菜单实为 menu_name='域控配置'/perms='ops:ad:config:list'（规划假设的『AD域控配置』不存在，G-59-3）；部署修复版二进制后 migration_218 按 perms 命中，用户实测权限树出现「对账查询」按钮（F 行已建并精准授权 2 个关联父菜单的角色）

### 5. GUARD-01/GUARD-02 验收口径人工复核（FLAGGED ASSUMPTION 闭环）
expected: 确认 ROADMAP SC1~SC4 原文即权威验收口径（四个 PLAN frontmatter 的 edge probe 均报 unclassified/unresolved，未假设 backstop）
result: pass
verified: 用户确认 SC1~SC4 即权威验收口径。证据映射：SC1=单测 6/6（Test 1 E2E 用户跳过）；SC2=同套单测放行分支；SC3=Test 2 pass + 生产真实数据 2031+66 行；SC4=Test 3 pass + 401/403 冒烟

### 6. 空态标题字重 600（computed style）
expected: 浏览器打开 AD 配置页 → 对账 Tab → 将筛选调成 0 结果 → DevTools 选中空态标题，确认 computed font-weight: 600、font-size: 16px
result: pass
verified: chrome-devtools 真实浏览器实测（2026-10-20，本地隔离环境 admin 登录 → AD域控 → 域控配置 → 对账 Tab）。空态标题「未发现不一致条目」computed style: font-weight=600, font-size=16px —— ReconciliationTab.css 的 !important 规则在真实层叠中生效（本地库 0 条数据,空态直接渲染,无需调筛选）。截图: 本机 %TEMP%/uat59/reconciliation-empty-state.png

### 7. Tab 栏不溢出视觉回归（backstop）
expected: 两个 Tab 标签不换行/不截断
result: pass
verified: chrome-devtools 实测两个视口 —— 1034px 宽屏: 4 个 Tab（页级 仪表盘/AD配置管理 + 内层 配置列表/对账）scrollWidth==clientWidth 零裁剪、高度 40px/22px 单行、nav 无溢出; 480px 窄屏模拟: 全部 Tab 依旧零裁剪零换行（navList 总宽仅 196px, nowrap）

## Summary

total: 7
passed: 6
issues: 0
pending: 0
skipped: 1
blocked: 0

## Gaps

- gap_id: G-59-2
  truth: "migration_217 视图在全新库上能建成（sys_dept_ou_mapping 等被引用表已存在）"
  failure_status: failed
  reason: "User reported: 实测 migration 217 失败 —— ERROR: relation \"sys_dept_ou_mapping\" does not exist (SQLSTATE 42P01)。该表由归档 .sql（archive/legacy-2026-06-15/126_create_dept_ou_mapping_table.sql）创建，而 .sql 迁移不被自动加载；常规启动路径（AutoMigrate / BootstrapMissingTables）从未注册 models.DeptOUMapping。生产库因当年手工执行过该 SQL 而有表，故仅全新库暴露。scripts/dbprovision 虽建该模型，但那是新部署专用工具，常规启动不跑。"
  severity: major
  test: 2
  root_cause: "models.DeptOUMapping 未注册进 AutoMigrate 模型列表（internal/core/db/database.go）；其建表 DDL 只存在于归档 .sql 文件与 scripts/dbprovision，而 .sql 不会被自动加载 —— 全新库走常规启动路径必然缺表 → migration_217 的 CREATE VIEW 引用该表报 42P01 → 视图建不出来 → 对账端点 500。"
  artifacts:
    - path: "internal/core/db/database.go"
      issue: "AutoMigrate 模型列表缺 &models.DeptOUMapping{}"
    - path: "internal/core/db/migrations/archive/legacy-2026-06-15/126_create_dept_ou_mapping_table.sql"
      issue: "建表 DDL 唯一来源是被归档的 .sql，不会被自动加载"
  missing:
    - "在 AutoMigrate 注册 models.DeptOUMapping（PG + sqlite 双分支），或新增显式 migration 建表"
  debug_session: ".planning/phases/59-ad-login-guard-reconciliation/59-UAT.md (Test 2)"
  status: resolved
  resolved_by: 11c26fbd
  resolved_at: 2026-10-20

- gap_id: G-59-3
  truth: "migration_218 能在生产库上找到父菜单并创建 + 授权对账按钮权限"
  failure_status: failed
  reason: "User reported: 实测 migration 218 恒降级跳过 —— 「Warning: 未找到『AD域控配置』父菜单, migration_218 跳过创建与授权」。该菜单名在 seed 与归档 SQL 中都不存在；库中真实父菜单是 menu_name='域控配置'、perms='ops:ad:config:list'（其下已有 5 个 F 型按钮）。规划时的 FLAGGED ASSUMPTION 未成立，且此缺陷在生产库同样存在（不限于全新库）→ 对账按钮权限永远建不出来 → 普通角色对账 Tab 恒 403。"
  severity: blocker
  test: 2
  root_cause: "migration_218 按 menu_name='AD域控配置' 查找父菜单，该名字在 init_data seed 与全部归档 SQL 中均不存在（真实名字为『域控配置』）；查不到即 return nil 静默降级，按钮权限行永不创建。同款假设亦存在于 migration_163（其按钮权限在实测库中同样不存在）。"
  artifacts:
    - path: "internal/core/db/migrations/migration_218_ad_reconciliation_permission.go"
      issue: "父菜单查找用 menu_name='AD域控配置'，与库中真实 menu_name='域控配置' 不符"
    - path: "internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go"
      issue: "测试 fixture 造的是假菜单名『AD域控配置』且无 perms，掩盖了真实缺陷"
  missing:
    - "改为按 perms='ops:ad:config:list' + menu_type='C' 定位父菜单"
    - "同步修正测试 fixture 为真实形态（menu_name='域控配置' + perms）"
  debug_session: ".planning/phases/59-ad-login-guard-reconciliation/59-UAT.md (Test 2)"
  status: resolved
  resolved_by: 11c26fbd
  resolved_at: 2026-10-20

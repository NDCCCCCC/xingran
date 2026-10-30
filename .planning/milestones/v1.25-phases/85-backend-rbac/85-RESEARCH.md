---
phase: 85
title: 后端动作级严格化 技术方案研究
date: 2026-10-28
researcher: gsd-phase-researcher（重派，复用前任 tmp_routes/tmp_mounts/tmp_ftype 盘点数据）
requirements: RBAC-01, RBAC-03, RBAC-04
confidence_marks: HIGH=读代码/数据实证(附 文件:行号) / MEDIUM=推导 / LOW=需实机验证
---

# Phase 85 研究报告：后端动作级严格化

> 只读研究产物。行号为 2026-10-28 工作区快照。复用前任遗留 `tmp_routes.json` / `tmp_mounts.json` / `tmp_ftype.json` / `tmp_all_routes.json`（保留于仓库根，勿删）。

## 现状复核

### 0. tmp_routes.json 覆盖完整性验证（HIGH，含重大缺口声明）

复用前先做抽查验证。**发现 tmp_routes.json 存在一处系统性缺口**：

- 扫描器口径：659 条路由、232 条受权限保护、84 个中间件挂载点（`RequirePermissions`×75 / `RequirePermissionsWithQuery`×8 / `RequireStrictPermissions`×1）、34 个受保护挂载组。
- **缺口（HIGH）**：`system` 域 4 组（roles/menus/departments/posts）在 tmp_routes.json 中为 **0 条**——这 4 组的组变量（`roles := authorized.Group("/roles")` 等，`internal/api/router.go:231/247/260/279`）虽是标准形态，但前任扫描产物未收录。本研究已**直接读源码补齐**（见 §C），结论不受影响；但**全站拆分阶段执行时不可把 tmp_routes.json 当完备事实源**，W2 每域动手前须重新扫描该域源码（重跑 `tmp_scan_final.cjs` 即可）。
- 抽查 10 处挂载点（router.go:590/618/658/735/763/800/826/860/886/937 + network_router.go:63/84/105/127/159/196 + ad_domain_router.go:27/63/74/105）：permKind / permCodes / gprefix 与源码逐一相符（HIGH）。
- `tmp_all_routes.json` 中 411 条"unprotected"路由**不能直接当真**（HIGH）：其中相当部分是 `r.POST("...", middleware.X(...), core.Handler)` 内联中间件形态（handler 名提取失败显示 `-> core`，如 captcha_background_handler.go:21-29），以及鉴权根路径组（auth/captcha）。W2 扫描脚本需补内联中间件解析。

### A. 死表现状（RBAC-03）

**A1. `GetRoutePermissions()` 全表登记（HIGH）** — `pkg/permission/config.go:224-290`，共 **49 条**：

| 域 | 条数 | 路径前缀 |
|---|---|---|
| 用户 | 10 | /system/users/*（与 78 已落地严格组一一对应） |
| 角色 | 10 | /system/roles/* |
| 菜单 | 5 | /system/menus/* |
| 部门 | 5 | /system/depts/* |
| 岗位 | 5 | /system/posts/* |
| 工位 | 6 | /system/workstations/* |
| 网络端口写 | 8 | /network/ports/write/*（Phase 52/56） |

**A2. 匹配语义（HIGH）** — `GetPermissionByPath`（config.go:293-302）+ `matchPath`（:305-330）：**精确串相等 或 段数一致逐段比较（`:id` 占位段跳过比较）**。支持 `:id`；**不支持前缀/通配**；不区分 trailing slash。方法必须完全相等（注意实际代码大量使用 GET/PUT/DELETE，而全站惯例是 POST——死表内全为 POST）。

**A3. ⚠️ 死表已与实际路由漂移（HIGH，关键发现）** — 即使对表内已登记的域，路径也对不上现行路由：

| 死表登记 | 实际路由（源码） | 漂移 |
|---|---|---|
| `/system/roles/batch-delete` | `POST /system/roles/batch`（role_router.go:33） | 路径不同 |
| `/system/depts/*` | 组前缀是 **`/departments`**（router.go:260），批量删除是 `/batch`（department_router.go:53） | 前缀+路径双漂移 |
| `/system/menus/:id` 表登记为 View | 实际 `POST /system/menus/:id` = GetByID（menu_router.go:36）语义一致，OK | — |
| 岗位批量 `/system/posts/batch-delete`? 表未登记 | 实际 `POST /system/posts/batch-delete`（post_router.go:35） | 缺登记 |

⇒ 死表**从未被消费也从未与路由对齐**（RBAC-03 的"死表"定性成立），接线前必须先按实际路由**重建**角色/菜单/部门 3 个域的登记。

**A4. 消费点现状（HIGH）**：全仓仅 `GetPermissionByPath` 定义处一个函数，无调用点（死表）；`PermissionCode` 常量集被 router 挂载点、`resource_action_map.go` 与前端 `permissionCodes.ts` 引用（常量集不是死的，映射表是死的）。

### B. ANY-of 现状（RBAC-01）

**B1. 规模统计（MEDIUM，基于 tmp_routes 口径，见 §0 缺口）**：

- 受保护挂载组 34 个，其中**ANY-of 混合读写组 29 个**（`RequirePermissions`×21 + `RequirePermissionsWithQuery`×8），涉及 **91 条写路由**（isWrite 按 handler 名正则判定，`tmp_scan_final.cjs` WRITE_HANDLER）。
- 补上 tmp_routes 缺失的 system 4 组（源码实数）：roles 5 写 + menus 5 写 + departments 5 写(+Excel 导入导出) + posts 4 写 ≈ **+19~21 条** → 全站需拆写路由约 **110~112 条**，分布在约 **33 个挂载组**。
- ROADMAP 写"133 处 ANY-of"（ROADMAP.md:69/184）：按"挂载点下全部路由条数（含读）"口径可达 133+；按"需改写的写路由"口径是 ~110。执行时以**源码重扫**为准，两口径都对不齐时以写路由清单为准（MEDIUM）。
- 纯读组 5 个、已严格组（users 写组）1 个 —— 均不动。

**B2. 病灶语义（HIGH）** — `pkg/middleware/permission.go`：
- `RequirePermissions`（:311-337）：超管 bypass → 任一码命中 `checkUserPermission` 即放行；`checkUserPermissionUncached`（:146-196）三步：精确匹配 → Step2 F 型子权限继承（:214-256）→ Step3 模块级降级（:261-278，`system:user:remove`→`system:user`）。
- `RequirePermissionsWithQuery`（:350-384）：读路径末段 list/tree 时并入额外读码；**写操作不受 queryExtra 影响**——但组级 codes 本身含写码（add/edit/delete），病灶①仍在。
- `RequireStrictPermissions`（:411-437）+ `checkUserPermissionStrict`（:444-462）：仅精确匹配，`strict|` 缓存键，30s TTL —— **勿重做，直接复用**。
- 兄弟分离范式（gin Group 继承父中间件链，写组必须是 authorized 的兄弟组）：`router.go:166-221` 注释 + `user_router.go` / `strictUserRouteGroup`（router.go:215-219 引用）。

**B3. 已落地基建（勿重做）**：RequireStrictPermissions、`/system/users` 读/写兄弟分离、`permissionCodes.ts` + `check-permission-codes.mjs`（双向校验 a/b/c/d 四类漂移，`ruoyi-react-frontend/scripts/check-permission-codes.mjs:1-29`）、`sys_schema_migrations` + registry.go（84-03）。

**B4. F 型菜单播种现状 vs 需求（HIGH，RBAC-05 关键前置）** — tmp_ftype.json 共 131 个 F 型码。**完全没有 F 型按钮的模块**（本研究拆分将产生的全部新码均在此列）：
`system:role`、`system:menu`、`system:dept`、`system:post`（**system 4 组一个码都没有**，过滤结果为空数组）、`network:device`、`network:credential`、`network:template`、`ops:location:alias` 等。有部分 F 型的：`network:backup`×3、`network:command`×1、`network:discovery`×1、`ops:ad`×9 等。

## 实现方案建议

### C. system 域其余 4 组拆分明细（RBAC-04，HIGH，源码实读）

拆分范式完全照抄 `/system/users`（router.go:166-221）：读组保留组级 ANY-of（可继续 WithQuery），写组升为 authorized 兄弟组逐动作 `RequireStrictPermissions([单一码])`。

| 组 | 挂载 | 读路由（保留组级） | 写路由（逐动作拆） | 现组级 codes |
|---|---|---|---|---|
| roles | router.go:231-241 | /list /statistics /all /:id | `""`(Create)→add；/:id/update→edit；/:id/status→edit；/:id/delete + **/batch**(BatchDelete)→remove | RoleList/Add/Edit/View |
| menus | router.go:247-257 | /tree /list /tree-select /role-menu-tree-select/:roleId /:id | `""`→add；/:id/update + /:id/status→edit；/:id/delete + /batch-delete→remove | MenuList/Add/Edit/View |
| departments | router.go:260-276 | /tree /list /tree-select /:id /role-dept-tree-select/:roleId /:id/users | `""`→add；/:id/update + /:id/status→edit；/:id/delete + **/batch**→remove；Excel import/export(SetupExcelRouter, department_router.go:64)→单独子组 import/export | DeptList/Add/Edit/View + OpsSelectorReadPerms(WithQuery) |
| posts | router.go:279-289 | /list /statistics /all /enabled /:id | `""`→add；/:id/update→edit；/:id/delete + /batch-delete→remove | PostList/Add/Edit/View |

**新权限码清单（对齐 `system:user:*` 规范，删除统一 `remove` —— WR-01/D-04 先例，config.go:90-94 注释）**：

| 模块 | 需播种的新 F 型码（Phase 87 播种，85 只建常量+挂载） |
|---|---|
| system:role | `system:role:add` `system:role:edit` `system:role:remove`（list/view 常量已有） |
| system:menu | `system:menu:add` `system:menu:edit` `system:menu:remove` |
| system:dept | `system:dept:add` `system:dept:edit` `system:dept:remove`（+建议 `system:dept:import` `system:dept:export`，Excel 入口） |
| system:post | `system:post:add` `system:post:edit` `system:post:remove` |

共 **12~14 个新码**（常量追加进 config.go 对应区块 + `permissionCodes.ts` 同步声明，否则 check-permission-codes.mjs b 类漂移 exit 1）。

### B'. 全站拆分策略与规模（RBAC-01）

1. **逐域推进**（每域 = 一个子批次）：读组保留组级 ANY-of（含 WithQuery 的 queryExtra 读放行）；组内写路由搬出到 authorized 兄弟写组，逐动作单一码 + RequireStrictPermissions。
2. **规模**：~33 组 / ~110 条写路由。各域清单以重扫为准（§0）。主要域：ops 楼宇/楼层/工位/工位设备/资产/信息点/机房/专线/房间设备/墙/门/平面图文本/location-alias（13 组）、network device/credential/template/command/executions/backup/discovery/port-write（8 组）、system ad_domain/manager_rule（6 组，其中 ad account edit/delete 组已是逐动作单码，只差 Require→Strict 一词）、rpa/duty/knowledge/monitor 等目前**无任何权限中间件**的模块（§0 unprotected）**不在本 phase 范围**——它们连 ANY-of 都没有，属授予闭环/后续 phase，避免范围爆炸（LOW 需与规划者确认）。
3. **新权限码总量（全站）**：MEDIUM 估算 ~60-80 个（每写域 add/edit/remove 至少 3 码 + execute/sync/test 等专用码沿用现状单码组直接升 Strict 不需新码）。**原则：优先复用已有常量码**（如 network:backup:restore 单码组只需 Require→RequireStrict）；仅当 ANY-of 组内无语义匹配的写码时新增。85 交付物 = 常量 + 挂载 + TS 声明；播种 migration 留给 87。
4. **回归面（拆分后谁失去权限）**（HIGH）：现 ANY-of 组下，持组级任一码（含 list！）的用户可写。拆分后仅持写码的用户不受影响；**只持 list/view（只读角色）自动从 200 变 403——这正是需求**；持"页面本体 C 型码"（如 `ops:building`）的角色原先靠 Step3 降级放行，拆分后也失去写权。判定方法：对每个域跑

```sql
-- 某域拆分前：哪些角色依赖 ANY-of 放行写操作
SELECT DISTINCT r.role_key FROM sys_user_role ur
JOIN sys_role_menu rm ON ur.role_id=rm.role_id
JOIN sys_menu m ON rm.menu_id=m.id
WHERE m.perms IN (<组级读码列表>)   -- 如 ops:building:list
  AND NOT EXISTS (SELECT 1 FROM sys_role_menu rm2 JOIN sys_menu m2 ON rm2.menu_id=m2.id
                  WHERE rm2.role_id=ur.role_id AND m2.perms = <对应写码>);
```
   生产基线（HIGH，.planning/notes/2026-10-28-pg-permission-baseline.md §1）：仅 6 个角色（admin/kjgl/xtgl/adyk/vm_user/user），admin bypass；user 写权限=0。**迁移策略**：Phase 87 播种时用 `GrantNewMenuToRolesHavingParent`（`internal/core/db/migrations/menu_grant_helpers.go:26`，已带幂等/参数化/PG-sqlite 双路径测试）给"持有父页面 C 型菜单的角色"补授全部动作码 → 现有管理类角色（xtgl/kjgl 等）零感知；**85 内不补授**（fail-closed 过渡期，窗口期=85 合入到 87 播种之间，需在 phase plan 显式声明）。

### A'. 死表接线方案（RBAC-03）

1. **接线点选型（推荐：构建期/CI 校验，不选运行时查表）**（HIGH）：
   - *运行时判定链查表*（每请求先查 GetPermissionByPath 再兜底）：与 RequireStrictPermissions 双轨判定、`matchPath` 与 gin 路由匹配语义可能不一致（gin 同段通配 vs 死表 `:id` 段）、热路径加一层查表（49 条线性扫）、缓存键语义混乱 —— **否决**。
   - *路由注册期校验*（启动时断言）：可做但启动失败策略激进。
   - **推荐**：`GetPermissionByPath`/`matchPath` 保留为**纯数据 + 单测**；真正接线做成 **CI 校验脚本**（RBAC-03 字面要求"校验脚本失败"）：事实源 = 从挂载代码扫描的写路由清单（`tmp_scan_final.cjs` 正规化为 `scripts/scan-routes.cjs`，产物 `route-registry.json` 入库），校验 = 每条走 `RequireStrictPermissions` 的写路由必须在 `GetRoutePermissions()` 表中有登记（路径+方法+码三对齐，允许 `:id` 占位语义）。**未登记新写路由 → exit≠0**。
2. **挂 lint:ci**：`ruoyi-react-frontend/package.json:14` 的 lint:ci 是既有聚合口（eslint && assert-lint-budget && check-exemption-reasons && css-contract）。同理把 `node scripts/check-route-registry.mjs` 串进该链（或后端 `go test ./pkg/permission/ -run TestRouteRegistry` 进 CI go test 步骤，二选一，推荐 go test 更贴近 Go 事实源）。
3. **补表规模**：现表 49 条中用户 10 条+端口写 8 条=18 条与实际对齐；角色/菜单/部门/岗位 ~30 条需按 §C 重建（含 batch/batch-delete、/departments 前缀修正）；ops/network/ad 等其余 ~90 条写路由待各域拆分时同步登记 → 最终表规模 ~110 条。**补表与拆分同批交付**（W2 每域：拆路由 + 更新常量 + 登记死表），避免表再次漂移。

## Validation Architecture（必需）

### E1. 决定性对照实验（#79 范式复用，HIGH）

模板 = `.planning/notes/2026-10-29-v124-perm-fixture.md`（受限角色/用户造数 → HTTP 直调 → DB 前后对照 → 清理自检），全流程已验证可复跑：

1. **夹具**：每抽验模块建 `p85_<mod>_test` 角色 + 测试用户（SM3 哈希用仓库 `internal/core/security/password.go` 实现，跨实现校验过），绑定：父页面 C 型菜单 + `:list` F 码（**不含写码**）——测"只读角色调写接口 403"；对照组临时补授写码后应 200（证明 403 来自权限门）。超管 bypass 键是 `role_key='admin'`（permission.go:281-302），夹具角色不得撞名。
2. **HTTP 403 判定**：`response.ErrForbidden` → 403（middleware 各分支一致，permission.go:122/334/381/434）。
3. **DB 前后对照**：写接口调用前后 `SELECT count(*)` 目标表（如 sys_role），403 时 count 不变 = RBAC-01 "不产生任何数据变更"。
4. **生产库夹具清理自检**：清理 SQL 后跑计数自检（v124 note §90_cleanup.sql 模式）；夹具造数脚本放 `%TEMP%` 不入库。
5. 抽样 ≥8 模块（ROADMAP.md:191）：建议 system 4 组 + ops building + network device + network command + ad account（覆盖 Require/WithQuery/单码升 Strict 三种拆法）。

### E2. sqlite vs PG（HIGH，已知坑确认）

- **坑确认**：`internal/services/system/role_service.go:425-430` `clause.OnConflict{DoNothing:true}` 在 sqlite（modernc）下方言不兼容 → **sqlite 环境无法经角色编辑 API 给角色分配菜单** → sqlite 夹具只能**直接 INSERT sys_role_menu/sys_user_role**，不能走 API 造夹具（v124 note §0 也记录了"role_menu 记录未走 API"的历史坑）。
- **结论**：决定性对照实验**在 PG（10.62.10.34 生产/准生产只读+受控写）跑**；sqlite 仅跑纯逻辑单测（middleware 缓存、matchPath、registry 校验）。`GrantNewMenuToRolesHavingParent` 已有 sqlite noop + PG 双路径测试（menu_grant_helpers_test.go:33-107），87 可直接沿用。

### E3. 回归守卫清单

| 守卫 | 内容 | 挂载点 |
|---|---|---|
| 路由注册表校验 | 写路由 ⊆ GetRoutePermissions()（RBAC-03 核心） | `go test ./pkg/permission/`（新增 route_registry_test.go） |
| 权限码三方一致 | TS ↔ Go 常量 ↔ 菜单种子（新增码会触发 b/c 类直到 87 播种 → **85 需给新码加临时豁免或在 TS 先声明**，豁免机制见 check-exemption-reasons.mjs 先例） | check-permission-codes.mjs（lint:ci 内） |
| 既有守卫不回退 | `router_80_04_test.go`、`menu_grant_helpers_test.go`、operlog regression_test.go（25 常量锁定） | go test ./... |
| 挂载点计数守卫（建议） | 断言 RequireStrictPermissions 挂载组数 ≥ 拆分交付数，防回退 | go test ./internal/api/ |

## 波次建议

```
W1  system 域 4 组（RBAC-04）              —— 全在 router.go:230-289 + 4 个 *_router.go，模式现成，先做小切片验证拆分范式与新码流程
W2a 死表重建 + registry 校验脚本（RBAC-03） —— 以 W1 拆完的 4 组为首批登记；go test + 脚本骨架落地
W2b 全站 ANY-of 拆分（RBAC-01），按域分批：ops 13 组 → network 8 组 → ad/manager 6 组；每批=拆+补常量+补 TS+登记死表
W3  对照实验（8 模块抽样）+ 回归守卫收口 + 生产回归面核查（§B4 SQL）
```

- **串行约束**：`internal/api/router.go` 是共享文件（W1 改 ：230-289，W2b 改 ：590-961 区段各组）→ **W1 与 W2b 不并行**，同域内各 `*_router.go` 互不相干可并行。
- **依赖**：W2b 每批依赖 W2a 的登记机制先立（否则拆完无守卫）；F 型播种 migration 在 87，85 只交付常量+TS 声明+挂载。按域分批优于一次性（110 条写路由一次性改 review 不可控，且生产回归面需逐域核查）。

## 边界与不做什么

- 不碰前端 86（useActionPermission 铺开）、不播种 F 型菜单（87，GrantNewMenuToRolesHavingParent 届时用）、不做 data_scope（84 已完成）、零新依赖。
- 无权限中间件的模块（duty/knowledge/rpa/monitor cache 等 ~411 条扫描口径路由）不在本 phase 加保护——它们没有 ANY-of 可拆，属授予闭环前置，超出 RBAC-01 字面（LOW：若规划者认为 RBAC-01"全站写操作"含这些模块，需先扩 phase 范围再动）。
- 不改 `RequirePermissions`/`checkUserPermission` 既有语义（纯增量加 Strict 挂载，Phase 78 先例）。
- tmp_* 文件保留勿删（重扫基线）。

## Sources

- `pkg/permission/config.go:10-213`（常量集）、`:224-290`（死表 49 条）、`:293-330`（匹配语义）
- `pkg/middleware/permission.go:146-196/214-278`（三步判定）、`:311-337`（ANY-of）、`:350-384`（WithQuery）、`:411-462`（Strict）、`:281-302`（admin bypass）
- `internal/api/router.go:160-309`（system 挂载+users 范式）、`:590-961`（ops 组挂载）
- `internal/api/v1/system/{role,menu,post,department,user}_router.go`（4 组路由明细）
- `internal/api/v1/network/network_router.go:63-196`、`internal/api/v1/system/{ad_domain,manager_rule}_router.go`
- `internal/services/system/role_service.go:425-430`（sqlite OnConflict 坑）
- `internal/core/db/migrations/menu_grant_helpers.go:7-26` + `_test.go`（播种范式，87 复用）
- `ruoyi-react-frontend/scripts/check-permission-codes.mjs`、`ruoyi-react-frontend/package.json:14`（lint:ci）
- `.planning/ROADMAP.md:183-192`、`.planning/REQUIREMENTS.md:29-33/125-129`
- `.planning/notes/2026-10-28-pg-permission-baseline.md`、`.planning/notes/2026-10-29-v124-perm-fixture.md`
- 前任数据：`tmp_routes.json`/`tmp_mounts.json`/`tmp_ftype.json`/`tmp_all_routes.json`/`tmp_scan_final.cjs`；本研究分析脚本 `tmp_analyze.cjs`（保留）

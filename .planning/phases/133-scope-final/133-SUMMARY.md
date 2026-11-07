# Phase 133 交付总结：扩展实体 + vdi 合并迁移 + 死实现删除 + DeptSidebar 联动 + UAT

- 日期：2026-11-07
- 波次：W3 收尾（130 → 131 → 132 → **133** 严格串行），前置全部落地：
  130 `06e892ee`/`cb04d0f1`、131 `e62bc7b6`/`ade0c823`、132 `b02e1bac`/`4ff4d648`
- 本 phase 提交（3 笔，详见 §4）：
  `5161b39f`（part1 C/D/E/F）、`bc381628`（part2 A/B，Lead 代提交）、
  第三笔 `fix(v1.52-p133): 补 BatchOperation operlog 四类计数 + 回填 SUMMARY 验收数据`
- 关联需求：`REQUIREMENTS.md` DS-01/02/03/04/05/06/07 + 映射总表 + 验收硬约束；`ROADMAP.md` Phase 133
- LOCKED 开工输入（只读消费，不重新裁定）：
  - `261107-v152-ds05-scope-decision.md`（归属链 / workstation-device / 显式豁免清单）
  - `261107-v152-self-scope-decision.md` §4（vdi 迁移 + middleware 死实现删除）
  - `261107-v152-batch-import-decision.md` §5（Dept/Self 批量语义推广到四实体）

---

## 0. 裁定补充登记（Phase 133 消费，不重新裁定）

### 0.1 dedicatedLine 归属链 schema 冲突补充裁定

- **依据**：2026-11-07 Lead 裁定（本 phase 开工前消息确认）。
- **冲突**：原 `.planning/notes/261107-v152-ds05-scope-decision.md` 裁定 1 假定
  `ops_dedicated_lines` 存在 `building_id`（或直挂 `org_id`）列，写
  「`building_id → ops_building.org_id`」。本 phase 逐行核实
  `internal/models/operations/dedicated_line.go:17-44` 与全部 migrations
  （`040_add_dedicated_line_rooms.sql` 只增加 `source_room_id`/`dest_room_id`），
  确认该表**既无 `building_id` 也无 `org_id` 列**，与裁定 1 文字假定不符。
- **补充口径（133 直接消费）**：按裁定总原则「归属链回溯到最近带机构列实体 + 链断 fail-closed」，
  落地为
  `source_room_id → ops_server_rooms.building_id → ops_buildings.org_id`
  **OR**
  `dest_room_id → 同源链 → ops_buildings.org_id`，
  **任一端归属链命中可见集即放行**。
  - 理由：专线天然是「两端」实体，只看源端会让「本机可见、对端不可见」的正常跨机构专线被误拒；
    任一端命中既符合业务语义，又不扩大越权面（两端都不在可见集时子查询 0 行 → 空集，仍 fail-closed）。
  - 写端点同理：Create/Update 提交的 `source_room_id` / `dest_room_id` 至少一端落可见集，否则 403。
  - **不新增迁移、不加列**（v1.52 新迁移从 236 起编，本裁定不消耗编号）。
- **登记约束**：`.planning/notes/*` 为只读，原裁定文字不改动；本补充口径**只在本 SUMMARY 登记**，
  实现与 403 矩阵以本口径为准。

### 0.2 其余 LOCKED 结论（原样消费，不重新裁定）

| 项 | 结论 |
|---|---|
| serverRoom | 楼宇链（`ops_server_rooms.building_id → ops_buildings.org_id`），收敛复用 `scope_filter.go::applyReadScope` |
| roomDevice | 两级链 `room_id → ops_server_rooms.building_id → ops_buildings.org_id`，任一级断链 → fail-closed 空集 |
| asset | `ops_asset.dept_id` 直挂 |
| workstation-device | 跟随 `workstation_id` 所指工位可见集；Self 随工位 `user_id` 收窄；无独立机构语义 |
| location-alias 及豁免面 | **显式豁免，只登记不收口**（见 §4） |
| Self 档 | 四实体/无机房设备无用户归属列 → 空集（fail-closed）；workspace-device 随工位 `user_id` |
| 0/unset → All | 超管零回归硬约束；异常枚举（非 0 且 ∉ 1..5）→ fail-closed |
| 批量语义 | 单条越界 403；批量按可见集裁剪（不整批 403），越界计数入 operlog |

### 0.3 两条实现决策登记（Phase 133 落地，Lead 2026-11-07 确认）

| 决策 | 内容 | 依据 |
|---|---|---|
| **dedicatedLine source/dest 双链 OR 命中** | 见 §0.1：原裁定 1 假定 `ops_dedicated_lines` 存在 `building_id` 列，与真实 schema 不符；按「归属链回溯到最近带机构列实体 + 链断 fail-closed」补充为 `source_room_id` / `dest_room_id` 各自经 `ops_server_rooms.building_id → ops_buildings.org_id`，**任一端命中可见集即放行**（读端点两端 EXISTS OR、写端点候选 org 任一命中放行），两端都断链/都越界 → 空集 / 403。**不新增迁移、不加列**。 | 2026-11-07 Lead 裁定（开工前消息）；`.planning/notes/*` 只读，只在本文登记 |
| **不改 132 `filterVisibleIDs` 白名单，四实体用同源 SQL 自行裁剪** | 132 `scope_write.go` 的 `filterVisibleIDsAllowedTables` 是刻意设的防误用边界，只含四域表（ops_buildings / ops_floors / sys_workstation / ops_info_points），**不含四实体表**。133 在 `scope_entities.go::filterEntityVisibleIDs` 用 131 的 `orgScopeClause` + `effectiveOrgExpr` 构造 `WHERE id IN ? AND CAST(<orgExpr> AS TEXT) IN ?`（EXISTS 子查询形态）的一次性查询，与 132 `filterVisibleIDs` 内部逐字同源：service 层、写 SQL 之前、DB 自筛、All 短路零查询、FailClosed/空集 → 空集、越界计数返回。**不放宽 132 白名单、不改 132 任何文件、不算新增第 5 套实现**。单条写仍直接复用 132 `checkWriteScope`（`ErrScopeDenied` 显式 403），读复用 131 `applyReadScope` / `wrapScopeExists` / `applyScopeToSub`。 | Lead 2026-11-07「按你不改 132 方案落地」确认；理由是放宽白名单会削弱 132 刻意设的防误用边界 |

**403 透传决策**（132 文件头 P0-2 陷阱）：四实体中 serverRoom / dedicatedLine / roomDevice 的
Create/Update/Delete 已用 `response.HandleServiceError`，对 `IsAppError` 天然透传 `ErrScopeDenied` 的 403；
仅 `asset_handler.go` 用裸 500 包装，新增 `writeAssetServiceError` 类型分派：
`*apperrors.AppError`（越界 403）→ 原样透传；普通 error → 恢复既有 `response.Error(c, 500, err.Error())`
（int-first-arg quirk，存量测试断言 400 零回归）。operlog.Record 位置不变。

---

## 1. 交付清单

### ① 四实体读写 403 矩阵（A）— 已落地（`bc381628`）

**交付文件**（8 文件，+1440 行）
- `internal/services/operations/scope_entities.go`（133 新建，659 行）：
  四实体 + workstation-device 的归属链 EXISTS 子查询、读侧可见集注入、
  写侧有效机构解析、批量裁剪统一基建。
- `internal/services/operations/scope_entities_403_test.go`（新建，492 行）：
  403 越权矩阵厚测试（8 个 `TestEntity403_*` / **115 个语义断言** / 21 个 `IsScopeDenied` 403 断言）。
- 6 个 service + `asset_handler.go`：读端点注入 / 写端点 403 / 批量裁剪接线。

**四实体 403 矩阵覆盖数 = 115 个断言**（分项，按语义断言组计，非仅返回值检查）：

| 分项 | 覆盖数 | 内容 |
|---|---|---|
| 读端点越界空集 | **28** | 四实体 × List / Statistics / GetByID / dropdown 口径（R-1：asset 3 + serverRoom 3 + dedicatedLine 4 + roomDevice 4，含计数与越界 not-found）+ 断链读空集（R-2：楼宇软删后 serverRoom / roomDevice / dedicatedLine 三档 Statistics=0）+ All/未注入全量对照（Z-1：4 实体 List 各 2 档）+ FailClosed/Self 读空集（Z-2） |
| 写端点单条 403 | **21** | W-1 Create×4 + Update×2 + Delete×4（含断链 Delete×2）+ W-2 双链 OR 放行 1 / 断链 Create 403 1 + Z-2 FailClosed/Self×2 档 Delete 403×2 实体 + B-1 workstation-device 写 403×5（Add/Delete/Update/SyncAD/SyncAsset）+ Self 他人工位 403 1 |
| 批量按可见集裁剪 | **13** | W-3 四实体 BatchDelete 各一组（asset / serverRoom / dedicatedLine / roomDevice），含越界行零变更语义断言（请求行数 / 可见行数 / 越界保留行数） |
| 归属链断链专项 | **11** | dedicatedLine 两端断链 Create/Delete/GetByID/Statistics/批量保留；roomDevice 两级断链 GetByID/Delete/Statistics/批量保留；楼宇软删触发的跨实体断链（R-2） |
| 档位边界（All/未注入/异常/Self/双链 OR） | **42** | 0/unset 与 `FromContext ok=false` 零回归（131/132 `scopeIsAll` 契约）、FailClosed、Self 无用户列空集、workstation-device Self 随工位 `user_id`、dedicatedLine 源/目的双链 OR 命中、All 全量放行 |

- [x] A 落地回填：四实体 403 矩阵覆盖数 = **115**（读 28 / 写 21 / 批量 13 / 断链 11 / 档位边界 42）
- [x] 收敛 `server_room_service.go` 的 `getDeptAndChildDeptIDs`：明确为「请求 orgId 业务筛选展开」（与 ScopeResult 强制权限边界正交、AND 叠加），
  展开逻辑本身复用 `BuildDeptRecursiveFilter` 单一真实源（dept_filter.go），不内联复制四条件、**不新增第 5 套子树实现**；
  `SearchServerRoomOptions` 的 orgId 等值筛选同步改为同款 `IN` 展开，消除 List/Options 口径不一致。
  登录人可见集的展开只由 `pkg/datascope.Resolve` 承担，读端点经 `applyEntityReadScope` / `wrapScopeExists` 注入。
- [x] **零回归硬约束**：A/B 接入后曾出现 3 个存量 CRUD 测试 + 4 个 asset handler 测试回归，均已按 131/132 契约修正
  （`!ok → All` 完全短路、普通 error 保持既有 500/400 语义、`ErrScopeDenied` 才透传 403），
  `go test -count=1 ./internal/services/operations/... ./internal/api/v1/operations/...` 全绿。

### ② vdi / 超管零回归 + 死实现与死测试删除（D，已落地自洽）

**vdi 生产调用方迁移**
- `internal/services/vdi/vm_service_impl.go::ListVMs` 签名保持
  `(ctx, req, userID, dataScope)` 不变（牵动 handler/其它调用方 = 0）；
  内部改为「优先 `datascope.FromContext(ctx)` 取 ScopeResult → `ApplyVMScopeResult`；
  无注入（ok=false）回落显式 `ApplyVMDataScopeFilter` + `ApplyBoundUserFilter`」。
- 保留审查 P1-4 语义：`dataScope != 0`（异常枚举走 fail-closed，绝不改回 `>0`）；0/unset 不过滤。
- 新增 `internal/services/vdi/vm_data_scope_filter.go::ApplyVMScopeResult` 纯函数（唯一新增实现）：
  - All → 短路不过滤、不追加 `IS NOT NULL`（超管零回归）；
  - Custom / Dept / DeptChild（DeptIDs 已由 `datascope.Resolve` 展开）→
    `bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?))`，空集 → `1=0`；
  - Self（HookSelfUser）→ `bound_user_id = ?` + UserID；
  - FailClosed（异常枚举 / Resolve 失败兜底）→ `1=0`；
  - 所有非 All 档统一追加 `bound_user_id IS NOT NULL`（等价旧 `ApplyBoundUserFilter`）。
- **旧实现保留不删**：`ApplyVMDataScopeFilter` / `ApplyBoundUserFilter` / vdi `getChildDepts`
  （既有 fail-closed 测试 `vm_service_test.go:498/516/541/543` 直接调用，零回归）。
- 新增迁移对照测试 `internal/services/vdi/vm_scope_migration_test.go`：
  五档（All/Custom/Dept/DeptChild/Self）+ 0/unset + 异常枚举（99 / -1）逐档
  「旧生产语义 == Resolve→FromContext 生产链路 == 纯函数 ScopeResult」三方对照；
  另覆盖 Custom 角色-部门映射、FailClosedScope 构造、未绑定用户隐藏、
  未注入 ctx 回落、既有 fail-closed 锚点。

**删除的死实现清单**（`pkg/middleware/permission.go`，按裁定 §4）
| 符号 | 原位置 | 删除理由 |
|---|---|---|
| `ApplyDataScope` | :569-643（原） | 全仓无生产调用方；Self 档 `deptField = userID` id 空间错位 |
| `ApplyDataScopeFromContext` | :659-685（原） | 同上；仅死测试使用 |
| `getChildDepts`（middleware 版） | :645-657（原） | 仅被上述死实现调用 |
| `allowedDataScopeFields` / `isValidDataScopeField` | :86-95（原） | 仅被 `ApplyDataScope` 调用；Lead 选保守方案 (a) 一并清 |
| `gorm.io/gorm` import | :15 | 删除后变孤儿 import |

**绝对保留（不可删，已复核）**
`DataScopePermission` / `getUserMaxDataScope` / `injectScopeRequestContext` /
`RequirePermissionsWithQuery` / `RequireStrictPermissions` / `RequirePermissions` /
`RequireAllPermissions` / `isSuperAdmin` / `checkUserPermission*` / `isQueryPath` 等生产链路。

**删除的死测试清单**（`pkg/middleware/permission_74_08_test.go`）
| 符号 | 原位置 | 连带说明 |
|---|---|---|
| `TestApplyDataScope` | :312-366（原） | 直接依赖 `ApplyDataScope` |
| `TestApplyDataScopeFromContext` | :368-395（原） | 直接依赖 `ApplyDataScopeFromContext` |
| `seedDataScopeTables` | :304-310（原） | 仅上述两测试引用，删后变未使用函数 → 必须一并删 |
| `TestIsValidDataScopeField` | :89-95（原） | 依赖 `isValidDataScopeField`（保守方案 a 一并清） |

**必须保留且已复核零回归的测试**：`TestDataScopePermission`（测中间件注入）、
`TestGetUserMaxDataScope`（测 MAX=最严档 + 无角色 COALESCE→Self）、
`TestIsQueryPath_7408`、`TestPermissionMiddleware`、`TestRequirePermissions`、
`TestRequireAllPermissions`、`TestHasUserPermission`、oper_log / 缓存 / 权限等全部其余 ~18 个测试。
删除后 `uuid` / `require` / `assert` / `models` / `gorm` import 均仍被保留测试使用，编译无孤儿。

### ③ 前端 DeptSidebar 可见集断言（E，已落地自洽）

- `DeptSidebar.tsx`：新增可选 `visibleDeptIds?: string[]`（后端 tree/dept-options 已收口的可见集），
  透传给 `DeptTree`。**以后端返回为唯一数据源，前端不硬编码任何过滤规则**
  （不按机构层级 / 名称 / 角色自行推断可见性）。不传时行为与 Phase 133 前完全一致（零回归）。
- `DeptTree/visibleDeptFilter.ts`（新文件）：`filterTreeByVisibleDeptIds` 纯函数，
  按可见集裁剪树且保持树形结构（自身命中→连同子树；祖先未命中→保留路径只留命中子树；
  全不命中→空集）。单独成文件以避开 `react-refresh/only-export-components` 门禁。
- `DeptTree/index.tsx`：`treeData` 计算末尾接入该函数；`visibleDeptIds` 为空/不传时原样返回。
- 新增 `DeptTree/__tests__/visibleDeptScope.test.tsx`：
  **两个不同 scope 用户（不同 visibleDeptIds）渲染不同部门集**（核心验收）+ All 档全量 +
  未传零回归 + 空集合 + fail-closed 空交集 + 纯函数 5 个边界分支。
- 结果：DeptTree/DeptSidebar 相关 vitest **24 passed**（新 9 + 既有 15）；
  `npx prettier --write` 改动文件；`npm run lint:ci` **ALL PASS**（含 antd 6 弃用 prop 门禁 C14 全表零命中）。

### ④ UAT 清单（F）

见 [133-UAT.md](133-UAT.md)（人工 UAT 清单，不阻断自动验收）。

---

## 2. 显式豁免面登记（C：只登记，不改代码）

> 依据 `.planning/notes/261107-v152-ds05-scope-decision.md` 附「v1.52 全量显式豁免清单（终态登记）」
> 与 `REQUIREMENTS.md` 映射总表。以下面 **Phase 133 不改代码**，仅在本文与 UAT 清单登记
> 理由、残余风险与人工复核项；「已知残余面、接受风险」，**不以「间接可见所以安全」结案**。

| 面 | 端点（router.go） | 豁免理由 | 残余风险 | 133 处置 |
|---|---|---|---|---|
| walls | :1133-1159 | 楼层平面几何坐标/展示文字，无业务归属列 | 几何图可能间接暴露布局；Three.js 编辑器联动回归面大 | 只登记，不收口 |
| doors | :1165-1191 | 同上 | 同上 | 只登记，不收口 |
| floor-plan-texts | :1197-1217 | 同上 | 同上 | 只登记，不收口 |
| rooms/photos | :1046-1049（`room_photo_handler.go` 独立 `GET /list/:roomId` + `GET /:id`） | 照片/房间附件，模型无机构列 | 附件内容人工录入可能含敏感信息 | 只登记，不收口 |
| location-alias | :861-882 | 展示型字典（`sys_dept_location_alias`），无 `org_id/dept_id` 业务归属列 | **别名文本可能包含机构名称等人工录入内容**；若未来录入规范允许含敏感机构名，应升级为按关联楼宇过滤 | 只登记，UAT 加一条「location-alias 豁免复核」人工项 |
| dept-mapping-template | :818（`excel_handler.go::DownloadDeptMappingTemplate`） | **不豁免**：审查 P1-1 补登记，已由 Phase 131 按可见集过滤部门树 | 全链路 `parent_path` 含父部门名，过滤后仍可能暴露上级名称 | **已收口（131）**，133 仅登记不再动 |

> 注：`dept-mapping-template` 不是豁免面，列入本表仅为区分「已收口」与「显式豁免」，
> 避免与 133 的「四项 + location-alias」混淆。豁免面共 **5 项**
> （walls / doors / floor-plan-texts / rooms-photos / location-alias）。

### 残余风险登记（ds05）
- **归属链断链脏数据**：dedicatedLine / roomDevice 的归属链断链按 fail-closed 处理，
  若生产存在历史脏数据（building 软删但专线/机房设备仍引用），非超管将看不到这些行 →
  UAT 增加「ds05 脏数据抽查」人工项；本 phase **不为脏数据开旁路**。
- **两级 JOIN 性能**：roomDevice 两级 JOIN + dedicatedLine 两端 OR 子查询需在 PG 上留
  一条 `EXPLAIN` 观察项（非阻断）。
- **location-alias 别名文本**：见上表，UAT 人工复核。

---

## 3. 内部分批验收（ROADMAP Phase 133 四批）

| 批 | 内容 | 状态 |
|---|---|---|
| ① | 四实体读写 403 矩阵（含归属链断链） | **已自洽**（`bc381628`：8 个 `TestEntity403_*` / 115 断言 / 21 个 403 断言；含 3 个存量 CRUD + 4 个 asset handler 零回归修正） |
| ② | vdi / 超管零回归 + middleware 死实现与死测试删除 | **已自洽**（`5161b39f`：vdi 对照测试全绿 / middleware 测试全绿） |
| ③ | 前端 DeptSidebar 可见集断言 | **已自洽**（`5161b39f`：vitest 24 绿 / prettier / lint:ci ALL PASS） |
| ④ | UAT 清单（含 location-alias 豁免复核、ds05 残余风险脏数据抽查） | **已产出** [133-UAT.md](133-UAT.md)（`5161b39f`） |

### 132 遗留补强（Lead 交办，本 phase 收尾）

- 132 的 `building_handler.go` / `floor_handler.go` / `infopoint_handler.go` 的
  `BatchOperation` success path 原本只有裸 `operlog.Record(OperTypeBatch)`，
  缺「范围档位 / 请求行数 / 可见集内行数 / 越界行数」四类聚合计数
  （`workstation_handler.go` 与 `excel_handler.go` 已由 132 补齐）。
- 本 phase 用 132 导出的 `opsServices.ScopeWriteOperParam(ctx, db, entity, action, ids)`
  经 `operlog.WithOperParam(...)` 一键接入三处，复用
  `ScopeWriteEntityBuilding` / `ScopeWriteEntityFloor` / `ScopeWriteEntityInfoPoint` 常量；
  `operlog.Record` 既有位置、响应形状、错误路径均不变。
- 计入本 phase 最后一笔提交（`fix(v1.52-p133): ...`），文件面与 `bc381628` 不重叠。

---

## 4. 闸门与提交

| 闸门 | 结果 |
|---|---|
| `go build ./...` | **EXIT=0 全绿**（第一批、A/B 后、132 遗留补强后各跑一次） |
| `go test -count=1 ./internal/... ./pkg/... ./cmd/...` | **EXIT=0 全绿**（第一批全量；不带 coverprofile 避免与厚测试 agent 竞态） |
| `go test -count=1 ./internal/services/operations/... ./internal/api/v1/operations/...` | **全绿**（含 8 个 Entity403 + 3 个存量 CRUD + 4 个 asset handler；A/B 接入后 2 轮回归均修正为零回归） |
| `go test -count=1 ./internal/api/v1/operations/...`（132 遗留补强后） | **ok**（building/floor/infopoint 三处 handler 改动零回归） |
| 前端 vitest | **592 文件 / 4128 用例全绿**（Lead 全量复核）+ 本 phase 相关 24 用例全绿（含新增 9 个 `visibleDeptScope`） |
| `npx prettier --write` 改动文件 / `npm run format:check` | **PASS** |
| `npm run lint:ci` | **ALL PASS**（eslint 0 problems；含 antd 6 弃用 prop 门禁 C14 全表零命中；`check-route-registry` OK 路由 702 / 写路由 311 / deferred 138） |
| `internal/api/strict_mount_count_test.go` 下限 16/188/164 | **通过**（本 phase 未动 router.go；130 已挂注入、registry 未重生成） |
| 全量 `go test ./...` 已知环境项 | `scripts/e2e/TestE2EAllEndpointsLogged` 因 Windows/bash 路径拼接报 127（环境既有、非本里程碑引入，如实登记，不修） |

**commit 记录**
- 第一批 `5161b39f` — `feat(v1.52-p133-part1): vdi 迁移 + 死实现删除 + DeptSidebar 联动 + UAT 骨架`（11 文件）
- 第二批 `bc381628` — `feat(v1.52-p133-part2): 四实体读写收口 + workstation-device 归属 + 403 矩阵`（8 文件，Lead 代提交）
- 第三笔 — `fix(v1.52-p133): 补 BatchOperation operlog 四类计数 + 回填 SUMMARY 验收数据`（3 个 handler + 本文；提交后 hash 回填至本行）

原子提交约束：`git add` 只加本 phase 文件，不 `git add -A`，不提交 `coverage.out` / `coverage_diffcov.out`，
不碰 `xingran-react-frontend/.env.production`（工作区该文件为其它 agent 在制、非本 phase 改动）。

---

## 5. 禁区复核（本 phase 未触碰）

`pkg/datascope/**`、`internal/services/operations/scope_filter.go`、`scope_write.go`、
`excel_query_builder.go` / `excel_service.go` 导出路径、`dropdown_helper.go`、
`floor_cache_impl.go`、四域 service 读/写函数（131/132 已交付）、
`internal/core/db/database.go`、`useBackfillDept.tsx`、任何 `.env*`、`.planning/notes/*`。
operlog 强制约定：业务写操作 success path 末尾、`response.Success` 之前 `operlog.Record`——
四实体 handler 既有位置保持不变；132 遗留的 building/floor/infopoint 三处 `BatchOperation`
仅补 `operlog.WithOperParam(ScopeWriteOperParam(...))` 四类聚合计数，`Record` 位置/形状不变（见 §3）。
router.go 本 phase 未改动（130 已挂 DataScopePermission、`scripts/route-registry.json` 未重生成）。

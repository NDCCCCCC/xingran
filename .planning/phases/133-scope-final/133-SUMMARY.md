# Phase 133 交付总结：扩展实体 + vdi 合并迁移 + 死实现删除 + DeptSidebar 联动 + UAT

- 日期：2026-11-07
- 波次：W3 收尾（130 → 131 → 132 → **133** 严格串行），依赖 130 commit `06e892ee` + `cb04d0f1`、131 commit、132 commit
- 提交信息：`feat(v1.52-p133): 扩展实体收口 + vdi 迁移 + 死实现删除 + DeptSidebar 联动 + UAT`
- commit hash：见文末「闸门与提交」（提交后回填）
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

---

## 1. 交付清单

### ① 四实体读写 403 矩阵（A）

> 交付文件与覆盖数在 A 落地后回填；口径已在 §0 锁定。
> 待填：asset / serverRoom / dedicatedLine / roomDevice 各
> List / Statistics / dropdown-options / GetByID 读端点 +
> Create / Update / Delete / Batch 写端点的 403 越权矩阵用例数，
> 含「归属链断链」专项用例（dedicatedLine 两端断链、roomDevice 两级断链）。

- [ ] A 落地后回填：四实体 403 矩阵覆盖数 = ____（读 ____ / 写 ____ / 断链 ____）
- [ ] 收敛 `server_room_service.go:153-170` 的 `getDeptAndChildDeptIDs` 第 4 套子树展开到 datascope 单一真实源（不新增第 5 套）

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
| ① | 四实体读写 403 矩阵（含归属链断链） | 待 A 落地后回填 |
| ② | vdi / 超管零回归 + middleware 死实现与死测试删除 | **已自洽**（vdi 对照测试全绿 / middleware 测试全绿） |
| ③ | 前端 DeptSidebar 可见集断言 | **已自洽**（vitest 24 绿 / lint:ci ALL PASS） |
| ④ | UAT 清单（含 location-alias 豁免复核、ds05 残余风险脏数据抽查） | **已产出** [133-UAT.md](133-UAT.md) |

---

## 4. 闸门与提交（提交后回填）

- `go build ./...`：（待全量跑）
- `go test -count=1 ./internal/... ./pkg/... ./cmd/...`：（待全量跑，不带 coverprofile 避免与厚测试 agent 竞态）
- 前端：vitest / prettier / `npm run lint:ci`（ALL PASS）
- `internal/api/strict_mount_count_test.go` 下限 16/188/164：（本 phase 原则上不动 router.go，130 已挂注入；如动则先 `node scripts/scan-routes.cjs` 重生成 `scripts/route-registry.json` 并跑该守卫）
- commit hash：（提交后回填）
- 原子提交：`git add` 只加本 phase 文件，不 `git add -A`，不提交 `coverage.out`，不碰 `xingran-react-frontend/.env.production`

---

## 5. 禁区复核（本 phase 未触碰）

`pkg/datascope/**`、`internal/services/operations/scope_filter.go`、`scope_write.go`、
`excel_query_builder.go` / `excel_service.go` 导出路径、`dropdown_helper.go`、
`floor_cache_impl.go`、四域 service 读/写函数（131/132 已交付）、
`internal/core/db/database.go`、`useBackfillDept.tsx`、任何 `.env*`、`.planning/notes/*`。
router.go 本 phase 未改动（130 已挂 DataScopePermission）。
operlog 强制约定：业务写操作 success path 末尾、`response.Success` 之前 `operlog.Record`——
四实体 handler 既有位置保持不变。

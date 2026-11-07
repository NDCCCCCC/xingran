# Requirements: XingRan-Next（RuoYi-Next 运维管理系统）

**Defined:** 2026-11-07
**Core Value:** 把数据权限（data_scope）从「用户/VDI 两个模块的局部实现」升级为**运维域全模块的声明式统一收口**——读写两端按登录人数据范围自动过滤与校验，使「中支管理员只能管本机构的工位/楼层/资产」从口头约定变为架构保证。本里程碑是对 v1.51 期间实测确认的越权面（operations 全模块未接 data_scope、strict 组不注入、List orgId 后端不校验）的**终态治理**，替代逐端点补丁。

## v1.52 Requirements

范围裁定（用户 2026-11-07）：**终态方案 = operations 全模块声明式统一收口**，不做逐端点手工接线（已在 quick-261106-wsb 审计中裁定为补丁）。独立按钮权限码 ops:workstation:backfill 已在分支先行落地（PR #75）。

### 起点基线（Lead 实跑复核 2026-11-07）

- **operations 全模块未接 data_scope**：工位 List/Statistics/SearchOptions 的 orgId 由前端传、后端不校验登录人范围（API 直调可读全库）；Create/Update 的 deptId/orgId 可传任意 UUID。
- **data_scope 注入为单点中间件**：注入只由 `DataScopePermission`（permission.go:494-523，`c.Set("data_scope"/"user_id")` 在 :518-519）完成；`RequirePermissionsWithQuery`（:350-384，权限校验 + queryExtra 放行）与 `RequireStrictPermissions`（:411-437）**均不注入**。router.go 中 `DataScopePermission` 仅挂载 usersRead(:187)/usersWrite(:212)/deptsRead(:295)/deptsWrite(:302)/vms(:1228) 五组；**operations 全域读组与写组均只挂权限中间件、未挂任何 data_scope 注入** ⇒ 读写两端都拿不到范围信息（DS-02 的根缺口）。
- **三份重复实现并存**：pkg/middleware `ApplyDataScope`（通用版）、user_service `applyUserDataScope`（PMODEL-01）、vdi `vm_data_scope_filter`——Custom/Dept/DeptChild 展开逻辑三处维护。
- **可用基建**：`models.DataScope` 五档（All/Custom/Dept/DeptChild/Self）、`getUserMaxDataScope`（多角色取 MAX）、工位域「有效机构」表达式已收敛单一真实源 `effectiveOrgExpr`（dept_filter.go）、`BuildDeptRecursiveFilter` 子树展开、PG contract 类型契约测试门控。
- **行业教训参照**：PMODEL-01 的缓存 key 遗漏 data_scope 维度导致跨用户泄漏（已修），新收口必须同步覆盖缓存 key。

### 数据权限统一收口（DS）

- [ ] **DS-01**: **ScopeProvider 单一真实源**——抽取「登录人可见部门集」统一实现（All→全量 / Custom→sys_role_dept 映射 / Dept→本部门 / DeptChild→ancestors 展开 / Self→**按资源语义钩子裁定，不纳入部门集展开** / 异常枚举→fail-closed 空集）。**设计约束（vdi Self 档语义不等价，vm_data_scope_filter.go:87-100 按 `bound_user_id` 按人绑定）**：ScopeProvider 只统一 Custom/Dept/DeptChild 三档部门集展开，Self/All 保留资源语义钩子供各域注入。**0/unset 边界**：data_scope=0（超管 `permission.go:504-507` 早退未 `c.Set`，或未挂注入中间件）**映射为 All、不进 fail-closed 分支**；fail-closed 仅针对「非 0 且不在 1-5」的异常枚举。验收须含超管零回归用例（迁移 user_service 到 ScopeProvider 后超管仍不过滤，对齐 `user_requests.go:30-31` 0=不过滤、`user_service.go:583 case DataScopeAll, 0`）。**多角色 MAX 边界**：`getUserMaxDataScope`（permission.go:526-537，`COALESCE(MAX(r.data_scope), 5)`）的 MAX = 数值最大 = **最严档**（1=All…5=Self），"语义不变"须配多角色组合 / 无角色 COALESCE→Self(5) 的边界用例。**合并归属（避免重叠）**：Phase 130 = 建 ScopeProvider + 迁移 **user_service** 这一处生产调用方 + 处理 pkg/middleware **死实现**（删除 + 清理 `permission_74_08_test.go` 死测试）；Phase 133 = 迁移 **vdi** 生产调用方 + 删除旧实现。**术语统一**：三份「实现」= user_service / vdi / pkg/middleware（后者无生产调用方、为死实现）；两处「生产调用方」= user_service / vdi。**验收口径** = 两处生产调用方迁移 + middleware 死实现与死测试删除；vdi 合并验收含既有 fail-closed 测试零回归。
- [ ] **DS-02**: **data_scope/user_id 注入覆盖 operations 全域**——在 operations 各读组（`RequirePermissionsWithQuery`）与 strict 写组（`RequireStrictPermissions`）**统一挂载/改造 `DataScopePermission`**（permission.go:494 起；现状仅挂 usersRead/usersWrite/deptsRead/deptsWrite/vms 五组），使读写端点均能从 gin context 取到 data_scope/user_id；**不改 `RequirePermissionsWithQuery`/`RequireStrictPermissions` 自身逻辑**（二者只做权限校验）。
- [ ] **DS-03**: **工位域读端点收口**——workstation List/Statistics/SearchOptions/dropdown-options/counts-by-floor/**GetByID（按可见集校验资源归属）**/**dept-options（:805，接收前端 orgId）**/**workstations-all（:813 GET /:floorId/workstations-all，全集无分页、`workstation_service.go:438-449` 仅 `Where("floor_id = ?")` 无范围过滤——真实越权面，必须收口不得豁免）**、floor List/**tree（:755，isQueryPath permission.go:388-396 对 list/tree 同等放行）**/Statistics/SearchFloorOptions/dropdown-options/GetByID、building List/Statistics/SearchBuildingOptions/GetByID、infopoint List/Statistics/dropdown-options/GetByID，以及 asset/serverRoom/dedicatedLine/roomDevice 各自的 /statistics、/dropdown-options、/:id（router.go:945-946/1021-1023/1067-1069/1104-1106）——机构筛选统一应用「有效机构 ∈ 可见集」（复用 `effectiveOrgExpr` + `BuildDeptRecursiveFilter`），后端强制，前端 orgId 参数不可越权读取。
- [ ] **DS-04**: **四域写端点收口（workstation/floor/building/infopoint）**——Create/Update 校验提交的 deptId/orgId 落在可见集内（越界 403 语义）；**Delete（`POST /:id/delete`，buildingDeletes :720-721、workstation :830-831）与 BatchOperation（`POST /batch`）按资源归属校验（越界 403 语义）**；floor 写端点 :767-776、infopoint 写端点 :996-1005（Create/Update/Delete/Batch/Import/Export）一并收口；backfill-dept 回填范围限定可见集，前端确认框显示「本次范围：全库 / XX 部门及子部门」。
- [ ] **DS-05**: **其余 ops 实体收口**——asset（dept_id 直挂）、serverRoom（楼宇链）、dedicatedLine/roomDevice（归属链 scope）、workstation-device 归属、location-alias 查询面：三项归属/范围裁定结论为 **Phase 133 开工输入**（须在 133 开工前由 130 设计阶段或单独裁定记录产出，落 `.planning/notes/`），**133 只消费结论 + 落地 + UAT，不承载裁定动作**。
- [ ] **DS-06**: **缓存与前端联动**——涉收口端点的缓存 key 增加数据范围维度段（对齐 PMODEL-01 防跨范围泄漏；All 档特判短路路径同样须含 scope 维度）；前端 DeptSidebar/下拉按可见集过滤由 Phase 133 落地（后端 tree 端点收口见 DS-03）；越界提交的统一错误文案由 Phase 132 落地。**Excel 导出独立收口（不豁免）**——导出**不走** `WorkstationService.List`，而走 `excel_query_builder.go:215-254` 的 `WorkstationQueryBuilder.BuildQuery`（自建 `sys_workstation` JOIN + FilterMapping）与 `excel_service.go:2407-2453 queryWorkstationIDsForExport`；须在这两条导出查询路径注入可见集过滤，并把导出路径纳入 PG contract 覆盖（现状 `pg_contract_test.go:87-208` 仅覆盖 List/Statistics/SearchOptions/Backfill、未覆盖导出）。落地归属 **Phase 131**（读端点收口波次）。
- [ ] **DS-07**: **范围语义裁定与豁免清单**——Dept/Self 档是否允许批量操作（回填/导入）、Excel 导入是否豁免（跨机构批量导入有正当场景→「豁免 + operlog 审计强化」或同样收口）的结论按 **Phase 130/132 已裁定的结论登记**（书面结论落 `.planning/notes/`，为 130/132 开工输入）；**导出不豁免**，按 DS-06 的「Excel 导出独立收口」任务在 **Phase 131** 落地；DS-05 三项裁定（dedicatedLine/roomDevice 归属链 scope、workstation-device 归属、location-alias 查询面）为 **Phase 133 开工输入**；**133 只消费已裁定结论 + 落地 + UAT，不承载裁定动作**；超管（admin）语义不变。

### ops 域端点/实体 → DS 映射总表（P1-5 覆盖缺口修订）

| ops 域端点/实体（router.go） | DS 归属 | 说明 |
|---|---|---|
| workstation（:805 dept-options / :806 statistics / :807 dropdown-options / :809 GetByID / :813 workstations-all / :830-831 delete+batch / List 系） | DS-03 / DS-04 | 主收口域；**workstations-all(:813) 无范围过滤，属真实越权面，必须收口** |
| building（:701 statistics / :702 dropdown-options / :703 GetByID / :720-721 delete+batch / List 系；:705 是 /geocode） | DS-03 / DS-04 | 同上 |
| floor（:754 statistics / :755 tree / :756 dropdown-options / :757 GetByID / :767-776 写端点） | DS-03 / DS-04 | tree 与 list 同等放行收口；四域写端点一并收口 |
| infopoint（:984 statistics / :985 dropdown-options / :986 GetByID / :996-1005 写端点） | DS-03 / DS-04 | 读 + 写收口 |
| asset（:945 statistics / :946 GetByID） | DS-05 / DS-03 | dept_id 直挂；读端点随 DS-03 收口 |
| serverRoom（:1021 statistics / :1022 dropdown-options / :1023 GetByID） | DS-05 / DS-03 | 楼宇链；读端点随 DS-03 收口 |
| dedicatedLine（:1067 statistics / :1068 dropdown-options / :1069 GetByID） | DS-05 / DS-03 | 归属链 scope 结论为 133 开工输入；读端点随 DS-03 收口 |
| roomDevice（:1104 statistics / :1105 dropdown-options / :1106 GetByID） | DS-05 / DS-03 | 同上 |
| workstation-device（:885-920） | DS-05 | 归属结论为 **133 开工输入**，133 按已裁定结论落地 |
| location-alias（:861-874） | DS-05 | 查询面裁定结论为 **133 开工输入**，133 按已裁定结论落地 |
| walls / doors / floor-plan-texts（:1145-1146 / :1177-1178 / :1205-1206 各有独立 /list + /:id） | **显式豁免（已知残余面，接受风险）** | 这些**独立可直接调用**的读端点返回楼层平面几何坐标 / 展示文字，不含 org_id/dept_id 业务归属字段、不承载机构敏感信息，越权读取不泄露业务归属；本里程碑不收口，登记理由与残余风险（不以「间接可见」结案；Three.js 编辑器联动回归面大） |
| rooms/photos（:1046；room_photo_handler.go:32-33 有独立 `GET /list/:roomId` 与 `GET /:id`） | **显式豁免（已知残余面，接受风险）** | 这些**独立可直接调用**的读端点返回照片/房间附件，模型（room_photo.go:8-16）不含 org_id/dept_id 业务归属字段、不承载机构敏感信息；本里程碑不收口，登记理由与残余风险（不以「间接可见」结案） |
| Excel 导出（excel_query_builder.go:215-254 / excel_service.go:2407-2453） | DS-06（Phase 131） | **不豁免**、不走 List；独立注入可见集过滤 + PG contract 覆盖 |

### 验收硬约束（从 v1.51 期间生产事故与本会话教训提炼）

- uuid/varchar 列比较双侧 CAST AS TEXT；SQLite/PG 双方言——**PG contract 测试门控扩展至新收口查询**（真实 PG 只读/回滚式验证）；**导出/导入路径须纳入 PG contract 覆盖**（现状 `pg_contract_test.go:87-208` 仅覆盖 List/Statistics/SearchOptions/Backfill）；新增测试统一门控变量名 `TEST_PG_DSN`（与 `pg_contract_test.go:36` 一致）或显式兼容 `XINGRAN_TEST_PG_DSN`，CI 中 skip 必须可观测。
- 「有效机构」表达式必须走 `effectiveOrgExpr` 单一真实源，禁止内联复制。
- data_scope 语义 fail-closed 对齐 PMODEL-01：**0/unset 映射 All（超管零回归），仅非 0 异常枚举 fail-closed**；多角色取 MAX（getUserMaxDataScope 语义不变，**MAX=数值最大=最严档 1=All…5=Self**，配多角色组合 / 无角色 COALESCE→Self(5) 边界用例）。
- **缓存 key 必含数据范围维度段**（PMODEL-01 跨范围泄漏教训）；All 档特判短路路径同样可断言；**Phase 131 必交 ops 缓存面清点表**（逐 key 现状→目标，`floor_cache_impl.go:39-45 floor:tree` 固定 key 因按可见集过滤必须改结构或停用）。
- 回归基线：eslint 0 problems / vitest 全绿 / `go build` + `go test`（含 pkg/permission 路由双层守卫、internal/api strict 守卫）全绿 / 新收口查询必须有 PG contract 或等价双方言测试。

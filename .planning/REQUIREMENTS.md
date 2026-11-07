# Requirements: XingRan-Next（RuoYi-Next 运维管理系统）

**Defined:** 2026-11-07
**Core Value:** 把数据权限（data_scope）从「用户/VDI 两个模块的局部实现」升级为**运维域全模块的声明式统一收口**——读写两端按登录人数据范围自动过滤与校验，使「中支管理员只能管本机构的工位/楼层/资产」从口头约定变为架构保证。本里程碑是对 v1.51 期间实测确认的越权面（operations 全模块未接 data_scope、strict 组不注入、List orgId 后端不校验）的**终态治理**，替代逐端点补丁。

## v1.52 Requirements

范围裁定（用户 2026-11-07）：**终态方案 = operations 全模块声明式统一收口**，不做逐端点手工接线（已在 quick-261106-wsb 审计中裁定为补丁）。独立按钮权限码 ops:workstation:backfill 已在分支先行落地（PR #75）。

### 起点基线（Lead 实跑复核 2026-11-07）

- **operations 全模块未接 data_scope**：工位 List/Statistics/SearchOptions 的 orgId 由前端传、后端不校验登录人范围（API 直调可读全库）；Create/Update 的 deptId/orgId 可传任意 UUID。
- **strict 权限组不注入**：`RequirePermissionsWithQuery` 注入 data_scope/user_id（permission.go:496-519），`RequireStrictPermissions`（:413-428）不注入——写端点拿不到范围信息。
- **三份重复实现并存**：pkg/middleware `ApplyDataScope`（通用版）、user_service `applyUserDataScope`（PMODEL-01）、vdi `vm_data_scope_filter`——Custom/Dept/DeptChild 展开逻辑三处维护。
- **可用基建**：`models.DataScope` 五档（All/Custom/Dept/DeptChild/Self）、`getUserMaxDataScope`（多角色取 MAX）、工位域「有效机构」表达式已收敛单一真实源 `effectiveOrgExpr`（dept_filter.go）、`BuildDeptRecursiveFilter` 子树展开、PG contract 类型契约测试门控。
- **行业教训参照**：PMODEL-01 的缓存 key 遗漏 data_scope 维度导致跨用户泄漏（已修），新收口必须同步覆盖缓存 key。

### 数据权限统一收口（DS）

- [ ] **DS-01**: **ScopeProvider 单一真实源**——抽取「登录人可见部门集」统一实现（All→全量 / Custom→sys_role_dept 映射 / Dept→本部门 / DeptChild→ancestors 展开 / Self→按资源语义裁定 / 未知枚举→fail-closed 空集），合并 pkg/middleware、user_service、vdi 三份重复实现为单一包级 API，三处调用方迁移。
- [ ] **DS-02**: **strict 权限组注入**——`RequireStrictPermissions` 与 `RequirePermissionsWithQuery` 同款注入 data_scope/user_id 到 gin context（写端点同样可取）。
- [ ] **DS-03**: **工位域读端点收口**——workstation List/Statistics/SearchOptions/counts-by-floor、floor List/SearchFloorOptions、building List/SearchBuildingOptions、infopoint List 的机构筛选统一应用「有效机构 ∈ 可见集」（复用 `effectiveOrgExpr` + `BuildDeptRecursiveFilter`），后端强制，前端 orgId 参数不可越权读取。
- [ ] **DS-04**: **工位域写端点收口**——Create/Update 校验提交的 deptId/orgId 落在可见集内（越界 403 语义）；backfill-dept 回填范围限定可见集，前端确认框显示「本次范围：全库 / XX 部门及子部门」。
- [ ] **DS-05**: **其余 ops 实体收口**——asset（dept_id 直挂）、serverRoom（楼宇链）、dedicatedLine/roomDevice（归属链 scope 裁定后收口）。
- [ ] **DS-06**: **缓存与前端联动**——涉收口端点的缓存 key 增加数据范围维度段（对齐 PMODEL-01 防跨范围泄漏）；前端 DeptSidebar/下拉按可见集过滤（可选：后端 tree 端点收口驱动）；越界提交的统一错误文案。
- [ ] **DS-07**: **范围语义裁定与豁免清单**——Dept/Self 档是否允许批量操作（回填/导入）；Excel 导入是否豁免（跨机构批量导入有正当场景→「豁免 + operlog 审计强化」或同样收口，实施前裁定）；超管（admin）语义不变。

### 验收硬约束（从 v1.51 期间生产事故与本会话教训提炼）

- uuid/varchar 列比较双侧 CAST AS TEXT；SQLite/PG 双方言——**PG contract 测试门控扩展至新收口查询**（真实 PG 只读/回滚式验证）。
- 「有效机构」表达式必须走 `effectiveOrgExpr` 单一真实源，禁止内联复制。
- data_scope 语义 fail-closed 对齐 PMODEL-01；多角色取 MAX（getUserMaxDataScope 语义不变）。
- 回归基线：eslint 0 problems / vitest 全绿 / `go build` + `go test`（含 pkg/permission 路由双层守卫、internal/api strict 守卫）全绿 / 新收口查询必须有 PG contract 或等价双方言测试。

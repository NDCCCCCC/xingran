# Phase 39: 工位部门物理位置映射 (workstation dept location alias) — Specification

**Created:** 2026-06-25
**Ambiguity score:** 0.089 (gate: ≤ 0.20 ✓)
**Requirements:** 12 locked

## Goal

让运营服务部等"组织编制与物理办公地分离"的部门（独立编制、人员实际在分公司本部或中心支公司办公）能在工位管理页面被选择为工位的"所属部门"，不修改 `sys_workstation` 表结构，不修改 `sys_dept.parent_id`，不引入新菜单。

## Background

公司组织架构中存在一类特殊部门：以"运营服务部"为代表，其在公司编制上是与"分公司本部"同级的独立部门，但人员实际在"分公司本部"或各"中心支公司"办公。当运维在工位管理页面选择"所属部门"时，部门下拉只渲染选中"所属机构"节点的子树，因此运营服务部的子部门永远无法被选中（它与分公司本部同级，不在任何中心支公司的子树下）。

当前状态：
- `sys_workstation` 已存在，含 `dept_id` (string, nullable)、`user_id`、`building_id`、`floor_id`，**无需 schema 变更**
- `internal/services/operations/workstation_service.go:17` `workstationJoinClause` 走 `sys_dept.id::text = sys_workstation.dept_id` 关联
- `internal/models/workstation.go:46` `DeptID *string` 已是 nullable
- 工位编辑弹窗 (`xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:87-92`) `subDeptTree = useMemo(() => findDeptNode(deptTreeData, watchedOrgId))` — 单一子树派生，**不含跨级映射入口**
- `sys_dept_location_alias` 表**不存在**（grep 验证：0 处引用）
- 用户管理、部门管理、空间管理（楼层/楼宇/工位）均不受影响，按用户/部门 tree 的 `parent_id` 正常渲染

## Requirements

1. **`sys_dept_location_alias` 表存在**：migration 165 创建该表，承载"逻辑部门 → 物理部门"的映射。
   - Current: 不存在该表；任何"跨级附加部门"的诉求都无法持久化
   - Target: 表结构 `id UUID PK`、`dept_id UUID NOT NULL`、`location_id UUID NOT NULL`、`scope VARCHAR DEFAULT 'workstation'`、`remark VARCHAR`、`created_at/updated_at TIMESTAMPTZ`、`UNIQUE(dept_id, scope)`
   - Acceptance: 启动后 `psql \d sys_dept_location_alias` 显示上述列与唯一约束；migration 日志输出 `migration 165 completed`

2. **alias 三级校验**：create/update 接口拒绝三类错误映射。
   - Current: 无校验层，任意映射都能入库
   - Target: create/update 时执行 ① `dept_id != location_id`（防自映射） ② `location_id` 对应部门 `is_external_org = 1`（防非外部机构作为物理位置） ③ `dept_id` 对应部门必须是 `location_id` 对应部门的后代（防叶子节点作为物理位置）
   - Acceptance: 任一校验失败 → HTTP 400 + 明确错误信息；三级校验各自有单元测试覆盖

3. **alias CRUD API 端点**：4 个 POST 端点提供完整的 alias 管理能力，每个端点都受独立权限点保护。
   - Current: 无该 API
   - Target:
     - `POST /api/v1/ops/location-alias/list`（需 `ops:location:alias:list`）
     - `POST /api/v1/ops/location-alias/create`（需 `ops:location:alias:add`）
     - `POST /api/v1/ops/location-alias/:id/update`（需 `ops:location:alias:edit`）
     - `POST /api/v1/ops/location-alias/:id/delete`（需 `ops:location:alias:delete`）
   - Acceptance: 4 个端点均注册到 `internal/api/router.go`；无权限时返回 HTTP 403；有权限时按 operlog 标准格式记录

4. **权限定义 + seed**：新增 4 个独立权限记录到 `sys_menu`（或对应权限表，按项目惯例），**不授予任何角色**。
   - Current: 无该权限点
   - Target: migration_165 内 INSERT 4 条权限记录 `ops:location:alias:list/add/edit/delete`；**不** INSERT `sys_role_menu` 关联记录
   - Acceptance: 数据库中 4 条权限记录可见；`sys_role_menu` 中无任何角色拥有这些权限；超级管理员（role_id=1）默认无该权限

5. **工位部门下拉查询 union 改写**：工位编辑弹窗的"所属部门"下拉数据源从单一子树派生改为「子树 + alias 命中 union」。
   - Current: `subDeptTree = findDeptNode(deptTreeData, watchedOrgId)` 只返回 orgId 节点子树
   - Target: 后端 API（工位"部门选项"接口）在 `subDeptTree` 之上 union 上 `scope='workstation' AND location_id=watchedOrgId` 的 `dept_id` 列表；前端 EditModal `subDeptTree` 改为接收 union 结果
   - Acceptance: 当存在 `dept_id=运营服务部/子部门A.id, location_id=中心支公司B.id, scope='workstation'` 的 alias 时，工位编辑选择"中心支公司B"后，"所属部门"下拉包含中心支公司B 子树 + "运营服务部/子部门A"（带 `[映射]` 标识）

6. **前端 Drawer 管理 UI**：工位管理页 list 页工具栏新增 `[⚙ 映射]` 按钮 + Drawer，承载 alias CRUD。
   - Current: 列表页无映射管理入口
   - Target: 列表页右上角工具栏新增 `[⚙ 映射]` 按钮（仅 `ops:location:alias:list` 权限下可见），点击打开 Drawer，Drawer 内部提供 alias 列表 + 新增表单（select 选 dept + select 选 location + scope 默认 `workstation` + remark 可选）+ 删除按钮
   - Acceptance: 无 `list` 权限时按钮不渲染；有权限时 Drawer 正常打开、列表加载、增删生效；Drawer 操作调用受 `add`/`delete` 权限保护

7. **映射条目在部门下拉中的标识**：union 注入的"映射部门"在 Select 组件中显示 `[映射]` 后缀，与原生子树部门视觉区分。
   - Current: 部门下拉中所有条目视觉一致
   - Target: union 注入的条目在 Select `optionLabel` 渲染时追加 `[映射]` 后缀；Ant Design `Tag` 组件包裹（橙色）
   - Acceptance: 视觉上可一眼区分映射条目；不影响 option value 匹配（value 仍是 dept_id）

8. **工位"所属用户"下拉自动可用**：当 `workstation.dept_id` 写为映射部门（如运营服务部/子部门A）时，用户下拉自然包含该部门的用户。
   - Current: 用户下拉走 `recursiveDeptId` 后端递归，已支持任意 dept_id 子树查询
   - Target: 不改动 user picker 代码；仅验证端到端行为
   - Acceptance: UAT 场景 — 选择工位"所属部门"=运营服务部/子部门A，"所属用户"下拉展示 子部门A 的全部用户（不含其他部门）

9. **operlog 集成**：alias 表 4 个写操作（create/update/delete）必须通过 `operlog.Record` / `operlog.RecordWithBody` 记录到 `sys_oper_log`。
   - Current: 无此操作的日志记录
   - Target: handler 内 success path 调用 `operlog.Record(...)`，模块名 "工位管理"，OperType 对应 `OperTypeCreate`/`OperTypeUpdate`/`OperTypeDelete`
   - Acceptance: 在 `sys_oper_log` 表可查到 4 条样例记录；`oper_url` 与 `business_type` 字段值正确

10. **缓存失效**：alias 表任何变更（create/update/delete）触发 dept tree 缓存失效。
    - Current: dept tree 缓存由 `dept_cache_service.go` 维护，alias 表不在缓存范围
    - Target: alias handler 在写操作成功后调用现有 dept cache invalidation 接口
    - Acceptance: 修改 alias 后，下次工位部门下拉查询返回新数据；缓存命中率监控显示正常失效

11. **现有工位数据零迁移**：所有现存 `sys_workstation.dept_id` 值不被自动改写。
    - Current: 现有工位的 `dept_id` 指向物理位置部门（运营服务部人员相关工位的 dept_id 当前为 NULL 或部分指向分公司本部）
    - Target: 本 phase 不回填任何数据；用户重新编辑时可选新条目；如需批量调整由运维通过 Drawer 手动新增映射后自行改工位
    - Acceptance: migration_165 不含 UPDATE `sys_workstation` 语句；上线后存量工位的 `dept_id` 字段值与上线前一致

12. **operlog 模块名 & 业务类型常量集约束**：遵循 CLAUDE.md 强制约定。
    - Current: 现有 operlog 已有完整 25 个 OperType 常量集
    - Target: alias CRUD 复用 `OperTypeCreate/Update/Delete`；模块名 `"工位管理"`（与现有工位模块共用）；不新增 OperType 常量
    - Acceptance: 代码不引用任何非法 OperType 常量；25 个 OperType 常量值不变（由 `internal/utils/operlog/regression_test.go` 守护）

## Boundaries

**In scope:**
- 新建 `sys_dept_location_alias` 表（migration 165）
- 后端 alias CRUD API（4 个端点）+ 三级校验
- 4 个独立权限点 seed（`ops:location:alias:list/add/edit/delete`）
- 工位部门下拉数据源 union 改写（后端 API + 前端 subDeptTree 接收）
- 工位管理页 list 页工具栏 `[⚙ 映射]` 按钮 + Drawer
- 部门下拉中映射条目 `[映射]` 标识渲染
- alias CRUD 的 operlog 记录
- alias 写操作触发 dept cache invalidation
- 权限中间件对 4 个端点的检查

**Out of scope:**
- ❌ 修改 `sys_workstation` 表结构 — 工位表不动，只改前端展示
- ❌ 修改 `sys_dept` 表结构 / `parent_id` — 部门表不动，逻辑层级关系不变
- ❌ 新增菜单项 — `[⚙ 映射]` 仅是 list 页工具栏按钮，不是独立菜单
- ❌ 修改 `sys_dept.parent_id` 渲染逻辑 — 用户管理、部门管理、其他模块按原 `parent_id` 渲染不受影响
- ❌ 修改工位"所属用户"下拉 — `recursiveDeptId` 后端递归已天然支持任意 dept_id
- ❌ 修改工位报表 / 统计 — 报表口径按 `sys_workstation.dept_id` 不变；`[映射]` 仅是工位编辑时的展示
- ❌ 自动化 seed 现有运营服务部子部门映射 — 业务侧在 Drawer 中手动录入
- ❌ 自动回填现有工位数据 — 存量 `sys_workstation.dept_id` 不被改写
- ❌ 工位 import/export 模板调整 — Excel 导入导出的部门列含义不变
- ❌ AD 域控 / sys_user.dept_id 调整 — 用户所属部门不动
- ❌ 跨模块（building/floor/infopoint）的同类问题 — 仅工位范围

## Constraints

- 必须沿用现有 Handler-Service 架构模式（参考 `internal/services/operations/building_service.go`）
- 必须沿用现有 operlog `Record` / `RecordWithBody` 调用模式（CLAUDE.md 强制约定）
- 必须复用现有 dept cache invalidation 接口（不新建缓存机制）
- 后端新增代码放在 `internal/services/operations/` 和 `internal/api/v1/operations/` 目录下
- 前端新增代码放在 `xingran-react-frontend/src/pages/operations/workstations/` 下
- 4 个新权限点 seed 到 `sys_menu` 表（按项目现有惯例），不插入 `sys_role_menu`
- migration 文件命名 `migration_165_sys_dept_location_alias.go`，遵循现有 migration 风格（log.Println + 幂等）
- 不得引入新三方依赖；复用项目已有的 gorm/gin/redis
- 不得修改 `pkg/middleware/`、`internal/core/`、`internal/models/sys_dept.go`、`internal/models/workstation.go`

## Acceptance Criteria

- [ ] migration_165 创建 `sys_dept_location_alias` 表，含 7 列 + 1 个 unique 约束（verified via `psql \d`）
- [ ] `POST /api/v1/ops/location-alias/list` 返回 alias 列表，需 `ops:location:alias:list` 权限
- [ ] `POST /api/v1/ops/location-alias/create` 新增 alias，三级校验生效（自映射/非外部机构/非祖先均返回 400）
- [ ] `POST /api/v1/ops/location-alias/:id/update` 更新 alias，需 `ops:location:alias:edit` 权限
- [ ] `POST /api/v1/ops/location-alias/:id/delete` 删除 alias（软删除），需 `ops:location:alias:delete` 权限
- [ ] 4 个权限点已 seed 到 `sys_menu`，`sys_role_menu` 中无任何角色拥有
- [ ] 工位编辑弹窗选择"中心支公司B"后，"所属部门"下拉包含中心支公司B 子树 + 通过 alias 映射过来的子部门（含 `[映射]` tag）
- [ ] UAT 场景：创建 alias(运营服务部/子部门A → 中心支公司B, scope=workstation) 后，工位编辑可见子部门A
- [ ] 当 `workstation.dept_id` = 子部门A，"所属用户"下拉展示子部门A 的全部用户（不需新代码改动）
- [ ] 4 个 alias 写操作在 `sys_oper_log` 留痕，模块名"工位管理"
- [ ] 无 `ops:location:alias:list` 权限时，工位列表页 `[⚙ 映射]` 按钮不渲染（前端权限 gating）
- [ ] 无 `ops:location:alias:add` 权限时，Drawer 内新增表单 disabled 或隐藏
- [ ] alias create/update/delete 后，工位部门下拉缓存被失效（下次查询返回新数据）
- [ ] migration_165 不含 UPDATE `sys_workstation` 语句，存量数据零迁移
- [ ] `go build ./...` 退出码 0
- [ ] `go test ./internal/services/operations/` 通过（含三级校验单元测试）
- [ ] `npm run type-check` 退出码 0
- [ ] `npm run lint` 无 error 级问题
- [ ] `internal/utils/operlog/regression_test.go` 仍通过（OperType 常量集未变）

## Ambiguity Report

| Dimension          | Score | Min  | Status | Notes                                                              |
|--------------------|-------|------|--------|--------------------------------------------------------------------|
| Goal Clarity       | 0.95  | 0.75 | ✓      | UAT 场景在 explore 阶段已锁定                                     |
| Boundary Clarity   | 0.93  | 0.70 | ✓      | 12 项 OUT 显式列出，覆盖所有邻接诱惑                              |
| Constraint Clarity | 0.85  | 0.65 | ✓      | 权限命名、默认分配、三级校验、缓存失效、seed 时机均在 seed closer 锁定 |
| Acceptance Criteria| 0.88  | 0.70 | ✓      | 18 项可勾选 pass/fail，覆盖 API/UI/权限/UAT/回归                  |
| **Ambiguity**      | 0.089 | ≤0.20| ✓      |                                                                    |

Status: ✓ = met minimum, ⚠ = below minimum (planner treats as assumption)

## Interview Log

| Round | Perspective      | Question summary                                                  | Decision locked                                                                  |
|-------|------------------|------------------------------------------------------------------|----------------------------------------------------------------------------------|
| 1     | Researcher       | 当前工位编辑弹窗"所属部门"下拉如何生成？                        | `subDeptTree = useMemo(() => findDeptNode(deptTreeData, watchedOrgId))` — 单一子树派生，无跨级入口 |
| 1     | Researcher       | `sys_workstation` 表结构是否需要改动？                          | 不需要；`DeptID *string` 已 nullable；最新 migration 164 → Phase 39 用 165        |
| 2     | Simplifier       | 映射存储用新表还是 sys_dept 加字段？                            | 新表 `sys_dept_location_alias` — 保持 sys_dept 不变                              |
| 2     | Simplifier       | UI 入口用菜单还是内联？                                          | 内联：list 页工具栏 `[⚙ 映射]` 按钮 + Drawer（不新增菜单）                       |
| 3     | Boundary Keeper  | 是否改 sys_workstation / sys_dept 表？                          | 都不改；不改 dept tree 渲染；不动 user picker；不动报表                          |
| 3     | Boundary Keeper  | 改造范围是否跨模块（building/floor/infopoint）？                | 仅工位；其他模块问题后续 phase 单独处理                                          |
| 4     | Failure Analyst  | 错误映射会有哪些？                                                | 自映射 / 非外部机构作为 location / 非祖先作为 location → 三级校验                |
| 4     | Failure Analyst  | alias 变更后如何让前端拿到最新数据？                             | 复用现有 dept cache invalidation 接口                                            |
| 5     | Seed Closer      | 初始映射数据从哪里来？                                            | 上线后业务侧在 Drawer 手动录入；migration 不 INSERT 业务数据                      |
| 5     | Seed Closer      | 权限默认分配给谁？                                                | **谁也不给** — migration 只插入 4 条权限记录，角色需手动授权                     |
| 6     | Seed Closer      | 权限命名空间？                                                    | `ops:location:alias:*`（独立模块，不复用 ops:workstation:*）                      |
| 6     | Seed Closer      | user picker 是否需要改动？                                        | 不需要；`recursiveDeptId` 后端递归天然支持任意 dept_id                            |

---

*Phase: 39-workstation-dept-location-alias*
*Spec created: 2026-06-25*
*Next step: /gsd:discuss-phase 39 — implementation decisions (table column types, Drawer form layout, etc.)*
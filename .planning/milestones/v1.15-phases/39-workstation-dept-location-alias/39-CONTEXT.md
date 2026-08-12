# Phase 39: 工位部门物理位置映射 (workstation dept location alias) - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning
**Source:** SPEC.md (12 requirements locked, ambiguity 0.089) + discuss-phase HOW 灰区 4 项决策

---

<domain>
## Phase Boundary

解决"组织编制与物理办公地分离"的部门（如运营服务部）在工位管理页面无法绑定"所属部门"的问题。引入 `sys_dept_location_alias` 表承载"逻辑部门 → 物理部门"映射，工位"所属部门"下拉在原 `subDeptTree` 之上 union 注入映射条目，使运营服务部等子部门能出现在对应中心支公司/楼宇下的工位部门下拉中。

**硬约束：**
- 不修改 `sys_workstation` 表结构
- 不修改 `sys_dept.parent_id` / 部门树渲染逻辑
- 不新增菜单（仅工位列表页工具栏 `[⚙ 映射]` 内联按钮 + Drawer）
- 不修改工位"所属用户"下拉（`recursiveDeptId` 后端递归天然支持任意 dept_id）
- 不动报表 / 统计 / Excel 导入导出口径

</domain>

<spec_lock>
## Requirements (locked via SPEC.md)

**12 requirements are locked.** See `39-SPEC.md` for full requirements, boundaries, and acceptance criteria (18 项 pass/fail checkboxes).

Downstream agents MUST read `39-SPEC.md` before planning or implementing. Requirements are not duplicated here.

**In scope (from SPEC.md):**
- 新建 `sys_dept_location_alias` 表（migration 165）
- 后端 alias CRUD API（4 个端点）+ 三级校验
- 4 个独立权限点 seed（`ops:location:alias:list/add/edit/delete`，不授角色）
- 工位部门下拉数据源 union 改写（后端 API + 前端 subDeptTree 接收）
- 工位管理页 list 页工具栏 `[⚙ 映射]` 按钮 + Drawer
- 部门下拉中映射条目 `[映射]` 标识渲染
- alias CRUD 的 operlog 记录
- alias 写操作触发 dept cache invalidation
- 权限中间件对 4 个端点的检查

**Out of scope (from SPEC.md):**
- ❌ 修改 `sys_workstation` 表结构
- ❌ 修改 `sys_dept` 表结构 / `parent_id`
- ❌ 新增菜单项
- ❌ 修改 `sys_dept.parent_id` 渲染逻辑
- ❌ 修改工位"所属用户"下拉
- ❌ 修改工位报表 / 统计
- ❌ 自动化 seed 现有运营服务部子部门映射
- ❌ 自动回填现有工位数据
- ❌ 工位 import/export 模板调整
- ❌ AD 域控 / `sys_user.dept_id` 调整
- ❌ 跨模块（building/floor/infopoint）的同类问题

</spec_lock>

<decisions>
## Implementation Decisions

### D-01: 映射节点展示 — 原名 + `[映射]` 后缀
- **决策**：在 `subDeptTree` 的 TreeSelect 节点 `title` 字段后追加 ` [映射]` 字符串后缀
- **实现位置**：`EditModal.tsx:87-92` 的 `subDeptTree` 派生逻辑
- **关键代码约束**：
  - 仍走原 `trimTitleToLastSegment` 短名裁剪（在标题收窄后追加 `[映射]`）
  - 不改 TreeSelect 组件类型（避免 ReactNode title 兼容性陷阱）
  - 后端 API 返回的 alias 节点 `is_alias = true` 标记，前端依此判断是否追加
- **示例渲染**：
  ```
  中心支公司B
  ├ 业务部
  ├ 财务部
  └ 运营服务部/子部门A [映射]
  ```

### D-02: alias 表单 UX — 两个独立 TreeSelect
- **决策**：Drawer 内 alias 新增/编辑表单采用两个独立的 TreeSelect，分别选 dept（原组织）+ location（物理位置）
- **dept TreeSelect**：
  - 全量部门树（不过滤 `isExternalOrg`）
  - 支持搜索
  - 默认值：从列表选中行带入（如有）
- **location TreeSelect**：
  - 仅 `isExternalOrg = 1` 的部门树（即 Phase 37 已有的 `orgTreeData` 模式）
  - 单一选择
- **三级校验触发**：
  - `onChange` 任一字段变化时实时校验（不阻塞输入，但显示错误状态）
  - `onSubmit` 时最终校验，失败禁用提交按钮
  - 错误提示用 Ant Design `Form.Item` 的 `validateStatus` + `help` 字段
- **三级校验实现位置**：**前端 + 后端双重校验**
  - 前端：表单 `onChange` 实时提示（即时反馈）
  - 后端：service 层 `validateAlias()` 函数（强约束，最后防线）
  - 不在 DB CHECK 约束里实现（DB 约束难表达 ancestor 关系）
- **Drawer 列表区**：
  - 列表列：dept 名称 / location 名称 / scope / 创建时间 / 操作（删除）
  - 列表分页：使用现有 pageSize=10（不重新发明）
  - 删除操作：二次确认 Modal

### D-03: 缓存失效粒度 — 调用现有 dept cache 失效接口
- **决策**：alias CRUD（create/update/delete）成功后调用现有 dept cache invalidation 接口，一次性失效全量部门树缓存
- **具体实现**：
  - 后端 service 调用 `dept_cache_service.InvalidateDeptCache()`（或等价接口，按 Phase 13/14 已有的失效模式）
  - **不**新建 subDeptTree 专用缓存层（避免缓存爆炸）
  - **不**走 alias 表独立缓存（修改频率低，全量失效可接受）
- **前端缓存联动**：
  - alias CRUD 成功后调用 `useInvalidateDept()` 失效前端 `['dept', 'tree']` 缓存
  - alias 数据本身不缓存（修改即失效即可）

### D-04: scope 字段 — 预留多 scope
- **决策**：`sys_dept_location_alias.scope` 字段保留，默认值 `'workstation'`，预留未来扩展
- **字段定义**：`scope VARCHAR(32) DEFAULT 'workstation' NOT NULL`
- **UNIQUE 约束**：`UNIQUE(dept_id, scope)` — 同一 dept 在同一 scope 下唯一映射
- **预留价值**：
  - 未来如果 building/floor/infopoint 有同类问题，可设置 `scope='building'` / `'floor'` 等
  - 同一 dept 可在不同 scope 下映射到不同 location（如运营服务部子部门A 在工位 scope 下映射到中心支公司B，在 infopoint scope 下映射到中心支公司C）
- **当前使用**：
  - 所有 CRUD API 默认 `scope='workstation'`
  - 工位部门下拉查询过滤 `scope='workstation'`

### D-05: Migration 编号与命名
- **决策**：migration 文件 `migration_165_sys_dept_location_alias.go`
- **风格对齐**：
  - 函数命名 `Migrate165SysDeptLocationAlias(db *gorm.DB) error`
  - `log.Println("Running migration 165: ...")` 启动日志
  - `log.Println("Migration 165 completed: ...")` 完成日志
  - 幂等：`CREATE TABLE IF NOT EXISTS` + `CREATE UNIQUE INDEX IF NOT EXISTS`（PG/SQLite 通用）
- **字段类型**：
  - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`（与现有 sys_* 表一致）
  - `dept_id VARCHAR(64) NOT NULL`（与 `sys_workstation.dept_id` 类型对齐，便于 JOIN）
  - `location_id VARCHAR(64) NOT NULL`（同上）
  - `scope VARCHAR(32) NOT NULL DEFAULT 'workstation'`
  - `remark VARCHAR(255)`（可选说明）
  - `created_at TIMESTAMPTZ DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ DEFAULT NOW()`
  - `deleted_at TIMESTAMPTZ`（GORM 软删除）
- **索引**：
  - PK on `id`
  - UNIQUE on `(dept_id, scope) WHERE deleted_at IS NULL`（partial unique，PostgreSQL 语法）
  - INDEX on `location_id`（用于 alias 反向查询"哪些 dept 映射到这个 location"）
- **权限 seed（同一 migration）**：
  - INSERT 4 条 `sys_menu` 记录，`perms` 分别为 `ops:location:alias:list/add/edit/delete`
  - `menu_type='F'`（按钮级别）
  - **不** INSERT `sys_role_menu` 关联记录（"谁也不给" 已 locked）

### D-06: 工位部门下拉 union 改写的实现位置
- **决策**：在后端 workstation service 层添加 `GetWorkstationDeptOptions(orgId)` 方法（不修改 `workstationJoinClause`）
- **新方法签名**：
  ```go
  type DeptOption struct {
      DeptID   string
      DeptName string
      IsAlias  bool  // 新增字段，前端依此判断是否追加 [映射]
  }
  func (s *workstationService) GetWorkstationDeptOptions(ctx context.Context, orgId string) ([]DeptOption, error)
  ```
- **SQL 实现**（单个 query 完成 union）：
  ```sql
  SELECT id::text AS dept_id, dept_name AS dept_name, false AS is_alias
  FROM sys_dept
  WHERE id::text IN (SELECT unnest(string_to_array(ancestors, ',')) FROM sys_dept WHERE id = ?)
     OR id = ?
  UNION
  SELECT a.dept_id, d.dept_name, true AS is_alias
  FROM sys_dept_location_alias a
  JOIN sys_dept d ON d.id::text = a.dept_id
  WHERE a.location_id = ? AND a.scope = 'workstation' AND a.deleted_at IS NULL
  ```
- **前端集成**：
  - `EditModal.tsx` 的 `subDeptTree` 改为接收后端 union 后的部门列表
  - 由于 `subDeptTree` 是 `DeptTreeNode[]` 结构（树形），需后端返回时**已经构造好树形**（每个 alias 节点挂到正确的父位置）
  - 实际上：原 `subDeptTree = findDeptNode(deptTreeData, watchedOrgId).children` 是 deptTreeData 的子树；alias 节点不在此树中
  - **正确实现**：保留 `deptTreeData`（来自 `useDeptTree`），alias 数据单独 fetch（per `watchedOrgId`），在 `subDeptTree` 渲染前注入 alias 节点
- **最终 subDeptTree 构造**：
  ```typescript
  const aliasList = useAliasByLocation(watchedOrgId);  // 新增 hook
  const subDeptTree = useMemo(() => {
    const baseTree = findDeptNode(deptTreeData, watchedOrgId)?.children || [];
    if (!aliasList.length) return trimTitleToLastSegment(baseTree);
    const aliasNodes = aliasList.map(a => ({
      id: a.dept_id,
      title: trimTitleToLastSegment([{ title: a.dept_name }])[0]?.title + ' [映射]',
      isLeaf: true,
      is_alias: true,
    }));
    return [...trimTitleToLastSegment(baseTree), ...aliasNodes];
  }, [deptTreeData, watchedOrgId, aliasList]);
  ```

### D-07: alias 数据前端获取方式 — useQuery hook
- **决策**：新增 `useAliasByLocation(locationId)` hook（React Query，queryKey `queryKeys.locationAlias.byLocation(locationId)`）
- **失效联动**：
  - `useInvalidateDept()` 同步失效 alias 数据（同一 React Query family）
  - 或新增 `useInvalidateLocationAlias()` 单独失效
  - 推荐**前者**（与 dept tree 一致的管理方式）
- **参数依赖**：`locationId` 为空时 `enabled: false`，不发起请求

### D-08: 前端权限 gating 实现位置
- **决策**：使用项目现有的 `hasPermission` hook 或 `authStore.permissions` 数组
- **渲染策略**：
  - 无 `ops:location:alias:list` 权限：工位列表页 `[⚙ 映射]` 按钮 **不渲染**（避免无效入口）
  - 无 `ops:location:alias:add` 权限：Drawer 内"新增映射"按钮 disabled + Tooltip 提示"无权限"
  - 无 `ops:location:alias:delete` 权限：Drawer 内"删除"按钮 hidden（Drawer 是只读视图）
- **后端兜底**：权限中间件按 `pkg/middleware/permission.go` 标准模式检查（与现有 ops 端点一致）

### Claude's Discretion

以下方面由实现者决定：
1. **migration 中 scope 字段长度**：选 `VARCHAR(32)` 是估计值，实际可调
2. **alias 列表分页**：用现有 pageSize=10 还是 pageSize=20，按工位数据量决定
3. **Drawer 宽度**：建议 600px（不挤压主表格），可调
4. **operlog 记录格式**：是否在 `request_param` 字段中记录 dept_name/location_name（便于审计可读性），建议记录
5. **三级校验中"祖先关系"的具体实现**：用 `ancestors LIKE '%X%'` 还是 `string_to_array(ancestors, ',') @> ARRAY[?]`（PostgreSQL 数组包含查询）
6. **alias 删除策略**：软删除（GORM `deleted_at`）还是硬删除 — 推荐软删除（与项目一致）
7. **operlog Record vs RecordWithBody**：alias CRUD 不涉及敏感字段，使用 `Record` 即可

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 本阶段设计依据（必读）
- `.planning/phases/39-workstation-dept-location-alias/39-SPEC.md` — **Locked requirements — MUST read before planning**（12 需求 + 18 项 acceptance criteria + 12 项 OUT 边界）
- `.planning/phases/39-workstation-dept-location-alias/39-CONTEXT.md` — 本文件（HOW 决策）
- `.planning/ROADMAP.md` Phase 39 条目 — locked phase 边界

### SPEC.md 与 discuss-phase 沿用决策（必读）
- `.planning/phases/37-dept-select-unify/37-CONTEXT.md` — 部门选择收敛（D-LOCKED `buildTreeData` + `trimTitleToLastSegment` 双向语义、`useDeptTree` canonical 数据源、`DepartmentTreeSelect` 受控模式、AD 域控 OU 树独立于系统部门树）
- `.planning/phases/28-workstation-device-association/28-CONTEXT.md` — 工位子表格（D-09 新表模式、D-27/D-28 权限、D-33/D-34 operlog 复用）

### 数据模型
- `internal/models/workstation.go` — `Workstation` 模型，含 `DeptID *string`、`UserID *string`、`BuildingID *string`、`FloorID *string`（**不变动**）
- `internal/models/sys_dept.go`（参考，未直接读但 SPEC 中已锁定 `isExternalOrg` 字段语义）

### 后端标准件（待改造/复用）
- `internal/services/operations/workstation_service.go:14-18` — `workstationJoinClause` / `workstationJoinSelect` 常量（**不动**，新建独立方法）
- `internal/services/operations/building_service.go` — Handler-Service 模式参考（含 `validateOrg()` UUID 校验）
- `internal/api/v1/operations/workstation_handler.go` — 工位 handler，新 alias handler 并列
- `internal/api/router.go` — 路由注册入口（新增 4 个 alias 路由 + 权限）
- `internal/core/db/migrations/migration_164_phase38_verify_admin_migrated.go` — 最新 migration 模式（log.Println 风格 + 幂等）
- `internal/utils/operlog/` — `Record()` / `RecordWithBody()` / 25 OperType 常量集 / 34 sensitiveKeys（CLAUDE.md 强制约定）
- `pkg/middleware/permission.go` — 权限中间件参考
- `pkg/response/` — 响应包装

### 前端标准件（待改造/复用）
- `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:87-92` — **`subDeptTree` 派生逻辑**，Phase 39 主要注入点
- `xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts:103-106` — `deptTreeData` 派生（来自 `useDeptTree()`）
- `xingran-react-frontend/src/hooks/useDeptTree.ts` — canonical 部门树数据源（5min stale / 30min gc / `useInvalidateDept()` 失效）
- `xingran-react-frontend/src/utils/deptUtils.ts` — `findDeptNode`、`trimTitleToLastSegment`、`filterExternalOrgDepts`、`toFullPathTree`、`DeptLikeNode`
- `xingran-react-frontend/src/lib/opsApi.ts` — Operations 模块 API 工厂，新增 `locationAliasApi`
- `xingran-react-frontend/src/lib/queryKeys.ts`（或等价）— queryKey 定义，添加 `locationAlias` 段
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — list 页，添加 `[⚙ 映射]` 按钮 + Drawer
- `xingran-react-frontend/src/components/modal/BaseEditModal.tsx` — 通用编辑 Modal 样式参考（用于 alias Drawer）

### Operlog 参考
- `.planning/phases/34-oper-log-full-coverage/34-CONTEXT.md`（如有）— operlog 集成规范
- `internal/utils/operlog/regression_test.go` — 25 OperType 常量集守护测试

### 缓存失效参考
- `internal/services/dept_cache_service.go`（参考）— dept cache invalidation 接口签名

### AD 域控对照（避免误改）
- `internal/services/addomain/` — AD OU 树独立于系统部门树（memory `ad-domain-ou-tree-separate-from-sys-dept`），本 phase 不动

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`useDeptTree()` hook** (canonical, Phase 37) — 全项目部门树单一数据源，alias 改造前端直接消费
- **`deptUtils.findDeptNode(tree, id)`** — 按 ID 查找部门树节点，alias 节点定位复用
- **`deptUtils.trimTitleToLastSegment(tree)`** — 全路径 → 短名收窄，alias 节点 title 处理复用
- **`deptUtils.filterExternalOrgDepts(tree)`** — 仅外部机构过滤，alias location TreeSelect 数据源复用
- **`operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeCreate)`** — 标准 5 参调用模式
- **Ant Design `Drawer` 组件** — Drawer UI 模板
- **Ant Design `Form` + `Form.Item` `validateStatus` + `help`** — 实时校验提示

### Established Patterns
- **Handler-Service 模式** — service interface + private impl + 构造函数
- **migration 命名** — `migration_NNN_{slug}.go` + `MigrateNNN{...}(db *gorm.DB) error` + `log.Println`
- **operlog 记录** — handler success path 末尾调用
- **权限 gating** — 后端 middleware 检查 + 前端 `hasPermission` hook 渲染
- **缓存失效** — 写操作成功后调现有 invalidation 接口
- **React Query queryKey 规范** — `queryKeys.{domain}.{subkey}(...)`

### Integration Points
- **后端入口**：`internal/api/router.go` 注册新 alias router（与现有 ops 路由并列）
- **后端 service**：`internal/services/operations/workstation_service.go` 添加 `GetWorkstationDeptOptions` 方法（不修改现有 `workstationJoinClause`）
- **后端 model**：新增 `internal/models/sys_dept_location_alias.go`
- **后端 migration**：`internal/core/db/migrations/migration_165_sys_dept_location_alias.go`
- **前端 Drawer 入口**：`pages/operations/workstations/index.tsx` 工具栏新增按钮 + state 管理
- **前端注入点**：`pages/operations/workstations/modals/EditModal.tsx:87-92` `subDeptTree` 派生
- **前端 hook**：新增 `hooks/useAliasByLocation.ts`（React Query）
- **前端 API**：`lib/opsApi.ts` 新增 `locationAliasApi`
- **权限 seed**：migration_165 内同步 INSERT `sys_menu` 4 条记录

### Known Constraints
- **Phase 37 D-LOCKED**：不能破坏 `deptTreeData`（全路径）+ `orgTreeData`（短名）双向语义
- **Phase 34 operlog 约定**：所有业务写操作必须 record；使用 25 OperType 常量；不引入 OperTypeOther 兜底
- **CLAUDE.md 强制**：`response.Success()` / `response.Error()` 包装；UUID Foreign Key；Status Convention 0/1
- **不能修改**：`sys_workstation` 表 / `sys_dept.parent_id` / `dept_tree` 渲染逻辑 / user picker / 报表

</code_context>

<specifics>
## Specific Ideas

### alias 节点 title 拼接示例
```
deptName = "运营服务部/子部门A"
trimmed = trimTitleToLastSegment([{title: deptName}])[0].title  // = "子部门A"
final = `${trimmed} [映射]`  // = "子部门A [映射]"
```

### alias Drawer 新增表单字段顺序（D-02 决策落实）
```tsx
<Form layout="vertical">
  <Form.Item label="所属部门（原组织）" name="deptId" rules={[{ required: true }]}>
    <TreeSelect treeData={fullDeptTree} ... />
  </Form.Item>
  <Form.Item label="物理位置" name="locationId" rules={[{ required: true }]}>
    <TreeSelect treeData={orgTreeDataOnlyExternal} ... />
  </Form.Item>
  <Form.Item label="scope" name="scope" initialValue="workstation" hidden>
    <Input />
  </Form.Item>
  <Form.Item label="备注" name="remark">
    <Input.TextArea rows={2} />
  </Form.Item>
</Form>
```

### operlog 记录示例
```go
// success path 末尾
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
  "工位管理", operlog.OperTypeCreate)
response.Success(c, alias)
```

### 三级校验 service 层实现（D-02 决策落实）
```go
func (s *locationAliasServiceImpl) validateAlias(ctx context.Context, alias *models.SysDeptLocationAlias) error {
    // 1. 防自映射
    if alias.DeptID == alias.LocationID {
        return errors.New("dept_id 与 location_id 不能相同")
    }
    // 2. location 必须是外部机构
    var location models.SysDept
    if err := s.db.WithContext(ctx).Where("id::text = ?", alias.LocationID).First(&location).Error; err != nil {
        return errors.New("物理位置部门不存在")
    }
    if location.IsExternalOrg != 1 {
        return errors.New("物理位置必须是外部机构（isExternalOrg=1）")
    }
    // 3. dept 必须是 location 的后代
    var dept models.SysDept
    if err := s.db.WithContext(ctx).Where("id::text = ?", alias.DeptID).First(&dept).Error; err != nil {
        return errors.New("原组织部门不存在")
    }
    // ancestors 字段是逗号分隔的 id 列表，包含该部门所有祖先
    if !strings.Contains(","+dept.Ancestors+",", ","+alias.LocationID+",") {
        return errors.New("原组织必须是物理位置的后代部门")
    }
    return nil
}
```

### 工位部门下拉 union SQL 示例（D-06 决策落实）
```sql
-- 单个 query 完成 union（替代两次查询拼接）
SELECT d.id::text AS dept_id,
       d.dept_name AS dept_name,
       false AS is_alias
FROM sys_dept d
WHERE d.deleted_at IS NULL
  AND (',' || d.ancestors || ',') LIKE ('%,' || $1 || ',%')  -- 子孙节点
  AND d.is_external_org = 0  -- 不含外部机构本身
UNION ALL
SELECT d.id::text AS dept_id,
       d.dept_name AS dept_name,
       true AS is_alias
FROM sys_dept_location_alias a
JOIN sys_dept d ON d.id::text = a.dept_id
WHERE a.deleted_at IS NULL
  AND a.location_id = $1
  AND a.scope = 'workstation'
```

### 前端 subDeptTree 注入示例（D-06 决策落实）
```typescript
// pages/operations/workstations/modals/EditModal.tsx
const watchedOrgId = Form.useWatch("orgId", form) as string | undefined;

// 新增：拉取该 location 下的所有 alias
const { data: aliasList = [] } = useAliasByLocation(watchedOrgId);

const subDeptTree = useMemo<DeptTreeNode[]>(() => {
  if (!watchedOrgId) return [];
  const node = findDeptNode(deptTreeData, watchedOrgId);
  const baseTree = node?.children ? trimTitleToLastSegment(node.children) : [];

  if (!aliasList.length) return baseTree;

  // 将 alias 节点附加到末尾，每个节点带 [映射] 后缀
  const aliasNodes: DeptTreeNode[] = aliasList.map(a => {
    const trimmedTitle = trimTitleToLastSegment([{ id: a.dept_id, title: a.dept_name, children: [] }])[0]?.title || a.dept_name;
    return {
      id: a.dept_id,
      title: `${trimmedTitle} [映射]`,
      isLeaf: true,
      value: a.dept_id,
      is_alias: true,  // 标记，前端按需使用
    };
  });
  return [...baseTree, ...aliasNodes];
}, [deptTreeData, watchedOrgId, aliasList]);
```

</specifics>

<deferred>
## Deferred Ideas

以下想法在 discuss 阶段被识别但不属于本 phase 范围：

- **跨模块扩展（building/floor/infopoint）**：scope 字段已预留支持，未来如果其他位置类实体有同类问题，可新建 phase 直接复用 `sys_dept_location_alias` 表
- **alias 批量导入**：Excel 批量导入 alias 映射（如运营服务部/子部门A~Z 一次性映射到各中心支公司），本期不做
- **alias 历史版本**：映射变更历史记录（不仅是 operlog，还要保留映射变更轨迹），本期通过 operlog 满足审计需求
- **alias 自动推荐**：基于用户部门历史工位数据，自动建议可能的映射（如某 dept 用户长期在工位 scope 下绑定 location X），本期不做
- **workstation org_id 历史映射数据回填工具**：批量把运营服务部相关人员的工位 dept_id 改为映射后的 dept，本期工具暂不提供（业务侧在 Drawer 内手动改）

---

*Phase: 39-workstation-dept-location-alias*
*Context gathered: 2026-06-25*
*Next step: /gsd:plan-phase 39*
</content>
</invoke>
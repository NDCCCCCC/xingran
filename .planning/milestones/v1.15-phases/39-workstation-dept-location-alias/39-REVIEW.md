---
phase: 39-workstation-dept-location-alias
reviewed: 2026-06-25T00:00:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - internal/models/sys_dept_location_alias.go
  - internal/core/db/migrations/migration_165_sys_dept_location_alias.go
  - internal/core/db/database.go
  - internal/services/operations/location_alias_service.go
  - internal/services/operations/location_alias_service_test.go
  - internal/api/v1/operations/location_alias_handler.go
  - internal/services/operations/workstation_service.go
  - internal/api/v1/operations/workstation_handler.go
  - internal/api/router.go
  - xingran-react-frontend/src/lib/opsApi.ts
  - xingran-react-frontend/src/lib/queryKeys.ts
  - xingran-react-frontend/src/hooks/useAliasByLocation.ts
  - xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx
  - xingran-react-frontend/src/pages/operations/workstations/index.tsx
  - xingran-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx
findings:
  critical: 3
  warning: 7
  info: 4
  total: 14
status: issues_found
---

# Phase 39: Code Review Report

**Reviewed:** 2026-06-25
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

Phase 39 "工位部门物理位置映射" 整体落地完整,模型/迁移/服务/handler/router/前端 hook/Drawer 全链路打通,operlog 与缓存失效约定基本到位,SQL 全部走 GORM 占位符参数化(无字符串拼接注入面)。但发现 **3 个 BLOCKER**:

1. **LocationAliasDrawer 分页完全失效** — handler 从 query string 读 `current`,前端从 POST body 发 `pageNum`,参数名+通道双错位,Drawer 切页永远是第 1 页。
2. **GetWorkstationDeptOptions 在 SQLite 端必崩** — Raw SQL 用了 PG 专有 `id::text` cast 语法,SQLite 不识别 `::` cast,会在测试/SQLite 部署环境直接报语法错误。
3. **alias 写操作后的 dept 缓存失效顺序错位 + service 没用事务** — `Create` 走 validate-then-insert 非原子,部分唯一索引冲突会把底层 GORM error 原样透传成 500,而非 400 友好提示。

其余 7 个 WARNING 集中在:权限门禁过宽(D-04 副作用)、Update 路径在 scope 单字段变更时漏跑重名校验、Delete 错误码泄漏、`as DeptTreeNode` 强转、union 查询未分页/未排序等。

---

## Critical Issues

### CR-01: LocationAliasDrawer 分页完全失效 — handler 与前端参数通道+名称双错位

**File:** `internal/api/v1/operations/location_alias_handler.go:77-98`
**File:** `xingran-react-frontend/src/lib/opsApi.ts:150-155`
**Issue:**

handler 仅从 **query string** 读取分页参数,且字段名为 `current`:
```go
if v := c.Query("current"); v != "" { ... pageNum = n }
if v := c.Query("pageSize"); v != "" { ... pageSize = n }
```

但前端 `locationAliasApi.list` 把 `pageNum`/`pageSize` 放进 **POST body** 发送:
```ts
list: async (params: { pageNum?: number; pageSize?: number } = {}) => {
  return await post<PageResponse<LocationAlias>>("/ops/location-alias/list", {
    pageNum: params.pageNum ?? 1,
    pageSize: params.pageSize ?? 10,
  });
}
```

后果:
- Drawer (`LocationAliasDrawer.tsx:64-71`) 调 `locationAliasApi.list({ pageNum, pageSize })` 切到第 2 页时,handler 读不到 `?current=2`,永远返回第 1 页数据。
- 后端 `result.Total` 是真实总数(可能 50+),前端分页器显示有 N 页,但点任何页码都回到第 1 页 — 用户体验上是"假分页"。
- `aliasListQuery` 结构体(handler.go:71-74)定义了 `json:"current" form:"current"` 标签却从未被 `ShouldBindJSON` 调用,是死代码。

**Fix:** 二选一,推荐改 handler 读取 body 与其他 ops 模块对齐(项目约定 `POST /list` body 传参):

```go
// 方案 A: handler 改为读 body
func (h *LocationAliasHandler) List(c *gin.Context) {
    var q aliasListQuery
    _ = c.ShouldBindJSON(&q)
    pageNum := q.Current
    if pageNum <= 0 { pageNum = DefaultCurrent }
    pageSize := q.PageSize
    if pageSize <= 0 { pageSize = DefaultPageSize }
    pageSize = clampPageSize(pageSize)
    result, err := h.service.List(c.Request.Context(), pageNum, pageSize)
    ...
}

// 同时把 aliasListQuery 字段对齐前端:
type aliasListQuery struct {
    PageNum  int `json:"pageNum"`
    PageSize int `json:"pageSize"`
}
```

或方案 B: 前端把 `pageNum/pageSize` 改为 query string:`post("/ops/location-alias/list?current=" + pageNum + "&pageSize=" + pageSize, {})`,但与项目其他 `/list` 风格不一致,不推荐。

---

### CR-02: GetWorkstationDeptOptions 的 Raw SQL 用 PG 专有 `::text` cast,SQLite 环境必崩

**File:** `internal/services/operations/workstation_service.go:92-105`
**Issue:**

```go
err := s.db.WithContext(ctx).Raw(`
    SELECT id::text AS dept_id, ...                -- PG 专有 cast
    FROM sys_dept
    WHERE ...
      AND ((',' || ancestors || ',') LIKE ('%,' || ? || ',%') OR id::text = ?)  -- PG 专有 cast
    UNION ALL
    SELECT a.dept_id, d.dept_name, true AS is_alias
    FROM sys_dept_location_alias a
    JOIN sys_dept d ON d.id::text = a.dept_id      -- PG 专有 cast
    WHERE ...
`, orgId, orgId, orgId).Scan(&result).Error
```

PostgreSQL `::text` 是 PG 历史语法,SQLite(本项目 SQLite 用于单元测试与 `config.database.type: "sqlite"` 备选部署) **不支持 `::` cast 操作符**,会直接报 `near "::": syntax error`。后果:

1. 任何在本机用 SQLite 跑后端的开发/演示环境,工位编辑模态框"所属部门"下拉会 500。
2. 与 `location_alias_service.go:233-237` 的注释("为保证 PG/SQLite 双 DB 行为一致,这里走 db.Where("id = ?", ...)") 自相矛盾 — 同一 Phase 39 的两个查询,一个刻意避开 `::text`,另一个反而大量用它。

**Fix:** 用 ANSI SQL 的 `CAST(... AS TEXT)` 替换所有 `::text`(PG/SQLite 双兼容):

```go
err := s.db.WithContext(ctx).Raw(`
    SELECT CAST(id AS TEXT) AS dept_id, dept_name AS dept_name, false AS is_alias
    FROM sys_dept
    WHERE deleted_at IS NULL
      AND is_external_org = 0
      AND ((',' || ancestors || ',') LIKE ('%,' || ? || ',%') OR CAST(id AS TEXT) = ?)
    UNION ALL
    SELECT a.dept_id, d.dept_name, true AS is_alias
    FROM sys_dept_location_alias a
    JOIN sys_dept d ON CAST(d.id AS TEXT) = a.dept_id
    WHERE a.deleted_at IS NULL
      AND a.scope = 'workstation'
      AND a.location_id = ?
`, orgId, orgId, orgId).Scan(&result).Error
```

注:`workstation_service.go:17` 既有 `workstationJoinClause` 也用了 `sys_dept.id::text = sys_workstation.dept_id`,是同一类问题,但属于既有代码(不在本次 Phase 39 scope);本次新增的 `GetWorkstationDeptOptions` 必须修复。

---

### CR-03: alias Create/Update validate-then-write 非原子 + 部分唯一索引冲突透传 500

**File:** `internal/services/operations/location_alias_service.go:134-160` (Create), `167-207` (Update)
**Issue:**

`Create` 流程: `validateAlias` (3 条 SELECT) → `db.Create`。两步无事务,且 `validateAlias` 不检查 `(dept_id, scope)` 唯一性(只查自映射/外部机构/ancestor)。

并发场景:
- 两个请求同时为同一 `(dept_id=A, scope=workstation)` 创建 alias,都通过 validateAlias,然后同时 INSERT,PG 的 partial unique index `idx_sys_dept_location_alias_dept_scope` 拒绝第二个,底层 GORM/PG 错误为 `ERROR: duplicate key value violates unique constraint "idx_sys_dept_location_alias_dept_scope"`。
- 该 error 经 `fmt.Errorf("创建别名映射失败: %w", err)` 包装后,handler 透传成 `400 Bad Request + err.Error()`(`location_alias_handler.go:111`)。

后果:
1. 错误信息泄漏底层 PG 约束名 + SQL 状态(`duplicate key value violates unique constraint "idx_sys_dept_location_alias_dept_scope"`),信息泄漏面虽小但不专业。
2. handler 用 `http.StatusBadRequest` 实际上是对的,但消息对终端用户不友好(应返回"该部门在此场景下已存在映射")。
3. `Update` 路径当只改 `Scope` 单字段时(`deptChanged=false && locationChanged=false`),不跑任何重名校验,直接 `Save` 撞唯一索引 → 同样的底层错误透传。

**Fix:** service 层显式预检 + 友好错误,并用事务保证一致性:

```go
// Create 内,validateAlias 之后:
var existingCount int64
if err := s.db.WithContext(ctx).Model(&models.SysDeptLocationAlias{}).
    Where("dept_id = ? AND scope = ? AND deleted_at IS NULL", alias.DeptID, alias.Scope).
    Count(&existingCount).Error; err != nil {
    return nil, fmt.Errorf("校验唯一性失败: %w", err)
}
if existingCount > 0 {
    return nil, errors.New("该部门在此场景下已存在映射,请勿重复创建")
}
```

并在 `Update` 当 `req.Scope != nil` 时(无论 dept/location 是否变更)也跑一次相同预检(排除自身 ID):

```go
if req.Scope != nil {
    var cnt int64
    s.db.WithContext(ctx).Model(&models.SysDeptLocationAlias{}).
        Where("dept_id = ? AND scope = ? AND id <> ? AND deleted_at IS NULL",
            existing.DeptID, *req.Scope, id).Count(&cnt)
    if cnt > 0 {
        return errors.New("目标 scope 下已存在该部门的映射")
    }
}
```

---

## Warnings

### WR-01: router.go 用 `RequirePermissions` 同时挂 4 个 alias 权限 + D-04 不授权任何角色 → UAT 前所有用户对 alias 接口 403

**File:** `internal/api/router.go:608-614`
**Issue:** `locationAlias.Use(middleware.RequirePermissions([]string{ "ops:location:alias:list", "ops:location:alias:add", "ops:location:alias:edit", "ops:location:alias:delete" }, core))`。migration 165 严格遵循 D-04 决策"不 INSERT 任何 sys_role_menu"。两个决策叠加效果:在管理员手动授权前,**包括超管在内的所有用户调 `/ops/location-alias/list` 都会 403**(`RequirePermissions` 通常要求至少持有一个 perm;若要求同时持有 4 个,门禁更严)。

前端 `LocationAliasDrawer` 已通过 `canListAlias` 隐藏入口,但即便入口隐藏,Phase 39 的 UAT 验证者必须先执行手动授权才能测 Drawer CRUD。这是 D-04 的有意决策,但需在交付文档显著标注"启用前需管理员授权 4 个 perms",否则会被误判为 bug。

**Fix:** 文档化即可,或为超管角色(`admin` / role_id=1)在 migration 165 内显式 INSERT sys_role_menu 一行兜底(与 D-04 决策冲突,需产品确认)。最低成本:在 SPEC/Plan 39-08 的 UAT 步骤里把"管理员手动授权 4 perms"作为前置步骤。

---

### WR-02: `Update` 仅在 dept/location 变更时跑 validateAlias,scope 变更或初始非法数据无防线

**File:** `internal/services/operations/location_alias_service.go:177-200`
**Issue:** 当前逻辑:`deptChanged || locationChanged` 才跑 `validateAlias`。两个隐患:

1. 既有 alias 记录可能在 Phase 39 上线前已通过其他渠道(例如直接 SQL)插入,其 `dept_id`/`location_id` 关系可能已违反三级校验(如 sys_dept 后被移动到别处,ancestor 关系失效)。用户只改 `Remark` 时不会触发重新校验,非法关系继续残留。
2. 若只改 `Scope`(如 workstation → floor),不重新跑 ancestor 校验。虽然 scope 单字段变更理论上不影响 dept/location 关系,但缺少"任何更新都重新校验"的简单不变量守护,容易在未来扩展时被遗漏。

**Fix:** 简化为"任何 Update 都跑 validateAlias",代价只是多 2 条 SELECT:

```go
// 删掉 deptChanged/locationChanged 判定,直接:
if err := s.validateAlias(ctx, &existing); err != nil {
    return err
}
```

---

### WR-03: Delete handler 把 GORM "record not found" 当 500 返回,且 err.Error() 直接透传

**File:** `internal/api/v1/operations/location_alias_handler.go:143-155`
**File:** `internal/services/operations/location_alias_service.go:210-212`
**Issue:** `Delete` service 用 `db.Where(...).Delete(...)` 直接返回 error。若 id 不存在,GORM/PG 不会报错(影响 0 行),所以这是 silent no-op 而非 500。但若 id 格式非法(例如非 UUID),PG 会报 `invalid input syntax for type uuid`,该 error 原样透传给前端为 `500 + err.Error()`,泄漏底层 PG 错误描述。

handler.Delete 一律返回 500(`response.Error(c, http.StatusInternalServerError, err.Error())`),对"记录不存在"的友好场景缺乏 404 分支。

**Fix:** service 层用 `RowsAffected` 区分:

```go
func (s *locationAliasServiceImpl) Delete(ctx context.Context, id string) error {
    res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.SysDeptLocationAlias{})
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}
```

handler 内对 `errors.Is(err, gorm.ErrRecordNotFound)` 返回 404,其余返回 500 但不透传 `err.Error()`(改用通用消息)。

---

### WR-04: union 查询无分页、无排序、无去重保护,可能返回重复 deptId

**File:** `internal/services/operations/workstation_service.go:84-107`
**Issue:**

1. **无分页**: `orgId` 子树 + alias 命中可能合计数百条,前端 TreeSelect 渲染所有节点,大数据量场景卡顿。
2. **无排序**: UNION ALL 不保证顺序,前端展示随机。
3. **潜在重复**: 若 `dept_id` 既是 `orgId` 的子孙(`is_external_org=0`)又通过 alias 命中(同一 dept 在 `sys_dept_location_alias` 有 `location_id=orgId` 的记录),该 deptId 在两个分支都出现,前端 TreeSelect 用 `value` 去重可能告警。

**Fix:** 加 `ORDER BY dept_name` 并/或用 `UNION`(而非 `UNION ALL`)去重;若担心 alias 那条 `is_alias=true` 信息丢失,可在前端按 `deptId` 去重(优先保留 alias 条目)。短期最小修复:在 SQL 末尾加 `ORDER BY 2` (按 dept_name 排序)。

---

### WR-05: handler.invalidateDeptCache 失败仅 warn,但 operlog.Record 仍按成功路径记录 — 审计与实际不一致

**File:** `internal/api/v1/operations/location_alias_handler.go:60-68, 115-119`
**Issue:** D-03 决策为"缓存失效失败不阻断业务",实现上 `invalidateDeptCache` 仅 `applogger.Warnf`。但随后的 `operlog.Record` 把整个请求记录为成功(OperTypeCreate + module "工位管理"),审计日志显示"映射创建成功",实际可能 dept 缓存仍含旧 union 结果,导致后续工位下拉短暂不一致。审计粒度无法区分"业务成功+缓存失效失败"场景,排障困难。

**Fix:** 在 operlog 的 `remark` 或额外字段记录缓存失效状态;或把 warn 升级为 error 但不阻断 response(仅影响审计可观测性)。最低成本:`applogger.Warnf` 的消息里带上 request_id 便于关联。

---

### WR-06: EditModal.tsx 用 `as DeptTreeNode` + `as Record<string, unknown>` 双层强转,绕过类型检查

**File:** `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:106-120`
**Issue:**

```tsx
const aliasNodes: DeptTreeNode[] = aliasList.map((a) => {
  ...
  return {
    title: `${trimmed} [映射]`,
    value: a.deptId,
    key: a.deptId,
    isLeaf: true,
    ...({ is_alias: true } as Record<string, unknown>),
  } as DeptTreeNode;       // <-- 强转,DeptTreeNode 接口可能不含 is_alias/isLeaf 字段
});
```

若 `DeptTreeNode` 接口未声明 `isLeaf` / `is_alias`,运行时虽然能跑(JS 宽松),但 TS 强转绕过了类型守护,未来 DeptTreeNode 字段重命名时这里不会报错。

**Fix:** 在 `DeptTreeNode` 类型定义中显式声明可选字段 `isLeaf?: boolean` 与 `is_alias?: boolean`,然后移除两处 `as`。

---

### WR-07: EditModal.tsx 的 `useMemo` 依赖 `aliasList`,但 aliasList 来自父组件 query 结果,引用每次 query refetch 都变 → useMemo 失效

**File:** `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:93-123`
**Issue:** `useMemo(() => {...}, [deptTreeData, watchedOrgId, aliasList])`。父组件 `index.tsx:88` 的 `const { data: aliasList = [] } = useAliasByLocation(watchedOrgId)` — react-query 每次后台 refetch(如窗口聚焦、staleTime 到期)会返回新数组引用,即便内容相同,useMemo 也会重算,触发 TreeSelect 重渲染。性能影响有限,但与 react-query 最佳实践(用 `select` 或结构化比较)不一致。

**Fix:** 可接受现状(staleTime=5min,refetchOnWindowFocus=false,影响很小),或用 `useMemo(() => aliasList, [aliasList?.map(a => a.deptId).join(',')])` 做内容签名。

---

## Info

### IN-01: migration_165 注释自相矛盾 — "7 列表" vs "BaseModel 7 + 业务 4 = 11"

**File:** `internal/core/db/migrations/migration_165_sys_dept_location_alias.go:26-32`
**Issue:** 注释先写"CREATE TABLE sys_dept_location_alias(7 列 + 软删除)",紧接着又写"实际列数 = BaseModel 7 + 业务 4 = 11"。两段描述对不上,后续维护者读会困惑。**Fix:** 统一表述,直接写"11 列(BaseModel 7 + 业务 4: dept_id/location_id/scope/remark)"。

---

### IN-02: `useAliasByLocation` 的 queryKey 在 locationId 为空字符串与 undefined 间无区分

**File:** `xingran-react-frontend/src/hooks/useAliasByLocation.ts:24, 29`
**Issue:** `queryKey: queryKeys.locationAlias.byLocation(locationId ?? "")`,同时 `enabled: !!locationId`。undefined 和 "" 都映射到同一个 key `["location-alias", "by-location", ""]`,但因为 disabled 不会发请求,缓存里这个 key 永远没有数据 — 无害,但语义不清晰。**Fix:** 可在 key 里用 `locationId ?? null` 并在 factory 里接受 `string | null`,语义更清晰。

---

### IN-03: locationAliasApi.list 字段命名 `pageNum` 与项目其他模块的 `current` 不一致

**File:** `xingran-react-frontend/src/lib/opsApi.ts:150-155`
**Issue:** 项目 CLAUDE.md Pagination Convention 规定 request 字段为 `current`/`pageSize`。本 Phase 39 新增的 `locationAliasApi.list` 用 `pageNum`,破坏一致性。**Fix:** 统一改为 `current`(与 CR-01 的修复一起做)。

---

### IN-04: locationAliasService 的 List JOIN 子句注释提到 PG `id::text` 但代码实际用 `id = ?`

**File:** `internal/services/operations/location_alias_service.go:19-27`
**Issue:** 注释先讨论"`id::text = ?` 把 uuid 转 text 再比较",然后说"此处使用裸 `id = ?`"。代码确实用裸 `id = ?`,但注释留下"曾经考虑过 ::text"的痕迹,对读者产生噪音。**Fix:** 精简注释,只保留"使用裸 `id = ?`,由 GORM 按 driver 适配"。

---

_Reviewed: 2026-06-25_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

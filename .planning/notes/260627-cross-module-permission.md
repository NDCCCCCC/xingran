# 跨模块权限边界声明 — ops/workstation ↔ asset/reconciliation

**记录日期**: 2026-06-27
**背景**: v0.3 架构中，工位详情页 (`/ops/workstation/:id`) 需要显示对账健康度，需调用 asset 模块的对账服务
**关联决策**: [.planning/notes/asset-reconciliation-strategy.md](./asset-reconciliation-strategy.md) §7.5, §15
**关联记忆**: `xingran-perm-namespace-split-readonly-page`

---

## 1. 跨模块调用关系

```
┌──────────────────────────────────────────────────────────┐
│  ops 模块（运维管理）                                       │
│  ┌────────────────────────────────────────────────────┐  │
│  │ 工位详情页 /ops/workstation/:id                     │  │
│  │ Handler: operations.WorkstationHandler.GetByID()   │  │
│  └─────────────────────┬──────────────────────────────┘  │
│                        │ 跨模块调用（非 HTTP）            │
│                        ▼                                  │
│  ┌────────────────────────────────────────────────────┐  │
│  │ asset 模块（资产管理）                               │  │
│  │ Service: asset.ReconciliationService                │  │
│  │ Method: GetByWorkstationID(ctx, wsID)              │  │
│  │ 权限: asset:reconciliation:list                     │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

---

## 2. 权限边界设计

### 2.1 设计原则

| 场景 | 行为 |
|------|------|
| 用户在 ops 模块（持有 `ops:workstation:*`） | 可看工位详情 |
| 用户同时持有 `asset:reconciliation:list` | 可看对账健康度（完整功能） |
| 用户仅持有 ops 权限 | 可看工位详情但**健康度卡片降级隐藏**（不报错、不留占位） |
| 用户仅持有 asset 权限 | 可直接访问 `/asset/reconciliation/*` 完整功能 |

### 2.2 路由级中间件

工位路由组的中间件用 `RequirePermissionsWithQuery` 扩展读权限：

```go
// internal/api/router.go（修改）
workstations := ops.Group("/workstation")
workstations.Use(middleware.RequirePermissionsWithQuery([]string{
    "ops:workstation:list",          // 主权限
    "ops:workstation:add",
    "ops:workstation:edit",
    "ops:workstation:delete",
}, []string{
    "ops:building:spaces:list",     // 已有：楼宇空间读权限
    "asset:reconciliation:list",     // 🆕 跨模块：对账读权限
}, core))
```

**效果**：
- `POST /ops/workstation/list` → 命中 queryExtraPermissions → 通过
- `POST /ops/workstation`（创建）→ 严格权限 `ops:workstation:add` → 拒绝（如无权限）

### 2.3 Handler 内权限降级（更优雅）

避免路由级一刀切，在 `WorkstationHandler.GetByID()` 内做降级：

```go
func (h *WorkstationHandler) GetByID(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")
    
    // 1. 查询工位基本信息（无额外权限要求，路由级中间件已保护）
    ws, err := h.workstationService.GetByID(ctx, id)
    if err != nil {
        response.Error(c, http.StatusNotFound, "工位不存在")
        return
    }
    
    // 2. 跨模块查询对账健康度（权限降级：无权限不报错）
    if h.hasReconciliationPerm(c) {
        recon, err := h.reconciliationService.GetByWorkstationID(ctx, id)
        if err == nil {
            ws.Reconciliation = recon
            ws.ReconciliationVisible = true
        } else {
            // 服务异常时静默降级（不暴露内部错误）
            ws.ReconciliationVisible = false
        }
    } else {
        ws.ReconciliationVisible = false
        ws.ReconciliationHiddenReason = "无资产对账查看权限"
    }
    
    response.Success(c, ws)
}

// hasReconciliationPerm 内部权限检查
func (h *WorkstationHandler) hasReconciliationPerm(c *gin.Context) bool {
    userID, _ := c.Get("userID")
    return h.core.PermissionService.UserHasPerm(userID, "asset:reconciliation:list")
}
```

**优势**：
- ✅ 用户无权限时不返回 403（避免破坏路径）
- ✅ UI 可根据 `ReconciliationVisible` 决定渲染/隐藏
- ✅ 服务异常时不暴露内部错误（静默降级）
- ✅ 仍保留路由级权限（无任何权限直接 403）

### 2.4 前端对应处理

```typescript
// src/pages/operations/workstation/[id].tsx
const WorkstationDetailPage = () => {
  const { id } = useParams();
  const { data: ws } = useWorkstationDetail(id);
  
  // 🆕 根据 ReconciliationVisible 决定渲染
  return (
    <PageContainer>
      <BasicInfoCard data={ws} />
      
      {ws.ReconciliationVisible ? (
        <HealthCard workstationId={id} />
      ) : (
        // 完全不渲染，不留占位
        null
      )}
      
      <Tabs>
        <TabPane tab="资产设备" key="asset">
          <AssetDevicesTable 
            extraColumns={[
              {
                title: '对账健康',
                render: (record) => (
                  <HealthBadge assetId={record.id} />
                ),
              },
            ]}
          />
        </TabPane>
      </Tabs>
    </PageContainer>
  );
};
```

---

## 3. 菜单权限矩阵

| 资源 | 路径 | 必需权限 | 缺权限行为 |
|------|------|---------|-----------|
| 资产对账-看板 | `/asset/reconciliation/dashboard` | `asset:reconciliation:dashboard` | 403 |
| 资产对账-异常列表 | `/asset/reconciliation/exceptions` | `asset:reconciliation:list` | 403 |
| 资产对账-例外规则 | `/asset/reconciliation/exception-rules` | `asset:reconciliation:exception:list` | 403 |
| 资产对账-例外新建 | `/asset/reconciliation/exception-rules` 新建按钮 | `asset:reconciliation:exception:create` | 按钮禁用 |
| 资产对账-例外更新 | 同上 编辑按钮 | `asset:reconciliation:exception:update` | 按钮禁用 |
| 资产对账-例外删除 | 同上 删除按钮 | `asset:reconciliation:exception:delete` | 按钮禁用 |
| **工位详情页-健康度卡片** | `/ops/workstation/:id` | （无独立权限，依赖路由级中间件） | **静默隐藏** |
| **资产详情页-对账摘要** | `/asset/card/:id` | （无独立权限） | **静默隐藏** |
| **自动转工单** | 对账引擎内部触发 | `ops:workorder:add` | 工单创建失败 |

---

## 4. RBAC 建议角色配置

### 4.1 系统管理员（super admin）

- 拥有所有 `*:*:*` 权限
- 不受任何权限降级影响

### 4.2 资产管理员（asset admin）

- `asset:reconciliation:*`（全部）
- `asset:card:*`（资产 CRUD）
- `ops:workstation:list`（可看工位基础信息）
- **不需要** `asset:reconciliation:list` 双重声明（路由级已包含）

### 4.3 工位管理员（workstation admin）

- `ops:workstation:*`（工位 CRUD）
- `ops:building:spaces:*`（空间可视化）
- `asset:reconciliation:list`（🆕 可看对账健康度）
- **不**包含 `asset:reconciliation:exception:manage`（仅看，不可改例外规则）

### 4.4 普通用户（regular user）

- 仅 `ops:workstation:list`（看工位）
- **不**包含 `asset:reconciliation:list` → 工位详情页健康度卡片**隐藏**

---

## 5. 权限 owner 与变更流程

| 资源 | 权限 owner | 变更流程 |
|------|----------|---------|
| `asset:reconciliation:*` | 资产管理 owner | 提交权限 owner 评审 |
| `ops:workstation:*` | 运维管理 owner | 运维 owner 评审 |
| `asset:reconciliation:list` 跨模块授予运维 | 双 owner 评审（资产 + 运维） | PR 必须双 owner approval |

---

## 6. 安全 invariants

| Invariant | 强制方式 |
|-----------|---------|
| 写操作（创建/更新/删除/转工单）必须严格权限 | `RequirePermissions`（不扩展） |
| 读操作可扩展权限（避免路径碎片化） | `RequirePermissionsWithQuery` |
| 无权限时不暴露内部错误 | Handler 内静默降级 |
| 权限边界变更需双 owner 评审 | PR review checklist |

---

## 7. 后续验证

R1 plan-phase 第一个 plan 必须包含：

- [ ] 单元测试：`hasReconciliationPerm` 在 4 种角色下的行为
- [ ] 集成测试：工位详情页 4 种角色的 UI 渲染
- [ ] E2E 测试：菜单 → 子页面 → 健康度卡片 → 抽屉详情
- [ ] 回归测试：现有工位路由不破坏
- [ ] 权限矩阵更新到 `.planning/permissions-matrix.md`（如不存在则创建）

---

## 8. 关联记忆

- `xingran-perm-namespace-split-readonly-page` — 读写权限命名空间割裂致 403，本设计显式声明跨模块边界避免
- `xingran-ops-menu-seed-perms-naming-mismatch` — 菜单 seed perms 与路由 perms 不一致，本设计统一 `asset:reconciliation:*`
- `migration-sql-name-must-match-model` — 菜单 migration 字段名匹配 model

---

## 9. 签字

| 角色 | 签字 | 日期 |
|------|------|------|
| 起草人 | gsd:explore session | 2026-06-27 |
| 资产 owner | （待评审） | — |
| 运维 owner | （待评审） | — |
| 权限 owner | （待评审） | — |

---

## 10. R4 实际接入清单 (2026-06-28)

Phase 45 R4 (Plan 01 + Plan 02) 在跨模块接入点上的**实际代码路径** — 供 Phase 46 (R5 半自动修复) + 未来审计快速检索。

### 10.1 后端集成点

| 接入点 | 文件 | 说明 |
|--------|------|------|
| `pkg/middleware/HasUserPermission(c, core, perm)` | `pkg/middleware/permission_query_helper.go` (Plan 01 新增) | 跨模块 perm 检查 helper，**不调 c.Abort()**(静默降级语义);接受 `*core.Core` 显式参数 |
| `ReconciliationService` 构造函数 | `internal/services/asset/reconciliation_service.go` `NewReconciliationService(db, cache, matcher)` | Plan 02 扩展为 3 参数,新增 matcher 例外规则服务 |
| `ReconciliationService.GetByWorkstation(wsID, window)` | `internal/services/asset/reconciliation_service.go` (Plan 01) | 工位对账健康度聚合(5 KPI + assets + visible);Plan 02 集成 IP 解析链 + per-asset MatchException |
| `ReconciliationService.InvalidateWorkstationHealth(wsID)` | `internal/services/asset/reconciliation_workorder.go` (Plan 02 新增) | 缓存主动失效 helper,nil-safe(单测场景跳过) |
| `ReconciliationWorkorderService.WorkstationIDForException(exceptionID)` | `internal/services/asset/reconciliation_workorder.go` (Plan 02 新增) | 旁路反查方法,**不动 CreateWorkorderFromException 签名**(B2 锁定) |
| `ReconciliationWorkorderService.InvalidateWorkstationHealth(wsID)` | `internal/services/asset/reconciliation_workorder.go` (Plan 02 新增) | scheduler 路径缓存失效(nil cache-safe) |
| `ReconciliationExceptionService.MatchException(ip, userID, conflictType)` | `internal/services/asset/reconciliation_exception.go` (Plan 02 新增) | per-asset 例外规则命中(返回 `*ExceptionMatch` 简化版) |
| `ReconciliationHandler.ResolveException` | `internal/api/v1/asset/reconciliation_handler.go` (Plan 02 改 success path) | 顺序:service → invalidate → operlog → response |
| `ReconciliationHandler.GetByWorkstation` | `internal/api/v1/asset/reconciliation_handler.go` (Plan 01) | handler 内 middleware.HasUserPermission 设置 visible 字段 |
| `WorkstationHandler.GetByID` | `internal/api/v1/operations/workstation_handler.go` (Plan 01) | 调 `hasReconciliationPerm` 门控;无权限时 `ReconciliationVisible=false` + `ReconciliationHiddenReason="无资产对账查看权限"` |
| `WorkstationHandler.WithReconciliationService` | `internal/api/v1/operations/workstation_handler.go` (Plan 01) | 链式 setter,跨模块注入 |
| `createWorkorderBySeverity` | `internal/scheduler/reconciliation_tasks.go` (Plan 02 改) | R2 转单完成后通过 `woSvc.WorkstationIDForException` + `woSvc.InvalidateWorkstationHealth` 失效缓存 |
| `RegisterReconciliationTasks(s, db, cache, wsHub, noticeSvc)` | `internal/scheduler/reconciliation_tasks.go` (Plan 02 改 5 参数) | 注入 cache 用于 R4 缓存失效 |
| `Core.RegisterReconciliationTasks` 调用 | `internal/core/core.go:332` (Plan 02 改) | 显式传 `c.Cache` 给 scheduler |

### 10.2 前端集成点

| 接入点 | 文件 | 说明 |
|--------|------|------|
| `useReconciliationVisibility` hook | `src/components/reconciliation/hooks/useReconciliationVisibility.ts` (Plan 01) | 读 `useMenuStore.permissions`(B4 修复),不读 `authStore.perms` |
| `useWorkstationHealth(workstationId)` hook | `src/components/reconciliation/hooks/useWorkstationHealth.ts` (Plan 01) | 5min staleTime + 10min gcTime;enabled gate: `visible && Boolean(workstationId)` |
| `useAssetHealth(assetId, workstationId)` hook | `src/components/reconciliation/hooks/useAssetHealth.ts` (Plan 01) | 从 `useWorkstationHealth` cache 切片(无 N+1) |
| `useExceptionMatch` hook | `src/components/reconciliation/hooks/useExceptionMatch.ts` (Plan 01) | R3 模式复用 |
| `<HealthCard>` 组件 | `src/components/reconciliation/HealthCard.tsx` (Plan 01) | `useReconciliationVisibility()===false` 时 `render null`;空态: "该工位暂无关联资产。" |
| `<HealthBadge>` 组件 | `src/components/reconciliation/HealthBadge.tsx` (Plan 01) | 8px 圆点 + Tooltip + useDict 颜色映射;不可见时 render "-" |
| `<ReconciliationDrawer>` 组件 | `src/components/reconciliation/ReconciliationDrawer.tsx` (Plan 01+02 改) | 780px + 3 Tabs;`onApplyException` 携带 assetIp + conflictType + workstationId query params(Plan 02) |
| `<ExceptionMatchList>` 组件 | `src/components/reconciliation/ExceptionMatchList.tsx` (Plan 01+02 改) | "去创建例外规则" 按钮携带 assetIp + conflictType query params(Plan 02) |
| `workstations/index.tsx` page lift | `src/pages/operations/workstations/index.tsx` (Plan 01) | page 顶层 lift `useWorkstationHealth` + `assetConflictMap` 传给 `<WorkstationDeviceTable>`(N+1 修复) |
| `WorkstationDeviceTable` conflictTypeMap | `src/components/operations/WorkstationDeviceTable/index.tsx` (Plan 01) | 接收 `conflictTypeMap` + `onBadgeClick` props |
| `assets/index.tsx` page lift | `src/pages/operations/assets/index.tsx` (Plan 01) | "对账健康" 列 + Drawer at page level |

### 10.3 operlog 覆盖范围

| 写路径 | Module 常量 | OperType | 备注 |
|--------|------------|----------|------|
| ResolveException | `ModuleReconciliation` ("资产对账") | `OperTypeUpdate` (2) | R2 已接,Plan 02 补 cache invalidate |
| R2 createWorkorderCritical/High | (cron 上下文) | (无) | **系统自动化行为,operlog 豁免**(comment 标注);workorder 后续 lifecycle handler 接手 |
| exception-rule CRUD | `ModuleReconciliationExceptionRule` ("资产对账-例外规则") | Create/Update/Delete | R3 接入,Plan 02 不变 |

### 10.4 关键设计决策(不变量)

1. **不动 service 签名**: `CreateWorkorderFromException`、`ResolveException`、`NewReconciliationService` 三个签名相关变更都在 B2 锁定下
2. **缓存失效入口**: `InvalidateWorkstationHealth` 在 3 处调用(ResolveException + R2 createWorkorder + 旁路)
3. **跨模块 perm**: `HasUserPermission` 显式接受 `*core.Core` 参数(handler 显式传 `h.core`);不调 `c.Abort()`
4. **前端 visible gate**: `useReconciliationVisibility` 单点读 `useMenuStore.permissions`;HealthCard/HealthBadge/Drawer 三处防御
5. **IP 解析链 inline**: 3 级降级(asset → workstation → network_device via port → "unknown");不抽新文件(B5 修复)
6. **N+1 修复**: Drawer state lift 到 page 顶层,WorkstationDeviceTable 接收 `conflictTypeMap` prop

### 10.5 R5 移交要点 (Phase 46)

- 半自动修复流程需复用 `ReconciliationService.GetByWorkstation` 响应(5 KPI + assets + visible)
- 修复操作走 `ModuleReconciliation` operlog path(同 ResolveException)
- 新增修复 action(待 R5 设计)需在 cross-module permission matrix 同步更新
- 当前 R4 实现已通过 `by-workstation` 端点暴露完整数据,R5 直接复用

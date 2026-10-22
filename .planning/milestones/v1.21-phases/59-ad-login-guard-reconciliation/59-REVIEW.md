---
phase: 59-ad-login-guard-reconciliation
reviewed: 2026-10-20T00:30:00Z
depth: standard
files_reviewed: 21
files_reviewed_list:
  - internal/api/v1/system/ad_domain_router.go
  - internal/api/v1/system/ad_reconciliation_handler.go
  - internal/api/v1/system/ad_reconciliation_handler_test.go
  - internal/core/db/database.go
  - internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go
  - internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go
  - internal/core/db/migrations/migration_218_ad_reconciliation_permission.go
  - internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go
  - internal/core/db/sqlite_reconciliation_views.go
  - internal/services/addomain/dept_ou_mismatch_service.go
  - internal/services/addomain/dept_ou_mismatch_service_test.go
  - internal/services/system/user_sync_service.go
  - internal/services/system/user_sync_service_phase59_test.go
  - ruoyi-react-frontend/src/lib/adDomainApi.test.ts
  - ruoyi-react-frontend/src/lib/adDomainApi.ts
  - ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css
  - ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx
  - ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx
  - ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/index.render.test.tsx
  - ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx
findings:
  critical: 0
  warning: 2
  info: 6
  total: 8
status: issues_found
---

# Phase 59: Code Review Report

**Reviewed:** 2026-10-20T00:30:00Z
**Depth:** standard
**Files Reviewed:** 21
**Status:** issues_found

## Summary

本 phase 交付四件套：GUARD-01 登录回写守卫（`SyncADUser` 的 dept_id 回写前查询活跃 `ad_push_queue` update_user 行）、migration_217 的 `v_dept_ou_mismatch` 普通 SQL VIEW（UNION ALL 四类口径）、migration_218 权限 seed + 只读端点 `POST /api/v1/ad-domain/reconciliation/mismatch`、前端只读 `ReconciliationTab`。

**逐项核验结论（均为独立复核，非采信实现自述）：**

- **权限字符串三方一致**：路由硬编码（`ad_domain_router.go:149`）= migration_218 单一事实源常量 `adReconciliationListPerms`（`migration_218_ad_reconciliation_permission.go:17`）= 前端 `RECONCILIATION_PERM`（`ReconciliationTab.tsx:37`），逐字比对一致。
- **守卫语义正确**：`findActiveUserUpdateQueue` 复用 `models.ADPushQueueActivePredicate`（`"status IN ('pending', 'failed')"`，与 migration_215 部分唯一索引同源）；`errors.Is(qerr, gorm.ErrRecordNotFound)` 三分支（失败保守跳过 / 命中拦截 / 无行照常回写）逻辑闭合；`ad_push_queue` 无软删列，谓词与部分唯一索引语义一致。六个守卫用例（pending/failed/终态/disable/无行/查询失败）锚定了 fixture 真实走进守卫分支，不是空跑。
- **migration_217 双方言红线全部守住**：无 ILIKE/REGEXP_REPLACE/`::` 强转；UNION ALL 各分支 ID/时间列双侧 CAST；布尔用 `NOT mp.sync_enabled`；DN 后缀用 `SUBSTR + LENGTH` 逗号边界（起算点 `LENGTH(dn)-LENGTH(base)` 恰指向逗号，手工推演正确）；空 base_dn 显式过滤。`sys_dept_ou_mapping.dept_id/ad_config_id` 经模型核验确为 `type:uuid`，与分支②③的裸 uuid 等值 JOIN 相容。
- **三处启动注册齐全**：PG advisory-lock 块（database.go:1080/1086）、sqlite 分支（:1130/1135）、SKIP_AUTOMIGRATE 旁路 BootstrapMissingTables（:1329/1335）均注册；视图前置 DROP 双方言就位（PG `dropDependentMaterializedViews` / sqlite `dropSQLiteReconciliationViews`），且 `ensureSQLiteReconciliationViews` 的 DROP 列表**未**误加对账视图（重建职责归 Migrate217，注释红线落实）。
- **D-12 Go↔SQL 口径一致性测试真实存在且双向断言**（`TestPhase59_MatchConsistency` 七组边界：最长后缀/逗号边界/空 base/停用/软删/等长歧义），不是摆设。
- **验证记录**：`go test` 四个包的 Phase 59 用例全绿（migrations / addomain / api/v1/system / services/system）；`npx vitest run` 三个前端测试文件 38/38 通过。

**总体评估：** 后端与迁移质量高，契约注释与实现一致性强，无 Critical。两个 Warning 都在前端 Tab 的请求纪律上：一是无权限时仍会发出部门树请求（违反组件头注的「零请求」契约，且测试 mock 掉了 `useDeptTree` 造成断言盲区），二是 `loadList` 无过期响应守卫（快速连点分页/回车可出现旧响应覆盖新响应）。

## Critical Issues

无。本 phase 未发现可证明的 BLOCKER 级缺陷。

## Warnings

### WR-01: 无权限时 ReconciliationTab 仍触发 getDeptTree 网络请求，违反「零请求」契约

**File:** `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx:80`（`const deptTree = useDeptTree();`）

**Issue:** 组件头注（第 7 行）与测试名都锁定了 D-17/T-59-09 契约：无 `ops:ad:config:reconciliation:list` 权限时「渲染 403 且**不发任何请求**」。但 `useDeptTree()` 在权限早退分支（:232 `if (!canView) return <Result/>`）**之前**无条件调用，且 `useDeptTree`（`src/hooks/useDeptTree.ts:27`）不接受 `enabled` 选项——react-query 挂载即发起 `getDeptTree()`。无权限用户点开「对账」Tab 仍会产生一次部门树请求。测试里 `useDeptTree` 被整体 mock（`ReconciliationTab.test.tsx:23-28`），该断言盲区使 403 用例只验证了 `getDeptOUMismatchList` 未调用。影响：契约违背 + 多余请求（部门树端点仅需登录态，无敏感数据泄露，故非 BLOCKER）。

**Fix:**

```ts
// useDeptTree.ts — 增加 enabled 选项
export function useDeptTree(options?: { enabled?: boolean }): UseQueryResult<DeptTreeNode[]> {
  return useQuery({
    queryKey: queryKeys.dept.tree(),
    queryFn: async () => { /* 原实现 */ },
    enabled: options?.enabled ?? true,
    staleTime: 5 * 60 * 1000,
    // ...
  });
}

// ReconciliationTab.tsx:80
const deptTree = useDeptTree({ enabled: canView });
```

### WR-02: loadList 无过期响应守卫，快速操作可出现旧响应覆盖新响应

**File:** `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx:96-119`

**Issue:** `loadList` 没有 AbortController / 请求序号守卫。「查询」按钮虽有 `loading` 门控，但：① 关键字输入框 `onPressEnter={handleQuery}`（:268）不受 loading 门控，加载中回车即可并发第二个请求；② antd Table 的分页器在 `loading` 期间仍可点击，连点第 2、3 页会并发两个请求。两个响应完成顺序不保证，后完成的**旧**响应会用旧筛选/旧页码的数据覆盖状态，出现「分页器停在第 3 页但表格显示第 2 页数据」。本仓库 duty 模块 pools 曾修过同类竞态（memory: duty-module-user-field-gotchas）。

**Fix:**

```tsx
const reqSeq = useRef(0);
const loadList = useCallback(async () => {
  if (!canView) return;
  const seq = ++reqSeq.current;
  setLoading(true);
  setError(null);
  try {
    const res = await getDeptOUMismatchList(params);
    if (seq !== reqSeq.current) return; // 过期响应丢弃
    setList(res.data?.list ?? []);
    setTotal(res.data?.total ?? 0);
  } catch (e) {
    if (seq !== reqSeq.current) return;
    setError(e instanceof Error ? e : new Error(String(e)));
  } finally {
    if (seq === reqSeq.current) setLoading(false);
  }
}, [/* 原依赖 */]);
```

## Info

### IN-01: BatchSyncADUsers 的 dept_id 回写无 GUARD-01 守卫（与登录路径不对称）

**File:** `internal/services/system/user_sync_service.go:647-651`（阶段5b catActiveAD）、`:662-666`（阶段5c catSoftDeleted）

**Issue:** 守卫只包住 `SyncADUser` 单用户路径的 dept_id 回写（D-04 按 plan 是「登录回写守卫」，范围正确）；但批量同步在阶段5b/5c 直接 `tx.Model(it.user).Update("dept_id", deptID)`，不查活跃队列行。管理员本地编辑入队未排水的 30s 窗口内若有人跑批量同步，本地 dept 被覆盖而队列快照仍持旧值，推送后两端再次分叉（会以 user_ou_mismatch 形式回到本 phase 的对账视图里）。管理员显式触发同步、覆盖属预期语义的概率高，建议在 plan/SUMMARY 登记「批量路径刻意无守卫」的决策记录，防后续被当 bug 重查。

**Fix:** 无需改码；在 59-PATTERNS 或模块注释补一句「GUARD-01 范围 = 登录路径，批量同步为显式管理动作不设守卫」。

### IN-02: 视图分支①存在「跨配置映射」盲区（紧邻已登记的 Known Gap）

**File:** `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go:198-207`

**Issue:** 分支① JOIN 条件是 `mp.dept_id = u.dept_id AND mp.ad_config_id = routed.config_id`。当用户路由到配置 A、而其部门映射行挂在配置 B 名下时：①无 JOIN 行（不报 user_ou_mismatch）、④因 routed 存在不报 config_unrouted——这条漂移对四类口径全部不可见。这与注释里已声明的第五类 Known Gap（「路由配置下无映射」）同族但形态不同（映射存在、只是挂错配置）。多配置场景下属真实误配置形态，建议登记进 Known Gap 清单或作为第五类候选。

**Fix:** 文档层面补充；若后续扩口径，在 routed CTE 之外加「映射 ad_config_id ∉ 用户可路由配置集」的分支。

### IN-03: handler 重复实现四值白名单循环，未复用 service 侧判定

**File:** `internal/api/v1/system/ad_reconciliation_handler.go:47-59`

**Issue:** handler 手写 `for _, mt := range addomainServices.MismatchTypes` 白名单循环；`dept_ou_mismatch_service.go:42-49` 已有语义完全相同的 `validMismatchType`（未导出）。两处判定将来对「空串/大小写归一」策略调整时容易只改一处。`MismatchTypes` 已导出，判定逻辑应单点化。

**Fix:** 在 addomain 包导出 `ValidMismatchType(t string) bool`（内部直接复用现有私有函数），handler 改调用。

### IN-04: dropDependentMaterializedViews 的「noop」收尾日志已与行为矛盾

**File:** `internal/core/db/database.go:478-497`

**Issue:** 本 phase 在该函数开头新增了 `DROP VIEW IF EXISTS v_dept_ou_mismatch`（:478），但函数尾部的 INFO 日志仍输出「当前架构下 noop(reconciliation_normalized MV 保留,普通 VIEW 由 175 CREATE OR REPLACE 重建)」（:496-497）。每次 PG 启动都会打出与实际行为相反的日志，排障时会误导运维。

**Fix:** 改写日志为「已定向 DROP v_dept_ou_mismatch（migration_217 在 AutoMigrate 后重建）；其余视图 noop」。

### IN-05: migration_218 写入 VisibleHidden 但实际落库 visible=1（GORM default tag 零值省略）

**File:** `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go:66-74`

**Issue:** 代码设 `Visible: models.VisibleHidden`（=0），但 `models.Menu.Visible` 带 `default:1` tag，GORM 对零值字段省略列 → DB 默认值 1（显示）落库。注释已声明该行为且 F 型按钮可见性不参与渲染判定、与 migration_163 同款，无功能影响；但「写 A 存 B」是复制粘贴陷阱——后人把同款写法搬到 C 型菜单时「隐藏」意图会静默变「显示」。

**Fix:** 无需改码；把注释升级为红线提示（「凡带 default tag 的枚举列，零值意图必须经显式 Select/Map 写入」），或直接省略 Visible 字段赋值以消除误导。

### IN-06: phase 触碰文件遗留 gofmt 不洁与缩进劣化

**File:** `internal/services/system/user_sync_service.go:846-847`、`ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx:346-374`

**Issue:** ① `user_sync_service.go` 本 phase 在文件尾新增的 `deptValue` 之后多出一个空行，且该文件整体不满足 `gofmt -l`（:207、:243-247 的对齐为历史遗留，非本 phase 引入）；② `index.tsx` 把既有 JSX 块包进新 Fragment 时未重排缩进，`<div style={{marginBottom:16}}>` 到 `</Table>` 段缩进层级与新外层 Tabs 结构脱节（纯外观，无功能影响）。

**Fix:** 对 user_sync_service.go 跑 `gofmt -w`；index.tsx 用编辑器格式化该段。

---

_Reviewed: 2026-10-20T00:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

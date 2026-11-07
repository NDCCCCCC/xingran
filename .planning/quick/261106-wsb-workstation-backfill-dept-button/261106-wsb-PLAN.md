---
quick_id: 261106-wsb
slug: workstation-backfill-dept-button
status: planned
date: 2026-11-06
commit: none
---

# 工位管理页新增「按人员反推部门」按钮（存量数据批量回填）

> 本轮为**审计 + 详细设计**交付物，用户明确要求「先出详细 PLAN 文档」，**不改动任何代码**。
> 实施需用户后续单独确认。

## 1. 用户决策（全部已锁定，不再重问）

| 项 | 决策 |
|---|---|
| 交互模式 | **预览 + 确认两步**（`dryRun:true` 预览 → 确认 → `dryRun:false` 执行） |
| 作用范围 | **全库所有工位** |
| 覆盖策略 | **默认仅填补空缺 + 可选「纠正不一致」开关，默认关闭** |
| 「纠正不一致」风险处置 | **保留但默认关闭**：预览逐条列出「现部门 → 目标部门」并高亮，二次确认明确「将覆盖 N 条已填写部门」 |
| 权限 | **复用 `ops:workstation:edit`**，不新增权限码 |
| 提交 | 不自动提交（项目 CLAUDE.md：提交前必须询问） |

历史决策（本会话前两轮，仍生效）：
- quick-261106-wsd：导入兜底采用「仅兜底」，不覆盖显式填对的部门。
- quick-261106-wso：楼层 `org_id` 可覆盖楼宇归属；一栋楼跨机构办公。

## 2. 审计结论（关键认知）

### 2.1 现有 `inferDeptFromUser` 不能直接复用为按钮能力
[internal/services/operations/excel_service.go:735](internal/services/operations/excel_service.go#L735) 的 `inferDeptFromUser` 是**纯内存函数**：输入 `[]map[string]any`（导入中、未落库），只改 map 里的 `dept_id` 键，**从不执行 SQL UPDATE**；写库靠后续 `BatchUpsert` 顺带完成。

按钮要处理的是**数据库中已存在的存量工位**，必须新增持久化写能力：新 service 方法 + 新 SQL + 新端点 + 新按钮。

### 2.2 可 1:1 复用的判定规则（已生产验证）
- 仅填补空缺（`dept_id IS NULL OR dept_id = ''`）
- `sys_user.dept_id` → `sys_dept.id` INNER JOIN 过滤 NULL / 部门名称字符串 / 悬空 ID
- **JOIN 必须两侧 `CAST(... AS TEXT)`**：`sys_dept.id` 是 uuid、`sys_user.dept_id` 是 varchar(64)，PG 无 `uuid = varchar` 运算符（范式见 [migration_217:201-206](internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go#L201)）
- `sys_dept` **无 `deleted_at` 列**（[reference_resolver.go:43](internal/services/operations/reference_resolver.go#L43)），不能加软删过滤
- `sys_user.deleted_at IS NULL` 必须保留

### 2.3 审计出的 7 个必处理风险点

| # | 风险 | 依据 | 处置 |
|---|---|---|---|
| 1 | 权限闭环 | 写路由全挂 `strictUserRouteGroup`（[router.go:819-832](internal/api/router.go#L819)） | 新端点挂 `workstationEdits` 严格组（`OpsWorkstationEdit`） |
| 2 | 路由登记守卫 | [route_registry_test.go](pkg/permission/route_registry_test.go#L1) Freshness + Coverage 双层；[strict_mount_count_test.go:36-40](internal/api/strict_mount_count_test.go#L36) 下限 16/188/164 | 登记 `config.go` + `route_registry_test.go`，并 **`node scripts/scan-routes.cjs` 重生成 `scripts/route-registry.json`** |
| 3 | `status` 联动 | `applyWorkstationOccupancyLink`（[workstation_service.go:266](internal/services/operations/workstation_service.go#L266)）只由 `user_id` 驱动；导入时 `prepareRecordsForUpsert` 会改 status（[excel_service.go:1093](internal/services/operations/excel_service.go#L1093)） | **本次只写 `dept_id` + `updated_at`，绝不碰 `status`** —— 存量状态是真实业务状态 |
| 4 | 缓存失效 | 工位 CRUD service **完全不做缓存失效**；只有 `ImportData` 清 `workstation:*`（[excel_service.go:526](internal/services/operations/excel_service.go#L526)） | 执行成功后 `InvalidateByPatterns(ctx, []string{"workstation:*","workstations:*"}, "workstation")`，需给 service 注入 `system.CacheProvider` |
| 5 | 数据质量 | `sys_workstation.dept_id` 是 `*string`（[workstation.go:46](internal/models/workstation.go#L46)），空串也算空 | WHERE 同时覆盖 NULL 与 `''`；INNER JOIN 过滤用户侧脏值 |
| 6 | 范围/幂等/可逆 | 全表 UPDATE 不可撤销 | 事务包裹 + dry-run 预览 + 天然幂等（第二次 0 条） |
| 7 | 审计日志 | CLAUDE.md 强制：业务写操作 handler 在 `response.Success` 前 `operlog.Record` | `OperTypeBatch(16)` + `WithOperParam` 记录条数/模式（先例 [excel_handler.go:140](internal/api/v1/operations/excel_handler.go#L140)） |

### 2.4 两个易漏点
- `WorkstationService` 是**接口**（[workstation_service.go:46](internal/services/operations/workstation_service.go#L46)）：新增方法需同步 3 个 mock —— `workstation_handler_test.go:64`、`workstation_handler_full_test.go:33/87`、`mockWorkstationServiceForStatisticsError`（[workstation_handler_test.go:252](internal/api/v1/operations/workstation_handler_test.go#L252)），否则 handler 测试编译失败。
- 扫描器 `WRITE_HANDLER` 正则（[scan-routes.cjs:33](scripts/scan-routes.cjs#L33)）含 `Preview`/`Apply`/`Batch`/`Sync` 等词；handler 用 `BackfillDeptFromUser`，按规范本来就要登记权限，不取巧绕过。
- `Modal.confirm` 是独立 React root，测试需手动清 `.ant-modal-confirm` 残留（[dedicated-lines/__tests__/index.pageTable.test.tsx:59](xingran-react-frontend/src/pages/operations/dedicated-lines/__tests__/index.pageTable.test.tsx#L59)）。

### 2.5 决策 3 的语义冲突与处置
「纠正不一致」会改写已有部门，与「仅兜底」冲突，且**工位部门 ≠ 人员部门常是有意配置**：
- 上一轮刚做的「浙商大厦 1/3 楼属武汉中支」：工位挂在楼层/机构下，人员编制可属另一部门；
- Phase 39 `sys_dept_location_alias` 存在的全部意义就是「组织编制与物理办公地分离」（[location_alias_service.go:245-251](internal/services/operations/location_alias_service.go#L245)）。

处置：两个互斥模式、`fixMismatch` **默认 false**；模式 B 只纠正「用户部门有效可解析」的行，用户部门 NULL/脏/指向不存在部门一律跳过；预览逐条列出「现部门 → 目标部门」；确认文案明确「将覆盖 N 条已填写部门」；UI 做成 Checkbox 而非按钮，文案警示。

## 3. 接口契约

单端点 + 两个 bool 参数（不新增第二个 URL，缩小 route-registry 面）：

```
POST /ops/workstation/backfill-dept
body: { "dryRun": bool, "fixMismatch": bool }
```

响应 `BackfillDeptResult`（`response.Success` 包装）：

```jsonc
{
  "dryRun": true,
  "fixMismatch": false,
  "candidates": 0,           // 命中「有用户且用户部门有效」的工位总数
  "toFill": 0,               // dept_id 空 → 将填补 / 实际填补
  "toCorrect": 0,            // dept_id 非空且不一致 → 将纠正 / 实际纠正（仅 fixMismatch=true）
  "skippedNoUser": 0,        // 无绑定用户
  "skippedBadUserDept": 0,   // 用户部门 NULL / 脏 / 指向不存在部门
  "skippedAlreadyValid": 0,  // 已有部门且一致
  "samples": [               // 前 20 条，供预览确认
    {
      "workstationId": "", "workstationName": "",
      "currentDeptId": "", "currentDeptName": "",
      "targetDeptId": "", "targetDeptName": "",
      "userName": "", "action": "fill|correct|skip"
    }
  ]
}
```

`dryRun=true` 不写库、不清缓存、不记 operlog（或记 `OperTypeOther` 读日志，实施时定）；`dryRun=false` 返回**实际更新条数**。幂等：第二次 `toFill=0`。

## 4. 后端逐文件实施清单

### 4.1 `internal/services/operations/workstation_service.go`
- 接口新增：
  ```go
  BackfillDeptFromUser(ctx context.Context, dryRun, fixMismatch bool) (*BackfillDeptResult, error)
  ```
- 实现采用 **「一次 SELECT 候选 → 内存分类 → 批量 CASE UPDATE」**，参考 [BatchUpdatePositions:391](internal/services/operations/workstation_service.go#L391) 的 PG/SQLite 双方言做法。**不用 PG `UPDATE ... FROM`**（SQLite 不支持）。
- 候选 SELECT（表名/列名均为常量，无用户输入拼接）：
  ```sql
  SELECT w.id AS workstation_id, w.workstation_name, w.dept_id AS current_dept_id,
         d2.dept_name AS current_dept_name,
         su.dept_id AS target_dept_id, d.dept_name AS target_dept_name,
         su.nickname AS user_name
    FROM sys_workstation w
    JOIN sys_user su ON CAST(su.id AS TEXT) = CAST(w.user_id AS TEXT)
    JOIN sys_dept d  ON CAST(d.id AS TEXT) = CAST(su.dept_id AS TEXT)
    LEFT JOIN sys_dept d2 ON CAST(d2.id AS TEXT) = CAST(w.dept_id AS TEXT)
   WHERE w.deleted_at IS NULL
     AND su.deleted_at IS NULL
     AND su.dept_id IS NOT NULL AND su.dept_id <> ''
  ```
  > 说明：`JOIN sys_dept d` 是 INNER，天然过滤用户侧脏 dept_id；`LEFT JOIN d2` 只用于展示现部门名，不参与过滤。
- 内存分类：
  - `current_dept_id` 空（NULL/''）→ `toFill`（两种模式都做）
  - `current_dept_id` 非空且 `CAST(current AS TEXT) != CAST(target AS TEXT)` → `fixMismatch ? toCorrect : skippedAlreadyValid`
  - 相等 → `skippedAlreadyValid`
  - 未命中候选的工位按 `skippedNoUser` / `skippedBadUserDept` 分别 COUNT（可用两条独立 COUNT 子查询，避免把全表拉进内存）
- `dryRun=false` 且 `toFill+toCorrect>0`：事务内批量 UPDATE，**只写 `dept_id` + `updated_at`**，用 `CASE WHEN id THEN CAST(? AS TEXT)` 形态（沿用 [workstation_position_sql_test.go](internal/services/operations/workstation_position_sql_test.go) 的 42804 类型安全教训，参数显式 CAST）。按 400 分块。
- 事务提交**之后**清缓存（cache 写不进事务，与 [excel_service.go:511-517](internal/services/operations/excel_service.go#L511) 的 post-commit 语义一致）。
- 注入 `system.CacheProvider`：优先给 `workstationService` struct 加字段 + `NewWorkstationService` 增加可选参数；若构造函数调用点过多，则在 handler 层用 `core.DataCacheService` 包一层 `system.NewCacheProvider` 传入。**实施时先 grep 调用点数量再定**。

### 4.2 `internal/api/v1/operations/workstation_handler.go`
- 新增 `BackfillDeptFromUser(c *gin.Context)`：`ShouldBindJSON` → 调 service → `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeBatch, operlog.WithOperParam(...))` → `response.Success(c, result)`。
- `WithOperParam` 记录 `dryRun/fixMismatch/toFill/toCorrect/candidates`，**不记录工位明细**（防信息泄露，对齐 [excel_handler.go:138-142](internal/api/v1/operations/excel_handler.go#L138)）。

### 4.3 `internal/api/router.go:823`
```go
workstationEdits.POST("/backfill-dept", workstationHandler.BackfillDeptFromUser)
```
挂在既有 `strictUserRouteGroup(workstationWrites, permission.OpsWorkstationEdit, core)` 组内。

### 4.4 权限登记（4 处，缺一测试红）
1. [pkg/permission/config.go:585](pkg/permission/config.go#L585) 附近加：
   `{"/ops/workstation/backfill-dept", "POST", OpsWorkstationEdit, "按人员反推部门"},`
2. [pkg/permission/route_registry_test.go:173](pkg/permission/route_registry_test.go#L173) 附近加映射条目。
3. 运行 `node scripts/scan-routes.cjs` 重生成 `scripts/route-registry.json`。
4. 确认 [strict_mount_count_test.go:36-40](internal/api/strict_mount_count_test.go#L36) 下限（16 / 188 / 164）不降；新增 1 条 Strict 写路由只会让 total/write 上升，不会触发下限失败。

### 4.5 测试（后端）
新增 `internal/services/operations/workstation_backfill_dept_test.go`：
| 用例 | 断言 |
|---|---|
| dept_id NULL / 空串 + 有效用户部门 | `toFill` 计数正确，落库 dept_id 正确 |
| dept_id 已填且一致 | `skippedAlreadyValid`，不更新 |
| dept_id 已填但不一致、`fixMismatch=false` | 不覆盖（默认仅兜底） |
| dept_id 已填但不一致、`fixMismatch=true` | `toCorrect`，落库改为用户部门 |
| 用户 dept_id NULL / 部门名称字符串 / 悬空 ID | `skippedBadUserDept`，不更新 |
| 工位无 user_id | `skippedNoUser` |
| `dryRun=true` | 返回预览计数，**数据库零变更** |
| 幂等 | 连跑两次，第二次 `toFill=0` |
| 状态不被改动 | 维护(2)/占用(1)/空闲(0) 工位 status 保持原值 |
| 事务回滚 | UPDATE 失败时零变更（可用重复键/伪造错误） |
| samples 上限 | ≤20 条 |
| 缓存失效被调用 | 有更新时触发（mock CacheProvider） |

新增/更新 handler 测试：`workstation_handler_full_test.go` 加路由挂载 + 成功/bind 错误/service 错误三态；**3 个 mock 补接口方法**。

## 5. 前端逐文件实施清单

### 5.1 `xingran-react-frontend/src/lib/opsApi.ts`
```ts
backfillDeptFromUser: async (params: { dryRun: boolean; fixMismatch: boolean }) => {
  const res = await post<BackfillDeptResult>("/ops/workstation/backfill-dept", params);
  return res.data;
}
```
`BackfillDeptResult` 类型定义在同文件或 `src/types/operations.ts`（实施时定，倾向后者与其他 DTO 一致）。

### 5.2 `xingran-react-frontend/src/pages/operations/workstations/index.tsx`
- 按钮置于 [L885-945 按钮区](xingran-react-frontend/src/pages/operations/workstations/index.tsx#L885)，与「映射」按钮并列，文案「按人员反推部门」，图标建议 `DeploymentUnitOutlined` 或 `SwapOutlined`。
- 门控：`permEdit = permGate.decide(PERMISSION_CODES.opsWorkstation.edit)`（已有变量），三态 `visible/disabled/hint`。**复用 edit，不新增码**。
- 交互流程：
  1. 点击 → `backfillDeptFromUser({dryRun:true, fixMismatch:false})`
  2. `Modal.confirm` 预览：展示 `candidates/toFill/skippedNoUser/skippedBadUserDept/skippedAlreadyValid`，`samples` 前若干条表格（工位名 / 现部门 / 目标部门 / 人员 / 动作），`fixMismatch` **Checkbox 默认不勾选**，文案警示「勾选后将覆盖 N 条已填写部门，仅确认填错时使用」。
  3. 勾选 `fixMismatch` 时**重新 dry-run** 以刷新 `toCorrect` 与样例（避免预览与执行口径不一致）。
  4. 确认 → `backfillDeptFromUser({dryRun:false, fixMismatch})` → `message.success(\`已填补 X 条、纠正 Y 条\`)` → `refreshData()`（[index.tsx:289](xingran-react-frontend/src/pages/operations/workstations/index.tsx#L289)）刷新列表与统计。
  5. 异常走 `handleApiError`；`dryRun=true` 不计入 operlog。
- 注意 `Modal.confirm` 独立 root，测试需手动清 `.ant-modal-confirm` 残留。

### 5.3 前端测试
- `permission.gate.test.tsx`：补「无 edit 权限 → 按钮不渲染 / 未就绪 → disabled」断言（现有 5 个用例的扩展）。
- 新增按钮交互测试：mock `workstationApi.backfillDeptFromUser`，断言 dry-run → confirm → execute 调用顺序与参数、`fixMismatch` 勾选重跑预览、成功后 `refreshData`。
- `npm run type-check`。

## 6. 完整验证清单

```
go build ./...
go vet ./internal/services/operations/ ./internal/api/v1/operations/ ./pkg/permission/
go test ./...                              # 含 pkg/permission 路由守卫 + internal/api strict 守卫
node scripts/scan-routes.cjs --out <tmp>  # 确认产物与入库 json 一致（或由 go test 代跑）
npm run type-check                         # xingran-react-frontend
npx vitest run <工位页相关测试>
```

## 7. 不做 / 边界

- **不改动** `inferDeptFromUser`（导入兜底保持原样，两套逻辑共用判定规则但各自独立）。
- **不碰** `status` 联动、`sys_dept_location_alias` 表、`batch_upserter.go`、楼层 org_id 逻辑。
- **不新增权限码**、不改菜单/角色种子。
- **不做** 后台异步任务/进度条（全库工位量级可控，一次同步事务；若实测 >10s 再评估分页）。
- **不做** 撤销/回滚按钮（靠 dry-run + 幂等 + 导出备份兜底）。
- 不碰无关的 `internal/services/system/asset_columns_schema.json`（用户既有未提交改动）。
- **本轮不写代码**；本 PLAN 待用户确认后再由 `/gsd-quick` 执行。

## 8. 预估改动面

约 **13 个文件**：后端 7（`workstation_service.go`、`workstation_handler.go`、`router.go`、`config.go`、`route_registry_test.go`、`scripts/route-registry.json`、新测试 + 3 个 mock 补方法）、前端 4（`opsApi.ts`、`types/operations.ts` 可选、`workstations/index.tsx`、新测试）、规划产物 2。

> 备注：本会话前两轮（261106-wsd / 261106-wso）的改动**均未提交**，同在 `fix/workstation-import-skip-reference` 分支；实施本 PLAN 前需一并 review。

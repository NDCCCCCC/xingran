---
quick_id: 261106-wsb
status: complete
date: 2026-11-06
commit: none (按项目约定，提交前须经用户确认)
---

# 完成：工位管理页「按人员反推部门」按钮（存量工位批量回填）

## 用户决策（已锁定）

| 项 | 决策 |
|---|---|
| 交互 | 预览 + 确认两步（`dryRun:true` → 确认 → `dryRun:false`） |
| 范围 | 全库所有工位 |
| 覆盖 | 默认仅填补空缺 + 可选「纠正不一致」开关，**默认关闭** |
| 「纠正不一致」风险 | 保留但默认关闭：预览逐条列「现部门→目标部门」、二次确认明确「将覆盖 N 条已填写部门」 |
| 权限 | 复用 `ops:workstation:edit` |

## 关键认知

现有 `inferDeptFromUser`（导入兜底）是**纯内存函数**，作用于导入中未落库的 `[]map[string]any`，本功能必须新增持久化写能力。判定规则 1:1 复用（INNER JOIN sys_dept 过滤脏值、两侧 CAST、sys_dept 无 deleted_at）。

## 实现

### 后端
- `internal/services/operations/workstation_service.go`
  - 接口 + DTO：`BackfillDeptFromUser(ctx, dryRun, fixMismatch)`、`BackfillDeptResult`、`BackfillDeptSample`
  - 实现：一次 SELECT 候选 → 内存分类 → 事务内批量 `CASE` UPDATE（400 分块）→ 事务提交后失效缓存
  - **双方言**：不用 PG 的 `UPDATE ... FROM`（SQLite 不支持）；`CASE` 两侧 `CAST(... AS TEXT)` 规避 PG `uuid = varchar`；`CURRENT_TIMESTAMP` 替代 `NOW()`（P-77-2 已知坑）；THEN 分支显式 CAST 规避 PG 42804
  - **只写 `dept_id` + `updated_at`，绝不碰 `status`**（存量状态是真实业务状态）
  - UUID 比较 `TrimSpace + EqualFold`，避免 varchar 列大小写差异在 fixMismatch 下误判
  - 新增 `NewWorkstationServiceWithCache`（保留原 14 个 `NewWorkstationService(db)` 调用点不变）+ `WithCacheProvider` setter
  - 缓存失效：`InvalidatePattern(ctx, ["workstation:*","workstations:*"])`，post-commit
- `internal/api/v1/operations/workstation_handler.go`：`BatchBackfillDeptFromUser`（名称含 `Batch` 让扫描器识别为写路由），`dryRun=false` 时 `operlog.Record(... OperTypeBatch, WithOperParam(条数/模式))`，不记明细
- `internal/api/router.go`：挂到 `workstationEdits` 严格写组；生产路径改用带缓存构造
- 权限登记 4 处：`pkg/permission/config.go`、`pkg/permission/route_registry_test.go`、`scripts/route-registry.json`（`node scripts/scan-routes.cjs` 重生成）、确认 `strict_mount_count_test.go` 下限不降
- 3 个 mock 补接口方法 + handler 测试路由表

### 前端
- `src/types/operations.ts`：`BackfillDeptResult` / `BackfillDeptSample`
- `src/lib/opsApi.ts`：`workstationApi.backfillDeptFromUser({dryRun, fixMismatch})`
- `src/pages/operations/workstations/hooks/useBackfillDept.tsx`（新）：预览 → `Modal.confirm` 展示候选/分布/前 20 条样例 → 「纠正不一致」Checkbox（默认关，勾选重跑预览）→ 确认执行 → 成功提示 → `onSuccess()` 刷新
  - 用命令式 `Modal.confirm` + 模块级 `getAppMessage()`，按 eslint 契约显式豁免（非删除类确认，同 FormModal.tsx）
- `src/pages/operations/workstations/index.tsx`：按钮，三态门控 `permEdit`，与「映射」并列

## 验证

| 命令 | 结果 |
|---|---|
| `go build ./...` + `go vet`（4 包） | PASS |
| `go test ./... -count=1`（`SKIP_E2E=1`） | **全绿** |
| 新增 `workstation_backfill_dept_test.go` | 8/8 PASS |
| 新增 handler 测试（成功/bind 错误/service 错误） | PASS |
| `pkg/permission` + `internal/api` 路由双层守卫 | PASS |
| `npm run type-check` | PASS |
| `npm run lint`（相关 6 文件，max-warnings=0） | PASS |
| `useBackfillDept` 单测 | 4/4 PASS |
| `permission.gate.test.tsx` | 5/5 PASS |

> `go test ./...` 不带 `SKIP_E2E=1` 时唯一失败是 `scripts/e2e/TestE2EAllEndpointsLogged`：Windows 下 `filepath.Join` 生成的反斜杠路径被 bash 吃掉（`....scriptsoperlog_e2e_verify.sh: No such file`），为**基线环境性失败**，与本次改动无关。

service 测试覆盖：dryRun 零写库、仅填补不覆盖、fixMismatch 开/关、脏用户部门跳过、无用户跳过、幂等、status 三种状态不被改动、软删排除、UUID 大小写不误判。

## 未改动 / 边界

- 不改动 `inferDeptFromUser`（导入兜底保持原样，两套逻辑判定规则一致但各自独立）
- 不碰 `status` 联动、`sys_dept_location_alias`、`batch_upserter.go`、楼层 org_id 逻辑
- 不新增权限码、不改菜单/角色种子
- `dryRun=true` 不写库、不清缓存、不记 operlog
- 不做后台异步/进度条（全库量级可控，一次同步事务）
- 不碰无关的 `internal/services/system/asset_columns_schema.json`（用户既有未提交改动）
- **未自动 commit**

## 已知边界

- `SkippedBadUserDept` 包含「绑定了已软删/不存在用户」的工位（语义上归入「用户部门不可用」，可接受）
- 楼层/工位机构筛选的 orgId 仍是精确等值 vs 递归口径不一致（前一轮遗留，未在本次扩大回归面）
- 「纠正不一致」会覆盖有意配置的跨机构部门（如浙商大厦 1/3 楼属武汉中支、运营服务部独立编制），故默认关闭 + 逐条预览 + 警示文案；建议仅在对数据有把握时勾选

## 生产回归修复（2026-11-07，部署后实测 500）

**现象**：`POST /ops/workstation/backfill-dept` 预览即 500「按人员反推部门失败」。

**根因**：`sys_user.dept_id` 在生产 PG 是**原生 uuid 列**（[user_service.go:463](internal/services/system/user_service.go#L463) 的 P4 fix 早有记载，且明确警告过 uuid 列上 `!= ''` 的类型推断错误），候选查询里裸写了 `su.dept_id <> ''` → PG 尝试把 `''` 隐式转 uuid 报 invalid input syntax。SQLite 动态类型全绿，与 pg_contract_test.go 头部记载的 42883 同族——实施时漏接了这条既有防线。

**修复**（全部双方言安全，SQLite 下 CAST 为 no-op）：
1. 候选 SELECT：`su.dept_id <> ''` → `CAST(su.dept_id AS TEXT) <> ''`
2. COUNT withUser：`user_id <> ''` → `CAST(user_id AS TEXT) <> ''`
3. 防御性加固 261106-wso 的 `COALESCE(NULLIF(f.org_id,''), b.org_id)` → 双侧 `CAST(... AS TEXT)`（org_id 若与 uuid 漂移，PG 会报 COALESCE types cannot be matched）：workstationOrgColumn + floor_service 两处
4. 补 `TestPGContract_WorkstationOrgAndBackfill`（pg_contract_test.go，`TEST_PG_DSN` 门控只读）：dryRun 预览 + List/Statistics/SearchOptions/楼层 List/下拉的 orgId 筛选全部真实打 PG

**部署前建议执行**（只读，需真实 PG DSN）：

```
TEST_PG_DSN="postgres://<user>:<pass>@10.62.10.39:5432/<db>?sslmode=disable" go test -v -run TestPGContract ./internal/services/operations/
```

> 本会话三轮改动（261106-wsd 导入反推 / 261106-wso 楼层 org_id / 261106-wsb 本按钮）**均未提交**，同在 `fix/workstation-import-skip-reference` 分支。

## 审计修复（2026-11-07，subagent 全面审计后）

subagent 审计结论 FAIL（P0x1 / P1x6 / P2x11），已修复：

- P0-1：导入兜底 inferDeptFromUser 的 su.dept_id 空串比较同款 uuid 缺陷（错误被 Warnf 吞掉导致兜底在生产静默失效）→ CAST 修复 + PG contract 补同款 SQL 直跑守护
- P1-1/P1-2：楼层 Create 软删恢复分支 / Update 换楼乐观锁分支丢 org_id → 补列与参数
- P1-3：楼层机构筛选精确等值 vs 楼宇/工位子树口径分裂（Cascader 一级可见二级为空）→ 两处改 Pluck 子树展开 + IN
- P1-4：前端新建楼层 orgId 残留 → DEFAULT_FORM_VALUES 显式置 undefined
- P1-5：信息点机构筛选漏接楼层覆盖口径 → 改同款 COALESCE
- P1-6：fixMismatch 勾选后预览刷新失败不回滚（确认框展示旧爆炸半径）→ 失败回滚开关
- P2-1/P2-3/P2-9：楼层 orgId UUID+存在性校验；PG contract 补 infer 同款 SQL；onOk 失败后 loading 挂起
- 附带行为改进：恢复 UPDATE 的 NOW() → CURRENT_TIMESTAMP，sqlite 恢复分支从必失败（P-77-2）变为可用，两个锁定旧缺陷的测试改正向断言

未修（记录待办）：P2 性能类（回填候选无 LIMIT / PK CAST 索引失效）、双实现合并、缓存 no-op 链路、样例表分拣、EditModal 子树 raw UUID —— 均为一致性/效率项，不影响正确性。

验证：go build 全绿；operations/api/permission/api/migrations 全绿；真实 PG（10.62.10.34）contract 全过；前端 type-check/lint/vitest 13/13。

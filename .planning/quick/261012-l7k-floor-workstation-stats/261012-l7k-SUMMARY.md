---
phase: quick
plan: 261012-l7k
status: complete
subsystem: operations
tags: [floor, workstation-stats, group-by-aggregation, frontend-column]
---

# Quick 261012-l7k 楼层管理页每行工位统计 Summary

## One-liner

楼层 List 响应内联每行工位统计（单条 `GROUP BY floor_id` 聚合，连同分页主查询共 2 条 SQL 零 N+1），前端楼层表格新增"工位统计"列（总/已用/空闲/维护 Tag，维护仅 >0 显示），状态语义严格按 `sys_workstation` 独立枚举 0=空闲/1=占用/2=维护。

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | 后端 OpsFloor.WorkstationStats 虚拟字段 + List 批量聚合注入 + 4 个 service 测试 | `0d3e9670` | `internal/models/operations/floor.go`, `internal/services/operations/floor_service.go`, `internal/services/operations/floor_workstation_stats_test.go`(新) |
| 2 | 前端 Floor 类型扩展 + "工位统计"列 + 列 render 测试 | `03624395` | `ruoyi-react-frontend/src/types/operations.ts`, `ruoyi-react-frontend/src/pages/operations/floors/components/FloorTableColumns.tsx`, `ruoyi-react-frontend/src/pages/operations/floors/__tests__/components.render.test.tsx` |
| 3 | 全量验证门禁（无代码改动） | — | — |

## Decisions

### D1: 聚合失败告警降级而非中断 List（plan 未明确，Rule 2 补错误处理）
- **Reasoning**: 工位统计是只读增强列，聚合 SQL 失败不应拖垮楼层列表主流程。沿用同文件 `syncWorkstationBuildingID` 的 best-effort 先例，但补 `logrus.WithError(err).Warn` 告警（非静默吞），失败行 `WorkstationStats` 保持 nil → 前端渲染 "-"。
- **Tradeoff**: 统计缺失与"真无工位"在 UI 上不可区分；告警日志 `[floor] 工位统计聚合失败` 可定位。

### D2: 既有列数断言 8→9 + 新增 3 个 workstationStats render 测试
- **Reasoning**: `components.render.test.tsx` 的 `返回 8 列含主键` 断言 `cols.length === 8`，新列使其必然失败；更新为 9 列 + arrayContaining 含 `workstationStats`（本次功能直接导致的测试更新）。另按该文件既有 render 测试模式补 3 个用例锁定新列语义（"-" 兜底 / 维护仅 >0 显示 / Tag 三段必现）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `floor_service.go` 携带先前未提交的 Phase 91-03 重构入 Task 1 提交**
- **Found during:** Task 1 commit
- **Issue:** 工作区 1635 文件未提交波（含整个 `pkg/constants` 包、`floor_service_crud_test.go`、Phase 91 repo 化重构）中，`floor_service.go` 自身在任务开始前已有大量未提交修改（listRepo/base.GORMRepository 迁移、v129 WR-05 constants、Statistics 常量化）。文件级 staging 该文件时这些修改必然随之入提交（orchestrator 示例命令本身即按整文件列出 `floor_service.go`）。
- **Fix:** 保持文件级提交（hunk 级拆分需交互式 `git add -p`，不可用且会使提交树脱离已验证状态）。已核实提交内容 = 磁盘上 build+test 全绿的工作版本；无任何计划外文件被卷入（仅 plan 列出的 3 文件）。
- **Files modified:** `internal/services/operations/floor_service.go`（额外携带 ~170 行先前工作）
- **Commit:** `0d3e9670`

**2. [同上] Task 2 三个前端文件同样携带先前未提交工作**
- **Found during:** Task 2 commit
- **Issue:** `types/operations.ts`（全文注释对齐格式化 ~120 行）、`FloorTableColumns.tsx`（先前多行列定义重构）、`components.render.test.tsx`（整文件此前未被 git 跟踪，本次以 create mode 入库）。
- **Fix:** 同 D1，文件级提交；提交版本即 type-check + 71 vitest 全绿的工作版本。
- **Commit:** `03624395`

**注意（非本次引入，预先存在的分支状态）**: 该 backup 分支的已提交树本就不能独立编译（`pkg/constants` 等整包未跟踪），两次提交未改变此事实；所有验证均针对真实工作树执行并全绿。

## Files Touched

### Created
- `internal/services/operations/floor_workstation_stats_test.go` — 4 个测试：基本聚合 / 软删排除 / 空楼层 nil+omitempty / 分页隔离
- `ruoyi-react-frontend/src/pages/operations/floors/__tests__/components.render.test.tsx` — 此前未跟踪，本次补 3 个新列测试后随 Task 2 入库

### Modified
- `internal/models/operations/floor.go` — `FloorWorkstationStats` 类型（total/used/unused/maintenance）+ `OpsFloor.WorkstationStats *FloorWorkstationStats`（`gorm:"-"` + `json:"workstationStats,omitempty"`，MarshalJSON 经嵌入 `*Alias` 自动带出）
- `internal/services/operations/floor_service.go` — List 末尾注入 `attachWorkstationStats`：页内非空 FloorID 去重 → 单条 `Model(&models.Workstation{})` + `floor_id IN ?` + `GROUP BY floor_id` 聚合（CASE WHEN 按 `models.WorkstationStatus*` 常量分桶，软删自动排除）→ map 回填；无工位楼层保持 nil
- `ruoyi-react-frontend/src/types/operations.ts` — 导出 `FloorWorkstationStats` + `Floor.workstationStats?: FloorWorkstationStats | null`
- `ruoyi-react-frontend/src/pages/operations/floors/components/FloorTableColumns.tsx` — "面积(m²)"后插入"工位统计"列（width 260，flex wrap + Tag：总 default / 已用 processing / 空闲 success / 维护 warning 仅 >0）

## Verification（真实输出）

**Task 1（后端）：**
- `go build ./...` — PASS（exit 0）
- `go test ./internal/services/operations/ -run TestFloorWorkstationStats -v` — PASS：4/4（BasicAggregation 0.13s / SoftDeleteExcluded 0.10s / EmptyFloorNilAndOmitted 0.08s / PaginationIsolation 0.09s）
- `go test ./internal/services/operations/` — PASS（`ok ... 5.626s`，既有测试零回归）

**Task 2（前端）：**
- `npm run type-check` — PASS（`tsc --noEmit -p tsconfig.app.json` exit 0，零错误）
- `npx vitest run src/pages/operations/floors` — PASS：Test Files 10 passed (10)，Tests 71 passed (71)

**Task 3（全量门禁，所有改动就位后重跑）：**
- `go build ./...` — PASS（GO_BUILD_OK）
- `go test ./internal/services/operations/` — PASS（`ok ... 5.712s`）
- `npm run type-check` — PASS（TYPECHECK_OK）
- `npx vitest run src/pages/operations/floors` — PASS：10 files / 71 tests

## Success Criteria 对照

- ✅ 楼层表格每行"工位统计"列：总 / 已用(status=1) / 空闲(status=0) / 维护(status=2，仅 >0 显示)
- ✅ 每页仅多 1 条 `GROUP BY floor_id` 聚合 SQL（2 条 SQL 零 N+1，分页隔离测试锁定）
- ✅ 软删工位不计入（SoftDeleteExcluded）；无工位楼层显示 "-"（omitempty 不出键，EmptyFloorNilAndOmitted）
- ✅ go build + operations 包测试 + type-check + floors vitest 全绿
- ✅ 无 operlog（只读扩展）、无 DB migration（`gorm:"-"` 虚拟字段）

## Known Stubs

None。

## Self-Check: PASSED

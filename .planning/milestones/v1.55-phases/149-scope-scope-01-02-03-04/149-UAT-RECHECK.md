# Phase 149 · D1/D2 UAT 作废登记与重跑窗口（UAT 载体）

**性质**：只读登记件，非 `.planning/milestones/` 归档件的修改。
**依据**：ROADMAP `:85`（149 执行期成功标准：D1/D2「确认未收口」结论作废 + 重跑窗口登记）、
ROADMAP `:186`（「不要动 `.planning/milestones/` 归档件」——对 roadmapper 的写作约束，
归档件本期只读）、Lead 收口裁决（checker 小问题 3）。

---

## 1. 载体说明（为什么落在本 phase 目录，而不改归档件）

- `133-UAT.md:79-91` 的 D1/D2 原文记载「确认未收口」，属于 **v1.52 归档件**
  （`.planning/milestones/v1.52-phases/133-scope-final/133-UAT.md`）。
- ROADMAP `:186` 明确「不要动 `.planning/milestones/` 归档件」，是对 roadmapper 的
  写作约束；而 `:85` 是 **149 执行期成功标准**（D1/D2 作废）。二者并不矛盾：
  **不改归档件原文**，改在 149 phase 目录新建本登记件承载作废结论，等效满足 `:85`。
- 与 Phase 150 新建 `150-UAT-BACKFILL.md` 的做法一致（同波并行 phase 统一载体）。
- **本期未修改任何 `.planning/milestones/` 归档件、未勾选任何 UAT 完成项**
  （79 条人工验证跟踪表保持 0 勾选，LOCKED 裁定原文不动）。

---

## 2. 作废登记（D1/D2 旧结论）

| UAT 项 | 归档原文位置 | 旧结论 | Phase 149 处置 |
|---|---|---|---|
| **D1 几何/附件面抽查** | `133-UAT.md:82-85` | 「确认未收口」：walls / doors / floor-plan-texts / rooms-photos 独立读端点返回不含业务归属列的几何/展示/附件数据，按裁定显式豁免 | **旧结论作废**。Phase 149 已按方案 D 零迁移收口：5 类读端点经归属链按可见集裁剪、写端点单条 403 / 批量裁剪、保存侧后端权威裁剪 + 前端增量 diff 双层防误删 |
| **D2 location-alias 豁免复核** | `133-UAT.md:86-91` | 「确认该面未收口」（按 ds05 裁定 3 显式豁免） | **旧结论作废**。实测 `sys_dept_location_alias.go:22` 已有 `DeptID`，裁定 3「5 类模型都不含归属字段」对该实体不成立；已按 `dept_id ∈ 可见集` 直挂收口 |

> **不作废**：D3（`dept-mapping-template` 已收口复核，`133-UAT.md:92-94`）、`## E` 及之后所有段、
> 150 的 `132/133 backfill UAT` 另一段（ROADMAP `:130/:200` 硬登记，归 Phase 150）。
> 本登记件只覆盖 D1/D2（`:79-91`，不含 D3 `:92-94`）。

---

## 3. 重跑要点（供 QA 裁定重跑窗口时执行）

1. **Dept 档越界 `floorId` / `roomId`**：walls / doors / floor-plan-texts / rooms-photos /
   location-alias 列表只得到可见集内行；越界 `floorId`/`roomId` 读返回**空集**（不是 403）。
2. **断链 fail-closed**：`floor_id` 指向软删 `ops_floors` 的几何行、`room_id` 指向软删 /
   不存在 `ops_server_rooms` 的照片行、`dept_id` 不在可见集的 alias 行 → Dept 档读空集、
   写端点 403、批量裁剪为空；All 档短路仍可见（不为脏数据开旁路）。
3. **写端点 403 / 批量裁剪**：单条 Create/Update/Delete 越界 → 显式 HTTP 403；
   BatchOperation / BatchDelete / BatchDeletePhotos 按可见集裁剪后继续（不是整批 403）。
4. **保存侧几何行数前后断言**：非超管在 Three.js 编辑器保存整层几何后，自己看不到的
   几何元素不因「整体覆写」被误删（保存前后越界几何行数不变）。
5. **双方言 403 矩阵**：PG + SQLite 对同一 `ScopeResult` 断言 COUNT 与 List 行数一致、
   403 矩阵逐字段一致（本期 `TEST_PG_DSN` unset，PG contract 仅「skip 可观测」，
   见 `149-SUMMARY.md` Known Gaps）。
6. **rooms/photos 注入生效**：`router.go` rooms/photos 组补挂 `DataScopePermission` 后，
   handler 取到的 ctx 含 `datascope.ScopeResult`（补挂前 `ok=false` → 补挂后 `ok=true`）。

---

## 4. 重跑窗口与验收人

- **重跑窗口**：**待 QA 裁定**，本 phase 只登记、不擅自设定日期。
- **验收人**：**待 QA 裁定**，本 phase 不指定、不代签。
- **勾选状态**：本登记件与 `133-UAT.md` 的 D1/D2 **均未勾选完成**（`- [ ]` 保持）。
  实际重跑由人工执行后按 `notes/261108-uat-writeback-policy.md` 回写，AI 不擅自勾选、
  不写生产数据。

---

## 5. 自动化证据（本 phase 已落地，供重跑参照）

- 后端：`internal/services/operations/scope_entities_scope149_test.go`（5 类 × 六档
  List/GetByID/count/primary 矩阵 + 写侧 403 + 批量裁剪 + 断链脏数据 + 双方言静态断言
  + router 补挂契约 + PG contract skip 可观测），13 PASS / 1 SKIP。
- 就近接线：`wall_service_test.go` / `door_service_test.go` / `room_photo_service_test.go`
  / `floor_plan_text_crud_test.go` / `location_alias_service_test.go`（各 1 个
  `TestScope149_*_ServiceWiring`，反向变异验证有效）。
- 前端：`useFloorPlanEditor.test.tsx` 新增 2 个 `Phase 149 保存侧可见基线增量 diff`
  用例（越界 ID 不进任何请求集、可见元素正常提交、保存前后几何行数不变），10 tests PASS。
- 命令：`go test -count=1 ./internal/services/operations/ ./internal/api/v1/operations/ ./pkg/middleware/`、
  `go build ./...`、`npm run type-check`、`npx vitest run .../useFloorPlanEditor.test.tsx`、
  反向变异 3 项（红→还原→绿）、`node scripts/check-route-registry.mjs`（713 条路由零漂移）。

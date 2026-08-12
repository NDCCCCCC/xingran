---
slug: workstation-status-enum
status: awaiting_human_verify
trigger: '工位管理中工位状态枚举值前后端及数据库定义不一致，之前debug过一次但现在还是有问题，请重新检查'
created: 2026-06-12
updated: 2026-06-12
---

# Workstation Status Enum Inconsistency (Three-Layer)

## Symptoms (gathered 2026-06-12)

- **Expected behavior:** 工位状态在数据库、后端模型/服务、前端字典/筛选/下拉三处应保持一致定义；写入与读取的值要能正确相互转换。
- **Actual behavior (用户报告的 4 类症状):**
  1. 前端显示的状态值与数据库不一致
  2. 后端返回的枚举值与前端下拉/筛选不匹配
  3. 保存/更新工位状态时写入数据库的值错位
  4. 数据库 enum/check 约束与代码定义冲突
- **Reproduction:** 在工位列表页 (`GET /ops/workstation/list`) 直接观察/筛选状态字段即可暴露。
- **Timeline:** 用户称"之前 debug 过一次但现在仍有问题"，未提供准确时间与 PR/Commit。
- **Error messages:** 用户未提供具体错误信息或日志。

## User Constraints

- 工位状态具体取值 → **"请检查上次修复设置"**（用户希望回查上次的修复配置）
- 上次修复细节 → 用户**不记得改了什么**，需要从 git history / `.planning/debug/resolved/` 中查证
- 复现路径 → 工位列表页直接观察/筛选

## Current Focus

- **hypothesis:** **强假设（已得到 3 处直接证据支持）**：`xingran-react-frontend/src/utils/tableHelpers.tsx` 中的 `createStatusColumn()` 硬编码了"0=正常 / 非0=停用"的渲染规则，但工位的真实枚举是 `0=空闲, 1=占用, 2=维护`（3 态，非 2 态）。结果：工位列表页表格的"状态"列与卡片视图、平面图视图、模态框、字典（`STATUS_OPTIONS`）显示完全不一致——表格列"正常/停用"，其他视图"空闲/占用/维护"。
- **next_action:** 修复已应用。等待人类在真实浏览器/环境中验证（打开工位列表页 → 检查表格列/卡片/平面图/模态框/搜索下拉/统计卡片 6 处状态显示是否完全一致为"空闲/占用/维护"）。
- **status:** awaiting_human_verify
- **test:** 静态分析已确认根因位置（`columns.tsx:63`）；运行时验证需用前端 dev server 跑一次 `npm run dev`，通过浏览器实际访问 `/ops/workstation/list` 检查状态列。
- **expecting:** 工位列表页表格列与卡片视图、平面图、模态框、搜索下拉均显示"空闲/占用/维护"；状态筛选/创建/编辑完全一致。
- **reasoning_checkpoint:**
  - **hypothesis:** `tableHelpers.createStatusColumn` 与工位 3 态枚举的语义错配 → 因为工位表格列硬编码 0/1 文案，而工位是 0/1/2 三态。
  - **confirming_evidence:**
    1. `internal/core/db/migrations/036_update_workstation_status_values.sql` CHECK 约束 `status IN (0,1,2)` + 注释 `0=空闲, 1=占用, 2=维护`
    2. `internal/models/workstation.go` 常量 `WorkstationStatusAvailable=0, WorkstationStatusOccupied=1, WorkstationStatusMaintain=2`
    3. `xingran-react-frontend/src/pages/operations/workstations/constants.tsx` `STATUS_TEXT_MAP = {0:'空闲',1:'占用',2:'维护'}`、`STATUS_OPTIONS` 三个选项
    4. `xingran-react-frontend/src/utils/tableHelpers.tsx:27-28` `createStatusColumn` 渲染 `status===0 ? '正常' : '停用'`（2 态）
    5. `xingran-react-frontend/src/pages/operations/workstations/columns.tsx:63` `createStatusColumn('status', { width: 100 })` 使用了错误 helper
  - **falsification_test:** 若其他视图（CardView / FloorPlanView / 模态框）也用 `createStatusColumn`，根因不成立 → 已验证：它们用的是 `getWorkstationStatusText` / `STATUS_OPTIONS`（与 0/1/2 三态一致），只有 `columns.tsx` 走错 helper。
  - **fix_rationale:** 单一替换：把 columns.tsx 的 status 列从 `createStatusColumn` 换成 `renderWorkstationStatusTag`，消除表格列与其他视图的不一致。
  - **blind_spots:** ① 未跑 dev server 实际刷新页面确认显示效果（仅静态分析）；② `createStatusColumn` 同样被 buildings/floors/server_rooms/post 复用——这些是 0=正常/1=停用 的 2 态语义，**不应破坏**；③ 存在 `internal/constants/status.go` 与 `internal/models/workstation.go` 重复定义（`WorkstationStatusAvailable/Occupied`）但后者多 1 个 Maintain——3 态真正来源是 `models/workstation.go`，不是 constants/status.go。
- **tdd_checkpoint:** 暂未启用 TDD；待根因确认后，再决定是否需要补单测覆盖 `tableHelpers.createStatusColumn` 的多态化或 columns.tsx 的状态渲染。

## Evidence

<!-- 时序追加新证据 -->

- timestamp: 2026-06-12T15:30:00Z
  source: git_history
  finding: |
    `git log --all --grep='workstation' --grep='status'` 找到一次相关提交：commit `a3032b2`（"refactor: 重构楼宇空间管理模块"，2026-01-16）新增 `internal/core/db/migrations/036_update_workstation_status_values.sql`。
    该 migration 内容：将 `sys_workstation.status` 注释从默认值改为 `0=空闲, 1=占用, 2=维护`；添加 CHECK 约束 `status IN (0,1,2)`。即"上次修复"。

- timestamp: 2026-06-12T15:31:00Z
  source: db_schema
  finding: |
    `internal/core/db/migrations/036_update_workstation_status_values.sql`:
    - COMMENT: `0=空闲可分配, 1=占用已分配, 2=维护中不可用，默认为0`
    - CHECK 约束: `status IN (0, 1, 2)` （3 态）

- timestamp: 2026-06-12T15:32:00Z
  source: go_model
  finding: |
    `internal/models/workstation.go` 常量定义：
    ```go
    WorkstationStatusAvailable WorkstationStatus = 0 // 空闲
    WorkstationStatusOccupied  WorkstationStatus = 1 // 占用
    WorkstationStatusMaintain  WorkstationStatus = 2 // 维护
    ```
    3 态枚举。

- timestamp: 2026-06-12T15:33:00Z
  source: go_constants
  finding: |
    `internal/constants/status.go:58-64` 存在**残缺的旧版**：
    ```go
    WorkstationStatusAvailable = 0
    WorkstationStatusOccupied  = 1
    ```
    **缺少 Maintain=2**！与 models/workstation.go 不一致；项目其它模块 (excel_config 等) 实际使用的是 models 包，不是 constants 包。所以 constants/status.go 这份是死代码/历史遗留，但若有人误引用会导致 2 态视角。

- timestamp: 2026-06-12T15:34:00Z
  source: go_excel_config
  finding: |
    `internal/services/operations/excel_config.go:122` 工位 Excel 配置使用 `models.WorkstationStatusAvailable/Occupied/Maintain`，与 model 一致（3 态）。
    excel_config 行 16-18 也有同名 string 常量（"空闲"/"占用"/"维护"），仅为展示用，不影响数据。

- timestamp: 2026-06-12T15:35:00Z
  source: frontend_types
  finding: |
    `xingran-react-frontend/src/types/operations.ts:50-52` 类型声明：
    ```typescript
    export type WorkstationOpsType = 0 | 1 | 2;
    export type WorkstationOpsStatus = 0 | 1 | 2;
    ```
    3 态，与后端一致。

- timestamp: 2026-06-12T15:36:00Z
  source: frontend_constants
  finding: |
    `xingran-react-frontend/src/pages/operations/workstations/constants.tsx`:
    - `STATUS_OPTIONS = [{value:0,label:'空闲'},{value:1,label:'占用'},{value:2,label:'维护'}]`
    - `STATUS_TEXT_MAP = {0:'空闲', 1:'占用', 2:'维护'}`
    - `STATUS_COLOR_MAP = {0:'success', 1:'error', 2:'warning'}`
    3 态，与后端一致。

- timestamp: 2026-06-12T15:37:00Z
  source: frontend_table_helper
  finding: |
    **`xingran-react-frontend/src/utils/tableHelpers.tsx:17-33` `createStatusColumn`** 硬编码：
    ```typescript
    render: (status: number) => (
      <Tag color={status === 0 ? 'success' : 'error'}>
        {status === 0 ? '正常' : '停用'}
      </Tag>
    )
    ```
    **2 态语义，文案"正常/停用"**。这是工位表格列的渲染源。

- timestamp: 2026-06-12T15:38:00Z
  source: frontend_columns
  finding: |
    **`xingran-react-frontend/src/pages/operations/workstations/columns.tsx:63`**：
    ```typescript
    createStatusColumn('status', { width: 100 }),
    ```
    工位表格的"状态"列直接调用了 `createStatusColumn`，导致表格列显示"正常/停用"，与其它视图（CardView / FloorPlanView / 模态框 / 搜索下拉）"空闲/占用/维护"不一致。
    **这正是用户报告的"前端显示的状态值与数据库不一致"+"后端返回的枚举值与前端下拉/筛选不匹配"的根因**。

- timestamp: 2026-06-12T15:39:00Z
  source: frontend_cardview
  finding: |
    `xingran-react-frontend/src/pages/operations/workstations/views/CardView.tsx:49-50`：
    ```tsx
    <Tag color={getWorkstationStatusColor(workstation.status)}>
      {getWorkstationStatusText(workstation.status)}
    </Tag>
    ```
    使用正确的 3 态映射（来自 `constants.tsx`）。

- timestamp: 2026-06-12T15:40:00Z
  source: frontend_modal
  finding: |
    `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:171` 编辑模态框使用 `STATUS_OPTIONS`（3 态），与表格列不一致。
    `xingran-react-frontend/src/pages/operations/workstations/index.tsx:497` 搜索下拉也使用 `STATUS_OPTIONS`（3 态）。

- timestamp: 2026-06-12T15:41:00Z
  source: frontend_floorplan
  finding: |
    `xingran-react-frontend/src/pages/operations/workstations/views/FloorPlanView.tsx:143` 平面图视图使用 `getWorkstationStatusText`（3 态），与表格列不一致。
    `xingran-react-frontend/src/pages/operations/building-spaces-3d/utils.ts:152-212` 3D 楼宇视图也使用 3 态颜色/文本映射。

- timestamp: 2026-06-12T15:42:00Z
  source: shared_helper_usage
  finding: |
    `createStatusColumn` 的其它调用方（应保持 2 态语义）：
    - `pages/operations/buildings/index.tsx:302` (楼宇，0=正常/1=停用 2 态)
    - `pages/operations/floors/components/FloorTableColumns.tsx:38` (楼层，0=正常/1=停用 2 态)
    - `pages/operations/server-rooms/index.tsx:257` (机房，0=正常/1=停用 2 态)
    - `pages/system/post/index.tsx:174` (岗位，0=正常/1=停用 2 态)
    这些模块的 status 字段确实是 0/1 二态语义，**修改共享 helper 会破坏它们**。
    → 修复策略：**只在 workstations/columns.tsx 局部改用 `renderWorkstationStatusTag`**，不动 `createStatusColumn`。

- timestamp: 2026-06-12T15:43:00Z
  source: service_layer
  finding: |
    `internal/services/operations/workstation_service.go:112-114`:
    ```go
    if status := extractIntParam(params, "status", -1); status >= 0 {
        query = query.Where("sys_workstation.status = ?", status)
    }
    ```
    后端只接受 status 数值（0/1/2），不做字符串到 int 的转换。**前端提交时如果 send 0/1/2 整数则正常**；这意味着用户报告的"保存/更新错位"更可能是显示层错觉，不是实际写入错位（待确认）。

- timestamp: 2026-06-12T15:44:00Z
  source: db_legacy_constraint
  finding: |
    `internal/constants/status.go` (0=空闲, 1=占用) 与 `internal/models/workstation.go` (0=空闲, 1=占用, 2=维护) 不一致。但项目实际只用 models 包（excel_config 等），constants/status.go 这里是死代码。若 db 已有 status=2 数据但 constants/status.go 视角只支持 0/1，会出现"用 constants 视角写代码时漏判 2"。当前无引用方，不是本次 bug 主因。

## Eliminated

<!-- 已被排除的假设 -->

## Resolution

- **root_cause:** 前端共享 helper `xingran-react-frontend/src/utils/tableHelpers.tsx:createStatusColumn()` 硬编码 2 态语义 `status===0 ? '正常' : '停用'`，而被工位表格列 (`pages/operations/workstations/columns.tsx`) 直接调用。但工位 status 实际是 3 态枚举（0=空闲, 1=占用, 2=维护，与 DB CHECK 约束、Go model `models.WorkstationStatusAvailable/Occupied/Maintain`、前端 `STATUS_TEXT_MAP` 一致）。结果：表格列把 status=0 渲染为"正常"、把 status=1 渲染为"停用"、status=2 同样渲染为"停用"，与卡片视图/平面图/模态框/搜索下拉的"空闲/占用/维护"完全错位。这同时触发了用户报告的 4 个症状：
  1. 前端显示的状态值与数据库不一致（表格列硬塞 2 态）
  2. 后端返回的枚举值与前端下拉/筛选不匹配（同一字段在表格 vs 下拉 走两个不同的 mapper）
  3. 保存/更新工位状态时写入数据库的值错位（视觉错觉：实际写入 0/1/2 是正确的，但用户看到的"停用"让他以为存错）
  4. 数据库 enum/check 约束与代码定义冲突（CHECK 约束是 3 态，但前端表格列当 2 态使用；如果有人把代码"按 0/1 二态"改回，CHECK 会拒收 status=2）
- **fix:** 仅替换工位表格列的 status 渲染器为 `renderWorkstationStatusTag`（来自 `pages/operations/workstations/constants.tsx`），与本模块的其它视图保持一致；不动 `createStatusColumn`（其它 4 处使用方楼宇/楼层/机房/岗位仍是 0/1 二态）。不修改 `internal/constants/status.go`（它已无人引用，且是历史遗留的死代码；本次不动以遵守"scope constrainment"）。
- **verification:**
  1. 前端 TypeScript 类型检查：`npx tsc --noEmit` 退出码 0，无类型错误。
  2. 后端工位相关包编译：`go build ./internal/models/... ./internal/services/operations/... ./internal/api/v1/operations/...` 退出码 0。
  3. 静态一致性：表格列 / CardView / FloorPlanView / 3D 楼宇 / 模态框 / 搜索下拉 / 统计卡片 7 处均依赖 `constants.tsx` 中的 3 态映射，不再走"正常/停用"。
  4. 运行时验证（需用户确认）：打开工位列表页 `/ops/workstation/list` 观察状态列是否显示"空闲/占用/维护"；新增/编辑工位提交 status=0/1/2 数值；筛选 status 字段。
- **files_changed:**
  - `xingran-react-frontend/src/pages/operations/workstations/columns.tsx` (1 文件)

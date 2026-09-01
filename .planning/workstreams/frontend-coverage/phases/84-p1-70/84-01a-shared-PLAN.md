---
phase: 84-p1-70
plan: 01a
type: execute
wave: 1
depends_on:
  - 84-00
files_modified:
  - xingran-react-frontend/src/components/shared/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-01
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-01] components/shared 21 文件 892 stmts 语句覆盖率 ≥70%(实测 −0.5pp 余量后 bump floor)"
    - "[D-11] 每个组件测试至少 1 次 user event + 1 次 props 渲染断言(模式 A 锁定)"
    - "[D-12] 纯展示组件允许单一渲染断言 + 至少 2 个 props 组合快照"
    - "[D-13] antd Drawer/Modal/Select polyfill 走 setup.ts 集中沉淀,不每文件 inline"
    - "[QUAL-01] 159 存量测试不回归,新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/components/shared/__tests__/*.test.tsx
      provides: 共享组件 family 聚合测试(ModernTag/EmptyStateWithAction/ActionButtons/ErrorAlertWithRetry/BatchDeleteButton/BatchExportModal/FileUpload/GlobalSearch/ImageGallery/NetworkExport/ColumnConfigModal/Excel 系列/FloorPlanEditor 系列/DepartmentTreeSelect)
    - path: .coverage-fe-floors
      provides: components/shared 行 bump 至实测 −0.5pp
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-01a ratchet 行追加
  key_links:
    - from: xingran-react-frontend/src/components/shared/__tests__/
      to: xingran-react-frontend/src/test/utils/renderWithProviders.tsx
      via: import { renderWithProviders } from "@/test/utils/renderWithProviders"(D-06 复用 wave 0 harness)
    - from: xingran-react-frontend/src/components/shared/__tests__/
      to: xingran-react-frontend/src/test/utils/createApiMock.ts
      via: import { createApiMock } from "@/test/utils/createApiMock"(D-03 端点工厂)
    - from: xingran-react-frontend/src/components/shared/FloorPlanEditor/
      to: xingran-react-frontend/src/test/setup.ts
      via: ResizeObserver polyfill 集中沉淀(D-13)
---

<objective>
将 `components/shared/` 21 文件 892 stmts(ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry / BatchDeleteButton / BatchExportModal / FileUpload / GlobalSearch / ImageGallery / NetworkExport / ColumnConfigModal / Excel 系列 ExcelImport / ExcelImportLazy / ExcelExport / FloorPlanEditor 系列 FloorPlanEditor.tsx + .constants + .hooks + .panZoom + .types + DepartmentTreeSelect 等)语句覆盖率拉升至 ≥70%。按 D-11 模式 A 写测试(每个组件测试至少 1 次 user event + 1 次 props 渲染断言),纯展示组件按 D-12 允许单一渲染断言 + 至少 2 个 props 组合快照。FloorPlanEditor 因含 panZoom 状态机需重点拆 `.tsx`(mount + 缩放/拖动事件)+ `.panZoom.ts`(纯函数测试 screenToWorld/worldToScreen)+ `.hooks.ts`(自定义 hook 测试)+ `.types.ts` + `.constants.ts`(静态断言)五个子测试,避免单文件覆盖率 < 50% 拉低整目录(D-13 Pitfall #3)。复用 wave 0 沉淀的 renderWithProviders + createApiMock;同 PR bump components/shared floor 至实测 −0.5pp 并追加基线文档 ratchet 行。

Purpose: COMP-01 是 P1 五个组件组中"最高价值共享组件"——shared/* 是上层页面与组件的高频复用底座,覆盖率拉平直接提升下游 pages/* 集成测试稳定性,且 shared 内 FloorPlanEditor 是当前 shared 家族 stmts 最大单文件(panZoom 状态机不调内部 ref 方法覆盖率会 < 50%),必须分文件单测。

Output: 21 个测试文件(按 family 聚合到 `__tests__/`)、components/shared floor bump、基线文档 ratchet 行。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/workstreams/frontend-coverage/REQUIREMENTS.md
@.planning/workstreams/frontend-coverage/ROADMAP.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-RESEARCH.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-VALIDATION.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-00-harness-and-gate-PLAN.md
@xingran-react-frontend/src/components/shared/
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@xingran-react-frontend/src/test/utils/createApiMock.ts
@xingran-react-frontend/src/test/setup.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 共享原子组件 family 聚合测试(ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry / BatchDeleteButton / BatchExportModal / FileUpload / GlobalSearch / ImageGallery / NetworkExport / ColumnConfigModal / DepartmentTreeSelect)</name>
  <files>
    xingran-react-frontend/src/components/shared/__tests__/atomicComponents.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/batchOperations.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/fileUpload.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/globalSearch.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/imageGallery.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/networkExport.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/columnConfigModal.test.tsx
    xingran-react-frontend/src/components/shared/__tests__/departmentTreeSelect.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/shared/index.ts（确认导出列表）
    - xingran-react-frontend/src/components/shared/ModernTag.tsx / EmptyStateWithAction.tsx / ActionButtons.tsx / ErrorAlertWithRetry.tsx
    - xingran-react-frontend/src/components/shared/BatchDeleteButton.tsx / BatchExportModal.tsx
    - xingran-react-frontend/src/components/shared/FileUpload.tsx / GlobalSearch.tsx
    - xingran-react-frontend/src/components/shared/ImageGallery.tsx / NetworkExport.tsx
    - xingran-react-frontend/src/components/shared/ColumnConfigModal.tsx / DepartmentTreeSelect.tsx
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    - xingran-react-frontend/src/test/utils/createApiMock.ts
  </read_first>
  <action>
    1. 创建 `__tests__/atomicComponents.test.tsx` —— ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry family 聚合:
       - ModernTag (D-12 纯展示): `expect(screen.getByText("标签")).toBeInTheDocument()` × 2 props 组合(不同 color/status)
       - EmptyStateWithAction (D-12 纯展示): props = `{ description: "...", actionText: "重试" }` vs `{ description: "..." }` 两个 props 组合快照
       - ActionButtons: fireEvent.click 每个按钮断言 onClick spy;按钮 disabled 状态断言
       - ErrorAlertWithRetry: fireEvent.click 重试按钮断言 onRetry 被调 + 错误信息显示
    3. 创建 `__tests__/batchOperations.test.tsx` —— BatchDeleteButton / BatchExportModal:
       - BatchDeleteButton: fireEvent.click 触发 Modal.confirm + 确认回调断言
       - BatchExportModal: fireEvent.click 提交按钮 + createApiMock 拦截 export 端点断言
    4. 创建 `__tests__/fileUpload.test.tsx` —— FileUpload:
       - vi.mock("@/lib/api", ...) 拦截 upload 端点
       - fireEvent.change input file + waitFor upload 回调断言
    5. 创建 `__tests__/globalSearch.test.tsx` —— GlobalSearch:
       - fireEvent.change input + debounce 等待 + 搜索端点 mock 断言
       - fireEvent.keyDown Enter 触发 onSearch 回调
    6. 创建 `__tests__/imageGallery.test.tsx` —— ImageGallery:
       - 渲染多张图片 + fireEvent.click 触发 onPreview;空数组空态断言
    7. 创建 `__tests__/networkExport.test.tsx` —— NetworkExport:
       - vi.mock 网络导出 API + fireEvent.click 导出按钮 + 端点调用断言
    8. 创建 `__tests__/columnConfigModal.test.tsx` —— ColumnConfigModal:
       - renderWithProviders 打开 Modal + 列拖拽事件 + onSave 回调断言
       - createApiMock 拦截列配置保存端点
    9. 创建 `__tests__/departmentTreeSelect.test.tsx` —— DepartmentTreeSelect:
       - createApiMock 拦截 `/system/dept/tree` 端点返回 mock dept tree
       - renderWithProviders 渲染 + fireEvent.click 展开节点 + 选择节点断言 onChange 被调
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/shared/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - 8 个共享组件 family 测试通过,所有 family 覆盖率 ≥ 70%
  </done>
  <acceptance_criteria>
    - 8 个测试文件覆盖 ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry / BatchDeleteButton / BatchExportModal / FileUpload / GlobalSearch / ImageGallery / NetworkExport / ColumnConfigModal / DepartmentTreeSelect 共 12 个共享组件
    - 每个交互类组件至少 1 个 fireEvent + 1 个 props 渲染断言(D-11 模式 A)
    - 纯展示组件至少 2 个 props 组合断言(D-12)
    - harness (renderWithProviders / createApiMock) 至少被 3 个测试 import 使用(SC-6 Reuse 维度)
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Excel 导入导出与 FloorPlanEditor 系列测试(模式 A + panZoom 纯函数拆分)</name>
  <files>
    xingran-react-frontend/src/components/shared/__tests__/excelOperations.test.tsx
    xingran-react-frontend/src/components/shared/FloorPlanEditor/__tests__/FloorPlanEditor.test.tsx
    xingran-react-frontend/src/components/shared/FloorPlanEditor/__tests__/FloorPlanEditor.panZoom.test.ts
    xingran-react-frontend/src/components/shared/FloorPlanEditor/__tests__/FloorPlanEditor.hooks.test.tsx
    xingran-react-frontend/src/components/shared/FloorPlanEditor/__tests__/FloorPlanEditor.constants.test.ts
    xingran-react-frontend/src/components/shared/FloorPlanEditor/__tests__/FloorPlanEditor.types.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/components/shared/ExcelImport.tsx / ExcelImportLazy.tsx / ExcelExport.tsx
    - xingran-react-frontend/src/components/shared/FloorPlanEditor.tsx / FloorPlanEditor.constants.ts / FloorPlanEditor.hooks.ts / FloorPlanEditor.panZoom.ts / FloorPlanEditor.types.ts
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    - xingran-react-frontend/src/test/setup.ts（ResizeObserver 验证）
  </read_first>
  <action>
    1. 创建 `__tests__/excelOperations.test.tsx` —— ExcelImport / ExcelImportLazy / ExcelExport:
       - vi.mock("xlsx", ...) 避免真实解析大文件
       - ExcelImport: fireEvent.change 上传 .xlsx 文件 + 解析回调断言
       - ExcelImportLazy: render 加载占位 + Suspense resolve 后组件渲染断言
       - ExcelExport: vi.mock 列数据端点 + fireEvent.click 导出按钮 + download 触发断言
    2. 创建 `FloorPlanEditor/__tests__/FloorPlanEditor.test.tsx` —— 主组件:
       - renderWithProviders(<FloorPlanEditor />) + 容器渲染断言
       - fireEvent.click 缩放按钮 → 缩放 level state 变化断言
       - fireEvent.mouseDown / mouseMove / mouseUp 触发拖动 → 位置 state 变化断言
       - fireEvent.click 元素 → select state 变化断言
    3. 创建 `FloorPlanEditor/__tests__/FloorPlanEditor.panZoom.test.ts` —— 纯函数:
       - screenToWorld(worldX, worldY, viewport): 输入坐标 + 视点 → 世界坐标断言
       - worldToScreen(worldX, worldY, viewport): 反向转换断言
       - clampZoom(zoom, min, max): 边界值断言
       - getBounds(elements): 元素包围盒计算断言
    4. 创建 `FloorPlanEditor/__tests__/FloorPlanEditor.hooks.test.tsx` —— 自定义 hook:
       - renderHook + renderWithProviders 调用 useFloorPlan
       - act 中调用返回的 zoomIn/zoomOut/pan/reset 断言状态变化
    5. 创建 `FloorPlanEditor/__tests__/FloorPlanEditor.constants.test.ts` —— 静态常量:
       - 导出 DEFAULT_ZOOM / MIN_ZOOM / MAX_ZOOM / GRID_SIZE 常量值断言
    6. 创建 `FloorPlanEditor/__tests__/FloorPlanEditor.types.test.ts` —— 类型守卫:
       - 导出 isFloorPlanElement / createEmptyElement 工具函数断言
    7. FloorPlanEditor.tsx 渲染走 setup.ts 集中 ResizeObserver(D-13 沉淀),无需 inline polyfill
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/shared/__tests__ src/components/shared/FloorPlanEditor/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - Excel 系列 + FloorPlanEditor 5 文件测试通过,panZoom 状态机覆盖率 ≥70%
  </done>
  <acceptance_criteria>
    - FloorPlanEditor.tsx 测试覆盖 mount + 缩放 + 拖动 + 选择 4 类交互(D-11 模式 A)
    - FloorPlanEditor.panZoom.ts 纯函数 4 个工具函数测试覆盖
    - FloorPlanEditor.hooks.ts 自定义 hook renderHook 测试
    - FloorPlanEditor.constants.ts / .types.ts 静态断言测试
    - Excel 系列 3 文件测试覆盖 import/lazy/export
    - FloorPlanEditor 渲染无 ResizeObserver 报错(setup.ts 集中 polyfill 生效)
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 全量 vitest 验证 + components/shared floor bump + 基线文档 ratchet</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 components/shared 0.0）
    - .planning/frontend-coverage-baseline.md（ratchet 行追加格式参考）
    - 82-CONTEXT.md D-06 / D-07（ratchet 余量纪律）
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量测试 + 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含 `PASS: components/shared` 行(实测 pct ≥70%)
    3. 读 components/shared 实测 pct(从 gate 输出或 coverage-final.json),按 D-14 纪律 bump .coverage-fe-floors components/shared 行至 max(70.0, pct − 0.5) 并保留一位小数(向下截断,不四舍五入)
    4. 在 .planning/frontend-coverage-baseline.md 追加 84-01a ratchet 行,包含日期、phase 84-01a、weighted_avg、total_stmts、total_covered、0pct_pkg_count、commit 短 SHA、ratchet_from(0.0)→ratchet_to(实测−0.5pp)
    5. 再跑一次 gate 脚本确认 components/shared PASS,且无 FAIL
    6. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS:' | wc -l` 确认 PASS 行数 ≥ 38
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/shared'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - components/shared floor bump 至实测 −0.5pp;基线文档 ratchet 行追加;gate PASS
  </done>
  <acceptance_criteria>
    - npm run test:coverage 全量 exit 0,Tests ≥ 159 + 新增测试数
    - gate 输出含 PASS: components/shared X.XX% >= 70.0%(X.XX 为新 floor)
    - .coverage-fe-floors components/shared 行更新为新 floor
    - .planning/frontend-coverage-baseline.md 新增 84-01a ratchet 行
    - gate 总 FAIL 行数 = 0
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 共享组件测试 → React + antd | jsdom 渲染 + setup.ts polyfill,真实 antd 行为 |
| 共享组件测试 → @/lib/api | createApiMock 拦截,无真实网络 |
| FloorPlanEditor → canvas | jsdom 不支持 canvas 渲染,走 setup.ts ResizeObserver + state machine 断言 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-1a-01 | Denial of Service | FloorPlanEditor panZoom 状态机未测 | mitigate | 拆 .tsx + .panZoom.ts + .hooks.ts + .constants.ts + .types.ts 五子测试 |
| T-84-1a-02 | Tampering | xlsx mock 影响真实解析 | mitigate | vi.mock("xlsx") 替代真实包,fireEvent.change 触发 File |
| T-84-1a-03 | Information Disclosure | DepartmentTreeSelect mock dept 端点 | accept | 仅测试用 fixture,不暴露真实部门树 |
| T-84-1a-04 | Tampering | FileUpload 上传路径测试污染 | mitigate | vi.mock upload 端点,文件走 Blob/Mock,不入真实文件系统 |
| T-84-1a-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包;无安装步骤 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests 计数 ≥ 159 + 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS: components/shared'` 输出 ≥1 行 PASS,pct ≥70%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
4. `git diff .coverage-fe-floors` 显示 components/shared 行从 0.0 → 新值(单向上调)
5. `git diff .planning/frontend-coverage-baseline.md` 追加 84-01a ratchet 行
6. `grep -r 'renderWithProviders\|createApiMock' src/components/shared/ | wc -l` ≥ 3(SC-6 Reuse 维度)
</verification>

<success_criteria>
- components/shared 21 文件 892 stmts 语句覆盖率 ≥70%(COMP-01 满足,实测 −0.5pp 余量 bump floor)
- 全量 vitest 0 失败,159 存量测试 + 新增测试全 PASS(QUAL-01 不回归)
- FloorPlanEditor 5 子文件测试覆盖 panZoom 状态机 + 纯函数 + 自定义 hook + 静态常量 + 类型守卫
- 共享组件测试至少 1 个 fireEvent + 1 个 props 渲染断言(D-11 模式 A)
- 纯展示组件 2 个 props 组合快照(D-12)
- .coverage-fe-floors components/shared 行 bump + 基线文档 ratchet 行追加(同 PR,D-14)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-01a-shared-SUMMARY.md` when done
</output>
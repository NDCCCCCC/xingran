---
phase: 84-p1-70
plan: 03a
type: execute
wave: 3
depends_on:
  - 84-02a
  - 84-02b
files_modified:
  - xingran-react-frontend/src/components/network/port-write/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/network/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/reconciliation/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/table/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/three/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/DeptTree/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/IconSelect/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/NoticeDetail/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/NotificationBell/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/TargetSelector/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/markdown/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/modal/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/charts/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/asset/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-05
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-05] components/network 324 stmts(现 50.6%)+ components/reconciliation 144 stmts(现 18.1%)+ 零散小组件(table/three/DeptTree/IconSelect/NoticeDetail/NotificationBell/TargetSelector/markdown/modal/charts/asset 等)各目录 ≥70%"
    - "[D-11] BulkWriteDrawer 已 5 用例,补充 PortBindingModal/PortWriteModal/SetAccessVlanModal + MACEventsTimeline/MACHeatmapChart + HealthBadge/ExceptionMatchList/ReconciliationDrawer/ReconciliationTimeline 等"
    - "[D-12] 零散小组件(table/three/DeptTree 等)按 family 聚合或单测,确保每个 family 实测 ≥70%(避免被大组件拖低)"
    - "[QUAL-01] 159 存量测试不回归 + 新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/components/network/port-write/__tests__/PortBindingModal.test.tsx
      provides: 端口绑定 Modal(createApiMock 拦截 /network/port/binding)
    - path: xingran-react-frontend/src/components/network/port-write/__tests__/PortWriteModal.test.tsx
      provides: 端口写入 Modal
    - path: xingran-react-frontend/src/components/network/port-write/__tests__/SetAccessVlanModal.test.tsx
      provides: VLAN 设置 Modal
    - path: xingran-react-frontend/src/components/network/__tests__/MACEventsTimeline.test.tsx
      provides: MAC 事件时间线(echarts mock)
    - path: xingran-react-frontend/src/components/network/__tests__/MACHeatmapChart.test.tsx
      provides: MAC 热力图(echarts mock)
    - path: xingran-react-frontend/src/components/reconciliation/__tests__/HealthBadge.test.tsx
      provides: 健康徽章 5 种状态渲染断言(D-12 props 组合)
    - path: xingran-react-frontend/src/components/reconciliation/__tests__/ExceptionMatchList.test.tsx
      provides: 异常匹配列表
    - path: xingran-react-frontend/src/components/reconciliation/__tests__/ReconciliationDrawer.test.tsx
      provides: 对账抽屉(子 hook mock)
    - path: xingran-react-frontend/src/components/reconciliation/__tests__/ReconciliationTimeline.test.tsx
      provides: 对账时间线
    - path: xingran-react-frontend/src/components/{table,three,DeptTree,IconSelect,NoticeDetail,NotificationBell,TargetSelector,markdown,modal,charts,asset}/__tests__/*.test.tsx
      provides: 零散组件 family 测试
    - path: .coverage-fe-floors
      provides: components/network + components/reconciliation + 各零散 subdir 行各自 bump
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-03a 多个 ratchet 行追加
  key_links:
    - from: xingran-react-frontend/src/components/network/port-write/__tests__/PortBindingModal.test.tsx
      to: xingran-react-frontend/src/lib/networkApi.ts
      via: createApiMock("/network/port/binding") 拦截
    - from: xingran-react-frontend/src/components/network/__tests__/MACEventsTimeline.test.tsx
      to: echarts-for-react
      via: vi.mock("echarts-for-react") stub
    - from: xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx（已有）
      to: ../hooks/useReconciliationVisibility / useWorkstationHealth
      via: vi.mock 子 hook 链(已实证模式)
---

<objective>
将 wave 3 中段计划落地:(1)`components/network/` 324 stmts(现 50.6%)—— port-write 三个 Modal(PortBindingModal / PortWriteModal / SetAccessVlanModal)+ MAC 图表(MACEventsTimeline / MACHeatmapChart),每个独立测试;(2)`components/reconciliation/` 144 stmts(现 18.1%)—— HealthBadge 状态渲染(D-12 props 组合)+ ExceptionMatchList + ReconciliationDrawer + ReconciliationTimeline,沿用 HealthCard 已实证的 vi.mock 子 hook 模式;(3)零散小组件 —— table/three/DeptTree/IconSelect/NoticeDetail/NotificationBell/TargetSelector/markdown/modal/charts/asset 按 family 聚合到各 `__tests__/`,确保每个 family 实测 ≥70%(避免被大组件拖低导致整目录 < 70%);(4)同 PR 多个 subdir floor bump(components/network + components/reconciliation + 各零散 subdir)+ 基线文档多个 ratchet 行。

Purpose: COMP-05 是 P1 收口阶段。network 现 50.6% 拉平到 ≥70% 需补充 PortBindingModal 等三 Modal + MAC 图;reconciliation 现 18.1% 是最低洼目录,HealthCard 已有 5 用例可作模式参考(子 hook mock + ECharts mock),其余 4 个组件按同模式展开;零散小组件按 family 聚合即可,无需每组件单独测。

Output: network 5 + reconciliation 4 + 零散 ~11 = 多个测试文件;多个 subdir floor 各自 bump;基线文档多个 ratchet 行。
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
@xingran-react-frontend/src/components/network/
@xingran-react-frontend/src/components/reconciliation/
@xingran-react-frontend/src/components/{table,three,DeptTree,IconSelect,NoticeDetail,NotificationBell,TargetSelector,markdown,modal,charts,asset}/
@xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx（既有 5 用例,作模式参考）
@xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx（既有 5 用例,作模式参考）
@xingran-react-frontend/src/lib/networkApi.ts
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@xingran-react-frontend/src/test/utils/createApiMock.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: components/network 测试(port-write 三个 Modal + MAC 图)</name>
  <files>
    xingran-react-frontend/src/components/network/port-write/__tests__/PortBindingModal.test.tsx
    xingran-react-frontend/src/components/network/port-write/__tests__/PortWriteModal.test.tsx
    xingran-react-frontend/src/components/network/port-write/__tests__/SetAccessVlanModal.test.tsx
    xingran-react-frontend/src/components/network/__tests__/MACEventsTimeline.test.tsx
    xingran-react-frontend/src/components/network/__tests__/MACHeatmapChart.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx / PortWriteModal.tsx / SetAccessVlanModal.tsx
    - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx / MACHeatmapChart.tsx
    - xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx（既有模式）
    - xingran-react-frontend/src/lib/networkApi.ts
    - xingran-react-frontend/src/test/utils/createApiMock.ts
    - xingran-react-frontend/src/test/setup.ts（ResizeObserver 验证）
  </read_first>
  <action>
    1. 创建 `port-write/__tests__/PortBindingModal.test.tsx` —— 端口绑定:
       - createApiMock("/network/port/binding") 拦截端点 + mockResolvedValue({ bound: true })
       - renderWithProviders(<PortBindingModal portId="p-1" open={true} />) + 表单字段渲染断言
       - fireEvent.change port / vlan 输入 → onChange 断言
       - fireEvent.click 提交 → waitFor apiMock.endpoint 被调用 + 端点参数含 portId/vlan
       - 错误路径:apiMock.endpoint.mockRejectedValueOnce(new Error("...")) → 错误提示断言
    2. 创建 `port-write/__tests__/PortWriteModal.test.tsx` —— 端口写入:
       - createApiMock("/network/port/write") 拦截 + 表单 + 提交 + 端点调用断言
       - 多端口批量提交场景 + 端点接收数组参数断言
    3. 创建 `port-write/__tests__/SetAccessVlanModal.test.tsx` —— VLAN 设置:
       - createApiMock("/network/port/vlan") 拦截 + VLAN select change + 提交断言
       - fireEvent.click 取消 → onCancel 被调
    4. 创建 `__tests__/MACEventsTimeline.test.tsx` —— MAC 事件时间线(echarts mock):
       - vi.mock("echarts-for-react", () => ({ default: () => <div data-testid="echarts-mock" /> }))
       - renderWithProviders(<MACEventsTimeline events={[...]} />) + chart-mock 元素渲染断言
       - 不同 events props 数组长度断言(D-12 props 组合: 0 / 1 / 多条)
    5. 创建 `__tests__/MACHeatmapChart.test.tsx` —— MAC 热力图:
       - vi.mock echarts-for-react 同上
       - renderWithProviders(<MACHeatmapChart data={...} />) + chart-mock 渲染断言
       - 不同 data props 组合(空 / 单点 / 多点)
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/network/port-write/__tests__ src/components/network/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - network 5 个测试通过,port-write 三 Modal + MAC 图全覆盖,components/network ≥70%
  </done>
  <acceptance_criteria>
    - PortBindingModal/PortWriteModal/SetAccessVlanModal 各自测试 + createApiMock 拦截各自端点
    - MACEventsTimeline / MACHeatmapChart mock echarts-for-react + 不同 props 组合
    - BulkWriteDrawer 5 既有用例不回归(setup.ts ResizeObserver 集中沉淀生效)
    - components/network 实测 ≥70%
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: components/reconciliation 测试(HealthBadge + ExceptionMatchList + ReconciliationDrawer + ReconciliationTimeline,沿用 HealthCard 模式)</name>
  <files>
    xingran-react-frontend/src/components/reconciliation/__tests__/HealthBadge.test.tsx
    xingran-react-frontend/src/components/reconciliation/__tests__/ExceptionMatchList.test.tsx
    xingran-react-frontend/src/components/reconciliation/__tests__/ReconciliationDrawer.test.tsx
    xingran-react-frontend/src/components/reconciliation/__tests__/ReconciliationTimeline.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/reconciliation/HealthBadge.tsx
    - xingran-react-frontend/src/components/reconciliation/ExceptionMatchList.tsx
    - xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx
    - xingran-react-frontend/src/components/reconciliation/ReconciliationTimeline.tsx
    - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx（既有 5 用例 + vi.mock 子 hook + echarts mock 模式）
    - xingran-react-frontend/src/components/reconciliation/hooks/useReconciliationVisibility.ts / useWorkstationHealth.ts
  </read_first>
  <action>
    1. 创建 `__tests__/HealthBadge.test.tsx` —— 健康徽章 5 种状态(D-12 纯展示):
       - renderWithProviders(<HealthBadge status="healthy" />) + 渲染 healthy className
       - 5 个 props 组合:status = "healthy" / "warning" / "error" / "unknown" / "syncing"
       - assert 各 status 对应不同 className / 图标 / 文案(D-12 多 props 组合)
    2. 创建 `__tests__/ExceptionMatchList.test.tsx` —— 异常匹配列表:
       - mock 子 hook useExceptionMatches 返回 fixture 数据
       - renderWithProviders(<ExceptionMatchList />) + 列表项渲染断言
       - fireEvent.click 列表项 → onSelect 回调断言
       - 空列表空态断言
    3. 创建 `__tests__/ReconciliationDrawer.test.tsx` —— 对账抽屉:
       - vi.mock("../hooks/useReconciliationVisibility", ...) 返回 fixture visible + data
       - vi.mock("echarts-for-react", ...) chart stub
       - renderWithProviders(<ReconciliationDrawer open={true} />) + 抽屉内容渲染断言
       - fireEvent.click 关闭 → onClose 被调断言
       - fireEvent.click 提交对账 → createApiMock 拦截 /reconciliation/submit + 端点调用断言
    4. 创建 `__tests__/ReconciliationTimeline.test.tsx` —— 对账时间线:
       - vi.mock 子 hook useReconciliationTimeline 返回 fixture 时间线数据
       - renderWithProviders(<ReconciliationTimeline />) + 时间线条目渲染断言
       - fireEvent.click 条目 → 详情展开 + 详情内容渲染断言
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/reconciliation/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - reconciliation 4 个新测试通过,HealthCard 5 + 新 4 = 9 用例,components/reconciliation ≥70%
  </done>
  <acceptance_criteria>
    - HealthBadge 5 种 status props 渲染(D-12)
    - ExceptionMatchList 列表 + 空态 + fireEvent 选择
    - ReconciliationDrawer mock 子 hook + echarts + createApiMock 端点
    - ReconciliationTimeline mock 子 hook + 详情展开
    - HealthCard 5 既有用例不回归
    - components/reconciliation 实测 ≥70%
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 零散小组件 family 聚合测试(table / three / DeptTree / IconSelect / NoticeDetail / NotificationBell / TargetSelector / markdown / modal / charts / asset)</name>
  <files>
    xingran-react-frontend/src/components/table/__tests__/index.test.tsx
    xingran-react-frontend/src/components/three/__tests__/index.test.tsx
    xingran-react-frontend/src/components/DeptTree/__tests__/index.test.tsx
    xingran-react-frontend/src/components/IconSelect/__tests__/index.test.tsx
    xingran-react-frontend/src/components/NoticeDetail/__tests__/index.test.tsx
    xingran-react-frontend/src/components/NotificationBell/__tests__/index.test.tsx
    xingran-react-frontend/src/components/TargetSelector/__tests__/index.test.tsx
    xingran-react-frontend/src/components/markdown/__tests__/index.test.tsx
    xingran-react-frontend/src/components/modal/__tests__/index.test.tsx
    xingran-react-frontend/src/components/charts/__tests__/index.test.tsx
    xingran-react-frontend/src/components/asset/__tests__/index.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/table/index.tsx
    - xingran-react-frontend/src/components/three/index.tsx（如 FloorPlanEditor 已在 plan 1a 测过;此处仅测 three 目录下其他文件）
    - xingran-react-frontend/src/components/DeptTree/index.tsx
    - xingran-react-frontend/src/components/IconSelect/index.tsx
    - xingran-react-frontend/src/components/NoticeDetail/index.tsx
    - xingran-react-frontend/src/components/NotificationBell/index.tsx
    - xingran-react-frontend/src/components/TargetSelector/index.tsx
    - xingran-react-frontend/src/components/markdown/index.tsx
    - xingran-react-frontend/src/components/modal/index.tsx
    - xingran-react-frontend/src/components/charts/index.tsx
    - xingran-react-frontend/src/components/asset/index.tsx
  </read_first>
  <action>
    1. 创建 table/__tests__/index.test.tsx —— 表格基础组件(若存在):
       - renderWithProviders + 列定义 props 渲染断言
       - fireEvent.click 表头排序 → onSort 回调断言
       - 空数据空态断言
    2. 创建 three/__tests__/index.test.tsx —— three 目录下非 FloorPlanEditor 组件:
       - vi.mock three 渲染依赖(如 zrender / fiber)
       - renderWithProviders 容器存在断言 + props 变化断言
    3. 创建 DeptTree/__tests__/index.test.tsx —— 部门树:
       - createApiMock("/system/dept/tree") + 渲染树节点 + fireEvent.click 展开 + onSelect 断言
    4. 创建 IconSelect/__tests__/index.test.tsx —— 图标选择:
       - renderWithProviders + 图标列表渲染 + fireEvent.click 选中 + onChange 断言
    5. 创建 NoticeDetail/__tests__/index.test.tsx —— 通知详情:
       - renderWithProviders + props 渲染 + fireEvent.click 关闭 → onClose 断言
    6. 创建 NotificationBell/__tests__/index.test.tsx —— 通知铃铛:
       - vi.mock("@/store/noticeStore") fixture unread count + 渲染徽章数字
       - fireEvent.click 铃铛 → 通知列表下拉断言 + fireEvent.click 通知 → onRead 断言
    7. 创建 TargetSelector/__tests__/index.test.tsx —— 目标选择器:
       - renderWithProviders + 选项列表 + fireEvent.click 选中 + onChange 断言
    8. 创建 markdown/__tests__/index.test.tsx —— Markdown 渲染:
       - renderWithProviders + markdown 字符串 props → HTML 元素渲染断言
    9. 创建 modal/__tests__/index.test.tsx —— Modal 包装:
       - renderWithProviders + 打开/关闭 + onConfirm/onCancel 回调断言
    10. 创建 charts/__tests__/index.test.tsx —— 图表组件:
        - vi.mock echarts-for-react 同 healthCard 模式 + 渲染断言
    11. 创建 asset/__tests__/index.test.tsx —— 资产组件:
        - renderWithProviders + props 渲染 + fireEvent 操作断言
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/table src/components/three src/components/DeptTree src/components/IconSelect src/components/NoticeDetail src/components/NotificationBell src/components/TargetSelector src/components/markdown src/components/modal src/components/charts src/components/asset 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - 零散 11 个组件 family 测试通过,每个 family 实测 ≥70%
  </done>
  <acceptance_criteria>
    - 11 个零散组件 family 各自单测
    - 若某 family 实测 < 70%,需拆分或补充断言(Pitfall #4 防 family 聚合过粗)
    - echarts mock 模式跨 charts/three 复用 HealthCard 实证
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 4: 全量 vitest 验证 + 多个 subdir floor bump + 基线文档多个 ratchet 行</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 components/network 0.0 / components/reconciliation 0.0 / 各零散 0.0）
    - .planning/frontend-coverage-baseline.md
    - 82-CONTEXT.md D-06/D-07
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量 + 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含多个 PASS: 行(network / reconciliation / 各零散 subdir,实测 pct 均 ≥70%)
    3. 按 D-14 + D-07 多个 subdir 各自独立 bump .coverage-fe-floors 多行:
       - components/network
       - components/reconciliation
       - components/table / three / DeptTree / IconSelect / NoticeDetail / NotificationBell / TargetSelector / markdown / modal / charts / asset(各 subdir 视 gate 是否识别为独立行决定是否 bump;若 gate 只聚合到 components 聚合行,只 bump 聚合行)
       - 注:84-CONTEXT D-04 锁 .coverage-fe-floors 新增 9 个 components subdir 行(已 plan 0 落地),本 plan 不新增行,只 bump 已有行
    4. 在 .planning/frontend-coverage-baseline.md 追加多个 84-03a ratchet 行(D-07 互不掩盖)
    5. 再跑 gate 确认所有 subdir PASS 且无 FAIL
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/(network|reconciliation)'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - 多个 subdir floor 各自 bump;基线文档追加多个 ratchet 行;gate 多行 PASS
  </done>
  <acceptance_criteria>
    - npm run test:coverage exit 0,Tests ≥ 159 + 新增测试数
    - gate 输出 PASS: components/network X.XX% >= 70.0% + PASS: components/reconciliation X.XX% >= 70.0%
    - .coverage-fe-floors 多行更新
    - .planning/frontend-coverage-baseline.md 新增多行 84-03a ratchet
    - gate 总 FAIL 行数 = 0
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| network 测试 → lib/networkApi | createApiMock 拦截 |
| network MAC 图 → echarts-for-react | vi.mock stub |
| reconciliation 测试 → 子 hooks | vi.mock 子 hook 链(HealthCard 实证模式) |
| reconciliation 测试 → echarts | vi.mock chart stub |
| 零散 family 测试 → 子 store / 子 API | 按需 mock |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-3a-01 | Tampering | port-write Modal 真实网络端口写入 | mitigate | createApiMock 拦截,无真实写入 |
| T-84-3a-02 | Denial of Service | echarts 真实渲染 jsdom 报错 | mitigate | vi.mock("echarts-for-react") stub |
| T-84-3a-03 | Information Disclosure | reconciliation 子 hook fixture 暴露真实对账数据 | mitigate | 测试用 fixture 数据,不含真实业务 |
| T-84-3a-04 | Tampering | 零散 family 聚合过粗覆盖率虚假 | mitigate | Pitfall #4: 单 family < 70% 拆分或补充断言 |
| T-84-3a-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests ≥ 159 + 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/(network|reconciliation)'` 输出 ≥2 行 PASS,各 pct ≥70%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
4. `git diff .coverage-fe-floors` 显示多行单向上调
5. `git diff .planning/frontend-coverage-baseline.md` 追加多行 84-03a ratchet
6. `grep -r 'renderWithProviders\|createApiMock' src/components/network/ src/components/reconciliation/ | wc -l` ≥ 3(SC-6 Reuse)
</verification>

<success_criteria>
- components/network 324 + components/reconciliation 144 + 零散小组件各目录 ≥70%(COMP-05 满足)
- 全量 vitest 0 失败,159 存量 + 新增测试全 PASS(QUAL-01 不回归)
- network 5 测试(PortBindingModal/PortWriteModal/SetAccessVlanModal/MACEventsTimeline/MACHeatmapChart)+ BulkWriteDrawer 5 既有用例
- reconciliation 4 新测试 + HealthCard 5 既有用例
- 零散 11 family 各 family ≥70%(Pitfall #4 防范)
- .coverage-fe-floors 多行 bump + 基线文档多 ratchet 行追加(同 PR)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-03a-network-recon-misc-SUMMARY.md` when done
</output>
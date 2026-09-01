---
phase: 84-p1-70
plan: 01b
type: execute
wave: 1
depends_on:
  - 84-00
files_modified:
  - xingran-react-frontend/src/components/dashboard/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/dashboard/widgets/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/dashboard/layout/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/dashboard/settings/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/dashboard/utils/__tests__/*.test.ts
  - xingran-react-frontend/src/components/dashboard/templates/__tests__/*.test.ts
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-02
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-02] components/dashboard 29 文件 1068 stmts 语句覆盖率 ≥70%(widgets + templates + utils + settings + layout 五子目录 family 聚合)"
    - "[D-11] Widget 体系测试每个 Widget 单独断言 props → 渲染产物,避免 mock widgetRegistry 整体聚合(Pitfall #4 防范)"
    - "[D-12] widgetRegistry / presets.ts 等纯数据模块允许单一静态断言 + 导出完整性"
    - "[QUAL-01] 159 存量测试不回归 + 新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/components/dashboard/widgets/__tests__/*.test.tsx
      provides: BaseWidget + ChartWidget/ListWidget/MetricWidget/ProgressWidget/StatCardWidget/TableWidget 各 *Widget 单测 + WidgetRenderer 分发测试
    - path: xingran-react-frontend/src/components/dashboard/layout/__tests__/*.test.tsx
      provides: DashboardGrid/GridItem/LayoutToolbar/TemplatePreview/TemplateSelector 测试(mock react-grid-layout + useDashboardStore)
    - path: xingran-react-frontend/src/components/dashboard/settings/__tests__/*.test.tsx
      provides: DashboardScopeSelector/DashboardSettings/DataSourceForm/DisplayConfigForm/EndpointSelector/ParamsEditor/RefreshIntervalSelector/WidgetSelector 表单交互测试(mock utils/dataFetcher)
    - path: xingran-react-frontend/src/components/dashboard/utils/__tests__/*.test.ts
      provides: dataFetcher 纯类测试(API mock + cache + error path)
    - path: xingran-react-frontend/src/components/dashboard/templates/__tests__/*.test.ts
      provides: presets.ts 纯数据 + DEFAULT_* 常量完整性
    - path: .coverage-fe-floors
      provides: components/dashboard 行 bump 至实测 −0.5pp
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-01b ratchet 行追加
  key_links:
    - from: xingran-react-frontend/src/components/dashboard/widgets/__tests__/
      to: xingran-react-frontend/src/components/dashboard/widgets/widgetRegistry.ts
      via: widgetRegistry dispatch → 真实 *Widget 渲染(避免 mock 整个 widgetRegistry)
    - from: xingran-react-frontend/src/components/dashboard/settings/__tests__/
      to: xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts
      via: vi.mock("../utils/dataFetcher") 子模块 mock
    - from: xingran-react-frontend/src/components/dashboard/layout/__tests__/
      to: xingran-react-frontend/src/store/dashboardStore.ts
      via: beforeEach useDashboardStore.setState({ widgets: [] }) reset
---

<objective>
将 `components/dashboard/` 29 文件 1068 stmts(widgets/* ChartWidget/ListWidget/MetricWidget/ProgressWidget/StatCardWidget/TableWidget/BaseWidget/WidgetRenderer/widgetRegistry + layout/* DashboardGrid/GridItem/LayoutToolbar/TemplatePreview/TemplateSelector + settings/* DashboardScopeSelector/DashboardSettings/DataSourceForm/DisplayConfigForm/EndpointSelector/ParamsEditor/RefreshIntervalSelector/WidgetSelector + utils/dataFetcher + templates/presets + 编辑器 WidgetDataFilter/WidgetEditor)语句覆盖率拉升至 ≥70%。重点防范 Pitfall #4:**widgets 测试必须按子组件单独测**——BaseWidget.test.tsx 独立断言 base 行为、各 *Widget.test.tsx 单独渲染断言 props → 渲染产物、widgetRegistry.test.tsx 静态断言每个 widget type 都注册到 component map、presets.ts 测试 createWidget 工具 + DEFAULT_* 常量完整性;不可走 family mock widgetRegistry 整体聚合的省事路径。layout 测试 mock react-grid-layout 第三方(避免 jsdom 渲染异常)+ useDashboardStore 按需 reset;settings 测试 mock dataFetcher 子模块;utils/dataFetcher 走独立纯类测试(API mock + cache + error path)。复用 wave 0 harness;同 PR bump components/dashboard floor 并追加基线文档 ratchet。

Purpose: COMP-02 是 P1 最大单目录(1068 stmts)——dashboard widget 体系是产品核心可视化能力,且 family 聚合 mock 风险最高(Pitfall #4),必须每个 Widget 独立单测确保覆盖率含金量。

Output: 多个 family 测试文件(按子目录聚合到 `__tests__/`)、components/dashboard floor bump、基线文档 ratchet 行。
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
@xingran-react-frontend/src/components/dashboard/
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@xingran-react-frontend/src/test/utils/createApiMock.ts
@xingran-react-frontend/src/store/dashboardStore.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: dashboard widgets 子组件 family 测试(防 Pitfall #4 mock 整体聚合)</name>
  <files>
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/BaseWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/ChartWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/ListWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/MetricWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/ProgressWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/StatCardWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/TableWidget.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/WidgetRenderer.test.tsx
    xingran-react-frontend/src/components/dashboard/widgets/__tests__/widgetRegistry.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/dashboard/widgets/BaseWidget.tsx
    - xingran-react-frontend/src/components/dashboard/widgets/ChartWidget.tsx / ListWidget.tsx / MetricWidget.tsx / ProgressWidget.tsx / StatCardWidget.tsx / TableWidget.tsx
    - xingran-react-frontend/src/components/dashboard/widgets/WidgetRenderer.tsx
    - xingran-react-frontend/src/components/dashboard/widgets/widgetRegistry.ts
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
  </read_first>
  <action>
    1. 创建 `widgets/__tests__/BaseWidget.test.tsx` —— 独立断言 base 行为:
       - renderWithProviders(<BaseWidget title="...">{content}</BaseWidget>) 渲染
       - title 文本断言(D-12 props 组合: 不同 title)
       - children 内容渲染断言
    2. 创建 `widgets/__tests__/ChartWidget.test.tsx` —— 真实渲染:
       - vi.mock("echarts-for-react", () => ({ default: () => <div data-testid="echarts-mock" /> }))
       - 传入 data + option props → chart-mock 元素渲染断言
       - fireEvent.click toolbar 按钮 → onToolClick 回调断言
    3. 创建 `widgets/__tests__/ListWidget.test.tsx` / `MetricWidget.test.tsx` / `ProgressWidget.test.tsx` / `StatCardWidget.test.tsx` / `TableWidget.test.tsx` —— 各 Widget 单独 props → 渲染断言:
       - renderWithProviders 渲染各 Widget
       - 断言 props 触发的文本/数字/列表项/进度条/表格行
       - 至少 1 个 fireEvent 事件(展开/排序/筛选/分页) + onXxx 回调断言
    4. 创建 `widgets/__tests__/WidgetRenderer.test.tsx` —— 分发测试:
       - 真实 widgetRegistry dispatch(不 mock) + 断言分发到正确组件
       - props.type = "chart" 渲染 ChartWidget → chart-mock 元素
       - props.type = "metric" 渲染 MetricWidget → 数字文本
    5. 创建 `widgets/__tests__/widgetRegistry.test.tsx` —— 静态断言(D-12 纯数据):
       - 遍历 registry: 每个 widget type 都注册到 component map
       - 默认 widget types = ["chart", "list", "metric", "progress", "stat-card", "table"]
       - assert getWidgetComponent(type) 返回非 undefined
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/dashboard/widgets/__tests__ 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - 9 个 widgets 测试文件通过,各 Widget 真实渲染覆盖率 ≥60%(防 mock 整体聚合)
  </done>
  <acceptance_criteria>
    - 9 个测试文件覆盖 BaseWidget + 6 个 *Widget + WidgetRenderer + widgetRegistry
    - 每个 *Widget 测试真实渲染(props → 渲染产物断言),非 mock 整体 widgetRegistry
    - vi.mock("echarts-for-react") 避免 jsdom canvas 报错(HealthCard 实证模式)
    - widgetRegistry.test.tsx 静态断言导出完整性
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: dashboard layout + settings + utils + templates 子目录测试(mock react-grid-layout + useDashboardStore reset + 子模块 mock)</name>
  <files>
    xingran-react-frontend/src/components/dashboard/layout/__tests__/DashboardGrid.test.tsx
    xingran-react-frontend/src/components/dashboard/layout/__tests__/GridItem.test.tsx
    xingran-react-frontend/src/components/dashboard/layout/__tests__/TemplatePreview.test.tsx
    xingran-react-frontend/src/components/dashboard/layout/__tests__/TemplateSelector.test.tsx
    xingran-react-frontend/src/components/dashboard/layout/__tests__/LayoutToolbar.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardSettings.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/DataSourceForm.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/DisplayConfigForm.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/EndpointSelector.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/ParamsEditor.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/RefreshIntervalSelector.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/WidgetSelector.test.tsx
    xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx
    xingran-react-frontend/src/components/dashboard/utils/__tests__/dataFetcher.test.ts
    xingran-react-frontend/src/components/dashboard/templates/__tests__/presets.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/components/dashboard/layout/DashboardGrid.tsx / GridItem.tsx / TemplatePreview.tsx / TemplateSelector.tsx / LayoutToolbar.tsx
    - xingran-react-frontend/src/components/dashboard/settings/*.tsx（8 文件）
    - xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts
    - xingran-react-frontend/src/components/dashboard/templates/presets.ts
    - xingran-react-frontend/src/store/dashboardStore.ts
    - xingran-react-frontend/src/test/utils/createApiMock.ts
  </read_first>
  <action>
    1. 创建 `layout/__tests__/` 5 文件 —— mock react-grid-layout + useDashboardStore reset:
       - vi.mock("react-grid-layout", () => ({ ResponsiveGridLayout: (p) => <div data-testid="grid-mock">{p.children}</div> }))
       - beforeEach: useDashboardStore.setState({ widgets: [] }) (D-05 reset)
       - DashboardGrid: 渲染 grid-mock + 子 widget 断言;fireEvent.drop 触发 onLayoutChange
       - GridItem: 渲染单项 + 删除按钮 fireEvent.click + onRemove 回调断言
       - TemplatePreview: 模板预览渲染(D-12 静态) + 不同 template props 组合
       - TemplateSelector: 模板列表渲染 + fireEvent.click 选中模板 + onSelect 回调
       - LayoutToolbar: 工具栏按钮 fireEvent + onAction 回调断言
    2. 创建 `settings/__tests__/` 8 文件 —— mock 子模块 utils/dataFetcher:
       - vi.mock("../../utils/dataFetcher", () => ({ fetchWidgetData: vi.fn() }))
       - DashboardSettings: 表单提交 fireEvent.click + onSave 回调断言
       - DataSourceForm: 数据源选择 fireEvent + 类型变化断言
       - DisplayConfigForm: 显示配置 toggle + onChange 断言
       - EndpointSelector: API 端点 select + fireEvent.change + onChange 断言
       - ParamsEditor: 参数编辑 + fireEvent.click 添加行 + onChange 断言
       - RefreshIntervalSelector: 刷新间隔 select + onChange 断言
       - WidgetSelector: widget 选择 fireEvent + onSelect 断言
       - DashboardScopeSelector: 作用域选择 + onChange 断言
    3. 创建 `utils/__tests__/dataFetcher.test.ts` —— 纯类测试:
       - createApiMock 拦截 widget data 端点 + fetchWidgetData 调用 + cache 命中/过期分支
       - 错误路径: 端点返回 4xx → 抛出 Error
       - WebSocket 路径: 简单 mock WebSocket 接口断言连接建立
    4. 创建 `templates/__tests__/presets.test.ts` —— 静态数据断言:
       - DEFAULT_WIDGET_TYPES / DEFAULT_LAYOUT 常量完整性
       - createWidget(type, options) 工具函数 + 不同 type 返回不同默认配置
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/dashboard/layout/__tests__ src/components/dashboard/settings/__tests__ src/components/dashboard/utils/__tests__ src/components/dashboard/templates/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - 14 个 layout + settings + utils + templates 测试通过,dashboard 全目录覆盖率 ≥70%
  </done>
  <acceptance_criteria>
    - layout 5 测试覆盖 DashboardGrid/GridItem/TemplatePreview/TemplateSelector/LayoutToolbar(mock react-grid-layout)
    - settings 8 测试覆盖 DashboardSettings 8 个表单组件(mock dataFetcher 子模块)
    - utils/dataFetcher.test.ts 独立纯类测试覆盖 cache + error path
    - templates/presets.test.ts 静态数据 + createWidget 工具测试
    - useDashboardStore.setState 在 layout 测试 beforeEach 中 reset(Pitfall #2 防范)
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 全量 vitest 验证 + components/dashboard floor bump + 基线文档 ratchet</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 components/dashboard 0.0）
    - .planning/frontend-coverage-baseline.md（追加行格式参考）
    - 82-CONTEXT.md D-06/D-07（ratchet 纪律）
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量 + 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含 `PASS: components/dashboard` 行(实测 pct ≥70%)
    3. 读 components/dashboard 实测 pct,按 D-14 bump .coverage-fe-floors components/dashboard 行至 max(70.0, pct − 0.5) 并保留一位小数(向下截断)
    4. 在 .planning/frontend-coverage-baseline.md 追加 84-01b ratchet 行(同 PR,D-14)
    5. 再跑 gate 确认 components/dashboard PASS 且无 FAIL
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/dashboard'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - components/dashboard floor bump 至实测 −0.5pp;基线文档 ratchet 行追加;gate PASS
  </done>
  <acceptance_criteria>
    - npm run test:coverage exit 0,Tests ≥ 159 + 新增测试数
    - gate 输出含 PASS: components/dashboard X.XX% >= 70.0%
    - .coverage-fe-floors components/dashboard 行更新
    - .planning/frontend-coverage-baseline.md 新增 84-01b ratchet 行
    - gate 总 FAIL 行数 = 0
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| dashboard widgets → echarts-for-react | vi.mock 第三方,jsdom 不渲染真实图表 |
| dashboard layout → react-grid-layout | vi.mock 第三方,避免 jsdom 渲染异常 |
| dashboard settings → utils/dataFetcher | vi.mock 子模块,避免真实 API 调用 |
| dashboard store → useDashboardStore | beforeEach setState reset,避免测试间状态泄漏 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-1b-01 | Tampering | widgetRegistry mock 整体聚合导致覆盖率虚假 | mitigate | Pitfall #4: 各 *Widget 独立真实渲染断言 |
| T-84-1b-02 | Denial of Service | react-grid-layout jsdom 渲染异常 | mitigate | vi.mock("react-grid-layout") stub 组件 |
| T-84-1b-03 | Information Disclosure | useDashboardStore 状态泄漏 | mitigate | beforeEach useDashboardStore.setState({ widgets: [] }) |
| T-84-1b-04 | Tampering | dataFetcher 真实 WebSocket 连接 | mitigate | mock WebSocket 接口,不入真实网络 |
| T-84-1b-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests ≥ 159 + 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS: components/dashboard'` 输出 ≥1 行 PASS,pct ≥70%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
4. `git diff .coverage-fe-floors` 显示 components/dashboard 行单向上调
5. `git diff .planning/frontend-coverage-baseline.md` 追加 84-01b ratchet 行
6. `grep -r 'renderWithProviders\|createApiMock' src/components/dashboard/ | wc -l` ≥ 3(SC-6 Reuse)
</verification>

<success_criteria>
- components/dashboard 29 文件 1068 stmts 语句覆盖率 ≥70%(COMP-02 满足)
- 全量 vitest 0 失败,159 存量 + 新增测试全 PASS(QUAL-01 不回归)
- widgets 测试按子组件单独测,避免 mock 整体 widgetRegistry 聚合(Pitfall #4)
- layout 测试 mock react-grid-layout + useDashboardStore reset(Pitfall #2 防范)
- settings 测试 mock 子模块 utils/dataFetcher
- utils/dataFetcher 独立纯类测试覆盖 cache + error path
- .coverage-fe-floors components/dashboard 行 bump + 基线文档 ratchet 行追加(同 PR)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-01b-dashboard-SUMMARY.md` when done
</output>
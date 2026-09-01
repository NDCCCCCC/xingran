---
phase: 84-p1-70
plan: 02b
type: execute
wave: 2
depends_on:
  - 84-01a
  - 84-01b
files_modified:
  - xingran-react-frontend/src/components/CronSelector/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/CronSelector/__tests__/*.test.ts
  - xingran-react-frontend/src/components/captcha/__tests__/*.test.tsx
  - xingran-react-frontend/src/components/operations/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-04
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-04] components/CronSelector 316 stmts + components/captcha 154 stmts + components/operations 149 stmts 各自语句覆盖率 ≥70%(D-07 三个 subdir 各自独立 floor,ratchet 互不掩盖)"
    - "[D-08 模式] CronSelector utils.ts / constants.ts 走真实 @breejs/later + cron-validate + cron-parser 向量直测(83 D-08 国密模式),非 mock"
    - "[D-11] captcha 测试 mock verifySliderCaptcha 返回不同结果(onVerified/onError),不模拟真实 PointerEvent 拖动过程(Pitfall #5 简化)"
    - "[QUAL-01] 159 存量测试不回归 + 新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/components/CronSelector/__tests__/utils.test.ts
      provides: expression ↔ config 往返 + getNextRunTimes 真实算法测试
    - path: xingran-react-frontend/src/components/CronSelector/__tests__/constants.test.ts
      provides: DEFAULT_CRON 常量 + 字段配置完整性
    - path: xingran-react-frontend/src/components/CronSelector/__tests__/CronSelector.test.tsx
      provides: CronSelector 组件交互测试(选择预设 / 自定义 / 输入验证)
    - path: xingran-react-frontend/src/components/captcha/__tests__/SliderCaptcha.test.tsx
      provides: 加载 / 刷新 / verify 成功/失败 4 类断言
    - path: xingran-react-frontend/src/components/captcha/__tests__/TextCaptcha.test.tsx
      provides: 文本验证码输入断言
    - path: xingran-react-frontend/src/components/captcha/__tests__/CaptchaModal.test.tsx
      provides: Modal 打开 / 验证 / 关闭 + onSuccess 传 token 断言
    - path: xingran-react-frontend/src/components/operations/__tests__/WorkstationDeviceTable.test.tsx
      provides: 工位设备表格 opsApi 包装型契约测试
    - path: .coverage-fe-floors
      provides: components/CronSelector + components/captcha + components/operations 三行 bump 至实测 −0.5pp
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-02b 三个 ratchet 行追加
  key_links:
    - from: xingran-react-frontend/src/components/CronSelector/__tests__/utils.test.ts
      to: @breejs/later / cron-validate / cron-parser
      via: 真实调用,确定性字符串向量
    - from: xingran-react-frontend/src/components/captcha/__tests__/SliderCaptcha.test.tsx
      to: xingran-react-frontend/src/services/captcha.ts
      vi.mock: 拦截 mock verifySliderCaptcha
    - from: xingran-react-frontend/src/components/operations/__tests__/WorkstationDeviceTable.test.tsx
      to: xingran-react-frontend/src/lib/opsApi.ts
      via: vi.mock("@/lib/opsApi") + createApiMock 拦截端点
---

<objective>
将三个 subdir 各自 ≥70%(D-07 独立 floor,ratchet 互不掩盖):(1)`components/CronSelector/` 316 stmts —— utils.ts 走真实 `@breejs/later` + `cron-validate` + `cron-parser` 算法向量直测(83 D-08 模式,边界字符串 `*/5 * * * * *` / `0 0 0 1 1 ?` / `0 0 9-17 * * *` 覆盖 6 段 + 范围 + 列表 + 问号),constants.ts 静态断言,CronSelector.tsx 组件交互(选择预设 / 自定义 / 输入验证)。**不**mock later / cron-validate(违反 Pitfall #6,失去真实 cron 解析断言)。(3) `components/captcha/` 154 stmts —— SliderCaptcha mock canvas getContext + mock verifySliderCaptcha 返回成功/失败 → onVerified/onError 触发(Pitfall #5:不模拟真实 PointerEvent 拖动过程,而是 mock API 返回断言),TextCaptcha 文本输入断言,CaptchaModal 打开 / 验证 / 关闭 + onSuccess 传 token 断言。(4) `components/operations/` 149 stmts —— WorkstationDeviceTable opsApi 包装型契约测试,createApiMock 拦截 workstation device 端点。三个 subdir 各自独立 floor bump(同 PR 三行);基线文档追加三个 ratchet 行。

Purpose: COMP-04 是 P1 中等量级三 subdir 组合(619 stmts 合计)——CronSelector 是纯算法 + antd(cron 库真实向量直测是关键,captcha 含 canvas / drag / PointerEvent 但 jsdom 下 rAF 节流跳过中间态故走 mock API 路径,operations 是 opsApi 包装型契约)。

Output: CronSelector 3 文件 + captcha 3 文件 + operations 1 文件 共 7 个测试文件;三个 subdir floor 各自 bump;基线文档三个 ratchet 行。
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
@xingran-react-frontend/src/components/CronSelector/
@xingran-react-frontend/src/components/captcha/
@xingran-react-frontend/src/components/operations/
@xingran-react-frontend/src/services/captcha.ts
@xingran-react-frontend/src/lib/opsApi.ts
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@xingran-react-frontend/src/test/utils/createApiMock.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: CronSelector utils + constants + CronSelector 组件测试(真实算法向量直测)</name>
  <files>
    xingran-react-frontend/src/components/CronSelector/__tests__/utils.test.ts
    xingran-react-frontend/src/components/CronSelector/__tests__/constants.test.ts
    xingran-react-frontend/src/components/CronSelector/__tests__/CronSelector.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/CronSelector/index.tsx / constants.ts / utils.ts / fields/*（如存在）
    - @breejs/later API（cron 解析）
    - cron-validate API（cron 字符串验证）
    - cron-parser API（cron 字符串解析）
  </read_first>
  <action>
    1. 创建 `__tests__/utils.test.ts` —— **真实算法向量直测**(83 D-08 模式,Pitfall #6):
       - expression ↔ config 往返测试:`"0 0 12 * * *"` / `"*/5 * * * * *"` / `"0 0 0 1 1 ?"` / `"0 0 9-17 * * *"` / `"30 0 0 1,15 * ?"`(覆盖 6 段 + 范围 + 列表 + 问号)
       - expect(cronConfigToExpression(expressionToCronConfig(expr))).toBe(expr)
       - validateCronExpression 合法 / 非法字符串边界断言
       - getNextRunTimes("0 0 12 * * *", 3) 真实 later 解析,断言长度 = 3 + next[0].getHours() = 12 + next[0].getMinutes() = 0
       - **不**mock @breejs/later / cron-validate(违反 Pitfall #6 失去真实解析断言)
    2. 创建 `__tests__/constants.test.ts` —— 静态常量完整性(D-12):
       - DEFAULT_CRON / FIELD_OPTIONS / PERIOD_PRESETS 常量值断言
       - assert 导出键数量 + 各 preset 的 cron 字符串格式合法
    3. 创建 `__tests__/CronSelector.test.tsx` —— 组件交互:
       - renderWithProviders(<CronSelector value="" onChange={onChange} />)
       - 预设列表渲染断言 + fireEvent.click 预设 → onChange 被调 + value 更新断言
       - fireEvent.click 自定义 + 自定义输入框 + fireEvent.change + onChange 断言
       - 输入非法表达式 + 错误提示渲染断言
       - fireEvent.click 字段配置(分钟/小时/日)→ 子组件状态变化
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/CronSelector/__tests__ 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - CronSelector 3 个测试通过,真实算法 + 静态常量 + 组件交互全覆盖
  </done>
  <acceptance_criteria>
    - utils.test.ts 真实调用 @breejs/later + cron-validate + cron-parser,不 mock
    - 边界字符串覆盖 6 段 + 范围 + 列表 + 问号 4 种
    - constants.test.ts 静态常量值断言 + 导出完整性
    - CronSelector.test.tsx 组件交互覆盖预设 / 自定义 / 非法输入提示
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: captcha + operations 测试(SliderCaptcha mock canvas + verify API / WorkstationDeviceTable opsApi 契约)</name>
  <files>
    xingran-react-frontend/src/components/captcha/__tests__/SliderCaptcha.test.tsx
    xingran-react-frontend/src/components/captcha/__tests__/TextCaptcha.test.tsx
    xingran-react-frontend/src/components/captcha/__tests__/CaptchaModal.test.tsx
    xingran-react-frontend/src/components/operations/__tests__/WorkstationDeviceTable.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx / TextCaptcha.tsx / CaptchaModal.tsx
    - xingran-react-frontend/src/services/captcha.ts（getCaptcha / verifySliderCaptcha / verifyTextCaptcha 接口）
    - xingran-react-frontend/src/components/operations/WorkstationDeviceTable.tsx / types.ts
    - xingran-react-frontend/src/lib/opsApi.ts
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    - xingran-react-frontend/src/test/utils/createApiMock.ts
  </read_first>
  <action>
    1. 创建 `captcha/__tests__/SliderCaptcha.test.tsx` —— mock canvas getContext + verify API(Pitfall #5):
       - HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue({ fillRect: vi.fn(), clearRect: vi.fn(), ...stubs })
       - vi.mock("@/services/captcha", ...) 提供 mock getCaptcha 返回 { backgroundImage, sliderImage, token }
       - renderWithProviders(<SliderCaptcha onVerified={onVerified} onError={onError} />) + 加载态断言
       - fireEvent.click 刷新 → getCaptcha 再次调用断言
       - 模拟 verify 成功:mock verifySliderCaptcha.mockResolvedValue({ success: true, token: "tk-1" }) → fireEvent 内部 onVerify → onVerified("tk-1") 被调断言
       - 模拟 verify 失败:mock verifySliderCaptcha.mockResolvedValue({ success: false }) → onError 被调断言
       - **不**模拟真实 PointerEvent 拖动过程(jsdom rAF 节流跳过中间态)
    2. 创建 `captcha/__tests__/TextCaptcha.test.tsx` —— 文本验证码输入:
       - renderWithProviders(<TextCaptcha />) + 输入框渲染断言
       - fireEvent.change input + onChange 回调断言
       - fireEvent.click 刷新 → 验证码图片 src 变化断言
       - fireEvent.click 验证 → onVerified/onError 触发断言
    3. 创建 `captcha/__tests__/CaptchaModal.test.tsx` —— Modal 打开 + 验证 + 关闭:
       - renderWithProviders(<CaptchaModal open={true} onSuccess={onSuccess} onCancel={onCancel} />)
       - 渲染 SliderCaptcha/TextCaptcha 断言(根据 props.type 切换)
       - fireEvent.click Modal 关闭 → onCancel 被调
       - fireEvent 内部验证成功 → onSuccess(token) 被调(token = mock 返回值)
    4. 创建 `operations/__tests__/WorkstationDeviceTable.test.tsx` —— opsApi 契约:
       - createApiMock("/ops/workstation/device/list") 拦截列表端点 + mockResolvedValue({ list: [...], total: 10 })
       - renderWithProviders(<WorkstationDeviceTable workstationId="ws-1" />) + 表格列渲染断言
       - 表格行渲染 + 数据填充断言
       - fireEvent.click 行操作按钮 → 端点调用断言
       - 分页 fireEvent.click → 端点带分页参数再次调用断言
       - createApiMock("/ops/workstation/device/export") 拦截导出端点 + fireEvent.click 导出 → 端点调用断言
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/captcha/__tests__ src/components/operations/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - captcha 3 文件 + operations 1 文件测试通过,verify API mock 路径 + opsApi 契约全覆盖
  </done>
  <acceptance_criteria>
    - SliderCaptcha 测试 mock canvas getContext + verify API,不模拟 PointerEvent 拖动(Pitfall #5)
    - TextCaptcha 测试输入 + 刷新 + 验证交互
    - CaptchaModal 测试 open + onSuccess(token) + onCancel
    - WorkstationDeviceTable 测试列表渲染 + 分页 + 导出契约(createApiMock)
    - captcha + operations 测试不依赖未 mock 的真实 API
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 全量 vitest 验证 + 三个 subdir floor bump + 基线文档 ratchet(D-07 三个 ratchet 行)</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 components/CronSelector / components/captcha / components/operations 三行 0.0）
    - .planning/frontend-coverage-baseline.md
    - 82-CONTEXT.md D-06/D-07
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量 + 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含三个 PASS: 行(实测 pct 均 ≥70%)
    3. 按 D-14 + D-07 三个 subdir 各自独立 bump .coverage-fe-floors 三行:
       - components/CronSelector: max(70.0, pct − 0.5)
       - components/captcha: max(70.0, pct − 0.5)
       - components/operations: max(70.0, pct − 0.5)
       - 三行各自保留一位小数(向下截断)
    4. 在 .planning/frontend-coverage-baseline.md 追加 **三个** ratchet 行(84-02b-CronSelector / 84-02b-captcha / 84-02b-operations,D-07 互不掩盖)
    5. 再跑 gate 确认三个 subdir 各自 PASS 且无 FAIL
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/(CronSelector|captcha|operations)'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - 三个 subdir floor 各自 bump;基线文档追加三个 ratchet 行;gate 三行 PASS
  </done>
  <acceptance_criteria>
    - npm run test:coverage exit 0,Tests ≥ 159 + 新增测试数
    - gate 输出三个 PASS: 行均 ≥70.0%
    - .coverage-fe-floors 三行各自更新(components/CronSelector / components/captcha / components/operations)
    - .planning/frontend-coverage-baseline.md 新增三个 84-02b ratchet 行(D-07 互不掩盖)
    - gate 总 FAIL 行数 = 0
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| CronSelector utils → @breejs/later / cron-validate / cron-parser | 真实调用,确定性字符串向量 |
| captcha SliderCaptcha → canvas | vi.mock HTMLCanvasElement.getContext 返回 stub |
| captcha → services/captcha | vi.mock 整个 services/captcha 模块 |
| operations → lib/opsApi | createApiMock 拦截端点 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-2b-01 | Tampering | CronSelector mock later 失去真实解析 | mitigate | Pitfall #6: 真实调用,边界字符串向量 |
| T-84-2b-02 | Denial of Service | SliderCaptcha canvas 在 jsdom 报错 | mitigate | Pitfall #5: HTMLCanvasElement.getContext stub |
| T-84-2b-03 | Tampering | PointerEvent 拖动 rAF 节流跳过 | mitigate | Pitfall #5: 不模拟拖动过程,改 mock verify API |
| T-84-2b-04 | Information Disclosure | captcha token 验证暴露真实 token | mitigate | mock 返回假 token,不写真实凭证 |
| T-84-2b-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests ≥ 159 + 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/(CronSelector|captcha|operations)'` 输出 3 行 PASS,各 pct ≥70%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
4. `git diff .coverage-fe-floors` 显示三个 subdir 行各自单向上调
5. `git diff .planning/frontend-coverage-baseline.md` 追加三个 84-02b ratchet 行
6. `grep -r 'renderWithProviders\|createApiMock' src/components/captcha/ src/components/operations/ | wc -l` ≥ 3(SC-6 Reuse)
</verification>

<success_criteria>
- components/CronSelector 316 stmts + components/captcha 154 stmts + components/operations 149 stmts 各自 ≥70%(COMP-04 满足,D-07 三个独立 floor)
- 全量 vitest 0 失败,159 存量 + 新增测试全 PASS(QUAL-01 不回归)
- CronSelector utils 真实算法向量直测,边界字符串覆盖 4 种语法
- captcha 测试 mock canvas getContext + verify API,不模拟 PointerEvent 拖动(Pitfall #5)
- operations 测试 createApiMock 拦截 opsApi 端点契约
- .coverage-fe-floors 三个 subdir 行各自 bump + 基线文档三个 ratchet 行追加(同 PR,D-07 互不掩盖)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-02b-cron-captcha-ops-SUMMARY.md` when done
</output>
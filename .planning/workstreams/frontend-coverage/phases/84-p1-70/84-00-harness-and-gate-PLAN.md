---
phase: 84-p1-70
plan: 00
type: execute
wave: 0
depends_on: []
files_modified:
  - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
  - xingran-react-frontend/src/test/utils/createApiMock.ts
  - xingran-react-frontend/src/test/setup.ts
  - .coverage-fe-floors
  - .github/scripts/check-frontend-coverage.sh
autonomous: true
requirements:
  - COMP-01
  - COMP-02
  - COMP-03
  - COMP-04
  - COMP-05
user_setup: []
must_haves:
  truths:
    - "[D-01/D-02/D-03] src/test/utils/ 存在 renderWithProviders.tsx 与 createApiMock.ts,store 按需注入 + 自动 reset,createApiMock 支持单端点 + mockApiBatch"
    - "[D-13] src/test/setup.ts 含 ResizeObserver 集中 polyfill(对齐 BulkWriteDrawer L27-36 inline 形态),matchMedia 保留"
    - "[D-04] .coverage-fe-floors 新增 9 个 components subdir 行 + design-system 行 + components 聚合行,初值 = 0"
    - "[D-05] .github/scripts/check-frontend-coverage.sh L219/L316/L381 三处镜像 components/<subdir> 二级聚合分支"
    - "[QUAL-01] BulkWriteDrawer.test.tsx 5 用例 plan 0 末尾验证仍 PASS(plan 0 不改其 Wrapper,仅 polyfill 沉淀后允许)"
  artifacts:
    - path: xingran-react-frontend/src/test/utils/renderWithProviders.tsx
      provides: Router + antd App + 按需 stores reset wrapper
    - path: xingran-react-frontend/src/test/utils/createApiMock.ts
      provides: 端点工厂 vi.fn() + mockApiBatch 批量注册
    - path: xingran-react-frontend/src/test/setup.ts
      provides: matchMedia + ResizeObserver polyfill 集中沉淀
    - path: .coverage-fe-floors
      provides: 9 components subdir 行 + design-system 行 + components 聚合行(初值 = 0)
    - path: .github/scripts/check-frontend-coverage.sh
      provides: L219/L316/L381 三处 components 二级聚合扩展
  key_links:
    - from: xingran-react-frontend/src/test/utils/renderWithProviders.tsx
      to: xingran-react-frontend/src/test/setup.ts
      via: setupFiles 自动加载 antd App context
    - from: xingran-react-frontend/src/test/utils/createApiMock.ts
      to: xingran-react-frontend/src/lib/api.ts
      via: vi.mock('@/lib/api') 模块级拦截
    - from: .github/scripts/check-frontend-coverage.sh
      to: .coverage-fe-floors
      via: 读取新增的 components/<subdir> 行做 dir-registered check
---

<objective>
落地 Phase 84 的 wave 0 基建:创建 `renderWithProviders` + `createApiMock` 两件 harness(D-01/D-02/D-03);把 BulkWriteDrawer L27-36 的 inline ResizeObserver 经验前移到 `src/test/setup.ts` 集中沉淀(D-13);扩展 `.coverage-fe-floors` 新增 9 个 components subdir 行 + 1 个 design-system 行 + 1 个 components 聚合行(D-04);在 `check-frontend-coverage.sh` L219/L316/L381 三处镜像 `pages/<subdir>` 模式扩展 `components/<subdir>` 二级聚合分支(D-05);不触碰 BulkWriteDrawer.test.tsx 本身,仅验证 5 用例在 setup.ts polyfill 沉淀后仍 PASS(D-13 不破坏既有测试)。

Purpose: wave 0 是 P1 五个组件组测试的"基础设施基线"——harness + polyfill + gate 扩展 + floors 新行四件事一次性就位,wave 1/2/3 各 plan 只需聚焦"写测试 + bump floor"两件,不再重复造轮子。

Output: harness 两件 + setup.ts 增补 + floors 新行 11 行 + gate 脚本三处扩展 + BulkWriteDrawer 5 用例不回归证据。
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
@.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-CONTEXT.md
@xingran-react-frontend/src/test/setup.ts
@xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx
@.coverage-fe-floors
@.github/scripts/check-frontend-coverage.sh
</context>

<tasks>

<task type="auto">
  <name>Task 1: 创建 renderWithProviders + createApiMock harness</name>
  <files>
    xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    xingran-react-frontend/src/test/utils/createApiMock.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/test/setup.ts
    - xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx
    - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx
    - xingran-react-frontend/src/lib/api.ts
    - xingran-react-frontend/vitest.config.ts
  </read_first>
  <action>
    1. 创建 `xingran-react-frontend/src/test/utils/renderWithProviders.tsx`(D-02 锁定形态):
       - 默认包裹 `<MemoryRouter initialEntries={["/"]}>` + antd `<App>`(提供 message/modal/notification context)
       - options 扩展 RenderOptions: `route?: string`(切换 initialEntries)/ `resetStores?: Array<() => void>`(调用方传 Zustand store 的 setState 函数)/ `queryClient?: unknown`(按需,widgets 体系若需 `@tanstack/react-query` 时用)
       - 导出 `RenderWithProvidersOptions` 类型 + `renderWithProviders(ui, options)` 函数
       - 不在 setupFiles 加载,而是按需 import(避免无关测试加载 router/antd context)
    2. 创建 `xingran-react-frontend/src/test/utils/createApiMock.ts`(D-03 锁定形态):
       - `createApiMock(endpoint)` 返回 `ApiMockHandle { post: Mock, get: Mock, put: Mock, del: Mock, endpoint: Mock }`
       - 模块级 `vi.mock("@/lib/api", ...)` 拦截:当 url 命中 endpoint 时路由到 endpointSpy,否则回退到 post/get/put/del 通用 spy
       - 同时导出 `mockApiBatch(handlers: Array<{endpoint, response?}>)` 批量注册多个端点(返回 Record<string, ApiMockHandle>)
       - 不引入 MSW,零新依赖纪律
    3. 文件落地后,跑 `cd xingran-react-frontend && npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` 验证既有 5 用例仍 PASS(plan 0 不改 BulkWriteDrawer 自身)
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && ls src/test/utils/renderWithProviders.tsx src/test/utils/createApiMock.ts && npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - harness 两件落地,既有 5 个 BulkWriteDrawer 用例不回归
  </done>
  <acceptance_criteria>
    - xingran-react-frontend/src/test/utils/ 下存在 renderWithProviders.tsx 与 createApiMock.ts
    - renderWithProviders 默认 <MemoryRouter> + <App> 包裹,options 至少含 route/resetStores
    - createApiMock 返回 ApiMockHandle 含 endpoint spy,且模块级 vi.mock("@/lib/api") 生效
    - mockApiBatch 接收 handlers 数组批量返回 handles map
    - BulkWriteDrawer.test.tsx 5 用例跑过后 Tests 计数与原 5 一致(plan 0 不强制改 Wrapper)
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: setup.ts 集中沉淀 ResizeObserver polyfill(D-13)</name>
  <files>
    xingran-react-frontend/src/test/setup.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/test/setup.ts
    - xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx（L27-36 inline ResizeObserverStub 形态）
  </read_first>
  <action>
    1. 在 `src/test/setup.ts` 现有 matchMedia polyfill 之后,追加 ResizeObserver 集中 polyfill(对齐 BulkWriteDrawer L27-36 inline 形态):
       - 检测 `typeof globalThis.ResizeObserver === "undefined"`,若未定义则 stub:
         - class ResizeObserverStub { observe(){} unobserve(){} disconnect(){} }
         - globalThis.ResizeObserver = ResizeObserverStub
    2. **不**在 plan 0 移除 BulkWriteDrawer.test.tsx 的 inline ResizeObserverStub(避免 plan 0 跨文件改动;Wave 1 plan 1a / 1b 起各 plan 顺手迁移)
    3. 追加 polyfill 后跑 `cd xingran-react-frontend && npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx src/components/reconciliation/__tests__/HealthCard.test.tsx` 验证 10 个既有用例不回归
    4. 不在 plan 0 加 IntersectionObserver / PointerEvent / canvas getContext(按 D-13 "按需沉淀" 纪律,Wave 1/2/3 失败时再补)
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && grep -c 'ResizeObserver' src/test/setup.ts && npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx src/components/reconciliation/__tests__/HealthCard.test.tsx 2>&1 | tail -10
    </automated>
  </verify>
  <done>
    - setup.ts 含 matchMedia + ResizeObserver 双 polyfill;BulkWriteDrawer / HealthCard 10 用例不回归
  </done>
  <acceptance_criteria>
    - src/test/setup.ts 含 ResizeObserver polyfill 块(grep -c ResizeObserver >= 2 含注释/类定义)
    - BulkWriteDrawer.test.tsx 5 用例 + HealthCard.test.tsx 5 用例 = 10 全 PASS
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 扩展 .coverage-fe-floors 新增 9 subdir 行 + design-system 行 + components 聚合行(D-04/D-06/D-15)</name>
  <files>
    .coverage-fe-floors
  </files>
  <read_first>
    - .coverage-fe-floors（当前 28 行,components 4.9 / design-system 15.0）
    - xingran-react-frontend/src/components/（确认 9 个目标 subdir 存在:shared / dashboard / layout / CronSelector / captcha / operations / network / reconciliation / design-system 不在 components 下,但走顶层行）
  </read_first>
  <action>
    1. 按字典序在 `.coverage-fe-floors` 现有 28 行后追加 11 行(保留原 28 行不动):
       - components/shared<TAB>0.0
       - components/dashboard<TAB>0.0
       - components/layout<TAB>0.0
       - components/CronSelector<TAB>0.0
       - components/captcha<TAB>0.0
       - components/operations<TAB>0.0
       - components/network<TAB>0.0
       - components/reconciliation<TAB>0.0
       - design-system<TAB>0.0
       - components（聚合行,沿用现有 components 4.9 上方追加一行 0.0 还是 bump 到 69.5?——D-15 wave 3 末尾 bump 至 69.5;plan 0 阶段先维持 4.9 不动,聚合行已是 4.9 故不需新增行）
       - 注:D-06 锁 design-system 走顶层行(与 hooks/store/services 同级),不走 components 二级;9 个 components subdir 各自独立 floor
    2. **不**新增 components 聚合行新值——现有 components 4.9 行保留,D-15 在 wave 3 末尾 bump 至 69.5
    3. 不修改现有 28 行,只追加 10 行新行(9 components subdir + design-system)
    4. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors --init` 验证新行生成正确(应输出 11 个新行 0.0,既存 28 行不变)
    5. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含 components/shared 等 10 个新 PASS: 行(实测覆盖率 ≥0.0)
  </action>
  <verify>
    <automated>
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors --init 2>&1 | grep -E '^components/|^design-system' | sort
    </automated>
  </verify>
  <done>
    - .coverage-fe-floors 新增 10 个 0.0 行;--init 与 gate 均能正确输出新行
  </done>
  <acceptance_criteria>
    - .coverage-fe-floors 含 components/shared、components/dashboard、components/layout、components/CronSelector、components/captcha、components/operations、components/network、components/reconciliation、design-system 共 9 个新行
    - 各新行初值 = 0.0,符合 D-04 / D-06 增量行约定
    - gate 脚本能识别新行,输出一一对应的 PASS: 行
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 4: gate 脚本 L219/L316/L381 三处扩展 components/<subdir> 二级聚合(D-05)</name>
  <files>
    .github/scripts/check-frontend-coverage.sh
  </files>
  <read_first>
    - .github/scripts/check-frontend-coverage.sh L215-L225 / L309-L320 / L374-L390(三处 awk 段)
    - 82-CONTEXT.md D-05 描述的 pages 二级拆分 awk 镜像
  </read_first>
  <action>
    1. 修改 `check-frontend-coverage.sh` 的三处 awk 路径聚合逻辑:
       - L219 行(在 `else if (seg[1] == "pages") key = "pages/" seg[2]` 之后)追加 `else if (seg[1] == "components") key = "components/" seg[2]`
       - L316 行(同上模式)追加
       - L381 行(同上模式)追加
       - `else key = seg[1]` 兜底保持不动——design-system 走顶层 seg[1] = "design-system"(D-06)
    2. 三处必须**完全镜像**;改后用 `grep -n 'seg\[1\] == "components"' .github/scripts/check-frontend-coverage.sh` 应输出 3 行(避免 Pitfall #1 漏改)
    3. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors --init | grep -E '^components/|^design-system'` 验证 --init 模式生成正确
    4. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/|^PASS: design-system'` 验证 gate 模式聚合到新行
    5. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS:' | wc -l` 验证总 PASS 行数 = 28 既有 + 10 新增 = 38(若其中有 FAIL 行,需排查 floor 与实测漂移)
  </action>
  <verify>
    <automated>
      grep -n 'seg\[1\] == "components"' .github/scripts/check-frontend-coverage.sh | wc -l  # 应输出 3
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors --init 2>&1 | grep -E '^components/|^design-system' | wc -l  # 应输出 10
    </automated>
  </verify>
  <done>
    - gate 脚本 L219/L316/L381 三处镜像扩展;--init 与 gate 均正确处理 components/<subdir> 与 design-system
  </done>
  <acceptance_criteria>
    - grep 验证三处扩展均到位(grep -c 'seg\[1\] == "components"' == 3)
    - gate 脚本 --init 输出含 10 个新行(9 components subdir + design-system)
    - gate 脚本跑覆盖率 gate 时 PASS: 行数 = 38(28 既有 + 10 新增,可能因实测覆盖率低于 floor 有 FAIL,但 floor 0.0 必 PASS)
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| harness → 测试代码 | harness 位于 src/test/utils/,被 coverage.exclude 排除(已配置),helper 代码不计入分母 |
| setup.ts polyfill → jsdom | 集中 polyfill 全局可见,避免每文件 inline;只 stub ResizeObserver 缺失场景 |
| gate 脚本 → .coverage-fe-floors | 读取数据文件,awk 三处镜像保持单一真相源 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-00-01 | Tampering | harness 文件被错放 coverage.include | mitigate | vitest.config.ts coverage.exclude 已含 `src/test/`,harness 必落 src/test/utils/ |
| T-84-00-02 | Denial of Service | setup.ts ResizeObserverStub 影响真实浏览器测试 | mitigate | setupFiles 仅 vitest 加载,production build 不走 setup.ts |
| T-84-00-03 | Information Disclosure | gate 脚本 awk 注入 | accept | bash awk 不解析外部不可信输入,profile 数据由 vitest 生成 |
| T-84-00-04 | Tampering | gate 三处 awk 漏改导致 subdir 不聚合 | mitigate | Pitfall #1 grep 验证 + --init + gate 双跑,3 处镜像必严格一致 |
| T-84-00-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包;无安装步骤 |
</threat_model>

<verification>
1. `ls xingran-react-frontend/src/test/utils/` 输出 renderWithProviders.tsx + createApiMock.ts
2. `grep -c ResizeObserver xingran-react-frontend/src/test/setup.ts` ≥ 2(含类定义 + 使用)
3. `cat .coverage-fe-floors | grep -E '^components/|^design-system' | wc -l` = 10
4. `grep -c 'seg\[1\] == "components"' .github/scripts/check-frontend-coverage.sh` = 3
5. `cd xingran-react-frontend && npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx src/components/reconciliation/__tests__/HealthCard.test.tsx` 全 PASS,Tests 计数 = 10
6. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS:' | wc -l` ≥ 38(28 既有 + 10 新增)
7. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
8. 全量 `npm run test:coverage` exit 0,QUAL-01 159 存量测试不回归(可能因 harness 不影响覆盖率,只新增 setup.ts polyfill)
</verification>

<success_criteria>
- src/test/utils/ 下存在 renderWithProviders.tsx 与 createApiMock.ts,store reset 按需注入(D-02/D-03 锁定形态)
- src/test/setup.ts 含 matchMedia + ResizeObserver 双 polyfill(D-13 集中沉淀)
- .coverage-fe-floors 新增 9 components subdir 行 + design-system 行,各初值 = 0.0(D-04/D-06 增量行)
- .github/scripts/check-frontend-coverage.sh L219/L316/L381 三处镜像 components/<subdir> 二级聚合(D-05)
- BulkWriteDrawer.test.tsx 5 用例 + HealthCard.test.tsx 5 用例 plan 0 末尾验证仍 PASS(QUAL-01 不回归)
- gate 脚本输出 38+ PASS,无 FAIL
- 全量 vitest 0 失败(159 存量测试不回归)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-00-harness-and-gate-SUMMARY.md` when done
</output>
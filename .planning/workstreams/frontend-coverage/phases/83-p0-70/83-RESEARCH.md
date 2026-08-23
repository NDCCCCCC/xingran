# Phase 83: P0 基建层全清 ≥70% - Research

**Researched:** 2026-08-24
**Domain:** 前端 Vitest 单元/组件测试覆盖率补齐（React 19 + TypeScript 5.9 + Zustand 5 + Ant Design 6 + jsdom）
**Confidence:** HIGH（基于实测 coverage json、已落库的 gate 脚本、19 个既有测试文件的模式验证）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** CR-01 修复作为 Phase 83 的**首个 plan**（wave 1 首位），走完整 plan 流程——与 harness 落地同 phase，时序上保证后续 harness PR 不踩 gate 红。
- **D-02:** 修复范围为**全部四项**：CR-01（diff pathspec 镜像 vitest coverage.exclude——白名单/.d.ts/src-test 不进 diff 分母）+ WR-01（json 缺失软跳过改 fail-closed `exit 2`，对齐后端孪生脚本 `check-diff-coverage.sh` 同位置行为）+ WR-02（白名单漂移检测改锚定匹配，`cad-editor-helpers.ts` 类合法新文件不再误判）+ WR-03（floors 数值结构校验，拒绝 `3..8`/`1.2.3` 类手误）。
- **D-03:** 修复验证 = 本地空树合成基线复现修复前后行为（82-03 已证可靠的技术）+ **试验 PR**（含 `src/test/` + `.d.ts` + 白名单文件变更）验证真实 CI 绿；试验 PR 验完关闭**不 merge**。顺带首次真实触发 GOV-04 的 join+阈值主路径（补 82-REVIEW IN-06 缺口——此前该路径只走过 docs-only 空 diff 分支）。
- **D-04:** harness **P0 尾声定稿**——P0 各 plan（纯逻辑测试为主）期间观察实际重复样板，harness 在 P0 尾声（store/组件接壤处）按实证需求定稿；不做前置全家桶，也不推迟到 84。
- **D-05:** `renderWithProviders` **按需注入**——默认 Router + AntD ConfigProvider；Zustand stores 走参数按需注入并自动 reset（对齐 Zustand 官方测试模式 resetBetweenTests），避免全量注入的测试间状态泄漏。
- **D-06:** `mockApi` 采用**端点工厂**形态——`createApiMock(endpoint, response)` 生成 post/get spy，按端点注册成功/失败/延迟响应；零新依赖，贴合现有 wrapped post/get 项目约定（CLAUDE.md 前端 API 规范）。不引入 MSW。
- **D-07:** api.ts 加密客户端**双轨**——业务层（hooks/store/组件）测试用 `vi.mock('@/lib/api')` 整模块 mock；api.ts 自身在 INFRA-01 内用真实链路 + mock 加密层（固定密钥/向量）直测加密编排、401 刷新、重试分支。
- **D-08:** sm2/sm4/encoding 国密工具用**真实算法 + 确定性测试向量**直测——加解密往返、篡改密文报错、密钥长度边界，固定密文样本验证；零 mock。
- **D-09:** TokenManager 定时刷新循环用 `vi.useFakeTimers()`——`advanceTimersByTime` 直达过期点；并发刷新队列与 401 重试同样 fake timers + mock api；不使用真实短 TTL（避免 flaky）。
- **D-10:** plan 按**依赖层切分**——plan0=CR-01 修复；随后 wave：utils+lib（底层先清，两 plan 并行）→ hooks+store → services/router/constants/types 收尾 + harness 定稿（P0 尾声，D-04）。依赖方向 utils ← lib ← hooks/store，底层未清不测上层。
- **D-11:** per-dir floor **逐 plan bump**——每个 plan 完成即 bump 对应目录 floor 至实测−0.5pp 并同 PR 追加基线文档（延续 82 的 D-06/D-07 纪律：ratchet 是纯数据变更 + 文档同 commit）。
- **D-12:** **plan 级验收**——每个 plan 的 verify 含 `npm run test:coverage` + gate 脚本目录断言 + 159 存量测试不回归（QUAL-01 基线）；phase 级 verify 只做汇总。

### Claude's Discretion
- 具体测试文件组织（同目录 `*.test.ts` 放置、describe/it 分组粒度）——按现有 19 个测试文件的既有模式。
- CR-01 pathspec 镜像的实现方式（硬编码镜像列表 vs 脚本内联注释指回 vitest.config.ts 真相源）——由 researcher/planner 定，保持单一真相源原则即可。
- 各 plan 内文件的测试优先级排序（先高扇出后长尾）。

### Deferred Ideas (OUT OF SCOPE)
- MSW 网络层 mock —— D-06 未采纳（零新依赖优先）；若 P1/P2 出现网络层真实拦截需求再评估。
- CI 缓存/分片优化 —— 82 遗留 deferred，本 phase 不动（41s Test (coverage) 余量充足）。
- harness 全家桶（fireEvent helpers / 表单填充 / IntersectionObserver 等 jsdom 补丁）—— D-04 按 P1/P2 实际需求渐进增补，不前置。
- E2E 测试层、视觉回归 —— REQUIREMENTS v2 候选，本里程碑外。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | lib 通信层 ≥70%（api.ts 加密客户端 / opsApi / networkApi / menuApi / profileApi / 列配置 api 等 20 文件 1042 stmts） | 实测 lib 当前 114/1042=10.94%；api.ts（213 stmts）为最大单文件，需双轨直测；其余 API 模块以 wrapper 契约测试覆盖 |
| INFRA-02 | utils 层 ≥70%（sm2 / sm4 / encoding 国密、token 三件套、dualLevelCache、errorHandler、authHelpers、deptUtils、datetime 等 23 文件 950 stmts） | 实测 utils 当前 78/950=8.21%；dualLevelCache（127 stmts）、geocodingCache（112 stmts）为最大；国密三件套用真实向量直测 |
| INFRA-03 | hooks 层 ≥70%（27 文件 1050 stmts，含 usePagination / useServerSort / usePersistedState 续推） | 实测 hooks 当前 85/1050=8.10%；useTableManager（123 stmts）、useColumnConfig（122 stmts）、useWidgetData（92 stmts）为前三；usePagination/useServerSort/usePersistedState 已有部分覆盖，模式可直接复用 |
| INFRA-04 | store 层 ≥70%（auth / menu / settings / tabs / dashboard / layout / notice / theme 9 文件 589 stmts） | 实测 store 当前 28/589=4.75%；tabsStore（145 stmts）、dashboardStore（137 stmts）最大；Zustand 官方 resetBetweenTests 模式 + 按需注入 |
| INFRA-05 | services + router + constants + types 收尾 ≥70%（~626 stmts） | 实测 services 238 stmts、router 272 stmts、constants 84 stmts、types 32 stmts，合计 626；router 当前 0% 需重点设计 |
| QUAL-03 | 测试公共 harness 沉淀（mock api / antd message / router wrapper 等高频样板），供 P1/P2 复用提效 | `src/test/setup.ts` 已提供 jsdom/polyfill 基础； harness 落点 `src/test/utils/` 为最优（该目录被 coverage.exclude 排除，helper 不计入分母） |
</phase_requirements>

## Summary

本 phase 的目标是把前端基建层（lib / utils / hooks / store / services / router / constants / types）从当前全量口径 3.85% 的低位拉到每个目录 statements 覆盖率 ≥70%，并在此过程中沉淀出 P1/P2 可复用的测试 harness。实测显示 P0 层当前共 4,278 stmts（含 `(src root)` 13 stmts 与 `api` 8 stmts），其中 lib / utils / hooks / store 四块合计 3,631 stmts，是主要工作量；services / router / constants / types 合计 626 stmts，量级较小但 router 当前 0% 且依赖 React Router，需要专门设计。

**关键发现：CR-01 / WR-01 / WR-02 / WR-03 四项 gate 修复已经落库。** 通过 `git log --oneline` 与脚本源码核对，四项修复分别以提交 `60f712c`、`27f275e`、`94d3a16`、`aa3bf0c` 存在于 main 分支 HEAD（`aa3bf0c`）。当前 `check-frontend-diff-coverage.sh` 的 pathspec 已完整镜像 `vitest.config.ts` 的 `coverage.exclude`（cad-editor / cad-elements / `**/*.d.ts` / src/test / *.test.* / __tests__），diff 脚本缺 profile 改为 `exit 2`，漂移检测已锚定路径前缀，floors 数值校验已用结构化正则。因此 Phase 83 的 "首个 plan" 应从 "重新实现修复" 调整为 **验证已落库修复 + 清理 ci.yml 注释措辞（82-REVIEW-FIX.md 备注）+ 发起 D-03 要求的试验 PR**。

**主要技术策略：**
- **分层推进、底层优先**：utils 与 lib 可并行（测试时通过 `vi.mock('@/lib/api')` 解耦），hooks/store 依赖底层，放在第二波；services/router/constants/types + harness 定稿收尾。
- **双轨 mock**：业务层测试统一 `vi.mock('@/lib/api')`；api.ts 自身用真实 axios + mock adapter / 固定 SM2/SM4 密钥直测加密编排、401 刷新队列、400 解密失败重放。
- **国密直测**：sm2 / sm4 / encoding 不 mock 算法，使用确定性密钥向量做往返、篡改、边界测试。
- **Zustand 测试**：按需传入 store 并 `resetBetweenTests`，避免全量 Provider 包裹导致状态泄漏。
- **harness 落点**：`src/test/utils/`（已被 coverage.exclude 排除），三件套为 `renderWithProviders` / `createApiMock` / `mockAntdMessage`。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| HTTP 请求加密/Token 刷新 | Frontend client lib (`src/lib/api.ts`) | — | 拦截器、SM2+SM4 编排、401 队列均在浏览器端 axios 实例内完成 |
| 国密算法封装 | Frontend utils (`src/utils/sm2.ts`, `sm4.ts`, `encoding.ts`) | — | 纯浏览器端计算，无服务端依赖 |
| Token 生命周期管理 | Frontend utils (`src/utils/token/`) | `src/store/authStore.ts` | TokenManager 是独立类，authStore 持有单例并驱动登录/初始化 |
| 表格/分页/排序状态 | Frontend hooks (`src/hooks/use*`) | `src/constants/storage.ts` | sessionStorage 持久化逻辑在 hooks 内落地 |
| 全局 UI 状态 | Frontend store (`src/store/*`) | — | Zustand 负责 auth/menu/tabs/settings/theme/notice/dashboard/layout |
| 路由配置与守卫 | Frontend router (`src/router/*`) | — | React Router v7 + 动态路由加载 |
| 静态常量/类型 | Frontend constants/types | — | 无运行时行为，测试以静态断言 + 导入触发 coverage 为主 |

## Standard Stack

### Core（已存在，无需新增）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| vitest | 4.1.10（lock） | 测试运行器 | 项目已锁定，Vitest 4 全量口径由 `coverage.include` 控制 [VERIFIED: package-lock + 本地 `npm run test:coverage`] |
| @vitest/coverage-v8 | 4.1.10（lock） | v8 coverage provider | 与 vitest 同版本，生成 `coverage-final.json` [VERIFIED: package-lock] |
| @testing-library/react | 16.3.2 | 组件/hook 渲染 | 现有 19 个测试文件全部使用 [VERIFIED: package.json + 既有测试] |
| @testing-library/jest-dom | 6.9.1 | DOM 断言 | 已在 `src/test/setup.ts` 引入 [VERIFIED: setup.ts] |
| jsdom | 27.4.0 | DOM 环境 | `vitest.config.ts` 的 `environment: "jsdom"` [VERIFIED: vitest.config.ts] |
| sm-crypto | 0.5.5 | SM2/SM4 国密算法 | 国密工具依赖，测试中真实调用 [VERIFIED: package.json] |
| zustand | 5.0.15 | 状态管理 | store 层测试对象 [VERIFIED: package.json] |
| react-router-dom | 7.18.2 | 路由 | hooks 依赖 `MemoryRouter` 测试 [VERIFIED: package.json + 既有测试] |

### 注意：版本声明失同步隐患
`package.json` 中 `vitest: ^4.0.18` 与 `@vitest/coverage-v8: ^4.1.10`、`@vitest/ui: ^4.0.18` 声明基线不同。当前 lockfile 把三者锁到 `4.1.10`，`npm ci` 下安全；但未来 `npm update` 可能使 provider 与核心版本失配。建议 Phase 83 顺带将三者声明统一为 `^4.1.10`（IN-06，低风险治理）。

## Package Legitimacy Audit

本 phase **不引入新的外部包**，全部复用现有 devDependencies。因此无需执行 slopcheck 注册表审计。唯一建议是统一 vitest 生态包的 caret range，但属于治理性版本调整而非新增依赖。

## Architecture Patterns

### System Architecture Diagram（测试数据流）

```
Vitest runner
    ↓
src/test/setup.ts  (jsdom + matchMedia polyfill + harness 可选挂载)
    ↓
测试文件 (*.test.ts / *.test.tsx)
    ├── utils 测试 ──→ 真实 sm-crypto / 模拟 @/lib/api / localStorage/sessionStorage mock
    ├── lib 测试   ──→ vi.mock('@/lib/api') 或 真实 axios + mock adapter
    ├── hooks 测试 ──→ renderHook + MemoryRouter + 按需 Zustand store reset
    ├── store 测试 ──→ 直接调用 store actions + vi.useFakeTimers
    └── router/constants/types ──→ 静态断言 / render / 路由契约
    ↓
coverage-final.json (v8 provider)
    ↓
check-frontend-coverage.sh / check-frontend-diff-coverage.sh
    ↓
CI gate (GLOBAL + per-dir floors + diff coverage ≥80%)
```

### Recommended Project Structure

```
xingran-react-frontend/src/
├── lib/                     # INFRA-01 测试对象
│   ├── api.test.ts          # api.ts 双轨直测（拦截器/加密/401/400 重放）
│   ├── opsApi.test.ts       # CRUD 工厂 + excel 下载契约
│   ├── api/
│   │   └── networkApi.test.ts  # 已存在，模式参考
│   └── ...
├── utils/                   # INFRA-02 测试对象
│   ├── sm2.test.ts          # 已存在（公钥缓存竞态）
│   ├── sm4.test.ts          # 新增：真实向量直测
│   ├── encoding.test.ts     # 新增：hex/base64/bytes 往返
│   ├── token/
│   │   ├── TokenManager.test.ts
│   │   └── SecureTokenStorageImpl.test.ts
│   ├── dualLevelCache.test.ts
│   ├── errorHandler.test.ts
│   └── ...
├── hooks/                   # INFRA-03 测试对象
│   ├── usePagination.test.tsx    # 已存在
│   ├── useServerSort.test.tsx    # 已存在
│   ├── usePersistedState.test.ts # 已存在
│   ├── useTableManager.test.tsx
│   ├── useColumnConfig.test.tsx
│   └── ...
├── store/                   # INFRA-04 测试对象
│   ├── authStore.test.ts
│   ├── tabsStore.test.ts
│   ├── menuStore.test.ts
│   └── ...
├── services/                # INFRA-05 测试对象
│   ├── encryptionConfig.test.ts
│   └── cache/
├── router/                  # INFRA-05 测试对象
│   ├── routeConfigManager.test.ts
│   └── routeGenerator.test.ts
├── constants/               # INFRA-05 测试对象
│   ├── status.test.ts       # 已存在
│   └── storage.test.ts
├── types/                   # INFRA-05 测试对象
│   └── *.type-guard.test.ts # 静态导入 + 类型工具断言
└── test/                    # QUAL-03 harness
    ├── setup.ts             # 已存在
    └── utils/               # 新增，被 coverage.exclude 排除
        ├── renderWithProviders.tsx
        ├── createApiMock.ts
        └── mockAntdMessage.ts
```

### Pattern 1: 业务层 API Mock（端点工厂）
**What:** 用 `vi.mock('@/lib/api')` 替换整个模块，按端点注册响应。
**When to use:** 测试 hooks / store / utils 中调用 `post/get/put/del` 的分支。
**Example（基于现有 `src/lib/api/__tests__/networkApi.test.ts`）：**
```typescript
// Source: 既有测试 src/lib/api/__tests__/networkApi.test.ts
const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));
```

### Pattern 2: api.ts 自身直测（双轨）
**What:** 不 mock `api.ts` 模块，而是 mock 其底层依赖：axios（用 `axios-mock-adapter` 或 `vi.spyOn(axios, 'create')`）、sm2 / sm4 / TokenManager。
**When to use:** 仅 INFRA-01 中测试 api.ts 的拦截器、加密编排、401 刷新队列、400 解密失败重放。
**Example 思路：**
```typescript
// Source: researcher 设计，需 planner 在 PLAN 中细化
vi.mock("@/utils/sm2", () => ({
  fetchPublicKey: vi.fn().mockResolvedValue("fixed-public-key-hex"),
  clearPublicKeyCache: vi.fn(),
}));
vi.mock("@/utils/sm4", () => ({
  generateSM4Key: vi.fn().mockReturnValue("fixed-key-hex"),
  generateIV: vi.fn().mockReturnValue("fixed-iv-hex"),
  encryptRequestBody: vi.fn().mockResolvedValue("encrypted-hex"),
  hexToBase64: vi.fn().mockReturnValue("base64"),
  // ...
}));
```

### Pattern 3: Zustand Store 按需注入 + Reset
**What:** 每个 store 测试在 `beforeEach` 调用 `useXxxStore.setState(initialState)`，不传 Provider。
**When to use:** 所有 store 测试。
**Example：**
```typescript
// Source: Zustand 官方测试模式 + 项目 store 结构
import { useAuthStore } from "@/store/authStore";

beforeEach(() => {
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    loading: false,
    menusLoaded: false,
    initialized: false,
  });
});
```

### Pattern 4: renderWithProviders
**What:** 默认包裹 `MemoryRouter` + AntD `ConfigProvider`；通过参数可选注入特定 store 初始状态。
**When to use:** 需要挂载组件或 `renderHook` 且组件依赖路由/AntD context 时。
**Example：**
```typescript
// Source: researcher 设计，基于 D-05
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ConfigProvider } from "antd";

export function renderWithProviders(ui, { route = "/", stores = {} } = {}) {
  // stores 参数用于后续按需预置；P0 阶段可先用空对象占位
  return render(
    <MemoryRouter initialEntries={[route]}>
      <ConfigProvider>{ui}</ConfigProvider>
    </MemoryRouter>
  );
}
```

### Anti-Patterns to Avoid
- **全量 Provider 注入所有 stores：** 导致测试间状态泄漏，违反 D-05。
- **在业务层测试中真实调用 axios：** 引入网络不确定性和循环依赖风险；业务层应 mock `@/lib/api`。
- **mock `sm-crypto` 算法：** 国密工具测试应真实调用（D-08），否则无法验证密钥长度边界和篡改检测。
- **使用真实短 TTL 测试 TokenManager：** 会导致 flaky；统一使用 `vi.useFakeTimers()`（D-09）。
- **把 harness 放在被 coverage 统计的目录：** harness 文件应放入 `src/test/utils/`（已被 exclude），否则 helper 代码会拉低覆盖率。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API 端点 mock 工厂 | 每个测试手写 `vi.mock('@/lib/api')` | `createApiMock(endpoint, response)` | 减少样板；统一成功/失败/延迟响应语义；P1/P2 数千个组件测试直接复用 |
| 路由 + AntD 测试包装 | 每个测试手写 `<MemoryRouter><ConfigProvider>` | `renderWithProviders` | 避免重复；集中处理 jsdom 下 antd 需要的 matchMedia polyfill |
| antd message/Modal mock | 每个测试 spy `getAppMessage` 或 `App.useApp` | `mockAntdMessage` helper | api.ts / errorHandler / hooks 中大量调用 message，统一 mock 避免未捕获 |
| Zustand store reset | 手动清 localStorage + 重新 mount | `useXxxStore.setState(initialState)` | Zustand 官方模式，速度快且无 DOM 副作用 |
| Token 刷新定时 | `setTimeout` 真实等待 | `vi.useFakeTimers()` | 控制精确时间点，避免测试超时或 flake |
| coverage 聚合与 gate | 手写脚本统计覆盖率 | `check-frontend-coverage.sh` + `.coverage-fe-floors` | 已由 Phase 82 落库，per-dir floor 和 ratchet 纪律已验证 |

## Runtime State Inventory

本 phase 为 greenfield 测试补齐（非 rename/refactor），无需数据迁移或 OS 注册表变更。唯一需要关注的是 **Zustand persist 存储**：
- `auth-storage`、`settings-storage`、`layout-storage`、`tabs-storage`、`menu-storage` 等使用 localStorage/sessionStorage。
- 测试中通过 `useXxxStore.setState(...)` reset 即可，不需要清理真实浏览器存储（jsdom 的 storage 在每个测试进程内隔离）。

**Nothing found in category:** Stored data / Live service config / OS-registered state / Secrets / Build artifacts — 不适用，明确无运行时状态变更。

## Common Pitfalls

### Pitfall 1: `vi.mock` 提升导致模块级变量未初始化
**What goes wrong:** 在 `vi.mock` 之前定义的变量被提升后访问不到，出现 `Cannot access 'xxx' before initialization`。
**Why it happens:** Vitest 的 `vi.mock` 会被提升到文件顶部。
**How to avoid:** 使用 `vi.hoisted(() => vi.fn())` 包装 mock 函数；或在 `vi.mock` 工厂函数内部直接返回箭头函数。
**Warning signs:** 运行测试时报 `ReferenceError` 且堆栈指向 mock 声明处。

### Pitfall 2: `renderHook` 的 wrapper 依赖不稳定对象导致无限循环
**What goes wrong:** `useEffect` 依赖数组中包含每次 render 都重新创建的对象/数组，触发反复请求。
**Why it happens:** 违反 CLAUDE.md "useEffect Dependencies" 规则；测试中 props 或 wrapper 配置对象未 memoize。
**How to avoid:** 在测试中用 `useMemo` 包裹传给 hook 的复杂参数，或用原始值做依赖。
**Warning signs:** 同一测试运行时间异常长、console 大量重复输出。

### Pitfall 3: Zustand persist 状态跨测试泄漏
**What goes wrong:** 上一个测试的 persist 状态影响下一个测试。
**Why it happens:** 直接调用 store action 后未 reset，或 persist middleware 把状态写入了 jsdom storage。
**How to avoid:** 每个 store 测试 `beforeEach` 调用 `useXxxStore.setState(initialState)`；如需清除 storage，额外 `localStorage.clear()` / `sessionStorage.clear()`。
**Warning signs:** 测试单独通过、一起运行失败。

### Pitfall 4: fake timers 与 axios 内部 Promise 链冲突
**What goes wrong:** `vi.useFakeTimers()` 后 axios 拦截器中的 `setTimeout`（如 api.ts 的 `initEncryptionConfig` 重试）没有被正确推进，测试挂起。
**Why it happens:** Vitest fake timers 默认不替换所有 timer；axios 可能使用 `setTimeout`。
**How to avoid:** 使用 `vi.useFakeTimers({ shouldAdvanceTime: true })` 或显式 `vi.advanceTimersByTimeAsync`；对 axios 实例测试优先用 `mock adapter` 而非真实 timer。
**Warning signs:** 测试超时、Coverage 步骤卡住。

### Pitfall 5: 新增测试文件被 gate 计分方式误导
**What goes wrong:** 以为 "新增测试 = 覆盖率线性上升"，但实际 coverage 按 statements 命中计算，一个未覆盖文件可能把平均分大幅拉低。
**Why it happens:** 全量口径下所有 src 文件都进分母；新增一个大文件即使部分测试，也可能因其余语句未覆盖而拉低目录 pct。
**How to avoid:** 每个 plan 先覆盖高扇出/高 stmt 文件；使用 `npx vitest run <file> --coverage` 本地快速验证单文件覆盖率。
**Warning signs:** `npm run test:coverage` 后目标目录 pct 未达预期。

### Pitfall 6: harness 代码被计入覆盖率分母
**What goes wrong:** 把 harness 放在 `src/utils/` 或 `src/hooks/` 下，其 helper 语句被计入覆盖率，拉低 P0 目录成绩。
**Why it happens:** `coverage.exclude` 只排除了 `src/test/`、`src/components/cad-*`、`.d.ts`、测试文件。
**How to avoid:** harness 落点必须严格在 `src/test/utils/` 下。
**Warning signs:** gate 输出中出现意料之外的 `test/` 目录覆盖率行或 harness 文件出现在 coverage-final.json。

## Code Examples

### 国密向量直测（sm4）
```typescript
// Source: researcher 设计，贴合 D-08
import { encryptSM4CBC, decryptSM4CBC } from "@/utils/sm4";

describe("SM4-CBC", () => {
  const key = "0123456789abcdeffedcba9876543210";
  const iv = "abcdef98765432100123456789abcdef";

  it("加解密往返", async () => {
    const plain = "hello 国密";
    const cipher = await encryptSM4CBC(plain, key, iv);
    expect(await decryptSM4CBC(cipher, key, iv)).toBe(plain);
  });

  it("篡改密文应抛错", async () => {
    const cipher = await encryptSM4CBC("test", key, iv);
    await expect(decryptSM4CBC(cipher + "00", key, iv)).rejects.toThrow();
  });
});
```

### TokenManager fake timers 自动刷新
```typescript
// Source: researcher 设计，贴合 D-09
import { TokenManager } from "@/utils/token/TokenManager";
import { vi } from "vitest";

it("过期前 30 秒自动触发刷新", async () => {
  vi.useFakeTimers();
  const storage = createFakeStorage();
  const manager = new TokenManager(storage, {
    refreshEndpoint: "/refresh",
    refreshBeforeSeconds: 30,
    refreshTimeout: 10000,
  });
  await manager.initializeTokens("acc", "ref", 60);

  const refreshSpy = vi.spyOn(manager, "refreshToken").mockResolvedValue({
    accessToken: "new-acc",
    refreshToken: "new-ref",
    expiresIn: 60,
  });

  vi.advanceTimersByTime(31_000);
  expect(refreshSpy).toHaveBeenCalled();
  vi.useRealTimers();
});
```

### 业务层 API wrapper 测试（opsApi）
```typescript
// Source: 既有模式扩展（networkApi.test.ts）
const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));

it("buildingApi.list 调用正确端点", async () => {
  mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
  await buildingApi.list({ current: 1, pageSize: 10 });
  expect(mockPost).toHaveBeenCalledWith("/ops/building/list", {
    current: 1,
    pageSize: 10,
  });
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 旧口径 coverage.all（Vitest ≤3） | Vitest 4 `coverage.include` 显式圈定 `src/**/*.{ts,tsx}` | Phase 82 | 未测试文件计入分母，真基线从 24.58% 降到 3.85% |
| 全局 threshold 配置 | 外部 bash gate + `.coverage-fe-floors` ratchet | Phase 82 | 避免阈值配置与实测漂移导致 `test:coverage` 自锁 |
| 单文件 mock `@/lib/api` | harness `createApiMock` 端点工厂 | Phase 83（计划） | P1/P2 组件测试复用，减少样板 |
| 真实 axios 调用测试拦截器 | mock adapter + 固定密钥直测 api.ts | Phase 83（计划） | 可控、无网络、可测 401 队列与加密重放 |

**Deprecated/outdated:**
- 旧 `coverage.all: true`：Vitest 4 已移除，当前 `include` 是唯一全量口径开关 [CITED: vitest.config.ts L18-22 注释]。
- 在测试中真实等待 Token 过期：应替换为 `vi.useFakeTimers()`（D-09）。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CR-01/WR-01/WR-02/WR-03 已以提交 `60f712c`/`27f275e`/`94d3a16`/`aa3bf0c` 落库 main，无需重复实现 | Summary / Gate 修复 | 若用户仍要求按原 plan0 重做，会浪费工作量；建议首个 plan 改为 "验证 + 试验 PR + 注释清理" |
| A2 | `src/test/utils/` 目录下的文件会被 `coverage.exclude` 的 `"src/test/"` 排除，helper 不计入分母 | Harness | 若 exclude 规则不递归匹配子目录，harness 文件会进入 coverage json 拉低覆盖率；可通过本地跑 coverage 验证 |
| A3 | 现有 19 个测试文件在 Phase 83 全周期内保持 159 tests 全绿（QUAL-01 基线） | Validation Architecture | 若新增测试破坏既有测试，需在 plan 级 verify 中修复 |
| A4 | utils 与 lib 的测试可并行（utils 测试中通过 `vi.mock('@/lib/api')` 解耦） | Wave 划分 | 若存在 utils 文件无法被 mock 且依赖 lib 真实行为，则并行 plan 会阻塞 |
| A5 | Phase 83 期间 `sm-crypto` 算法行为保持稳定，国密向量可复用 | INFRA-02 | 若库升级改变密文格式，固定向量测试会失败；当前 lock 在 0.5.5，风险低 |

## Open Questions (RESOLVED)

1. **CR-01 plan 是否需要显式存在？** *(RESOLVED)*
   - Resolution: Plan 01 设为 "CR-01/WR-01~03 已落库修复验证 + ci.yml 注释措辞清理 + 发起并关闭试验 PR"，不重新实现代码。对应 commit: 60f712c / 27f275e / 94d3a16 / aa3bf0c 已在 main。

2. **router 目录当前 0% 且依赖 React Router v7 动态加载，如何低成本达到 70%？** *(RESOLVED)*
   - Resolution: Plan 05 Task 2 覆盖 routeConfigManager 纯配置转换与 routeGenerator 输出结构；componentLoader / DynamicRoutes 用 `renderWithProviders` + 最小路由 fixture，必要时在测试内 mock `import()`。

3. **types 目录大量文件为纯类型声明（0 stmts），是否只需覆盖带运行时代码的文件？** *(RESOLVED)*
   - Resolution: Plan 05 Task 1 仅覆盖含运行时代码的文件（`config.ts`、`dashboard.ts`、`notice.ts`、`widgets/helpers.ts`、`operations.ts`、`common.ts`）；0-stmt 纯接口文件不测试。

4. **utils 中 `cad/geometry.ts`（70 stmts）和 `three/colors.ts`（12 stmts）是否属于 P0 范围？** *(RESOLVED)*
   - Resolution: Plan 02 Task 2 正常覆盖；二者是纯几何/颜色函数，不依赖 canvas，计入 utils 分母。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Frontend build + test | ✓ | 24.19.0 | — |
| npm | Package manager | ✓ | 11.17.0 | — |
| node_modules | Vitest / testing-library / sm-crypto | ✓ | present | `npm install` |
| vitest | Test runner | ✓ | 4.1.10 (lock) | — |
| jsdom | DOM environment | ✓ | 27.4.0 | — |
| sm-crypto | SM2/SM4 tests | ✓ | 0.5.5 | — |
| Git Bash / sh | Gate scripts | ✓ | Git for Windows | — |
| GitHub Actions CI | Gate 真实验证 | ✓ | 已接线 | 本地 gate 脚本 |

**Missing dependencies with no fallback:** 无。

**Missing dependencies with fallback:** 无。

## Validation Architecture

> nyquist_validation 在 `.planning/config.json` 中显式启用（`"nyquist_validation": true`），本节必须存在。

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.10 + @vitest/coverage-v8 4.1.10 + jsdom + @testing-library/react 16.3.2 |
| Config file | `xingran-react-frontend/vitest.config.ts` |
| Quick run command | `cd xingran-react-frontend && npx vitest run <path/to/test.ts>` |
| Full suite command | `cd xingran-react-frontend && npm run test:coverage` |
| Gate script command | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` |
| Diff gate command | `bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json <base-ref> 80` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| INFRA-01 | lib 目录 statements 覆盖率 ≥70% | coverage gate per-dir | `bash .github/scripts/check-frontend-coverage.sh ...` 后检查 lib 行 | ❌ Wave 0 后创建 |
| INFRA-02 | utils 目录 statements 覆盖率 ≥70% | coverage gate per-dir | 同上 | ❌ Wave 0 后创建 |
| INFRA-03 | hooks 目录 statements 覆盖率 ≥70% | coverage gate per-dir | 同上 | ❌ Wave 0 后创建 |
| INFRA-04 | store 目录 statements 覆盖率 ≥70% | coverage gate per-dir | 同上 | ❌ Wave 0 后创建 |
| INFRA-05 | services / router / constants / types 各自 ≥70% | coverage gate per-dir | 同上 | ❌ Wave 0 后创建 |
| QUAL-01 | 159 存量测试零回归 | vitest 全量 | `npm run test:coverage` | ✅ 已存在 |
| QUAL-03 | harness 三件套存在且至少有一个 P0 测试使用示例 | 文件存在性 + 示例测试 | `npx vitest run src/test/utils/*.test.ts`（示例） | ❌ Wave 0 创建 |

### Success Criteria Observable Signals
| Success Criterion | Signal | How to Observe |
|-------------------|--------|----------------|
| SC-1: lib ≥70% | `lib` 行在 gate 输出中 `pct >= 70.00%` | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^lib '` |
| SC-2: utils/hooks/store/services/router/constants/types 各目录 ≥70% | 对应目录 gate 行全部 PASS | gate 输出 `PASS: <dir> ... >= 70.0%` |
| SC-3: harness 沉淀 | `src/test/utils/` 存在 `renderWithProviders.tsx`、`createApiMock.ts`、`mockAntdMessage.ts` 且至少一个 P0 测试导入使用 | `ls` + `grep` |
| SC-4: 159 测试零回归 + CI 绿 | `npm run test:coverage` 输出 `Tests 159 passed`；CI frontend job 与 frontend-coverage-diff job 均 success | 本地命令 + GitHub Actions run |
| SC-5: ratchet 单调上升 | `.coverage-fe-floors` 中 P0 目录 floor 被 bump 且 `.planning/frontend-coverage-baseline.md` 追加新行 | git diff |

### Sampling Rate
- **Per task commit:** 运行 `npx vitest run <新增/修改的测试文件>` 确保单文件通过；如涉及覆盖率，加 `--coverage` 局部验证。
- **Per wave merge:** 运行完整 `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh ...` + `bash .github/scripts/check-frontend-diff-coverage.sh ... HEAD 80`；按 D-11 bump 对应目录 floor 并追加基线文档。
- **Phase gate:** 全量测试 159 passed、gate 28/28 目录 PASS（含 P0 目录 ≥70%）、试验 PR CI 双绿后关闭不 merge。

### Wave 0 Gaps
- [ ] `src/test/utils/renderWithProviders.tsx` — Router + AntD ConfigProvider 包装器
- [ ] `src/test/utils/createApiMock.ts` — 端点工厂形态的 `@/lib/api` mock
- [ ] `src/test/utils/mockAntdMessage.ts` — antd message / Modal 统一 mock
- [ ] `src/test/utils/*.test.ts` — harness 自身使用示例（不计入 coverage）
- [ ] `src/lib/api.test.ts` — api.ts 双轨直测（INFRA-01 关键）
- [ ] `src/utils/sm4.test.ts`、`src/utils/encoding.test.ts` — 国密向量直测
- [ ] `src/utils/token/TokenManager.test.ts`、`src/utils/token/SecureTokenStorageImpl.test.ts` — TokenManager fake timers
- [ ] `src/store/*.test.ts` — 各 Zustand store 测试
- [ ] `src/router/*.test.ts` — routeConfigManager / routeGenerator 等

## Security Domain

本 phase 不新增业务功能，仅补测试。涉及的安全相关代码已在运行中，测试策略需确保：

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | 是（TokenManager / authStore 测试） | 验证 token 刷新锁、401 处理、refresh token 加密存储行为不变 |
| V3 Session Management | 是（tabsStore / authStore） | 验证登出清理 sessionStorage/localStorage 相关状态 |
| V5 Input Validation | 是（security.ts / errorHandler.ts） | 验证 XSS 检测/转义、错误消息提取边界 |
| V6 Cryptography | 是（sm2/sm4/encoding/SecureTokenStorageImpl） | 真实国密向量直测，覆盖加密/解密/篡改/密钥长度边界 |
| V8 Data Protection | 是（SecureTokenStorageImpl） | 验证 refresh token SM4 加密写入 sessionStorage，损坏数据回退为 null |

### Known Threat Patterns for Stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Token 并发刷新竞态 | Elevation of Privilege / Denial of Service | TokenManager 刷新锁 + 超时；测试需覆盖锁复用与超时 |
| Refresh token 明文落盘 | Information Disclosure | SecureTokenStorageImpl SM4 加密；测试需覆盖加密写入与损坏回退 |
| 401 响应未正确处理 | Elevation of Privilege | api.ts 401 拦截器跳转登录 + 刷新队列；测试需覆盖登录短路、刷新失败跳转 |
| XSS payload 渲染 | Tampering | security.ts escapeHtml / containsXSS；测试覆盖字符串/对象/数组场景 |
| 国密密文篡改 | Tampering | sm4 decrypt 抛错；测试覆盖篡改后异常分支 |
| 公钥缓存竞态（旧请求覆盖新公钥） | Tampering | sm2.ts generation 机制；已有 sm2.test.ts 覆盖，需保持 |

## Sources

### Primary (HIGH confidence)
- 实测 coverage json：`xingran-react-frontend/coverage/coverage-final.json`（2026-08-24 本地 `npm run test:coverage` 生成）——per-dir / per-file statements 数据全部来自此文件。
- `xingran-react-frontend/vitest.config.ts` — coverage.include / exclude 真相源、reporter、jsdom 环境、timeout。
- `.github/scripts/check-frontend-coverage.sh` — 全局阈值、per-dir floor、白名单漂移检测、WR-03 数值校验实现。
- `.github/scripts/check-frontend-diff-coverage.sh` — CR-01 pathspec 镜像、WR-01 fail-closed 实现。
- `.coverage-fe-floors` — 当前 28 目录 floor 表。
- `.planning/frontend-coverage-baseline.md` — ratchet 起点与 per-dir 快照。

### Secondary (MEDIUM confidence)
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW.md` — CR-01/WR-01/02/03 发现与复现记录。
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-VERIFICATION.md` — verifier 判定 CR-01 为 WARNING 及处置建议。
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW-FIX.md` — 修复已落库的明确记录。
- `.github/scripts/check-diff-coverage.sh` — 后端孪生脚本 fail-closed 语义参照。

### Tertiary (LOW confidence)
- Zustand 官方测试模式（resetBetweenTests）——建议在 PLAN 阶段用实际测试验证。
- React Router v7 在 jsdom 下动态加载行为——需在 router plan 中实测确认。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 版本、配置、既有测试模式均已实测核对。
- Architecture: HIGH — 源码已读，依赖方向清晰，既有测试模式可复用。
- Pitfalls: MEDIUM-HIGH — 基于既有 19 个测试文件和项目历史问题（CLAUDE.md useEffect 规则、Phase 82 gate 修复经验）。
- CR-01 已落库判定: HIGH — git log + 脚本源码 + diff gate 本地运行三重验证。

**Research date:** 2026-08-24
**Valid until:** 2026-09-07（Vitest 4 稳定，但需关注 `package.json` 中 vitest 与 coverage provider 版本声明是否统一）

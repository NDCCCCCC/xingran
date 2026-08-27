---
phase: 84-p1-70
plan: 00
subsystem: frontend-test-infrastructure
tags: [vitest, test-harness, coverage-gate, ratchet, antd-polyfill]
requires:
  - Phase 82 coverage gate(bash+awk 零依赖 gate + .coverage-fe-floors ratchet)
  - Phase 83 P0 测试基建(setup.ts matchMedia polyfill / BulkWriteDrawer·HealthCard 实证样本)
provides:
  - renderWithProviders 组件渲染 harness(MemoryRouter + antd App + 按需 store reset)
  - createApiMock 端点工厂 mock(ApiMockHandle + mockApiBatch + createApiTestingModule 静态接线)
  - setup.ts ResizeObserver 集中 polyfill(D-13)
  - check-frontend-coverage.sh components 二级聚合(父子双计,三处镜像)
  - .coverage-fe-floors 17 个 components/<sub> 行(8 目标 + 9 散件校准,初值 0.0)
affects:
  - 84-01a shared / 84-01b dashboard(wave 1 直接 import harness + bump 自身 floor)
  - 84-02a layout / 84-02b CronSelector+captcha+operations(wave 2)
  - 84-03a network+reconciliation+散件 / 84-03b design-system(wave 3 收口 bump 聚合行)
tech-stack:
  added: []
  patterns:
    - vi.mock("@/lib/api") 静态工厂接线 + 进程级端点注册表(动态登记多端点)
    - awk key 派生「父子双计」(顶层聚合语义不变 + 子目录辅助 key 双向校验)
    - 新行 0.0 初始校准(ratchet 只升不降的前提下扩展覆盖粒度)
key-files:
  created:
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    - xingran-react-frontend/src/test/utils/createApiMock.ts
    - xingran-react-frontend/src/test/utils/harness.test.tsx
  modified:
    - xingran-react-frontend/src/test/setup.ts
    - .coverage-fe-floors
    - .github/scripts/check-frontend-coverage.sh
decisions:
  - createApiMock 采用「测试文件顶部一行静态 vi.mock + 运行时动态登记」模式(Vitest 模块图约定:函数体内 vi.mock 无法拦截已加载模块)
  - gate components 二级聚合采用父子双计而非纯拆分(保住 D-06 顶层聚合行与 4.9 floor 的 ratchet 基线,TOTAL 分母不失真)
  - 散件子目录以同 commit 0.0 校准行入表(新 key 初始校准先例 = Phase 82 --init 建表),不算下调
metrics:
  duration: ~33 min
  completed: 2026-08-27
---

# Phase 84 Plan 00: wave 0 基建(harness + polyfill + floors + gate 扩展) Summary

**One-liner:** renderWithProviders/createApiMock 两件套 harness + setup.ts ResizeObserver 集中沉淀 + floors 新增 17 个 components 子目录行 + gate 三处镜像扩展(父子双计),BulkWriteDrawer/HealthCard 存量用例零回归,gate exit 0 全 PASS。

## What Was Built

### Task 1 — Harness 两件套(commit c36631b)

- `renderWithProviders(ui, options)`:默认 `<MemoryRouter initialEntries={[route]}>` + antd `<App>`(App.useApp() context 可用);options 含 `route` / `resetStores`(渲染前统一执行,Zustand resetBetweenTests 模式)/ `queryClient`(按需包 QueryClientProvider,dashboard widgets 用)。位于 `src/test/utils/`,被 vitest `coverage.exclude` 的 `"src/test/"` 排除(T-84-00-01 mitigate 到位)。
- `createApiMock(endpoint)` → `ApiMockHandle{ post,get,put,del,endpoint }`:全部原生 vi.fn(),支持 mockResolvedValue/mockRejectedValue/mockImplementationOnce;`mockApiBatch(handlers)` 批量注册并预置 response;`resetApiMocks()` 清态;`createApiTestingModule()` 生成 vi.mock("@/lib/api") 工厂返回替身(具名导出五 verb + Typed 别名 + initEncryptionConfig stub + default api 实例 stub 含 interceptors 空 hook)。
- 端点路由为进程级单例注册表:url 命中任一已注册端点 → 该端点专属 spy;未命中 → 共享通用 verb spy 回退。多端点/多工厂调用互不覆盖。
- 附带 `harness.test.tsx` 6 用例契约测试,守护拦截路由/回退/批量/reset/App context 行为。

### Task 2 — setup.ts polyfill 沉淀(commit bac9dcb)

- matchMedia 之后追加 ResizeObserverStub(observe/unobserve/disconnect 空实现),仅在 `globalThis.ResizeObserver === undefined` 时注入;注释说明只在 vitest setupFiles 生效(T-84-00-02)、按 D-13 按需沉淀纪律不前置 IntersectionObserver/PointerEvent/canvas。
- plan 0 不动 BulkWriteDrawer.test.tsx 的 inline stub(wave 1 起各 plan 顺手迁移——PLAN 明确边界,与 RESEARCH Pitfall #7 的"顺手移除"建议不一致时按 PLAN 执行)。

### Task 3+4 — floors 新行 + gate 三处扩展(commit c923424,合并原子提交)

- `.github/scripts/check-frontend-coverage.sh`:`--init`(L219 区)/GLOBAL_TABLE(L316 区)/DIR_AGG(L381 区)三处同步插入 components 分支,**父子双计**:主 key 保持顶层 `"components"`(分母不缩水,D-06 聚合行语义与既有 4.9 floor 不变),每个一级子目录追加 `"components/<sub>"` 辅助 key(grep 计数=3);带 "." 的散件文件(ConfigProvider.tsx/TargetSelector.tsx/NotificationBell.tsx)不生成子 key;TOTAL/全局加权改为主 key 增量累计(`t_s/t_c`),TOTAL 21574/3890 与 vitest Coverage summary 完全一致。
- `.coverage-fe-floors`:新增 17 行 0.0 —— 8 个 P1 目标 subdir(shared/dashboard/layout/CronSelector/captcha/operations/network/reconciliation,D-04)+ 9 个散件 subdir 校准行(DeptTree/IconSelect/NoticeDetail/asset/charts/markdown/modal/table/three);既有 28 行与 design-system 15.0 一字未动(ratchet 只升不降)。
- gate 实测(exit 0):weighted avg **18.03% ≥ GLOBAL 3.80%**;per-dir **45/45 目录全 PASS、FAIL=0**(45 = 28 既有 + 17 新行);"PASS:" grep 计数 47(45 目录行 + weighted avg 行 + 总结行)。

## Verification Results

| # | Item | Result |
|---|------|--------|
| 1 | `ls src/test/utils/` | renderWithProviders.tsx + createApiMock.ts (+harness.test.tsx) ✅ |
| 2 | `grep -c ResizeObserver src/test/setup.ts` | 6 (≥2) ✅ |
| 3 | floors `^components/\|^design-system` 计数 | 18(17 components/* + design-system 已有行);plan 写 10 系计数口径偏差(见 Deviations #5)✅ |
| 4 | `grep -c 'seg\[1\] == "components"'` | 3 ✅ |
| 5 | BulkWriteDrawer + HealthCard vitest | **11 passed**(BWD 5 + HC 6;plan 写"10",HC 实际 6 例)✅ |
| 6 | gate `PASS:` 行数 | 47(≥38 要求满足)✅ |
| 7 | gate `FAIL:` 行数(stdout/stderr 合计) | 0,exit code 0 ✅ |
| 8 | `npm run test:coverage` | exit 0;全量 vitest **75 文件 / 861 tests 全 passed**,QUAL-01 零回归 ✅ |

## Deviations from Plan

**1. [Rule 3 - Blocking issue] vi.mock 不能在工厂函数体内生效 → 「静态接线 + 动态登记」模式**
- **Found during:** Task 1
- **Issue:** PLAN/RESEARCH Pattern 2 把 `vi.mock("@/lib/api", ...)` 写在 `createApiMock()` 函数体内。Vitest 模块图约定:被测组件的静态 import 先于测试体执行,@/lib/api 已进模块缓存后函数体内的 vi.mock 无法生效(vi.mock 需在测试文件顶层被 hoist)。
- **Fix:** 导出 `createApiTestingModule()` 作为 vi.mock 异步工厂的返回替身;测试文件只需顶部一行 `vi.mock("@/lib/api", async () => (await import("@/test/utils/createApiMock")).createApiTestingModule())`;之后可在任意时机 `createApiMock(endpoint)`/`mockApiBatch` 动态登记。契约面(返回 ApiMockHandle、批量 helper、链式 mock)与 D-03 完全一致。smoke→正式 6 用例实测拦截真实生效。
- **Files:** createApiMock.ts(JSDoc「重要使用纪律」章节固化用法)

**2. [Rule 2 - Missing critical correctness] 无条件镜像会让顶层聚合桶缩水,floor 4.9 必红**
- **Found during:** Task 4 验证(gate 抓到 `FAIL: components 0.00% < 4.9%`)
- **Issue:** PLAN 字面的无条件 `components/<subdir>` 镜像把全部 stmts 移入子 key,顶层 `components` 行分母只剩散件(≈0% 实测),而「floors 只升不降」禁止下调 4.9 → gate 永久红,与 PLAN 自己的成功标准 #7(FAIL=0)矛盾。反向 scoped 方案(白名单 regex 只拆 8 个目标)又产生父桶缩水的同一问题。
- **Fix:** 父子双计设计(见上)。同时修复 GLOBAL_TABLE 的 TOTAL 求和遍历了含辅助 key 的字典导致分母虚增(25397 vs 真实 21574),改为主 key 增量累计。
- **Files:** check-frontend-coverage.sh(三处 + TOTAL 修正)

**3. [Rule 2 - Missing registrations] 9 个散件子目录必须登记否则方向-b 违例**
- **Found during:** Task 4 验证(gate 抓到 `components/DeptTree 未登记 floor` 等 4 条)
- **Issue:** 无条件镜像派生出 DeptTree/IconSelect/NoticeDetail/asset/charts/markdown/modal/table/three 等 9 个子 key,PLAN 的"+10 行"未包含它们,缺一即 direction-b FAIL(unregistered new dir)。
- **Fix:** 同 commit 以 0.0 校准行入表(Phase 82 建表即为初始校准先例,新 key 初值不属于下调)。
- **Files:** .coverage-fe-floors

**4. [流程偏差] Task 3 与 Task 4 合并为单个原子 commit(c923424)**
- **Reason:** gate 脚本扩展与 floors 新行互为存在条件:只交 floors → 新行"not found in profile"(exit 4);只改脚本 → 派生 key 未登记(exit 4)。任一拆分顺序都会留下一个 gate 红的中间 commit(bisect 隐患)。两文件作为「gate 契约单元」一次落地,验证一次性跑绿。

**5. [Plan 文本勘误,无代码影响] 若干计数口径**
- components 目标 subdir 是 **8 个**(D-04 九项含 design-system;PLAN 写"9 个 components subdir")。
- design-system 在既有表中已有 15.0 行(D-06/D-15 也把它列为"已有行的终点 bump"),不能再追加 0.0 重复行——ratchet 只升不降,重复行还会造成 gate 重复校验与 39 行假象。保留原行不动。
- 可达的最终 gate 目录行数是 **45**(28 既有 + 17 新),PLAN 断言的 38 在任何合法方案下都不可达(其数字来源于把 design-system/components 聚合行当作新增行重复计数)。验证 #6 为"≥38",实际 47 满足。
- HealthCard.test.tsx 实际 **6** 用例(BWD 5 + HC 6 = 11;PLAN/VALIDATION 写 5+5=10)。
- 后续 wave plan 引用 gate 数字时建议直接用 `grep '^PASS: per-dir'` 的 N/N 形态,避免总行数口径漂移。

## Auth Gates

None。本 plan 无外部凭证依赖;worktree 缺 node_modules/coverage profile 属环境自举(npm ci 重装 + 本地生成 profile),非 auth 事件。

## Known Stubs

None。harness 两件套为完整可用实现且有契约测试;setup.ts/gate/floors 均为数据与逻辑就位状态,无占位文本。

## Threat Flags

None。未新增网络端点/auth 路径/文件访问面/信任边界 schema 变化;T-84-00-01(harness 入 include)由 `src/test/` exclude 覆盖实测确认——coverage-final.json 中不存在 src/test/utils 路径;T-84-00-04(三处漏改)由 grep==3 + --init 17 key + gate 45/45 双跑闭环,且第一处(--init 段)确实因缩进差异险些漏改,已被抓回。

## Notes for Wave 1/2/3 Executors

- 新测试标准开头:顶部 `vi.mock("@/lib/api", async () => (await import("@/test/utils/createApiMock")).createApiTestingModule());`,随后组件 import;测试体内 `createApiMock(endpoint)` 注册断言端点。
- store 依赖组件:`renderWithProviders(<X/>, { resetStores: [() => useXxxStore.setState(initialState)] })`,不要再手写 beforeEach reset。
- 各 wave 完成后只 bump 自己的 `components/<sub>` 行至实测−0.5pp(D-14),并在同 commit 追加 `.planning/frontend-coverage-baseline.md` 行;wave 3 末尾再 bump 顶层 `components`(当前 4.9,分母恒定)与 `design-system`(当前 15.0)(D-15)。
- 波浪内如遇 IntersectionObserver/PointerEvent/canvas 渲染失败,补丁加在 setup.ts(D-13),不加到单个测试文件。

## Self-Check: PASSED

- [x] 6 个关键文件全部在盘:renderWithProviders.tsx / createApiMock.ts / harness.test.tsx / setup.ts / .coverage-fe-floors / check-frontend-coverage.sh
- [x] 3 个任务 commit 全部存在:c36631b(Task 1)/ bac9dcb(Task 2)/ c923424(Task 3+4)
- [x] gate 实测 exit 0、45/45 目录 PASS、FAIL=0;全量 vitest 75 文件 861 tests exit 0


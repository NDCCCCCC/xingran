/**
 * Phase 84 D-02 — 组件测试渲染 harness(plan 0 落地)
 *
 * 默认包裹形态对齐既有实证样本(BulkWriteDrawer.test.tsx L48-54 Wrapper):
 *   <MemoryRouter><App>{ui}</App></MemoryRouter>
 *   - MemoryRouter 提供 useNavigate / useParams 等 routing context
 *   - antd <App> 提供 message / modal / notification context(App.useApp())
 *
 * stores 走参数按需注入(D-05/Zustand 官方 resetBetweenTests 模式):
 * 调用方在 renderWithProviders 的 options.resetStores 里传入 store reset 函数,
 * 本 harness 在每次 render 前统一执行,避免测试间状态泄漏。
 *
 * 注意:本文件位于 src/test/utils/(vitest.config.ts coverage.exclude 含
 * "src/test/"),harness 自身不进入覆盖率分母(T-84-00-01 mitigations)。
 */
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntdApp } from "antd";
import { QueryClientProvider } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";

/**
 * renderWithProviders 扩展选项。
 * 故意 Omit 掉原生 "wrapper"(wrapper 形态由本 harness 锁定,
 * 不允许调用方再注入第二层 wrapper 造成路由/context 层叠错乱)。
 */
export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  /** MemoryRouter initialEntries 首项,默认 "/" */
  route?: string;
  /**
   * Zustand store reset 列表(D-02 自动 reset):每个元素是一个无参函数,
   * 例如 () => useLayoutStore.setState(initialLayoutState)。harness 在
   * render 前按顺序调用一次;调用方一般在 describe 级 define 后传入即可,
   * 无需再手写 beforeEach。
   */
  resetStores?: Array<() => void>;
  /**
   * 按需注入 react-query(仅 dashboard widgets 等使用 @tanstack/react-query
   * 的组件需要);传入已构造的 QueryClient 时最外层包一层 QueryClientProvider。
   * 不传则完全不走该 provider(避免无关测试加载查询上下文)。
   */
  queryClient?: QueryClient;
}

/**
 * 渲染任意业务组件并自动提供 Router + antd App context。
 *
 * @example
 * const handle = renderWithProviders(<HybridLayout />, {
 *   route: "/monitor",
 *   resetStores: [() => useLayoutStore.setState({ currentLayout: "hybrid" })],
 * });
 */
export function renderWithProviders(
  ui: ReactElement,
  options: RenderWithProvidersOptions = {}
): RenderResult {
  const { route = "/", resetStores, queryClient, ...renderOptions } = options;

  // D-02 自动 store reset:渲染前一次性执行调用方提供的 reset 列表
  if (resetStores?.length) {
    for (const reset of resetStores) {
      reset();
    }
  }

  let tree: ReactElement = (
    <MemoryRouter initialEntries={[route]}>
      <AntdApp>{ui}</AntdApp>
    </MemoryRouter>
  );

  if (queryClient) {
    tree = <QueryClientProvider client={queryClient}>{tree}</QueryClientProvider>;
  }

  return render(tree, renderOptions);
}

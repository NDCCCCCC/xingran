/**
 * Phase 88 D-01 — 页面渲染测试 helper(真实 hooks + mock API 模式)
 *
 * 用法(测试文件顶层,vi.mock 会自动 hoist):
 * ```ts
 * vi.mock("@/lib/api", async () => {
 *   const { createApiTestingModule } = await import("@/test/utils/createApiMock");
 *   return createApiTestingModule();
 * });
 * import { renderPageWithEndpoints } from "@/test/utils/renderPage";
 * import MyPage from "../index";
 *
 * it("renders", async () => {
 *   const { screen } = await renderPageWithEndpoints(<MyPage />, {
 *     endpoints: { "/system/things/list": { data: { list: [fixture], total: 1 } } },
 *   });
 *   expect(await screen.findByText("...")).not.toBeNull();
 * });
 * ```
 *
 * 模式要点:
 * - 不 mock 页面自身 hooks(useTableManager/usePagination 等真实执行,计入覆盖率)
 * - 只 mock @/lib/api 端点(数据由 fixture 提供)
 * - form 实例走 antd 真实 Form.useForm(不可 mock,已实证)
 * - 未注册端点统一走安全空结构 fallback(见 setGenericFallback)
 */
import type { ReactElement } from "react";
import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import {
  createApiMock,
  resetApiMocks,
  setGenericFallback,
  type ApiMockHandle,
} from "./createApiMock";
import { renderWithProviders } from "./renderWithProviders";

export interface RenderPageOptions {
  /** 端点 → mockResolvedValue 响应(自动注册端点) */
  endpoints?: Record<string, unknown>;
  /** 初始路由(默认 "/") */
  route?: string;
  /**
   * 未注册端点的通用回退响应(默认 { data: { list: [], total: 0 } })。
   * 页面渲染常并发打多个端点(statistics/tree/...)——回退给安全空结构
   * 避免逐端点登记;传 null 关闭。
   */
  fallbackResponse?: unknown | null;
}

/**
 * 常见"形状敏感"端点内置注册(batch52 发现):generic fallback 统一给
 * { data: { list: [], total: 0 } },但 tree/dropdown 类端点的 data 必须是
 * 数组、statistics 类必须是对象——形状错会让页面主 UI 崩溃
 * (deptUtils `(list ?? []).map is not a function`),表格与按钮全不渲染,
 * 页面只剩 spin 空壳,coverage 收益趋近 0。这里按 URL 语义预登记正确形状。
 */
const COMMON_ENDPOINT_SHAPES: Record<string, unknown> = {
  "/system/departments/tree": { data: [] },
  "/system/dept/tree": { data: [] },
  "/system/roles/list": { data: { list: [], total: 0 } },
  "/system/users/list": { data: { list: [], total: 0 } },
  "/system/posts/list": { data: { list: [], total: 0 } },
  "/system/menus/list": { data: { list: [], total: 0 } },
  "/system/dict/data/list": { data: { list: [], total: 0 } },
};

/** 以 /statistics 或 /dropdown-options 结尾的端点按语义给形状 */
function shapeForUrl(url: string): unknown | undefined {
  if (url.endsWith("/tree") || url.endsWith("/dropdown-options")) return { data: [] };
  if (url.endsWith("/statistics") || url.endsWith("/stats")) return { data: {} };
  return undefined;
}

export interface RenderPageResult {
  /** testing-library screen(共享查询 API) */
  screen: typeof screen;
  /** 各端点的 mock 句柄,供调用断言 */
  handles: Record<string, ApiMockHandle>;
  /** renderWithProviders 的完整返回(container/unmount/...) */
  rendered: ReturnType<typeof renderWithProviders>;
}

/**
 * 渲染页面组件并预注册 API 端点 mock。
 * 每次调用前自动 resetApiMocks,避免跨用例串扰。
 */
export function renderPageWithEndpoints(
  ui: ReactElement,
  options: RenderPageOptions = {}
): RenderPageResult {
  resetApiMocks();
  const handles: Record<string, ApiMockHandle> = {};
  for (const [endpoint, response] of Object.entries(options.endpoints ?? {})) {
    const handle = createApiMock(endpoint);
    handle.endpoint.mockResolvedValue(response);
    handles[endpoint] = handle;
  }
  // 内置形状端点:显式 endpoints 未覆盖时按语义注册(树=数组/统计=对象)
  const routeCall = (url: string): unknown =>
    options.endpoints?.[url] ?? COMMON_ENDPOINT_SHAPES[url] ?? shapeForUrl(url);
  for (const [url, response] of Object.entries(COMMON_ENDPOINT_SHAPES)) {
    if (options.endpoints?.[url] === undefined) {
      const handle = createApiMock(url);
      handle.endpoint.mockResolvedValue(response);
      handles[url] = handle;
    }
  }
  if (options.fallbackResponse !== null) {
    // generic fallback 保持对象形状;数组类端点若未被上面覆盖仍可能需要显式注册
    setGenericFallback(options.fallbackResponse ?? { data: { list: [], total: 0 } });
  }
  // 页面普遍经 useDeptTree/useDict 等 React Query hook 取数,默认注入 QueryClient
  // (retry:0 让失败立即暴露;staleTime 短保证同一用例内可重复取数)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: 0 } },
  });
  const rendered = renderWithProviders(ui, { route: options.route, queryClient });
  return { screen, handles, rendered };
}

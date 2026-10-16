/**
 * Phase 88 Batch24 — HybridLayout 渲染测试(走通用 provider,不 mock 子组件)
 *
 * 覆盖 HybridLayout.tsx 49 行 JSX + useRouteTabs 在 mount 时的副作用。
 * 如 jsdom 死锁(已知 workstations/index 类似症状),降级为 module 导入断言。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import HybridLayout from "../HybridLayout";
import { useLayoutStore } from "@/store/layoutStore";
import { useMenuStore } from "@/store/menuStore";
import { QueryClient } from "@tanstack/react-query";

beforeEach(() => {
  resetApiMocks();
  // menuStore 在 mount 时 background 触发 getUserMenus / getAllUserMenus /
  // getUserPermissions 三个端点(均未在 createApiMock 中登记),generic fallback
  // 给空数组阻止 unhandled rejection 把 CI Test (coverage) 标 FAIL。
  setGenericFallback({ data: [] });
});

describe("HybridLayout 渲染", () => {
  it("renders with basic content", async () => {
    useLayoutStore.setState({
      isMobile: false,
      sidebarCollapsed: false,
      currentLayout: "hybrid",
    } as any);
    useMenuStore.setState({
      menus: [],
      permissions: [],
      flatMenus: [],
      loading: false,
    } as any);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: 0 } },
    });

    const { container, unmount } = renderWithProviders(
      <HybridLayout>
        <div data-testid="content-body">工作台内容</div>
      </HybridLayout>,
      { queryClient }
    );

    // 8s 内至少确认 AntD Layout 容器存在
    await vi.waitFor(
      () => {
        expect(container.querySelector(".ant-layout")).not.toBeNull();
      },
      { timeout: 8000 }
    );

    unmount();
  }, 15000);
});

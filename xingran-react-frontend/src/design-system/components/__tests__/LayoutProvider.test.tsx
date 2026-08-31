/**
 * Phase 88 Batch318 — design-system/components/LayoutProvider 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, renderHook } from "@testing-library/react";
import { act } from "react";
import type { ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockLayoutStore: any = {
  currentLayout: "classic",
  sidebarCollapsed: false,
  density: "comfortable",
  setLayout: vi.fn(),
  setDensity: vi.fn(),
  toggleSidebar: vi.fn(),
  setSidebarCollapsed: vi.fn(),
};
vi.mock("@/store/layoutStore", () => ({
  useLayoutStore: vi.fn(() => mockLayoutStore),
}));

import LayoutProvider, { useLayoutContext } from "../LayoutProvider";

describe("design-system/components/LayoutProvider", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("data-layout");
    document.documentElement.removeAttribute("data-density");
    document.documentElement.removeAttribute("data-sidebar-collapsed");
    vi.clearAllMocks();
  });

  it("挂载时设置 data-layout + data-density", () => {
    mockLayoutStore.currentLayout = "compact";
    mockLayoutStore.density = "compact";
    render(
      <LayoutProvider>
        <div>child</div>
      </LayoutProvider>
    );
    expect(document.documentElement.getAttribute("data-layout")).toBe("compact");
    expect(document.documentElement.getAttribute("data-density")).toBe("compact");
  });

  it("sidebarCollapsed=true → data-sidebar-collapsed='true'", () => {
    mockLayoutStore.sidebarCollapsed = true;
    render(
      <LayoutProvider>
        <div>child</div>
      </LayoutProvider>
    );
    expect(document.documentElement.getAttribute("data-sidebar-collapsed")).toBe("true");
  });

  it("sidebarCollapsed=false → 移除属性", () => {
    document.documentElement.setAttribute("data-sidebar-collapsed", "true");
    mockLayoutStore.sidebarCollapsed = false;
    render(
      <LayoutProvider>
        <div>child</div>
      </LayoutProvider>
    );
    expect(document.documentElement.hasAttribute("data-sidebar-collapsed")).toBe(false);
  });

  it("useLayoutContext 返回 layout + sidebarCollapsed", () => {
    mockLayoutStore.currentLayout = "modern";
    mockLayoutStore.sidebarCollapsed = true;
    const wrapper = ({ children }: { children: ReactNode }) => (
      <LayoutProvider>{children}</LayoutProvider>
    );
    const { result } = renderHook(() => useLayoutContext(), { wrapper });
    expect(result.current.layout).toBe("modern");
    expect(result.current.sidebarCollapsed).toBe(true);
  });

  it("useLayoutContext 默认值 (无 provider)", () => {
    const { result } = renderHook(() => useLayoutContext());
    expect(result.current.layout).toBe("classic");
    expect(result.current.sidebarCollapsed).toBe(false);
  });

  it("children 被渲染", () => {
    render(
      <LayoutProvider>
        <span data-testid="c">child</span>
      </LayoutProvider>
    );
    expect(screen.getByTestId("c")).toBeInTheDocument();
  });
});

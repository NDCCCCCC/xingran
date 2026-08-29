/**
 * Phase 88 Batch72 — layout TabBar 渲染测试(134 行组件 + tabs store)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { useTabsStore, useTabs } from "@/store/tabsStore";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import TabBar from "../TabBar";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("TabBar 渲染", () => {
  it("tabs=[] 渲染空", () => {
    useTabsStore.setState({
      tabs: [],
      activeTab: "",
      tabHistory: {},
    } as any);
    const { baseElement } = renderWithProviders(<TabBar />);
    expect(baseElement).toBeDefined();
  });

  it("tabs 非空 + active 渲染 Tabs", () => {
    useTabsStore.setState({
      tabs: [
        { key: "/home", label: "首页", path: "/home", closable: true, pinned: false },
        { key: "/system", label: "系统", path: "/system", closable: true, pinned: true },
      ],
      activeTab: "/home",
      tabHistory: { "/home": "/home", "/system": "/system" },
    } as any);
    const { baseElement } = renderWithProviders(<TabBar />);
    expect(baseElement.querySelector(".ant-tabs")).not.toBeNull();
  });
});

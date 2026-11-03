/**
 * 契约回归：TabBar 的 hover 预取必须走「按路由路径预取」的 API，
 * 且传入的是**路由路径**（tab.path），不能把路由路径当组件路径
 * 直接喂给 componentLoader.load()（那正是本次报错的根因）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { useTabsStore } from "@/store/tabsStore";

const { preloadRouteComponentMock } = vi.hoisted(() => ({
  preloadRouteComponentMock: vi.fn(async () => true),
}));

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/router/componentLoader", () => ({
  preloadRouteComponent: preloadRouteComponentMock,
}));

import TabBar from "../TabBar";

describe("TabBar hover 预取调用契约", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTabsStore.setState({
      tabs: [
        {
          key: "/network/discoveries",
          title: "设备发现",
          path: "/network/discoveries",
          closable: true,
          pinned: false,
        },
      ],
      activeTab: "/network/discoveries",
    } as unknown);
  });

  afterEach(() => {
    vi.clearAllMocks();
    useTabsStore.setState({ tabs: [], activeTab: "" } as unknown);
  });

  it("悬停时按路由路径触发预取", async () => {
    renderWithProviders(<TabBar />);

    const label = await screen.findByText("设备发现");
    fireEvent.mouseEnter(label);

    expect(preloadRouteComponentMock).toHaveBeenCalledWith("/network/discoveries");
  });
});

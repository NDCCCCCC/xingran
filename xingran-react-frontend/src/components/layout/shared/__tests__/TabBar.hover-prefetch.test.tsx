/**
 * 回归：悬停顶部多标签栏 Tab 触发 hover 预取时，控制台不应出现
 *   [ComponentLoader] Invalid component path: ...
 * 也不应抛出未捕获的 promise rejection（Uncaught (in promise)）。
 *
 * 复现路径：TabBar 的 tab.path 是**路由 URL**，此前被直接当作组件路径
 * 传给 componentLoader.load()，被白名单拒绝后 reject，且调用方没有 .catch。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { useTabsStore } from "@/store/tabsStore";
import { routeConfigManager } from "@/router/routeConfigManager";
import type { Menu } from "@/types";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import TabBar from "../TabBar";

function makeMenu(overrides: Partial<Menu>): Menu {
  return {
    id: 1,
    parentId: 0,
    menuName: "默认",
    menuType: "M",
    path: "",
    component: "",
    icon: "",
    sortOrder: 0,
    visible: 1,
    ...overrides,
  } as Menu;
}

describe("TabBar hover 预取", () => {
  let errorSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;
  let unhandled: unknown[] = [];

  function onUnhandledRejection(reason: unknown) {
    unhandled.push(reason);
  }

  beforeEach(() => {
    routeConfigManager.clear();
    unhandled = [];

    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    process.on("unhandledRejection", onUnhandledRejection);

    // 菜单指向一个不存在的组件：预取必然失败，但必须静默
    routeConfigManager.initialize([
      makeMenu({
        id: 1,
        menuName: "幽灵页面",
        path: "ghost/page",
        menuType: "C",
        component: "ghost/not-exist/index",
      }),
    ]);

    useTabsStore.setState({
      tabs: [
        {
          key: "/ghost/page",
          title: "幽灵页面",
          path: "/ghost/page",
          closable: true,
          pinned: false,
        },
      ],
      activeTab: "/ghost/page",
    } as unknown);
  });

  afterEach(() => {
    process.off("unhandledRejection", onUnhandledRejection);
    vi.restoreAllMocks();
    routeConfigManager.clear();
    useTabsStore.setState({ tabs: [], activeTab: "" } as unknown);
  });

  it("悬停 Tab 不报错、不产生未捕获异常", async () => {
    renderWithProviders(<TabBar />);

    const label = await screen.findByText("幽灵页面");
    fireEvent.mouseEnter(label);

    // 让预取的 promise 有机会 settle / 让未处理的 rejection 冒泡
    await waitFor(() => {
      expect(true).toBe(true);
    });
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(errorSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
    expect(unhandled).toEqual([]);
  });
});

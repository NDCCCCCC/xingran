/**
 * layoutStore 布局状态(衍生状态)测试
 *
 * 覆盖:syncFromSettings、toggleSidebar/setSidebarCollapsed/setDensity/setLayout
 * (运行时临时操作+applyToDOM)、saveState 事件、useLayout hook 的
 * settings-changed 监听与 DOM 同步、layoutConfigs 静态结构。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useLayoutStore, useLayout, layoutConfigs } from "./layoutStore";
import { defaultLayoutConfiguration } from "@/types/config";

const hybridConfig = {
  ...defaultLayoutConfiguration,
  type: "hybrid" as const,
  sidebar: { ...defaultLayoutConfiguration.sidebar, collapsed: true },
  density: "compact" as const,
};

function resetLayout() {
  useLayoutStore.setState({
    configuration: defaultLayoutConfiguration,
    currentLayout: "classic",
    sidebarCollapsed: false,
    density: "comfortable",
  });
}

describe("layoutStore", () => {
  beforeEach(() => {
    localStorage.clear();
    resetLayout();
  });

  afterEach(() => {
    resetLayout();
    document.documentElement.removeAttribute("data-layout");
    document.documentElement.removeAttribute("data-density");
    document.documentElement.removeAttribute("data-sidebar-collapsed");
  });

  it("syncFromSettings 从 SettingsStore 配置同步并应用到 DOM", () => {
    useLayoutStore.getState().syncFromSettings(hybridConfig);

    const state = useLayoutStore.getState();
    expect(state.currentLayout).toBe("hybrid");
    expect(state.sidebarCollapsed).toBe(true);
    expect(state.density).toBe("compact");
    expect(state.configuration).toEqual(hybridConfig);
    expect(document.documentElement.getAttribute("data-layout")).toBe("hybrid");
    expect(document.documentElement.getAttribute("data-density")).toBe("compact");
    expect(document.documentElement.getAttribute("data-sidebar-collapsed")).toBe("true");
  });

  it("toggleSidebar 翻转折叠并写 DOM", () => {
    useLayoutStore.getState().toggleSidebar();
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true);
    expect(useLayoutStore.getState().configuration.sidebar.collapsed).toBe(true);
    expect(document.documentElement.getAttribute("data-sidebar-collapsed")).toBe("true");

    useLayoutStore.getState().toggleSidebar();
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false);
  });

  it("setSidebarCollapsed / setDensity / setLayout 运行时临时操作", () => {
    useLayoutStore.getState().setSidebarCollapsed(true);
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true);

    useLayoutStore.getState().setDensity("spacious");
    expect(useLayoutStore.getState().density).toBe("spacious");
    expect(useLayoutStore.getState().configuration.density).toBe("spacious");

    useLayoutStore.getState().setLayout("innovative");
    expect(useLayoutStore.getState().currentLayout).toBe("innovative");
    expect(useLayoutStore.getState().configuration.type).toBe("innovative");
    expect(document.documentElement.getAttribute("data-layout")).toBe("innovative");
  });

  it("saveState 广播 save-layout-settings 事件(detail=当前配置)", () => {
    useLayoutStore.getState().setLayout("hybrid");
    const listener = vi.fn();
    window.addEventListener("save-layout-settings", listener);

    useLayoutStore.getState().saveState();

    expect(listener).toHaveBeenCalledTimes(1);
    expect(listener.mock.calls[0][0].detail).toMatchObject({ type: "hybrid" });
    window.removeEventListener("save-layout-settings", listener);
  });

  it("layoutConfigs 三种布局静态结构完整", () => {
    expect(Object.keys(layoutConfigs).sort()).toEqual(["classic", "hybrid", "innovative"]);
    for (const cfg of Object.values(layoutConfigs)) {
      expect(cfg.features.sidebar).toBeDefined();
      expect(cfg.features.header).toBeDefined();
      expect(cfg.features.tabs).toBeDefined();
      expect(cfg.features.content).toBeDefined();
    }
    expect(layoutConfigs.hybrid.features.tabs.enabled).toBe(true);
    expect(layoutConfigs.classic.features.tabs.enabled).toBe(false);
  });
});

describe("useLayout hook", () => {
  beforeEach(() => {
    localStorage.clear();
    resetLayout();
  });

  afterEach(() => {
    resetLayout();
  });

  it("派生布尔便捷值 + DOM 属性同步", () => {
    const { result } = renderHook(() => useLayout());

    expect(result.current.isClassic).toBe(true);
    expect(result.current.isCompact).toBe(false);
    expect(result.current.layoutConfig).toBe(layoutConfigs.classic);

    act(() => result.current.setLayout("hybrid"));
    expect(result.current.isHybrid).toBe(true);
    expect(document.documentElement.getAttribute("data-layout")).toBe("hybrid");
  });

  it("监听 settings-changed 事件并同步配置(卸载后移除监听)", () => {
    const { unmount } = renderHook(() => useLayout());

    act(() => {
      window.dispatchEvent(
        new CustomEvent("settings-changed", { detail: { layout: hybridConfig } })
      );
    });

    const state = useLayoutStore.getState();
    expect(state.currentLayout).toBe("hybrid");
    expect(state.sidebarCollapsed).toBe(true);

    unmount();
    // 卸载后事件不再影响 store
    act(() => {
      window.dispatchEvent(
        new CustomEvent("settings-changed", {
          detail: { layout: defaultLayoutConfiguration },
        })
      );
    });
    expect(useLayoutStore.getState().currentLayout).toBe("hybrid");
  });
});

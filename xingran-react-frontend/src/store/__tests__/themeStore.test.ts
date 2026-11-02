/**
 * Phase 88 Batch196 — store/themeStore 测试
 */
import { describe, it, expect, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useThemeStore } from "../themeStore";

describe("store/themeStore", () => {
  beforeEach(() => {
    useThemeStore.setState({ mode: "light" });
    document.documentElement.setAttribute("data-color-mode", "light");
  });

  it("初始 mode = light", () => {
    expect(useThemeStore.getState().mode).toBe("light");
  });

  it("setMode 改 mode + DOM 属性", () => {
    useThemeStore.getState().setMode("dark");
    expect(useThemeStore.getState().mode).toBe("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("syncFromSettings 应用配置", () => {
    useThemeStore.getState().syncFromSettings({ mode: "dark" });
    expect(useThemeStore.getState().mode).toBe("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("applyToDOM 写入 data-color-mode", () => {
    useThemeStore.setState({ mode: "dark" });
    useThemeStore.getState().applyToDOM();
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("切回 light", () => {
    useThemeStore.getState().setMode("dark");
    useThemeStore.getState().setMode("light");
    expect(useThemeStore.getState().mode).toBe("light");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("light");
  });

  it("注册 settings-changed 事件监听器", () => {
    // 触发事件
    window.dispatchEvent(
      new CustomEvent("settings-changed", {
        detail: { theme: { mode: "dark" } },
      })
    );
    expect(useThemeStore.getState().mode).toBe("dark");
  });
});

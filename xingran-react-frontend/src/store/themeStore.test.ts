/**
 * themeStore 明暗模式(单一 color-mode store)测试
 *
 * 覆盖:syncFromSettings、setMode、applyToDOM 写 data-color-mode、
 * 模块级 settings-changed 监听器(P1-M3 先移除再注册,防 HMR 重复)。
 */
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { useThemeStore } from "./themeStore";

describe("themeStore", () => {
  beforeEach(() => {
    useThemeStore.setState({ mode: "light" });
  });

  afterEach(() => {
    useThemeStore.setState({ mode: "light" });
    document.documentElement.removeAttribute("data-color-mode");
  });

  it("初始为 light", () => {
    expect(useThemeStore.getState().mode).toBe("light");
  });

  it("setMode 更新模式并写 DOM 属性", () => {
    useThemeStore.getState().setMode("dark");

    expect(useThemeStore.getState().mode).toBe("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");

    useThemeStore.getState().setMode("light");
    expect(useThemeStore.getState().mode).toBe("light");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("light");
  });

  it("syncFromSettings 从 SettingsStore 配置同步", () => {
    useThemeStore.getState().syncFromSettings({ mode: "dark" });

    expect(useThemeStore.getState().mode).toBe("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("applyToDOM 只写 data-color-mode 一个属性", () => {
    useThemeStore.getState().setMode("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
    // 品牌色由 index.css 静态提供,store 不注入任何运行时样式变量
    expect(document.documentElement.getAttribute("style")).toBeNull();
  });

  it("settings-changed 事件驱动模式同步(模块级监听器)", () => {
    window.dispatchEvent(
      new CustomEvent("settings-changed", {
        detail: { theme: { mode: "dark" } },
      })
    );

    expect(useThemeStore.getState().mode).toBe("dark");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");

    // 切回 light 再验证一次(P1-M3: 监听器只注册一次)
    window.dispatchEvent(
      new CustomEvent("settings-changed", {
        detail: { theme: { mode: "light" } },
      })
    );
    expect(useThemeStore.getState().mode).toBe("light");
  });
});

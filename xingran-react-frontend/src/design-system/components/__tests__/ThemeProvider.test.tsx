/**
 * Phase 88 Batch319 — design-system/components/ThemeProvider 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let currentMode: string = "light";
vi.mock("@/store/themeStore", () => ({
  useThemeStore: vi.fn((selector: any) => {
    const state = { mode: currentMode };
    return typeof selector === "function" ? selector(state) : state;
  }),
}));

import { ThemeProvider, useThemeContext } from "../ThemeProvider";

describe("design-system/components/ThemeProvider", () => {
  beforeEach(() => {
    currentMode = "light";
    document.documentElement.removeAttribute("data-color-mode");
    document.documentElement.classList.remove("theme-switching");
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("挂载时设置 data-color-mode='light'", () => {
    render(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>
    );
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("light");
  });

  it("mode 变化 → 添加 theme-switching 类", () => {
    currentMode = "light";
    const { rerender } = render(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>
    );
    currentMode = "dark";
    rerender(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>
    );
    expect(document.documentElement.classList.contains("theme-switching")).toBe(true);
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("300ms 后移除 theme-switching", () => {
    currentMode = "light";
    const { rerender } = render(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>
    );
    currentMode = "dark";
    rerender(
      <ThemeProvider>
        <span>child</span>
      </ThemeProvider>
    );
    expect(document.documentElement.classList.contains("theme-switching")).toBe(true);
    vi.advanceTimersByTime(300);
    expect(document.documentElement.classList.contains("theme-switching")).toBe(false);
  });

  it("useThemeContext 返回 mode", () => {
    currentMode = "light";
    const wrapper = ({ children }: { children: ReactNode }) => (
      <ThemeProvider>{children}</ThemeProvider>
    );
    const { result } = renderHook(() => useThemeContext(), { wrapper });
    expect(result.current.mode).toBe("light");
  });

  it("useThemeContext 默认 mode='light'", () => {
    const { result } = renderHook(() => useThemeContext());
    expect(result.current.mode).toBe("light");
  });
});

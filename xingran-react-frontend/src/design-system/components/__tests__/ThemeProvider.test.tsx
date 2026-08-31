/**
 * Phase 88 Batch250 — design-system/components/ThemeProvider 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let currentMode: any = "light";
vi.mock("@/store/themeStore", () => ({
  useThemeStore: vi.fn((selector: any) => selector({ mode: currentMode })),
}));

import { ThemeProvider, useThemeContext } from "../ThemeProvider";

describe("design-system/components/ThemeProvider", () => {
  beforeEach(() => {
    currentMode = "light";
    document.documentElement.setAttribute("data-color-mode", "light");
    document.documentElement.classList.remove("theme-switching");
  });

  it("渲染 children + 提供 context", () => {
    function Reader() {
      const ctx = useThemeContext();
      return <span data-testid="mode">{ctx.mode}</span>;
    }
    render(
      <ThemeProvider>
        <Reader />
      </ThemeProvider>
    );
    expect(document.querySelector('[data-testid="mode"]')?.textContent).toBe("light");
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("light");
  });

  it("mode 变化 → 写 data-color-mode", () => {
    const { rerender } = render(
      <ThemeProvider>
        <span>x</span>
      </ThemeProvider>
    );
    currentMode = "dark";
    rerender(
      <ThemeProvider>
        <span>x</span>
      </ThemeProvider>
    );
    expect(document.documentElement.getAttribute("data-color-mode")).toBe("dark");
  });

  it("mode 变化 → 添加 theme-switching 类 (3s 内)", () => {
    vi.useFakeTimers();
    const { rerender } = render(
      <ThemeProvider>
        <span>x</span>
      </ThemeProvider>
    );
    currentMode = "dark";
    rerender(
      <ThemeProvider>
        <span>x</span>
      </ThemeProvider>
    );
    expect(document.documentElement.classList.contains("theme-switching")).toBe(true);
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(document.documentElement.classList.contains("theme-switching")).toBe(false);
    vi.useRealTimers();
  });
});

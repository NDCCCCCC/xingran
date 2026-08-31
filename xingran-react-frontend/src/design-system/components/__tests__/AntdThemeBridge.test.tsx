/**
 * Phase 88 Batch374 — design-system/components/AntdThemeBridge 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockThemeMode = "light";
let mockDensity = "comfortable";

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn((selector: any) => {
    const state = {
      preferences: {
        theme: { mode: mockThemeMode },
        layout: { density: mockDensity },
      },
    };
    return typeof selector === "function" ? selector(state) : state;
  }),
}));

vi.mock("@/utils/antdMessage", () => ({
  setAppMessageInstance: vi.fn(),
}));

import AntdThemeBridge from "../AntdThemeBridge";

describe("design-system/components/AntdThemeBridge", () => {
  it("渲染 children", () => {
    render(
      <AntdThemeBridge>
        <span data-testid="child">hello</span>
      </AntdThemeBridge>
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });

  it("light + comfortable 不抛错", () => {
    mockThemeMode = "light";
    mockDensity = "comfortable";
    expect(() =>
      render(
        <AntdThemeBridge>
          <span>x</span>
        </AntdThemeBridge>
      )
    ).not.toThrow();
  });

  it("dark mode 不抛错", () => {
    mockThemeMode = "dark";
    mockDensity = "comfortable";
    expect(() =>
      render(
        <AntdThemeBridge>
          <span>x</span>
        </AntdThemeBridge>
      )
    ).not.toThrow();
  });

  it("compact density 不抛错", () => {
    mockThemeMode = "light";
    mockDensity = "compact";
    expect(() =>
      render(
        <AntdThemeBridge>
          <span>x</span>
        </AntdThemeBridge>
      )
    ).not.toThrow();
  });

  it("dark + compact 不抛错", () => {
    mockThemeMode = "dark";
    mockDensity = "compact";
    expect(() =>
      render(
        <AntdThemeBridge>
          <span>x</span>
        </AntdThemeBridge>
      )
    ).not.toThrow();
  });

  it("调用 setAppMessageInstance", async () => {
    const { setAppMessageInstance } = await import("@/utils/antdMessage");
    render(
      <AntdThemeBridge>
        <span>x</span>
      </AntdThemeBridge>
    );
    expect(setAppMessageInstance).toHaveBeenCalled();
  });
});

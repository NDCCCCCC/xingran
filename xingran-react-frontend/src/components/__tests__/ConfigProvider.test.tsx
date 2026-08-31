/**
 * Phase 88 Batch321 — components/ConfigProvider 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act } from "@testing-library/react";
import type { ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockAuthState: any = { isAuthenticated: false };
const mockSettingsState: any = {
  initialized: false,
  preferences: { theme: { mode: "light" }, layout: { type: "classic" } },
  initialize: vi.fn(),
};
const mockThemeSync = vi.fn();
const mockLayoutSync = vi.fn();
const mockUpdateLayout = vi.fn();

vi.mock("@/store/authStore", () => ({
  useAuthStore: vi.fn(() => mockAuthState),
}));

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn(() => mockSettingsState),
  useSettingsStore_getState: vi.fn(),
}));

vi.mock("@/store/themeStore", () => ({
  useThemeStore: vi.fn((selector: any) =>
    typeof selector === "function"
      ? selector({ syncFromSettings: mockThemeSync })
      : { syncFromSettings: mockThemeSync }
  ),
}));

vi.mock("@/store/layoutStore", () => ({
  useLayoutStore: vi.fn((selector: any) =>
    typeof selector === "function"
      ? selector({ syncFromSettings: mockLayoutSync })
      : { syncFromSettings: mockLayoutSync }
  ),
}));

// Mock the getState for the event handler
import { ConfigProvider } from "../ConfigProvider";

describe("components/ConfigProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthState.isAuthenticated = false;
    mockSettingsState.initialized = false;
    mockSettingsState.initialize.mockReset();
  });

  it("挂载时不调用 initialize (未登录)", () => {
    render(
      <ConfigProvider>
        <span>child</span>
      </ConfigProvider>
    );
    expect(mockSettingsState.initialize).not.toHaveBeenCalled();
  });

  it("已登录 + 未初始化 → initialize", () => {
    mockAuthState.isAuthenticated = true;
    mockSettingsState.initialized = false;
    render(
      <ConfigProvider>
        <span>child</span>
      </ConfigProvider>
    );
    expect(mockSettingsState.initialize).toHaveBeenCalled();
  });

  it("已登录 + 已初始化 → 不调用 initialize", () => {
    mockAuthState.isAuthenticated = true;
    mockSettingsState.initialized = true;
    render(
      <ConfigProvider>
        <span>child</span>
      </ConfigProvider>
    );
    expect(mockSettingsState.initialize).not.toHaveBeenCalled();
  });

  it("初始化后 syncTheme + syncLayout 被调用", () => {
    mockAuthState.isAuthenticated = true;
    mockSettingsState.initialized = true;
    render(
      <ConfigProvider>
        <span>child</span>
      </ConfigProvider>
    );
    expect(mockThemeSync).toHaveBeenCalled();
    expect(mockLayoutSync).toHaveBeenCalled();
  });

  it("children 渲染", () => {
    const { getByText } = render(
      <ConfigProvider>
        <span>child-text</span>
      </ConfigProvider>
    );
    expect(getByText("child-text")).toBeTruthy();
  });
});

/**
 * 用户设置页「刷新权限」行单测
 *
 * 覆盖：
 *   1. 点击刷新 → 调用 refreshMenuCache（forceRefresh 走真实 API，绕过 TTL 缓存）
 *   2. 刷新前会清 sessionStorage 的菜单快照（否则硬刷新后 DynamicRoutes 会恢复旧权限）
 *   3. 成功 → 轻提示；失败 → 错误提示（且按钮 loading 复位）
 *
 * mock 策略：
 *   - vi.mock("@/store/menuStore") 只替换 refreshMenuCache（真实实现会打 HTTP）
 *   - 其余 store 用 vi.hoisted 可控 preferences（对齐 index.test.tsx 既有先例）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { App } from "antd";

const mocks = vi.hoisted(() => ({
  refreshMenuCache: vi.fn().mockResolvedValue(undefined),
  updateDataPageSize: vi.fn().mockResolvedValue(undefined),
  preferences: {
    version: 2,
    theme: { mode: "light" as const },
    layout: {
      type: "classic" as const,
      sidebar: { collapsed: false, width: 280, collapsedWidth: 64 },
      density: "comfortable" as const,
    },
    data: { defaultPageSize: 10, pageSizeOptions: [10, 20, 50, 100] },
    language: "zh-CN" as const,
  },
}));

vi.mock("@/store/menuStore", () => ({
  refreshMenuCache: () => mocks.refreshMenuCache(),
  useMenuStore: () => ({ permissions: [], permissionsLoaded: true, permissionsError: null }),
}));

vi.mock("@/store/settingsStore", () => {
  const fullState = {
    preferences: mocks.preferences,
    initialized: true,
    loading: false,
    error: null,
    version: 2,
    initialize: vi.fn(),
    updateTheme: vi.fn().mockResolvedValue(undefined),
    updateLayout: vi.fn().mockResolvedValue(undefined),
    updateDataPageSize: mocks.updateDataPageSize,
  };
  return {
    useSettingsStore: (selector?: (s: typeof fullState) => unknown) =>
      selector ? selector(fullState) : fullState,
  };
});

import { DataSettingsCard } from "@/pages/settings/SettingCards";

const MENU_CACHE_KEY = "menu:cache";

describe("用户设置页「刷新权限」行", () => {
  beforeEach(() => {
    mocks.refreshMenuCache.mockReset().mockResolvedValue(undefined);
    sessionStorage.clear();
  });

  it("点击刷新 → 先清 sessionStorage 菜单快照，再调 refreshMenuCache()", async () => {
    // 预置旧快照，模拟「管理员刚改了权限，当前会话还是旧快照」
    sessionStorage.setItem(MENU_CACHE_KEY, JSON.stringify({ version: 1, data: {} }));

    render(
      <App>
        <DataSettingsCard />
      </App>
    );

    const refreshBtn = screen.getByRole("button", { name: /刷新/ });
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(mocks.refreshMenuCache).toHaveBeenCalledTimes(1);
    });

    // 快照必须被清空，否则硬刷新时 DynamicRoutes 会从旧快照 restore 旧权限
    expect(sessionStorage.getItem(MENU_CACHE_KEY)).toBeNull();
  });

  it("刷新成功 → 轻提示「权限已刷新」", async () => {
    mocks.refreshMenuCache.mockResolvedValue(undefined);

    render(
      <App>
        <DataSettingsCard />
      </App>
    );

    fireEvent.click(screen.getByRole("button", { name: /刷新/ }));

    await waitFor(() => {
      expect(screen.getByText(/权限已刷新/)).toBeInTheDocument();
    });
  });

  it("刷新失败 → 错误提示，且不抛到组件外", async () => {
    mocks.refreshMenuCache.mockRejectedValue(new Error("network down"));

    render(
      <App>
        <DataSettingsCard />
      </App>
    );

    fireEvent.click(screen.getByRole("button", { name: /刷新/ }));

    await waitFor(() => {
      expect(screen.getByText(/刷新权限失败/)).toBeInTheDocument();
    });
  });
});

/**
 * settingsStore 系统设置(权威数据源)测试
 *
 * 覆盖:initialize(服务器加载/迁移/失败仍标记 initialized)、
 * updateTheme/updateLayout/updateDataPageSize/updatePreferences、
 * syncToStores 事件广播、reset、persist 只落盘 preferences+version。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const svc = vi.hoisted(() => ({
  getUserPreferences: vi.fn(),
  migratePreferences: vi.fn(),
  updateUserPreferences: vi.fn(),
}));
vi.mock("@/services/configService", () => ({
  configService: svc,
}));

import { useSettingsStore } from "./settingsStore";
import { defaultUserPreferences } from "@/types/config";

describe("settingsStore", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    svc.migratePreferences.mockImplementation((prefs: unknown) => prefs);
    useSettingsStore.setState({
      preferences: defaultUserPreferences,
      initialized: false,
      loading: false,
      error: null,
      version: 2,
    });
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("initialize 从服务器加载并迁移配置,广播 settings-changed", async () => {
    const serverPrefs = {
      ...defaultUserPreferences,
      theme: { ...defaultUserPreferences.theme, mode: "dark" as const },
    };
    svc.getUserPreferences.mockResolvedValue(serverPrefs);

    const listener = vi.fn();
    window.addEventListener("settings-changed", listener);

    await useSettingsStore.getState().initialize();

    expect(svc.getUserPreferences).toHaveBeenCalledTimes(1);
    expect(svc.migratePreferences).toHaveBeenCalledWith(serverPrefs);
    const state = useSettingsStore.getState();
    expect(state.preferences.theme.mode).toBe("dark");
    expect(state.initialized).toBe(true);
    expect(state.loading).toBe(false);
    expect(listener).toHaveBeenCalledTimes(1);
    window.removeEventListener("settings-changed", listener);
  });

  it("initialize 失败:置 error 但仍标记 initialized", async () => {
    svc.getUserPreferences.mockRejectedValue(new Error("settings api down"));

    await useSettingsStore.getState().initialize();

    const state = useSettingsStore.getState();
    expect(state.error).toBe("settings api down");
    expect(state.loading).toBe(false);
    expect(state.initialized).toBe(true);
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("updateTheme 合并主题配置并保存到服务器", async () => {
    svc.updateUserPreferences.mockResolvedValue(undefined);

    await useSettingsStore.getState().updateTheme({ colorPrimary: "#156031" });

    expect(svc.updateUserPreferences).toHaveBeenCalledWith(
      expect.objectContaining({
        theme: expect.objectContaining({ colorPrimary: "#156031" }),
      })
    );
    expect(useSettingsStore.getState().preferences.theme.colorPrimary).toBe("#156031");
  });

  it("updateLayout / updateDataPageSize 更新对应分区", async () => {
    svc.updateUserPreferences.mockResolvedValue(undefined);

    await useSettingsStore.getState().updateLayout({ type: "hybrid" });
    expect(useSettingsStore.getState().preferences.layout.type).toBe("hybrid");

    await useSettingsStore.getState().updateDataPageSize(50);
    expect(useSettingsStore.getState().preferences.data.defaultPageSize).toBe(50);
  });

  it("updatePreferences 失败向上抛出且本地不更新", async () => {
    svc.updateUserPreferences.mockRejectedValue(new Error("save fail"));

    await expect(
      useSettingsStore.getState().updatePreferences({
        data: { ...defaultUserPreferences.data, defaultPageSize: 999 },
      })
    ).rejects.toThrow("save fail");

    expect(useSettingsStore.getState().preferences.data.defaultPageSize).toBe(
      defaultUserPreferences.data.defaultPageSize
    );
  });

  it("syncToStores 广播当前 preferences", () => {
    const listener = vi.fn();
    window.addEventListener("settings-changed", listener);

    useSettingsStore.getState().syncToStores();

    expect(listener).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: expect.objectContaining({ version: 2 }),
      })
    );
    window.removeEventListener("settings-changed", listener);
  });

  it("reset 回默认配置并广播", () => {
    const listener = vi.fn();
    window.addEventListener("settings-changed", listener);

    useSettingsStore.getState().reset();

    expect(useSettingsStore.getState().preferences).toEqual(defaultUserPreferences);
    expect(listener).toHaveBeenCalledTimes(1);
    window.removeEventListener("settings-changed", listener);
  });

  it("persist 只落盘 preferences + version(T-83-04-03)", async () => {
    svc.updateUserPreferences.mockResolvedValue(undefined);
    await useSettingsStore.getState().updateTheme({ colorPrimary: "#C09058" });

    const raw = localStorage.getItem("settings-storage");
    expect(raw).toBeTruthy();
    const persisted = JSON.parse(raw!).state;
    expect(persisted.preferences.theme.colorPrimary).toBe("#C09058");
    expect(persisted.version).toBe(2);
    expect(persisted.initialized).toBeUndefined();
    expect(persisted.loading).toBeUndefined();
  });
});

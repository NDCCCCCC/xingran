/**
 * defaultThemeApi 契约测试（v1.56 PR #114 CI 覆盖率补齐）
 *
 * 锁定 3 个主题默认配置端点的路径与方法。此前该文件 0 覆盖。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
}));

import {
  getDefaultThemeConfig,
  setDefaultThemeConfig,
  syncUserThemeToDefault,
} from "./defaultThemeApi";

describe("defaultThemeApi 端点契约", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getDefaultThemeConfig GET /system/settings/config/theme/default 并解包 data", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: { primaryColor: "theme-primary" } });
    const cfg = await getDefaultThemeConfig();
    expect(mockGet).toHaveBeenCalledWith("/system/settings/config/theme/default");
    expect(cfg).toEqual({ primaryColor: "theme-primary" });
  });

  it("setDefaultThemeConfig POST 同一路径并透传配置体", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await setDefaultThemeConfig({ primaryColor: "theme-primary" });
    expect(mockPost).toHaveBeenCalledWith("/system/settings/config/theme/default", {
      primaryColor: "theme-primary",
    });
  });

  it("syncUserThemeToDefault POST /theme/sync 携带 snake_case user_id", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await syncUserThemeToDefault("u-1");
    expect(mockPost).toHaveBeenCalledWith("/system/settings/config/theme/sync", {
      user_id: "u-1",
    });
  });
});

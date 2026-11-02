/**
 * menuApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:三个菜单端点 URL + 空 data 回退 [] 的行为。
 * 所有 HTTP 调用经 vi.mock("@/lib/api") 拦截,不发真实网络请求。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import { getAllUserMenus, getUserMenus, getUserPermissions } from "./menuApi";

describe("menuApi", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  it("getUserMenus 调用 /system/my-menus 并返回 data", async () => {
    const menus = [{ id: "1", menuName: "用户管理" }];
    mockPost.mockResolvedValueOnce({ code: 0, data: menus });

    const result = await getUserMenus();

    expect(mockPost).toHaveBeenCalledWith("/system/my-menus", {});
    expect(result).toBe(menus);
  });

  it("getUserMenus 在 data 为空时回退 []", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await getUserMenus()).toEqual([]);
  });

  it("getAllUserMenus 调用 /system/my-menus/all(含隐藏菜单)", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: [{ id: "2" }] });

    const result = await getAllUserMenus();

    expect(mockPost).toHaveBeenCalledWith("/system/my-menus/all", {});
    expect(result).toEqual([{ id: "2" }]);
  });

  it("getAllUserMenus 空 data 回退 []", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: undefined });
    expect(await getAllUserMenus()).toEqual([]);
  });

  it("getUserPermissions 调用 /system/my-menus/permissions", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: ["system:user:list"] });

    const result = await getUserPermissions();

    expect(mockPost).toHaveBeenCalledWith("/system/my-menus/permissions", {});
    expect(result).toEqual(["system:user:list"]);
  });

  it("getUserPermissions 空 data 回退 []", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await getUserPermissions()).toEqual([]);
  });
});

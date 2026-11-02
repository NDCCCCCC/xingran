/**
 * profileApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:个人信息 / 系统设置两组端点与 HTTP 方法(get/put/post/upload)。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockUpload = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
  put: (...args: unknown[]) => mockPut(...args),
  upload: (...args: unknown[]) => mockUpload(...args),
}));
vi.mock("./api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
  put: (...args: unknown[]) => mockPut(...args),
  upload: (...args: unknown[]) => mockUpload(...args),
}));

import {
  changePassword,
  getProfileInfo,
  getUserPreferences,
  updateProfileInfo,
  updateUserPreferences,
  uploadAvatar,
} from "./profileApi";

describe("profileApi", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
    mockPut.mockReset();
    mockUpload.mockReset();
  });

  it("getProfileInfo GET /system/profile/info 并解包 data", async () => {
    const profile = { id: "u1", username: "admin" };
    mockGet.mockResolvedValueOnce({ code: 0, data: profile });

    const result = await getProfileInfo();

    expect(mockGet).toHaveBeenCalledWith("/system/profile/info");
    expect(result).toBe(profile);
  });

  it("updateProfileInfo PUT /system/profile/info", async () => {
    mockPut.mockResolvedValueOnce({ code: 0, data: { message: "ok" } });
    const payload = { nickname: "新昵称" };

    const result = await updateProfileInfo(payload);

    expect(mockPut).toHaveBeenCalledWith("/system/profile/info", payload);
    expect(result).toEqual({ message: "ok" });
  });

  it("changePassword POST /system/profile/change-password", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { message: "ok" } });
    const payload = { oldPassword: "a", newPassword: "b" };

    await changePassword(payload);

    expect(mockPost).toHaveBeenCalledWith("/system/profile/change-password", payload);
  });

  it("uploadAvatar 上传文件到 /system/profile/avatar", async () => {
    mockUpload.mockResolvedValueOnce({
      code: 0,
      data: { avatar: "/avatars/1.png", message: "ok" },
    });
    const file = new File(["img"], "avatar.png");

    const result = await uploadAvatar(file);

    expect(mockUpload).toHaveBeenCalledWith("/system/profile/avatar", file);
    expect(result).toEqual({ avatar: "/avatars/1.png", message: "ok" });
  });

  it("getUserPreferences GET /system/settings/preferences", async () => {
    const prefs = { theme_mode: "light" };
    mockGet.mockResolvedValueOnce({ code: 0, data: prefs });

    const result = await getUserPreferences();

    expect(mockGet).toHaveBeenCalledWith("/system/settings/preferences");
    expect(result).toBe(prefs);
  });

  it("updateUserPreferences PUT /system/settings/preferences", async () => {
    mockPut.mockResolvedValueOnce({ code: 0, data: { message: "ok" } });
    const prefs = { theme_mode: "dark" };

    await updateUserPreferences(prefs);

    expect(mockPut).toHaveBeenCalledWith("/system/settings/preferences", prefs);
  });
});

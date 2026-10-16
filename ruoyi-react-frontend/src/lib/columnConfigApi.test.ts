/**
 * columnConfigApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:全局列自定义 3 端点(get/save/reset)与 pageKey 拼接。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockDel = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
  del: (...args: unknown[]) => mockDel(...args),
}));

import { columnConfigApi } from "./columnConfigApi";

describe("columnConfigApi", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
    mockDel.mockReset();
  });

  it("getByPageKey GET /system/column-config/:pageKey", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: [] });

    await columnConfigApi.getByPageKey("system/user");

    expect(mockGet).toHaveBeenCalledWith("/system/column-config/system/user");
  });

  it("save POST /system/column-config 携带列配置数组", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    const data = {
      pageKey: "system/user",
      columnConfigs: [
        { columnKey: "username", visible: true },
        { columnKey: "status", visible: false, width: 80 },
      ],
    };

    await columnConfigApi.save(data);

    expect(mockPost).toHaveBeenCalledWith("/system/column-config", data);
  });

  it("reset DELETE /system/column-config/:pageKey", async () => {
    mockDel.mockResolvedValueOnce({ code: 0 });

    await columnConfigApi.reset("system/user");

    expect(mockDel).toHaveBeenCalledWith("/system/column-config/system/user");
  });
});

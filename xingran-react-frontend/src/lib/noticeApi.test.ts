/**
 * noticeApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:管理端通知 CRUD / 用户端通知 / 未读数 / WebSocket URL 构造。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
const mockGet = vi.fn();
const mockDel = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
  del: (...args: unknown[]) => mockDel(...args),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
  del: (...args: unknown[]) => mockDel(...args),
}));

// buildWebSocketUrl 经 SecureTokenStorageImpl 读取 accessToken
const mockGetAccessToken = vi.fn();
vi.mock("@/utils/token/SecureTokenStorageImpl", () => ({
  SecureTokenStorageImpl: class {
    getAccessToken() {
      return mockGetAccessToken();
    }
  },
}));

import {
  batchDeleteNotices,
  buildWebSocketUrl,
  createNotice,
  deleteNotice,
  getMyNoticeDetail,
  getMyNotices,
  getNoticeDetail,
  getNoticeList,
  getNoticeStatusStatistics,
  getNoticeStatistics,
  getNotificationList,
  getUnreadCount,
  ignoreNotice,
  markAllNoticesAsRead,
  markNoticeAsRead,
  publishNotice,
  unignoreNotice,
  updateNotice,
  withdrawNotice,
} from "./noticeApi";

const OK = { code: 0 };

describe("noticeApi — 管理端", () => {
  beforeEach(() => {
    mockPost.mockReset();
    mockGet.mockReset();
    mockDel.mockReset();
  });

  it("getNoticeList POST /system/notices/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10, title: "通知" };
    await getNoticeList(params);
    expect(mockPost).toHaveBeenCalledWith("/system/notices/list", params);
  });

  it("getNoticeStatusStatistics POST /system/notices/statistics", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getNoticeStatusStatistics();
    expect(mockPost).toHaveBeenCalledWith("/system/notices/statistics", {});
  });

  it("getNoticeDetail / create / update / batchDelete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await getNoticeDetail("n1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/system/notices/n1", {});
    const create = { title: "停机通知", content: "内容", noticeType: 1 };
    await createNotice(create);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/system/notices", create);
    await updateNotice("n1", { title: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/system/notices/n1/update", { title: "改名" });
    await deleteNotice("n1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/system/notices/n1/delete", {});
    await batchDeleteNotices(["n1", "n2"]);
    expect(mockPost).toHaveBeenNthCalledWith(5, "/system/notices/batch-delete", {
      ids: ["n1", "n2"],
    });
  });

  it("getNoticeStatistics GET /system/notices/:id/statistics", async () => {
    mockGet.mockResolvedValueOnce(OK);
    await getNoticeStatistics("n1");
    expect(mockGet).toHaveBeenCalledWith("/system/notices/n1/statistics");
  });

  it("publishNotice / withdrawNotice", async () => {
    mockPost.mockResolvedValue(OK);
    await publishNotice("n1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/system/notices/n1/publish", {});
    await withdrawNotice("n1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/system/notices/n1/withdraw", {});
  });
});

describe("noticeApi — 用户端", () => {
  beforeEach(() => {
    mockPost.mockReset();
    mockGet.mockReset();
    mockDel.mockReset();
  });

  it("getMyNotices GET /system/my-notices(params 可省略)", async () => {
    mockGet.mockResolvedValue(OK);
    await getMyNotices();
    expect(mockGet).toHaveBeenNthCalledWith(1, "/system/my-notices", {});
    await getMyNotices({ current: 2, pageSize: 20 });
    expect(mockGet).toHaveBeenNthCalledWith(2, "/system/my-notices", { current: 2, pageSize: 20 });
  });

  it("getMyNoticeDetail GET /system/my-notices/:id", async () => {
    mockGet.mockResolvedValueOnce(OK);
    await getMyNoticeDetail("n1");
    expect(mockGet).toHaveBeenCalledWith("/system/my-notices/n1");
  });

  it("markNoticeAsRead / markAllNoticesAsRead / ignoreNotice / unignoreNotice", async () => {
    mockPost.mockResolvedValue(OK);
    mockDel.mockResolvedValue(OK);
    await markNoticeAsRead("n1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/system/my-notices/n1/read", {});
    await markAllNoticesAsRead();
    expect(mockPost).toHaveBeenNthCalledWith(2, "/system/my-notices/read-all", {});
    await ignoreNotice("n1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/system/my-notices/n1/ignore", {});
    await unignoreNotice("n1");
    expect(mockDel).toHaveBeenCalledWith("/system/my-notices/n1/ignore");
  });

  it("getUnreadCount GET /system/my-notices/unread-count", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: { count: 3 } });
    await getUnreadCount();
    expect(mockGet).toHaveBeenCalledWith("/system/my-notices/unread-count");
  });

  it("getNotificationList 默认 current=1/pageSize=10 且允许覆盖(铃铛下拉)", async () => {
    mockGet.mockResolvedValue(OK);
    await getNotificationList();
    expect(mockGet).toHaveBeenNthCalledWith(1, "/system/my-notices", { current: 1, pageSize: 10 });
    await getNotificationList({ current: 2, pageSize: 5 });
    expect(mockGet).toHaveBeenNthCalledWith(2, "/system/my-notices", { current: 2, pageSize: 5 });
  });
});

describe("noticeApi — buildWebSocketUrl", () => {
  const originalEnv = { ...import.meta.env };

  afterEach(() => {
    // 还原 stub 的环境变量
    for (const key of ["VITE_API_BASE_URL", "VITE_WS_BASE_URL"] as const) {
      if (key in originalEnv) {
        import.meta.env[key] = originalEnv[key];
      } else {
        delete import.meta.env[key];
      }
    }
    vi.unstubAllEnvs();
  });

  it("从 API base 推导:去掉 /api/vN 后缀并转换 http→ws", () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://ops.example.com/api/v1");
    vi.stubEnv("VITE_WS_BASE_URL", "");
    mockGetAccessToken.mockReturnValue("tok");

    const url = buildWebSocketUrl();

    expect(url).toBe("ws://ops.example.com/system/ws/notices?token=tok");
  });

  it("优先使用独立 VITE_WS_BASE_URL", () => {
    vi.stubEnv("VITE_WS_BASE_URL", "wss://ws.example.com");
    mockGetAccessToken.mockReturnValue("");

    const url = buildWebSocketUrl();

    expect(url).toBe("wss://ws.example.com/system/ws/notices?token=");
  });

  it("相对路径 /api/v1 保持原样(走同源反代)", () => {
    vi.stubEnv("VITE_API_BASE_URL", "");
    vi.stubEnv("VITE_WS_BASE_URL", "");
    mockGetAccessToken.mockReturnValue("tok");

    const url = buildWebSocketUrl();

    expect(url).toBe("/system/ws/notices?token=tok");
  });
});

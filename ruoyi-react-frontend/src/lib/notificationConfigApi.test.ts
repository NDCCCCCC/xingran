/**
 * notificationConfigApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:邮箱配置 / API 通知配置 CRUD + test 端点 + cron 表达式 + 常量映射。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockDel = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
  put: (...args: unknown[]) => mockPut(...args),
  del: (...args: unknown[]) => mockDel(...args),
}));

import {
  APIConfigTypes,
  AuthTypes,
  createAPINotificationConfig,
  createEmailConfig,
  deleteAPINotificationConfig,
  deleteEmailConfig,
  getAPINotificationConfig,
  getAPINotificationConfigList,
  getCommonCronExpressions,
  getEmailConfig,
  getEmailConfigList,
  testAPINotificationConfig,
  testEmailConfig,
  updateAPINotificationConfig,
  updateEmailConfig,
} from "./notificationConfigApi";

const EMAIL_BASE = "/system/settings/notification/email-configs";
const API_BASE = "/system/settings/notification/api-notification-configs";

describe("notificationConfigApi — 邮箱配置", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
    mockPut.mockReset();
    mockDel.mockReset();
  });

  it("getEmailConfigList POST list 端点", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    const params = { current: 1, pageSize: 10 };
    await getEmailConfigList(params);
    expect(mockPost).toHaveBeenCalledWith(`${EMAIL_BASE}/list`, params);
  });

  it("getEmailConfig GET /:id", async () => {
    mockGet.mockResolvedValueOnce({ code: 0 });
    await getEmailConfig("e1");
    expect(mockGet).toHaveBeenCalledWith(`${EMAIL_BASE}/e1`);
  });

  it("createEmailConfig POST 根端点,updateEmailConfig PUT /:id,deleteEmailConfig DEL /:id", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    mockPut.mockResolvedValue({ code: 0 });
    mockDel.mockResolvedValue({ code: 0 });

    const create = { configName: "企业邮箱", host: "smtp.example.com", port: 465 };
    await createEmailConfig(create);
    expect(mockPost).toHaveBeenCalledWith(EMAIL_BASE, create);

    await updateEmailConfig("e1", { configName: "改名" });
    expect(mockPut).toHaveBeenCalledWith(`${EMAIL_BASE}/e1`, { configName: "改名" });

    await deleteEmailConfig("e1");
    expect(mockDel).toHaveBeenCalledWith(`${EMAIL_BASE}/e1`);
  });

  it("testEmailConfig POST /:id/test 携带 testTo", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await testEmailConfig("e1", "ops@example.com");
    expect(mockPost).toHaveBeenCalledWith(`${EMAIL_BASE}/e1/test`, { testTo: "ops@example.com" });
  });
});

describe("notificationConfigApi — API 通知配置", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
    mockPut.mockReset();
    mockDel.mockReset();
  });

  it("list/get/create/update/delete 全链路", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    mockGet.mockResolvedValue({ code: 0 });
    mockPut.mockResolvedValue({ code: 0 });
    mockDel.mockResolvedValue({ code: 0 });

    const params = { current: 1, pageSize: 10 };
    await getAPINotificationConfigList(params);
    expect(mockPost).toHaveBeenNthCalledWith(1, `${API_BASE}/list`, params);

    await getAPINotificationConfig("a1");
    expect(mockGet).toHaveBeenCalledWith(`${API_BASE}/a1`);

    const create = {
      configName: "webhook",
      configType: "webhook",
      url: "https://hook.example.com",
    };
    await createAPINotificationConfig(create);
    expect(mockPost).toHaveBeenNthCalledWith(2, API_BASE, create);

    await updateAPINotificationConfig("a1", { configName: "改名" });
    expect(mockPut).toHaveBeenCalledWith(`${API_BASE}/a1`, { configName: "改名" });

    await deleteAPINotificationConfig("a1");
    expect(mockDel).toHaveBeenCalledWith(`${API_BASE}/a1`);
  });

  it("testAPINotificationConfig POST /:id/test", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await testAPINotificationConfig("a1");
    expect(mockPost).toHaveBeenCalledWith(`${API_BASE}/a1/test`);
  });
});

describe("notificationConfigApi — 常量与 cron", () => {
  it("APIConfigTypes / AuthTypes 常量值锁定", () => {
    expect(APIConfigTypes).toEqual({ SMS: "sms", WEBHOOK: "webhook", PUSH: "push" });
    expect(AuthTypes).toEqual({ NONE: "none", BASIC: "basic", BEARER: "bearer", APIKEY: "apikey" });
  });

  it("getCommonCronExpressions GET /system/notices/cron-expressions", async () => {
    mockGet.mockReset();
    mockGet.mockResolvedValueOnce({ code: 0, data: [] });
    await getCommonCronExpressions();
    expect(mockGet).toHaveBeenCalledWith("/system/notices/cron-expressions");
  });
});

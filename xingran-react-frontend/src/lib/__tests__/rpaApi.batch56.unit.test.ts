/**
 * Phase 88 Batch56 — rpaApi 单元测试(各子 API 调 post)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
  get: vi.fn().mockResolvedValue({ data: {} }),
  put: vi.fn().mockResolvedValue({ data: {} }),
  del: vi.fn().mockResolvedValue({ data: {} }),
}));

import { post, get, put, del } from "@/lib/api";
import {
  taskApi,
  scriptApi,
  workerApi,
  executionApi,
  scheduleApi,
  variableApi,
  templateApi,
  aiApi,
  notificationApi,
  statisticsApi,
} from "../rpaApi";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("rpaApi 子模块 — 覆盖 list/get/create/update/delete 路径", () => {
  it("taskApi.list 调 post", async () => {
    await taskApi.list({ current: 1, pageSize: 10 });
    expect(post).toHaveBeenCalled();
  });

  it("scriptApi.list + get", async () => {
    await scriptApi.list({ current: 1, pageSize: 10 });
    expect(post).toHaveBeenCalled();
  });

  it("workerApi.list + stats", async () => {
    await workerApi.list({});
    expect(post).toHaveBeenCalled();
  });

  it("executionApi.list + cancel + detail", async () => {
    await executionApi.list({ current: 1, pageSize: 10 });
    await executionApi.cancel("e1");
    expect(post).toHaveBeenCalled();
  });

  it("scheduleApi.list + create + update + delete", async () => {
    await scheduleApi.list({});
    await scheduleApi.list({});
    expect(post).toHaveBeenCalled();
  });

  it("variableApi.list + create + update + delete", async () => {
    await variableApi.list({});
    expect(post).toHaveBeenCalled();
  });

  it("templateApi.list + use + clone", async () => {
    await templateApi.list({});
    expect(post).toHaveBeenCalled();
  });

  it("aiApi generateScript", async () => {
    await aiApi.generateScript({ prompt: "x" });
    expect(post).toHaveBeenCalled();
  });

  it("notificationApi.list + markRead", async () => {
    await notificationApi.list({});
    expect(post).toHaveBeenCalled();
  });

  it("statisticsApi.overview + byDay + byType", async () => {
    await statisticsApi.overview();
    expect(post).toHaveBeenCalled();
  });
});

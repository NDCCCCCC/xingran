/**
 * Phase 144 ALRT-03 — lib/adDriftAlertApi 测试
 *
 * 断言：5 个 API 全走 `@/lib/api` 的 wrapped `post`（禁止裸 axios）、
 * 路径正确、直接返回 `result.data`（不手动判 code===0）、分页 current 从 1。
 *
 * ⚠️ vi.mock 工厂会被 hoist 到文件顶部，不能引用普通顶层 const；用 vi.hoisted()
 * 定义 spy 后再在工厂中引用（vitest 官方约定）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

const { postMock } = vi.hoisted(() => {
  const postMock = vi.fn(async (url: string, data?: unknown) => ({
    code: 0,
    message: "success",
    timestamp: 1,
    request_id: "req-1",
    data: {
      // 通用回显，便于断言「返回的是 result.data 而非整个 BaseResponse」
      url,
      payload: data,
      enabled: true,
      tolerance: 0,
      throttleMinutes: 60,
      noticeEnabled: true,
      wsEnabled: true,
      webhookEnabled: false,
      webhookConfigId: "",
      webhookTemplate: "",
      baseline: "20,4",
      lastAlertAt: null,
      lastAlertDrifted: false,
      alertCount: 7,
      list: [{ id: "log-1", createdAt: "2026-11-09 00:00:00", drifted: true }],
      total: 1,
      current: 1,
      pageSize: 10,
    },
  }));
  return { postMock };
});

vi.mock("@/lib/api", () => ({
  // 只暴露 post：被测模块只应依赖 wrapped post；无 axios / get / put / del 导出，
  // 若被测模块偷用裸 axios 或其它 verb 会在 import 阶段直接报错（结构性防回归）。
  post: postMock,
}));

import { post } from "@/lib/api";
import {
  getStatus,
  listTriggerLogs,
  getConfig,
  updateConfig,
  triggerOnce,
  markTriggerLogRead,
  type ADDriftAlertConfig,
} from "../adDriftAlertApi";

const fullConfig: ADDriftAlertConfig = {
  enabled: true,
  tolerance: 0,
  throttleMinutes: 60,
  noticeEnabled: true,
  wsEnabled: true,
  webhookEnabled: false,
  webhookConfigId: "",
  webhookTemplate: "",
  baseline: "20,4",
};

describe("lib/adDriftAlertApi — wrapped post 契约", () => {
  beforeEach(() => {
    postMock.mockClear();
  });

  it("getStatus 走 post(/status) 并返回 result.data", async () => {
    const r = await getStatus();
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/status");
    expect(r).toMatchObject({ enabled: true, alertCount: 7, baseline: "20,4" });
    // 必须是 result.data 本身，而非整个 BaseResponse（不应回带 code/message）
    expect(r).not.toHaveProperty("code");
    expect(r).not.toHaveProperty("request_id");
  });

  it("listTriggerLogs 走 post(/logs) + current 从 1 + 分页形状", async () => {
    const r = await listTriggerLogs({ current: 1, pageSize: 10 });
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/logs", {
      current: 1,
      pageSize: 10,
    });
    expect(r).toMatchObject({ total: 1, current: 1, pageSize: 10 });
    expect(Array.isArray(r.list)).toBe(true);
  });

  it("listTriggerLogs 只发分页参数（B-1 / v1.56：后端不支持时间/状态筛选，不伪造）", async () => {
    await listTriggerLogs({ current: 2, pageSize: 20 });
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/logs", {
      current: 2,
      pageSize: 20,
    });
  });

  it("markTriggerLogRead 走 post(/logs/read) + {logId} 并返回 result.data", async () => {
    const r = await markTriggerLogRead("log-1");
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/logs/read", { logId: "log-1" });
    expect(r).not.toHaveProperty("code");
  });

  it("getConfig 走 post(/config) 并返回 9 旋钮", async () => {
    const r = await getConfig();
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/config");
    expect(r).toHaveProperty("webhookConfigId");
    expect(r).toHaveProperty("webhookTemplate");
    expect(r).not.toHaveProperty("code");
  });

  it("updateConfig 走 post(/config/update) 并原样透传 payload", async () => {
    const r = await updateConfig(fullConfig);
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/config/update", fullConfig);
    expect(r).toMatchObject({ enabled: true });
  });

  it("triggerOnce 走 post(/trigger)（写操作）", async () => {
    const r = await triggerOnce();
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/trigger");
    expect(r).not.toHaveProperty("code");
  });

  it("6 个 API 全部只调用 wrapped post（零裸 axios / 零其它 verb）", async () => {
    await getStatus();
    await listTriggerLogs({ current: 1, pageSize: 10 });
    await getConfig();
    await updateConfig(fullConfig);
    await triggerOnce();
    await markTriggerLogRead("log-1");
    expect(postMock).toHaveBeenCalledTimes(6);
    for (const call of postMock.mock.calls) {
      expect(typeof call[0]).toBe("string");
      expect(call[0]).toMatch(/^\/system\/ad-drift-alert\//);
    }
  });
});

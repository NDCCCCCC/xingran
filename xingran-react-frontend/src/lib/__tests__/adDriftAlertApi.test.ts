/**
 * Phase 144 ALRT-03 / quick 261110-jci — lib/adDriftAlertApi 测试
 *
 * 断言：6 个 API 全走 `@/lib/api` 的 wrapped `post`（禁止裸 axios）、
 * 路径正确、直接返回 `result.data`（不手动判 code===0）、分页 current 从 1。
 *
 * config 契约（quick 261110-jci）：后端 `GET /config` 返回 9 个
 * `ad_sync.drift_alert.*` dotted key 的 `map[string]string`，`POST /config/update`
 * 用同样形状 + 白名单消费。本文件锁死：
 *   - getConfig 把 dotted-key map 还原成 camelCase 类型化对象（string→bool/int）
 *   - updateConfig 发 dotted-key 全 string map（不再发 camelCase 对象 / 不发 400）
 *   - webhookTemplate 死控件已删（后端根本不存在该旋钮）
 *   - parseConfigView / toConfigPayload 的边界（空串 / 非法值 / 往返一致）
 *
 * ⚠️ vi.mock 工厂会被 hoist 到文件顶部，不能引用普通顶层 const；用 vi.hoisted()
 * 定义 spy 后再在工厂中引用（vitest 官方约定）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

const { postMock } = vi.hoisted(() => {
  // 后端 GET /config 真实形状：dotted-key + 全 string（9 个旋钮）。
  const configView: Record<string, string> = {
    "ad_sync.drift_alert.enabled": "true",
    "ad_sync.drift_alert.tolerance": "0",
    "ad_sync.drift_alert.throttle_minutes": "60",
    "ad_sync.drift_alert.notice_enabled": "true",
    "ad_sync.drift_alert.ws_enabled": "true",
    "ad_sync.drift_alert.webhook_enabled": "false",
    "ad_sync.drift_alert.webhook_config_id": "",
    "ad_sync.drift_alert.baseline": "20,4",
    "ad_sync.drift_alert.baseline_key": "ad_sync.drift_alert.baseline",
  };
  const postMock = vi.fn(async (url: string, data?: unknown) => ({
    code: 0,
    message: "success",
    timestamp: 1,
    request_id: "req-1",
    data: {
      // 通用回显，便于断言「返回的是 result.data 而非整个 BaseResponse」
      url,
      payload: data,
      ...configView,
      enabled: true,
      tolerance: 0,
      throttleMinutes: 60,
      noticeEnabled: true,
      wsEnabled: true,
      webhookEnabled: false,
      webhookConfigId: "",
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
  parseConfigView,
  toConfigPayload,
  AD_DRIFT_CONFIG_KEYS,
  type ADDriftAlertConfig,
  type ADDriftAlertConfigView,
} from "../adDriftAlertApi";

const fullConfig: ADDriftAlertConfig = {
  enabled: true,
  tolerance: 0,
  throttleMinutes: 60,
  noticeEnabled: true,
  wsEnabled: true,
  webhookEnabled: false,
  webhookConfigId: "",
  baseline: "20,4",
};

const fullView: ADDriftAlertConfigView = {
  "ad_sync.drift_alert.enabled": "true",
  "ad_sync.drift_alert.tolerance": "0",
  "ad_sync.drift_alert.throttle_minutes": "60",
  "ad_sync.drift_alert.notice_enabled": "true",
  "ad_sync.drift_alert.ws_enabled": "true",
  "ad_sync.drift_alert.webhook_enabled": "false",
  "ad_sync.drift_alert.webhook_config_id": "",
  "ad_sync.drift_alert.baseline": "20,4",
  "ad_sync.drift_alert.baseline_key": "ad_sync.drift_alert.baseline",
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

  it("getConfig 把后端 dotted-key map 还原成 camelCase 类型化对象（不再全落空）", async () => {
    const r = await getConfig();
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/config");
    expect(r).toEqual({
      enabled: true,
      tolerance: 0,
      throttleMinutes: 60,
      noticeEnabled: true,
      wsEnabled: true,
      webhookEnabled: false,
      webhookConfigId: "",
      baseline: "20,4",
    });
    // 死字段已删
    expect(r).not.toHaveProperty("webhookTemplate");
    expect(r).not.toHaveProperty("code");
  });

  it("updateConfig 发 9 个 dotted-key 全 string payload（与后端白名单对齐，不再 400）", async () => {
    const r = await updateConfig(fullConfig);
    const expected: ADDriftAlertConfigView = {
      "ad_sync.drift_alert.enabled": "true",
      "ad_sync.drift_alert.tolerance": "0",
      "ad_sync.drift_alert.throttle_minutes": "60",
      "ad_sync.drift_alert.notice_enabled": "true",
      "ad_sync.drift_alert.ws_enabled": "true",
      "ad_sync.drift_alert.webhook_enabled": "false",
      "ad_sync.drift_alert.webhook_config_id": "",
      "ad_sync.drift_alert.baseline": "20,4",
      "ad_sync.drift_alert.baseline_key": "ad_sync.drift_alert.baseline",
    };
    expect(post).toHaveBeenCalledWith("/system/ad-drift-alert/config/update", expected);
    // 绝不再发 camelCase 对象或虚构字段
    const sent = postMock.mock.calls[0][1] as Record<string, unknown>;
    expect(sent).not.toHaveProperty("webhookTemplate");
    expect(sent).not.toHaveProperty("enabled");
    // 值必须全是 string
    for (const v of Object.values(sent)) expect(typeof v).toBe("string");
    // 返回合并后的表单对象供页面刷新
    expect(r).toEqual(fullConfig);
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

describe("quick 261110-jci — config 双向适配边界", () => {
  it("AD_DRIFT_CONFIG_KEYS 与后端 9 个 dotted key 逐字一致", () => {
    expect(Object.values(AD_DRIFT_CONFIG_KEYS)).toEqual([
      "ad_sync.drift_alert.enabled",
      "ad_sync.drift_alert.tolerance",
      "ad_sync.drift_alert.throttle_minutes",
      "ad_sync.drift_alert.notice_enabled",
      "ad_sync.drift_alert.ws_enabled",
      "ad_sync.drift_alert.webhook_enabled",
      "ad_sync.drift_alert.webhook_config_id",
      "ad_sync.drift_alert.baseline",
      "ad_sync.drift_alert.baseline_key",
    ]);
  });

  it("parseConfigView：空 / 缺键 / 非法数值安全回落，不抛错", () => {
    expect(parseConfigView(null)).toMatchObject({
      enabled: false,
      tolerance: 0,
      throttleMinutes: 0,
      noticeEnabled: false,
      wsEnabled: false,
      webhookEnabled: false,
      webhookConfigId: "",
      baseline: "",
    });
    expect(
      parseConfigView({
        "ad_sync.drift_alert.enabled": "TRUE", // 仅严格 'true' 为真
        "ad_sync.drift_alert.tolerance": "abc",
        "ad_sync.drift_alert.throttle_minutes": "-5",
      })
    ).toMatchObject({ enabled: false, tolerance: 0, throttleMinutes: 0 });
  });

  it("toConfigPayload → parseConfigView 往返一致（合法输入）", () => {
    const roundTrip = parseConfigView(toConfigPayload(fullConfig));
    expect(roundTrip).toEqual(fullConfig);
  });

  it("toConfigPayload 只发后端白名单内的 9 个 key，零 camelCase / 零 webhookTemplate", () => {
    const payload = toConfigPayload(fullConfig);
    expect(Object.keys(payload).sort()).toEqual(Object.values(AD_DRIFT_CONFIG_KEYS).sort());
    expect(payload).not.toHaveProperty("webhookTemplate");
    expect(payload).not.toHaveProperty("enabled");
  });

  it("parseConfigView(fullView) 与 fullConfig 一致", () => {
    expect(parseConfigView(fullView)).toEqual(fullConfig);
  });
});

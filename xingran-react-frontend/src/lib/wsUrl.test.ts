import { afterEach, beforeAll, afterAll, describe, expect, it, vi } from "vitest";
import { buildWsUrl, dashboardWsUrl, deploymentPrefix, noticeWsUrl } from "./wsUrl";

// 前端 WebSocket URL 构造的直达测试。
//
// 关注两件事：
//   1. 子路径部署（生产经 nginx 以 /xingran/ 暴露）时，WS 握手必须带同一前缀，
//      否则会落到同端口的其它服务上，必然失败；
//   2. WS 端点路径由 API 基址派生，与后端注册的 /api/v1/system/ws/* 保持一致，
//      不能另写一份常量（否则必然漂移）。

const originalLocation = window.location;

beforeAll(() => {
  // jsdom 的 location 不可直接改写，整体替换成可控值以覆盖 https/wss 分支。
  Object.defineProperty(window, "location", {
    configurable: true,
    value: {
      ...originalLocation,
      protocol: "http:",
      host: "localhost:3000",
      origin: "http://localhost:3000",
    },
  });
});

afterAll(() => {
  Object.defineProperty(window, "location", {
    configurable: true,
    value: originalLocation,
  });
});

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("deploymentPrefix", () => {
  it("根路径部署：API 基址以 /api/v1 结尾 → 前缀为空", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/api/v1");
    expect(deploymentPrefix()).toBe("");
  });

  it("子路径部署：/xingran/api/v1 → 前缀为 /xingran", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/api/v1");
    expect(deploymentPrefix()).toBe("/xingran");
  });

  it("绝对 URL 的 API 基址取其 pathname 判定前缀", () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://ops.example.com/xingran/api/v1");
    expect(deploymentPrefix()).toBe("/xingran");
  });

  it("不以 /api/vN 结尾的基址 → 前缀为空", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/custom");
    expect(deploymentPrefix()).toBe("");
  });

  it("未配置 VITE_API_BASE_URL → 回落到默认 /api/v1，前缀为空", () => {
    vi.stubEnv("VITE_API_BASE_URL", "");
    expect(deploymentPrefix()).toBe("");
  });

  it("非法 URL → 走 catch 分支，仍能取到基址", () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://[");
    expect(deploymentPrefix()).toBe("");
  });
});

describe("buildWsUrl", () => {
  it("路径缺少前导斜杠时自动补齐", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/api/v1");
    expect(buildWsUrl("system/ws/notices")).toBe("ws://localhost:3000/system/ws/notices");
  });

  it("路径已含部署前缀时不重复拼接", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/api/v1");
    expect(buildWsUrl("/xingran/api/v1/system/ws/notices")).toBe(
      "ws://localhost:3000/xingran/api/v1/system/ws/notices"
    );
  });

  it("路径不含部署前缀时自动补上", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/api/v1");
    expect(buildWsUrl("/api/v1/system/ws/notices")).toBe(
      "ws://localhost:3000/xingran/api/v1/system/ws/notices"
    );
  });

  it("VITE_WS_HOST 覆盖当前 host", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/api/v1");
    vi.stubEnv("VITE_WS_HOST", "ws.internal:9443");
    expect(buildWsUrl("/api/v1/system/ws/notices")).toBe(
      "ws://ws.internal:9443/api/v1/system/ws/notices"
    );
  });

  it("https 页面 → wss 协议", () => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        ...originalLocation,
        protocol: "https:",
        host: "secure.example.com",
        origin: "https://secure.example.com",
      },
    });
    vi.stubEnv("VITE_API_BASE_URL", "/api/v1");

    expect(buildWsUrl("/api/v1/system/ws/notices")).toBe(
      "wss://secure.example.com/api/v1/system/ws/notices"
    );

    // 复位，后续用例仍跑 http 分支
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        ...originalLocation,
        protocol: "http:",
        host: "localhost:3000",
        origin: "http://localhost:3000",
      },
    });
  });
});

describe("noticeWsUrl / dashboardWsUrl", () => {
  it("通知端点与后端注册的 /api/v1/system/ws/notices 一致", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/api/v1");
    expect(noticeWsUrl()).toBe("ws://localhost:3000/api/v1/system/ws/notices");
  });

  it("子路径部署时通知端点带 /xingran 前缀", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/api/v1");
    expect(noticeWsUrl()).toBe("ws://localhost:3000/xingran/api/v1/system/ws/notices");
  });

  it("仪表盘频道端点按 channel 拼接", () => {
    vi.stubEnv("VITE_API_BASE_URL", "/xingran/api/v1");
    expect(dashboardWsUrl("dashboard/42")).toBe(
      "ws://localhost:3000/xingran/api/v1/system/ws/dashboard/42"
    );
  });
});

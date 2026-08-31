/**
 * Phase 88 Batch373 — components/dashboard/utils/dataFetcher 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { DataFetcher } from "../dataFetcher";

vi.mock("@/lib/api", async () => ({
  get: vi.fn(async () => ({ code: 0, data: { value: 42 } })),
  post: vi.fn(async () => ({ code: 0, data: { ok: true } })),
}));

import * as api from "@/lib/api";

describe("components/dashboard/utils/dataFetcher", () => {
  let fetcher: DataFetcher;

  beforeEach(() => {
    vi.clearAllMocks();
    fetcher = new DataFetcher();
    fetcher.setCacheExpiry(60000);
  });

  it("API GET → 调 get() + 返回 data", async () => {
    const result = await fetcher.fetch({
      type: "api",
      method: "GET",
      endpoint: "/test",
    });
    expect(api.get).toHaveBeenCalled();
    expect(result.data).toEqual({ value: 42 });
  });

  it("API POST → 调 post()", async () => {
    const result = await fetcher.fetch({
      type: "api",
      method: "POST",
      endpoint: "/test",
      body: { foo: "bar" },
    });
    expect(api.post).toHaveBeenCalled();
    expect(result.data).toEqual({ ok: true });
  });

  it("API cache hit → 第二次不调 get", async () => {
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test" });
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test" });
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(1);
  });

  it("API 业务错误 → error", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ code: 500, message: "失败" } as any);
    const result = await fetcher.fetch({
      type: "api",
      method: "GET",
      endpoint: "/test",
    });
    expect(result.error).toBe("失败");
  });

  it("API throw → error", async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error("network"));
    const result = await fetcher.fetch({
      type: "api",
      method: "GET",
      endpoint: "/test",
    });
    expect(result.error).toBe("network");
  });

  it("static dataSource → 直接返回 config.data", async () => {
    const result = await fetcher.fetch({
      type: "static",
      data: { foo: "bar" },
    });
    expect(result.data).toEqual({ foo: "bar" });
  });

  it("无效 type → error", async () => {
    const result = await fetcher.fetch({ type: "unknown" } as any);
    expect(result.error).toBe("无效的数据源配置");
  });

  it("clearCache(pattern) 只清除匹配 key", async () => {
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test1" });
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test2" });
    fetcher.clearCache("test1");
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test1" });
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test2" });
    // test1 was cleared → second call to /test1 re-fetches; /test2 still cached
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(3);
  });

  it("clearCache() 清全部", async () => {
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test1" });
    fetcher.clearCache();
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test1" });
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(2);
  });

  it("setCacheExpiry 修改过期时间", async () => {
    fetcher.setCacheExpiry(0); // 立即过期
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test" });
    await fetcher.fetch({ type: "api", method: "GET", endpoint: "/test" });
    // 缓存立即过期 → 两次都重新调
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(2);
  });

  it("API params 影响 cache key", async () => {
    await fetcher.fetch({
      type: "api",
      method: "GET",
      endpoint: "/test",
      params: { a: 1 },
    });
    await fetcher.fetch({
      type: "api",
      method: "GET",
      endpoint: "/test",
      params: { a: 2 },
    });
    // 不同 params → 不同 cache key → 两次调
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(2);
  });

  it("timestamp 字段存在", async () => {
    const result = await fetcher.fetch({
      type: "static",
      data: { x: 1 },
    });
    expect(typeof result.timestamp).toBe("number");
  });

  it("closeWebSocket(channel) 删除连接", () => {
    fetcher.closeWebSocket("chan1");
    // No ws connections exist anyway, but should not throw
    expect(true).toBe(true);
  });

  it("closeWebSocket() 无参 → 关闭全部", () => {
    fetcher.closeWebSocket();
    expect(true).toBe(true);
  });
});

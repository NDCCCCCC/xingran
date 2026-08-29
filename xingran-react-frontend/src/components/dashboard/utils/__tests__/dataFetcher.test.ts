/**
 * Phase 88 Batch85 — dashboard/utils/dataFetcher 测试(65 stmts, 10.8% → 高)
 */
import { describe, it, expect, beforeEach } from "vitest";
import { DataFetcher } from "../dataFetcher";
import { createApiMock } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("DataFetcher", () => {
  let fetcher: DataFetcher;

  beforeEach(() => {
    fetcher = new DataFetcher();
  });

  describe("fetchFromAPI via fetch()", () => {
    it("GET 成功 + 标准 code=0 响应", async () => {
      const api = createApiMock("/test/list");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { list: [1, 2, 3] } } as any);
      const result = await fetcher.fetch({
        type: "api",
        method: "GET",
        endpoint: "/test/list",
      });
      expect(result.data).toEqual({ list: [1, 2, 3] });
      expect(result.error).toBeUndefined();
    });

    it("POST 成功 → 调用 endpoint", async () => {
      const api = createApiMock("/test/post");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { ok: true } } as any);
      const result = await fetcher.fetch({
        type: "api",
        method: "POST",
        endpoint: "/test/post",
        body: { foo: "bar" },
      });
      expect(result.data).toEqual({ ok: true });
      expect(api.endpoint).toHaveBeenCalled();
    });

    it("POST 无 body 时 fallback params", async () => {
      const api = createApiMock("/x");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: {} } as any);
      await fetcher.fetch({
        type: "api",
        method: "POST",
        endpoint: "/x",
        params: { a: 1 },
      });
      expect(api.endpoint).toHaveBeenCalledWith("/x", { a: 1 });
    });

    it("业务错误 code != 0 → result.error", async () => {
      const api = createApiMock("/err");
      api.endpoint.mockResolvedValueOnce({ code: 500, message: "服务器错误" } as any);
      const result = await fetcher.fetch({
        type: "api",
        method: "GET",
        endpoint: "/err",
      });
      expect(result.data).toBeNull();
      expect(result.error).toBe("服务器错误");
    });

    it("请求抛错 → catch 路径", async () => {
      const api = createApiMock("/x");
      api.endpoint.mockRejectedValueOnce(new Error("network"));
      const result = await fetcher.fetch({
        type: "api",
        method: "GET",
        endpoint: "/x",
      });
      expect(result.error).toBe("network");
    });

    it("缓存命中: 第二次调用不发起请求", async () => {
      const api = createApiMock("/cached");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { v: 1 } } as any);
      const cfg = { type: "api" as const, method: "GET" as const, endpoint: "/cached" };
      const r1 = await fetcher.fetch(cfg);
      const r2 = await fetcher.fetch(cfg);
      expect(r1.data).toEqual({ v: 1 });
      expect(r2.data).toEqual({ v: 1 });
      expect(api.endpoint).toHaveBeenCalledTimes(1);
    });

    it("响应无 code 字段 → 直接返回 response", async () => {
      const api = createApiMock("/raw");
      api.endpoint.mockResolvedValueOnce({ raw: "data" } as any);
      const result = await fetcher.fetch({
        type: "api",
        method: "GET",
        endpoint: "/raw",
      });
      expect(result.data).toEqual({ raw: "data" });
    });

    it("响应 code=0 但 data=undefined → 走 error 路径", async () => {
      const api = createApiMock("/nodata");
      api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
      const result = await fetcher.fetch({
        type: "api",
        method: "GET",
        endpoint: "/nodata",
      });
      expect(result.data).toBeNull();
      expect(result.error).toBe("API请求失败");
    });
  });

  describe("fetch() 路由分发", () => {
    it("type=static → 静态数据直接返回", async () => {
      const result = await fetcher.fetch({
        type: "static",
        data: { foo: "bar" },
      });
      expect(result.data).toEqual({ foo: "bar" });
    });

    it("未知 type → 返回 error", async () => {
      const result = await fetcher.fetch({
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        type: "unknown" as any,
      });
      expect(result.error).toBe("无效的数据源配置");
    });
  });

  describe("clearCache", () => {
    it("无 pattern → 清空全部", async () => {
      const api = createApiMock("/x");
      api.endpoint.mockResolvedValue({ code: 0, data: {} } as any);
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/x" });
      fetcher.clearCache();
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/x" });
      expect(api.endpoint).toHaveBeenCalledTimes(2);
    });

    it("pattern 匹配 → 清空匹配项", async () => {
      const apiKeep = createApiMock("/keep");
      const apiDrop = createApiMock("/drop");
      apiKeep.endpoint.mockResolvedValue({ code: 0, data: {} } as any);
      apiDrop.endpoint.mockResolvedValue({ code: 0, data: {} } as any);
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/keep" });
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/drop" });
      fetcher.clearCache("drop");
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/keep" });
      await fetcher.fetch({ type: "api", method: "GET", endpoint: "/drop" });
      expect(apiKeep.endpoint).toHaveBeenCalledTimes(1);
      expect(apiDrop.endpoint).toHaveBeenCalledTimes(2);
    });
  });

  describe("setCacheExpiry", () => {
    it("设置短过期 → 立即失效", async () => {
      const api = createApiMock("/x");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { v: 1 } } as any);
      fetcher.setCacheExpiry(0);
      const cfg = { type: "api" as const, method: "GET" as const, endpoint: "/x" };
      await fetcher.fetch(cfg);
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { v: 2 } } as any);
      const r2 = await fetcher.fetch(cfg);
      expect(r2.data).toEqual({ v: 2 });
      expect(api.endpoint).toHaveBeenCalledTimes(2);
    });
  });

  describe("closeWebSocket", () => {
    it("无参数 → 不抛错", () => {
      fetcher.closeWebSocket();
      expect(true).toBe(true);
    });

    it("指定 channel → 不抛错", () => {
      fetcher.closeWebSocket("nonexistent");
      expect(true).toBe(true);
    });
  });
});

/**
 * Phase 88 Batch91 — operations/building-spaces-3d hooks/useGeocoding 测试(41 stmts, 17.1% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useGeocoding } from "../useGeocoding";
import { createApiMock } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("useGeocoding", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useGeocoding());
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("geocode: 空地址 → error & return null", async () => {
    const { result } = renderHook(() => useGeocoding());
    let r: any;
    await act(async () => {
      r = await result.current.geocode("");
    });
    expect(r).toBeNull();
    expect(result.current.error?.message).toBe("地址不能为空");
  });

  it("geocode: 缓存命中 → 不发请求", async () => {
    const api = createApiMock("/ops/building/geocode");
    // 第一次调用 → 缓存 miss
    api.endpoint.mockResolvedValueOnce({
      data: {
        longitude: 114.4,
        latitude: 30.5,
        formattedAddress: "武汉",
        province: "湖北",
        city: "武汉",
        district: "洪山",
        street: "珞喻路",
      },
    } as any);

    const { result } = renderHook(() => useGeocoding());

    let r1: any;
    await act(async () => {
      r1 = await result.current.geocode("武汉", "武汉");
    });
    expect(r1?.longitude).toBe(114.4);
    expect(api.endpoint).toHaveBeenCalledTimes(1);

    // 第二次相同地址 → 应该命中缓存,不再调用 API
    let r2: any;
    await act(async () => {
      r2 = await result.current.geocode("武汉", "武汉");
    });
    expect(r2?.longitude).toBe(114.4);
    expect(api.endpoint).toHaveBeenCalledTimes(1);
  });

  it("geocode: 成功 → 返回坐标", async () => {
    const api = createApiMock("/ops/building/geocode");
    api.endpoint.mockResolvedValueOnce({
      data: {
        longitude: 116.4,
        latitude: 39.9,
        formattedAddress: "北京",
        province: "北京",
        city: "北京",
        district: "朝阳区",
        street: "长安街",
      },
    } as any);

    const { result } = renderHook(() => useGeocoding());
    let r: any;
    await act(async () => {
      r = await result.current.geocode("北京");
    });
    expect(r?.longitude).toBe(116.4);
    expect(r?.formattedAddress).toBe("北京");
  });

  it("geocode: response.data 缺失 → error & null", async () => {
    const api = createApiMock("/ops/building/geocode");
    api.endpoint.mockResolvedValueOnce({ data: null } as any);

    const { result } = renderHook(() => useGeocoding());
    let r: any;
    await act(async () => {
      r = await result.current.geocode("未知地址xyz");
    });
    expect(r).toBeNull();
    expect(result.current.error?.message).toBe("地址解析失败");
  });

  it("geocode: 抛错 → catch 路径", async () => {
    const api = createApiMock("/ops/building/geocode");
    api.endpoint.mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useGeocoding());
    let r: any;
    await act(async () => {
      r = await result.current.geocode("上海");
    });
    expect(r).toBeNull();
    expect(result.current.error?.message).toBe("net");
    expect(result.current.loading).toBe(false);
  });
});

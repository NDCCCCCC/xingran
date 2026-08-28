/**
 * Phase 88 Batch25 — buildings useGeocodingForm 钩子补测
 *
 * 补齐 hooks.test.tsx 未覆盖的 useGeocodingForm(3 hook 中唯一无测试的)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const geocodeMock = vi.fn();

vi.mock("@/pages/operations/building-spaces-3d/hooks/useGeocoding", () => ({
  useGeocoding: () => ({ geocode: geocodeMock }),
}));

import { useGeocodingForm } from "../useGeocodingForm";

describe("useGeocodingForm", () => {
  beforeEach(() => {
    geocodeMock.mockReset();
  });

  it("initial state 全空", () => {
    const { result } = renderHook(() => useGeocodingForm());
    expect(result.current.geocodingLoading).toBe(false);
    expect(result.current.geocodingResult).toBeNull();
    expect(result.current.geocodingWarning).toBeNull();
  });

  it("handleGeocode 空地址直接清空不调用 api", async () => {
    const { result } = renderHook(() => useGeocodingForm());
    await act(async () => {
      await result.current.handleGeocode("   ");
    });
    expect(geocodeMock).not.toHaveBeenCalled();
    expect(result.current.geocodingResult).toBeNull();
    expect(result.current.geocodingWarning).toBeNull();
  });

  it("handleGeocode 成功路径写入 result", async () => {
    geocodeMock.mockResolvedValue({
      longitude: 114.3,
      latitude: 30.6,
      formattedAddress: "湖北省武汉市",
    });
    const { result } = renderHook(() => useGeocodingForm());

    await act(async () => {
      await result.current.handleGeocode("武汉市");
    });

    expect(geocodeMock).toHaveBeenCalledWith("武汉市");
    expect(result.current.geocodingLoading).toBe(false);
    expect(result.current.geocodingResult).toEqual({
      longitude: 114.3,
      latitude: 30.6,
      formattedAddress: "湖北省武汉市",
    });
    expect(result.current.geocodingWarning).toBeNull();
  });

  it("handleGeocode null result 写 warning", async () => {
    geocodeMock.mockResolvedValue(null);
    const { result } = renderHook(() => useGeocodingForm());

    await act(async () => {
      await result.current.handleGeocode("不存在地址");
    });

    expect(result.current.geocodingResult).toBeNull();
    expect(result.current.geocodingWarning).toBe("地址解析失败，将保存不含经纬度的信息");
    expect(result.current.geocodingLoading).toBe(false);
  });

  it("handleGeocode throw 写 warning + finally 清 loading", async () => {
    geocodeMock.mockRejectedValue(new Error("network"));
    const { result } = renderHook(() => useGeocodingForm());

    await act(async () => {
      await result.current.handleGeocode("任意地址");
    });

    expect(result.current.geocodingResult).toBeNull();
    expect(result.current.geocodingWarning).toBe("地址解析出错，将保存不含经纬度的信息");
    expect(result.current.geocodingLoading).toBe(false);
  });

  it("resetGeocodingState 清空全部", async () => {
    geocodeMock.mockResolvedValue({ longitude: 1, latitude: 2, formattedAddress: "x" });
    const { result } = renderHook(() => useGeocodingForm());

    await act(async () => {
      await result.current.handleGeocode("addr");
    });
    expect(result.current.geocodingResult).not.toBeNull();

    act(() => {
      result.current.resetGeocodingState();
    });
    expect(result.current.geocodingResult).toBeNull();
    expect(result.current.geocodingWarning).toBeNull();
    expect(result.current.geocodingLoading).toBe(false);
  });

  it("setGeocodingResultFromRecord 有经纬度写入", () => {
    const { result } = renderHook(() => useGeocodingForm());
    act(() => {
      result.current.setGeocodingResultFromRecord({ longitude: 100.1, latitude: 20.2 });
    });
    expect(result.current.geocodingResult).toEqual({ longitude: 100.1, latitude: 20.2 });
  });

  it("setGeocodingResultFromRecord 缺经纬度不写入", () => {
    const { result } = renderHook(() => useGeocodingForm());
    act(() => {
      result.current.setGeocodingResultFromRecord({ longitude: 100.1 });
    });
    expect(result.current.geocodingResult).toBeNull();
    act(() => {
      result.current.setGeocodingResultFromRecord({});
    });
    expect(result.current.geocodingResult).toBeNull();
  });
});

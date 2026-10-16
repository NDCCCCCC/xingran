/**
 * Phase 88 Batch364 — components/reconciliation/hooks/useAssetHealth 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";

let mockData: any = { assets: [] };

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../useWorkstationHealth", () => ({
  useWorkstationHealth: vi.fn(() => ({ data: mockData, loading: false })),
}));

import { useAssetHealth } from "../useAssetHealth";

describe("components/reconciliation/hooks/useAssetHealth", () => {
  it("assetId 命中 assets 数组", () => {
    mockData = {
      assets: [
        { assetId: "a1", name: "Asset 1" },
        { assetId: "a2", name: "Asset 2" },
      ],
    };
    const { result } = renderHook(() => useAssetHealth("a1", "w1"));
    expect(result.current).toEqual({ assetId: "a1", name: "Asset 1" });
  });

  it("assetId 未命中 → undefined", () => {
    mockData = { assets: [{ assetId: "a1", name: "X" }] };
    const { result } = renderHook(() => useAssetHealth("xxx", "w1"));
    expect(result.current).toBeUndefined();
  });

  it("assetId=null → undefined", () => {
    mockData = { assets: [{ assetId: "a1" }] };
    const { result } = renderHook(() => useAssetHealth(null, "w1"));
    expect(result.current).toBeUndefined();
  });

  it("data 无 assets → undefined", () => {
    mockData = {};
    const { result } = renderHook(() => useAssetHealth("a1", "w1"));
    expect(result.current).toBeUndefined();
  });

  it("data=null → undefined", () => {
    mockData = null;
    const { result } = renderHook(() => useAssetHealth("a1", "w1"));
    expect(result.current).toBeUndefined();
  });
});

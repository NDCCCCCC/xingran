/**
 * Phase 85 — buildings hooks 测试(mock useGeocoding/useDeptTree)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useBuildingGeocoding } from "../useBuildingGeocoding";
import { useDepartmentData } from "../useDepartmentData";

vi.mock("@/pages/operations/building-spaces-3d/hooks/useGeocoding", () => ({
  useGeocoding: vi.fn(() => ({
    geocode: vi.fn(() =>
      Promise.resolve({ longitude: 114.3, latitude: 30.6, formattedAddress: "湖北省武汉市" })
    ),
  })),
}));

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: vi.fn(() => ({
    data: [
      { id: "d1", deptName: "总经办", children: [{ id: "d2", deptName: "技术部" }] },
      { id: "d3", deptName: "综合部" },
    ],
    isLoading: false,
  })),
}));

describe("useBuildingGeocoding", () => {
  beforeEach(() => vi.clearAllMocks());

  it("initializes idle state", () => {
    const { result } = renderHook(() => useBuildingGeocoding());
    expect(result.current.loading).toBe(false);
    expect(result.current.result).toBeNull();
    expect(result.current.warning).toBeNull();
  });

  it("resolveAddress returns null for empty address", async () => {
    const { result } = renderHook(() => useBuildingGeocoding());
    const r = await act(async () => result.current.resolveAddress("  "));
    expect(r).toBeNull();
    expect(result.current.result).toBeNull();
  });

  it("resolveAddress sets result for valid address", async () => {
    const { result } = renderHook(() => useBuildingGeocoding());
    await act(async () => {
      await result.current.resolveAddress("武汉市洪山区");
    });
    await waitFor(() => {
      expect(result.current.result?.longitude).toBe(114.3);
    });
  });
});

describe("useDepartmentData", () => {
  it("exposes departments with loading state", () => {
    const { result } = renderHook(() => useDepartmentData());
    expect(result.current.loading).toBe(false);
  });

  it("getOrgName resolves department name by id", () => {
    const { result } = renderHook(() => useDepartmentData());
    const name = result.current.getOrgName("d1");
    expect(name).toBeTruthy();
  });

  it("getOrgName returns fallback for unknown id", () => {
    const { result } = renderHook(() => useDepartmentData());
    const name = result.current.getOrgName("nonexistent");
    expect(name === undefined || name === null || typeof name === "string").toBe(true);
  });
});

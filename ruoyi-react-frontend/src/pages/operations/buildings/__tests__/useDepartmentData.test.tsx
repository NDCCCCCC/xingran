/**
 * Phase 88 Batch296 — pages/operations/buildings/useDepartmentData 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockData: any = [
  { id: "d1", deptName: "Dept 1", children: [{ id: "d2", deptName: "Dept 2" }] },
];
vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: vi.fn(() => ({ data: mockData, isLoading: false })),
}));

import { useDepartmentData } from "../useDepartmentData";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("operations/buildings/useDepartmentData", () => {
  it("返回 departments + loading", () => {
    const { result } = renderHook(() => useDepartmentData(), { wrapper });
    expect(result.current.departments).toBe(mockData);
    expect(result.current.loading).toBe(false);
  });

  it("loadDepartments no-op", () => {
    const { result } = renderHook(() => useDepartmentData(), { wrapper });
    expect(() => result.current.loadDepartments()).not.toThrow();
  });

  it("getOrgName 已知 id", () => {
    const { result } = renderHook(() => useDepartmentData(), { wrapper });
    expect(result.current.getOrgName("d1")).toBe("Dept 1");
    expect(result.current.getOrgName("d2")).toBe("Dept 2");
  });

  it("getOrgName 未知 id → '-'", () => {
    const { result } = renderHook(() => useDepartmentData(), { wrapper });
    expect(result.current.getOrgName("xxx")).toBe("-");
  });

  it("getOrgName undefined → '-'", () => {
    const { result } = renderHook(() => useDepartmentData(), { wrapper });
    expect(result.current.getOrgName(undefined)).toBe("-");
  });
});

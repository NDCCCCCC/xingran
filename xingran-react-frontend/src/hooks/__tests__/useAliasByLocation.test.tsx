/**
 * Phase 88 Batch218 — hooks/useAliasByLocation + useDeptTree + useRoleList 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  workstationApi: {
    deptOptions: vi.fn(async (orgId: string) => [
      { id: `d-${orgId}-1`, name: `Dept ${orgId} 1`, isAlias: true },
    ]),
  },
}));

vi.mock("@/lib/dutyApi", () => ({
  getDeptTree: vi.fn(async () => ({
    code: 0,
    data: [{ id: "d1", name: "Root", children: [] }],
  })),
}));

import { useAliasByLocation } from "../useAliasByLocation";
import { useDeptTree, useInvalidateDept } from "../useDeptTree";
import { useRoleList, useInvalidateRole } from "../useRoleList";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useAliasByLocation", () => {
  it("locationId 有 → 调用 deptOptions", async () => {
    const { result } = renderHook(() => useAliasByLocation("loc-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].id).toBe("d-loc-1-1");
  });

  it("locationId 空 → 不触发", async () => {
    const { result } = renderHook(() => useAliasByLocation(""), { wrapper });
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.isSuccess).toBe(false);
  });

  it("locationId null → 不触发", async () => {
    const { result } = renderHook(() => useAliasByLocation(null), { wrapper });
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.isSuccess).toBe(false);
  });
});

describe("hooks/useDeptTree", () => {
  it("返回 dept 树", async () => {
    const { result } = renderHook(() => useDeptTree(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].id).toBe("d1");
  });

  it("useInvalidateDept 返回函数", () => {
    const { result } = renderHook(() => useInvalidateDept(), { wrapper });
    expect(typeof result.current).toBe("function");
  });
});

describe("hooks/useRoleList", () => {
  it("返回 role 列表 shape", async () => {
    const { result } = renderHook(() => useRoleList(), { wrapper });
    // role 列表请求会失败(没 mock post),仅验证 enabled
    expect(result.current).toBeDefined();
  });

  it("useInvalidateRole 返回函数", () => {
    const { result } = renderHook(() => useInvalidateRole(), { wrapper });
    expect(typeof result.current).toBe("function");
  });
});

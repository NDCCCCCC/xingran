/**
 * Phase 88 Batch365 — components/reconciliation/hooks/useWorkstationHealth 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

let mockVisible = true;

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../useReconciliationVisibility", () => ({
  useReconciliationVisibility: vi.fn(() => mockVisible),
}));

vi.mock("@/lib/assetApi", () => ({
  reconciliationApi: {
    byWorkstation: vi.fn(async () => ({ workstation: "w1", healthScore: 90 })),
  },
}));

import { useWorkstationHealth } from "../useWorkstationHealth";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("components/reconciliation/hooks/useWorkstationHealth", () => {
  beforeEach(() => {
    mockVisible = true;
  });

  it("visible=true + workstationId → 调用 queryFn", async () => {
    const { result } = renderHook(() => useWorkstationHealth("w1"), { wrapper });
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
  });

  it("visible=false → enabled=false → 不调 queryFn", async () => {
    mockVisible = false;
    const { result } = renderHook(() => useWorkstationHealth("w1"), { wrapper });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it("workstationId 为空 → 不调 queryFn", async () => {
    const { result } = renderHook(() => useWorkstationHealth(""), { wrapper });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it("返回 data 含 healthScore", async () => {
    const { result } = renderHook(() => useWorkstationHealth("w1"), { wrapper });
    await waitFor(() => {
      expect(result.current.data).toEqual({ workstation: "w1", healthScore: 90 });
    });
  });

  it("shape 包含 isLoading/isError/data", () => {
    const { result } = renderHook(() => useWorkstationHealth("w1"), { wrapper });
    expect(typeof result.current.isLoading).toBe("boolean");
    expect(typeof result.current.isError).toBe("boolean");
    expect(result.current.data === undefined || typeof result.current.data === "object").toBe(true);
  });
});

/**
 * Phase 88 Batch279 — components/reconciliation/hooks/useExceptionMatch 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockTest = vi.fn();
vi.mock("@/lib/assetApi", () => ({
  reconciliationApi: {
    exceptionRule: {
      test: (...args: any[]) => mockTest(...args),
    },
  },
}));

import { useExceptionMatch } from "../useExceptionMatch";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("reconciliation/hooks/useExceptionMatch", () => {
  it("ip 有 → 调用 test", async () => {
    mockTest.mockResolvedValue({
      matchedRules: [],
      mergedActions: [],
      finalSeverity: "low",
      isSilence: false,
      needsUserDept: false,
    });
    const { result } = renderHook(() => useExceptionMatch({ ip: "10.0.0.1" }), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockTest).toHaveBeenCalled();
  });

  it("ip 空 → 不触发", async () => {
    mockTest.mockReset();
    const { result } = renderHook(() => useExceptionMatch({ ip: "" }), { wrapper });
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.isSuccess).toBe(false);
    expect(mockTest).not.toHaveBeenCalled();
  });

  it("传 conflictType", async () => {
    mockTest.mockReset();
    mockTest.mockResolvedValue({});
    const { result } = renderHook(
      () => useExceptionMatch({ ip: "10.0.0.1", conflictType: "ip_conflict" }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockTest).toHaveBeenCalledWith({ ip: "10.0.0.1" });
  });
});

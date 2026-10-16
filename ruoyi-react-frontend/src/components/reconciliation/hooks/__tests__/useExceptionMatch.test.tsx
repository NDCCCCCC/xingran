/**
 * Phase 88 Batch366 — components/reconciliation/hooks/useExceptionMatch 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/assetApi", () => ({
  reconciliationApi: {
    exceptionRule: {
      test: vi.fn(async ({ ip }: any) => ({
        matchedRules: [],
        mergedActions: ["allow"],
        finalSeverity: "info",
        isSilence: true,
        needsUserDept: false,
        ip,
      })),
    },
  },
}));

import { useExceptionMatch } from "../useExceptionMatch";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("components/reconciliation/hooks/useExceptionMatch", () => {
  it("返回 shape", () => {
    const { result } = renderHook(() => useExceptionMatch({ ip: "10.0.0.1" }), { wrapper });
    expect(typeof result.current.isError).toBe("boolean");
    expect(typeof result.current.isLoading).toBe("boolean");
  });

  it("enabled=true 时调 queryFn", async () => {
    const { result } = renderHook(() => useExceptionMatch({ ip: "10.0.0.1" }), { wrapper });
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    expect(result.current.data?.ip).toBe("10.0.0.1");
  });

  it("ip 为空 → enabled=false", () => {
    const { result } = renderHook(() => useExceptionMatch({ ip: "" }), { wrapper });
    expect(result.current.isLoading).toBe(false);
  });

  it("含 conflictType 不报错", async () => {
    const { result } = renderHook(() => useExceptionMatch({ ip: "10.0.0.2", conflictType: "ip" }), {
      wrapper,
    });
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
  });

  it("返回 ExceptionMatchResult shape", async () => {
    const { result } = renderHook(() => useExceptionMatch({ ip: "10.0.0.3" }), { wrapper });
    await waitFor(() => {
      expect(result.current.data).toHaveProperty("matchedRules");
      expect(result.current.data).toHaveProperty("mergedActions");
      expect(result.current.data).toHaveProperty("finalSeverity");
      expect(result.current.data).toHaveProperty("isSilence");
      expect(result.current.data).toHaveProperty("needsUserDept");
    });
  });
});

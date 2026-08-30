/**
 * Phase 88 Batch199 — components/reconciliation/hooks/useWorkstationHealth 测试
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
    byWorkstation: vi.fn(async ({ workstationId }: any) => ({
      workstation: { id: workstationId },
      healthScore: 92,
      visible: true,
      assets: [],
    })),
  },
}));

vi.mock("../useReconciliationVisibility", () => ({
  useReconciliationVisibility: vi.fn(() => true),
}));

import { useWorkstationHealth } from "../useWorkstationHealth";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("reconciliation/hooks/useWorkstationHealth", () => {
  it("visible=true + workstationId → 调用 byWorkstation", async () => {
    const { result } = renderHook(() => useWorkstationHealth("ws-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.healthScore).toBe(92);
  });

  it("workstationId 空 → 不触发", async () => {
    const { result } = renderHook(() => useWorkstationHealth(""), { wrapper });
    // enabled=false → isPending, 不会变 success
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.isSuccess).toBe(false);
  });

  it("visible=false → 不触发", async () => {
    vi.mocked(
      await import("../useReconciliationVisibility")
    ).useReconciliationVisibility.mockReturnValueOnce(false);
    const { result } = renderHook(() => useWorkstationHealth("ws-2"), { wrapper });
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.isSuccess).toBe(false);
  });
});

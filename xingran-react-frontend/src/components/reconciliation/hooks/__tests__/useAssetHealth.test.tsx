/**
 * Phase 88 Batch248 — components/reconciliation/hooks/useAssetHealth 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockData: any = {
  workstation: { id: "ws1" },
  healthScore: 92,
  visible: true,
  assets: [
    { assetId: "a1", assetCode: "A1", healthScore: 95 },
    { assetId: "a2", assetCode: "A2", healthScore: 80 },
  ],
};

vi.mock("../useWorkstationHealth", () => ({
  useWorkstationHealth: vi.fn(() => ({ data: mockData })),
}));

vi.mock("../useReconciliationVisibility", () => ({
  useReconciliationVisibility: vi.fn(() => true),
}));

import { useAssetHealth } from "../useAssetHealth";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("reconciliation/hooks/useAssetHealth", () => {
  it("assetId 有 → 返回 asset", () => {
    const { result } = renderHook(() => useAssetHealth("a1", "ws1"), { wrapper });
    expect(result.current?.assetId).toBe("a1");
  });

  it("assetId = a2 → 返回 a2", () => {
    const { result } = renderHook(() => useAssetHealth("a2", "ws1"), { wrapper });
    expect(result.current?.healthScore).toBe(80);
  });

  it("assetId null → undefined", () => {
    const { result } = renderHook(() => useAssetHealth(null, "ws1"), { wrapper });
    expect(result.current).toBeUndefined();
  });

  it("assetId 找不到 → undefined", () => {
    const { result } = renderHook(() => useAssetHealth("a99", "ws1"), { wrapper });
    expect(result.current).toBeUndefined();
  });

  it("data 无 assets → undefined", async () => {
    const ws = await import("../useWorkstationHealth");
    vi.mocked(ws.useWorkstationHealth).mockReturnValueOnce({
      data: { workstation: {}, healthScore: 0, visible: true },
    } as any);
    const { result } = renderHook(() => useAssetHealth("a1", "ws1"), { wrapper });
    expect(result.current).toBeUndefined();
  });
});

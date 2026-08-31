/**
 * Phase 88 Batch255 — hooks/useDict + useInvalidateDict 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async () => ({
      data: {
        list: [
          { id: "1", dictLabel: "Male", dictValue: "0" },
          { id: "2", dictLabel: "Female", dictValue: "1" },
        ],
        total: 2,
      },
    })),
  };
});

import { useInvalidateDict } from "../useDict";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("hooks/useDict", () => {
  it("useInvalidateDict 返回函数", () => {
    const { result } = renderHook(() => useInvalidateDict(), { wrapper });
    expect(typeof result.current).toBe("function");
  });

  it("调用 invalidate 函数不抛错", () => {
    const { result } = renderHook(() => useInvalidateDict(), { wrapper });
    expect(() => result.current()).not.toThrow();
  });
});

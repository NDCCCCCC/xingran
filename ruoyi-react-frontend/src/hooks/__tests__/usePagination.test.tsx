/**
 * Phase 88 Batch386 — hooks/usePagination 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let persistedStore: any = {};
vi.mock("@/hooks/usePersistedState", () => ({
  usePersistedStateController: vi.fn(({ defaultValue }: any) => {
    const k = JSON.stringify(defaultValue);
    if (persistedStore[k] === undefined) persistedStore[k] = defaultValue;
    return [
      persistedStore[k],
      (v: any) => {
        persistedStore[k] = v;
      },
    ] as any;
  }),
}));

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn(() => ({
    preferences: { data: { defaultPageSize: 20 } },
  })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter initialEntries={["/test"]}>{children}</MemoryRouter>;
}

import { usePagination } from "../usePagination";

describe("hooks/usePagination", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    persistedStore = {};
  });

  it("返回 current/pageSize/total 数值", () => {
    const { result } = renderHook(() => usePagination(), { wrapper });
    expect(typeof result.current.current).toBe("number");
    expect(typeof result.current.pageSize).toBe("number");
    expect(typeof result.current.total).toBe("number");
  });

  it("返回所有操作方法", () => {
    const { result } = renderHook(() => usePagination(), { wrapper });
    expect(typeof result.current.setCurrent).toBe("function");
    expect(typeof result.current.setPageSize).toBe("function");
    expect(typeof result.current.setTotal).toBe("function");
    expect(typeof result.current.reset).toBe("function");
  });

  it("paginationProps 包含必要字段", () => {
    const { result } = renderHook(() => usePagination(), { wrapper });
    expect(typeof result.current.paginationProps).toBe("object");
    expect(typeof result.current.paginationProps.current).toBe("number");
    expect(typeof result.current.paginationProps.pageSize).toBe("number");
    expect(typeof result.current.paginationProps.total).toBe("number");
  });

  it("setTotal 是函数", () => {
    const { result } = renderHook(() => usePagination(), { wrapper });
    expect(typeof result.current.setTotal).toBe("function");
  });

  it("options.pageSize 覆盖默认值", () => {
    const { result } = renderHook(() => usePagination({ pageSize: 50 }), { wrapper });
    expect(result.current.pageSize).toBe(50);
  });

  it("reset 是函数", () => {
    const { result } = renderHook(() => usePagination(), { wrapper });
    expect(typeof result.current.reset).toBe("function");
  });
});

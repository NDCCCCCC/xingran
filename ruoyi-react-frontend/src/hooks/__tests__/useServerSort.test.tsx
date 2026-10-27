/**
 * Phase 88 Batch380 — hooks/useServerSort + resolveSorter 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const persistedStore: unknown = {};
vi.mock("@/hooks/usePersistedState", () => ({
  usePersistedStateController: vi.fn(({ defaultValue }: unknown) => {
    const k = JSON.stringify(defaultValue);
    if (persistedStore[k] === undefined) persistedStore[k] = defaultValue;
    return [
      persistedStore[k],
      (v: unknown) => {
        persistedStore[k] = v;
      },
    ] as unknown;
  }),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter initialEntries={["/test"]}>{children}</MemoryRouter>;
}

import { useServerSort, resolveSorter } from "../useServerSort";

describe("hooks/useServerSort", () => {
  it("导出 shape", () => {
    const { result } = renderHook(() => useServerSort(), { wrapper });
    expect(typeof result.current.handleTableChange).toBe("function");
    expect(typeof result.current.resetSort).toBe("function");
  });

  it("sortOrder 初始为 null", () => {
    const { result } = renderHook(() => useServerSort(), { wrapper });
    expect(result.current.sortOrder).toBeNull();
  });
});

describe("resolveSorter", () => {
  const metas: unknown = [{ field: "name" }, { field: "age" }];

  it("ascend", () => {
    const r = resolveSorter({ field: "name", order: "ascend" } as unknown, metas);
    expect(r.orderByColumn).toBe("name");
    expect(r.isAsc).toBe(true);
  });

  it("descend", () => {
    const r = resolveSorter({ field: "age", order: "descend" } as unknown, metas);
    expect(r.orderByColumn).toBe("age");
    expect(r.isAsc).toBe(false);
  });

  it("no order → undefined", () => {
    const r = resolveSorter({ field: "name", order: null } as unknown, metas);
    expect(r.orderByColumn).toBeUndefined();
  });

  it("empty sorter → undefined", () => {
    const r = resolveSorter({} as unknown, metas);
    expect(r.orderByColumn).toBeUndefined();
  });

  it("数组 sorter → 取最后", () => {
    const r = resolveSorter(
      [
        { field: "name", order: "ascend" },
        { field: "age", order: "descend" },
      ] as unknown,
      metas
    );
    expect(r.orderByColumn).toBe("age");
  });

  it("未知字段 → undefined", () => {
    const r = resolveSorter({ field: "unknown", order: "ascend" } as unknown, metas);
    expect(r.orderByColumn).toBeUndefined();
  });
});

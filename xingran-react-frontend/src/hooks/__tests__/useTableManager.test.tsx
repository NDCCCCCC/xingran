/**
 * Phase 88 Batch383 — hooks/useTableManager 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/constants/storage", () => ({
  TABLE_STATE_PREFIX: "table_state:",
  sanitizePathForKey: vi.fn((p: string) => p),
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

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

vi.mock("@/hooks/useServerSort", () => ({
  useServerSort: vi.fn(() => ({
    handleTableChange: vi.fn(),
    resetSort: vi.fn(),
    sortOrder: null,
    orderByColumn: undefined,
    isAsc: undefined,
  })),
  resolveSorter: vi.fn(() => ({ orderByColumn: undefined, isAsc: undefined })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter initialEntries={["/test"]}>{children}</MemoryRouter>;
}

import { useTableManager } from "../useTableManager";

const mockLoadFn = vi.fn(async () => ({
  list: [{ id: "1", name: "item1" }],
  total: 1,
}));

describe("hooks/useTableManager", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    persistedStore = {};
    mockLoadFn.mockResolvedValue({ list: [], total: 0 });
  });

  it("返回必要状态字段", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(typeof result.current.loading).toBe("boolean");
    expect(Array.isArray(result.current.data)).toBe(true);
    expect(typeof result.current.total).toBe("number");
    expect(typeof result.current.current).toBe("number");
    expect(typeof result.current.pageSize).toBe("number");
    expect(Array.isArray(result.current.selectedRowKeys)).toBe(true);
  });

  it("返回表单实例", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(result.current.searchForm).toBeDefined();
    expect(result.current.editForm).toBeDefined();
  });

  it("返回 editModal 状态", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(typeof result.current.editModalVisible).toBe("boolean");
    expect(result.current.editingItem).toBeNull();
  });

  it("返回所有操作方法", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(typeof result.current.handleSearch).toBe("function");
    expect(typeof result.current.applyFilters).toBe("function");
    expect(typeof result.current.handleReset).toBe("function");
    expect(typeof result.current.handleRefresh).toBe("function");
    expect(typeof result.current.handleAdd).toBe("function");
    expect(typeof result.current.handleEdit).toBe("function");
    expect(typeof result.current.handleModalClose).toBe("function");
    expect(typeof result.current.loadData).toBe("function");
    expect(typeof result.current.resetSelection).toBe("function");
  });

  it("返回排序相关字段", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(result.current.orderByColumn).toBeUndefined();
    expect(result.current.isAsc).toBeUndefined();
    expect(typeof result.current.getColumnSortOrder).toBe("function");
    expect(typeof result.current.handleTableChange).toBe("function");
    expect(typeof result.current.resetSort).toBe("function");
  });

  it("handleAdd 不抛错", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(() => result.current.handleAdd()).not.toThrow();
  });

  it("handleEdit 不抛错", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(() => result.current.handleEdit({ id: "1", name: "test" })).not.toThrow();
  });

  it("handleModalClose 不抛错", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(() => result.current.handleModalClose()).not.toThrow();
  });

  it("resetSelection 不抛错", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(() => result.current.resetSelection()).not.toThrow();
  });

  it("loadData 调用 loadFunction", async () => {
    mockLoadFn.mockResolvedValue({ list: [{ id: "1" }], total: 1 });
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    await result.current.loadData();
    expect(mockLoadFn).toHaveBeenCalled();
  });

  it("loadData 失败不抛错", async () => {
    mockLoadFn.mockRejectedValue(new Error("net"));
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    await result.current.loadData();
    expect(Array.isArray(result.current.data)).toBe(true);
  });

  it("handleSearch 不抛错", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(() => result.current.handleSearch()).not.toThrow();
  });

  it("setData / setTotal / setLoading 函数存在", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(typeof result.current.setData).toBe("function");
    expect(typeof result.current.setTotal).toBe("function");
    expect(typeof result.current.setLoading).toBe("function");
  });

  it("setEditModalVisible / setEditingItem 函数存在", () => {
    const { result } = renderHook(() => useTableManager(mockLoadFn), { wrapper });
    expect(typeof result.current.setEditModalVisible).toBe("function");
    expect(typeof result.current.setEditingItem).toBe("function");
  });
});

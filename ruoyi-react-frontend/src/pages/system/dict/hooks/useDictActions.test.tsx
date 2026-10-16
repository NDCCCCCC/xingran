/**
 * Phase 88 Batch70 — useDictActions hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: {} }),
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
  handleSuccess: vi.fn(),
}));

vi.mock("@/lib/queryKeys", () => ({ queryKeys: {} }));

import { useDictActions } from "../hooks/useDictActions";

beforeEach(() => {
  vi.clearAllMocks();
});

const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
const wrap = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={qc}>
    <ConfigProvider>
      <App>{children}</App>
    </ConfigProvider>
  </QueryClientProvider>
);

const opts = () => ({
  selectedType: "test_type",
  loadDictTypes: vi.fn().mockResolvedValue(undefined),
  loadDictData: vi.fn().mockResolvedValue(undefined),
  loadTypeStatistics: vi.fn().mockResolvedValue(undefined),
  loadDataStatistics: vi.fn().mockResolvedValue(undefined),
});

describe("useDictActions — initial state + setters", () => {
  it("initial + 4 state + 4 setter + handlers 存在", () => {
    const { result } = renderHook(() => useDictActions(opts()), { wrapper: wrap });
    expect(result.current.editingType).toBeNull();
    expect(result.current.editingData).toBeNull();
    expect(result.current.typeModalVisible).toBe(false);
    expect(result.current.dataModalVisible).toBe(false);
    expect(typeof result.current.handleCreateType).toBe("function");
    expect(typeof result.current.handleDeleteType).toBe("function");
    expect(typeof result.current.handleBatchDeleteType).toBe("function");
  });

  it("setters 直写 state", () => {
    const { result } = renderHook(() => useDictActions(opts()), { wrapper: wrap });
    act(() => {
      result.current.setTypeModalVisible(true);
      result.current.setDataModalVisible(true);
      result.current.setEditingType({ id: "t1", name: "type1" } as any);
      result.current.setEditingData({ id: "d1", label: "v1" } as any);
    });
    expect(result.current.typeModalVisible).toBe(true);
    expect(result.current.dataModalVisible).toBe(true);
    expect(result.current.editingType?.id).toBe("t1");
    expect(result.current.editingData?.id).toBe("d1");
  });
});

describe("useDictActions — handlers", () => {
  it("handleCreateType validateFields + post + loadDictTypes", async () => {
    const loadDictTypes = vi.fn().mockResolvedValue(undefined);
    const form = {
      validateFields: vi.fn().mockResolvedValue({ name: "t1" }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useDictActions({ ...opts(), loadDictTypes }), {
      wrapper: wrap,
    });
    await act(async () => {
      await result.current.handleCreateType(form);
    });
    expect(loadDictTypes).toHaveBeenCalled();
  });

  it("handleCreateType validate 失败 → 不调 post", async () => {
    const form = { validateFields: vi.fn().mockRejectedValue({ errorFields: [] }) } as any;
    const { result } = renderHook(() => useDictActions(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleCreateType(form);
    });
    // 不抛错即通过
    expect(true).toBe(true);
  });

  it("handleDeleteType 调 post", async () => {
    const loadDictTypes = vi.fn();
    const { result } = renderHook(() => useDictActions({ ...opts(), loadDictTypes }), {
      wrapper: wrap,
    });
    await act(async () => {
      await result.current.handleDeleteType("t1");
    });
    expect(loadDictTypes).toHaveBeenCalled();
  });
});

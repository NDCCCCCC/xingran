/**
 * Phase 88 Batch115 — system/dict/hooks/useDictActions 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useDictActions } from "../useDictActions";
import { createApiMock } from "@/test/utils/createApiMock";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <AntdApp>{children}</AntdApp>
    </QueryClientProvider>
  );
}

const baseParams = {
  selectedType: "t1",
  loadDictTypes: vi.fn().mockResolvedValue(undefined),
  loadDictData: vi.fn().mockResolvedValue(undefined),
  loadTypeStatistics: vi.fn().mockResolvedValue(undefined),
  loadDataStatistics: vi.fn().mockResolvedValue(undefined),
};

describe("useDictActions", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    expect(result.current.editingType).toBeNull();
    expect(result.current.editingData).toBeNull();
    expect(result.current.typeModalVisible).toBe(false);
    expect(result.current.dataModalVisible).toBe(false);
  });

  it("handleCreateType 新建模式 → POST /system/dicts/types", async () => {
    const api = createApiMock("/system/dicts/types");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const fakeForm = {
      validateFields: vi.fn().mockResolvedValue({ name: "t" }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleCreateType(fakeForm);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleCreateType 校验失败 → 不调 API", async () => {
    const api = createApiMock("/system/dicts/types");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const fakeForm = {
      validateFields: vi
        .fn()
        .mockRejectedValue({ errorFields: [{ name: "name", errors: ["required"] }] }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleCreateType(fakeForm);
    });
    expect(api.endpoint).not.toHaveBeenCalled();
  });

  it("handleCreateType 编辑模式 → POST update", async () => {
    const api = createApiMock("/system/dicts/types/d1/update");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const fakeForm = {
      validateFields: vi.fn().mockResolvedValue({ name: "t" }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    act(() => result.current.setEditingType({ id: "d1", name: "old", dictType: "x" } as any));
    await act(async () => {
      await result.current.handleCreateType(fakeForm);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleDeleteType → POST delete", async () => {
    const api = createApiMock("/system/dicts/types/d1/delete");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleDeleteType("d1");
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleBatchDeleteType → POST batch-delete", async () => {
    const api = createApiMock("/system/dicts/types/batch-delete");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const setKeys = vi.fn();
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleBatchDeleteType(["d1", "d2"], setKeys);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleCreateData → POST /system/dicts/data", async () => {
    const api = createApiMock("/system/dicts/data");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const fakeForm = {
      validateFields: vi.fn().mockResolvedValue({ label: "x" }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleCreateData(fakeForm);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleDeleteData → POST delete", async () => {
    const api = createApiMock("/system/dicts/data/d1/delete");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleDeleteData("d1");
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleBatchDeleteData → POST batch-delete", async () => {
    const api = createApiMock("/system/dicts/data/batch-delete");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const setKeys = vi.fn();
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleBatchDeleteData(["d1"], setKeys);
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("handleRefreshCache → POST refresh", async () => {
    const api = createApiMock("/system/dicts/refresh-cache");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    await act(async () => {
      await result.current.handleRefreshCache();
    });
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("openTypeModal 无 record → 新建模式", () => {
    const fakeForm = { resetFields: vi.fn(), setFieldsValue: vi.fn() } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    act(() => result.current.openTypeModal(undefined, fakeForm));
    expect(result.current.typeModalVisible).toBe(true);
    expect(result.current.editingType).toBeNull();
  });

  it("openTypeModal 有 record → 编辑模式", () => {
    const fakeForm = { resetFields: vi.fn(), setFieldsValue: vi.fn() } as any;
    const record = { id: "d1", name: "t1", dictType: "x" } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    act(() => result.current.openTypeModal(record, fakeForm));
    expect(result.current.editingType).toEqual(record);
  });

  it("openDataModal 有 record → 编辑模式", () => {
    const fakeForm = { resetFields: vi.fn(), setFieldsValue: vi.fn() } as any;
    const record = { id: "d1", label: "x", value: "1" } as any;
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    act(() => result.current.openDataModal(record, fakeForm));
    expect(result.current.editingData).toEqual(record);
  });

  it("setEditingType/setTypeModalVisible 直接写入", () => {
    const { result } = renderHook(() => useDictActions(baseParams), { wrapper });
    act(() => {
      result.current.setEditingType({ id: "d1" } as any);
      result.current.setTypeModalVisible(true);
      result.current.setDataModalVisible(true);
    });
    expect(result.current.editingType?.id).toBe("d1");
    expect(result.current.typeModalVisible).toBe(true);
  });
});

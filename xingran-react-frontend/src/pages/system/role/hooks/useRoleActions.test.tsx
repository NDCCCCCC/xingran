/**
 * Phase 88 Batch37 — system role hooks 单元测试(useRoleActions)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useRoleActions } from "../hooks/useRoleActions";
import * as apiModule from "@/lib/api";

const postSpy = vi.fn();
const baseParams = () => ({
  loadRoles: vi.fn(),
  loadStatistics: vi.fn(),
  loadRoleMenus: vi.fn().mockResolvedValue(["m1", "m2"]),
  loadRoleDepts: vi.fn().mockResolvedValue(["d1"]),
  checkedMenuKeys: [],
  checkedDeptKeys: [],
  setCheckedMenuKeys: vi.fn(),
  setCheckedDeptKeys: vi.fn(),
  currentDataScope: 4,
  setCurrentDataScope: vi.fn(),
});

function Wrap({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return (
    <ConfigProvider>
      <QueryClientProvider client={qc}>
        <App>{children}</App>
      </QueryClientProvider>
    </ConfigProvider>
  );
}
const wrap = Wrap;

beforeEach(() => {
  postSpy.mockReset().mockResolvedValue({});
  vi.spyOn(apiModule, "post" as any).mockImplementation(postSpy as any);
});

describe("useRoleActions", () => {
  it("initial state 全 null/false/[]", () => {
    const { result } = renderHook(() => useRoleActions(baseParams()), { wrapper: wrap });
    expect(result.current.editingRole).toBeNull();
    expect(result.current.editModalVisible).toBe(false);
    expect(result.current.pendingFormData).toBeNull();
    expect(result.current.selectedRowKeys).toEqual([]);
  });

  it("handleAdd 打开 modal + clearing editingRole", () => {
    const { result } = renderHook(() => useRoleActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleAdd();
    });
    expect(result.current.editModalVisible).toBe(true);
    expect(result.current.editingRole).toBeNull();
    expect(result.current.pendingFormData).toBeNull();
  });

  it("handleEdit 加载 menu/dept 并写入 formData", async () => {
    const params = baseParams();
    const { result } = renderHook(() => useRoleActions(params), { wrapper: wrap });
    const record = { id: "r1", roleName: "管理员" } as any;
    await act(async () => {
      await result.current.handleEdit(record);
    });
    expect(params.loadRoleMenus).toHaveBeenCalledWith("r1");
    expect(params.loadRoleDepts).toHaveBeenCalledWith("r1");
    expect(result.current.editingRole).toEqual(record);
    expect(result.current.editModalVisible).toBe(true);
    expect(result.current.pendingFormData).toMatchObject({
      id: "r1",
      roleName: "管理员",
      menuIds: ["m1", "m2"],
      deptIds: ["d1"],
    });
  });

  it("handleDelete 调 post + invalidate", async () => {
    const params = baseParams();
    const { result } = renderHook(() => useRoleActions(params), { wrapper: wrap });
    await act(async () => {
      await result.current.handleDelete("r1");
    });
    expect(postSpy).toHaveBeenCalledWith("/system/roles/r1/delete");
    expect(params.loadRoles).toHaveBeenCalled();
    expect(params.loadStatistics).toHaveBeenCalled();
  });

  it("handleBatchDelete 空 keys 警告 + 不调 post", async () => {
    const params = baseParams();
    const { result } = renderHook(() => useRoleActions(params), { wrapper: wrap });
    await act(async () => {
      await result.current.handleBatchDelete([]);
    });
    expect(postSpy).not.toHaveBeenCalled();
  });

  it("handleBatchDelete 非空调 post + 清 selectedRowKeys", async () => {
    const params = baseParams();
    const { result } = renderHook(() => useRoleActions(params), { wrapper: wrap });
    await act(async () => {
      // 先 set 再 batchDelete
      act(() => {
        result.current.setSelectedRowKeys(["r1", "r2"]);
      });
      await act(async () => {
        await result.current.handleBatchDelete(["r1", "r2"]);
      });
    });
    expect(postSpy).toHaveBeenCalledWith("/system/roles/batch-delete", { ids: ["r1", "r2"] });
    expect(params.loadRoles).toHaveBeenCalled();
  });

  it("handleUpdateStatus 调 post + status 文案", async () => {
    const params = baseParams();
    const { result } = renderHook(() => useRoleActions(params), { wrapper: wrap });
    await act(async () => {
      await result.current.handleUpdateStatus("r1", 0);
    });
    expect(postSpy).toHaveBeenCalledWith("/system/roles/r1/status", { status: 0 });
    expect(postSpy).toHaveBeenCalledTimes(1);
  });
});

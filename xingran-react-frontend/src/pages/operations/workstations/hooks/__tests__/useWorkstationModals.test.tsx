/**
 * Phase 88 Batch117 — operations/workstations/hooks/useWorkstationModals 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  workstationApi: {
    update: vi.fn(() => Promise.resolve({ code: 0 })),
    create: vi.fn(() => Promise.resolve({ code: 0 })),
    delete: vi.fn(() => Promise.resolve({ code: 0 })),
    batch: vi.fn(() => Promise.resolve({ code: 0 })),
  },
}));

import { useWorkstationModals } from "../useWorkstationModals";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useWorkstationModals", () => {
  it("openModal 无 record → 返回 undefined", async () => {
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    let r: any;
    await act(async () => {
      r = await result.current.openModal();
    });
    expect(r).toBeUndefined();
  });

  it("openModal 有 record + loadUserOptions → 调用 loadUserOptions", async () => {
    const loadUserOptions = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useWorkstationModals(loadUserOptions), { wrapper });
    await act(async () => {
      await result.current.openModal({ id: "w1", deptId: "d1" } as any);
    });
    expect(loadUserOptions).toHaveBeenCalledWith("d1");
  });

  it("closeModal 有 form → resetFields", () => {
    const form = { resetFields: vi.fn() } as any;
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    act(() => result.current.closeModal(form));
    expect(form.resetFields).toHaveBeenCalled();
  });

  it("closeModal 无 form → 不抛错", () => {
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    expect(() => act(() => result.current.closeModal())).not.toThrow();
  });

  it("handleSave 新建模式 → POST create", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.create).mockClear();
    const form = { validateFields: vi.fn().mockResolvedValue({ name: "w1" }) } as any;
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleSave(null, form, vi.fn());
    });
    expect(workstationApi.create).toHaveBeenCalledWith({ name: "w1" });
  });

  it("handleSave 编辑模式 → POST update", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.update).mockClear();
    const form = { validateFields: vi.fn().mockResolvedValue({ name: "w1" }) } as any;
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleSave({ id: "w1" } as any, form, vi.fn());
    });
    expect(workstationApi.update).toHaveBeenCalledWith("w1", { name: "w1" });
  });

  it("handleSave 校验失败 → 不调 API", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.create).mockClear();
    const form = {
      validateFields: vi
        .fn()
        .mockRejectedValue({ errorFields: [{ name: "name", errors: ["required"] }] }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleSave(null, form, vi.fn());
    });
    expect(workstationApi.create).not.toHaveBeenCalled();
  });

  it("handleDelete → POST delete", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.delete).mockClear();
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleDelete("w1", onSuccess);
    });
    expect(workstationApi.delete).toHaveBeenCalledWith("w1");
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleBatchDelete 空数组 → 不调 API", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.batch).mockClear();
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleBatchDelete([], vi.fn(), vi.fn());
    });
    expect(workstationApi.batch).not.toHaveBeenCalled();
  });

  it("handleBatchDelete 有 ids → POST batch delete + resetSelection", async () => {
    const { workstationApi } = await import("@/lib/opsApi");
    vi.mocked(workstationApi.batch).mockClear();
    const onSuccess = vi.fn();
    const resetSelection = vi.fn();
    const { result } = renderHook(() => useWorkstationModals(), { wrapper });
    await act(async () => {
      await result.current.handleBatchDelete(["w1", "w2"], onSuccess, resetSelection);
    });
    expect(workstationApi.batch).toHaveBeenCalledWith("delete", { ids: ["w1", "w2"] });
    expect(resetSelection).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalled();
  });
});

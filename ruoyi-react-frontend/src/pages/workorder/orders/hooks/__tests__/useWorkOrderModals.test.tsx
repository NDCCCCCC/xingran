/**
 * Phase 88 Batch126 — workorder/orders/hooks/useWorkOrderModals 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/workorderApi", () => ({
  getWorkOrderComments: vi.fn(() =>
    Promise.resolve({ data: [{ id: "c1", content: "C", isInternal: false }] })
  ),
  getWorkOrderHistory: vi.fn(() => Promise.resolve({ data: [{ id: "h1", action: "create" }] })),
}));

import { useWorkOrderModals } from "../useWorkOrderModals";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useWorkOrderModals", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    expect(result.current.modalVisible).toBe(false);
    expect(result.current.detailDrawerVisible).toBe(false);
    expect(result.current.editingRecord).toBeNull();
    expect(result.current.selectedRecord).toBeNull();
    expect(result.current.comments).toEqual([]);
    expect(result.current.history).toEqual([]);
    expect(result.current.commentInternal).toBe(false);
  });

  it("openAddModal → modalVisible=true + editingRecord=null", () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    act(() => result.current.openAddModal());
    expect(result.current.modalVisible).toBe(true);
    expect(result.current.editingRecord).toBeNull();
  });

  it("openEditModal → modalVisible=true + editingRecord=record", async () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    const editForm = { setFieldsValue: vi.fn() } as any;
    const record = { id: "w1", title: "T" } as any;
    act(() => result.current.openEditModal(record, editForm));
    expect(result.current.modalVisible).toBe(true);
    expect(result.current.editingRecord).toEqual(record);
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
    expect(editForm.setFieldsValue).toHaveBeenCalled();
  });

  it("openDetailDrawer → 加载 comments + history", async () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    const record = { id: "w1" } as any;
    await act(async () => {
      await result.current.openDetailDrawer(record);
    });
    expect(result.current.detailDrawerVisible).toBe(true);
    expect(result.current.selectedRecord).toEqual(record);
    expect(result.current.comments.length).toBe(1);
    expect(result.current.history.length).toBe(1);
  });

  it("openDetailDrawer 加载失败 → console.error + visible 仍 true", async () => {
    const { getWorkOrderComments } = await import("@/lib/workorderApi");
    vi.mocked(getWorkOrderComments).mockRejectedValueOnce(new Error("net"));
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    await act(async () => {
      await result.current.openDetailDrawer({ id: "w1" } as any);
    });
    expect(errSpy).toHaveBeenCalled();
    expect(result.current.detailDrawerVisible).toBe(true);
    errSpy.mockRestore();
  });

  it("closeModals → 重置所有状态", () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    act(() => result.current.openAddModal());
    act(() => result.current.closeModals());
    expect(result.current.modalVisible).toBe(false);
    expect(result.current.detailDrawerVisible).toBe(false);
    expect(result.current.editingRecord).toBeNull();
    expect(result.current.selectedRecord).toBeNull();
  });

  it("setCommentInternal / setComments / setHistory 直接写入", () => {
    const { result } = renderHook(() => useWorkOrderModals(), { wrapper });
    act(() => {
      result.current.setCommentInternal(true);
      result.current.setComments([{ id: "x", content: "y", isInternal: true, createdAt: "" }]);
      result.current.setHistory([{ id: "z", action: "update", createdAt: "" }]);
    });
    expect(result.current.commentInternal).toBe(true);
    expect(result.current.comments.length).toBe(1);
    expect(result.current.history.length).toBe(1);
  });
});

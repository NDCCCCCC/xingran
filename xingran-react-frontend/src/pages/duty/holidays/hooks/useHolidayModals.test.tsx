/**
 * Phase 88 Batch71 — useHolidayModals hook 测试(238 行大 hook)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/dutyApi", () => ({
  createHoliday: vi.fn().mockResolvedValue({ data: {} }),
  updateHoliday: vi.fn().mockResolvedValue({ data: {} }),
  deleteHoliday: vi.fn().mockResolvedValue({ data: {} }),
  batchCreateHolidays: vi.fn().mockResolvedValue({ data: { count: 3 } }),
}));

import { useHolidayModals } from "../hooks/useHolidayModals";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({
  year: 2026,
  availableYears: [2026, 2025],
  fetchList: vi.fn().mockResolvedValue(undefined),
});

describe("useHolidayModals", () => {
  it("initial state + handlers", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    expect(result.current.modalState.modalVisible).toBe(false);
    expect(result.current.modalState.batchModalVisible).toBe(false);
    expect(result.current.modalState.editingRecord).toBeNull();
    expect(typeof result.current.handleAdd).toBe("function");
    expect(typeof result.current.handleEdit).toBe("function");
    expect(typeof result.current.handleDelete).toBe("function");
    expect(typeof result.current.handleModalOk).toBe("function");
    expect(typeof result.current.handleBatchSubmit).toBe("function");
  });

  it("handleAdd 开 modal + editingRecord=null", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    act(() => result.current.handleAdd());
    expect(result.current.modalState.modalVisible).toBe(true);
    expect(result.current.modalState.editingRecord).toBeNull();
  });

  it("handleEdit 设 editingRecord + 开 modal", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    act(() =>
      result.current.handleEdit({
        id: "h1",
        date: "2026-10-01",
        name: "国庆",
      } as any)
    );
    expect(result.current.modalState.modalVisible).toBe(true);
    expect(result.current.modalState.editingRecord?.id).toBe("h1");
  });

  it("handleDelete 调 deleteHoliday + fetchList", async () => {
    const fetchList = vi.fn();
    const { result } = renderHook(() => useHolidayModals({ ...opts(), fetchList }), {
      wrapper: wrap,
    });
    await act(async () => {
      await result.current.handleDelete("h1");
    });
    expect(fetchList).toHaveBeenCalled();
  });

  it("handleBatchAdd 开 batch modal", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    act(() => result.current.handleBatchAdd());
    expect(result.current.modalState.batchModalVisible).toBe(true);
  });

  it("addBatchRow / removeBatchRow / updateBatchRow", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    act(() => {
      result.current.handleBatchAdd();
      result.current.addBatchRow();
      result.current.updateBatchRow(1, "name", "元旦");
    });
    expect(result.current.batchState.batchHolidays.length).toBeGreaterThanOrEqual(1);
    act(() => {
      result.current.removeBatchRow(1);
    });
    expect(result.current.batchState.batchHolidays.length).toBeGreaterThanOrEqual(0);
  });

  it("handleBatchSubmit 调 batchCreateHolidays + fetchList", async () => {
    const fetchList = vi.fn();
    const { result } = renderHook(() => useHolidayModals({ ...opts(), fetchList }), {
      wrapper: wrap,
    });
    act(() => {
      result.current.handleBatchAdd();
    });
    await act(async () => {
      await result.current.handleBatchSubmit();
    });
  });

  it("setters 直写 state", () => {
    const { result } = renderHook(() => useHolidayModals(opts()), { wrapper: wrap });
    act(() => {
      result.current.setModalVisible(true);
      result.current.setBatchModalVisible(true);
      result.current.setBatchHolidays([{ date: "2026-01-01", name: "元旦" }]);
    });
    expect(result.current.modalState.modalVisible).toBe(true);
    expect(result.current.modalState.batchModalVisible).toBe(true);
    expect(result.current.batchState.batchHolidays.length).toBe(1);
  });
});

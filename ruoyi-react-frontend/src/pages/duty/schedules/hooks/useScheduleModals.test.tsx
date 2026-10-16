/**
 * Phase 88 Batch69 — useScheduleModals hook 测试(简化 open/close + delete)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/dutyApi", () => ({
  generateSchedule: vi.fn().mockResolvedValue({ data: { count: 5 } }),
  swapDuty: vi.fn().mockResolvedValue({ data: {} }),
  manualDuty: vi.fn().mockResolvedValue({ data: {} }),
  deleteDutySchedule: vi.fn().mockResolvedValue({ data: {} }),
  batchDeleteDutySchedules: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useScheduleModals } from "../hooks/useScheduleModals";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({
  onLoad: vi.fn(),
  allSchedules: [],
  dataSource: [],
  current: 1,
});

describe("useScheduleModals", () => {
  it("initial state + handlers", () => {
    const { result } = renderHook(() => useScheduleModals(opts()), { wrapper: wrap });
    expect(result.current.generateModalVisible).toBe(false);
    expect(result.current.swapModalVisible).toBe(false);
    expect(result.current.manualModalVisible).toBe(false);
    expect(result.current.generateForm).toBeDefined();
    expect(result.current.swapForm).toBeDefined();
    expect(result.current.manualForm).toBeDefined();
    expect(typeof result.current.openGenerateModal).toBe("function");
    expect(typeof result.current.handleDelete).toBe("function");
  });

  it("openGenerateModal + closeGenerateModal", () => {
    const { result } = renderHook(() => useScheduleModals(opts()), { wrapper: wrap });
    act(() => result.current.openGenerateModal());
    expect(result.current.generateModalVisible).toBe(true);
    act(() => result.current.closeGenerateModal());
    expect(result.current.generateModalVisible).toBe(false);
  });

  it("openSwapModal + closeSwapModal", () => {
    const { result } = renderHook(() => useScheduleModals(opts()), { wrapper: wrap });
    act(() => result.current.openSwapModal());
    expect(result.current.swapModalVisible).toBe(true);
    act(() => result.current.closeSwapModal());
    expect(result.current.swapModalVisible).toBe(false);
  });

  it("openManualModal + closeManualModal", () => {
    const { result } = renderHook(() => useScheduleModals(opts()), { wrapper: wrap });
    act(() => result.current.openManualModal());
    expect(result.current.manualModalVisible).toBe(true);
    act(() => result.current.closeManualModal());
    expect(result.current.manualModalVisible).toBe(false);
  });

  it("handleDelete 调 deleteDutySchedule + onLoad", async () => {
    const onLoad = vi.fn();
    const { deleteDutySchedule } = await import("@/lib/dutyApi");
    const { result } = renderHook(
      () => useScheduleModals({ onLoad, allSchedules: [], dataSource: [], current: 1 }),
      { wrapper: wrap }
    );
    await act(async () => {
      await result.current.handleDelete("sch1");
    });
    expect(deleteDutySchedule).toHaveBeenCalled();
    expect(onLoad).toHaveBeenCalled();
  });

  it("handleBatchDelete 调 batch endpoint", async () => {
    const onLoad = vi.fn();
    const setKeys = vi.fn();
    const { batchDeleteDutySchedules } = await import("@/lib/dutyApi");
    const { result } = renderHook(
      () => useScheduleModals({ onLoad, allSchedules: [], dataSource: [], current: 1 }),
      { wrapper: wrap }
    );
    await act(async () => {
      await result.current.handleBatchDelete(["k1", "k2"], setKeys);
    });
    expect(batchDeleteDutySchedules).toHaveBeenCalled();
    expect(onLoad).toHaveBeenCalled();
    expect(setKeys).toHaveBeenCalledWith([]);
  });
});

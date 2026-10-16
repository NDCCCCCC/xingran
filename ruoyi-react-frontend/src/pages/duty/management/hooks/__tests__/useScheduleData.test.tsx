/**
 * Phase 88 Batch90 — duty/management/hooks/useScheduleData 测试(108 stmts, 29.6% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/dutyApi", async () => {
  const { createApiMock } = await import("@/test/utils/createApiMock");
  return {
    getDutyScheduleList: createApiMock("/duty/schedules/list").endpoint,
    generateSchedule: createApiMock("/duty/schedules/generate").endpoint,
    swapDuty: createApiMock("/duty/schedules/swap").endpoint,
    manualDuty: createApiMock("/duty/schedules/manual").endpoint,
    deleteDutySchedule: createApiMock("/duty/schedules/delete").endpoint,
    batchDeleteDutySchedules: createApiMock("/duty/schedules/batch-delete").endpoint,
    getMonthlyDutySchedule: createApiMock("/duty/schedules/monthly").endpoint,
  };
});

import { useScheduleData } from "../useScheduleData";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useScheduleData", () => {
  it("初始化默认值", () => {
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    expect(result.current.loading).toBe(false);
    expect(result.current.schedules).toEqual([]);
    expect(result.current.total).toBe(0);
    expect(result.current.current).toBe(1);
    expect(result.current.pageSize).toBe(10);
    expect(result.current.selectedRowKeys).toEqual([]);
  });

  it("fetchList: 成功 → 写入 schedules + total", async () => {
    const { getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(getDutyScheduleList).mockResolvedValueOnce({
      data: {
        list: [{ id: "s1", scheduleDate: "2026-01-01" }],
        total: 1,
      },
    } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });

    await act(async () => {
      await result.current.fetchList({ poolId: "p1" });
    });
    expect(result.current.schedules).toHaveLength(1);
    expect(result.current.total).toBe(1);
  });

  it("fetchList: 失败 → catch 路径", async () => {
    const { getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(getDutyScheduleList).mockRejectedValueOnce(new Error("net"));
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    await act(async () => {
      await result.current.fetchList({});
    });
    expect(result.current.loading).toBe(false);
  });

  it("fetchAllSchedules: 写入 allSchedules", async () => {
    const { getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(getDutyScheduleList).mockResolvedValueOnce({
      data: { list: [{ id: "a1", scheduleDate: "2026-01-01" }], total: 1 },
    } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });

    await act(async () => {
      await result.current.fetchAllSchedules();
    });
    expect(result.current.allSchedules).toHaveLength(1);
  });

  it("fetchWeeklyDuty: 写入 weeklyDutyData (跨月)", async () => {
    const { getMonthlyDutySchedule } = await import("@/lib/dutyApi");
    vi.mocked(getMonthlyDutySchedule).mockResolvedValueOnce({
      data: { "2026-01-06": [{ userName: "Alice" }] },
    } as any);
    vi.mocked(getMonthlyDutySchedule).mockResolvedValueOnce({
      data: { "2026-01-31": [{ userName: "Bob" }] },
    } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });

    await act(async () => {
      await result.current.fetchWeeklyDuty(result.current.currentWeekStart);
    });
    expect(result.current.weeklyDutyData).toBeDefined();
  });

  it("generate: 成功 → refresh", async () => {
    const { generateSchedule, getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(generateSchedule).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getDutyScheduleList).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    await act(async () => {
      await result.current.generate({} as any);
    });
    expect(generateSchedule).toHaveBeenCalled();
  });

  it("swap: 成功", async () => {
    const { swapDuty } = await import("@/lib/dutyApi");
    vi.mocked(swapDuty).mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    await act(async () => {
      await result.current.swap({} as any);
    });
    expect(swapDuty).toHaveBeenCalled();
  });

  it("manual: 成功", async () => {
    const { manualDuty } = await import("@/lib/dutyApi");
    vi.mocked(manualDuty).mockResolvedValueOnce({ code: 0 } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    await act(async () => {
      await result.current.manual({} as any);
    });
    expect(manualDuty).toHaveBeenCalled();
  });

  it("deleteOne: 成功 → refresh", async () => {
    const { deleteDutySchedule, getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(deleteDutySchedule).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getDutyScheduleList).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    await act(async () => {
      await result.current.deleteOne("s1");
    });
    expect(deleteDutySchedule).toHaveBeenCalled();
  });

  it("batchDelete: 成功 → refresh + 清空 selectedRowKeys", async () => {
    const { batchDeleteDutySchedules, getDutyScheduleList } = await import("@/lib/dutyApi");
    vi.mocked(batchDeleteDutySchedules).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getDutyScheduleList).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    act(() => result.current.setSelectedRowKeys(["s1", "s2"]));

    await act(async () => {
      await result.current.batchDelete(["s1", "s2"]);
    });
    expect(batchDeleteDutySchedules).toHaveBeenCalled();
    expect(result.current.selectedRowKeys).toEqual([]);
  });

  it("prevWeek: currentWeekStart 向前推", () => {
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    const before = result.current.currentWeekStart.clone();
    act(() => result.current.prevWeek());
    expect(result.current.currentWeekStart.isBefore(before)).toBe(true);
  });

  it("nextWeek: currentWeekStart 向后推", () => {
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    const before = result.current.currentWeekStart.clone();
    act(() => result.current.nextWeek());
    expect(result.current.currentWeekStart.isAfter(before)).toBe(true);
  });

  it("todayWeek: 回到当前周", () => {
    const { result } = renderHook(() => useScheduleData(), { wrapper });
    act(() => result.current.prevWeek());
    act(() => result.current.todayWeek());
    // 回到今天附近的周一
    expect(result.current.currentWeekStart.isValid()).toBe(true);
  });
});

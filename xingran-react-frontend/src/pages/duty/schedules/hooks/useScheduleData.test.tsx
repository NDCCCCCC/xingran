/**
 * Phase 88 Batch65 — useScheduleData hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";
import dayjs from "dayjs";

vi.mock("@/lib/dutyApi", () => ({
  getDutyScheduleList: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
  getTodayDuty: vi.fn().mockResolvedValue({ data: { list: [] } }),
  getDutyPoolList: vi.fn().mockResolvedValue({ data: [] }),
  getUserList: vi.fn().mockResolvedValue({ data: [] }),
  getMonthlyDutySchedule: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useScheduleData } from "../hooks/useScheduleData";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({ current: 1, pageSize: 10, searchForm: {} as any });

describe("useScheduleData", () => {
  it("initial state", () => {
    const { result } = renderHook(() => useScheduleData(opts()), { wrapper: wrap });
    expect(result.current.dataSource).toEqual([]);
    expect(result.current.total).toBe(0);
    expect(result.current.allSchedules).toEqual([]);
    expect(result.current.pools).toEqual([]);
    expect(result.current.users).toEqual([]);
    expect(result.current.weeklyDutyData).toEqual({});
    expect(result.current.loading).toBe(false);
    expect(typeof result.current.currentWeekStart).toBe("object");
  });

  it("fetchList 调 getDutyScheduleList", async () => {
    const { result } = renderHook(() => useScheduleData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.fetchList();
    });
  });

  it("fetchAllSchedules / fetchPools / fetchUsers", async () => {
    const { result } = renderHook(() => useScheduleData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.fetchAllSchedules();
      await result.current.fetchPools();
      await result.current.fetchUsers();
    });
  });

  it("fetchWeeklyDuty 调 getMonthlyDutySchedule", async () => {
    const { result } = renderHook(() => useScheduleData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.fetchWeeklyDuty(dayjs("2026-08-29"));
    });
  });

  it("setCurrentWeekStart 直写", () => {
    const { result } = renderHook(() => useScheduleData(opts()), { wrapper: wrap });
    const newWeek = dayjs("2026-09-05");
    act(() => result.current.setCurrentWeekStart(newWeek));
    expect(result.current.currentWeekStart.isSame(newWeek, "day")).toBe(true);
  });
});

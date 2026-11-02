/**
 * Phase 88 Batch62 — useHolidayData hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/dutyApi", () => ({
  getHolidayList: vi.fn().mockResolvedValue({ data: [] }),
  getHolidayYears: vi.fn().mockResolvedValue({ data: [2026, 2025] }),
}));

import { useHolidayData } from "../hooks/useHolidayData";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

describe("useHolidayData", () => {
  it("initial state + fetchAvailableYears 自动调", async () => {
    const { result } = renderHook(() => useHolidayData(), { wrapper: wrap });
    await waitFor(() => {
      expect(result.current.availableYears.length).toBeGreaterThan(0);
    });
    expect(result.current.year).toBe(2026); // 最新年份
  });

  it("fetchList 调 getHolidayList + setDataSource", async () => {
    const { result } = renderHook(() => useHolidayData(), { wrapper: wrap });
    await waitFor(() => {
      expect(result.current.availableYears.length).toBeGreaterThan(0);
    });
    await act(async () => {
      await result.current.fetchList(2025);
    });
    expect(result.current.year).toBe(2025);
  });

  it("setters 直写 state", async () => {
    const { result } = renderHook(() => useHolidayData(), { wrapper: wrap });
    await waitFor(() => {
      expect(result.current.availableYears.length).toBeGreaterThan(0);
    });
    act(() => {
      result.current.setYear(2024);
      result.current.setDataSource([{ id: "h1", year: 2024, date: "2024-01-01", name: "元旦" }]);
      result.current.setAvailableYears([2024]);
    });
    expect(result.current.year).toBe(2024);
    expect(result.current.dataSource.length).toBe(1);
    expect(result.current.availableYears).toEqual([2024]);
  });
});

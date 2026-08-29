/**
 * Phase 88 Batch89 — duty/management/hooks/useHolidayData 测试(62 stmts, 27.4% → 高)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/dutyApi", async () => {
  const { createApiMock } = await import("@/test/utils/createApiMock");
  const getHolidayList = createApiMock("/duty/holidays/list");
  const getHolidayYears = createApiMock("/duty/holidays/years");
  const createHoliday = createApiMock("/duty/holidays");
  const updateHoliday = createApiMock("/duty/holidays/update");
  const deleteHoliday = createApiMock("/duty/holidays/delete");
  const batchCreateHolidays = createApiMock("/duty/holidays/batch");
  return {
    getHolidayList: getHolidayList.endpoint,
    getHolidayYears: getHolidayYears.endpoint,
    createHoliday: createHoliday.endpoint,
    updateHoliday: updateHoliday.endpoint,
    deleteHoliday: deleteHoliday.endpoint,
    batchCreateHolidays: batchCreateHolidays.endpoint,
  };
});

import { useHolidayData } from "../useHolidayData";

function wrapper({ children }: { children: React.ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("useHolidayData", () => {
  beforeEach(async () => {
    const { resetApiMocks } = await import("@/test/utils/createApiMock");
    resetApiMocks();
  });

  it("初始化默认值", () => {
    const { result } = renderHook(() => useHolidayData(), { wrapper });
    expect(result.current.holidays).toEqual([]);
    expect(result.current.loading).toBe(false);
    expect(result.current.availableYears).toEqual([]);
    expect(result.current.holidayYear).toBeUndefined();
  });

  it("fetchList: year 未定义 → 不发请求", async () => {
    const { result } = renderHook(() => useHolidayData(), { wrapper });
    await act(async () => {
      await result.current.fetchList();
    });
    expect(result.current.holidays).toEqual([]);
  });

  it("fetchList: getHolidayList 成功 → 写入 holidays", async () => {
    const { getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(getHolidayList).mockResolvedValueOnce({
      data: [
        {
          id: "h1",
          holidayDate: "2026-01-01",
          holidayName: "元旦",
          holidayType: "legal",
          year: 2026,
          isOffday: true,
        },
      ],
    } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    act(() => result.current.setHolidayYear(2026));

    await act(async () => {
      await result.current.fetchList();
    });
    await waitFor(() => {
      expect(result.current.holidays).toHaveLength(1);
    });
  });

  it("fetchList: getHolidayList 失败 → catch 路径", async () => {
    const { getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(getHolidayList).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    act(() => result.current.setHolidayYear(2026));

    await act(async () => {
      await result.current.fetchList();
    });
    expect(result.current.loading).toBe(false);
  });

  it("fetchYears: 成功 → 写入 availableYears + 自动选 firstYear", async () => {
    const { getHolidayYears, getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(getHolidayYears).mockResolvedValueOnce({ data: [2024, 2025, 2026] } as any);
    vi.mocked(getHolidayList).mockResolvedValueOnce({ data: [] } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });

    await act(async () => {
      await result.current.fetchYears();
    });
    expect(result.current.availableYears).toEqual([2024, 2025, 2026]);
    expect(result.current.holidayYear).toBe(2024);
  });

  it("fetchYears: 失败 → catch 路径", async () => {
    const { getHolidayYears } = await import("@/lib/dutyApi");
    vi.mocked(getHolidayYears).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    await act(async () => {
      await result.current.fetchYears();
    });
    expect(result.current.loading).toBe(false);
  });

  it("create: 成功 → refresh", async () => {
    const { createHoliday, getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(createHoliday).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getHolidayList).mockResolvedValueOnce({ data: [] } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    act(() => result.current.setHolidayYear(2026));

    await act(async () => {
      await result.current.create({
        holidayDate: "2026-05-01",
        holidayName: "劳动节",
        holidayType: "legal",
        year: 2026,
        isOffday: true,
      } as any);
    });
    expect(createHoliday).toHaveBeenCalled();
  });

  it("update: 成功 → refresh", async () => {
    const { updateHoliday, getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(updateHoliday).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getHolidayList).mockResolvedValueOnce({ data: [] } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    act(() => result.current.setHolidayYear(2026));

    await act(async () => {
      await result.current.update("h1", { holidayName: "测试" } as any);
    });
    expect(updateHoliday).toHaveBeenCalled();
  });

  it("deleteOne: 成功 → refresh", async () => {
    const { deleteHoliday, getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(deleteHoliday).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getHolidayList).mockResolvedValueOnce({ data: [] } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });
    act(() => result.current.setHolidayYear(2026));

    await act(async () => {
      await result.current.deleteOne("h1");
    });
    expect(deleteHoliday).toHaveBeenCalled();
  });

  it("batchCreate: 成功 → fetchYears + fetchList", async () => {
    const { batchCreateHolidays, getHolidayYears, getHolidayList } = await import("@/lib/dutyApi");
    vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);
    vi.mocked(getHolidayYears).mockResolvedValueOnce({ data: [2026] } as any);
    vi.mocked(getHolidayList).mockResolvedValueOnce({ data: [] } as any);

    const { result } = renderHook(() => useHolidayData(), { wrapper });

    await act(async () => {
      await result.current.batchCreate([
        {
          holidayDate: "2026-01-01",
          holidayName: "元旦",
          holidayType: "legal",
          year: 2026,
          isOffday: true,
        },
      ] as any);
    });
    expect(batchCreateHolidays).toHaveBeenCalled();
    expect(getHolidayYears).toHaveBeenCalled();
  });
});

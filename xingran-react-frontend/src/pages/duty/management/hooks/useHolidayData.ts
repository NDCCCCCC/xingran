import { useState, useCallback } from "react";
import { App } from "antd";
import {
  getHolidayList,
  createHoliday,
  updateHoliday,
  deleteHoliday,
  batchCreateHolidays,
  getHolidayYears,
  type Holiday,
} from "@/lib/dutyApi";

export function useHolidayData() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [holidays, setHolidays] = useState<Holiday[]>([]);
  const [holidayYear, setHolidayYear] = useState<number | undefined>(undefined);
  const [availableYears, setAvailableYears] = useState<number[]>([]);

  // 获取节假日列表
  const fetchList = useCallback(
    async (year?: number) => {
      setLoading(true);
      try {
        const targetYear = year ?? holidayYear;
        if (targetYear === undefined) {
          setLoading(false);
          return;
        }
        const result = await getHolidayList(targetYear);
        setHolidays(result.data || []);
        if (year !== undefined) setHolidayYear(year);
      } catch (_error) {
        message.error("获取节假日列表失败");
      } finally {
        setLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [holidayYear]
  );

  // 获取可用年份
  // DATA-04: 首次加载时年份 + 节假日列表并行请求（原实现年份→列表串行,2 次 RTT → 1 次）
  const fetchYears = useCallback(async () => {
    try {
      // 已有选中年份（年份切换后的重入）：仅刷新年份列表，不动列表数据（与原实现一致）
      if (holidayYear !== undefined) {
        const result = await getHolidayYears();
        setAvailableYears(result.data || []);
        return;
      }

      setLoading(true);
      const currentYear = new Date().getFullYear();
      const [yearsResult, listResult] = await Promise.all([
        getHolidayYears(),
        getHolidayList(currentYear),
      ]);
      const years = yearsResult.data || [];
      setAvailableYears(years);

      if (years.length === 0) {
        setHolidays([]);
        return;
      }

      // 默认选择最新的年份（第一个，因为后端已按降序返回）
      const latestYear = years[0];
      setHolidayYear(latestYear);
      // 展示最新年份数据；并行预取的是当前年份，两者不一致时补拉一次纠正
      if (latestYear === currentYear) {
        setHolidays(listResult.data || []);
      } else {
        const corrected = await getHolidayList(latestYear);
        setHolidays(corrected.data || []);
      }
    } catch (error) {
      console.error("获取年份列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, [holidayYear]);

  // 创建节假日
  const create = useCallback(
    async (data: Omit<Holiday, "id" | "createdAt" | "updatedAt" | "createdBy">) => {
      try {
        await createHoliday(data);
        message.success("创建成功");
        await fetchList();
        return true;
      } catch (_error) {
        message.error("创建失败");
        return false;
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchList]
  );

  // 更新节假日
  const update = useCallback(
    async (id: string, data: Partial<Holiday>) => {
      try {
        await updateHoliday(id, data);
        message.success("更新成功");
        await fetchList();
        return true;
      } catch (_error) {
        message.error("更新失败");
        return false;
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchList]
  );

  // 删除节假日
  const deleteOne = useCallback(
    async (id: string) => {
      try {
        await deleteHoliday(id);
        message.success("删除成功");
        await fetchList();
        return true;
      } catch (_error) {
        message.error("删除失败");
        return false;
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchList]
  );

  // 批量创建节假日
  const batchCreate = useCallback(
    async (
      dataList: Array<{
        holidayDate: string;
        holidayName: string;
        isOffday: boolean;
        holidayType: "legal" | "workday" | "custom";
        year: number;
        remark?: string;
      }>
    ) => {
      try {
        await batchCreateHolidays(dataList);
        message.success(`成功创建 ${dataList.length} 条节假日记录`);
        await fetchList();
        await fetchYears();
        return true;
      } catch (_error) {
        message.error("批量创建失败");
        return false;
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchList, fetchYears]
  );

  return {
    loading,
    holidays,
    holidayYear,
    setHolidayYear,
    availableYears,
    fetchList,
    fetchYears,
    create,
    update,
    deleteOne,
    batchCreate,
  };
}

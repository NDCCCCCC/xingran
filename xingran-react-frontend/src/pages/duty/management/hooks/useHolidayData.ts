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
      } catch (error) {
        message.error("获取节假日列表失败");
      } finally {
        setLoading(false);
      }
    },
    [holidayYear]
  );

  // 获取可用年份
  const fetchYears = useCallback(async () => {
    try {
      const result = await getHolidayYears();
      const years = result.data || [];
      setAvailableYears(years);

      if (years.length > 0 && holidayYear === undefined) {
        const latestYear = years[0];
        setHolidayYear(latestYear);
        fetchList(latestYear);
      }
    } catch (error) {
      console.error("获取年份列表失败:", error);
    }
  }, [holidayYear, fetchList]);

  // 创建节假日
  const create = useCallback(
    async (data: Omit<Holiday, "id" | "createdAt" | "updatedAt" | "createdBy">) => {
      try {
        await createHoliday(data);
        message.success("创建成功");
        await fetchList();
        return true;
      } catch (error) {
        message.error("创建失败");
        return false;
      }
    },
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
      } catch (error) {
        message.error("更新失败");
        return false;
      }
    },
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
      } catch (error) {
        message.error("删除失败");
        return false;
      }
    },
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
      } catch (error) {
        message.error("批量创建失败");
        return false;
      }
    },
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

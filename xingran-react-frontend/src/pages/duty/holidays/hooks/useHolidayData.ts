/**
 * Holiday Data Hook
 * 节假日数据管理 Hook
 */

import { useState, useCallback, useEffect } from "react";
import { App } from "antd";
import type { Holiday } from "@/lib/dutyApi";
import { getHolidayList, getHolidayYears } from "@/lib/dutyApi";

export interface UseHolidayDataReturn {
  loading: boolean;
  dataSource: Holiday[];
  year: number | undefined;
  availableYears: number[];

  setDataSource: React.Dispatch<React.SetStateAction<Holiday[]>>;
  setYear: React.Dispatch<React.SetStateAction<number | undefined>>;
  setAvailableYears: React.Dispatch<React.SetStateAction<number[]>>;

  fetchList: (y?: number) => Promise<void>;
  fetchAvailableYears: () => Promise<void>;
}

export function useHolidayData(): UseHolidayDataReturn {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<Holiday[]>([]);
  const [year, setYear] = useState<number | undefined>(undefined);
  const [availableYears, setAvailableYears] = useState<number[]>([]);

  // 加载节假日列表
  const fetchList = useCallback(
    async (y?: number) => {
      setLoading(true);
      try {
        const targetYear = y ?? year;
        if (targetYear === undefined) {
          setLoading(false);
          return;
        }
        const result = await getHolidayList(targetYear);
        const data = result as { code: number; data: Holiday[] };
        setDataSource(data.data);
        if (y !== undefined) setYear(y);
      } catch (error) {
        message.error("获取节假日列表失败");
      } finally {
        setLoading(false);
      }
    },
    [year]
  );

  // 加载可用年份列表
  const fetchAvailableYears = useCallback(async () => {
    try {
      const result = await getHolidayYears();
      const years = result.data || [];
      setAvailableYears(years);

      // 默认选择最新的年份（第一个，因为后端已按降序返回）
      if (years.length > 0 && year === undefined) {
        const latestYear = years[0];
        setYear(latestYear);
        fetchList(latestYear);
      }
    } catch (error) {
      console.error("获取年份列表失败:", error);
    }
  }, [year, fetchList]);

  // 初始化时加载年份列表
  useEffect(() => {
    fetchAvailableYears();
  }, [fetchAvailableYears]);

  return {
    loading,
    dataSource,
    year,
    availableYears,
    setDataSource,
    setYear,
    setAvailableYears,
    fetchList,
    fetchAvailableYears,
  };
}

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
      } catch (_error) {
        message.error("获取节假日列表失败");
      } finally {
        setLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [year]
  );

  // 加载可用年份列表
  // DATA-04: 首次加载时年份 + 节假日列表并行请求（原实现年份→列表串行,2 次 RTT → 1 次）
  const fetchAvailableYears = useCallback(async () => {
    try {
      // 已有选中年份（年份切换后的重入）：仅刷新年份列表，不动列表数据（与原实现一致）
      if (year !== undefined) {
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
        setDataSource([]);
        return;
      }

      // 默认选择最新的年份（第一个，因为后端已按降序返回）
      const latestYear = years[0];
      setYear(latestYear);
      // 展示最新年份数据；并行预取的是当前年份，两者不一致时补拉一次纠正
      if (latestYear === currentYear) {
        setDataSource(listResult.data || []);
      } else {
        const corrected = await getHolidayList(latestYear);
        setDataSource(corrected.data || []);
      }
    } catch (error) {
      console.error("获取年份列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, [year]);

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

/**
 * Duty Schedule Data Hook
 * 值班排班数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import type { FormInstance } from "antd/es/form";
import {
  getDutyScheduleList,
  getTodayDuty,
  getDutyPoolList,
  getUserList,
  getMonthlyDutySchedule,
  type DutySchedule,
  type DutyPool,
  type SimpleUser,
  type MonthlyDutyMember,
} from "@/lib/dutyApi";

export interface UseScheduleDataParams {
  current: number;
  pageSize: number;
  searchForm: FormInstance<unknown>;
}

export interface UseScheduleDataReturn {
  // 数据状态
  dataSource: DutySchedule[];
  total: number;
  allSchedules: DutySchedule[];
  pools: DutyPool[];
  users: SimpleUser[];
  weeklyDutyData: Record<string, MonthlyDutyMember[]>;
  loading: boolean;
  currentWeekStart: Dayjs;

  // 数据加载方法
  fetchList: (
    page?: number,
    pageSize?: number,
    sortCol?: string,
    sortAsc?: boolean
  ) => Promise<void>;
  fetchAllSchedules: () => Promise<void>;
  fetchPools: () => Promise<void>;
  fetchUsers: () => Promise<void>;
  fetchWeeklyDuty: (weekStart: Dayjs) => Promise<void>;
  setCurrentWeekStart: (weekStart: Dayjs) => void;
}

export function useScheduleData(params: UseScheduleDataParams): UseScheduleDataReturn {
  const { current, pageSize, searchForm } = params;
  const { message } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<DutySchedule[]>([]);
  const [total, setTotal] = useState(0);
  const [allSchedules, setAllSchedules] = useState<DutySchedule[]>([]);
  const [pools, setPools] = useState<DutyPool[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);
  const [weeklyDutyData, setWeeklyDutyData] = useState<Record<string, MonthlyDutyMember[]>>({});
  const [currentWeekStart, setCurrentWeekStart] = useState<Dayjs>(dayjs().startOf("week"));

  // 获取排班列表
  const fetchList = useCallback(
    async (page?: number, size?: number, sortCol?: string, sortAsc?: boolean) => {
      setLoading(true);
      try {
        const values = searchForm.getFieldsValue() as {
          poolId?: string;
          userId?: string;
          dateRange?: [Dayjs, Dayjs];
          dutyType?: string;
          expired?: number;
        };
        const result = await getDutyScheduleList({
          current: page ?? current,
          pageSize: size ?? pageSize,
          poolId: values.poolId,
          userId: values.userId,
          startDate: values.dateRange?.[0]?.format("YYYY-MM-DD"),
          endDate: values.dateRange?.[1]?.format("YYYY-MM-DD"),
          dutyType: values.dutyType,
          expired: values.expired,
          // 服务端排序透传（避坑：详见 memory server-sort-loadfunc-param-drop）
          ...(sortCol ? { orderByColumn: sortCol, isAsc: sortAsc } : {}),
        });
        setDataSource(result.data?.list ?? []);
        setTotal(result.data?.total ?? 0);
      } catch (error) {
        message.error("获取排班列表失败");
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize, searchForm]
  );

  // 获取今日值班
  const fetchTodayDuty = useCallback(async () => {
    try {
      await getTodayDuty();
    } catch (error) {
      console.error("获取今日值班失败", error);
    }
  }, []);

  // 获取所有排班（用于调班）
  const fetchAllSchedules = useCallback(async () => {
    try {
      const result = await getDutyScheduleList({ current: 1, pageSize: 50 });
      setAllSchedules(result.data?.list ?? []);
    } catch (error) {
      console.error("获取所有排班失败", error);
    }
  }, []);

  // 获取值班池列表（只获取启用的）
  const fetchPools = useCallback(async () => {
    try {
      const result = await getDutyPoolList({ current: 1, pageSize: 50, status: 0 });
      setPools(result.data?.list || []);
    } catch (error) {
      console.error("获取值班池列表失败", error);
    }
  }, []);

  // 获取用户列表（只获取启用的）
  const fetchUsers = useCallback(async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error("获取用户列表失败", error);
    }
  }, []);

  // 获取周度值班数据
  const fetchWeeklyDuty = useCallback(async (weekStart: Dayjs) => {
    try {
      const weekEnd = weekStart.endOf("week");
      const year = weekStart.year();
      const monthStart = weekStart.month() + 1;
      const monthEnd = weekEnd.month() + 1;

      // 如果周跨月，需要获取两个月的数据
      if (monthStart === monthEnd) {
        const result = await getMonthlyDutySchedule(year, monthStart);
        setWeeklyDutyData(result.data || {});
      } else {
        // 跨月情况，获取两个月的数据并合并
        const [result1, result2] = await Promise.all([
          getMonthlyDutySchedule(year, monthStart),
          getMonthlyDutySchedule(year, monthEnd),
        ]);
        setWeeklyDutyData({
          ...(result1.data || {}),
          ...(result2.data || {}),
        });
      }
    } catch (error) {
      console.error("获取周度值班失败", error);
    }
  }, []);

  return {
    dataSource,
    total,
    allSchedules,
    pools,
    users,
    weeklyDutyData,
    loading,
    currentWeekStart,
    fetchList,
    fetchAllSchedules,
    fetchPools,
    fetchUsers,
    fetchWeeklyDuty,
    setCurrentWeekStart,
  };
}

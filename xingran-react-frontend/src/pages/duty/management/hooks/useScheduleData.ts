import { useState, useCallback } from "react";
import { App } from "antd";
import {
  getDutyScheduleList,
  generateSchedule,
  swapDuty,
  manualDuty,
  deleteDutySchedule,
  batchDeleteDutySchedules,
  getMonthlyDutySchedule,
  type DutySchedule,
  type GenerateScheduleRequest,
  type ManualDutyRequest,
  type MonthlyDutyMember,
} from "@/lib/dutyApi";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";

interface UseScheduleDataOptions {
  poolId?: string;
  userId?: string;
  dutyType?: string;
  startDate?: string;
  endDate?: string;
  expired?: number;
}

type DutyType = "weekday" | "weekend" | "holiday";

export function useScheduleData() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [schedules, setSchedules] = useState<DutySchedule[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [allSchedules, setAllSchedules] = useState<DutySchedule[]>([]);

  // 周视图数据
  const [weeklyDutyData, setWeeklyDutyData] = useState<Record<string, MonthlyDutyMember[]>>({});
  const [currentWeekStart, setCurrentWeekStart] = useState(dayjs().startOf("week"));

  // 获取排班列表
  const fetchList = useCallback(
    async (page?: number, size?: number, options?: UseScheduleDataOptions) => {
      setLoading(true);
      try {
        const result = await getDutyScheduleList({
          current: page ?? current,
          pageSize: size ?? pageSize,
          poolId: options?.poolId,
          userId: options?.userId,
          startDate: options?.startDate,
          endDate: options?.endDate,
          dutyType: options?.dutyType,
          expired: options?.expired,
        });
        setSchedules(result.data?.list ?? []);
        setTotal(result.data?.total ?? 0);
        if (result.data?.current) setCurrent(result.data.current);
        if (result.data?.pageSize) setPageSize(result.data.pageSize);
      } catch (error) {
        message.error("获取排班列表失败");
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize]
  );

  // 获取所有排班（用于调班）
  const fetchAllSchedules = useCallback(async () => {
    try {
      const result = await getDutyScheduleList({ current: 1, pageSize: 50 });
      setAllSchedules(result.data?.list ?? []);
    } catch (error) {
      console.error("获取所有排班失败", error);
    }
  }, []);

  // 获取周度值班数据
  const fetchWeeklyDuty = useCallback(async (weekStart: Dayjs) => {
    try {
      const weekEnd = weekStart.endOf("week");
      const yearStart = weekStart.year();
      const yearEnd = weekEnd.year();
      const monthStart = weekStart.month() + 1;
      const monthEnd = weekEnd.month() + 1;

      // 处理跨年情况
      if (yearStart !== yearEnd) {
        const [result1, result2] = await Promise.all([
          getMonthlyDutySchedule(yearStart, monthStart),
          getMonthlyDutySchedule(yearEnd, monthEnd),
        ]);
        setWeeklyDutyData({
          ...(result1.data || {}),
          ...(result2.data || {}),
        });
      } else if (monthStart === monthEnd) {
        // 同年同月
        const result = await getMonthlyDutySchedule(yearStart, monthStart);
        setWeeklyDutyData(result.data || {});
      } else {
        // 同年不同月
        const [result1, result2] = await Promise.all([
          getMonthlyDutySchedule(yearStart, monthStart),
          getMonthlyDutySchedule(yearStart, monthEnd),
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

  // 生成排班
  const generate = useCallback(
    async (data: Omit<GenerateScheduleRequest, "dutyType"> & { dutyType: string }) => {
      try {
        const result = await generateSchedule({
          ...data,
          dutyType: data.dutyType as DutyType,
        });
        message.success(`成功生成 ${result.data?.count || 0} 条排班记录`);
        await fetchList();
        await fetchWeeklyDuty(currentWeekStart);
        return true;
      } catch (error) {
        message.error("生成排班失败");
        return false;
      }
    },
    [fetchList, fetchWeeklyDuty, currentWeekStart]
  );

  // 调班
  const swap = useCallback(
    async (data: { fromScheduleId: string; toScheduleId: string; reason: string }) => {
      try {
        await swapDuty(data);
        message.success("调班成功");
        await fetchList();
        await fetchWeeklyDuty(currentWeekStart);
        return true;
      } catch (error) {
        message.error("调班失败");
        return false;
      }
    },
    [fetchList, fetchWeeklyDuty, currentWeekStart]
  );

  // 手动排班
  const manual = useCallback(
    async (data: Omit<ManualDutyRequest, "dutyType"> & { dutyType: string }) => {
      try {
        await manualDuty({
          ...data,
          dutyType: data.dutyType as DutyType,
        });
        message.success("手动排班成功");
        await fetchList();
        await fetchWeeklyDuty(currentWeekStart);
        return true;
      } catch (error) {
        message.error("手动排班失败");
        return false;
      }
    },
    [fetchList, fetchWeeklyDuty, currentWeekStart]
  );

  // 删除排班
  const deleteOne = useCallback(
    async (id: string) => {
      try {
        await deleteDutySchedule(id);
        message.success("删除成功");
        // 计算正确的页码：如果当前页只有一条数据且不是第一页，则向前翻页
        const newPage = schedules.length === 1 && current > 1 ? current - 1 : current;
        await fetchList(newPage);
        await fetchWeeklyDuty(currentWeekStart);
        return true;
      } catch (error) {
        message.error("删除失败");
        return false;
      }
    },
    [schedules.length, current, fetchList, fetchWeeklyDuty, currentWeekStart]
  );

  // 批量删除排班
  const batchDelete = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) {
        message.warning("请先选择要删除的排班记录");
        return false;
      }
      try {
        await batchDeleteDutySchedules(ids);
        message.success(`成功删除 ${ids.length} 条排班记录`);
        setSelectedRowKeys([]);
        // 如果删除了当前页的所有数据且不是第一页，则向前翻页
        const isCurrentPageCleared = ids.length >= schedules.length;
        const newPage = isCurrentPageCleared && current > 1 ? current - 1 : current;
        await fetchList(newPage);
        await fetchWeeklyDuty(currentWeekStart);
        return true;
      } catch (error) {
        message.error("批量删除失败");
        return false;
      }
    },
    [schedules.length, current, fetchList, fetchWeeklyDuty, currentWeekStart]
  );

  // 周视图导航
  const prevWeek = useCallback(() => {
    const newWeekStart = currentWeekStart.subtract(1, "week");
    setCurrentWeekStart(newWeekStart);
    // 直接传递新值，避免状态异步更新的竞态条件
    fetchWeeklyDuty(newWeekStart);
  }, [currentWeekStart, fetchWeeklyDuty]);

  const nextWeek = useCallback(() => {
    const newWeekStart = currentWeekStart.add(1, "week");
    setCurrentWeekStart(newWeekStart);
    // 直接传递新值，避免状态异步更新的竞态条件
    fetchWeeklyDuty(newWeekStart);
  }, [currentWeekStart, fetchWeeklyDuty]);

  const todayWeek = useCallback(() => {
    const todayWeekStart = dayjs().startOf("week");
    setCurrentWeekStart(todayWeekStart);
    // 直接传递新值，避免状态异步更新的竞态条件
    fetchWeeklyDuty(todayWeekStart);
  }, [fetchWeeklyDuty]);

  return {
    loading,
    schedules,
    total,
    current,
    pageSize,
    selectedRowKeys,
    setSelectedRowKeys,
    allSchedules,
    weeklyDutyData,
    currentWeekStart,
    setCurrentWeekStart,
    fetchList,
    fetchAllSchedules,
    fetchWeeklyDuty,
    generate,
    swap,
    manual,
    deleteOne,
    batchDelete,
    prevWeek,
    nextWeek,
    todayWeek,
  };
}

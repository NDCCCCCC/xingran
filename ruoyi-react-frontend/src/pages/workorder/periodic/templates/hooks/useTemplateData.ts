/**
 * Periodic Template Data Hook
 * 周期性工单模板数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import {
  getPeriodicTemplateList,
  getPeriodicTemplateStatistics,
  getPeriodicLogs,
  type PeriodicWorkOrderTemplate,
  type PeriodicWorkOrderLog,
} from "@/lib/workorderApi";
import {
  getEnabledWorkOrderCategories,
  type WorkOrderCategory,
} from "@/lib/workorderApi";
import {
  getUserList,
  type SimpleUser,
} from "@/lib/workorderApi";
import {
  getDutyPoolList,
  type DutyPool,
} from "@/lib/dutyApi";

export interface TemplateStatistics {
  total: number;
  enabled: number;
  disabled: number;
  totalGenerated: number;
}

export interface UseTemplateDataReturn {
  // 数据状态
  dataSource: PeriodicWorkOrderTemplate[];
  total: number;
  loading: boolean;
  categories: WorkOrderCategory[];
  users: SimpleUser[];
  dutyPools: DutyPool[];
  statistics: TemplateStatistics;
  logs: PeriodicWorkOrderLog[];
  selectedTemplate: PeriodicWorkOrderTemplate | null;

  // 数据加载方法
  fetchList: (page?: number, pageSize?: number) => Promise<void>;
  fetchCategories: () => Promise<void>;
  fetchUsers: () => Promise<void>;
  fetchDutyPools: () => Promise<void>;
  fetchLogs: (templateId: string) => Promise<void>;
  setSelectedTemplate: (template: PeriodicWorkOrderTemplate | null) => void;
}

export function useTemplateData(searchForm: FormInstance<unknown>, current: number, pageSize: number): UseTemplateDataReturn {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<PeriodicWorkOrderTemplate[]>([]);
  const [total, setTotal] = useState(0);
  const [categories, setCategories] = useState<WorkOrderCategory[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);
  const [dutyPools, setDutyPools] = useState<DutyPool[]>([]);
  const [logs, setLogs] = useState<PeriodicWorkOrderLog[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<PeriodicWorkOrderTemplate | null>(null);
  const [statistics, setStatistics] = useState<TemplateStatistics>({
    total: 0,
    enabled: 0,
    disabled: 0,
    totalGenerated: 0,
  });

  // 获取统计数据（专用端点 COUNT 聚合，全局计数，不受分页/筛选影响）。
  // 旧实现用当前页 list 算 total/enabled/disabled/totalGenerated，多页时严重偏小。
  const fetchStats = useCallback(async () => {
    try {
      const result = await getPeriodicTemplateStatistics();
      setStatistics({
        total: result.data?.total ?? 0,
        enabled: result.data?.enabled ?? 0,
        disabled: result.data?.disabled ?? 0,
        totalGenerated: result.data?.totalGenerated ?? 0,
      });
    } catch (error) {
      console.error("获取模板统计失败:", error);
    }
  }, []);

  // 获取列表数据
  const fetchList = useCallback(async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue() as { title?: string; isEnabled?: boolean };
      const result = await getPeriodicTemplateList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
        title: values.title,
        isEnabled: values.isEnabled,
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);

      // 列表加载后顺带刷新统计（全局 COUNT）；搜索/分页/增删改均经 fetchList，统计始终为真实全局计数。
      fetchStats();
    } catch {
      message.error("获取模板列表失败");
    } finally {
      setLoading(false);
    }
  }, [current, pageSize, searchForm, message, fetchStats]);

  // 获取分类列表
  const fetchCategories = useCallback(async () => {
    try {
      const result = await getEnabledWorkOrderCategories();
      setCategories(result.data || []);
    } catch (error) {
      console.error("获取分类列表失败:", error);
    }
  }, []);

  // 获取用户列表
  const fetchUsers = useCallback(async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error("获取用户列表失败:", error);
    }
  }, []);

  // 获取值班池列表
  const fetchDutyPools = useCallback(async () => {
    try {
      const result = await getDutyPoolList({ status: 0 });
      setDutyPools(result.data?.list || []);
    } catch (error) {
      console.error("获取值班池列表失败:", error);
    }
  }, []);

  // 获取执行记录
  const fetchLogs = useCallback(async (templateId: string) => {
    try {
      const result = await getPeriodicLogs(templateId);
      setLogs(result.data || []);
    } catch (error) {
      console.error("获取执行记录失败:", error);
    }
  }, []);

  return {
    dataSource,
    total,
    loading,
    categories,
    users,
    dutyPools,
    statistics,
    logs,
    selectedTemplate,
    fetchList,
    fetchCategories,
    fetchUsers,
    fetchDutyPools,
    fetchLogs,
    setSelectedTemplate,
  };
}

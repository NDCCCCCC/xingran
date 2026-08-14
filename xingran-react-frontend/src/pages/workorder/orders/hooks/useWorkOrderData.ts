/**
 * WorkOrder 数据管理 Hook
 */

import { useState, useEffect, useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import {
  getWorkOrderList,
  getWorkOrderStatusStatistics,
  getUserList,
  getEnabledWorkOrderCategories,
  type WorkOrder,
  type SimpleUser,
  type SimpleDept,
  type WorkOrderCategory,
} from "@/lib/workorderApi";
import { useDeptTree } from "@/hooks/useDeptTree";

export interface WorkOrderListParams {
  current?: number;
  pageSize?: number;
  workOrderNo?: string;
  title?: string;
  categoryId?: string;
  type?: string;
  priority?: number;
  status?: number;
  assigneeId?: string;
  deptId?: string;
  orderByColumn?: string;
  isAsc?: boolean;
}

export interface WorkOrderStats {
  total: number;
  pending: number;
  processing: number;
  completed: number;
  closed: number;
}

export interface UseWorkOrderDataOptions {
  form: FormInstance;
}

export interface UseWorkOrderDataReturn {
  loading: boolean;
  dataSource: WorkOrder[];
  total: number;
  current: number;
  pageSize: number;
  stats: WorkOrderStats;
  users: SimpleUser[];
  depts: SimpleDept[];
  categories: WorkOrderCategory[];
  fetchList: (page?: number, pageSize?: number, sortParams?: { orderByColumn?: string; isAsc?: boolean }) => Promise<void>;
  fetchStats: () => Promise<void>;
  fetchUsers: () => Promise<void>;
  fetchCategories: () => Promise<void>;
}

export function useWorkOrderData(
  options: UseWorkOrderDataOptions
): UseWorkOrderDataReturn {
  const { form } = options;
  const { message } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<WorkOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [stats, setStats] = useState<WorkOrderStats>({
    total: 0,
    pending: 0,
    processing: 0,
    completed: 0,
    closed: 0,
  });

  const [users, setUsers] = useState<SimpleUser[]>([]);
  // 部门树数据 — 全项目共享 ['dept','tree'] 缓存条目 (D-LOCKED: 单一数据源 useDeptTree)
  const { data: depts = [] } = useDeptTree();
  const [categories, setCategories] = useState<WorkOrderCategory[]>([]);

  // 获取统计数据（专用端点 COUNT 聚合，全局计数，不受分页/筛选影响）。
  // 旧实现用当前页 list.filter() 算统计，多页或筛选时严重偏小。
  const fetchStats = useCallback(async () => {
    try {
      const result = await getWorkOrderStatusStatistics();
      setStats({
        total: result.data?.total ?? 0,
        pending: result.data?.pending ?? 0,
        processing: result.data?.processing ?? 0,
        completed: result.data?.completed ?? 0,
        closed: result.data?.closed ?? 0,
      });
    } catch (error) {
      console.error("获取工单统计失败:", error);
    }
  }, []);

  // 获取列表数据;sortParams 携带 orderByColumn/isAsc(由 useServerSort 注入)
  const fetchList = useCallback(async (page?: number, size?: number, sortParams?: { orderByColumn?: string; isAsc?: boolean }) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue() as Record<string, unknown>;
      const params: WorkOrderListParams = {
        current: page ?? current,
        pageSize: size ?? pageSize,
        workOrderNo: values.workOrderNo as string,
        title: values.title as string,
        categoryId: values.categoryId as string,
        type: values.type as string,
        priority: values.priority as number,
        status: values.status as number,
        assigneeId: values.assigneeId as string,
        deptId: values.deptId as string,
        ...(sortParams?.orderByColumn ? { orderByColumn: sortParams.orderByColumn, isAsc: sortParams.isAsc } : {}),
      };

      const result = await getWorkOrderList(params);
      const list = result.data?.list ?? [];
      setDataSource(list);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.pageSize ?? 10);

      // 列表加载后顺带刷新统计(全局 COUNT,不受分页/筛选影响)。
      // 这样搜索/分页/增删改(均经 fetchList)都会保持统计卡片为真实全局计数。
      fetchStats();
    } catch (_error) {
      message.error("获取工单列表失败");
    } finally {
      setLoading(false);
    }
  }, [form, current, pageSize, message, fetchStats]);

  // 获取用户列表
  const fetchUsers = useCallback(async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error("获取用户列表失败:", error);
    }
  }, []);

  // 部门树数据由顶层 useDeptTree() 提供,不再手动 fetch (workorderApi.getDeptTree 副本已删,类型 re-export from dutyApi)

  // 获取工单分类
  const fetchCategories = useCallback(async () => {
    try {
      const result = await getEnabledWorkOrderCategories();
      setCategories(result.data || []);
    } catch (error) {
      console.error("获取工单分类失败:", error);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    fetchList(1, 10); // 内部会顺带刷新统计
    fetchUsers();
    fetchCategories();
  }, [fetchList, fetchUsers, fetchCategories]);

  return {
    loading,
    dataSource,
    total,
    current,
    pageSize,
    stats,
    users,
    depts,
    categories,
    fetchList,
    fetchStats,
    fetchUsers,
    fetchCategories,
  };
}

/**
 * WorkOrder 数据管理 Hook
 */

import { useState, useEffect, useCallback } from "react";
import { message } from "antd";
import type { FormInstance } from "antd/es/form";
import {
  getWorkOrderList,
  getUserList,
  getDeptTree,
  getEnabledWorkOrderCategories,
  type WorkOrder,
  type SimpleUser,
  type SimpleDept,
  type WorkOrderCategory,
  WorkOrderStatus,
} from "@/lib/workorderApi";

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
  fetchList: (page?: number, pageSize?: number) => Promise<void>;
  fetchUsers: () => Promise<void>;
  fetchDepts: () => Promise<void>;
  fetchCategories: () => Promise<void>;
}

export function useWorkOrderData(
  options: UseWorkOrderDataOptions
): UseWorkOrderDataReturn {
  const { form } = options;

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
  const [depts, setDepts] = useState<SimpleDept[]>([]);
  const [categories, setCategories] = useState<WorkOrderCategory[]>([]);

  // 获取列表数据
  const fetchList = useCallback(async (page?: number, size?: number) => {
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
      };

      const result = await getWorkOrderList(params);
      const list = result.data?.list ?? [];
      setDataSource(list);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.pageSize ?? 10);

      // 计算统计数据
      setStats({
        total: list.length,
        pending: list.filter((o) => o.status === WorkOrderStatus.Pending).length,
        processing: list.filter((o) => o.status === WorkOrderStatus.Processing).length,
        completed: list.filter((o) => o.status === WorkOrderStatus.Completed).length,
        closed: list.filter((o) => o.status === WorkOrderStatus.Closed).length,
      });
    } catch (error) {
      message.error("获取工单列表失败");
    } finally {
      setLoading(false);
    }
  }, [form, current, pageSize]);

  // 获取用户列表
  const fetchUsers = useCallback(async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error("获取用户列表失败:", error);
    }
  }, []);

  // 获取部门列表
  const fetchDepts = useCallback(async () => {
    try {
      const result = await getDeptTree();
      setDepts(result.data || []);
    } catch (error) {
      console.error("获取部门列表失败:", error);
    }
  }, []);

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
    fetchList(1, 10);
    fetchUsers();
    fetchDepts();
    fetchCategories();
  }, [fetchList, fetchUsers, fetchDepts, fetchCategories]);

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
    fetchUsers,
    fetchDepts,
    fetchCategories,
  };
}

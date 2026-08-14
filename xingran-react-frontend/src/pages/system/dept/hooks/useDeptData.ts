/**
 * Department Data Hook
 * 部门数据管理 Hook
 */

import { useState, useCallback, useEffect } from "react";
import type { Department } from "@/types";
import { post, get } from "@/lib/api";
import type { DeptStatistics, DeptUser, ParentOption } from "../types";
import { flattenTreeToList, transformToParentTreeOptions } from "../utils";

export interface UseDeptDataReturn {
  departments: Department[];
  parentOptions: ParentOption[];
  deptUsers: DeptUser[];
  loading: boolean;
  loadingUsers: boolean;
  statistics: DeptStatistics;

  setDepartments: React.Dispatch<React.SetStateAction<Department[]>>;
  setParentOptions: React.Dispatch<React.SetStateAction<ParentOption[]>>;
  setDeptUsers: React.Dispatch<React.SetStateAction<DeptUser[]>>;

  loadDepartments: (searchParams?: Record<string, unknown>) => Promise<void>;
  loadDeptUsers: (deptId: string) => Promise<void>;
}

export function useDeptData(): UseDeptDataReturn {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [parentOptions, setParentOptions] = useState<ParentOption[]>([]);
  const [deptUsers, setDeptUsers] = useState<DeptUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);

  const [statistics, setStatistics] = useState<DeptStatistics>({
    total: 0,
    topLevel: 0,
    subLevel: 0,
  });

  const loadDepartments = useCallback(async (searchParams?: Record<string, unknown>) => {
    setLoading(true);
    try {
      const result = (await post("/system/departments/tree", searchParams || {})) as {
        data: Department[];
      };
      const deptData = result.data || [];
      setDepartments(deptData);

      const flatList = flattenTreeToList(deptData);

      // 组建父级选项（树形结构）
      const options = transformToParentTreeOptions(deptData);
      setParentOptions(options);

      // 统计数据
      setStatistics({
        total: flatList.length,
        topLevel: flatList.filter((d) => !d.parentId).length,
        subLevel: flatList.filter((d) => d.parentId).length,
      });
    } catch (error) {
      console.error("加载部门列表失败:", error);
      setDepartments([]);
      setParentOptions([{ title: "顶级部门", value: "", key: "" }]);
      setStatistics({ total: 0, topLevel: 0, subLevel: 0 });
    } finally {
      setLoading(false);
    }
  }, []);

  const loadDeptUsers = useCallback(async (deptId: string) => {
    setLoadingUsers(true);
    try {
      const result = (await get(`/system/departments/${deptId}/users`)) as { data: DeptUser[] };
      setDeptUsers(result.data || []);
    } catch (error) {
      console.error("加载部门用户失败:", error);
      setDeptUsers([]);
    } finally {
      setLoadingUsers(false);
    }
  }, []);

  return {
    departments,
    parentOptions,
    deptUsers,
    loading,
    loadingUsers,
    statistics,
    setDepartments,
    setParentOptions,
    setDeptUsers,
    loadDepartments,
    loadDeptUsers,
  };
}

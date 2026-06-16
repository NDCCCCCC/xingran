import { useState, useCallback, useEffect } from "react";
import { post } from "@/lib/api";
import type { BaseResponse } from "@/types";

interface DepartmentOption {
  id: string;
  deptName: string;
  children?: DepartmentOption[];
}

// 后端返回的部门节点类型
interface DepartmentNode {
  id: string;
  deptName: string;
  children?: DepartmentNode[];
}

/**
 * 部门数据管理 Hook
 * 处理部门列表的加载和格式转换
 */
export function useDepartmentData() {
  const [departments, setDepartments] = useState<DepartmentOption[]>([]);
  const [loading, setLoading] = useState(false);

  const loadDepartments = useCallback(async () => {
    setLoading(true);
    try {
      const result = await post<DepartmentOption[]>(
        "/system/departments/tree",
        {}
      );
      setDepartments(convertToTreeData(result.data || []));
    } catch (error) {
      console.error("加载部门列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 转换为 TreeSelect 需要的格式
  const convertToTreeData = (nodes: DepartmentNode[]): DepartmentOption[] => {
    return nodes.map(node => ({
      id: node.id,
      deptName: node.deptName,
      children: (node.children?.length ?? 0) > 0 ? convertToTreeData(node.children!) : undefined,
    }));
  };

  // 扁平化部门数据用于查找
  const getDeptMap = useCallback(() => {
    const deptMap = new Map<string, string>();
    const flattenDeptsToMap = (nodes: DepartmentOption[]) => {
      nodes.forEach(node => {
        deptMap.set(node.id, node.deptName);
        if ((node.children?.length ?? 0) > 0) {
          flattenDeptsToMap(node.children!);
        }
      });
    };
    flattenDeptsToMap(departments);
    return deptMap;
  }, [departments]);

  // 获取所属机构名称
  const getOrgName = useCallback((orgId?: string): string => {
    if (!orgId) return "-";
    const deptMap = getDeptMap();
    return deptMap.get(orgId) || "-";
  }, [getDeptMap]);

  return {
    departments,
    loading,
    loadDepartments,
    getOrgName,
  };
}

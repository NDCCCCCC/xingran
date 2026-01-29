/**
 * 部门树数据处理 Hook
 */

import { useState, useCallback, useMemo } from 'react';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';
import type { BaseResponse } from '@/types';

interface DepartmentOption {
  id: string;
  deptName: string;
  children?: DepartmentOption[];
}

export function useDepartmentTree() {
  const [departments, setDepartments] = useState<DepartmentOption[]>([]);
  const [loading, setLoading] = useState(false);

  const loadDepartments = useCallback(async () => {
    setLoading(true);
    try {
      const result = await post<BaseResponse<DepartmentOption[]>>('/system/departments/tree', {});
      const convertToTreeData = (nodes: any[]): DepartmentOption[] => {
        return nodes.map(node => ({
          id: node.id,
          deptName: node.deptName,
          children: node.children?.length > 0 ? convertToTreeData(node.children) : undefined,
        }));
      };
      setDepartments(convertToTreeData(result.data || []));
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
    } finally {
      setLoading(false);
    }
  }, []);

  // 扁平化部门数据为 Map
  const deptMap = useMemo(() => {
    const map = new Map<string, string>();
    const flattenDeptsToMap = (nodes: DepartmentOption[]) => {
      nodes.forEach(node => {
        map.set(node.id, node.deptName);
        if (node.children?.length > 0) {
          flattenDeptsToMap(node.children);
        }
      });
    };
    flattenDeptsToMap(departments);
    return map;
  }, [departments]);

  const getOrgName = useCallback((orgId?: string) => {
    if (!orgId) return '-';
    return deptMap.get(orgId) || '-';
  }, [deptMap]);

  return {
    departments,
    loading,
    loadDepartments,
    getOrgName,
  };
}

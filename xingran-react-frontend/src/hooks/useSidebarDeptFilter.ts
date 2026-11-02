/**
 * 部门侧边栏筛选 Hook
 * 用于运营管理页面的部门树筛选功能
 */

import { useState, useCallback } from "react";
import type { Key } from "react";
import type { FormInstance } from "antd/es/form";
import type { DataNode } from "antd/es/tree";

interface UseSidebarDeptFilterOptions {
  /** 搜索表单实例，用于在部门变化时清空相关筛选 */
  searchForm?: FormInstance;
  /** 部门变化时需要清空的表单字段名 */
  clearFieldNames?: string[];
  /** 部门变化时的回调函数 */
  onDeptChange?: (deptId: string) => void;
}

interface UseSidebarDeptFilterReturn {
  /** 选中的部门 ID */
  selectedDeptId: string;
  /** 设置选中的部门 ID */
  setSelectedDeptId: (deptId: string) => void;
  /** 部门树选择处理函数 */
  handleDeptSelect: (selectedKeys: Key[], info: { selected: boolean; node: DataNode }) => void;
}

/**
 * 管理部门侧边栏筛选的状态和逻辑
 */
export function useSidebarDeptFilter(
  options: UseSidebarDeptFilterOptions = {}
): UseSidebarDeptFilterReturn {
  const { searchForm, clearFieldNames = [], onDeptChange } = options;

  const [selectedDeptId, setSelectedDeptId] = useState<string>("");

  const handleDeptSelect = useCallback(
    (selectedKeys: Key[], _info: { selected: boolean; node: DataNode }) => {
      const deptId = selectedKeys.length > 0 ? (selectedKeys[0] as string) : "";
      setSelectedDeptId(deptId);

      // 清空指定的表单字段
      if (searchForm && clearFieldNames.length > 0) {
        clearFieldNames.forEach((fieldName) => {
          searchForm.setFieldValue(fieldName, undefined);
        });
      }

      // 触发部门变化回调
      onDeptChange?.(deptId);
    },
    [searchForm, clearFieldNames, onDeptChange]
  );

  return {
    selectedDeptId,
    setSelectedDeptId,
    handleDeptSelect,
  };
}

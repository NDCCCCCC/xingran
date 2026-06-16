/**
 * 合并搜索表单 + 分页 + 部门筛选的通用工具
 * 用于消除 buildings/workstations 中重复的"表单值过滤 + 部门ID注入 + 分页合并"逻辑
 */
import type { FormInstance } from "antd";

export interface BuildSearchParamsOptions {
  searchForm?: FormInstance<unknown>;
  deptId?: string;
  page?: { current?: number; pageSize?: number };
}

/**
 * 构建搜索参数：
 * 1. 从 searchForm 提取非空字段
 * 2. 注入 deptId（如果提供且非空）
 * 3. 注入分页参数（如果提供）
 */
export function buildSearchParams(opts: BuildSearchParamsOptions): Record<string, unknown> {
  const { searchForm, deptId, page } = opts;
  const params: Record<string, unknown> = {};

  if (searchForm) {
    const values = searchForm.getFieldsValue() as Record<string, unknown>;
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        params[key] = value;
      }
    });
  }

  if (deptId) {
    params.orgId = deptId;
  }

  if (page) {
    if (page.current !== undefined) params.current = page.current;
    if (page.pageSize !== undefined) params.pageSize = page.pageSize;
  }

  return params;
}

/**
 * 表格配置 Hook
 * Table Settings Hook
 *
 * 集成分页、排序、筛选等配置
 */

import { usePagination } from "./usePagination";
import type { TableProps } from "antd";

export interface UseTableSettingsOptions {
  // 是否启用全局分页（默认 true）
  enableGlobalPagination?: boolean;

  // 页面级分页覆盖
  pageSize?: number;
}

/**
 * 表格配置 Hook
 *
 * 集成分页、排序、筛选等配置，自动应用用户设置
 *
 * @example
 * const { pagination, getTableProps } = useTableSettings();
 *
 * <Table
 *   {...getTableProps()}
 *   columns={columns}
 *   dataSource={data}
 * />
 */
export function useTableSettings(options: UseTableSettingsOptions = {}) {
  const { enableGlobalPagination = true, pageSize } = options;

  // 获取分页配置
  const pagination = usePagination({
    pageSize: enableGlobalPagination ? pageSize : undefined,
  });

  /**
   * 获取 Table props
   * 自动集成分页配置
   */
  const getTableProps = <T = unknown>(): Partial<TableProps<T>> => ({
    pagination: pagination.paginationProps,
  });

  return {
    pagination,
    getTableProps,
  };
}

export default useTableSettings;

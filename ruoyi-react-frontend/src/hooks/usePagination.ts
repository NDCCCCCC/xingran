/**
 * 全局分页 Hook
 * Global Pagination Hook
 *
 * 自动从用户配置读取默认分页大小
 */

import { useState, useCallback, useMemo } from 'react';
import { useSettingsStore } from '@/store/settingsStore';
import type { TablePaginationConfig } from 'antd/es/table';

export interface UsePaginationOptions {
	// 页面级覆盖（可选）
	pageSize?: number;

	// 回调
	onChange?: (page: number, pageSize: number) => void;
}

export interface UsePaginationReturn {
	// 状态
	current: number;
	pageSize: number;
	total: number;

	// 操作
	setCurrent: (page: number) => void;
	setPageSize: (size: number) => void;
	setTotal: (total: number) => void;
	reset: () => void;

	// Ant Design Pagination props（可直接传给 Table）
	paginationProps: TablePaginationConfig;
}

/**
 * 全局分页 Hook
 * 自动从用户配置读取默认分页大小
 *
 * @example
 * const { current, pageSize, paginationProps } = usePagination();
 *
 * <Table
 *   pagination={paginationProps}
 *   // ...
 * />
 */
export function usePagination(options: UsePaginationOptions = {}): UsePaginationReturn {
	const { preferences } = useSettingsStore();

	// 优先级：页面覆盖 > 用户配置 > 默认值
	const defaultPageSize = options.pageSize
		?? preferences?.data?.defaultPageSize
		?? 10;

	const [current, setCurrent] = useState(1);
	const [pageSize, setPageSize] = useState(defaultPageSize);
	const [total, setTotal] = useState(0);

	// 处理分页变化
	const handlePageChange = useCallback((page: number, size: number) => {
		setCurrent(page);
		setPageSize(size);
		options.onChange?.(page, size);
	}, [options]);

	// 重置
	const reset = useCallback(() => {
		setCurrent(1);
		setPageSize(defaultPageSize);
	}, [defaultPageSize]);

	// Ant Design Pagination 配置
	const paginationProps: TablePaginationConfig = useMemo(() => ({
		current,
		pageSize,
		total,
		showSizeChanger: true,
		showQuickJumper: true,
		showTotal: (t) => `共 ${t} 条`,
		onChange: handlePageChange,
	}), [current, pageSize, total, handlePageChange]);

	return {
		current,
		pageSize,
		total,
		setCurrent,
		setPageSize,
		setTotal,
		reset,
		paginationProps,
	};
}

export default usePagination;

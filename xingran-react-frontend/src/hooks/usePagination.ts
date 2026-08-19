/**
 * 全局分页 Hook
 * Global Pagination Hook
 *
 * 自动从用户配置读取默认分页大小，并将 current/pageSize 持久化到 sessionStorage
 * （按 location.pathname 隔离），使切换应用内标签页 / F5 刷新后能恢复到上次的页码与页大小。
 * total 为运行时查询结果，不持久化。
 *
 * 优先级：页面显式 options.pageSize > 持久化值 > 用户配置 defaultPageSize > 10。
 * 关闭标签页 / 登出由 tabsStore.removeTab / authStore.logout 通过
 * clearTableStateByPath / clearAllTableState 统一清理。
 */

import { useState, useCallback, useMemo, useRef, useEffect } from "react";
import { useLocation } from "react-router-dom";
import { useSettingsStore } from "@/store/settingsStore";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import type { TablePaginationConfig } from "antd/es/table";

export interface UsePaginationOptions {
  // 页面级覆盖（可选）；显式传入时作为硬约束，始终优先于持久化值
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
 * 自动从用户配置读取默认分页大小，current/pageSize 持久化到 sessionStorage。
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
  const location = useLocation();

  // fallback：无持久化值时的默认页大小（页面覆盖 > 用户配置 > 10）
  const fallbackPageSize = options.pageSize ?? preferences?.data?.defaultPageSize ?? 10;

  // current 持久化（切 tab / 刷新保留页码）
  const [current, setCurrent, resetCurrent] = usePersistedStateController<number>({
    keyPrefix: location.pathname,
    keySuffix: "current",
    defaultValue: 1,
  });

  // pageSize 持久化；页面显式 options.pageSize 始终优先（硬约束，不被历史值覆盖）
  const [persistedPageSize, setPersistedPageSize, resetPageSize] =
    usePersistedStateController<number>({
      keyPrefix: location.pathname,
      keySuffix: "pageSize",
      defaultValue: fallbackPageSize,
    });
  const pageSize = options.pageSize ?? persistedPageSize;

  const setPageSize = useCallback(
    (size: number) => {
      // 页面硬约束（options.pageSize）时不改持久化值，保持显示恒定
      if (options.pageSize === undefined) {
        setPersistedPageSize(size);
      }
    },
    [options.pageSize, setPersistedPageSize]
  );

  const [total, setTotal] = useState(0);

  // 使用 ref 存储最新的 onChange 回调，避免 handlePageChange 依赖变化
  // 遵循 Vercel React Best Practices: rerender-defer-reads
  const onChangeRef = useRef(options.onChange);
  useEffect(() => {
    onChangeRef.current = options.onChange;
  }, [options.onChange]);

  // 处理分页变化 - setCurrent/setPageSize 来自持久化 setter（签名与 useState 兼容）
  const handlePageChange = useCallback(
    (page: number, size: number) => {
      setCurrent(page);
      setPageSize(size);
      onChangeRef.current?.(page, size);
    },
    [setCurrent, setPageSize]
  );

  // 重置：回到第 1 页 + 默认页大小（清理持久化）
  const reset = useCallback(() => {
    resetCurrent();
    resetPageSize();
  }, [resetCurrent, resetPageSize]);

  // Ant Design Pagination 配置
  const paginationProps: TablePaginationConfig = useMemo(
    () => ({
      current,
      pageSize,
      total,
      showSizeChanger: true,
      showQuickJumper: true,
      showTotal: (t) => `共 ${Math.max(1, Math.ceil(t / pageSize))} 页`,
      onChange: handlePageChange,
    }),
    [current, pageSize, total, handlePageChange]
  );

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

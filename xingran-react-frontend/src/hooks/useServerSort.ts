import { useCallback, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import type { TablePaginationConfig, FilterValue, SorterResult } from "antd/es/table/interface";
import type { SorterMeta } from "@/utils/tableHelpers";

/**
 * 排序方向字面量,与 antd Table sorter.order 保持一致。
 * 注意: undefined / null / "" 表示"未排序"（清空状态）。
 */
export type SortOrder = "ascend" | "descend" | null | undefined;

export interface UseServerSortOptions<T = unknown> {
  /**
   * 列元数据数组。元素可以是 SorterMeta,也可以是 undefined（不可排序列）。
   * 传入后,hook 会按"用户点的 sorter 落在哪一列"自动解析出 orderByColumn。
   */
  sorterMetas?: Array<SorterMeta<T> | undefined>;
  /**
   * 默认排序。可选;同时作为"无持久化值时"的首屏排序。
   */
  defaultSort?: { orderByColumn: string; isAsc?: boolean };
}

export interface UseServerSortReturn {
  /** 后端白名单字段名(对应 BaseListRequest.OrderByColumn);undefined 表示无排序 */
  orderByColumn: string | undefined;
  /** 后端排序方向:true=升序 false=降序 undefined=无排序 */
  isAsc: boolean | undefined;
  /** 当前 antd 排序方向字面量("ascend"/"descend"/null),由 orderByColumn/isAsc 派生 */
  sortOrder: SortOrder;
  /**
   * 接 antd Table onChange 第三参数。
   * 用法:
   *   onChange={(pagination, filters, sorter) => {
   *     handleTableChange(pagination, filters, sorter);
   *     // 同时记得 setCurrent/setPageSize + fetchList
   *   }}
   */
  handleTableChange: <T>(
    _pagination: TablePaginationConfig,
    _filters: Record<string, FilterValue | null>,
    sorter: SorterResult<T> | SorterResult<T>[]
  ) => void;
  /**
   * 主动重置排序（回到 undefined 状态）。
   * 适用于"清空筛选时连排序一起清"的场景。
   */
  resetSort: () => void;
}

/**
 * 通用服务端排序 hook:把 antd Table 的 sorter 状态翻译成
 * 后端 BaseListRequest 的 { orderByColumn, isAsc } 参数，并按 location.pathname
 * 持久化到 sessionStorage，使切换应用内标签页 / 刷新后能恢复到上次的排序。
 *
 * 设计要点:
 *   - 零依赖:不调用 antd message / 不修改 URL;持久化走 sessionStorage
 *   - sortOrder 由 orderByColumn/isAsc 派生,不单独持久化（避免三态不一致）
 *   - 兼容单列与多列(取最后一列有效 sorter;antd v5 多列排序通常被禁用)
 *   - 兜底:若 sorter.field 不在 sorterMetas 列表中(用户点了不可排序列),
 *     则忽略该次操作并保持当前 state,避免污染
 *
 * 用法:
 *
 *   const { orderByColumn, isAsc, handleTableChange } = useServerSort<User>({
 *     sorterMetas: [
 *       createSorterMeta<User>("username"),
 *       createSorterMeta<User>("createdAt", "date"),
 *       // undefined for action column
 *     ],
 *   });
 *
 *   const result = await post("/system/users/list", {
 *     current, pageSize,
 *     orderByColumn, isAsc,           // 新增
 *     ...filters,
 *   });
 *
 *   <Table
 *     onChange={(pagination, filters, sorter) => {
 *       handleTableChange(pagination, filters, sorter);
 *       setCurrent(pagination.current ?? 1);
 *       setPageSize(pagination.pageSize ?? 10);
 *       fetchList();
 *     }}
 *   />
 */
export function useServerSort<T = unknown>(
  options: UseServerSortOptions<T> = {}
): UseServerSortReturn {
  const { sorterMetas = [], defaultSort } = options;
  const location = useLocation();

  // 持久化排序：defaultSort 作为"无持久化值时"的首屏值与 reset 回退目标。
  const [orderByColumn, setOrderByColumn] = usePersistedStateController<string | undefined>({
    keyPrefix: location.pathname,
    keySuffix: "orderByColumn",
    defaultValue: defaultSort?.orderByColumn,
  });
  const [isAsc, setIsAsc] = usePersistedStateController<boolean | undefined>({
    keyPrefix: location.pathname,
    keySuffix: "isAsc",
    defaultValue: defaultSort?.isAsc ?? true,
  });

  // sortOrder 派生（不单独持久化，避免与 orderByColumn/isAsc 三态不一致）
  const sortOrder: SortOrder = useMemo(() => {
    if (orderByColumn === undefined) return null;
    return isAsc === false ? "descend" : "ascend";
  }, [orderByColumn, isAsc]);

  const resetSort = useCallback(() => {
    // 清空排序：强制 undefined（即使存在 defaultSort 也回到无排序状态）。
    // 注：setValue(undefined) 写入的脏值会在下次 mount 时由 readInitial 自愈回 defaultValue。
    setOrderByColumn(undefined);
    setIsAsc(undefined);
  }, [setOrderByColumn, setIsAsc]);

  const handleTableChange = useCallback(
    <R>(
      _pagination: TablePaginationConfig,
      _filters: Record<string, FilterValue | null>,
      sorter: SorterResult<R> | SorterResult<R>[]
    ) => {
      // antd 偶发传数组(多列模式);取最后一个有效 sorter
      const s = Array.isArray(sorter) ? sorter[sorter.length - 1] : sorter;
      if (!s || !s.field) {
        // 用户点了"清空"或不可排序列
        resetSort();
        return;
      }

      // 仅当 sorter.field 对应 sorterMetas 中已注册的可排序列时才更新 state
      const meta = sorterMetas.find((m) => m !== undefined && m.field === String(s.field));
      if (!meta) {
        // 未在白名单的列(可能是不该排序列,如操作列)→ 忽略
        return;
      }

      const order = s.order;
      if (order === "ascend") {
        setOrderByColumn(meta.field);
        setIsAsc(true);
      } else if (order === "descend") {
        setOrderByColumn(meta.field);
        setIsAsc(false);
      } else {
        // antd 'null' / undefined: 清空排序
        setOrderByColumn(undefined);
        setIsAsc(undefined);
      }
    },
    [sorterMetas, resetSort, setOrderByColumn, setIsAsc]
  );

  return { orderByColumn, isAsc, sortOrder, handleTableChange, resetSort };
}

/**
 * 纯函数:从 antd Table 的 sorter 解析出后端排序参数 { orderByColumn, isAsc }。
 *
 * 用途:antd 服务端排序在 onChange 内同步获取新排序值,避免 useServerSort 的
 * setState 时序问题(同一事件周期内读 hook state 会得到旧值)。
 *
 *   onChange={(pagination, filters, sorter) => {
 *     handleTableChange(pagination, filters, sorter); // 更新受控 UI state
 *     const { orderByColumn, isAsc } = resolveSorter(sorter, sorterMetas); // 同步拿新值
 *     loadUsers({ current, pageSize, orderByColumn, isAsc });
 *   }}
 *
 * 返回 { orderByColumn: undefined, isAsc: undefined } 表示无排序(清空状态)。
 * 注:antd 仅对 sorter:true 的列触发 sorter onChange,而这些列理应在 sorterMetas 注册,
 * 故"有 field 但未注册"在实践中不会触发,兜底返回 undefined。
 */
export function resolveSorter<T>(
  sorter: SorterResult<T> | SorterResult<T>[],
  sorterMetas: Array<SorterMeta<T> | undefined>
): { orderByColumn: string | undefined; isAsc: boolean | undefined } {
  const s = Array.isArray(sorter) ? sorter[sorter.length - 1] : sorter;
  if (!s || !s.field) {
    return { orderByColumn: undefined, isAsc: undefined };
  }
  const meta = sorterMetas.find((m) => m !== undefined && m.field === String(s.field));
  if (!meta) {
    return { orderByColumn: undefined, isAsc: undefined };
  }
  if (s.order === "ascend") return { orderByColumn: meta.field, isAsc: true };
  if (s.order === "descend") return { orderByColumn: meta.field, isAsc: false };
  return { orderByColumn: undefined, isAsc: undefined };
}

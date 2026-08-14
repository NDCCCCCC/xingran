/**
 * Widget 数据过滤器
 * 根据配置在数据请求时添加过滤参数
 */
import { useAuthStore } from "@/store/authStore";
import type { WidgetConfig, WidgetDataFilter } from "@/types/dashboard";

/**
 * 应用 Widget 数据过滤
 * 根据 Widget 配置在数据请求时添加过滤参数
 *
 * @param widget - Widget 配置
 * @param baseParams - 基础请求参数
 * @returns 应用过滤后的参数
 */
export const applyWidgetDataFilter = (
  widget: WidgetConfig,
  baseParams: Record<string, unknown>
): Record<string, unknown> => {
  if (!widget.dataFilter?.enabled) {
    return baseParams;
  }

  const { filterType, filterConfig } = widget.dataFilter;
  const user = useAuthStore.getState().user;

  if (!user) {
    return baseParams;
  }

  switch (filterType) {
    case "dept": {
      // 按部门过滤
      const field = filterConfig.field || "deptId";
      return {
        ...baseParams,
        [field]: user.deptId,
      };
    }

    case "user": {
      // 按用户过滤
      const userField = filterConfig.field || "userId";
      return {
        ...baseParams,
        [userField]: user.id,
      };
    }

    case "custom":
      // 自定义过滤
      return {
        ...baseParams,
        ...filterConfig,
      };

    default:
      return baseParams;
  }
};

/**
 * 获取用户可用的数据过滤配置
 * 返回用户可选择的过滤类型和默认配置
 */
export const getAvailableDataFilters = (): Array<{
  value: string;
  label: string;
  defaultConfig: Partial<WidgetDataFilter>;
}> => {
  return [
    {
      value: "none",
      label: "不过滤",
      defaultConfig: { enabled: false, filterType: "dept", filterConfig: {} },
    },
    {
      value: "dept",
      label: "按部门过滤",
      defaultConfig: {
        enabled: true,
        filterType: "dept",
        filterConfig: { field: "deptId" },
      },
    },
    {
      value: "user",
      label: "按用户过滤",
      defaultConfig: {
        enabled: true,
        filterType: "user",
        filterConfig: { field: "userId" },
      },
    },
    {
      value: "custom",
      label: "自定义过滤",
      defaultConfig: {
        enabled: true,
        filterType: "custom",
        filterConfig: {},
      },
    },
  ];
};

export default { applyWidgetDataFilter, getAvailableDataFilters };

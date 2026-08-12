/**
 * 页面标题常量
 * 用于标签页标题显示
 */

export const PAGE_TITLES = {
  DASHBOARD: "仪表盘",
  MONITOR_DASHBOARD: "监控仪表盘",
  HOME: "首页",
  NOTICE_DETAIL: "通知详情",
} as const;

/**
 * 特殊路径标题映射
 * 用于直接通过路径获取标题的场景
 */
export const SPECIAL_PATH_TITLES: Record<string, string> = {
  "/": PAGE_TITLES.HOME,
  "/dashboard": PAGE_TITLES.DASHBOARD,
  "/monitor/dashboard": PAGE_TITLES.MONITOR_DASHBOARD,
} as const;

/**
 * 动态路由模式配置
 * 用于匹配带参数的路由（如详情页）
 */
export interface DynamicRoutePattern {
  pattern: RegExp;
  title: string;
}

export const DYNAMIC_ROUTE_PATTERNS: DynamicRoutePattern[] = [
  {
    pattern: /^\/my-notices\/[a-f0-9-]+$/i,
    title: PAGE_TITLES.NOTICE_DETAIL,
  },
];

/**
 * 根据路径匹配动态路由标题
 * @param path - 当前路径
 * @returns 匹配到的标题，如果未匹配则返回 null
 */
export function matchDynamicRouteTitle(path: string): string | null {
  for (const { pattern, title } of DYNAMIC_ROUTE_PATTERNS) {
    if (pattern.test(path)) {
      return title;
    }
  }
  return null;
}

/**
 * 获取特殊路径标题
 * @param path - 当前路径
 * @returns 特殊路径标题，如果未匹配则返回 null
 */
export function getSpecialPathTitle(path: string): string | null {
  return SPECIAL_PATH_TITLES[path] || null;
}

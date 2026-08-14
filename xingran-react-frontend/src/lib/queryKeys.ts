/**
 * Centralized React Query key factory.
 *
 * Prevents key-string drift across components by giving every queryable
 * resource a single source of truth for its query key shape.
 *
 * Conventions:
 * - All keys are tuples (`as const`) so consumers get narrow literal types.
 * - The root segment identifies the resource family (e.g. 'dict', 'dept').
 * - Nested factories accept their discriminators (dict type, role params, etc.).
 */

import type { ExceptionListParams, FixSuggestionListParams } from "@/lib/assetApi";

export const queryKeys = {
  dict: {
    all: ["dict"] as const,
    list: (dictType: string) => ["dict", dictType] as const,
  },
  list: {
    all: (resource: string) => ["list", resource] as const,
    page: (resource: string, params: Record<string, unknown>) =>
      ["list", resource, params] as const,
  },
  dept: {
    all: ["dept"] as const,
    tree: () => ["dept", "tree"] as const,
  },
  duty: {
    all: ["duty"] as const,
    // 值班池成员候选:按所选部门(+子部门)缓存,切换部门回切时即时显示(stale-while-revalidate)
    poolMembers: (deptId: string) => ["duty", "pool-members", deptId] as const,
  },
  role: {
    all: ["role"] as const,
    list: (params?: Record<string, unknown>) => ["role", "list", params ?? {}] as const,
  },
  // Phase 39: 工位部门物理位置映射 — 按 location 缓存,用于工位编辑下拉 union 注入
  locationAlias: {
    all: ["location-alias"] as const,
    byLocation: (locationId: string) => ["location-alias", "by-location", locationId] as const,
    list: (params?: Record<string, unknown>) => ["location-alias", "list", params ?? {}] as const,
  },
  // Phase 42 R1: 资产对账观测底座 — Dashboard + 异常列表
  reconciliation: {
    all: ["reconciliation"] as const,
    /** 5 KPI 卡片聚合 */
    summary: (windowDays: number) => ["reconciliation", "summary", windowDays] as const,
    /** 按冲突类型分组(Type A-F 6 keys) */
    byConflictType: (windowDays: number) =>
      ["reconciliation", "by-conflict-type", windowDays] as const,
    /** 按严重级别分组(low/medium/high/critical 4 keys) */
    bySeverity: (windowDays: number) => ["reconciliation", "by-severity", windowDays] as const,
    /** 健康度趋势点序列(7d/30d/90d) */
    healthTrend: (windowDays: number) => ["reconciliation", "health-trend", windowDays] as const,
    /** Top N 长期未解决异常(默认 limit=10) */
    topUnresolved: (limit: number) => ["reconciliation", "top-unresolved", limit] as const,
    /** 异常列表(分页 + 筛选 + 服务端排序) */
    exceptionList: (params: ExceptionListParams) =>
      ["reconciliation", "exception-list", params] as const,
    /** 异常详情(单条) */
    exceptionDetail: (id: string) => ["reconciliation", "exception-detail", id] as const,
    /** 例外规则命中统计(R3 启用后才有数据) */
    ruleStats: () => ["reconciliation", "rule-stats"] as const,
    /** 例外规则列表(admin 页,分页 + 筛选) */
    ruleList: (params?: object) => ["reconciliation", "rule-list", params ?? {}] as const,
    /** 例外规则详情(单条,编辑回填) */
    ruleDetail: (id: string) => ["reconciliation", "rule-detail", id] as const,
    /** 命中测试结果(按 IP+userID+deptID 入参缓存) */
    matchTest: (params: { ip: string; userId?: string; deptId?: string }) =>
      ["reconciliation", "match-test", params] as const,
    /** 降噪基线对比结果(dashboard 降噪卡片) */
    baselineCompare: () => ["reconciliation", "baseline-compare"] as const,
    /** Phase 45 R4 — 工位对账健康度(POST /asset/reconciliation/by-workstation) */
    workstationHealth: (workstationId: string) =>
      ["reconciliation", "workstation-health", workstationId] as const,
    /** Phase 45 R4 — 资产对账健康度(从 workstationHealth 切片,无独立端点) */
    assetHealth: (assetId: string) => ["reconciliation", "asset-health", assetId] as const,
    /** Phase 46 R5 — 修复建议列表(分页 + 筛选) */
    fixSuggestionList: (params: FixSuggestionListParams) =>
      ["reconciliation", "fix-suggestion-list", params] as const,
    /** Phase 46 R5 — 修复建议详情 */
    fixSuggestionDetail: (id: string) => ["reconciliation", "fix-suggestion-detail", id] as const,
    /** Phase 46 R5 — 修复建议统计(7d KPI 卡片 + 误修复率) */
    fixSuggestionStats: (windowDays: number) =>
      ["reconciliation", "fix-suggestion-stats", windowDays] as const,
  },
} as const;

export type QueryKeys = typeof queryKeys;

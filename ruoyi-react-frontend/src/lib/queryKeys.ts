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
} as const;

export type QueryKeys = typeof queryKeys;
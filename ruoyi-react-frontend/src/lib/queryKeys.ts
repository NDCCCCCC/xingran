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
} as const;

export type QueryKeys = typeof queryKeys;
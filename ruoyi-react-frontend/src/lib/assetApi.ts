/**
 * 资产对账模块 API 工厂 (Phase 42 R1)
 *
 * 依据 .planning/phases/42-r1/42-05-PLAN.md:
 *   - 6 statistics 端点(summary / byConflictType / bySeverity / healthTrend / topUnresolved / exceptionRuleStats)
 *   - 2 exception 端点(exceptionList / exceptionGetByID)
 *
 * 后端契约:internal/api/v1/asset/reconciliation_statistics_handler.go + reconciliation_handler.go
 * 类型严格对齐后端 JSON 字段(无 list.length 路径,所有统计走独立 COUNT/GROUP BY 端点,见 memory: stat-cards-from-list-length-capped-at-100)
 */

import { post } from "@/lib/api";

// ==================== 类型定义 ====================

/**
 * 5 KPI 卡片数据(D-06)
 */
export interface SummaryResult {
  /** 全量资产数(SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL) */
  totalAssets: number;
  /** 未解决异常数(resolved_at IS NULL AND deleted_at IS NULL) */
  openExceptions: number;
  /** critical 级未解决数 */
  criticalOpen: number;
  /** 7d 新增异常数(detected_at >= NOW() - INTERVAL '7 days') */
  last7dNew: number;
  /** Top1 冲突类型(空字符串表示无数据) */
  topConflictType: string;
  /** Top1 冲突类型计数(0 表示无数据) */
  topConflictCount: number;
}

/**
 * 7d/30d 健康度趋势点
 */
export interface TrendPoint {
  /** YYYY-MM-DD */
  date: string;
  /** 当日未解决异常数 */
  openCount: number;
  /** 当日 critical 级未解决数 */
  criticalCount: number;
  /** 当日新增异常数 */
  newCount: number;
}

/**
 * Top N 长期未解决异常摘要(TopUnresolved 端点)
 */
export interface ExceptionSummary {
  id: string;
  assetCode: string;
  conflictType: string;
  severity: string;
  detectedAt: string;
  daysUnresolved: number;
}

/**
 * 例外规则命中统计(R3 启用后才有数据,R1 返回空 slice)
 */
export interface RuleStats {
  ruleId: string;
  ruleName: string;
  matchedCount: number;
}

/**
 * 异常列表查询参数(POST /asset/reconciliation/exception/list)
 */
export interface ExceptionListParams {
  conflictType?: string;
  severity?: string;
  assetCode?: string;
  detectedFrom?: string;
  detectedTo?: string;
  /** R3 / D-R3-A1-01: 是否显示 silence 记录(默认 false 隐藏) */
  showSilenced?: boolean;
  current: number;
  pageSize: number;
  orderByColumn?: string;
  isAsc?: boolean;
}

/**
 * 异常列表返回行(JOIN ops_asset + sys_user + reconciliation_normalized)
 *
 * 字段名与后端 reconciliation_service.go ExceptionListItem 严格对齐:
 *   - assetCode / assetIpDisplay(避开 SysDataReconciliation.AssetIP 冲突,见 42-02 SUMMARY)
 *   - physicalUsername / responsibleUsername
 *   - exceptionRuleId 可空(R3 命中才非空)
 */
export interface ExceptionListItem {
  id: string;
  conflictType: string;
  severity: string;
  confidenceScore: number;
  detectedAt: string;
  resolvedAt: string | null;
  exceptionRuleId: string | null;
  assetCode: string;
  assetIpDisplay: string;
  physicalUsername: string;
  responsibleUsername: string;
}

export interface PageResult<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// ==================== Phase 45 R4 — Types ====================

/**
 * 工位对账健康度聚合响应(POST /asset/reconciliation/by-workstation)
 * D-A4-02 锁定:Workstation/HealthScore/Assets/Visible
 */
export interface ByWorkstationResponse {
  workstation: { id: string; name: string; code: string };
  healthScore: HealthScore;
  assets: AssetHealthItem[];
  /** 权限降级标记(D-A1-03):false 时 Assets 已被后端清空,前端渲染 "-" */
  visible: boolean;
}

/**
 * 5 KPI 卡片数据 + 趋势 — 复用 R1 TrendPoint 形态
 * 字段命名严格对齐后端 HealthScore struct
 */
export interface HealthScore {
  total: number;
  normal: number;
  drift: number;
  conflict: number;
  noData: number;
  exceptionHit: number;
  score: number;
  trend: TrendPoint[];
}

export interface AssetHealthItem {
  assetId: string;
  assetCode: string;
  /** 冲突类型 A-F 或 ""(健康) */
  conflictType: string;
  /** 严重度 low/medium/high/critical 或 "" */
  severity: string;
  exceptionRuleId?: string | null;
  appliedActions?: string[];
  confidenceScore: number;
  /** 解析后的 IP(asset.ip → workstation.ip → "unknown",D-A4-02) */
  ip?: string;
}

// ==================== reconciliationApi 工厂 ====================

/**
 * ReconciliationApi — 8 个端点(6 statistics + 2 exception)
 *
 * 使用统一的 post() 封装,NOT raw axios(见 CLAUDE.md 强约束)。
 * 每个方法返回后端响应的 data 字段(BaseResponse.data)。
 */
export const reconciliationApi = {
  /**
   * 5 KPI 卡片聚合
   * @param days 统计窗口(默认 7d,R1 走 7d,handler/service 已 clamp 到 [1, 365])
   */
  summary: async (days: number = 7): Promise<SummaryResult> => {
    const res = await post<SummaryResult>(
      "/asset/reconciliation/statistics/summary",
      { days }
    );
    return res.data as SummaryResult;
  },

  /**
   * 按冲突类型分组(Type A-F 6 keys,seed merge 保证空数据 = 0)
   */
  byConflictType: async (days: number = 7): Promise<Record<string, number>> => {
    const res = await post<Record<string, number>>(
      "/asset/reconciliation/statistics/by-conflict-type",
      { days }
    );
    return (res.data ?? {}) as Record<string, number>;
  },

  /**
   * 按严重级别分组(low/medium/high/critical 4 keys)
   */
  bySeverity: async (days: number = 7): Promise<Record<string, number>> => {
    const res = await post<Record<string, number>>(
      "/asset/reconciliation/statistics/by-severity",
      { days }
    );
    return (res.data ?? {}) as Record<string, number>;
  },

  /**
   * 健康度趋势(7d/30d/90d 切换)
   * 返回 TrendPoint[](date/openCount/criticalCount/newCount)
   */
  healthTrend: async (days: number = 7): Promise<TrendPoint[]> => {
    const res = await post<TrendPoint[]>(
      "/asset/reconciliation/statistics/health-trend",
      { days }
    );
    return (res.data ?? []) as TrendPoint[];
  },

  /**
   * Top N 长期未解决异常(默认 limit=10,按 detected_at ASC 排序)
   * R4 工位详情页可能复用。
   */
  topUnresolved: async (limit: number = 10): Promise<ExceptionSummary[]> => {
    const res = await post<ExceptionSummary[]>(
      "/asset/reconciliation/statistics/top-unresolved",
      { limit }
    );
    return (res.data ?? []) as ExceptionSummary[];
  },

  /**
   * 例外规则命中统计(R3 启用后才有数据,R1 返回空)
   */
  exceptionRuleStats: async (): Promise<RuleStats[]> => {
    const res = await post<RuleStats[]>(
      "/asset/reconciliation/statistics/exception-rule-stats",
      {}
    );
    return (res.data ?? []) as RuleStats[];
  },

  /**
   * 异常列表(带分页 + 筛选 + 服务端排序)
   */
  exceptionList: async (
    params: ExceptionListParams
  ): Promise<PageResult<ExceptionListItem>> => {
    const res = await post<PageResult<ExceptionListItem>>(
      "/asset/reconciliation/exception/list",
      params
    );
    return (
      (res.data ?? { list: [], total: 0, current: 1, pageSize: 20 }) as PageResult<ExceptionListItem>
    );
  },

  /**
   * 异常详情(单条)
   */
  exceptionGetByID: async (id: string): Promise<ExceptionListItem> => {
    const res = await post<ExceptionListItem>(
      `/asset/reconciliation/exception/${id}`,
      {}
    );
    return res.data as ExceptionListItem;
  },

  /**
   * 标记异常已解决 (Phase 43 R2 / D-A4-04)
   * 后端:POST /asset/reconciliation/exception/:id/resolve
   * Body: { resolutionNote?: string }
   * 权限:前端按钮按 asset:reconciliation:resolve perm 控制可见性(R3 后端强制)
   * OperLog:后端 handler 自动写 sys_oper_log(OperTypeUpdate,WORKORDER-02)
   */
  exceptionResolve: async (
    id: string,
    body: { resolutionNote?: string } = {}
  ): Promise<{ id: string; resolvedAt: string; resolvedBy: string; resolutionNote?: string }> => {
    const res = await post<{ id: string; resolvedAt: string; resolvedBy: string; resolutionNote?: string }>(
      `/asset/reconciliation/exception/${id}/resolve`,
      body
    );
    return res.data as { id: string; resolvedAt: string; resolvedBy: string; resolutionNote?: string };
  },

  /**
   * Phase 45 R4 — 工位对账健康度聚合(D-A4-01/02/03)
   *
   * 单次拿完顶部卡片 + 资产子表徽标 + 详情跳转锚点(避免 N+1,与 SC7 一致)。
   * 缓存 TTL 5min(后端 CacheProvider.GetOrSet,handler 注入 visible 标记)。
   */
  byWorkstation: async (data: {
    workstationId: string;
    window?: string;
  }): Promise<ByWorkstationResponse> => {
    const res = await post<ByWorkstationResponse>(
      "/asset/reconciliation/by-workstation",
      {
        workstationId: data.workstationId,
        window: data.window ?? "7d",
      }
    );
    return res.data as ByWorkstationResponse;
  },

  /**
   * Phase 44 R3 — 例外规则 CRUD + 命中测试 (admin 页用)
   *
   * 后端契约:internal/api/v1/asset/reconciliation_exception_{handler,router}.go
   * 权限命名空间:asset:reconciliation:exception:{list,create,update,delete,test}
   */
  exceptionRule: {
    list: async <T = unknown>(
      params: Record<string, unknown> = {}
    ): Promise<PageResult<T>> => {
      const res = await post<PageResult<T>>(
        "/asset/reconciliation/exception-rule/list",
        params
      );
      return (
        (res.data ?? { list: [], total: 0, current: 1, pageSize: 20 }) as PageResult<T>
      );
    },
    getById: async <T = unknown>(
      id: string
    ): Promise<T> => {
      const res = await post<T>(
        `/asset/reconciliation/exception-rule/${id}`,
        {}
      );
      return res.data as T;
    },
    create: async <T = unknown>(
      data: Record<string, unknown>
    ): Promise<T> => {
      const res = await post<T>(
        "/asset/reconciliation/exception-rule/create",
        data
      );
      return res.data as T;
    },
    update: async <T = unknown>(
      id: string,
      data: Record<string, unknown>
    ): Promise<T> => {
      const res = await post<T>(
        `/asset/reconciliation/exception-rule/${id}/update`,
        data
      );
      return res.data as T;
    },
    delete: async <T = unknown>(id: string): Promise<T> => {
      const res = await post<T>(
        `/asset/reconciliation/exception-rule/${id}/delete`,
        {}
      );
      return res.data as T;
    },
    test: async (data: {
      ip: string;
      userId?: string;
      deptId?: string;
    }): Promise<{
      matchedRules: unknown[];
      mergedActions: string[];
      finalSeverity: string;
      isSilence: boolean;
      needsUserDept: boolean;
    }> => {
      const res = await post<{
        matchedRules: unknown[];
        mergedActions: string[];
        finalSeverity: string;
        isSilence: boolean;
        needsUserDept: boolean;
      }>("/asset/reconciliation/exception-rule/test", data);
      return (
        (res.data ?? {
          matchedRules: [],
          mergedActions: [],
          finalSeverity: "",
          isSilence: false,
          needsUserDept: false,
        }) as {
          matchedRules: unknown[];
          mergedActions: string[];
          finalSeverity: string;
          isSilence: boolean;
          needsUserDept: boolean;
        }
      );
    },
  },

  /**
   * Phase 44 R3 — 降噪基线 snapshot / compare (D-R3-A4-01)
   *
   * 后端契约:44-02 plan 实现(本 plan 仅 wire 前端基建,UI 44-02 接入)
   * 调用 reconciliationApi.baseline.snapshot() 记录当前为基线,
   * 后续 dashboard "降噪效果"卡片读 compare 对比下降百分比。
   */
  baseline: {
    snapshot: async (): Promise<{ snapshot_at: string }> => {
      const res = await post<{ snapshot_at: string }>(
        "/asset/reconciliation/baseline/snapshot",
        {}
      );
      return res.data as { snapshot_at: string };
    },
    compare: async (): Promise<{
      baseline: { snapshot_at: string; total_exceptions: number; total_workorders: number; critical_exceptions: number };
      current: { total_exceptions: number; total_workorders: number; critical_exceptions: number };
      reductions: Record<string, number>;
    }> => {
      const res = await post<{
        baseline: { snapshot_at: string; total_exceptions: number; total_workorders: number; critical_exceptions: number };
        current: { total_exceptions: number; total_workorders: number; critical_exceptions: number };
        reductions: Record<string, number>;
      }>("/asset/reconciliation/baseline/compare", {});
      return (
        (res.data ?? {
          baseline: { snapshot_at: "", total_exceptions: 0, total_workorders: 0, critical_exceptions: 0 },
          current: { total_exceptions: 0, total_workorders: 0, critical_exceptions: 0 },
          reductions: {},
        }) as {
          baseline: { snapshot_at: string; total_exceptions: number; total_workorders: number; critical_exceptions: number };
          current: { total_exceptions: number; total_workorders: number; critical_exceptions: number };
          reductions: Record<string, number>;
        }
      );
    },
  },
  /**
   * Phase 45 R5: 手动刷新对账(运维/UAT 调试用)
   * 1) REFRESH MATERIALIZED VIEW reconciliation_normalized
   * 2) 立即 DetectLayer3(绕过 5min/6min cron)
   * 返回 inserted / skipped / skippedSilence / skippedThrottle 计数
   */
  refresh: async (): Promise<{
    inserted: number;
    skipped: number;
    skippedSilence: number;
    skippedThrottle: number;
  }> => {
    const res = await post<{
      inserted: number;
      skipped: number;
      skippedSilence: number;
      skippedThrottle: number;
    }>("/asset/reconciliation/refresh", {});
    // res.data 实际是 T | undefined(BaseResponse 契约),这里业务契约保证非空,
    // 兜底 {} 以满足返回类型。
    return res.data ?? { inserted: 0, skipped: 0, skippedSilence: 0, skippedThrottle: 0 };
  },
};

// ==================== Phase 46 R5 — 修复建议 Types + API ====================

/**
 * 修复建议状态机 6 状态(D-B2 锁定)
 */
export type FixStatus =
  | "pending"
  | "accepted"
  | "rejected"
  | "applied"
  | "rolled_back"
  | "failed";

/**
 * 修复建议列表查询参数
 * (POST /asset/reconciliation/fix-suggestion/list)
 */
export interface FixSuggestionListParams {
  fixStatus?: FixStatus;
  conflictType?: "A" | "B" | "C" | "D" | "E" | "F";
  responsibleDeptId?: string;
  createdFrom?: string;
  createdTo?: string;
  current: number;
  pageSize: number;
  orderByColumn?: string;
  isAsc?: boolean;
}

/**
 * 修复建议列表行(JOIN sys_data_reconciliation + ops_asset + sys_user)
 */
export interface FixSuggestionListItem {
  id: string;
  exceptionId: string;
  assetId: string;
  assetCode: string;
  conflictType: "A" | "B" | "C" | "D" | "E" | "F";
  currentUserId: string | null;
  suggestedUserId: string;
  suggestedUsername: string | null;
  preFixUserId: string | null;
  confidenceScore: number;
  reason: string;
  fixStatus: FixStatus;
  acceptedAt: string | null;
  rejectedAt: string | null;
  appliedAt: string | null;
  rolledBackAt: string | null;
  rejectionReason: string | null;
  rollbackReason: string | null;
  rollbackWindowUntil: string | null;
  createdAt: string;
  updatedAt: string;
  acceptedBy: string | null;
  rejectedBy: string | null;
  appliedBy: string | null;
  rolledBackBy: string | null;
  /** 同 exception 的全部 fix_suggestion 数量(详情页用) */
  superseded: boolean;
}

/**
 * 修复建议详情(包含异常元数据 + 同 exception_id 历史)
 */
export interface FixSuggestionDetail {
  suggestion: FixSuggestionListItem;
  exception?: {
    id: string;
    assetId: string;
    assetCode?: string;
    conflictType: string;
    severity: string;
    confidenceScore: number;
    rawSnapshot: object;
    detectedAt: string;
    [key: string]: unknown;
  };
  history: FixSuggestionListItem[];
}

/**
 * 7d 统计响应(D-C5)
 */
export interface FixSuggestionStatsResponse {
  windowDays: number;
  /** 7d 窗口内 created pending */
  pending: number;
  /** 全量 pending(无 7d 窗口,W-I3 修订) */
  pendingAll: number;
  accepted: number;
  rejected: number;
  /** 按 applied_at 过滤(W-2 修订) */
  applied: number;
  /** 按 rolled_back_at 过滤 */
  rolledBack: number;
  failed: number;
  misFixRate: number;
  threshold: number;
  thresholdBreached: boolean;
  trendSeries: TrendPoint[];
}

/**
 * 修复建议 API 工厂(7 个端点)
 */
export const fixSuggestionApi = {
  /**
   * 列表(分页 + 过滤 + 白名单排序)
   */
  list: async (
    params: FixSuggestionListParams
  ): Promise<PageResult<FixSuggestionListItem>> => {
    const res = await post<PageResult<FixSuggestionListItem>>(
      "/asset/reconciliation/fix-suggestion/list",
      params
    );
    return (
      (res.data ?? { list: [], total: 0, current: 1, pageSize: 20 }) as PageResult<FixSuggestionListItem>
    );
  },

  /**
   * 详情(单条 + 异常 + 历史)
   */
  getById: async (id: string): Promise<FixSuggestionDetail> => {
    const res = await post<FixSuggestionDetail>(
      `/asset/reconciliation/fix-suggestion/${id}`,
      {}
    );
    return res.data as FixSuggestionDetail;
  },

  /**
   * 接受(pending → accepted)
   */
  accept: async (id: string): Promise<{ id: string; acceptedBy: string }> => {
    const res = await post<{ id: string; acceptedBy: string }>(
      `/asset/reconciliation/fix-suggestion/${id}/accept`,
      {}
    );
    return res.data as { id: string; acceptedBy: string };
  },

  /**
   * 拒绝(pending → rejected,reason ≥10 字符)
   */
  reject: async (
    id: string,
    rejectionReason: string
  ): Promise<{ id: string; rejectedBy: string; rejectionReason: string }> => {
    const res = await post<{
      id: string;
      rejectedBy: string;
      rejectionReason: string;
    }>(`/asset/reconciliation/fix-suggestion/${id}/reject`, {
      rejectionReason,
    });
    return res.data as {
      id: string;
      rejectedBy: string;
      rejectionReason: string;
    };
  },

  /**
   * 应用(accepted → applied,写 ops_asset.user_id)
   */
  apply: async (id: string): Promise<{ id: string; appliedBy: string }> => {
    const res = await post<{ id: string; appliedBy: string }>(
      `/asset/reconciliation/fix-suggestion/${id}/apply`,
      {}
    );
    return res.data as { id: string; appliedBy: string };
  },

  /**
   * 回滚(applied → rolled_back,7d 窗口内)
   */
  rollback: async (
    id: string,
    rollbackReason: string
  ): Promise<{ id: string; rolledBackBy: string; rollbackReason: string }> => {
    const res = await post<{
      id: string;
      rolledBackBy: string;
      rollbackReason: string;
    }>(`/asset/reconciliation/fix-suggestion/${id}/rollback`, {
      rollbackReason,
    });
    return res.data as {
      id: string;
      rolledBackBy: string;
      rollbackReason: string;
    };
  },

  /**
   * 7d 统计(KPI 卡片 + 误修复率监控)
   * @param windowDays 统计窗口(默认 7,后端已 clamp 到 [1, 365])
   */
  stats: async (windowDays: number = 7): Promise<FixSuggestionStatsResponse> => {
    const res = await post<FixSuggestionStatsResponse>(
      "/asset/reconciliation/fix-suggestion/stats",
      { windowDays }
    );
    return (
      (res.data ?? {
        windowDays,
        pending: 0,
        pendingAll: 0,
        accepted: 0,
        rejected: 0,
        applied: 0,
        rolledBack: 0,
        failed: 0,
        misFixRate: 0,
        threshold: 0.01,
        thresholdBreached: false,
        trendSeries: [],
      }) as FixSuggestionStatsResponse
    );
  },
};

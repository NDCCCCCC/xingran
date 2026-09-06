/**
 * RPA 系统 API 客户端（Phase 100 V130R-10/11 对账端态）
 *
 * 全族 116 方法对账后裁剪至 17 个存活方法（后端路由实际注册的实测集合,
 * 真相源 internal/api/v1/rpa/rpa_router.go）。对账台账:
 * .planning/phases/100-frontend-contract-fixes/RECONCILIATION.md。
 *
 * D-100-1/D-100-9: createResourceApi 三站点 spread 改显式 pick——工厂注入的
 * batch/statistics/searchOptions 后端无对应路由（幽灵 404 面），不得出现在
 * 导出对象上；方法集由 apiFactory.invariants.test.ts keys 基线双向锁定。
 * D-100-12: workerApi.register/heartbeat 前端方法已删（RPA Worker 是独立
 * 进程直连后端 HTTP,不经 admin bundle;后端公开路由保留不动）。
 */

import { post } from "./api";
import { createResourceApi } from "./apiFactory";
import type { PageParams, PageResponse } from "@/types/base";
import type {
  // 任务相关
  Task,
  // Worker相关
  Worker,
  // 执行相关
  Execution,
  ExecutionLog,
  // AI相关
  AIScriptGenerateRequest,
  AIScriptGenerateResponse,
  AIScriptOptimizeRequest,
  AIScriptOptimizeResponse,
  AIAgentDecisionRequest,
  AIAgentDecisionResponse,
  AIFailureAnalysisRequest,
  AIFailureAnalysisResponse,
} from "@/types/rpa";

// ==================== 任务管理 API ====================

const taskCrudApi = createResourceApi<Task>({ basePath: "/rpa/tasks" });

/**
 * RPA 任务 API（6 方法,后端 rpa_router.go:48-53）
 */
export const taskApi = {
  list: taskCrudApi.list,
  get: taskCrudApi.get,
  create: taskCrudApi.create,
  update: taskCrudApi.update,
  delete: taskCrudApi.delete,

  /**
   * 执行任务
   */
  execute: async (id: string, variables?: Record<string, unknown>) => {
    return await post<Execution>(`/rpa/tasks/${id}/execute`, { variables });
  },
};

// ==================== Worker 管理 API ====================

const workerCrudApi = createResourceApi<Worker>({ basePath: "/rpa/workers" });

/**
 * RPA Worker API（2 方法,后端 rpa_router.go:64/:66）
 */
export const workerApi = {
  list: workerCrudApi.list,

  /**
   * 获取 Worker 统计（D-100-9 收窄:POST /rpa/workers/:id/statistics 未注册,
   * id 分支已删,仅保留无参全量统计端点）
   */
  statistics: async () => {
    return await post<
      Array<{
        workerId: string;
        workerName: string;
        currentTasks: number;
        completedTasks: number;
        failedTasks: number;
      }>
    >("/rpa/workers/statistics", {});
  },
};

// ==================== 执行记录 API ====================

const executionCrudApi = createResourceApi<Execution>({ basePath: "/rpa/executions" });

/**
 * RPA 执行记录 API（5 方法,后端 rpa_router.go:84-89）
 */
export const executionApi = {
  list: executionCrudApi.list,
  get: executionCrudApi.get,
  statistics: executionCrudApi.statistics,

  /**
   * 取消执行
   */
  cancel: async (id: string, reason?: string) => {
    return await post(`/rpa/executions/${id}/cancel`, { reason });
  },

  /**
   * 获取执行日志
   */
  logs: async (id: string, params?: PageParams) => {
    return await post<PageResponse<ExecutionLog>>(`/rpa/executions/${id}/logs`, params || {});
  },
};

// ==================== AI 辅助 API ====================

/**
 * RPA AI API（4 方法,后端 rpa_router.go:101-108）
 */
export const aiApi = {
  /**
   * 自然语言生成脚本
   */
  generateScript: async (request: AIScriptGenerateRequest) => {
    return await post<AIScriptGenerateResponse>("/rpa/ai/generate", request);
  },

  /**
   * 优化脚本
   */
  optimizeScript: async (request: AIScriptOptimizeRequest) => {
    return await post<AIScriptOptimizeResponse>("/rpa/ai/optimize", request);
  },

  /**
   * AI Agent 决策（选择器失效时调用）
   */
  decide: async (request: AIAgentDecisionRequest) => {
    return await post<AIAgentDecisionResponse>("/rpa/ai/decide", request);
  },

  /**
   * 分析失败原因并修复
   */
  analyzeFailure: async (request: AIFailureAnalysisRequest) => {
    return await post<AIFailureAnalysisResponse>("/rpa/ai/analyze-failure", request);
  },
};

// ==================== 导出汇总 ====================

/**
 * RPA API 汇总导出（Phase 100 端态:4 键 17 方法）
 */
export const rpaApi = {
  task: taskApi,
  worker: workerApi,
  execution: executionApi,
  ai: aiApi,
};

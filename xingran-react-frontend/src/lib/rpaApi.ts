/**
 * RPA 系统 API 客户端
 */

import { post } from "./api";
import { getAuthHeaders } from "@/utils/authHelpers";
import type { BaseResponse, PageParams, PageResponse } from "@/types/base";
import type {
  // 任务相关
  Task,
  TaskListParams,
  TaskFormData,
  Script,
  Action,
  // Worker相关
  Worker,
  WorkerListParams,
  WorkerRegisterRequest,
  WorkerHeartbeatRequest,
  WorkerStatus,
  // 执行相关
  Execution,
  ExecutionListParams,
  RPAExecutionStatus,
  ExecutionLog,
  ExecutionProgress,
  // 调度相关
  Schedule,
  ScheduleListParams,
  ScheduleConfig,
  // 变量相关
  Variable,
  VariableListParams,
  // 模板相关
  Template,
  TemplateListParams,
  TemplateCategory,
  // AI相关
  AIScriptGenerateRequest,
  AIScriptGenerateResponse,
  AIScriptOptimizeRequest,
  AIScriptOptimizeResponse,
  AIScriptExplainRequest,
  AIScriptExplainResponse,
  AIAgentDecisionRequest,
  AIAgentDecisionResponse,
  AIFailureAnalysisRequest,
  AIFailureAnalysisResponse,
  CaptureStateRequest,
  CaptureStateResponse,
  // 通知相关
  NotificationConfig,
  NotificationListParams,
  // 统计相关
  RPAStatistics,
} from "@/types/rpa";

// ==================== 通用 CRUD 工厂函数 ====================

interface CrudApiConfig<T> {
  basePath: string;
}

function createCrudApi<T>(config: CrudApiConfig<T>) {
  const { basePath } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<PageResponse<T>>(`${basePath}/list`, params);
    },

    get: async (id: string) => {
      return await post<T>(`${basePath}/${id}`, {});
    },

    create: async (data: Partial<T>) => {
      return await post(basePath, data);
    },

    update: async (id: string, data: Partial<T>) => {
      return await post(`${basePath}/${id}/update`, data);
    },

    delete: async (id: string) => {
      return await post(`${basePath}/${id}/delete`, {});
    },
  };
}

// ==================== 任务管理 API ====================

/**
 * 任务列表查询参数
 */
export interface TaskListSearchParams extends PageParams {
  name?: string;
  status?: string;
  categoryId?: string;
  tags?: string[];
  createdBy?: string;
  dateRange?: [string, string];
}

const taskCrudApi = createCrudApi<Task>({ basePath: "/rpa/tasks" });

/**
 * RPA 任务 API
 */
export const taskApi = {
  ...taskCrudApi,

  /**
   * 执行任务
   */
  execute: async (id: string, variables?: Record<string, unknown>) => {
    return await post<Execution>(`/rpa/tasks/${id}/execute`, { variables });
  },

  /**
   * 取消正在执行的任务
   */
  cancelExecution: async (id: string) => {
    return await post(`/rpa/tasks/${id}/cancel`, {});
  },

  /**
   * 复制任务
   */
  duplicate: async (id: string, newName?: string) => {
    return await post<Task>(`/rpa/tasks/${id}/duplicate`, { newName });
  },

  /**
   * 获取任务执行历史
   */
  executions: async (id: string, params: PageParams) => {
    return await post<PageResponse<Execution>>(`/rpa/tasks/${id}/executions`, params);
  },

  /**
   * 验证脚本
   */
  validateScript: async (script: Script) => {
    return await post<{ valid: boolean; errors?: string[] }>("/rpa/tasks/validate-script", {
      script,
    });
  },

  /**
   * 导出任务
   */
  export: async (id: string) => {
    return await post<{ data: string; filename: string }>(`/rpa/tasks/${id}/export`, {});
  },

  /**
   * 导入任务
   */
  import: async (data: string) => {
    return await post<Task>("/rpa/tasks/import", { data });
  },
};

// ==================== 脚本管理 API ====================

/**
 * 脚本 API
 */
export const scriptApi = {
  /**
   * 获取脚本列表
   */
  list: async (params: PageParams) => {
    return await post<PageResponse<Script>>("/rpa/scripts/list", params);
  },

  /**
   * 获取脚本详情
   */
  get: async (id: string) => {
    return await post<Script>(`/rpa/scripts/${id}`, {});
  },

  /**
   * 创建脚本
   */
  create: async (data: Partial<Script>) => {
    return await post<Script>("/rpa/scripts", data);
  },

  /**
   * 更新脚本
   */
  update: async (id: string, data: Partial<Script>) => {
    return await post<Script>(`/rpa/scripts/${id}/update`, data);
  },

  /**
   * 删除脚本
   */
  delete: async (id: string) => {
    return await post(`/rpa/scripts/${id}/delete`, {});
  },

  /**
   * 测试脚本动作
   */
  testAction: async (action: Action, url?: string) => {
    return await post<{ success: boolean; result?: unknown; error?: string }>(
      "/rpa/scripts/test-action",
      {
        action,
        url,
      }
    );
  },

  /**
   * 格式化脚本
   */
  format: async (script: Script) => {
    return await post<Script>("/rpa/scripts/format", { script });
  },
};

// ==================== Worker 管理 API ====================

/**
 * Worker 列表查询参数
 */
export interface WorkerListSearchParams extends PageParams {
  status?: WorkerStatus;
  hostname?: string;
  name?: string;
}

const workerCrudApi = createCrudApi<Worker>({ basePath: "/rpa/workers" });

/**
 * RPA Worker API
 */
export const workerApi = {
  ...workerCrudApi,

  /**
   * 注册 Worker（Worker 节点调用）
   */
  register: async (data: WorkerRegisterRequest) => {
    return await post<{ workerId: string; token: string }>("/rpa/workers/register", data);
  },

  /**
   * 心跳上报（Worker 节点调用）
   */
  heartbeat: async (id: string, data: WorkerHeartbeatRequest) => {
    return await post(`/rpa/workers/${id}/heartbeat`, data);
  },

  /**
   * 进度上报（Worker 节点调用）
   */
  progress: async (id: string, data: ExecutionProgress) => {
    return await post(`/rpa/workers/${id}/progress`, data);
  },

  /**
   * 获取在线 Worker 列表
   */
  getOnline: async () => {
    return await post<Worker[]>("/rpa/workers/online", {});
  },

  /**
   * 获取 Worker 统计
   */
  statistics: async (id?: string) => {
    if (id) {
      return await post<{
        workerId: string;
        currentTasks: number;
        completedTasks: number;
        failedTasks: number;
        avgDuration: number;
      }>(`/rpa/workers/${id}/statistics`, {});
    }
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

  /**
   * 下线 Worker
   */
  offline: async (id: string) => {
    return await post(`/rpa/workers/${id}/offline`, {});
  },

  /**
   * 重启 Worker（Docker）
   */
  restart: async (id: string) => {
    return await post(`/rpa/workers/${id}/restart`, {});
  },
};

// ==================== 执行记录 API ====================

/**
 * 执行记录查询参数
 */
export interface ExecutionListSearchParams extends PageParams {
  taskId?: string;
  workerId?: string;
  status?: RPAExecutionStatus;
  dateRange?: [string, string];
}

const executionCrudApi = createCrudApi<Execution>({ basePath: "/rpa/executions" });

/**
 * RPA 执行记录 API
 */
export const executionApi = {
  ...executionCrudApi,

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

  /**
   * 获取实时日志流（WebSocket）
   * 返回 WebSocket URL，客户端需要建立连接
   */
  streamLogs: async (id: string) => {
    return await post<{ wsUrl: string; token: string }>(`/rpa/executions/${id}/stream-logs`, {});
  },

  /**
   * 获取执行截图
   */
  screenshots: async (id: string) => {
    return await post<string[]>(`/rpa/executions/${id}/screenshots`, {});
  },

  /**
   * 下载执行报告
   */
  downloadReport: async (id: string, format: "pdf" | "html" = "pdf") => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || "/api/v1";
    const authHeaders = await getAuthHeaders();
    const response = await fetch(`${baseUrl}/rpa/executions/${id}/report?format=${format}`, {
      method: "POST",
      headers: {
        ...authHeaders,
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      throw new Error("下载报告失败");
    }

    const blob = await response.blob();
    const blobUrl = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = blobUrl;
    a.download = `execution_report_${id}.${format}`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(blobUrl);
    document.body.removeChild(a);
  },

  /**
   * 重试执行
   */
  retry: async (id: string) => {
    return await post<Execution>(`/rpa/executions/${id}/retry`, {});
  },
};

// ==================== 定时调度 API ====================

/**
 * 定时调度查询参数
 */
export interface ScheduleListSearchParams extends PageParams {
  taskId?: string;
  status?: string;
  name?: string;
}

const scheduleCrudApi = createCrudApi<Schedule>({ basePath: "/rpa/schedules" });

/**
 * RPA 定时调度 API
 */
export const scheduleApi = {
  ...scheduleCrudApi,

  /**
   * 激活调度
   */
  activate: async (id: string) => {
    return await post(`/rpa/schedules/${id}/activate`, {});
  },

  /**
   * 暂停调度
   */
  pause: async (id: string) => {
    return await post(`/rpa/schedules/${id}/pause`, {});
  },

  /**
   * 禁用调度
   */
  disable: async (id: string) => {
    return await post(`/rpa/schedules/${id}/disable`, {});
  },

  /**
   * 立即执行一次
   */
  runNow: async (id: string) => {
    return await post<Execution>(`/rpa/schedules/${id}/run-now`, {});
  },

  /**
   * 验证 Cron 表达式
   */
  validateCron: async (expression: string) => {
    return await post<{
      valid: boolean;
      nextRuns?: string[];
      error?: string;
    }>("/rpa/schedules/validate-cron", { expression });
  },

  /**
   * 获取下次执行时间
   */
  nextRunTime: async (id: string) => {
    return await post<{ nextRunTime: string }>(`/rpa/schedules/${id}/next-run`, {});
  },
};

// ==================== 变量管理 API ====================

/**
 * 变量查询参数
 */
export interface VariableListSearchParams extends PageParams {
  scope?: "global" | "task";
  taskId?: string;
  name?: string;
}

const variableCrudApi = createCrudApi<Variable>({ basePath: "/rpa/variables" });

/**
 * RPA 变量 API
 */
export const variableApi = {
  ...variableCrudApi,

  /**
   * 获取全局变量
   */
  getGlobal: async () => {
    return await post<Variable[]>("/rpa/variables/global", {});
  },

  /**
   * 获取任务变量
   */
  getByTask: async (taskId: string) => {
    return await post<Variable[]>(`/rpa/variables/task/${taskId}`, {});
  },

  /**
   * 批量设置变量
   */
  batchSet: async (variables: Array<{ name: string; value: string }>) => {
    return await post("/rpa/variables/batch-set", { variables });
  },

  /**
   * 解密变量值
   */
  decrypt: async (id: string) => {
    return await post<{ value: string }>(`/rpa/variables/${id}/decrypt`, {});
  },
};

// ==================== 脚本模板 API ====================

/**
 * 模板查询参数
 */
export interface TemplateListSearchParams extends PageParams {
  categoryId?: string;
  tags?: string[];
  isPublic?: boolean;
  name?: string;
}

const templateCrudApi = createCrudApi<Template>({ basePath: "/rpa/templates" });

/**
 * RPA 脚本模板 API
 */
export const templateApi = {
  ...templateCrudApi,

  /**
   * 获取模板分类
   */
  categories: async () => {
    return await post<TemplateCategory[]>("/rpa/templates/categories", {});
  },

  /**
   * 使用模板创建任务
   */
  useTemplate: async (templateId: string, taskName: string) => {
    return await post<Task>(`/rpa/templates/${templateId}/use`, { taskName });
  },

  /**
   * 评分模板
   */
  rate: async (id: string, rating: number) => {
    return await post(`/rpa/templates/${id}/rate`, { rating });
  },

  /**
   * 收藏模板
   */
  favorite: async (id: string) => {
    return await post(`/rpa/templates/${id}/favorite`, {});
  },

  /**
   * 取消收藏
   */
  unfavorite: async (id: string) => {
    return await post(`/rpa/templates/${id}/unfavorite`, {});
  },
};

// ==================== AI 辅助 API ====================

/**
 * RPA AI API
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
   * 解释脚本
   */
  explainScript: async (request: AIScriptExplainRequest) => {
    return await post<AIScriptExplainResponse>("/rpa/ai/explain", request);
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

  /**
   * 捕获页面状态
   */
  captureState: async (request: CaptureStateRequest) => {
    return await post<CaptureStateResponse>("/rpa/ai/capture-state", request);
  },
};

// ==================== 通知配置 API ====================

/**
 * 通知配置查询参数
 */
export interface NotificationListSearchParams extends PageParams {
  taskId?: string;
  enabled?: boolean;
}

const notificationCrudApi = createCrudApi<NotificationConfig>({ basePath: "/rpa/notifications" });

/**
 * RPA 通知配置 API
 */
export const notificationApi = {
  ...notificationCrudApi,

  /**
   * 启用通知
   */
  enable: async (id: string) => {
    return await post(`/rpa/notifications/${id}/enable`, {});
  },

  /**
   * 禁用通知
   */
  disable: async (id: string) => {
    return await post(`/rpa/notifications/${id}/disable`, {});
  },

  /**
   * 测试通知
   */
  test: async (id: string) => {
    return await post<{ success: boolean; message?: string }>(`/rpa/notifications/${id}/test`, {});
  },

  /**
   * 获取全局通知配置
   */
  getGlobal: async () => {
    return await post<NotificationConfig[]>("/rpa/notifications/global", {});
  },
};

// ==================== 统计 API ====================

/**
 * RPA 统计 API
 */
export const statisticsApi = {
  /**
   * 获取综合统计
   */
  overview: async () => {
    return await post<RPAStatistics>("/rpa/statistics/overview", {});
  },

  /**
   * 获取任务统计
   */
  tasks: async (taskId?: string) => {
    return await post<{
      total: number;
      byStatus: Record<string, number>;
      todayExecuted: number;
      todayFailed: number;
      successRate: number;
    }>("/rpa/statistics/tasks", { taskId });
  },

  /**
   * 获取 Worker 统计
   */
  workers: async () => {
    return await post<{
      total: number;
      online: number;
      offline: number;
      busy: number;
      error: number;
      totalCapacity: number;
      usedCapacity: number;
      availableCapacity: number;
    }>("/rpa/statistics/workers", {});
  },

  /**
   * 获取执行统计
   */
  executions: async (params?: {
    taskId?: string;
    workerId?: string;
    dateRange?: [string, string];
  }) => {
    return await post<{
      total: number;
      byStatus: Record<string, number>;
      avgDuration: number;
      todayCount: number;
      weeklyTrend: Array<{
        date: string;
        count: number;
        successRate: number;
      }>;
    }>("/rpa/statistics/executions", params || {});
  },

  /**
   * 获取趋势数据
   */
  trends: async (params: {
    type: "executions" | "tasks" | "workers";
    period: "day" | "week" | "month";
    startDate?: string;
    endDate?: string;
  }) => {
    return await post<
      Array<{
        date: string;
        value: number;
        label?: string;
      }>
    >("/rpa/statistics/trends", params);
  },
};

// ==================== 导出汇总 ====================

/**
 * RPA API 汇总导出
 */
export const rpaApi = {
  task: taskApi,
  script: scriptApi,
  worker: workerApi,
  execution: executionApi,
  schedule: scheduleApi,
  variable: variableApi,
  template: templateApi,
  ai: aiApi,
  notification: notificationApi,
  statistics: statisticsApi,
};

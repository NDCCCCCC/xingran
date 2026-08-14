/**
 * RPA 系统相关类型定义
 */

import type { Status, PageParams } from "./base";

// ==================== 任务相关 ====================

/**
 * 任务状态
 */
export type TaskStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

/**
 * 动作类型
 */
export type ActionType =
  | "navigate" // 导航到URL
  | "click" // 点击元素
  | "fill" // 填写表单
  | "select" // 下拉选择
  | "wait" // 等待
  | "screenshot" // 截图
  | "extract" // 提取数据
  | "scroll" // 滚动
  | "hover" // 悬停
  | "press" // 按键
  | "upload" // 上传文件
  | "download" // 下载文件
  | "iframe"; // 切换iframe

/**
 * 选择器类型
 */
export type SelectorType =
  | "css" // CSS选择器
  | "xpath" // XPath选择器
  | "text" // 文本选择器
  | "aria" // ARIA标签
  | "data-testid"; // 测试ID

/**
 * 选择器定义
 */
export interface Selector {
  type: SelectorType;
  value: string;
  fallback?: string; // 降级选择器
}

/**
 * 动作定义
 */
export interface Action {
  id: string;
  type: ActionType;
  description?: string; // 动作描述（用于AI理解）
  selector?: Selector; // 目标选择器
  selectors?: Selector[]; // 多选择器备选
  params?: Record<string, unknown>; // 动作参数
  timeout?: number; // 超时时间（毫秒）
  retry?: number; // 重试次数
  continueOnError?: boolean; // 失败是否继续
  aiAssisted?: boolean; // 是否启用AI辅助
}

/**
 * RPA 脚本
 */
export interface Script {
  id: string;
  name: string;
  description?: string;
  actions: Action[];
  variables?: VariableValue[]; // 脚本变量
  timeout?: number; // 全局超时
  retryStrategy?: "sequential" | "parallel"; // 重试策略
  aiEnabled?: boolean; // 是否启用AI降级
  createdAt: string;
  updatedAt: string;
}

/**
 * RPA 任务
 */
export interface Task {
  id: string;
  taskName: string; // 任务名称（后端字段名）
  name?: string; // 兼容字段（可能不存在）
  description?: string;
  script?: Action[] | Script; // 脚本动作数组或脚本对象
  priority: number; // 优先级 (0-100)
  status: number; // 任务状态 (0=启用, 1=停用)
  tags?: string; // 标签（字符串格式）
  timeout?: number; // 任务超时（秒）
  retryCount?: number; // 重试次数（后端字段）
  maxRetries?: number; // 兼容字段
  retryOnFailure?: boolean; // 失败自动重试
  lastExecutionTime?: string; // 最后执行时间
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 任务列表查询参数
 */
export interface TaskListParams extends PageParams {
  name?: string;
  status?: number; // 任务状态 (0=启用, 1=停用)
  priority?: number; // 优先级
  tags?: string;
}

/**
 * 任务创建/更新参数
 */
export interface TaskFormData {
  name: string; // 表单使用 name
  description?: string;
  script?: Action[]; // 脚本动作数组
  priority?: number;
  tags?: string;
  timeout?: number;
  retryOnFailure?: boolean;
  maxRetries?: number;
  status?: string; // 'pending' | 'disabled'
}

// ==================== Worker 相关 ====================

/**
 * Worker 状态
 */
export type WorkerStatus = "online" | "offline" | "busy" | "error";

/**
 * Worker 能力
 */
export interface WorkerCapabilities {
  browsers: string[]; // 支持的浏览器 ['chromium', 'firefox', 'webkit']
  maxConcurrency: number; // 最大并发任务数
  headless: boolean; // 是否支持无头模式
  screenshot: boolean; // 是否支持截图
  video: boolean; // 是否支持录制视频
  aiAgent: boolean; // 是否支持AI Agent
  proxy: boolean; // 是否支持代理
  geolocation: boolean; // 是否支持地理位置模拟
}

/**
 * RPA Worker 节点
 */
export interface Worker {
  id: string;
  workerId?: string; // Worker 标识符 (配置中的 worker.id)
  workerName?: string; // Worker 名称 (配置中的 worker.name)
  ipAddress?: string; // IP 地址
  port?: number; // 端口
  status: WorkerStatus;
  maxConcurrency?: number; // 最大并发数
  currentTasks: number; // 当前执行任务数
  lastHeartbeat?: number; // 最后心跳时间戳
  capabilities?: WorkerCapabilities; // 能力配置
  createdAt?: string; // 注册时间
  updatedAt?: string; // 更新时间
  // 兼容字段 (可能不存在)
  name?: string;
  hostname?: string;
  version?: string;
  dockerContainerId?: string;
  completedTasks?: number;
  failedTasks?: number;
  metadata?: Record<string, unknown>;
  registeredAt?: string;
}

/**
 * Worker 列表查询参数
 */
export interface WorkerListParams extends PageParams {
  status?: WorkerStatus;
  hostname?: string;
  name?: string;
}

/**
 * Worker 注册请求
 */
export interface WorkerRegisterRequest {
  name: string;
  hostname: string;
  capabilities: WorkerCapabilities;
  version?: string;
  ipAddress?: string;
  dockerContainerId?: string;
  metadata?: Record<string, unknown>;
}

/**
 * Worker 心跳请求
 */
export interface WorkerHeartbeatRequest {
  status: WorkerStatus;
  currentTasks: number;
}

// ==================== 执行记录相关 ====================

/**
 * 执行状态
 */
export type RPAExecutionStatus =
  "pending" | "running" | "completed" | "failed" | "cancelled" | "timeout";

/**
 * 执行日志级别
 */
export type LogLevel = "debug" | "info" | "warn" | "error";

/**
 * 执行日志
 */
export interface ExecutionLog {
  id: string;
  executionId: string;
  level: LogLevel;
  step: number; // 步骤序号
  message: string; // 日志消息
  detail?: string; // 详细信息
  screenshotUrl?: string; // 截图URL
  timestamp: string;
}

/**
 * 执行进度
 */
export interface ExecutionProgress {
  executionId: string;
  step: number;
  total: number;
  message: string;
  status: RPAExecutionStatus;
  screenshotUrl?: string;
  timestamp: number;
}

/**
 * RPA 执行记录
 */
export interface Execution {
  id: string;
  taskId: string;
  taskName?: string;
  workerId?: string;
  workerName?: string;
  status: RPAExecutionStatus;
  step: number; // 当前步骤
  totalSteps: number; // 总步骤数
  progress: number; // 进度百分比 (0-100)
  message?: string; // 当前消息
  error?: string; // 错误信息
  logs?: ExecutionLog[]; // 执行日志
  screenshots?: string[]; // 截图URL列表
  videoUrl?: string; // 视频录制URL
  resultData?: Record<string, unknown>; // 执行结果数据
  startedAt: string;
  completedAt?: string;
  duration?: number; // 执行时长（毫秒）
  createdAt: string;
}

/**
 * 执行记录查询参数
 */
export interface ExecutionListParams extends PageParams {
  taskId?: string;
  workerId?: string;
  status?: RPAExecutionStatus;
  dateRange?: [string, string];
}

// ==================== 定时调度相关 ====================

/**
 * 调度类型
 */
export type ScheduleType = "cron" | "interval" | "once";

/**
 * 调度状态
 */
export type ScheduleStatus = "active" | "paused" | "disabled";

/**
 * Cron 调度配置
 */
export interface CronSchedule {
  type: "cron";
  expression: string; // Cron表达式: "* * * * *"
  timezone?: string; // 时区
}

/**
 * 间隔调度配置
 */
export interface IntervalSchedule {
  type: "interval";
  interval: number; // 间隔（毫秒）
  unit?: "ms" | "s" | "m" | "h" | "d";
}

/**
 * 一次性调度配置
 */
export interface OnceSchedule {
  type: "once";
  executeAt: string; // ISO 8601时间格式
}

/**
 * 调度配置
 */
export type ScheduleConfig = CronSchedule | IntervalSchedule | OnceSchedule;

/**
 * RPA 定时调度
 */
export interface Schedule {
  id: string;
  name: string;
  description?: string;
  taskId: string;
  taskName?: string;
  config: ScheduleConfig;
  status: ScheduleStatus;
  nextRunTime?: string; // 下次执行时间
  lastRunTime?: string; // 上次执行时间
  runCount: number; // 累计执行次数
  failCount: number; // 失败次数
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 定时调度查询参数
 */
export interface ScheduleListParams extends PageParams {
  taskId?: string;
  status?: ScheduleStatus;
  name?: string;
}

// ==================== 变量相关 ====================

/**
 * 变量类型
 */
export type VariableType = "string" | "number" | "boolean" | "json" | "encrypted";

/**
 * 变量值
 */
export interface VariableValue {
  name: string;
  type: VariableType;
  value: string;
  description?: string;
}

/**
 * RPA 变量（全局或任务级别）
 */
export interface Variable {
  id: string;
  name: string;
  type: VariableType;
  value: string;
  description?: string;
  scope: "global" | "task"; // 作用域
  taskId?: string; // 关联任务ID（scope=task时）
  masked?: boolean; // 是否脱敏显示（如密码）
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 变量查询参数
 */
export interface VariableListParams extends PageParams {
  scope?: "global" | "task";
  taskId?: string;
  name?: string;
}

// ==================== 脚本模板相关 ====================

/**
 * 脚本模板分类
 */
export interface TemplateCategory {
  id: string;
  name: string;
  description?: string;
  icon?: string;
  orderNum: number;
}

/**
 * 脚本模板
 */
export interface Template {
  id: string;
  name: string;
  description?: string;
  categoryId?: string;
  category?: TemplateCategory;
  thumbnail?: string; // 预览图
  tags?: string[];
  script: Script;
  isPublic: boolean; // 是否公开
  usageCount: number; // 使用次数
  rating?: number; // 评分 (0-5)
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 模板查询参数
 */
export interface TemplateListParams extends PageParams {
  categoryId?: string;
  tags?: string[];
  isPublic?: boolean;
  name?: string;
}

// ==================== AI 相关 ====================

/**
 * AI 脚本生成请求
 */
export interface AIScriptGenerateRequest {
  description: string; // 自然语言描述
  url?: string; // 目标URL（可选）
  context?: string; // 额外上下文
}

/**
 * AI 脚本生成响应
 */
export interface AIScriptGenerateResponse {
  script: Script;
  confidence: number; // 置信度 (0-1)
  warnings?: string[]; // 警告信息
  suggestions?: string[]; // 优化建议
}

/**
 * AI 脚本优化请求
 */
export interface AIScriptOptimizeRequest {
  script: Script;
  optimizationType?: "performance" | "reliability" | "readability";
}

/**
 * AI 脚本优化响应
 */
export interface AIScriptOptimizeResponse {
  script: Script;
  improvements: string[]; // 改进点说明
  beforeAfter?: Array<{
    // 优化对比
    action: Action;
    optimized: Action;
    reason: string;
  }>;
}

/**
 * AI 脚本解释请求
 */
export interface AIScriptExplainRequest {
  script: Script;
  detailLevel?: "brief" | "normal" | "detailed";
}

/**
 * AI 脚本解释响应
 */
export interface AIScriptExplainResponse {
  summary: string; // 脚本摘要
  steps: Array<{
    // 步骤说明
    step: number;
    action: Action;
    explanation: string;
  }>;
  warnings?: string[]; // 潜在问题警告
  suggestions?: string[]; // 改进建议
}

/**
 * AI Agent 决策请求（选择器失效时调用）
 */
export interface AIAgentDecisionRequest {
  taskDescription: string;
  currentStep: number;
  failedAction: Action;
  screenshotBase64?: string; // 页面截图
  htmlSnippet?: string; // HTML片段
  availableSelectors: string[]; // 已尝试的选择器
  error?: string; // 错误信息
}

/**
 * AI Agent 决策响应
 */
export interface AIAgentDecisionResponse {
  type: ActionType; // 推荐的动作类型
  selector?: Selector; // 推荐的选择器
  coordinates?: [number, number]; // 坐标降级方案 [x, y]
  params?: Record<string, unknown>;
  reasoning: string; // 推理过程
  confidence: number; // 置信度 (0-1)
  suggestedFix?: string; // 建议的修复说明
}

/**
 * AI 失败分析请求
 */
export interface AIFailureAnalysisRequest {
  taskDescription: string;
  failedStep: number;
  error: string;
  screenshotBase64?: string;
  htmlSnippet?: string;
  executionHistory?: Action[]; // 执行历史
}

/**
 * AI 失败分析响应
 */
export interface AIFailureAnalysisResponse {
  rootCause: string; // 根本原因
  suggestedFix: string; // 修复建议
  alternativeAction?: Action; // 替代动作
  canRecover: boolean; // 是否可以恢复
}

/**
 * 页面状态捕获请求
 */
export interface CaptureStateRequest {
  url?: string;
  selector?: string; // 捕获特定元素
  includeHtml: boolean;
  includeScreenshot: boolean;
}

/**
 * 页面状态捕获响应
 */
export interface CaptureStateResponse {
  screenshotBase64?: string;
  htmlSnippet?: string;
  url: string;
  title: string;
  timestamp: string;
}

// ==================== 通知相关 ====================

/**
 * 通知类型
 */
export type NotificationType = "success" | "failure" | "timeout" | "warning";

/**
 * 通知渠道
 */
export type NotificationChannel = "email" | "webhook" | "websocket" | "sms";

/**
 * RPA 通知配置
 */
export interface NotificationConfig {
  id: string;
  name: string;
  events: NotificationType[]; // 触发事件
  channels: NotificationChannel[]; // 通知渠道
  recipients: string[]; // 接收人
  webhookUrl?: string; // Webhook URL
  template?: string; // 消息模板
  taskId?: string; // 关联任务（空=全局）
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * 通知查询参数
 */
export interface NotificationListParams extends PageParams {
  taskId?: string;
  enabled?: boolean;
}

// ==================== 统计相关 ====================

/**
 * 任务统计
 */
export interface TaskStatistics {
  total: number;
  byStatus: Record<TaskStatus, number>;
  todayExecuted: number;
  todayFailed: number;
  successRate: number;
}

/**
 * Worker 统计
 */
export interface WorkerStatistics {
  total: number;
  online: number;
  offline: number;
  busy: number;
  error: number;
  totalCapacity: number; // 总并发能力
  usedCapacity: number; // 已用并发
  availableCapacity: number; // 可用并发
}

/**
 * 执行统计
 */
export interface ExecutionStatistics {
  total: number;
  byStatus: Record<RPAExecutionStatus, number>;
  avgDuration: number; // 平均执行时长（毫秒）
  todayCount: number;
  weeklyTrend: Array<{
    // 周趋势
    date: string;
    count: number;
    successRate: number;
  }>;
}

/**
 * RPA 综合统计
 */
export interface RPAStatistics {
  tasks: TaskStatistics;
  workers: WorkerStatistics;
  executions: ExecutionStatistics;
}

// ==================== WebSocket 实时进度相关 ====================

/**
 * RPA 进度消息类型
 */
export type RPAProgressMessageType = "rpa_progress" | "rpa_completed" | "rpa_failed";

/**
 * RPA 进度消息（WebSocket 推送格式）
 */
export interface RPAProgressMessage {
  type: RPAProgressMessageType;
  executionId: string;
  taskId: string;
  taskName: string;
  step: number;
  total: number;
  message: string;
  status: string;
  timestamp: number;
}

/**
 * RPA 进度步骤
 */
export interface RPAProgressStep {
  stepNumber: number;
  name: string;
  status: "pending" | "running" | "success" | "failed";
  message: string;
  startTime: number;
  endTime?: number;
}

/**
 * RPA 进度详情
 */
export interface RPAProgressDetail {
  executionId: string;
  taskId: string;
  taskName: string;
  currentStep: number;
  totalSteps: number;
  progress: number; // 0-100
  status: string;
  steps: RPAProgressStep[];
  startTime: number;
  estimatedEndTime?: number;
  triggeredBy: string;
  workerName: string;
}

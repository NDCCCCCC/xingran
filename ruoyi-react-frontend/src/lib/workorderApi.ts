import { post } from "./api";
import type { BaseResponse, PageResponse } from "@/types";
// canonical 部门类型锚点来自 dutyApi (Phase 37-05 类型去重)。
// 本地 import 用于文件内引用 (line 84 WorkOrder.department / line 569 getDeptList 返回类型)。
// 文件末尾的 `export type { SimpleDept } from "./dutyApi"` 仅用于 re-export 暴露给消费方,
// 不会引入本地 binding —— 故需在顶部单独 import。
import type { SimpleDept } from "./dutyApi";

// 时间格式化函数
export { formatTime, formatDateTime, formatDate } from "@/utils/datetime";

// ==================== 类型定义 ====================

export enum WorkOrderStatus {
  Pending = 0,
  Processing = 1,
  Completed = 2,
  Closed = 3,
  Rejected = 4,
}

export enum WorkOrderPriority {
  Low = 0,
  Medium = 1,
  High = 2,
  Urgent = 3,
}

export enum WorkOrderType {
  Fault = "fault",
  Request = "request",
  Change = "change",
  Incident = "incident",
  Question = "question",
}

export enum PeriodicAssignType {
  Manual = "manual",
  DutyPool = "duty_pool",
  Rotation = "rotation",
}

// ==================== 工具类型 ====================

export interface CategoryTreeNode {
  title: string;
  value: string;
  key: string;
  children?: CategoryTreeNode[];
}

export function buildCategoryTree(categories: WorkOrderCategory[]): CategoryTreeNode[] {
  return categories.map((cat) => ({
    title: cat.categoryName,
    value: cat.id,
    key: cat.id,
    children: cat.children?.length ? buildCategoryTree(cat.children) : undefined,
  }));
}

// ==================== 工单相关类型 ====================

export interface WorkOrder {
  id: string;
  title: string;
  workOrderNo: string;
  categoryId: string;
  type: WorkOrderType;
  priority: WorkOrderPriority;
  status: WorkOrderStatus;
  description: string;
  solution?: string;
  submitterId: string;
  assigneeId?: string;
  deptId?: string;
  expectedResolveAt?: string;
  resolvedAt?: string;
  closedAt?: string;
  attachmentIds?: string;
  // 自动分配相关
  isAutoAssigned: boolean;
  dutyPoolId?: string;
  dutyType?: string;
  assignStrategy?: string;
  // 关联
  category?: WorkOrderCategory;
  submitter?: SimpleUser;
  assignee?: SimpleUser;
  department?: SimpleDept;
  comments?: WorkOrderComment[];
  history?: WorkOrderHistory[];
  ratings?: WorkOrderRating[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface WorkOrderCategory {
  id: string;
  categoryName: string;
  description?: string;
  status: number; // 0=启用 1=停用
  sortOrder: number;
  parentId?: string;
  parent?: WorkOrderCategory;
  children?: WorkOrderCategory[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface WorkOrderComment {
  id: string;
  workOrderId: string;
  userId: string;
  content: string;
  isInternal: boolean;
  createdAt: string;
  user?: SimpleUser;
}

export interface WorkOrderHistory {
  id: string;
  workOrderId: string;
  action: string;
  field?: string;
  oldValue?: string;
  newValue?: string;
  remark?: string;
  operatorId: string;
  createdAt: string;
  operator?: SimpleUser;
}

export interface WorkOrderRating {
  id: string;
  workOrderId: string;
  ratingType: string; // user 或 handler
  completionScore: number; // 用户评价：完成度评分 (1-5)
  cooperationScore: number; // 处理人员评价：配合度评分 (1-5)
  comment?: string;
  raterId: string;
  createdAt: string;
  rater?: SimpleUser;
}

export interface WorkOrderConfig {
  id: string;
  autoAssignEnabled: boolean;
  autoAssignTarget: string;
  autoAssignStrategy: string;
  autoCloseDays: number;
  allowUserClose: boolean;
  notificationEnabled: boolean;
  emailNotification: boolean;
  smsNotification: boolean;
  ratingEnabled: boolean;
  knowledgeConvertEnabled: boolean;
  createdAt: string;
  updatedAt?: string;
}

// ==================== 周期性工单相关类型 ====================

export interface PeriodicWorkOrderTemplate {
  id: string;
  templateName: string;
  workOrderTitle: string;
  description?: string;
  categoryId: string;
  type: WorkOrderType;
  priority: WorkOrderPriority;
  cronExpression: string;
  assignType: PeriodicAssignType;
  assignTargetId?: string;
  isEnabled: boolean;
  nextRunAt?: string;
  jobId?: string;
  totalGenerated: number;
  notifyAssignee: boolean;
  // 关联
  category?: WorkOrderCategory;
  assignee?: SimpleUser;
  job?: { id: string; jobName: string } | null;
  executionLogs?: PeriodicWorkOrderLog[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface PeriodicWorkOrderLog {
  id: string;
  templateId: string;
  workOrderId: string;
  executedAt: string;
  jobId?: string;
  status: string; // success, failed
  result?: string;
  errorMsg?: string;
  // 关联
  template?: PeriodicWorkOrderTemplate;
  workOrder?: WorkOrder;
  job?: { id: string; jobName: string } | null;
}

// ==================== 工单统计相关类型 ====================

export interface WorkOrderStatistics {
  total: number;
  pending: number;
  processing: number;
  completed: number;
  closed: number;
  rejected: number;
  byPriority: Record<string, number>;
  byCategory: Record<string, number>;
  byAssignee: AssigneeStatistics[];
  byDepartment: DepartmentStatistics[];
  trend: TrendData[];
  avgProcessTime: number;
}

export interface AssigneeStatistics {
  assigneeId: string;
  assigneeName: string;
  totalCount: number;
  pendingCount: number;
  doneCount: number;
  avgProcessTime: number;
}

export interface DepartmentStatistics {
  deptId: string;
  deptName: string;
  totalCount: number;
  doneCount: number;
}

export interface TrendData {
  date: string;
  count: number;
}

// ==================== 通用类型 ====================

export interface SimpleUser {
  id: string;
  username: string;
  nickName: string;
  deptId?: string;
  deptName?: string;
  status: number;
}

// SimpleDept 全项目唯一定义在 dutyApi.ts (canonical 锚点)。
// 此处 re-export 保持外部 import 路径不变 (T-37-09 mitigate: 消费方仍可 import { SimpleDept } from "@/lib/workorderApi")。
// 本文件内 WorkOrder.department 字段 (line 84) 亦通过此 re-export 引用。
export type { SimpleDept } from "./dutyApi";

// ==================== 请求参数类型 ====================

export interface WorkOrderListRequest {
  current?: number;
  pageSize?: number;
  workOrderNo?: string;
  title?: string;
  categoryId?: string;
  type?: string;
  priority?: number;
  status?: number;
  submitterId?: string;
  assigneeId?: string;
  deptId?: string;
  startDate?: string;
  endDate?: string;
}

export interface WorkOrderCreateRequest {
  title: string;
  categoryId: string;
  type: WorkOrderType;
  priority?: number;
  description?: string;
  deptId?: string;
  expectedResolveAt?: string;
  attachmentIds?: string;
  isAutoAssigned?: boolean;
  assigneeId?: string;
}

export interface WorkOrderUpdateRequest {
  title?: string;
  categoryId?: string;
  type?: WorkOrderType;
  priority?: number;
  status?: number;
  description?: string;
  solution?: string;
  assigneeId?: string;
  deptId?: string;
  expectedResolveAt?: string;
  attachmentIds?: string;
  resolvedAt?: string;
  closedAt?: string;
}

export interface AssignWorkOrderRequest {
  assigneeId: string;
  remark?: string;
}

export interface UpdateStatusRequest {
  status: number;
  solution?: string;
  remark?: string;
}

export interface AddCommentRequest {
  content: string;
  isInternal?: boolean;
}

export interface WorkOrderCategoryCreateRequest {
  categoryName: string;
  description?: string;
  parentId?: string;
  sortOrder?: number;
  status?: number;
}

export interface WorkOrderCategoryUpdateRequest {
  categoryName?: string;
  description?: string;
  parentId?: string;
  sortOrder?: number;
  status?: number;
}

export interface PeriodicTemplateListRequest {
  current?: number;
  pageSize?: number;
  title?: string;
  isEnabled?: boolean;
}

export interface CreatePeriodicTemplateRequest {
  templateName: string;
  workOrderTitle: string;
  description?: string;
  categoryId: string;
  type: WorkOrderType;
  priority?: number;
  cronExpression: string;
  assignType?: PeriodicAssignType;
  assignTargetId?: string;
  notifyAssignee?: boolean;
}

export interface UpdatePeriodicTemplateRequest {
  templateName?: string;
  workOrderTitle?: string;
  description?: string;
  categoryId?: string;
  type?: WorkOrderType;
  priority?: number;
  cronExpression?: string;
  assignType?: PeriodicAssignType;
  assignTargetId?: string;
  isEnabled?: boolean;
  notifyAssignee?: boolean;
}

export interface CreateWorkOrderRatingRequest {
  ratingType: string; // user 或 handler
  completionScore?: number; // 用户评价：完成度评分
  cooperationScore?: number; // 处理人员评价：配合度评分
  comment?: string;
}

// ==================== 工单管理 API ====================

export function getWorkOrderList(params: WorkOrderListRequest): Promise<BaseResponse<PageResponse<WorkOrder>>> {
  return post("/workorder/orders/list", params);
}

/** 工单状态统计（总数 / 待处理 / 处理中 / 已完成 / 已关闭），供列表页统计卡片 */
export interface WorkOrderStatusStatistics {
  total: number;
  pending: number;
  processing: number;
  completed: number;
  closed: number;
}

/**
 * 获取工单状态统计（后端 COUNT 聚合，不受分页/筛选影响）
 */
export function getWorkOrderStatusStatistics(): Promise<BaseResponse<WorkOrderStatusStatistics>> {
  return post("/workorder/orders/status-statistics", {});
}

export function getMyPendingWorkOrders(params?: { limit?: number }): Promise<BaseResponse<{ list: WorkOrder[]; total: number }>> {
  return post("/workorder/orders/my-pending", params ?? {});
}

export function getWorkOrder(id: string): Promise<BaseResponse<WorkOrder>> {
  return post(`/workorder/orders/${id}`, {});
}

export function createWorkOrder(data: WorkOrderCreateRequest): Promise<BaseResponse<WorkOrder>> {
  return post("/workorder/orders", data);
}

export function updateWorkOrder(id: string, data: WorkOrderUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/update`, data);
}

export function deleteWorkOrder(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/delete`);
}

export function batchDeleteWorkOrders(ids: string[]): Promise<BaseResponse<{ message: string; count: number }>> {
  return post("/workorder/orders/batch-delete", { ids });
}

export function assignWorkOrder(id: string, data: AssignWorkOrderRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/assign`, data);
}

export function assignToTodayDuty(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/assign-duty`);
}

export function updateWorkOrderStatus(id: string, data: UpdateStatusRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/status`, data);
}

// ==================== 工单评论 API ====================

export function getWorkOrderComments(id: string): Promise<BaseResponse<WorkOrderComment[]>> {
  return post(`/workorder/orders/${id}/comments/list`, {});
}

export function addWorkOrderComment(id: string, data: AddCommentRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/comments`, data);
}

export function getWorkOrderHistory(id: string): Promise<BaseResponse<WorkOrderHistory[]>> {
  return post(`/workorder/orders/${id}/history`, {});
}

// ==================== 工单评价 API ====================

export function createWorkOrderRating(id: string, data: CreateWorkOrderRatingRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/orders/${id}/rating`, data);
}

export function getWorkOrderRatings(id: string): Promise<BaseResponse<WorkOrderRating[]>> {
  return post(`/workorder/orders/${id}/ratings`, {});
}

export function getRatingStatistics(id: string): Promise<BaseResponse<{ averageRating: number; ratingCounts: Record<number, number> }>> {
  return post(`/workorder/orders/${id}/rating-statistics`, {});
}

// ==================== 工单分类 API ====================

export function getWorkOrderCategoryList(): Promise<BaseResponse<WorkOrderCategory[]>> {
  return post("/workorder/categories/list", {});
}

export function getEnabledWorkOrderCategories(): Promise<BaseResponse<WorkOrderCategory[]>> {
  return post("/workorder/categories/enabled", {});
}

export function getWorkOrderCategory(id: string): Promise<BaseResponse<WorkOrderCategory>> {
  return post(`/workorder/categories/${id}`, {});
}

export function createWorkOrderCategory(data: WorkOrderCategoryCreateRequest): Promise<BaseResponse<WorkOrderCategory>> {
  return post("/workorder/categories", data);
}

export function updateWorkOrderCategory(id: string, data: WorkOrderCategoryUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/categories/${id}/update`, data);
}

export function deleteWorkOrderCategory(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/categories/${id}/delete`);
}

// ==================== 工单统计 API ====================

export function getWorkOrderStatistics(): Promise<BaseResponse<WorkOrderStatistics>> {
  return post("/workorder/statistics", {});
}

// ==================== 周期性工单 API ====================

export function getPeriodicTemplateList(params: PeriodicTemplateListRequest): Promise<BaseResponse<PageResponse<PeriodicWorkOrderTemplate>>> {
  return post("/workorder/periodic/templates/list", params);
}

/** 周期性工单模板统计（总数 / 启用 / 停用 / 累计生成数） */
export interface PeriodicTemplateStatistics {
  total: number;
  enabled: number;
  disabled: number;
  totalGenerated: number;
}

/**
 * 获取周期性工单模板统计（后端 COUNT 聚合，不受分页影响）
 */
export function getPeriodicTemplateStatistics(): Promise<BaseResponse<PeriodicTemplateStatistics>> {
  return post("/workorder/periodic/templates/statistics", {});
}

export function getPeriodicTemplate(id: string): Promise<BaseResponse<PeriodicWorkOrderTemplate>> {
  return post(`/workorder/periodic/templates/${id}`, {});
}

export function createPeriodicTemplate(data: CreatePeriodicTemplateRequest): Promise<BaseResponse<PeriodicWorkOrderTemplate>> {
  return post("/workorder/periodic/templates", data);
}

export function updatePeriodicTemplate(id: string, data: UpdatePeriodicTemplateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/periodic/templates/${id}/update`, data);
}

export function deletePeriodicTemplate(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/periodic/templates/${id}/delete`);
}

export function enablePeriodicTemplate(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/periodic/templates/${id}/enable`);
}

export function disablePeriodicTemplate(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/workorder/periodic/templates/${id}/disable`);
}

export function generateWorkOrderNow(id: string): Promise<BaseResponse<WorkOrder>> {
  return post(`/workorder/periodic/templates/${id}/generate`);
}

export function getPeriodicLogs(id: string): Promise<BaseResponse<PeriodicWorkOrderLog[]>> {
  return post(`/workorder/periodic/templates/${id}/logs`, {});
}

// ==================== 工单配置 API ====================

export function getWorkOrderConfig(): Promise<BaseResponse<WorkOrderConfig>> {
  return post("/workorder/config", {});
}

export function updateWorkOrderConfig(data: Partial<WorkOrderConfig>): Promise<BaseResponse<{ message: string }>> {
  return post("/workorder/config/update", data);
}

// ==================== 用户和部门（复用dutyApi中的）====================

export function getUserList(params?: { current?: number; pageSize?: number; deptId?: string; status?: number }): Promise<BaseResponse<PageResponse<SimpleUser>>> {
  return post("/system/users/list", {
    current: 1,
    pageSize: 1000,
    ...params,
  });
}

export function getDeptList(): Promise<BaseResponse<SimpleDept[]>> {
  return post("/system/departments/list");
}

// getDeptTree 副本已删除 (Phase 37-05): 全项目唯一 fetch 部门树的函数在 dutyApi.ts。
// 消费方应改用 useDeptTree hook (共享 ['dept','tree'] React Query 缓存),
// 或直接 import { getDeptTree } from "@/lib/dutyApi"。

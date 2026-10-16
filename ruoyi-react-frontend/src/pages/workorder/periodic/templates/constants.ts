/**
 * Periodic Template Constants
 * 周期性工单模板常量定义
 */

import { PeriodicAssignType, WorkOrderPriority, WorkOrderType } from "@/lib/workorderApi";

/** 优先级配置 */
export const PRIORITY_CONFIG = {
  [WorkOrderPriority.Low]: { text: "低", color: "default" },
  [WorkOrderPriority.Medium]: { text: "中", color: "blue" },
  [WorkOrderPriority.High]: { text: "高", color: "orange" },
  [WorkOrderPriority.Urgent]: { text: "紧急", color: "red" },
} as const;

/** 工单类型配置 */
export const TYPE_CONFIG = {
  [WorkOrderType.Fault]: { text: "故障" },
  [WorkOrderType.Request]: { text: "请求" },
  [WorkOrderType.Change]: { text: "变更" },
  [WorkOrderType.Incident]: { text: "事件" },
  [WorkOrderType.Question]: { text: "咨询" },
} as const;

/** 分配类型配置 */
export const ASSIGN_TYPE_CONFIG = {
  [PeriodicAssignType.Manual]: { text: "手动指定" },
  [PeriodicAssignType.DutyPool]: { text: "当天值班人员" },
  [PeriodicAssignType.Rotation]: { text: "轮询" },
} as const;

/** 默认表单值 */
export const DEFAULT_FORM_VALUES = {
  type: WorkOrderType.Fault,
  priority: WorkOrderPriority.Medium,
  assignType: PeriodicAssignType.DutyPool,
  notifyAssignee: true,
};

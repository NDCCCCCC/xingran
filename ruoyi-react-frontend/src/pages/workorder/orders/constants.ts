/**
 * WorkOrder 常量定义
 */

import { WorkOrderStatus, WorkOrderPriority, WorkOrderType } from "@/lib/workorderApi";

// 工单状态配置
export const STATUS_CONFIG = {
  [WorkOrderStatus.Pending]: { text: "待处理", color: "default" },
  [WorkOrderStatus.Processing]: { text: "处理中", color: "processing" },
  [WorkOrderStatus.Completed]: { text: "已完成", color: "success" },
  [WorkOrderStatus.Closed]: { text: "已关闭", color: "default" },
  [WorkOrderStatus.Rejected]: { text: "已拒绝", color: "error" },
};

// 优先级配置
export const PRIORITY_CONFIG = {
  [WorkOrderPriority.Low]: { text: "低", color: "default" },
  [WorkOrderPriority.Medium]: { text: "中", color: "blue" },
  [WorkOrderPriority.High]: { text: "高", color: "orange" },
  [WorkOrderPriority.Urgent]: { text: "紧急", color: "red" },
};

// 工单类型配置
export const TYPE_CONFIG = {
  [WorkOrderType.Fault]: { text: "故障" },
  [WorkOrderType.Request]: { text: "请求" },
  [WorkOrderType.Change]: { text: "变更" },
  [WorkOrderType.Incident]: { text: "事件" },
  [WorkOrderType.Question]: { text: "咨询" },
};

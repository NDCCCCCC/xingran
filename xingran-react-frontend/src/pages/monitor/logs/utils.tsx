/**
 * Log Utilities
 * 日志管理工具函数
 */

import { Tag } from "antd";
import { LogStatus } from "./types";
import { BUSINESS_TYPE_OPTIONS } from "./constants";

/** 格式化时间为本地字符串 */
export function formatLocalTime(time: string | null | undefined): string {
  if (!time) return "-";
  return new Date(time).toLocaleString();
}

/** 获取业务类型标签 */
export function getBusinessTypeLabel(type: number): string {
  const option = BUSINESS_TYPE_OPTIONS.find(opt => opt.value === type);
  return option?.label ?? "-";
}

/** 渲染请求方式标签 */
export function renderRequestMethodTag(method: string) {
  const colorMap: Record<string, string> = {
    GET: "blue",
    POST: "green",
    PUT: "orange",
    DELETE: "red",
  };
  const color = colorMap[method] || "default";
  return <Tag color={color}>{method}</Tag>;
}

/** 渲染日志状态标签 */
export function renderLogStatusTag(status: number, type: "oper" | "login" = "oper") {
  const color = status === LogStatus.Success ? "success" : "error";
  const text = type === "oper"
    ? (status === LogStatus.Success ? "正常" : "异常")
    : (status === LogStatus.Success ? "成功" : "失败");
  return <Tag color={color}>{text}</Tag>;
}

/** 处理时间范围参数 */
export function processTimeRangeParams(timeRange: any, params: Record<string, any>) {
  if (timeRange && timeRange.length === 2) {
    params.startTime = timeRange[0].toISOString();
    params.endTime = timeRange[1].toISOString();
  }
}

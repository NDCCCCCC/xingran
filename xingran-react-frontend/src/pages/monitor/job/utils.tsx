/**
 * Job Utilities
 * 定时任务管理工具函数
 */

import { Tag, Tooltip } from "antd";
import { JobStatus } from "./types";
import { formatDateTime } from "@/utils/datetime";

/** 格式化时间为本地字符串 */
export function formatLocalTime(time: string | null | undefined): string {
  return formatDateTime(time);
}

/** 格式化执行时长 */
export function formatDuration(duration: number): string {
  if (!duration) return "-";
  if (duration >= 1000) {
    const seconds = (duration / 1000).toFixed(2);
    return `${seconds}s`;
  }
  return `${duration}ms`;
}

/** 渲染任务状态标签 */
export function renderJobStatusTag(status: number) {
  const color = status === JobStatus.Normal ? "success" : "warning";
  const text = status === JobStatus.Normal ? "正常" : "暂停";
  return <Tag color={color}>{text}</Tag>;
}

/** 渲染并发执行标签 */
export function renderConcurrentTag(concurrent: boolean) {
  const color = concurrent ? "green" : "orange";
  const text = concurrent ? "允许" : "禁止";
  return <Tag color={color}>{text}</Tag>;
}

/** 渲染 Cron 表达式 */
export function renderCronExpression(text: string) {
  return (
    <Tooltip title={text}>
      <code className="bg-gray-100 px-2 py-1 rounded text-xs">{text}</code>
    </Tooltip>
  );
}

/** 渲染异常信息 */
export function renderExceptionInfo(info: string) {
  return info ? (
    <Tooltip title={info}>
      <span className="text-red-500">查看错误</span>
    </Tooltip>
  ) : (
    "-"
  );
}

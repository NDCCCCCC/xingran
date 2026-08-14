/**
 * RPA Worker 表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import type { Worker } from "@/types/rpa";
import { renderWorkerStatusTag } from "../constants";
import dayjs from "dayjs";
import React from "react";

// 格式化时间，处理后端返回的时区问题
function formatDateTime(date: string | Date | null | undefined): string {
  if (!date) return "-";
  // 移除时区后缀（Z 或 +08:00），将时间作为本地时间处理
  const dateStr =
    typeof date === "string" ? date.replace(/[Zz]$/, "").replace(/\+[0-9]{2}:[0-9]{2}$/, "") : date;
  try {
    return dayjs(dateStr).format("YYYY-MM-DD HH:mm:ss");
  } catch {
    return String(date);
  }
}

// 心跳超时时间（秒）
const HEARTBEAT_TIMEOUT = 120; // 2分钟无心跳视为离线

// 检查 Worker 是否真正在线（基于最后心跳时间）
function isWorkerActuallyOnline(worker: Worker): boolean {
  if (!worker.lastHeartbeat) return false;
  const lastHeartbeatTime = dayjs.unix(worker.lastHeartbeat);
  const now = dayjs();
  const diffSeconds = now.diff(lastHeartbeatTime, "second");
  return diffSeconds <= HEARTBEAT_TIMEOUT;
}

export function getWorkerColumns(options?: {
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}): ColumnsType<Worker> {
  const getSortOrder = options?.getSortOrder;
  return [
    {
      title: "Worker ID",
      dataIndex: "workerId",
      key: "workerId",
      width: 200,
      ellipsis: true,
      sorter: true,
      sortOrder: getSortOrder?.("workerId"),
      render: (workerId, record) => workerId || record.id || "-",
    },
    {
      title: "Worker 名称",
      dataIndex: "workerName",
      key: "workerName",
      width: 200,
      ellipsis: true,
      sorter: true,
      sortOrder: getSortOrder?.("workerName"),
      render: (name) => name || "-",
    },
    {
      title: "主机名",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 150,
      ellipsis: true,
      sorter: true,
      sortOrder: getSortOrder?.("ipAddress"),
      render: (ip, record) => ip || record.workerId || "-",
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status, record) => {
        // 基于最后心跳时间判断真实状态
        const actualStatus = isWorkerActuallyOnline(record) ? "online" : "offline";
        const displayStatus = status === "busy" ? "busy" : actualStatus;
        return renderWorkerStatusTag(displayStatus || "offline");
      },
    },
    {
      title: "当前任务数",
      dataIndex: "currentTasks",
      key: "currentTasks",
      width: 120,
      render: (count, record) => {
        const max = record.maxConcurrency || record.capabilities?.maxConcurrency || 3;
        return `${count ?? 0}/${max}`;
      },
    },
    {
      title: "最大并发",
      dataIndex: "maxConcurrency",
      key: "maxConcurrency",
      width: 100,
      render: (max, record) => {
        return String(max ?? record.capabilities?.maxConcurrency ?? 3);
      },
    },
    {
      title: "IP 地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 150,
      render: (ip) => ip || "-",
    },
    {
      title: "端口",
      dataIndex: "port",
      key: "port",
      width: 80,
      render: (port) => (port ? String(port) : "-"),
    },
    {
      title: "最后心跳",
      dataIndex: "lastHeartbeat",
      key: "lastHeartbeat",
      width: 180,
      render: (timestamp) => {
        if (!timestamp) return "-";
        // timestamp 是 Unix 时间戳（秒）
        const date = dayjs.unix(timestamp);
        const now = dayjs();
        const diffMinutes = now.diff(date, "minute");

        let timeStr = date.format("YYYY-MM-DD HH:mm:ss");
        if (diffMinutes > 2) {
          timeStr += ` (${diffMinutes}分钟前)`;
        }
        return timeStr;
      },
    },
    {
      title: "注册时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("createdAt"),
      render: (date) => formatDateTime(date),
    },
  ];
}

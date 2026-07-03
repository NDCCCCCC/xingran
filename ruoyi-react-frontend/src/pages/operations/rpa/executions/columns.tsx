/**
 * RPA 执行记录表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { EyeOutlined } from "@ant-design/icons";
import type { Execution } from "@/types/rpa";
import ActionButtons from "@/components/shared/ActionButtons";
import { renderExecutionStatusTag } from "../constants";
import { createDateTimeColumn } from "@/utils/tableHelpers";

export interface ExecutionColumnsParams {
  handleViewDetail: (record: Execution) => void;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

export function getExecutionColumns(params: ExecutionColumnsParams): ColumnsType<Execution> {
  const { handleViewDetail, getSortOrder } = params;

  return [
    {
      title: "任务名称",
      dataIndex: "taskName",
      key: "taskName",
      width: 200,
      ellipsis: true,
      sorter: true,
      sortOrder: getSortOrder?.("taskName"),
    },
    {
      title: "执行状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status) => renderExecutionStatusTag(status || "pending"),
    },
    {
      title: "Worker",
      dataIndex: "workerName",
      key: "workerName",
      width: 150,
      ellipsis: true,
      sorter: true,
      sortOrder: getSortOrder?.("workerName"),
      render: (text) => text || "-",
    },
    {
      title: "当前步骤",
      dataIndex: "step",
      key: "step",
      width: 120,
      render: (current, record) => {
        const total = record.totalSteps || 0;
        return total > 0 ? `${current ?? 0}/${total}` : "-";
      },
    },
    {
      title: "进度",
      dataIndex: "progress",
      key: "progress",
      width: 150,
      render: (progress) => {
        if (progress === undefined || progress === null) return "-";
        return `${Math.round(progress)}%`;
      },
    },
    {
      title: "开始时间",
      dataIndex: "startedAt",
      key: "startedAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("startTime"),
      render: (text) => text || "-",
    },
    {
      title: "结束时间",
      dataIndex: "completedAt",
      key: "completedAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("endTime"),
      render: (text) => text || "-",
    },
    {
      title: "耗时(秒)",
      dataIndex: "duration",
      key: "duration",
      width: 120,
      render: (duration) => duration ? `${Math.round(duration / 1000)}s` : "-",
    },
    {
      title: "错误信息",
      dataIndex: "error",
      key: "error",
      width: 200,
      ellipsis: true,
      render: (text) => text ? (
        <span title={text} style={{ color: "var(--theme-error, #ff4d4f)" }}>
          {text}
        </span>
      ) : "-",
    },
    createDateTimeColumn("createdAt", { width: 180, sorter: true, sortOrder: getSortOrder?.("createdAt") }),
    {
      title: "操作",
      key: "action",
      width: 120,
      fixed: "right",
      render: (_: unknown, record: Execution) => {
        const actions = [
          {
            key: "detail",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

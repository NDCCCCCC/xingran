/**
 * RPA 任务表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { PlayCircleOutlined } from "@ant-design/icons";
import type { Task } from "@/types/rpa";
import ActionButtons from "@/components/shared/ActionButtons";
import { renderTaskStatusTag } from "../constants";
import { createDateTimeColumn } from "@/utils/tableHelpers";

export interface TaskColumnsParams {
  handleEdit: (record: Task) => void;
  handleDelete: (id: string) => void;
  handleExecute: (id: string, name: string) => void;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

export function getTaskColumns(params: TaskColumnsParams): ColumnsType<Task> {
  const { handleEdit, handleDelete, handleExecute, getSortOrder } = params;

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
      title: "描述",
      dataIndex: "description",
      key: "description",
      width: 250,
      ellipsis: true,
      render: (text) => text || "-",
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("priority"),
      render: (priority) => {
        const color = priority >= 80 ? "red" : priority >= 50 ? "orange" : "default";
        return <span style={{ color }}>{priority ?? 0}</span>;
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status) => renderTaskStatusTag(status ?? 0),
    },
    {
      title: "超时(秒)",
      dataIndex: "timeout",
      key: "timeout",
      width: 120,
      render: (timeout) => timeout || "-",
    },
    {
      title: "最后执行",
      dataIndex: "lastExecutionTime",
      key: "lastExecutionTime",
      width: 180,
      render: (text) => text || "-",
    },
    createDateTimeColumn("createdAt", { width: 180, sorter: true, sortOrder: getSortOrder?.("createdAt") }),
    {
      title: "操作",
      key: "action",
      width: 200,
      fixed: "right",
      render: (_: unknown, record: Task) => {
        const actions = [
          {
            key: "execute",
            label: "执行",
            icon: <PlayCircleOutlined />,
            onClick: () => handleExecute(record.id, record.taskName || record.name || ""),
          },
          {
            key: "edit",
            label: "编辑",
            onClick: () => handleEdit(record),
          },
          {
            key: "delete",
            label: "删除",
            danger: true,
            onClick: () => handleDelete(record.id),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

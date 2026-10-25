/**
 * 任务日志表格列定义
 */

import { Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { JobLog } from "../types";
import { formatDateTime } from "@/utils/datetime";
import { formatDuration, renderExceptionInfo } from "../utils";

interface GetJobLogColumnsParams {
  /** 服务端排序受控箭头（字段名对齐后端 jobLogAllowedSortFields；startTime 列映射 createdAt） */
  getSortOrder: (field: string) => "ascend" | "descend" | null;
}

export function getJobLogColumns(params: GetJobLogColumnsParams): ColumnsType<JobLog> {
  const { getSortOrder } = params;
  return [
    {
      title: "执行时间",
      dataIndex: "startTime",
      key: "startTime",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder("startTime"),
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "执行时长",
      dataIndex: "duration",
      key: "duration",
      width: 100,
      render: formatDuration,
    },
    {
      title: "执行消息",
      dataIndex: "jobMessage",
      key: "jobMessage",
      width: 200,
      ellipsis: true,
    },
    {
      title: "执行状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder("status"),
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "成功" : "失败"}</Tag>
      ),
    },
    {
      title: "异常信息",
      dataIndex: "exceptionInfo",
      key: "exceptionInfo",
      ellipsis: true,
      render: renderExceptionInfo,
    },
  ];
}

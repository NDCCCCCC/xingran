/**
 * 任务日志表格列定义
 */

import { Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { JobLog } from "../types";
import { formatDateTime } from "@/utils/datetime";
import { formatDuration, renderJobStatusTag, renderExceptionInfo } from "../utils";
import { createSorter } from "@/utils/tableHelpers";

export function getJobLogColumns(): ColumnsType<JobLog> {
  return [
    {
      title: "执行时间",
      dataIndex: "startTime",
      key: "startTime",
      width: 180,
      sorter: createSorter<JobLog>("startTime", "date"),
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "执行时长",
      dataIndex: "duration",
      key: "duration",
      width: 100,
      sorter: createSorter<JobLog>("duration", "number"),
      render: formatDuration,
    },
    {
      title: "执行消息",
      dataIndex: "jobMessage",
      key: "jobMessage",
      width: 200,
      ellipsis: true,
      sorter: createSorter<JobLog>("jobMessage", "string"),
    },
    {
      title: "执行状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: createSorter<JobLog>("status", "number"),
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "成功" : "失败"}</Tag>
      ),
    },
    {
      title: "异常信息",
      dataIndex: "exceptionInfo",
      key: "exceptionInfo",
      ellipsis: true,
      sorter: createSorter<JobLog>("exceptionInfo", "string"),
      render: renderExceptionInfo,
    },
  ];
}

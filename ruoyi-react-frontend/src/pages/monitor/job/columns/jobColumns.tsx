/**
 * 定时任务表格列定义
 */

import { Space, Button, Tooltip } from "antd";
import { confirmDelete } from "@/components/shared/confirmDelete";
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  EditOutlined,
  DeleteOutlined,
  HistoryOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import ActionButtons from "@/components/shared/ActionButtons";
import type { JobInfo } from "../types";
import { formatDateTime } from "@/utils/datetime";
import { renderCronExpression, renderConcurrentTag, renderJobStatusTag } from "../utils";

interface GetJobColumnsParams {
  handleToggleStatus: (record: JobInfo) => void;
  handleExecute: (record: JobInfo) => void;
  handleViewLogs: (record: JobInfo) => void;
  openModal: (record: JobInfo) => void;
  handleDelete: (record: JobInfo) => void;
  /** 服务端排序受控箭头（字段名对齐后端 jobAllowedSortFields） */
  getSortOrder: (field: string) => "ascend" | "descend" | null;
}

export function getJobColumns(params: GetJobColumnsParams): ColumnsType<JobInfo> {
  const { handleToggleStatus, handleExecute, handleViewLogs, openModal, handleDelete, getSortOrder } = params;

  return [
    {
      title: "任务名称",
      dataIndex: "jobName",
      key: "jobName",
      width: 150,
      sorter: true,
      sortOrder: getSortOrder("jobName"),
    },
    {
      title: "任务组",
      dataIndex: "jobGroup",
      key: "jobGroup",
      width: 120,
      sorter: true,
      sortOrder: getSortOrder("jobGroup"),
    },
    {
      title: "调用目标",
      dataIndex: "invokeTarget",
      key: "invokeTarget",
      width: 200,
      ellipsis: true,
    },
    {
      title: "Cron表达式",
      dataIndex: "cronExpression",
      key: "cronExpression",
      width: 150,
      render: renderCronExpression,
    },
    {
      title: "并发执行",
      dataIndex: "concurrent",
      key: "concurrent",
      width: 100,
      render: renderConcurrentTag,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder("status"),
      render: (status: number, record: JobInfo) => (
        <Space>
          {renderJobStatusTag(status)}
          <Tooltip title={status === 0 ? "暂停任务" : "启动任务"}>
            <Button
              type="text"
              size="small"
              icon={status === 0 ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => handleToggleStatus(record)}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: "下次执行时间",
      dataIndex: "nextRunTime",
      key: "nextRunTime",
      width: 180,
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder("createdAt"),
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: JobInfo) => {
        const actions = [
          {
            key: "execute",
            label: "立即执行",
            icon: <PlayCircleOutlined />,
            onClick: () => handleExecute(record),
          },
          {
            key: "logs",
            label: "查看日志",
            icon: <HistoryOutlined />,
            onClick: () => handleViewLogs(record),
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openModal(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              confirmDelete({
                target: `定时任务 ${record.jobName}`,
                onDelete: async () => { await handleDelete(record); },
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

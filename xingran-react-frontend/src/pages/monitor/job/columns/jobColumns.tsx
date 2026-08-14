/**
 * 定时任务表格列定义
 */

import { Space, Button, Tooltip, Tag, Modal } from "antd";
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  HistoryOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import ActionButtons from "@/components/shared/ActionButtons";
import type { JobInfo } from "../types";
import { formatDateTime } from "@/utils/datetime";
import { renderCronExpression, renderConcurrentTag, renderJobStatusTag } from "../utils";
import { createSorter } from "@/utils/tableHelpers";

interface GetJobColumnsParams {
  handleToggleStatus: (record: JobInfo) => void;
  handleExecute: (record: JobInfo) => void;
  handleViewLogs: (record: JobInfo) => void;
  openModal: (record: JobInfo) => void;
  handleDelete: (record: JobInfo) => void;
}

export function getJobColumns(params: GetJobColumnsParams): ColumnsType<JobInfo> {
  const { handleToggleStatus, handleExecute, handleViewLogs, openModal, handleDelete } = params;

  return [
    {
      title: "任务名称",
      dataIndex: "jobName",
      key: "jobName",
      width: 150,
      sorter: createSorter<JobInfo>("jobName", "string"),
    },
    {
      title: "任务组",
      dataIndex: "jobGroup",
      key: "jobGroup",
      width: 120,
      sorter: createSorter<JobInfo>("jobGroup", "string"),
    },
    {
      title: "调用目标",
      dataIndex: "invokeTarget",
      key: "invokeTarget",
      width: 200,
      ellipsis: true,
      sorter: createSorter<JobInfo>("invokeTarget", "string"),
    },
    {
      title: "Cron表达式",
      dataIndex: "cronExpression",
      key: "cronExpression",
      width: 150,
      sorter: createSorter<JobInfo>("cronExpression", "string"),
      render: renderCronExpression,
    },
    {
      title: "并发执行",
      dataIndex: "concurrent",
      key: "concurrent",
      width: 100,
      sorter: createSorter<JobInfo>("concurrent", "number"),
      render: renderConcurrentTag,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: createSorter<JobInfo>("status", "number"),
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
      sorter: createSorter<JobInfo>("nextRunTime", "date"),
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: createSorter<JobInfo>("createdAt", "date"),
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
              Modal.confirm({
                title: "确定要删除这个任务吗？",
                content: "删除后将无法恢复",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

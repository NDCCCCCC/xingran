/**
 * Config Execution Table Columns
 * 配置执行表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Button, Tag, Progress, Tooltip, Modal } from "antd";
import { EyeOutlined, StopOutlined } from "@ant-design/icons";
import type { NetworkDevice, ConfigExecution, ConfigExecutionDetail } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { STATUS_CONFIG } from "./constants";
import ActionButtons from "@/components/shared/ActionButtons";

export interface ExecutionColumnsParams {
  handleViewDetail: (record: ConfigExecution) => void;
  handleCancelExecution: (id: string) => Promise<void>;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

export interface DetailColumnsParams {
  handleViewOutput: (output: string) => void;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

/** 设备表格列 */
export const deviceColumns: ColumnsType<NetworkDevice> = [
  {
    title: "设备名称",
    dataIndex: "deviceName",
    key: "deviceName",
  },
  { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress" },
  { title: "厂商", dataIndex: "vendor", key: "vendor" },
  { title: "型号", dataIndex: "model", key: "model" },
];

/** 执行记录表格列 */
export function getExecutionColumns(params: ExecutionColumnsParams): ColumnsType<ConfigExecution> {
  const { handleViewDetail, handleCancelExecution, getSortOrder } = params;

  return [
    {
      title: "任务名称",
      dataIndex: "executionName",
      key: "executionName",
      width: 200,
      sorter: true,
      sortOrder: getSortOrder?.("executionName"),
    },
    {
      title: "模板名称",
      dataIndex: "templateName",
      key: "templateName",
      width: 150,
      sorter: true,
      sortOrder: getSortOrder?.("templateName"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status: string) => {
        const config = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG] || STATUS_CONFIG.pending;
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        );
      },
    },
    { title: "设备总数", dataIndex: "totalDevices", key: "totalDevices", width: 100 },
    { title: "成功数", dataIndex: "successCount", key: "successCount", width: 80 },
    { title: "失败数", dataIndex: "failureCount", key: "failureCount", width: 80 },
    {
      title: "进度",
      key: "progress",
      width: 150,
      render: (_, record) => {
        const percent =
          record.totalDevices > 0
            ? Math.round(((record.successCount + record.failureCount) / record.totalDevices) * 100)
            : 0;
        return (
          <Progress
            percent={percent}
            size="small"
            status={record.status === "failed" ? "exception" : undefined}
          />
        );
      },
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("createdAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 180,
      render: (_, record) => {
        const actions = [
          {
            key: "view",
            label: "查看",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record),
          },
          ...(record.status === "pending" || record.status === "running"
            ? [
                {
                  key: "cancel",
                  label: "取消",
                  icon: <StopOutlined />,
                  danger: true,
                  onClick: () => {
                    Modal.confirm({
                      title: "确认取消执行?",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleCancelExecution(record.id),
                    });
                  },
                },
              ]
            : []),
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

/** 执行明细表格列 */
export function getDetailColumns(params: DetailColumnsParams): ColumnsType<ConfigExecutionDetail> {
  const { handleViewOutput, getSortOrder } = params;

  return [
    { title: "设备名称", dataIndex: "deviceName", key: "deviceName", width: 150 },
    { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress", width: 140 },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status: string) => {
        const config = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG] || STATUS_CONFIG.pending;
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: "输出",
      key: "output",
      width: 100,
      render: (_, record) =>
        record.outputReceived ? (
          <Button
            type="link"
            size="small"
            onClick={() => handleViewOutput(record.outputReceived ?? "")}
          >
            查看
          </Button>
        ) : (
          "-"
        ),
    },
    { title: "开始时间", dataIndex: "startedAt", key: "startedAt", width: 180 },
    { title: "完成时间", dataIndex: "completedAt", key: "completedAt", width: 180 },
    { title: "耗时(秒)", dataIndex: "duration", key: "duration", width: 100 },
    {
      title: "错误信息",
      dataIndex: "errorMessage",
      key: "errorMessage",
      ellipsis: true,
      render: (msg: string) =>
        msg ? (
          <Tooltip title={msg}>
            <span style={{ color: "var(--theme-error, #ba3630)" }}>{msg}</span>
          </Tooltip>
        ) : (
          "-"
        ),
    },
  ];
}

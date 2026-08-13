/**
 * Network Command Columns
 * 网络命令表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Progress, Button, Modal, Tooltip } from "antd";
import { EyeOutlined, StopOutlined } from "@ant-design/icons";
import type { ConfigExecution, ConfigExecutionDetail, NetworkDevice } from "@/types";
import ActionButtons from "@/components/shared/ActionButtons";
import { formatDateTime } from "@/utils/datetime";
import { renderExecutionStatusTag, renderSimpleStatusTag } from "./constants";

// 设备表格列
export const deviceColumns: ColumnsType<NetworkDevice> = [
  { title: "设备名称", dataIndex: "deviceName", key: "deviceName" },
  { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress" },
  { title: "厂商", dataIndex: "vendor", key: "vendor" },
];

// 执行记录表格列
export interface ExecutionColumnsParams {
  handleViewDetail: (record: ConfigExecution) => void;
  handleCancel: (id: string) => void;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

export function getExecutionColumns(
  params: ExecutionColumnsParams
): ColumnsType<ConfigExecution> {
  const { handleViewDetail, handleCancel, getSortOrder } = params;

  return [
    { title: "任务名称", dataIndex: "executionName", key: "executionName", width: 200, sorter: true, sortOrder: getSortOrder?.("executionName") },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status: string) => renderExecutionStatusTag(status),
    },
    { title: "设备总数", dataIndex: "totalDevices", key: "totalDevices", width: 100 },
    { title: "成功数", dataIndex: "successCount", key: "successCount", width: 80 },
    { title: "失败数", dataIndex: "failureCount", key: "failureCount", width: 80 },
    {
      title: "进度",
      key: "progress",
      width: 150,
      render: (_: unknown, record: ConfigExecution) => {
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
      render: (_: unknown, record: ConfigExecution) => {
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
                      onOk: () => handleCancel(record.id),
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

// 执行明细表格列
export const detailColumns: ColumnsType<ConfigExecutionDetail> = [
  { title: "设备名称", dataIndex: "deviceName", key: "deviceName", width: 150 },
  { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress", width: 140 },
  {
    title: "状态",
    dataIndex: "status",
    key: "status",
    width: 100,
    render: (status: string) => renderSimpleStatusTag(status),
  },
  {
    title: "命令内容",
    dataIndex: "commandSent",
    key: "commandSent",
    ellipsis: true,
    render: (command: string) => command || "-",
  },
  { title: "开始时间", dataIndex: "startedAt", key: "startedAt", width: 180 },
  { title: "完成时间", dataIndex: "completedAt", key: "completedAt", width: 180 },
  { title: "耗时(秒)", dataIndex: "duration", key: "duration", width: 100 },
  {
    title: "输出",
    key: "output",
    width: 100,
    render: (_: unknown, record: ConfigExecutionDetail) =>
      record.outputReceived ? (
        <Button
          type="link"
          size="small"
          onClick={() => {
            Modal.info({
              title: "命令输出",
              width: 700,
              content: (
                <pre
                  style={{
                    maxHeight: 400,
                    overflow: "auto",
                    background: "#f5f5f5",
                    padding: 12,
                  }}
                >
                  {record.outputReceived}
                </pre>
              ),
            });
          }}
        >
          查看
        </Button>
      ) : (
        "-"
      ),
  },
  {
    title: "错误信息",
    dataIndex: "errorMessage",
    key: "errorMessage",
    width: 150,
    ellipsis: true,
    render: (msg: string) =>
      msg ? (
        <Tooltip title={msg}>
          <span style={{ color: "var(--theme-error, #ff4d4f)" }}>{msg}</span>
        </Tooltip>
      ) : (
        "-"
      ),
  },
];

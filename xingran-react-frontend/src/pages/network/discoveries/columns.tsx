/**
 * Device Discovery Table Columns
 * 设备发现表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Button, Tag, Progress, Modal } from "antd";
import { EyeOutlined, DeleteOutlined } from "@ant-design/icons";
import type { DeviceDiscovery, NetworkDevice } from "@/types";
import { DISCOVERY_TYPE_OPTIONS, STATUS_CONFIG, renderStatusTag } from "./constants";
import ActionButtons from "@/components/shared/ActionButtons";

export interface DiscoveryColumnsParams {
  handleViewResult: (record: DeviceDiscovery) => void;
  handleDelete: (id: string) => Promise<void>;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

export function getDiscoveryColumns(params: DiscoveryColumnsParams): ColumnsType<DeviceDiscovery> {
  const { handleViewResult, handleDelete, getSortOrder } = params;

  return [
    {
      title: "任务名称",
      dataIndex: "taskName",
      key: "taskName",
      width: 200,
      sorter: true,
      sortOrder: getSortOrder?.("taskName"),
    },
    {
      title: "发现类型",
      dataIndex: "discoveryType",
      key: "discoveryType",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("discoveryType"),
      render: (type: string) => {
        const option = DISCOVERY_TYPE_OPTIONS.find((o) => o.value === type);
        return <Tag color="blue">{option?.label}</Tag>;
      },
    },
    {
      title: "IP范围",
      dataIndex: "ipRanges",
      key: "ipRanges",
      width: 200,
      render: (ranges: string[]) => ranges.join(", "),
    },
    { title: "总IP数", dataIndex: "totalIPs", key: "totalIPs", width: 100 },
    { title: "发现数", dataIndex: "discoveredCount", key: "discoveredCount", width: 100 },
    {
      title: "进度",
      key: "progress",
      width: 150,
      render: (_, record) => {
        if (record.status === "completed") {
          return <Progress percent={100} size="small" />;
        }
        if (record.status === "running") {
          const percent =
            record.totalIPs > 0 ? Math.round((record.discoveredCount / record.totalIPs) * 100) : 0;
          return <Progress percent={percent} size="small" status="active" />;
        }
        return "-";
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getSortOrder?.("status"),
      render: (status: string) =>
        renderStatusTag(status as "pending" | "running" | "completed" | "failed"),
    },
    {
      title: "开始时间",
      dataIndex: "startedAt",
      key: "startedAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("startedAt"),
    },
    {
      title: "完成时间",
      dataIndex: "completedAt",
      key: "completedAt",
      width: 180,
      sorter: true,
      sortOrder: getSortOrder?.("completedAt"),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 180,
      render: (_, record) => {
        const actions = [
          {
            key: "view-result",
            label: "查看结果",
            icon: <EyeOutlined />,
            onClick: () => handleViewResult(record),
            disabled: record.status === "pending",
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确认删除?",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

/** 发现结果表格列 */
export const deviceColumns: ColumnsType<NetworkDevice> = [
  { title: "设备名称", dataIndex: "deviceName", key: "deviceName" },
  { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress" },
  { title: "厂商", dataIndex: "vendor", key: "vendor" },
  { title: "型号", dataIndex: "model", key: "model" },
  { title: "设备类型", dataIndex: "deviceType", key: "deviceType" },
];

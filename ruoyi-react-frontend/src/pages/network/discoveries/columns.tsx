/**
 * Device Discovery Table Columns
 * 设备发现表格列定义
 *
 * Phase 67 T1 迁移收窄：操作列整体挪至页面 rowActions（查看结果 + createDeleteAction
 * → ConfirmDelete）；getSortOrder 手工视觉与删除 confirm 块删除——排序视觉/确认
 * 由 antd 原生受控排序与 ConfirmDelete 唯一入口承接。其余列逐字未动。
 */

import type { ColumnsType } from "antd/es/table";
import { Tag, Progress } from "antd";
import type { DeviceDiscovery, NetworkDevice } from "@/types";
import { DISCOVERY_TYPE_OPTIONS, renderStatusTag } from "./constants";

export function getDiscoveryColumns(): ColumnsType<DeviceDiscovery> {
  return [
    {
      title: "任务名称",
      dataIndex: "taskName",
      key: "taskName",
      width: 200,
      sorter: true,
    },
    {
      title: "发现类型",
      dataIndex: "discoveryType",
      key: "discoveryType",
      width: 100,
      sorter: true,
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
      render: (status: string) =>
        renderStatusTag(status as "pending" | "running" | "completed" | "failed"),
    },
    {
      title: "开始时间",
      dataIndex: "startedAt",
      key: "startedAt",
      width: 180,
      sorter: true,
    },
    {
      title: "完成时间",
      dataIndex: "completedAt",
      key: "completedAt",
      width: 180,
      sorter: true,
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

/**
 * 网络设备表格列定义工厂（v1.24 HYG-04 / D1）。
 *
 * 从 index.tsx 原样搬迁（零行为变化）：
 *   - 列集合、宽度、sorter 标记、render 实现、字典回退语义逐字保留；
 *   - 「列 sorter:true 由 PageTable/useTableManager 排序管线承接」的注释保留。
 *
 * 为什么独立成文件：列定义是纯函数（无 hook、无副作用、无 state），
 * 抽出对本页渲染链路零影响，是拆分中风险最低的一刀。
 *
 * ⚠️ 依赖纪律：本工厂接收的 props 与调用点保持一一对应（props 里包含
 * `handleCollectPorts` / `openDetailModal` / `openModal` / `collectingDeviceId`
 * / `canViewMACHistory`），调用点不在本刀改动 —— 见 index.tsx 的 columns useMemo
 * （其依赖数组**故意不完整**，见该处 eslint-disable 注释，拆分时不得"顺手修正"）。
 */

import { Tag } from "antd";
import type { NetworkDevice } from "@/types";
import type { ColumnsType } from "antd/es/table";
import {
  DEVICE_TYPE_TAG_COLOR,
  STATUS_OPTIONS,
  VENDOR_OPTIONS,
  VENDOR_TAG_COLOR,
} from "../constants";
import { formatDateTime, getOptionLabel, getStatusColor } from "../utils";

export interface DeviceTableColumnsProps {
  navigate: (path: string) => void;
  handleCollectPorts: (device: NetworkDevice) => void;
  openDetailModal: (device: NetworkDevice) => void;
  openModal: (record?: NetworkDevice) => void;
  collectingDeviceId: string | null;
  canViewMACHistory: boolean;
  /** Phase 69 DICT-03: 设备类型 label 渲染（优先字典，回退静态 DEVICE_TYPE_OPTIONS） */
  getDeviceTypeText: (type: string) => string | undefined;
}

/** 列 sorter:true 由 PageTable/useTableManager 排序管线承接；原手工 sortOrder 视觉字段随契约删除 */
export function getDeviceTableColumns(props: DeviceTableColumnsProps): ColumnsType<NetworkDevice> {
  const { navigate, getDeviceTypeText } = props;

  return [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 200,
      sorter: true,
      render: (deviceName: string, record: NetworkDevice) => (
        <a
          onClick={() => navigate(`/network/ports?deviceId=${record.id}`)}
          style={{ color: "var(--theme-info)", cursor: "pointer" }}
        >
          {deviceName}
        </a>
      ),
    },
    {
      title: "设备类型",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 120,
      sorter: true,
      render: (deviceType: string) => (
        <Tag color={DEVICE_TYPE_TAG_COLOR}>{getDeviceTypeText(deviceType)}</Tag>
      ),
    },
    {
      title: "厂商",
      dataIndex: "vendor",
      key: "vendor",
      width: 100,
      sorter: true,
      render: (vendor: string) => (
        <Tag color={VENDOR_TAG_COLOR}>{getOptionLabel(VENDOR_OPTIONS, vendor)}</Tag>
      ),
    },
    {
      title: "型号",
      dataIndex: "model",
      key: "model",
      width: 150,
      sorter: true,
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
      sorter: true,
    },
    {
      title: "端口",
      dataIndex: "port",
      key: "port",
      width: 80,
      sorter: true,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 90,
      sorter: true,
      render: (status: number) => (
        <Tag color={getStatusColor(status)}>{getOptionLabel(STATUS_OPTIONS, status)}</Tag>
      ),
    },
    { title: "部门", dataIndex: "deptName", key: "deptName", width: 180 },
    {
      title: "最后连接",
      dataIndex: "lastSeenAt",
      key: "lastSeenAt",
      width: 180,
      sorter: true,
      render: (lastSeenAt: string) => formatDateTime(lastSeenAt),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      render: (createdAt: string) => formatDateTime(createdAt),
    },
  ];
}

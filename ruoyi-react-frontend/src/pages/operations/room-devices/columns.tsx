import type { ColumnsType } from "antd/es/table";
import { Tag } from "antd";
import { EditOutlined } from "@ant-design/icons";
import type { RoomDevice } from "@/types";
import { createDateTimeColumn } from "@/utils/tableHelpers";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { getStatusText, getDeviceTypeText } from "./constants";

export const getRoomDeviceColumns = (): ColumnsType<RoomDevice> => [
  {
    title: "设备名称",
    dataIndex: "name",
    key: "name",
    width: 150,
    sorter: true,
  },
  {
    title: "设备编码",
    dataIndex: "deviceCode",
    key: "deviceCode",
    width: 150,
    sorter: true,
  },
  {
    title: "所属机房",
    dataIndex: "roomName",
    key: "roomName",
    width: 120,
    sorter: true,
  },
  {
    title: "设备类型",
    dataIndex: "deviceType",
    key: "deviceType",
    width: 120,
    sorter: true,
    render: (type: string) => getDeviceTypeText(type),
  },
  {
    title: "厂商",
    dataIndex: "vendor",
    key: "vendor",
    width: 120,
    render: (v) => v || "-",
  },
  {
    title: "型号",
    dataIndex: "model",
    key: "model",
    width: 120,
    render: (v) => v || "-",
  },
  {
    title: "位置",
    dataIndex: "positionDesc",
    key: "positionDesc",
    width: 100,
    render: (v) => v || "-",
  },
  {
    title: "状态",
    dataIndex: "status",
    key: "status",
    width: 80,
    sorter: true,
    render: (status) => {
      const colors: Record<number, string> = { 0: "success", 1: "error", 2: "default" };
      return (
        <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status as number)}</Tag>
      );
    },
  },
  createDateTimeColumn("createdAt", {
    width: 180,
    sorter: true,
  }),
];

export const getRoomDeviceRowActions = (
  openModal: (record?: RoomDevice) => void,
  handleDelete: (id: string) => Promise<void>
) => (record: RoomDevice) => [
  {
    key: "edit",
    label: "编辑",
    icon: <EditOutlined />,
    onClick: () => openModal(record),
    permission: PERMISSION_CODES.opsRoomDevice.edit,
  },
  createDeleteAction(record.id, handleDelete, {
    target: `机房设备 ${record.name}`,
    permission: PERMISSION_CODES.opsRoomDevice.delete,
  }),
];

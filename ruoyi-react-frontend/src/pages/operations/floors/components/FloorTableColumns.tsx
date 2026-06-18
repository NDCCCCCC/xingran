/**
 * 楼层表格列配置
 */

import type { ColumnsType } from "antd/es/table";
import { Modal, Button } from "antd";
import { EditOutlined, DeleteOutlined, BgColorsOutlined } from "@ant-design/icons";
import type { Floor } from "@/types";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import type { SortOrder } from "@/hooks/useServerSort";

interface FloorTableColumnCallbacks {
  onEdit: (floor: Floor) => void;
  onEditFloorPlan: (floor: Floor) => void;
  onDelete: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

export function createFloorTableColumns(callbacks: FloorTableColumnCallbacks): ColumnsType<Floor> {
  const { onEdit, onEditFloorPlan, onDelete, getColumnSortOrder } = callbacks;

  return [
    { title: "楼层名称", dataIndex: "name", key: "name", width: 150, sorter: true, sortOrder: getColumnSortOrder("name") },
    { title: "楼层号", dataIndex: "floorNo", key: "floorNo", width: 100, sorter: true, sortOrder: getColumnSortOrder("floorNo") },
    {
      title: "所属楼宇",
      dataIndex: "buildingName",
      key: "buildingName",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("buildingName"),
      render: (_, record) => record.buildingName || record.buildingCode,
    },
    {
      title: "面积(m²)",
      dataIndex: "area",
      key: "area",
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("area"),
      render: (value) => value || "-",
    },
    createStatusColumn("status", { width: 100, sorter: true, sortOrder: getColumnSortOrder("status") }),
    { title: "描述", dataIndex: "description", key: "description", ellipsis: true },
    createDateTimeColumn("createdAt", { width: 180, sorter: true, sortOrder: getColumnSortOrder("createdAt") }),
    {
      title: "操作",
      key: "action",
      render: (_, record) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => onEdit(record),
          },
          {
            key: "floorPlan",
            label: "查看平面图",
            icon: <BgColorsOutlined />,
            onClick: () => onEditFloorPlan(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个楼层吗？",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => onDelete(record.id),
              });
            },
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

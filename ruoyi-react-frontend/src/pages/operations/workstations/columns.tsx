/**
 * Workstation Columns
 * 工位表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { WorkstationOps } from "@/types";
import ActionButtons from "@/components/shared/ActionButtons";
import { renderWorkstationTypeTag, renderWorkstationStatusTag } from "./constants";
import { createDateTimeColumn } from "@/utils/tableHelpers";
import type { SortOrder } from "@/hooks/useServerSort";

export interface WorkstationColumnsParams {
  handleEdit: (record: WorkstationOps) => void;
  handleDelete: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
  /**
   * Phase 86 W2：行操作权限码**由页面层传入**（列工厂内不得调 hook）。
   * 不传 = 不做权限判定（保持既有调用方行为零变更）。
   * ⚠️ ops 码组只有 delete 键，没有 remove 键。
   */
  editPermission?: string;
  deletePermission?: string;
}

export function getWorkstationColumns(
  params: WorkstationColumnsParams
): ColumnsType<WorkstationOps> {
  const { handleEdit, handleDelete, getColumnSortOrder, editPermission, deletePermission } = params;

  return [
    {
      title: "工位名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("name"),
    },
    {
      title: "所属楼宇",
      dataIndex: "buildingName",
      key: "buildingName",
      width: 120,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("buildingName"),
      render: (text) => text || "-",
    },
    {
      title: "所属楼层",
      dataIndex: "floorName",
      key: "floorName",
      width: 120,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("floorName"),
      render: (text) => text || "-",
    },
    {
      title: "所属部门",
      dataIndex: "deptName",
      key: "deptName",
      width: 120,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("deptName"),
      render: (text) => text || "-",
    },
    {
      title: "所属用户",
      dataIndex: "userName",
      key: "userName",
      width: 100,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("userName"),
      render: (text) => text || "-",
    },
    {
      title: "主设备序列号",
      dataIndex: "primaryDeviceSerial",
      key: "primaryDeviceSerial",
      width: 140,
      ellipsis: true,
      render: (text) => text || "-",
    },
    {
      title: "工位类型",
      dataIndex: "type",
      key: "type",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("type"),
      render: (type) => renderWorkstationTypeTag(type),
    },
    // 工位状态为 3 态枚举（0=空闲, 1=占用, 2=维护），不能使用 createStatusColumn（该 helper 硬编码 0=正常/非0=停用的 2 态语义）。
    // 改用本模块的 renderWorkstationStatusTag，与卡片视图/平面图/模态框/搜索下拉保持一致。
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: (status: number) => renderWorkstationStatusTag(status),
    },
    { title: "描述", dataIndex: "description", key: "description", ellipsis: true },
    createDateTimeColumn("createdAt", {
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
    }),
    createDateTimeColumn("updatedAt", {
      width: 180,
      title: "更新时间",
      sorter: true,
      sortOrder: getColumnSortOrder("updatedAt"),
    }),
    {
      title: "操作",
      key: "action",
      render: (_: unknown, record: WorkstationOps) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
            permission: editPermission,
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            // Phase 86 W2：confirmDelete 无 permission 位 → 由 ActionButtons 的
            // permission 字段承担三态门控（ready 无码不渲染 / 未就绪 disabled）
            permission: deletePermission,
            onClick: () =>
              confirmDelete({
                target: `工位 ${record.name}`,
                onDelete: async () => { await handleDelete(record.id); },
              }),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

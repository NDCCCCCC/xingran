/**
 * 角色表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { EditOutlined, DeleteOutlined, StopOutlined, CheckCircleOutlined } from "@ant-design/icons";
import { confirmDelete } from "@/components/shared/confirmDelete";
import ActionButtons from "@/components/shared/ActionButtons";
import type { Role } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { renderRoleName, renderRoleKeyTag, renderStatusTag } from "../utils";
import type { SortOrder } from "@/hooks/useServerSort";
// Phase 86 W1（RBAC-02）：行操作权限码单一真相源（不在列工厂内调 hook —— 列工厂保持纯函数）。
import { PERMISSION_CODES } from "@/constants/permissionCodes";

interface GetRoleColumnsParams {
  handleEdit: (record: Role) => void;
  handleUpdateStatus: (id: string, status: number) => void;
  handleDelete: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

export function getRoleColumns(params: GetRoleColumnsParams): ColumnsType<Role> {
  const { handleEdit, handleUpdateStatus, handleDelete, getColumnSortOrder } = params;

  return [
    {
      title: "角色名称",
      dataIndex: "roleName",
      key: "roleName",
      sorter: true,
      sortOrder: getColumnSortOrder("roleName"),
      render: renderRoleName,
    },
    {
      title: "权限字符",
      dataIndex: "roleKey",
      key: "roleKey",
      sorter: true,
      sortOrder: getColumnSortOrder("roleKey"),
      render: renderRoleKeyTag,
    },
    {
      title: "显示顺序",
      dataIndex: "roleSort",
      key: "roleSort",
      sorter: true,
      sortOrder: getColumnSortOrder("roleSort"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: renderStatusTag,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (text: string) => formatDateTime(text),
    },
    {
      title: "操作",
      key: "action",
      render: (_: unknown, record: Role) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
            // Phase 86 W1（RBAC-02）：行编辑 → system:role:edit。
            // permission 字段由 ActionButtons 内建三态门控消费（就绪且无码 → 不渲染；未就绪 → 渲染但 disabled + 提示）。
            permission: PERMISSION_CODES.systemRole.edit,
          },
          {
            key: "toggle-status",
            label: record.status === 0 ? "停用" : "启用",
            icon: record.status === 0 ? <StopOutlined /> : <CheckCircleOutlined />,
            onClick: () => handleUpdateStatus(record.id, record.status === 0 ? 1 : 0),
            // 启停（停用/启用）是状态写动作，就近归 system:role:edit —— systemRole 码组只有
            // list/view/add/edit/remove/export，无独立 status 键；按「归最接近既有码」裁定处理，
            // 不自造新码（Phase 86 零新增权限码）。
            permission: PERMISSION_CODES.systemRole.edit,
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            // 行删除与批量删除同键 → system:role:remove。
            permission: PERMISSION_CODES.systemRole.remove,
            onClick: () => {
              confirmDelete({
                target: `角色 ${record.roleName}`,
                onDelete: async () => { await handleDelete(record.id); },
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

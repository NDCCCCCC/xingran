/**
 * 角色表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { EditOutlined, DeleteOutlined, StopOutlined, CheckCircleOutlined } from "@ant-design/icons";
import { Modal, Button, Space, Tag } from "antd";
import ActionButtons from "@/components/shared/ActionButtons";
import type { Role } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { renderRoleName, renderRoleKeyTag, renderStatusTag } from "../utils";
import type { SortOrder } from "@/hooks/useServerSort";

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
          },
          {
            key: "toggle-status",
            label: record.status === 0 ? "停用" : "启用",
            icon: record.status === 0 ? <StopOutlined /> : <CheckCircleOutlined />,
            onClick: () => handleUpdateStatus(record.id, record.status === 0 ? 1 : 0),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个角色吗？",
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

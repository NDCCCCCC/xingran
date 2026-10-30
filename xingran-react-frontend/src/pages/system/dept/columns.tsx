/**
 * Department Columns
 * 部门表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Space, Tag } from "antd";
import {
  ApartmentOutlined,
  PhoneOutlined,
  MailOutlined,
  EditOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { Department } from "@/types";
import { renderStatusTag, renderExternalOrgTag } from "./constants";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { formatDateTime } from "@/utils/datetime";
import ActionButtons from "@/components/shared/ActionButtons";
// Phase 86 W1（RBAC-02）：行操作权限码单一真相源（不在列工厂内调 hook —— 列工厂保持纯函数）。
import { PERMISSION_CODES } from "@/constants/permissionCodes";

export interface DeptColumnsParams {
  handleEdit: (record: Department) => void;
  handleDelete: (id: string) => Promise<void>;
}

export function getDeptColumns(params: DeptColumnsParams): ColumnsType<Department> {
  const { handleEdit, handleDelete } = params;

  return [
    {
      title: "部门名称",
      dataIndex: "deptName",
      key: "deptName",
      render: (text, record) => (
        <Space style={{ opacity: record.accessible === false ? 0.5 : 1 }}>
          <ApartmentOutlined />
          {text}
          {record.accessible === false && (
            <Tag color="default" style={{ fontSize: "12px", marginLeft: 4 }}>
              无权限
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: "负责人",
      dataIndex: "leader",
      key: "leader",
      render: (_: unknown, record: Department) => {
        if (record.leaderName && record.leaderUsername) {
          return `${record.leaderName}（${record.leaderUsername}）`;
        }
        return "-";
      },
    },
    {
      title: "联系电话",
      dataIndex: "phone",
      key: "phone",
      render: (phone) =>
        phone ? (
          <Space>
            <PhoneOutlined />
            {phone}
          </Space>
        ) : (
          "-"
        ),
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
      render: (email) =>
        email ? (
          <Space>
            <MailOutlined />
            {email}
          </Space>
        ) : (
          "-"
        ),
    },
    {
      title: "显示顺序",
      dataIndex: "orderNum",
      key: "orderNum",
    },
    {
      title: "外部机构",
      dataIndex: "isExternalOrg",
      key: "isExternalOrg",
      render: (isExternalOrg: number) => renderExternalOrgTag(isExternalOrg),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (status) => renderStatusTag(status),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      render: (text) => formatDateTime(text),
    },
    {
      title: "操作",
      key: "action",
      render: (_: unknown, record: Department) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
            disabled: record.accessible === false,
            // Phase 86 W1（RBAC-02）：行编辑 → system:dept:edit。
            // permission 字段由 ActionButtons 内建三态门控消费（就绪且无码 → 不渲染；未就绪 → 渲染但 disabled + 提示）。
            permission: PERMISSION_CODES.systemDept.edit,
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            disabled: record.accessible === false,
            // 行删除 → system:dept:remove。
            permission: PERMISSION_CODES.systemDept.remove,
            onClick: () =>
              confirmDelete({
                target: `部门 ${record.deptName}`,
                onDelete: () => handleDelete(record.id),
              }),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

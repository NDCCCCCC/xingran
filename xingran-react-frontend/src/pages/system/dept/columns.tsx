/**
 * Department Columns
 * 部门表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Space, Tag, Modal } from "antd";
import {
  ApartmentOutlined,
  PhoneOutlined,
  MailOutlined,
  EditOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { Department } from "@/types";
import { renderStatusTag, renderExternalOrgTag } from "./constants";
import { formatDateTime } from "@/utils/datetime";
import ActionButtons from "@/components/shared/ActionButtons";

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
            <Tag color="default" style={{ fontSize: "12px", marginLeft: 4 }}>无权限</Tag>
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
      render: (phone) => phone ? (
        <Space>
          <PhoneOutlined />
          {phone}
        </Space>
      ) : "-",
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
      render: (email) => email ? (
        <Space>
          <MailOutlined />
          {email}
        </Space>
      ) : "-",
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
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            disabled: record.accessible === false,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个部门吗？",
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

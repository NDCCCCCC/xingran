/**
 * Network Template Columns
 * 网络模板表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Modal } from "antd";
import { EyeOutlined, CopyOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { ConfigTemplate } from "@/types";
import ActionButtons from "@/components/shared/ActionButtons";
import { formatDateTime } from "@/utils/datetime";
import {
  renderVendorTag,
  renderDeviceTypeTag,
  renderTemplateTypeTag,
  renderSystemTemplateTag,
} from "./constants";
import type { SorterMeta } from "@/utils/tableHelpers";

export interface TemplateColumnsParams {
  handlePreview: (id: string) => void;
  handleClone: (id: string) => void;
  handleDelete: (id: string) => void;
  openModal: (record: ConfigTemplate) => void;
  /** 列级 sortOrder：返回当前排序列的方向，其余 undefined（受控高亮） */
  getColumnSortOrder?: (field: string) => "ascend" | "descend" | null | undefined;
  /** 可排序列白名单（对应后端白名单 key） */
  sorterMetas?: Array<SorterMeta<ConfigTemplate> | undefined>;
}

export function getTemplateColumns(params: TemplateColumnsParams): ColumnsType<ConfigTemplate> {
  const { handlePreview, handleClone, handleDelete, openModal, getColumnSortOrder } = params;

  return [
    {
      title: "模板名称",
      dataIndex: "templateName",
      key: "templateName",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder?.("templateName"),
    },
    {
      title: "模板编码",
      dataIndex: "templateCode",
      key: "templateCode",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder?.("templateCode"),
    },
    {
      title: "模板类型",
      dataIndex: "templateType",
      key: "templateType",
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder?.("templateType"),
      render: (templateType: string) => renderTemplateTypeTag(templateType),
    },
    {
      title: "适用厂商",
      dataIndex: "vendor",
      key: "vendor",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder?.("vendor"),
      render: (vendor: string) => renderVendorTag(vendor),
    },
    {
      title: "适用设备",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder?.("deviceType"),
      render: (deviceType: string) => renderDeviceTypeTag(deviceType),
    },
    {
      title: "系统模板",
      dataIndex: "isSystem",
      key: "isSystem",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder?.("isSystem"),
      render: (isSystem: boolean) => renderSystemTemplateTag(isSystem),
    },
    {
      title: "版本",
      dataIndex: "version",
      key: "version",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder?.("version"),
    },
    {
      title: "备注",
      dataIndex: "description",
      key: "description",
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder?.("description"),
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder?.("updatedAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_: unknown, record: ConfigTemplate) => {
        const actions = [
          {
            key: "preview",
            label: "预览",
            icon: <EyeOutlined />,
            onClick: () => handlePreview(record.id),
          },
          {
            key: "clone",
            label: "克隆",
            icon: <CopyOutlined />,
            onClick: () => handleClone(record.id),
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openModal(record),
          },
          ...(!record.isSystem
            ? [
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
              ]
            : []),
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

/**
 * Network Template Columns
 * 网络模板表格列定义
 */

import type { ColumnsType } from 'antd/es/table';
import { Button, Space, Tag, Popconfirm } from 'antd';
import {
  EyeOutlined,
  CopyOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { ConfigTemplate } from '@/types';
import ActionButtons from '@/components/shared/ActionButtons';
import { formatDateTime } from '@/utils/datetime';
import {
  renderVendorTag,
  renderDeviceTypeTag,
  renderTemplateTypeTag,
  renderSystemTemplateTag,
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  TEMPLATE_TYPE_OPTIONS,
} from './constants';

export interface TemplateColumnsParams {
  handlePreview: (id: string) => void;
  handleClone: (id: string) => void;
  handleDelete: (id: string) => void;
  openModal: (record: ConfigTemplate) => void;
}

export function getTemplateColumns(params: TemplateColumnsParams): ColumnsType<ConfigTemplate> {
  const { handlePreview, handleClone, handleDelete, openModal } = params;

  return [
    { title: '模板名称', dataIndex: 'templateName', key: 'templateName', width: 180 },
    { title: '模板编码', dataIndex: 'templateCode', key: 'templateCode', width: 150 },
    {
      title: '模板类型',
      dataIndex: 'templateType',
      key: 'templateType',
      width: 120,
      render: (templateType: string) => renderTemplateTypeTag(templateType),
    },
    {
      title: '适用厂商',
      dataIndex: 'vendor',
      key: 'vendor',
      width: 100,
      render: (vendor: string) => renderVendorTag(vendor),
    },
    {
      title: '适用设备',
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 120,
      render: (deviceType: string) => renderDeviceTypeTag(deviceType),
    },
    {
      title: '系统模板',
      dataIndex: 'isSystem',
      key: 'isSystem',
      width: 100,
      render: (isSystem: boolean) => renderSystemTemplateTag(isSystem),
    },
    { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
    { title: '备注', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 100,
      render: (_: unknown, record: ConfigTemplate) => {
        const actions = [
          {
            key: 'preview',
            label: '预览',
            icon: <EyeOutlined />,
            onClick: () => handlePreview(record.id),
          },
          {
            key: 'clone',
            label: '克隆',
            icon: <CopyOutlined />,
            onClick: () => handleClone(record.id),
          },
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => openModal(record),
          },
          ...(!record.isSystem
            ? [
                {
                  key: 'delete',
                  label: '删除',
                  icon: <DeleteOutlined />,
                  danger: true,
                  render: () => (
                    <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
                      <Button type="link" danger ghost icon={<DeleteOutlined />} size="small">
                        删除
                      </Button>
                    </Popconfirm>
                  ),
                },
              ]
            : []),
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

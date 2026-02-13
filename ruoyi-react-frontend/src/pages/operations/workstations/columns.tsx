/**
 * Workstation Columns
 * 工位表格列定义
 */

import type { ColumnsType } from 'antd/es/table';
import { Modal } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { WorkstationOps } from '@/types';
import ActionButtons from '@/components/shared/ActionButtons';
import { renderWorkstationTypeTag } from './constants';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';

export interface WorkstationColumnsParams {
  handleEdit: (record: WorkstationOps) => void;
  handleDelete: (id: string) => void;
}

export function getWorkstationColumns(params: WorkstationColumnsParams): ColumnsType<WorkstationOps> {
  const { handleEdit, handleDelete } = params;

  return [
    { title: '工位名称', dataIndex: 'name', key: 'name', width: 150 },
    {
      title: '所属楼宇',
      dataIndex: 'buildingName',
      key: 'buildingName',
      width: 120,
      ellipsis: true,
      render: (text) => text || '-',
    },
    {
      title: '所属楼层',
      dataIndex: 'floorName',
      key: 'floorName',
      width: 120,
      ellipsis: true,
      render: (text) => text || '-',
    },
    {
      title: '工位类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type) => renderWorkstationTypeTag(type),
    },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: WorkstationOps) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: '确定要删除这个工位吗？',
                okText: '确定',
                cancelText: '取消',
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

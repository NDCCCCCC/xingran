/**
 * 楼层表格列配置
 */

import type { ColumnsType } from 'antd/es/table';
import { Popconfirm, Button } from 'antd';
import { EditOutlined, DeleteOutlined, BgColorsOutlined } from '@ant-design/icons';
import type { Floor } from '@/types';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';

interface FloorTableColumnCallbacks {
  onEdit: (floor: Floor) => void;
  onEditFloorPlan: (floor: Floor) => void;
  onDelete: (id: string) => void;
}

export function createFloorTableColumns(callbacks: FloorTableColumnCallbacks): ColumnsType<Floor> {
  const { onEdit, onEditFloorPlan, onDelete } = callbacks;

  return [
    { title: '楼层名称', dataIndex: 'name', key: 'name', width: 150 },
    { title: '楼层号', dataIndex: 'floorNo', key: 'floorNo', width: 100 },
    {
      title: '所属楼宇',
      dataIndex: 'buildingName',
      key: 'buildingName',
      width: 150,
      render: (_, record) => record.buildingName || record.buildingCode,
    },
    {
      title: '面积(m²)',
      dataIndex: 'area',
      key: 'area',
      width: 120,
      render: (value) => value || '-',
    },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作',
      key: 'action',
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => onEdit(record),
          },
          {
            key: 'floorPlan',
            label: '查看平面图',
            icon: <BgColorsOutlined />,
            onClick: () => onEditFloorPlan(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个楼层吗？"
                onConfirm={() => onDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

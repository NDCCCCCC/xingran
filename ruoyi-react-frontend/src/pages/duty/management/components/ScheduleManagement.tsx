import React from 'react';
import { Card, Form, Select, Table, Button, Space, Popconfirm, Tag, DatePicker } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  SwapOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { DutySchedule, DutyPool, SimpleUser } from '@/lib/dutyApi';

const { RangePicker } = DatePicker;

interface ScheduleManagementProps {
  schedules: DutySchedule[];
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  selectedRowKeys: string[];
  pools: DutyPool[];
  users: SimpleUser[];
  onSearch: (values: any) => void;
  onReset: () => void;
  onPageChange: (page: number, size: number) => void;
  onSelectedChange: (keys: string[]) => void;
  onDelete: (id: string) => void;
  onBatchDelete: () => void;
  onGenerateClick: () => void;
  onSwapClick: () => void;
  onManualClick: () => void;
}

export const ScheduleManagement: React.FC<ScheduleManagementProps> = ({
  schedules,
  loading,
  total,
  current,
  pageSize,
  selectedRowKeys,
  pools,
  users,
  onSearch,
  onReset,
  onPageChange,
  onSelectedChange,
  onDelete,
  onBatchDelete,
  onGenerateClick,
  onSwapClick,
  onManualClick,
}) => {
  const [searchForm] = Form.useForm();

  const getDutyTypeColor = (type: string) => {
    switch (type) {
      case 'weekday':
        return 'blue';
      case 'weekend':
        return 'orange';
      case 'holiday':
        return 'red';
      default:
        return 'default';
    }
  };

  const getDutyTypeText = (type: string) => {
    switch (type) {
      case 'weekday':
        return '工作日';
      case 'weekend':
        return '周末';
      case 'holiday':
        return '节假日';
      default:
        return type;
    }
  };

  const columns: ColumnsType<DutySchedule> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: '值班日期',
      dataIndex: 'scheduleDate',
      key: 'scheduleDate',
      width: 120,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD'),
    },
    {
      title: '星期',
      key: 'weekday',
      width: 80,
      render: (_: unknown, record: DutySchedule) => {
        const weekday = dayjs(record.scheduleDate).day();
        const weekdays = ['日', '一', '二', '三', '四', '五', '六'];
        return `周${weekdays[weekday]}`;
      },
    },
    {
      title: '值班池',
      dataIndex: ['pool', 'poolName'],
      key: 'poolName',
      width: 120,
    },
    {
      title: '值班人员',
      dataIndex: ['user', 'nickName'],
      key: 'userName',
      width: 100,
      render: (nickName: string, record: DutySchedule) =>
        nickName || record.user?.username || '-',
    },
    {
      title: '值班类型',
      dataIndex: 'dutyType',
      key: 'dutyType',
      width: 100,
      render: (type: string) => <Tag color={getDutyTypeColor(type)}>{getDutyTypeText(type)}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => {
        if (status === 0) return <Tag color="green">正常</Tag>;
        if (status === 1) return <Tag color="orange">已调换</Tag>;
        if (status === 2) return <Tag color="red">已取消</Tag>;
        return <Tag>未知</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: DutySchedule) => (
        <Popconfirm title="确定要删除这条排班吗？" onConfirm={() => onDelete(record.id)} okText="确定" cancelText="取消">
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Card variant="borderless">
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="poolId">
          <Select placeholder="值班池" allowClear style={{ width: 140 }}>
            {pools.map((pool) => (
              <Select.Option key={pool.id} value={pool.id}>{pool.poolName}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="userId">
          <Select placeholder="值班人员" allowClear style={{ width: 120 }}>
            {users.map((user) => (
              <Select.Option key={user.id} value={user.id}>{user.nickName || user.username}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="dutyType">
          <Select placeholder="值班类型" allowClear style={{ width: 120 }}>
            <Select.Option value="weekday">工作日</Select.Option>
            <Select.Option value="weekend">周末</Select.Option>
            <Select.Option value="holiday">节假日</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item name="dateRange">
          <RangePicker style={{ width: 240 }} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => onSearch(searchForm.getFieldsValue())}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); onReset(); }}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={onGenerateClick}>
            生成排班
          </Button>
          <Button size="small" icon={<SwapOutlined />} onClick={onSwapClick}>
            调班
          </Button>
          <Button size="small" icon={<EditOutlined />} onClick={onManualClick}>
            手动排班
          </Button>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title="确定要批量删除选中的排班记录吗？"
              onConfirm={onBatchDelete}
              okText="确定"
              cancelText="取消"
            >
              <Button size="small" danger icon={<DeleteOutlined />}>
                批量删除 ({selectedRowKeys.length})
              </Button>
            </Popconfirm>
          )}
        </Space>
      </div>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={schedules}
        loading={loading}
        scroll={{ x: 1100 }}
        rowSelection={{
          selectedRowKeys,
          onChange: (selectedKeys) => onSelectedChange(selectedKeys.map(String)),
        }}
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t) => `共 ${t} 条`,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: onPageChange,
        }}
      />
    </Card>
  );
};

export default ScheduleManagement;

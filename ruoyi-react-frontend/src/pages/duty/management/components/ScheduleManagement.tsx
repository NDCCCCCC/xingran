import React from 'react';
import { Card, Form, Select, Table, Button, Space, Modal, Tag, DatePicker } from 'antd';
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
import { formatDate } from '@/utils/datetime';
import type { DutySchedule, DutyPool, SimpleUser } from '@/lib/dutyApi';
import { createSorter } from '@/utils/tableHelpers';

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
  onSearch: (values: Record<string, unknown>) => void;
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

  // 初始化默认值：只显示未过期的排班
  React.useEffect(() => {
    searchForm.setFieldsValue({ expired: 0 });
  }, []);

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
      sorter: createSorter<DutySchedule>('scheduleDate', 'date'),
      render: (date: string) => formatDate(date),
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
      sorter: createSorter<DutySchedule>('poolName', 'string'),
    },
    {
      title: '值班人员',
      dataIndex: ['user', 'nickname'],
      key: 'userName',
      width: 100,
      sorter: createSorter<DutySchedule>('userName', 'string'),
      render: (nickname: string, record: DutySchedule) =>
        nickname || record.user?.username || '-',
    },
    {
      title: '值班类型',
      dataIndex: 'dutyType',
      key: 'dutyType',
      width: 100,
      sorter: createSorter<DutySchedule>('dutyType', 'string'),
      render: (type: string) => <Tag color={getDutyTypeColor(type)}>{getDutyTypeText(type)}</Tag>,
    },
    {
      title: '过期状态',
      key: 'expired',
      width: 100,
      render: (_: unknown, record: DutySchedule) => {
        const isExpired = dayjs(record.scheduleDate).isBefore(dayjs().startOf('day'));
        return isExpired ? (
          <Tag color="red">已过期</Tag>
        ) : (
          <Tag color="green">未过期</Tag>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      sorter: createSorter<DutySchedule>('status', 'number'),
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
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => {
            Modal.confirm({
              title: '确定要删除这条排班吗？',
              okText: '确定',
              cancelText: '取消',
              okButtonProps: { danger: true },
              onOk: () => onDelete(record.id),
            });
          }}
        >
          删除
        </Button>
      ),
    },
  ];

  return (
    <Card variant="borderless">
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="poolId">
          <Select placeholder="值班池" allowClear style={{ width: 140 }} onSearch={() => {}}>
            {pools.map((pool) => (
              <Select.Option key={pool.id} value={pool.id}>{pool.poolName}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="userId">
          <Select placeholder="值班人员" allowClear style={{ width: 120 }} onSearch={() => {}}>
            {users.map((user) => (
              <Select.Option key={user.id} value={user.id}>{user.nickname || user.username}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="dutyType">
          <Select placeholder="值班类型" allowClear style={{ width: 120 }} onSearch={() => {}}>
            <Select.Option value="weekday">工作日</Select.Option>
            <Select.Option value="weekend">周末</Select.Option>
            <Select.Option value="holiday">节假日</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item name="expired">
          <Select placeholder="排班状态" allowClear style={{ width: 120 }} onSearch={() => {}}>
            <Select.Option value={0}>未过期</Select.Option>
            <Select.Option value={1}>已过期</Select.Option>
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
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => {
                Modal.confirm({
                  title: '确定要批量删除选中的排班记录吗？',
                  okText: '确定',
                  cancelText: '取消',
                  okButtonProps: { danger: true },
                  onOk: onBatchDelete,
                });
              }}
            >
              批量删除 ({selectedRowKeys.length})
            </Button>
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

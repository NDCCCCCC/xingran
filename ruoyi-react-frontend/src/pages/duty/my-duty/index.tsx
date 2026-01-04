import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Tag,
  Table,
  Form,
  Select,
  DatePicker,
  Alert,
  Empty,
  Spin,
  Typography,
  Space,
  Button,
} from 'antd';
import {
  CalendarOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CoffeeOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  getMyDutyStats,
  getDutyScheduleList,
  type MyDutyStats,
  type DutySchedule,
  type DutyScheduleListRequest,
} from '@/lib/dutyApi';
import { useAuthStore } from '@/store/authStore';

const { RangePicker } = DatePicker;
const { Text } = Typography;
const { Option } = Select;

const MyDutyPage: React.FC = () => {
  const [form] = Form.useForm();
  const { user } = useAuthStore();

  // 统计数据
  const [stats, setStats] = useState<MyDutyStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);

  // 值班记录列表
  const [schedules, setSchedules] = useState<DutySchedule[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);

  // 加载统计数据
  const loadStats = async () => {
    setStatsLoading(true);
    try {
      const data = await getMyDutyStats();
      setStats(data.data);
    } catch (error) {
      console.error('加载值班统计失败', error);
    } finally {
      setStatsLoading(false);
    }
  };

  // 加载值班记录列表
  const loadSchedules = async (page?: number, size?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getDutyScheduleList({
        current: page ?? current,
        pageSize: size ?? pageSize,
        userId: user?.id,
        startDate: values.dateRange?.[0]?.format('YYYY-MM-DD'),
        endDate: values.dateRange?.[1]?.format('YYYY-MM-DD'),
        dutyType: values.dutyType,
      });
      setSchedules(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.page_size ?? 10);
    } catch (error) {
      console.error('加载值班记录失败', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStats();
    loadSchedules(1, 10);
  }, []);

  // 搜索
  const handleSearch = () => {
    loadSchedules(1, 10);
  };

  // 重置
  const handleReset = () => {
    form.resetFields();
    loadSchedules(1, 10);
  };

  // 分页变化
  const handleTableChange = (pagination: TablePaginationConfig) => {
    loadSchedules(pagination.current, pagination.pageSize);
  };

  // 获取值班类型颜色
  const getDutyTypeColor = (type: string) => {
    switch (type) {
      case 'weekday': return 'blue';
      case 'weekend': return 'orange';
      case 'holiday': return 'red';
      default: return 'default';
    }
  };

  // 获取值班类型文本
  const getDutyTypeText = (type: string) => {
    switch (type) {
      case 'weekday': return '工作日';
      case 'weekend': return '周末';
      case 'holiday': return '节假日';
      default: return type;
    }
  };

  // 获取状态标签
  const getStatusTag = (status: number) => {
    if (status === 0) return <Tag color="green">正常</Tag>;
    if (status === 1) return <Tag color="orange">已调换</Tag>;
    if (status === 2) return <Tag color="red">已取消</Tag>;
    return <Tag>未知</Tag>;
  };

  // 待办工单表格列定义
  const workOrderColumns: ColumnsType<any> = [
    {
      title: '工单编号',
      dataIndex: 'orderNo',
      key: 'orderNo',
      width: 150,
    },
    {
      title: '工单标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: number) => {
        const priorityMap: Record<number, { text: string; color: string }> = {
          1: { text: '紧急', color: 'red' },
          2: { text: '高', color: 'orange' },
          3: { text: '中', color: 'blue' },
          4: { text: '低', color: 'default' },
        };
        const config = priorityMap[priority] || priorityMap[4];
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          pending: { text: '待处理', color: 'orange' },
          processing: { text: '处理中', color: 'blue' },
          completed: { text: '已完成', color: 'green' },
        };
        const config = statusMap[status] || statusMap.pending;
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
  ];

  // 值班记录表格列定义
  const scheduleColumns: ColumnsType<DutySchedule> = [
    {
      title: '值班日期',
      dataIndex: 'scheduleDate',
      key: 'scheduleDate',
      width: 110,
      render: (date: string) => dayjs(date).format('MM-DD'),
    },
    {
      title: '星期',
      key: 'weekday',
      width: 70,
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
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'dutyType',
      key: 'dutyType',
      width: 80,
      render: (type: string) => (
        <Tag color={getDutyTypeColor(type)}>{getDutyTypeText(type)}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => getStatusTag(status),
    },
  ];

  // 模拟待办工单数据
  const mockWorkOrders = [
    {
      id: '1',
      orderNo: 'WO202501040001',
      title: '服务器CPU使用率异常告警',
      priority: 1,
      status: 'pending',
      createdAt: '2025-01-04 09:30',
    },
    {
      id: '2',
      orderNo: 'WO202501040002',
      title: '数据库备份任务检查',
      priority: 3,
      status: 'processing',
      createdAt: '2025-01-04 10:15',
    },
    {
      id: '3',
      orderNo: 'WO202501040003',
      title: '网络设备固件升级',
      priority: 2,
      status: 'pending',
      createdAt: '2025-01-04 11:00',
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      {/* 统计卡片区域 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '24px' }}>
        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="今日状态"
              value={stats?.isOnDutyToday ? "值班中" : "休息"}
              prefix={stats?.isOnDutyToday ? <ClockCircleOutlined /> : <CoffeeOutlined />}
              valueStyle={{
                color: stats?.isOnDutyToday ? '#3f8600' : '#8c8c8c',
                fontSize: '20px'
              }}
            />
          </Card>
        </Col>

        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="本月值班"
              value={stats?.thisMonthCount || 0}
              suffix="次"
              prefix={<CalendarOutlined />}
              valueStyle={{ color: '#1890ff', fontSize: '20px' }}
            />
          </Card>
        </Col>

        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="累计值班"
              value={stats?.totalCount || 0}
              suffix="次"
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#722ed1', fontSize: '20px' }}
            />
          </Card>
        </Col>

        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <div>
              <Text type="secondary" style={{ fontSize: '14px' }}>下次值班</Text>
              <div style={{ marginTop: '8px' }}>
                {stats?.nextDutyDate ? (
                  <Space>
                    <span style={{ fontSize: '18px', fontWeight: 500 }}>
                      {dayjs(stats.nextDutyDate).format('MM-DD')}
                    </span>
                    {stats.nextDutyPoolName && (
                      <Tag color="blue">{stats.nextDutyPoolName}</Tag>
                    )}
                  </Space>
                ) : (
                  <span style={{ color: '#8c8c8c' }}>暂无排班</span>
                )}
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 待办工单和值班记录 - 左右布局 */}
      <Row gutter={[16, 16]}>
        {/* 左侧：待办工单 - 70% */}
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <UnorderedListOutlined />
                <span>待办工单</span>
                <Tag color="purple">即将推出</Tag>
              </Space>
            }
            extra={<Tag color="orange">3 条待处理</Tag>}
          >
            <Alert
              message="工单功能开发中"
              description="工单模块即将上线，上线后将支持查看和处理值班期间的相关工单。"
              type="info"
              showIcon
              style={{ marginBottom: '16px' }}
            />

            <Table
              rowKey="id"
              columns={workOrderColumns}
              dataSource={mockWorkOrders}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        {/* 右侧：值班记录 - 30% */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <ClockCircleOutlined />
                <span>值班记录</span>
              </Space>
            }
          >
            <div style={{ marginBottom: '12px' }}>
              <Form form={form} layout="inline">
                <Form.Item name="dutyType" style={{ marginBottom: 0 }}>
                  <Select
                    placeholder="全部类型"
                    allowClear
                    style={{ width: 100 }}
                    onChange={handleSearch}
                  >
                    <Option value="weekday">工作日</Option>
                    <Option value="weekend">周末</Option>
                    <Option value="holiday">节假日</Option>
                  </Select>
                </Form.Item>
              </Form>
            </div>

            <Table
              rowKey="id"
              columns={scheduleColumns}
              dataSource={schedules}
              loading={loading}
              pagination={{
                current,
                pageSize: 10,
                total,
                size: 'small',
                showSizeChanger: false,
                showTotal: (t) => `共 ${t} 条`,
              }}
              onChange={handleTableChange}
              size="small"
              scroll={{ y: 400 }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default MyDutyPage;

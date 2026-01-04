import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Calendar,
  Tag,
  Table,
  Form,
  Select,
  DatePicker,
  Badge,
  Alert,
  Collapse,
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
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  getMyDutyStats,
  getDutyScheduleList,
  getMonthlyDutySchedule,
  type MyDutyStats,
  type DutySchedule,
  type MonthlyDutyMember,
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

  // 日历数据
  const [calendarData, setCalendarData] = useState<Record<string, MonthlyDutyMember[]>>({});
  const [selectedMonth, setSelectedMonth] = useState<Dayjs>(dayjs());
  const [calendarLoading, setCalendarLoading] = useState(false);

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

  // 加载日历数据
  const loadCalendarData = async (month: Dayjs) => {
    setCalendarLoading(true);
    try {
      const year = month.year();
      const monthNum = month.month() + 1;
      const result = await getMonthlyDutySchedule(year, monthNum);
      // 只保留当前用户的值班记录
      const filteredData: Record<string, MonthlyDutyMember[]> = {};
      Object.entries(result.data || {}).forEach(([date, members]) => {
        const userMembers = members.filter(m => m.userId === user?.id);
        if (userMembers.length > 0) {
          filteredData[date] = userMembers;
        }
      });
      setCalendarData(filteredData);
    } catch (error) {
      console.error('加载日历数据失败', error);
    } finally {
      setCalendarLoading(false);
    }
  };

  useEffect(() => {
    loadStats();
    loadSchedules(1, 10);
    loadCalendarData(selectedMonth);
  }, []);

  // 月份变化
  const handleMonthChange = (date: Dayjs) => {
    setSelectedMonth(date);
    loadCalendarData(date);
  };

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

  // 日历单元格渲染
  const dateCellRender = (date: Dayjs) => {
    const dateStr = date.format('YYYY-MM-DD');
    const duties = calendarData[dateStr] || [];

    if (duties.length === 0) return null;

    return (
      <div style={{ fontSize: '12px' }}>
        {duties.map((duty, index) => (
          <Badge
            key={index}
            color={getDutyTypeColor(duty.dutyType)}
            text={duty.poolName}
            style={{ marginBottom: '2px' }}
          />
        ))}
      </div>
    );
  };

  // 表格列定义
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
      width: 150,
    },
    {
      title: '值班类型',
      dataIndex: 'dutyType',
      key: 'dutyType',
      width: 100,
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

  return (
    <div style={{ padding: '24px' }}>
      {/* 统计卡片区域 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '24px' }}>
        <Col xs={24} sm={12} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="今日状态"
              value={stats?.isOnDutyToday ? "值班中" : "休息"}
              prefix={stats?.isOnDutyToday ? <ClockCircleOutlined /> : <CoffeeOutlined />}
              valueStyle={{
                color: stats?.isOnDutyToday ? '#3f8600' : '#8c8c8c',
                fontSize: '24px'
              }}
            />
            {stats?.isOnDutyToday && stats.todayDutyRecords && stats.todayDutyRecords.length > 0 && (
              <div style={{ marginTop: '8px' }}>
                {stats.todayDutyRecords.map((record, index) => (
                  <Tag key={index} color={getDutyTypeColor(record.dutyType)} style={{ marginBottom: '4px' }}>
                    {record.poolName}
                  </Tag>
                ))}
              </div>
            )}
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="本月值班"
              value={stats?.thisMonthCount || 0}
              suffix="次"
              prefix={<CalendarOutlined />}
              valueStyle={{ color: '#1890ff', fontSize: '24px' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="累计值班"
              value={stats?.totalCount || 0}
              suffix="次"
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#722ed1', fontSize: '24px' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={statsLoading}>
            <div>
              <div style={{ fontSize: '14px', color: 'rgba(0,0,0,0.45)', marginBottom: '8px' }}>下次值班</div>
              <div>
                {stats?.nextDutyDate ? (
                  <Space direction="vertical" size={0}>
                    <div style={{ fontSize: '20px', fontWeight: 500 }}>
                      {dayjs(stats.nextDutyDate).format('YYYY-MM-DD')}
                    </div>
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

      {/* 日历视图 */}
      <Card
        title={<Space><CalendarOutlined />值班日历</Space>}
        style={{ marginBottom: '24px' }}
      >
        <div style={{ marginBottom: '16px' }}>
          <DatePicker.MonthPicker
            value={selectedMonth}
            onChange={(date) => date && handleMonthChange(date)}
            format="YYYY年MM月"
            style={{ marginRight: '8px' }}
          />
          <Text type="secondary">
            选择月份查看该月的值班安排
          </Text>
        </div>

        <Spin spinning={calendarLoading}>
          <Calendar
            value={selectedMonth}
            fullscreen={false}
            dateCellRender={dateCellRender}
          />
        </Spin>

        <div style={{ marginTop: '16px', display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
          <Badge color="blue" text={<Text type="secondary">工作日</Text>} />
          <Badge color="orange" text={<Text type="secondary">周末</Text>} />
          <Badge color="red" text={<Text type="secondary">节假日</Text>} />
        </div>
      </Card>

      {/* 值班记录列表 */}
      <Card title={<Space><ClockCircleOutlined />值班记录</Space>}>
        <div style={{ marginBottom: '16px' }}>
          <Form form={form} layout="inline">
            <Form.Item name="dateRange" label="日期范围">
              <RangePicker />
            </Form.Item>
            <Form.Item name="dutyType" label="值班类型">
              <Select placeholder="请选择" allowClear style={{ width: 120 }}>
                <Option value="weekday">工作日</Option>
                <Option value="weekend">周末</Option>
                <Option value="holiday">节假日</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" onClick={handleSearch}>
                  查询
                </Button>
                <Button onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={schedules}
          loading={loading}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
          onChange={handleTableChange}
        />
      </Card>

      {/* 工单扩展区域（预留） */}
      <Collapse
        style={{ marginTop: '24px' }}
        items={[
          {
            key: 'work-orders',
            label: (
              <Space>
                <span>值班期间工单</span>
                <Tag color="purple">即将推出</Tag>
              </Space>
            ),
            children: (
              <Alert
                message="工单模块即将上线"
                description="工单功能开发中，上线后将支持查看值班期间处理的工单记录、工单统计等信息。"
                type="info"
                showIcon
              />
            ),
          },
        ]}
      />
    </div>
  );
};

export default MyDutyPage;

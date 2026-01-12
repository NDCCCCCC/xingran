import { useState, useEffect } from 'react';
import {
  Button,
  Form,
  Select,
  Table,
  Modal,
  Input,
  message,
  Popconfirm,
  Card,
  Space,
  Tag,
  DatePicker,
  Alert,
  Switch,
  Badge,
  Row,
  Col,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  CalendarOutlined,
  SwapOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import {
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import {
  getDutyScheduleList,
  generateSchedule,
  getTodayDuty,
  swapDuty,
  manualDuty,
  deleteDutySchedule,
  batchDeleteDutySchedules,
  getDutyPoolList,
  getUserList,
  getMonthlyDutySchedule,
  type DutySchedule,
  type DutyPool,
  type SimpleUser,
  type MonthlyDutyMember,
} from '@/lib/dutyApi';
import type { Dayjs } from 'dayjs';

const { Text } = Typography;

const { RangePicker } = DatePicker;
const { Option } = Select;

import type { FC } from 'react';

const DutySchedulePage: FC = () => {
  const [form] = Form.useForm();
  const [generateForm] = Form.useForm();
  const [swapForm] = Form.useForm();
  const [manualForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<DutySchedule[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [swapModalVisible, setSwapModalVisible] = useState(false);
  const [manualModalVisible, setManualModalVisible] = useState(false);

  const [allSchedules, setAllSchedules] = useState<DutySchedule[]>([]); // 用于调班
  const [currentWeekStart, setCurrentWeekStart] = useState(dayjs().startOf('week'));
  const [weeklyDutyData, setWeeklyDutyData] = useState<Record<string, MonthlyDutyMember[]>>({});

  // 值班池和用户数据
  const [pools, setPools] = useState<DutyPool[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);

  const fetchList = async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getDutyScheduleList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
        poolId: values.poolId,
        userId: values.userId,
        startDate: values.dateRange?.[0]?.format('YYYY-MM-DD'),
        endDate: values.dateRange?.[1]?.format('YYYY-MM-DD'),
        dutyType: values.dutyType,
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.page_size ?? 10);
    } catch (error) {
      message.error('获取排班列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchTodayDuty = async () => {
    try {
      await getTodayDuty();
      // 暂时不使用今日值班数据
    } catch (error) {
      console.error('获取今日值班失败', error);
    }
  };

  const fetchAllSchedules = async () => {
    try {
      const result = await getDutyScheduleList({ current: 1, pageSize: 1000 });
      setAllSchedules(result.data?.list ?? []);
    } catch (error) {
      console.error('获取所有排班失败', error);
    }
  };

  // 获取值班池列表（只获取启用的）
  const fetchPools = async () => {
    try {
      const result = await getDutyPoolList({ current: 1, pageSize: 1000, status: 0 });
      setPools(result.data?.list || []);
    } catch (error) {
      console.error('获取值班池列表失败', error);
    }
  };

  // 获取用户列表（只获取启用的）
  const fetchUsers = async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取用户列表失败', error);
    }
  };

  // 获取周度值班数据
  const fetchWeeklyDuty = async (weekStart: Dayjs) => {
    try {
      const weekEnd = weekStart.endOf('week');
      const year = weekStart.year();
      const monthStart = weekStart.month() + 1;
      const monthEnd = weekEnd.month() + 1;

      // 如果周跨月，需要获取两个月的数据
      if (monthStart === monthEnd) {
        const result = await getMonthlyDutySchedule(year, monthStart);
        setWeeklyDutyData(result.data || {});
      } else {
        // 跨月情况，获取两个月的数据并合并
        const [result1, result2] = await Promise.all([
          getMonthlyDutySchedule(year, monthStart),
          getMonthlyDutySchedule(year, monthEnd),
        ]);
        setWeeklyDutyData({
          ...(result1.data || {}),
          ...(result2.data || {}),
        });
      }
    } catch (error) {
      console.error('获取周度值班失败', error);
    }
  };

  // 周视图切换
  const handlePrevWeek = () => {
    const newWeekStart = currentWeekStart.subtract(1, 'week');
    setCurrentWeekStart(newWeekStart);
    fetchWeeklyDuty(newWeekStart);
  };

  const handleNextWeek = () => {
    const newWeekStart = currentWeekStart.add(1, 'week');
    setCurrentWeekStart(newWeekStart);
    fetchWeeklyDuty(newWeekStart);
  };

  const handleTodayWeek = () => {
    const todayWeekStart = dayjs().startOf('week');
    setCurrentWeekStart(todayWeekStart);
    fetchWeeklyDuty(todayWeekStart);
  };

  // 获取周范围显示文本
  const getWeekRangeText = () => {
    const weekEnd = currentWeekStart.endOf('week');
    return `${currentWeekStart.format('YYYY年MM月DD日')} - ${weekEnd.format('YYYY年MM月DD日')}`;
  };

  // 获取一周的日期数组
  const getWeekDays = () => {
    const days = [];
    for (let i = 0; i < 7; i++) {
      days.push(currentWeekStart.add(i, 'day'));
    }
    return days;
  };

  // 获取星期几文本
  const getWeekdayText = (day: Dayjs) => {
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    return weekdays[day.day()];
  };

  // 获取值班类型颜色
  const getDutyTypeBadgeColor = (type: string) => {
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

  useEffect(() => {
    fetchList(1, 10);
    fetchTodayDuty();
    fetchAllSchedules();
    fetchPools();
    fetchUsers();
    fetchWeeklyDuty(currentWeekStart);
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    fetchList(pagination.current, pagination.pageSize);
  };

  const handleSearch = () => {
    fetchList(1, 10);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, 10);
  };

  const handleGenerate = async () => {
    try {
      const values = await generateForm.validateFields();
      const result = await generateSchedule({
        poolId: values.poolId,
        startDate: values.dateRange[0].format('YYYY-MM-DD'),
        endDate: values.dateRange[1].format('YYYY-MM-DD'),
        dutyType: values.dutyType,
        clearExists: values.clearExists || false,
      });
      message.success(`成功生成 ${result.data?.count || 0} 条排班记录`);
      setGenerateModalVisible(false);
      fetchList();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('生成排班失败');
    }
  };

  const handleSwap = async () => {
    try {
      const values = await swapForm.validateFields();
      await swapDuty({
        fromScheduleId: values.fromScheduleId,
        toScheduleId: values.toScheduleId,
        reason: values.reason,
      });
      message.success('调班成功');
      setSwapModalVisible(false);
      fetchList();
      fetchTodayDuty();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('调班失败');
    }
  };

  const handleManual = async () => {
    try {
      const values = await manualForm.validateFields();
      await manualDuty({
        poolId: values.poolId,
        dutyDate: values.dutyDate.format('YYYY-MM-DD'),
        dutyType: values.dutyType,
        userIds: values.userIds,
        reason: values.reason,
      });
      message.success('手动排班成功');
      setManualModalVisible(false);
      fetchList();
      fetchTodayDuty();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('手动排班失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteDutySchedule(id);
      message.success('删除成功');
      // 如果当前页只有一条数据且不是第一页，跳转到前一页
      const newPage = dataSource.length === 1 && current > 1 ? current - 1 : current;
      fetchList(newPage);
      fetchTodayDuty();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的排班记录');
      return;
    }
    try {
      await batchDeleteDutySchedules(selectedRowKeys as string[]);
      message.success(`成功删除 ${selectedRowKeys.length} 条排班记录`);
      setSelectedRowKeys([]);
      // 如果删除后当前页没有数据了且不是第一页，跳转到前一页
      const newPage = dataSource.length === selectedRowKeys.length && current > 1 ? current - 1 : current;
      fetchList(newPage);
      fetchTodayDuty();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

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
      title: '是否手动',
      dataIndex: 'isManual',
      key: 'isManual',
      width: 80,
      render: (isManual: boolean) => (isManual ? <Tag color="purple">手动</Tag> : <Tag>自动</Tag>),
    },
    {
      title: '调班原因',
      dataIndex: 'swapReason',
      key: 'swapReason',
      width: 150,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: DutySchedule) => (
        <Popconfirm title="确定要删除这条排班吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="p-6">
      {/* 今日值班展示 - 暂时隐藏 */}
      {/* <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>今日值班 - {dayjs().format('YYYY年MM月DD日')}</span>
          </Space>
        }
        className="mb-4"
        extra={
          <Button type="primary" icon={<ReloadOutlined />} onClick={fetchTodayDuty}>
            刷新
          </Button>
        }
      >
        {todayDuty.length > 0 ? (
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            {todayDuty.map((item, index) => (
              <Card key={index} size="small">
                <Space>
                  <Avatar icon={<UserOutlined />} />
                  <Space orientation="vertical" size={0}>
                    <Text strong>{item.nickName || item.username}</Text>
                    <Text type="secondary">用户名: {item.username}</Text>
                  </Space>
                </Space>
              </Card>
            ))}
          </Space>
        ) : (
          <Alert title="今日暂无值班人员" type="info" showIcon />
        )}
      </Card> */}

      {/* 值班周视图 */}
      <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>值班周历</span>
          </Space>
        }
        className="mb-4"
        extra={
          <Space>
            <Button icon={<LeftOutlined />} onClick={handlePrevWeek}>
              上一周
            </Button>
            <Button onClick={handleTodayWeek}>
              本周
            </Button>
            <Button icon={<RightOutlined />} onClick={handleNextWeek}>
              下一周
            </Button>
          </Space>
        }
      >
        <div className="mb-3">
          <Text strong style={{ fontSize: '16px' }}>{getWeekRangeText()}</Text>
        </div>
        <Row gutter={[8, 8]}>
          {getWeekDays().map((day) => {
            const dateStr = day.format('YYYY-MM-DD');
            const members = weeklyDutyData[dateStr] || [];
            const isToday = day.format('YYYY-MM-DD') === dayjs().format('YYYY-MM-DD');

            return (
              <Col key={dateStr} span={3}>
                <Card
                  size="small"
                  style={{
                    height: '120px',
                    borderColor: isToday ? '#1890ff' : undefined,
                    backgroundColor: isToday ? '#e6f7ff' : undefined,
                  }}
                  styles={{ body: { padding: '8px', height: '100%', overflow: 'auto' } }}
                >
                  <div style={{ marginBottom: '4px' }}>
                    <Text
                      strong
                      style={{
                        fontSize: '12px',
                        color: isToday ? '#1890ff' : undefined,
                      }}
                    >
                      {getWeekdayText(day)}
                    </Text>
                    <br />
                    <Text style={{ fontSize: '11px', color: '#999' }}>
                      {day.format('MM-DD')}
                    </Text>
                  </div>
                  <Space orientation="vertical" size="small" style={{ width: '100%' }}>
                    {members.length > 0 ? (
                      members.map((member, index) => (
                        <Badge
                          key={index}
                          color={getDutyTypeBadgeColor(member.dutyType)}
                          text={
                            <span
                              style={{
                                fontSize: '11px',
                                whiteSpace: 'nowrap',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                display: 'block',
                                maxWidth: '80px',
                              }}
                            >
                              {member.username}
                            </span>
                          }
                        />
                      ))
                    ) : (
                      <Text style={{ fontSize: '11px', color: '#ccc' }}>无值班</Text>
                    )}
                  </Space>
                </Card>
              </Col>
            );
          })}
        </Row>
        <div className="mt-3 flex gap-4">
          <Space>
            <Badge color="blue" text={<Text type="secondary">工作日</Text>} />
            <Badge color="orange" text={<Text type="secondary">周末</Text>} />
            <Badge color="red" text={<Text type="secondary">节假日</Text>} />
          </Space>
        </div>
      </Card>

      {/* 排班管理 */}
      <Card title="排班管理">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="poolId" label="值班池">
              <Select placeholder="请选择值班池" allowClear style={{ width: 150 }}>
                {pools.map((pool) => (
                  <Option key={pool.id} value={pool.id}>
                    {pool.poolName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="userId" label="值班人员">
              <Select placeholder="请选择值班人员" allowClear style={{ width: 120 }}>
                {users.map((user) => (
                  <Option key={user.id} value={user.id}>
                    {user.nickName || user.username}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="dutyType" label="值班类型">
              <Select placeholder="请选择值班类型" allowClear style={{ width: 120 }}>
                <Option value="weekday">工作日</Option>
                <Option value="weekend">周末</Option>
                <Option value="holiday">节假日</Option>
              </Select>
            </Form.Item>
            <Form.Item name="dateRange" label="日期范围">
              <RangePicker style={{ width: 240 }} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setGenerateModalVisible(true)}>
              生成排班
            </Button>
            <Button icon={<SwapOutlined />} onClick={() => setSwapModalVisible(true)}>
              调班
            </Button>
            <Button icon={<EditOutlined />} onClick={() => setManualModalVisible(true)}>
              手动排班
            </Button>
            {selectedRowKeys.length > 0 && (
              <Popconfirm
                title="确定要批量删除选中的排班记录吗？"
                onConfirm={handleBatchDelete}
                okText="确定"
                cancelText="取消"
              >
                <Button danger icon={<DeleteOutlined />}>
                  批量删除 ({selectedRowKeys.length})
                </Button>
              </Popconfirm>
            )}
          </Space>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1200 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys as string[]),
          }}
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

      {/* 生成排班弹窗 */}
      <Modal
        title="生成排班"
        open={generateModalVisible}
        onOk={handleGenerate}
        onCancel={() => setGenerateModalVisible(false)}
        destroyOnHidden
      >
        <Form form={generateForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolId"
            label="值班池"
            rules={[{ required: true, message: '请选择值班池' }]}
          >
            <Select placeholder="请选择值班池">
              {pools.map((pool) => (
                <Option key={pool.id} value={pool.id}>
                  {pool.poolName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="dateRange"
            label="日期范围"
            rules={[{ required: true, message: '请选择日期范围' }]}
          >
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="dutyType"
            label="值班类型"
            rules={[{ required: true, message: '请选择值班类型' }]}
            initialValue="weekday"
          >
            <Select placeholder="请选择值班类型">
              <Option value="weekday">工作日值班</Option>
              <Option value="weekend">周末值班</Option>
              <Option value="holiday">节假日值班</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="clearExists"
            label="清除已有排班"
            valuePropName="checked"
            initialValue={false}
          >
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>

          <Alert
            title="提示"
            description="系统将根据值班池中的成员顺序，按轮询方式自动生成排班记录。只对符合所选值班类型的日期进行排班。"
            type="info"
            showIcon
          />
        </Form>
      </Modal>

      {/* 调班弹窗 */}
      <Modal
        title="调班"
        open={swapModalVisible}
        onOk={handleSwap}
        onCancel={() => setSwapModalVisible(false)}
        destroyOnHidden
        width={600}
      >
        <Form form={swapForm} layout="vertical" preserve={false}>
          <Form.Item
            name="fromScheduleId"
            label="原排班记录"
            rules={[{ required: true, message: '请选择排班记录' }]}
          >
            <Select placeholder="请选择原排班记录" showSearch optionFilterProp="label">
              {allSchedules.map((s) => (
                <Option
                  key={s.id}
                  value={s.id}
                  label={`${dayjs(s.scheduleDate).format('YYYY-MM-DD')} - ${s.user?.nickName || s.user?.username}`}
                >
                  {dayjs(s.scheduleDate).format('YYYY-MM-DD')} ({getDutyTypeText(s.dutyType)}) - {s.user?.nickName || s.user?.username}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="toScheduleId"
            label="目标排班记录"
            rules={[{ required: true, message: '请选择排班记录' }]}
          >
            <Select placeholder="请选择目标排班记录" showSearch optionFilterProp="label">
              {allSchedules.map((s) => (
                <Option
                  key={s.id}
                  value={s.id}
                  label={`${dayjs(s.scheduleDate).format('YYYY-MM-DD')} - ${s.user?.nickName || s.user?.username}`}
                >
                  {dayjs(s.scheduleDate).format('YYYY-MM-DD')} ({getDutyTypeText(s.dutyType)}) - {s.user?.nickName || s.user?.username}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="reason" label="调班原因">
            <Input.TextArea rows={3} placeholder="请输入调班原因" />
          </Form.Item>

          <Alert
            title="提示"
            description="选择两条排班记录进行互换，两位值班人员将交换各自的值班日期。"
            type="info"
            showIcon
          />
        </Form>
      </Modal>

      {/* 手动排班弹窗 */}
      <Modal
        title="手动排班"
        open={manualModalVisible}
        onOk={handleManual}
        onCancel={() => setManualModalVisible(false)}
        destroyOnHidden
      >
        <Form form={manualForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolId"
            label="值班池"
            rules={[{ required: true, message: '请选择值班池' }]}
          >
            <Select placeholder="请选择值班池">
              {pools.map((pool) => (
                <Option key={pool.id} value={pool.id}>
                  {pool.poolName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="dutyDate"
            label="值班日期"
            rules={[{ required: true, message: '请选择值班日期' }]}
          >
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="dutyType"
            label="值班类型"
            rules={[{ required: true, message: '请选择值班类型' }]}
          >
            <Select placeholder="请选择值班类型">
              <Option value="weekday">工作日</Option>
              <Option value="weekend">周末</Option>
              <Option value="holiday">节假日</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="userIds"
            label="值班人员"
            rules={[{ required: true, message: '请选择值班人员' }]}
          >
            <Select mode="multiple" placeholder="请选择值班人员" optionLabelProp="label">
              {users.map((user) => (
                <Option key={user.id} value={user.id} label={user.nickName || user.username}>
                  {user.username} - {user.nickName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="reason" label="备注">
            <Input.TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DutySchedulePage;

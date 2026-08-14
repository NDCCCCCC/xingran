import { useState, useEffect } from "react";
import {
  Card,
  Row,
  Col,
  Statistic,
  Tag,
  Table,
  Form,
  Select,
  Empty,
  Typography,
  Space,
} from "antd";
import {
  CalendarOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CoffeeOutlined,
  UnorderedListOutlined,
} from "@ant-design/icons";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import dayjs from "dayjs";
import { formatDateTime } from "@/utils/datetime";
import { createSorter } from "@/utils/tableHelpers";
import {
  getMyDutyStats,
  getDutyScheduleList,
  type MyDutyStats,
  type DutySchedule,
} from "@/lib/dutyApi";
import {
  getMyPendingWorkOrders,
  type WorkOrder,
  WorkOrderStatus,
  WorkOrderPriority,
} from "@/lib/workorderApi";
import { useAuthStore } from "@/store/authStore";
import { usePagination } from "@/hooks/usePagination";

const { Text } = Typography;
const { Option } = Select;

import type { FC } from "react";

const MyDutyPage: FC = () => {
  const [form] = Form.useForm();
  const { user } = useAuthStore();

  // 统计数据
  const [stats, setStats] = useState<MyDutyStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 值班记录列表
  const [schedules, setSchedules] = useState<DutySchedule[]>([]);
  const [loading, setLoading] = useState(false);

  // 待办工单列表
  const [workOrders, setWorkOrders] = useState<WorkOrder[]>([]);
  const [workOrdersTotal, setWorkOrdersTotal] = useState(0);
  const [workOrdersLoading, setWorkOrdersLoading] = useState(false);

  // 加载统计数据
  const loadStats = async () => {
    setStatsLoading(true);
    try {
      const data = await getMyDutyStats();
      setStats(data.data || null);
    } catch (error) {
      console.error("加载值班统计失败", error);
    } finally {
      setStatsLoading(false);
    }
  };

  // 加载待办工单
  const loadWorkOrders = async () => {
    setWorkOrdersLoading(true);
    try {
      const data = await getMyPendingWorkOrders({ limit: 5 });
      setWorkOrders(data.data?.list ?? []);
      setWorkOrdersTotal(data.data?.total ?? 0);
    } catch (error) {
      console.error("加载待办工单失败", error);
      // 即使失败也设置为空数组，避免显示undefined
      setWorkOrders([]);
      setWorkOrdersTotal(0);
    } finally {
      setWorkOrdersLoading(false);
    }
  };

  // 加载值班记录列表
  const loadSchedules = async (page?: number, size?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getDutyScheduleList({
        current: page ?? paginationProps.current,
        pageSize: size ?? paginationProps.pageSize,
        userId: user?.id,
        startDate: values.dateRange?.[0]?.format("YYYY-MM-DD"),
        endDate: values.dateRange?.[1]?.format("YYYY-MM-DD"),
        dutyType: values.dutyType,
      });
      setSchedules(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
    } catch (error) {
      console.error("加载值班记录失败", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStats();
    loadWorkOrders();
    loadSchedules();
  }, [paginationProps.current, paginationProps.pageSize]);

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    loadSchedules();
  };

  // 分页变化
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setCurrent(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? 10);
    loadSchedules();
  };

  // 获取值班类型颜色
  const getDutyTypeColor = (type: string) => {
    switch (type) {
      case "weekday":
        return "blue";
      case "weekend":
        return "orange";
      case "holiday":
        return "red";
      default:
        return "default";
    }
  };

  // 获取值班类型文本
  const getDutyTypeText = (type: string) => {
    switch (type) {
      case "weekday":
        return "工作日";
      case "weekend":
        return "周末";
      case "holiday":
        return "节假日";
      default:
        return type;
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
  const workOrderColumns: ColumnsType<WorkOrder> = [
    {
      title: "工单编号",
      dataIndex: "workOrderNo",
      key: "workOrderNo",
      width: 150,
      ellipsis: true,
      sorter: createSorter<WorkOrder>("workOrderNo", "string"),
    },
    {
      title: "工单标题",
      dataIndex: "title",
      key: "title",
      ellipsis: true,
      sorter: createSorter<WorkOrder>("title", "string"),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      sorter: createSorter<WorkOrder>("priority", "string"),
      render: (priority: WorkOrderPriority) => {
        const priorityMap: Record<WorkOrderPriority, { text: string; color: string }> = {
          [WorkOrderPriority.Urgent]: { text: "紧急", color: "red" },
          [WorkOrderPriority.High]: { text: "高", color: "orange" },
          [WorkOrderPriority.Medium]: { text: "中", color: "blue" },
          [WorkOrderPriority.Low]: { text: "低", color: "default" },
        };
        const config = priorityMap[priority] || priorityMap[WorkOrderPriority.Low];
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: createSorter<WorkOrder>("status", "string"),
      render: (status: WorkOrderStatus) => {
        const statusMap: Record<WorkOrderStatus, { text: string; color: string }> = {
          [WorkOrderStatus.Pending]: { text: "待处理", color: "orange" },
          [WorkOrderStatus.Processing]: { text: "处理中", color: "blue" },
          [WorkOrderStatus.Completed]: { text: "已完成", color: "green" },
          [WorkOrderStatus.Closed]: { text: "已关闭", color: "default" },
          [WorkOrderStatus.Rejected]: { text: "已拒绝", color: "red" },
        };
        const config = statusMap[status] || statusMap[WorkOrderStatus.Pending];
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: createSorter<WorkOrder>("createdAt", "date"),
      render: (date: string) => formatDateTime(date, "YYYY-MM-DD HH:mm"),
    },
  ];

  // 值班记录表格列定义
  const scheduleColumns: ColumnsType<DutySchedule> = [
    {
      title: "值班日期",
      dataIndex: "scheduleDate",
      key: "scheduleDate",
      width: 110,
      sorter: createSorter<DutySchedule>("scheduleDate", "date"),
      render: (date: string) => formatDateTime(date, "MM-DD"),
    },
    {
      title: "星期",
      key: "weekday",
      width: 70,
      render: (_: unknown, record: DutySchedule) => {
        const weekday = dayjs(record.scheduleDate).day();
        const weekdays = ["日", "一", "二", "三", "四", "五", "六"];
        return `周${weekdays[weekday]}`;
      },
    },
    {
      title: "值班池",
      dataIndex: ["pool", "poolName"],
      key: "poolName",
      ellipsis: true,
      sorter: createSorter<DutySchedule>("poolName", "string"),
    },
    {
      title: "类型",
      dataIndex: "dutyType",
      key: "dutyType",
      width: 80,
      sorter: createSorter<DutySchedule>("dutyType", "string"),
      render: (type: string) => <Tag color={getDutyTypeColor(type)}>{getDutyTypeText(type)}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      sorter: createSorter<DutySchedule>("status", "number"),
      render: (status: number) => getStatusTag(status),
    },
  ];

  return (
    <div style={{ padding: "24px" }}>
      {/* 统计卡片区域 */}
      <Row gutter={[16, 16]} style={{ marginBottom: "24px" }}>
        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <Statistic
              title="今日状态"
              value={stats?.isOnDutyToday ? "值班中" : "休息"}
              prefix={stats?.isOnDutyToday ? <ClockCircleOutlined /> : <CoffeeOutlined />}
              styles={{
                content: {
                  color: stats?.isOnDutyToday ? "#3f8600" : "#8c8c8c",
                  fontSize: "20px",
                },
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
              styles={{ content: { color: "#1890ff", fontSize: "20px" } }}
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
              styles={{ content: { color: "#722ed1", fontSize: "20px" } }}
            />
          </Card>
        </Col>

        <Col xs={12} sm={6} lg={6}>
          <Card loading={statsLoading}>
            <div>
              <Text type="secondary" style={{ fontSize: "14px" }}>
                下次值班
              </Text>
              <div style={{ marginTop: "8px" }}>
                {stats?.nextDutyDate ? (
                  <Space>
                    <span style={{ fontSize: "18px", fontWeight: 500 }}>
                      {formatDateTime(stats.nextDutyDate, "MM-DD")}
                    </span>
                    {stats.nextDutyPoolName && <Tag color="blue">{stats.nextDutyPoolName}</Tag>}
                  </Space>
                ) : (
                  <span style={{ color: "#8c8c8c" }}>暂无排班</span>
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
              </Space>
            }
            extra={
              workOrdersTotal > 0 ? <Tag color="orange">{workOrdersTotal} 条待处理</Tag> : null
            }
          >
            {workOrders.length === 0 && !workOrdersLoading ? (
              <Empty
                description="暂无待办工单"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                style={{ margin: "40px 0" }}
              />
            ) : (
              <Table
                rowKey="id"
                columns={workOrderColumns}
                dataSource={workOrders}
                loading={workOrdersLoading}
                pagination={false}
                size="small"
              />
            )}
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
            <div style={{ marginBottom: "12px" }}>
              <Form form={form} layout="inline">
                <Form.Item name="dutyType" style={{ marginBottom: 0 }}>
                  <Select
                    placeholder="全部类型"
                    allowClear
                    style={{ width: 100 }}
                    onChange={handleSearch}
                    onSearch={() => {}}
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
                ...paginationProps,
                pageSize: 10,
                size: "small",
                showSizeChanger: false,
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

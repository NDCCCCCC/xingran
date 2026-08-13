/**
 * 值班排班管理页面
 * Duty Schedule Management Page
 */

import { useState, useEffect, type FC } from "react";
import {
  Button,
  Form,
  Select,
  Table,
  Modal,
  Input,
  App,
  Card,
  Space,
  Tag,
  DatePicker,
  Alert,
  Switch,
  Badge,
  Row,
  Col,
  Typography,
} from "antd";
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
} from "@ant-design/icons";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";

import type { DutySchedule, DutyPool, SimpleUser, MonthlyDutyMember } from "@/lib/dutyApi";

// 导入提取的模块
import { DUTY_TYPE_OPTIONS, DUTY_STATUS_CONFIG, getDutyTypeColor, getDutyTypeText } from "./constants";
import {
  formatDate,
  getWeekdayText,
  getWeekRangeText,
  getWeekDays,
  formatScheduleOptionLabel,
  formatScheduleOptionContent,
} from "./utils";
import { useScheduleData, useScheduleModals } from "./hooks";
import { usePagination } from "@/hooks/usePagination";
import { createSorter } from "@/utils/tableHelpers";

const { Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

// ==================== 周视图控制 Hook ====================

function useWeekView(
  currentWeekStart: Dayjs,
  setCurrentWeekStart: (weekStart: Dayjs) => void,
  fetchWeeklyDuty: (weekStart: Dayjs) => Promise<void>
) {
  const handlePrevWeek = () => {
    const newWeekStart = currentWeekStart.subtract(1, "week");
    setCurrentWeekStart(newWeekStart);
    fetchWeeklyDuty(newWeekStart);
  };

  const handleNextWeek = () => {
    const newWeekStart = currentWeekStart.add(1, "week");
    setCurrentWeekStart(newWeekStart);
    fetchWeeklyDuty(newWeekStart);
  };

  const handleTodayWeek = () => {
    const todayWeekStart = dayjs().startOf("week");
    setCurrentWeekStart(todayWeekStart);
    fetchWeeklyDuty(todayWeekStart);
  };

  return {
    handlePrevWeek,
    handleNextWeek,
    handleTodayWeek,
  };
}

// ==================== 表格列定义 ====================

interface DutyTableColumnsProps {
  current: number;
  pageSize: number;
  handleDelete: (id: string) => void;
  /** 列级 sortOrder：返回当前排序列方向，其余 undefined（受控高亮） */
  getColumnSortOrder?: (field: string) => "ascend" | "descend" | undefined;
}

function getDutyTableColumns(props: DutyTableColumnsProps): ColumnsType<DutySchedule> {
  const { current, pageSize, handleDelete, getColumnSortOrder } = props;
  const gso = getColumnSortOrder ?? (() => undefined);

  return [
    {
      title: "序号",
      key: "index",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: "值班日期",
      dataIndex: "scheduleDate",
      key: "scheduleDate",
      width: 120,
      sorter: true,
      sortOrder: gso("scheduleDate"),
      render: formatDate,
    },
    {
      title: "星期",
      key: "weekday",
      width: 80,
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
      width: 120,
      sorter: createSorter<DutySchedule>("poolName", "string"),
    },
    {
      title: "值班人员",
      dataIndex: ["user", "nickname"],
      key: "userName",
      width: 100,
      sorter: createSorter<DutySchedule>("userName", "string"),
      render: (nickname: string, record: DutySchedule) =>
        nickname || record.user?.username || "-",
    },
    {
      title: "值班类型",
      dataIndex: "dutyType",
      key: "dutyType",
      width: 100,
      sorter: true,
      sortOrder: gso("dutyType"),
      render: (type: string) => <Tag color={getDutyTypeColor(type)}>{getDutyTypeText(type)}</Tag>,
    },
    {
      title: "过期状态",
      key: "expired",
      width: 100,
      render: (_: unknown, record: DutySchedule) => {
        const isExpired = dayjs(record.scheduleDate).isBefore(dayjs().startOf("day"));
        return isExpired ? (
          <Tag color="red">已过期</Tag>
        ) : (
          <Tag color="green">未过期</Tag>
        );
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      sorter: true,
      sortOrder: gso("status"),
      render: (status: number) => {
        const config = DUTY_STATUS_CONFIG[status];
        return config ? <Tag color={config.color}>{config.text}</Tag> : <Tag>未知</Tag>;
      },
    },
    {
      title: "是否手动",
      dataIndex: "isManual",
      key: "isManual",
      width: 80,
      sorter: createSorter<DutySchedule>("isManual", "boolean"),
      render: (isManual: boolean) => (isManual ? <Tag color="purple">手动</Tag> : <Tag>自动</Tag>),
    },
    {
      title: "调班原因",
      dataIndex: "swapReason",
      key: "swapReason",
      width: 150,
      ellipsis: true,
      sorter: createSorter<DutySchedule>("swapReason", "string"),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: DutySchedule) => (
        <Button
          type="link"
          size="small"
          icon={<DeleteOutlined />}
          style={{ color: "#ff4d4f" }}
          onClick={() => {
            Modal.confirm({
              title: "确定要删除这条排班吗？",
              okText: "确定",
              cancelText: "取消",
              okButtonProps: { danger: true },
              onOk: () => handleDelete(record.id),
            });
          }}
        >
          删除
        </Button>
      ),
    },
  ];
}

// ==================== 主组件 ====================

const DutySchedulePage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize } = usePagination();

  // 服务端排序状态（field 对应后端 dutyScheduleAllowedSortFields 白名单 key）
  const [sortField, setSortField] = useState<string>("");
  const [sortOrder, setSortOrder] = useState<"ascend" | "descend" | null>(null);
  // 列级 sortOrder：只对当前排序列返回方向，其余 undefined（受控高亮）
  const getColumnSortOrder = (field: string): "ascend" | "descend" | undefined =>
    sortField === field ? (sortOrder ?? undefined) : undefined;

  // 使用数据管理 Hook
  const {
    dataSource,
    total,
    allSchedules,
    pools,
    users,
    weeklyDutyData,
    loading,
    currentWeekStart,
    fetchList,
    fetchAllSchedules,
    fetchPools,
    fetchUsers,
    fetchWeeklyDuty,
    setCurrentWeekStart,
  } = useScheduleData({ current: paginationProps.current ?? 1, pageSize: paginationProps.pageSize ?? 10, searchForm: form });

  // 使用周视图 Hook
  const { handlePrevWeek, handleNextWeek, handleTodayWeek } = useWeekView(
    currentWeekStart,
    setCurrentWeekStart,
    fetchWeeklyDuty
  );

  // 使用弹窗管理 Hook
  const {
    generateModalVisible,
    swapModalVisible,
    manualModalVisible,
    generateForm,
    swapForm,
    manualForm,
    openGenerateModal,
    closeGenerateModal,
    openSwapModal,
    closeSwapModal,
    openManualModal,
    closeManualModal,
    handleGenerate,
    handleSwap,
    handleManual,
    handleDelete,
    handleBatchDelete,
  } = useScheduleModals({
    onLoad: () => {
      fetchList(paginationProps.current ?? 1, paginationProps.pageSize ?? 10);
      fetchAllSchedules();
    },
    allSchedules,
    dataSource,
    current: paginationProps.current ?? 1,
  });

  // 初始化加载
  useEffect(() => {
    // 设置默认值：只显示未过期的排班
    form.setFieldsValue({ expired: 0 });
    fetchList(1, paginationProps.pageSize);
    fetchAllSchedules();
    fetchPools();
    fetchUsers();
    fetchWeeklyDuty(currentWeekStart);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig, _filters: Record<string, unknown>, sorter: any) => {
    // 排序：用 local const 持有新值传 fetchList，规避 React 18 setState 异步时序
    const field = sorter && !Array.isArray(sorter) ? (sorter.field as string) || "" : "";
    const order = sorter && !Array.isArray(sorter) ? (sorter.order ?? null) : null;
    setSortField(field);
    setSortOrder(order);
    fetchList(pagination.current || 1, pagination.pageSize || 10, field, order === "ascend");
  };

  const handleSearch = () => {
    fetchList(1, paginationProps.pageSize);
  };

  const handleReset = () => {
    form.resetFields();
    // 重置后默认只显示未过期的排班
    form.setFieldsValue({ expired: 0 });
    fetchList(1, paginationProps.pageSize);
  };

  // 获取值班类型 Badge 颜色
  const getDutyTypeBadgeColor = (type: string) => {
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

  // 表格列
  const columns = getDutyTableColumns({ current: paginationProps.current ?? 1, pageSize: paginationProps.pageSize ?? 10, handleDelete, getColumnSortOrder });

  return (
    <div className="p-6">
      {/* 值班周视图 */}
      <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>值班周历</span>
          </Space>
        }
        style={{ marginBottom: 16 }}
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
          <Text strong style={{ fontSize: "16px" }}>{getWeekRangeText(currentWeekStart)}</Text>
        </div>
        <Row gutter={16}>
          {getWeekDays(currentWeekStart).map((day) => {
            const dateStr = day.format("YYYY-MM-DD");
            const members = weeklyDutyData[dateStr] || [];
            const isToday = day.format("YYYY-MM-DD") === dayjs().format("YYYY-MM-DD");

            return (
              <Col key={dateStr} span={3}>
                <Card
                  size="small"
                  style={{
                    borderColor: isToday ? "#1890ff" : undefined,
                    backgroundColor: isToday ? "#e6f7ff" : undefined,
                    borderRadius: "4px",
                    minHeight: "80px",
                  }}
                  styles={{ body: { padding: "12px" } }}
                >
                  <div style={{ display: "flex", alignItems: "flex-start" }}>
                    {/* 左侧：日期 */}
                    <div style={{ minWidth: "50px", flexShrink: 0 }}>
                      <div>
                        <Text
                          strong
                          style={{
                            fontSize: "12px",
                            color: isToday ? "#1890ff" : undefined,
                          }}
                        >
                          {getWeekdayText(day)}
                        </Text>
                      </div>
                      <div>
                        <Text style={{ fontSize: "11px", color: "#999" }}>
                          {day.format("MM-DD")}
                        </Text>
                      </div>
                    </div>

                    {/* 右侧：值班人员 */}
                    <div style={{ flex: 1, marginLeft: "8px" }}>
                      <Space size={[4, 4]} wrap>
                        {members.length > 0 ? (
                          members.map((member, index) => (
                            <Tag
                              key={index}
                              color={getDutyTypeBadgeColor(member.dutyType)}
                              style={{
                                fontSize: "11px",
                                borderRadius: "4px",
                                margin: 0,
                              }}
                            >
                              {member.username}
                            </Tag>
                          ))
                        ) : (
                          <Text style={{ fontSize: "11px", color: "#ccc" }}>无值班</Text>
                        )}
                      </Space>
                    </div>
                  </div>
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

      {/* 排班管理 - 筛选表单 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="poolId" label="值班池">
              <Select placeholder="请选择值班池" allowClear className="user-form-input" style={{ width: 150 }} onSearch={() => {}}>
                {pools.map((pool) => (
                  <Option key={pool.id} value={pool.id}>
                    {pool.poolName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="userId" label="值班人员">
              <Select placeholder="请选择值班人员" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                {users.map((user) => (
                  <Option key={user.id} value={user.id}>
                    {user.nickname || user.username}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="dutyType" label="值班类型">
              <Select placeholder="请选择值班类型" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                {DUTY_TYPE_OPTIONS.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="expired" label="排班状态">
              <Select placeholder="请选择排班状态" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                <Option value={0}>未过期</Option>
                <Option value={1}>已过期</Option>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={openGenerateModal}>
              生成排班
            </Button>
            <Button icon={<SwapOutlined />} onClick={openSwapModal}>
              调班
            </Button>
            <Button icon={<EditOutlined />} onClick={openManualModal}>
              手动排班
            </Button>
            {selectedRowKeys.length > 0 && (
              <Button
                icon={<DeleteOutlined />}
                style={{ color: "#ff4d4f" }}
                onClick={() => {
                  Modal.confirm({
                    title: "确定要批量删除选中的排班记录吗？",
                    okText: "确定",
                    cancelText: "取消",
                    okButtonProps: { danger: true },
                    onOk: () => handleBatchDelete(selectedRowKeys, setSelectedRowKeys),
                  });
                }}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* 排班管理 - 数据表格 */}
      <Card>
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
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 生成排班弹窗 */}
      <Modal
        title="生成排班"
        open={generateModalVisible}
        onOk={handleGenerate}
        onCancel={closeGenerateModal}
        destroyOnHidden
      >
        <Form form={generateForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolId"
            label="值班池"
            rules={[{ required: true, message: "请选择值班池" }]}
          >
            <Select placeholder="请选择值班池" onSearch={() => {}}>
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
            rules={[{ required: true, message: "请选择日期范围" }]}
          >
            <RangePicker style={{ width: "100%" }} />
          </Form.Item>

          <Form.Item
            name="dutyType"
            label="值班类型"
            rules={[{ required: true, message: "请选择值班类型" }]}
            initialValue="weekday"
          >
            <Select placeholder="请选择值班类型" onSearch={() => {}}>
              {DUTY_TYPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}值班</Option>
              ))}
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
        onCancel={closeSwapModal}
        destroyOnHidden
        width={600}
      >
        <Form form={swapForm} layout="vertical" preserve={false}>
          <Form.Item
            name="fromScheduleId"
            label="原排班记录"
            rules={[{ required: true, message: "请选择排班记录" }]}
          >
            <Select placeholder="请选择原排班记录" showSearch optionFilterProp="label" onSearch={() => {}}>
              {allSchedules.map((s) => (
                <Option
                  key={s.id}
                  value={s.id}
                  label={formatScheduleOptionLabel(s)}
                >
                  {formatScheduleOptionContent(s, getDutyTypeText)}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="toScheduleId"
            label="目标排班记录"
            rules={[{ required: true, message: "请选择排班记录" }]}
          >
            <Select placeholder="请选择目标排班记录" showSearch optionFilterProp="label" onSearch={() => {}}>
              {allSchedules.map((s) => (
                <Option
                  key={s.id}
                  value={s.id}
                  label={formatScheduleOptionLabel(s)}
                >
                  {formatScheduleOptionContent(s, getDutyTypeText)}
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
        onCancel={closeManualModal}
        destroyOnHidden
      >
        <Form form={manualForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolId"
            label="值班池"
            rules={[{ required: true, message: "请选择值班池" }]}
          >
            <Select placeholder="请选择值班池" onSearch={() => {}}>
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
            rules={[{ required: true, message: "请选择值班日期" }]}
          >
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>

          <Form.Item
            name="dutyType"
            label="值班类型"
            rules={[{ required: true, message: "请选择值班类型" }]}
          >
            <Select placeholder="请选择值班类型" onSearch={() => {}}>
              {DUTY_TYPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="userIds"
            label="值班人员"
            rules={[{ required: true, message: "请选择值班人员" }]}
          >
            <Select mode="multiple" placeholder="请选择值班人员" optionLabelProp="label" onSearch={() => {}}>
              {users.map((user) => (
                <Option key={user.id} value={user.id} label={user.nickname || user.username}>
                  {user.username} - {user.nickname}
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


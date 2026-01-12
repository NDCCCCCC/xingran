import { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Collapse,
  Button,
  Form,
  Select,
  Table,
  Modal,
  Input,
  InputNumber,
  Switch,
  message,
  Popconfirm,
  Space,
  Tag,
  DatePicker,
  Alert,
  Typography,
  Badge,
  Descriptions,
  Upload,
  Spin,
} from 'antd';
import type { UploadProps } from 'antd';
import type { ColumnsType } from 'antd/es/table';
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
  ClockCircleOutlined,
  SettingOutlined,
  SaveOutlined,
  UploadOutlined,
  FileExcelOutlined,
  BellOutlined,
} from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import * as XLSX from 'xlsx';
import {
  getDutyScheduleList,
  generateSchedule,
  swapDuty,
  manualDuty,
  deleteDutySchedule,
  batchDeleteDutySchedules,
  getDutyPoolList,
  getUserList,
  getMonthlyDutySchedule,
  getHolidayList,
  createHoliday,
  updateHoliday,
  deleteHoliday,
  batchCreateHolidays,
  getHolidayYears,
  getDutyConfig,
  updateDutyConfig,
  type DutySchedule,
  type DutyPool,
  type SimpleUser,
  type MonthlyDutyMember,
  type Holiday,
  type DutyConfig,
} from '@/lib/dutyApi';

const { Panel } = Collapse;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { TextArea } = Input;
const { Text } = Typography;

// ============================================
// 值班管理统一页面组件
// ============================================
import type { FC } from 'react';

const DutyManagementPage: FC = () => {
  // ==================== 表单实例 ====================
  const [scheduleForm] = Form.useForm();
  const [generateForm] = Form.useForm();
  const [swapForm] = Form.useForm();
  const [manualForm] = Form.useForm();
  const [holidayEditForm] = Form.useForm();
  const [configForm] = Form.useForm();

  // ==================== 排班管理状态 ====================
  const [scheduleLoading, setScheduleLoading] = useState(false);
  const [schedules, setSchedules] = useState<DutySchedule[]>([]);
  const [scheduleTotal, setScheduleTotal] = useState(0);
  const [scheduleCurrent, setScheduleCurrent] = useState(1);
  const [schedulePageSize, setSchedulePageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const [allSchedules, setAllSchedules] = useState<DutySchedule[]>([]);
  const [currentWeekStart, setCurrentWeekStart] = useState(dayjs().startOf('week'));
  const [weeklyDutyData, setWeeklyDutyData] = useState<Record<string, MonthlyDutyMember[]>>({});

  // 排班弹窗状态
  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [swapModalVisible, setSwapModalVisible] = useState(false);
  const [manualModalVisible, setManualModalVisible] = useState(false);

  // ==================== 节假日管理状态 ====================
  const [holidayLoading, setHolidayLoading] = useState(false);
  const [holidays, setHolidays] = useState<Holiday[]>([]);
  const [holidayYear, setHolidayYear] = useState<number | undefined>(undefined);
  const [availableYears, setAvailableYears] = useState<number[]>([]);

  const [holidayModalVisible, setHolidayModalVisible] = useState(false);
  const [batchHolidayModalVisible, setBatchHolidayModalVisible] = useState(false);
  const [editingHoliday, setEditingHoliday] = useState<Holiday | null>(null);

  const [batchHolidays, setBatchHolidays] = useState<
    Array<{ holidayDate: Dayjs; holidayName: string; isOffday: boolean; holidayType: string; year: number; remark?: string }>
  >([]);

  // ==================== 值班配置状态 ====================
  const [configLoading, setConfigLoading] = useState(false);
  const [configSaving, setConfigSaving] = useState(false);
  const [config, setConfig] = useState<DutyConfig | null>(null);

  // ==================== 共享数据 ====================
  const [pools, setPools] = useState<DutyPool[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);

  // ==================== 数据获取函数 ====================

  // 获取值班池列表
  const fetchPools = async () => {
    try {
      const result = await getDutyPoolList({ current: 1, pageSize: 1000, status: 0 });
      setPools(result.data?.list || []);
    } catch (error) {
      console.error('获取值班池列表失败', error);
    }
  };

  // 获取用户列表
  const fetchUsers = async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取用户列表失败', error);
    }
  };

  // 获取排班列表
  const fetchScheduleList = async (page?: number, pageSize?: number) => {
    setScheduleLoading(true);
    try {
      const values = scheduleForm.getFieldsValue();
      const result = await getDutyScheduleList({
        current: page ?? scheduleCurrent,
        pageSize: pageSize ?? schedulePageSize,
        poolId: values.poolId,
        userId: values.userId,
        startDate: values.dateRange?.[0]?.format('YYYY-MM-DD'),
        endDate: values.dateRange?.[1]?.format('YYYY-MM-DD'),
        dutyType: values.dutyType,
      });
      setSchedules(result.data?.list ?? []);
      setScheduleTotal(result.data?.total ?? 0);
      setScheduleCurrent(result.data?.current ?? 1);
      setSchedulePageSize(result.data?.page_size ?? 10);
    } catch (error) {
      message.error('获取排班列表失败');
    } finally {
      setScheduleLoading(false);
    }
  };

  // 获取所有排班（用于调班）
  const fetchAllSchedules = async () => {
    try {
      const result = await getDutyScheduleList({ current: 1, pageSize: 1000 });
      setAllSchedules(result.data?.list ?? []);
    } catch (error) {
      console.error('获取所有排班失败', error);
    }
  };

  // 获取周度值班数据
  const fetchWeeklyDuty = async (weekStart: Dayjs) => {
    try {
      const weekEnd = weekStart.endOf('week');
      const year = weekStart.year();
      const monthStart = weekStart.month() + 1;
      const monthEnd = weekEnd.month() + 1;

      if (monthStart === monthEnd) {
        const result = await getMonthlyDutySchedule(year, monthStart);
        setWeeklyDutyData(result.data || {});
      } else {
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

  // 获取节假日列表
  const fetchHolidayList = async (y?: number) => {
    setHolidayLoading(true);
    try {
      const targetYear = y ?? holidayYear;
      if (targetYear === undefined) {
        setHolidayLoading(false);
        return;
      }
      const result = await getHolidayList(targetYear);
      const data = result as any;
      setHolidays(data.data);
      if (y !== undefined) setHolidayYear(y);
    } catch (error) {
      message.error('获取节假日列表失败');
    } finally {
      setHolidayLoading(false);
    }
  };

  // 获取可用年份
  const fetchAvailableYears = async () => {
    try {
      const result = await getHolidayYears();
      const years = result.data || [];
      setAvailableYears(years);

      if (years.length > 0 && holidayYear === undefined) {
        const latestYear = years[0];
        setHolidayYear(latestYear);
        fetchHolidayList(latestYear);
      }
    } catch (error) {
      console.error('获取年份列表失败:', error);
    }
  };

  // 获取值班配置
  const fetchConfig = async () => {
    setConfigLoading(true);
    try {
      const result = await getDutyConfig();
      const configData = result as any;
      setConfig(configData.data);
      configForm.setFieldsValue({
        reminderEnabled: configData.data.reminderEnabled,
        reminderTime: dayjs(configData.data.reminderTime, 'HH:mm'),
        reminderChannels: configData.data.reminderChannels.split(','),
        beforeReminderMinutes: configData.data.beforeReminderMinutes,
      });
    } catch (error) {
      message.error('获取配置失败');
    } finally {
      setConfigLoading(false);
    }
  };

  // ==================== 初始化 ====================
  useEffect(() => {
    fetchPools();
    fetchUsers();
    fetchScheduleList(1, 10);
    fetchAllSchedules();
    fetchWeeklyDuty(currentWeekStart);
    fetchAvailableYears();
    fetchConfig();
  }, []);

  // ==================== 排班管理操作 ====================

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

  // 生成排班
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
      fetchScheduleList();
      fetchWeeklyDuty(currentWeekStart);
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('生成排班失败');
    }
  };

  // 调班
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
      fetchScheduleList();
      fetchWeeklyDuty(currentWeekStart);
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('调班失败');
    }
  };

  // 手动排班
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
      fetchScheduleList();
      fetchWeeklyDuty(currentWeekStart);
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('手动排班失败');
    }
  };

  // 删除排班
  const handleDeleteSchedule = async (id: string) => {
    try {
      await deleteDutySchedule(id);
      message.success('删除成功');
      const newPage = schedules.length === 1 && scheduleCurrent > 1 ? scheduleCurrent - 1 : scheduleCurrent;
      fetchScheduleList(newPage);
      fetchWeeklyDuty(currentWeekStart);
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除排班
  const handleBatchDeleteSchedule = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的排班记录');
      return;
    }
    try {
      await batchDeleteDutySchedules(selectedRowKeys as string[]);
      message.success(`成功删除 ${selectedRowKeys.length} 条排班记录`);
      setSelectedRowKeys([]);
      const newPage = schedules.length === selectedRowKeys.length && scheduleCurrent > 1 ? scheduleCurrent - 1 : scheduleCurrent;
      fetchScheduleList(newPage);
      fetchWeeklyDuty(currentWeekStart);
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // ==================== 节假日管理操作 ====================

  // 新增节假日
  const handleAddHoliday = () => {
    setEditingHoliday(null);
    holidayEditForm.resetFields();
    holidayEditForm.setFieldsValue({
      holidayDate: dayjs(),
      isOffday: true,
      holidayType: 'legal',
      year: holidayYear ?? availableYears[0] ?? new Date().getFullYear(),
    });
    setHolidayModalVisible(true);
  };

  // 编辑节假日
  const handleEditHoliday = (record: Holiday) => {
    setEditingHoliday(record);
    holidayEditForm.setFieldsValue({
      holidayDate: dayjs(record.holidayDate),
      holidayName: record.holidayName,
      isOffday: record.isOffday,
      holidayType: record.holidayType,
      year: record.year,
      remark: record.remark,
    });
    setHolidayModalVisible(true);
  };

  // 删除节假日
  const handleDeleteHoliday = async (id: string) => {
    try {
      await deleteHoliday(id);
      message.success('删除成功');
      fetchHolidayList();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 保存节假日
  const handleHolidayModalOk = async () => {
    try {
      const values = await holidayEditForm.validateFields();
      const data = {
        holidayDate: values.holidayDate.format('YYYY-MM-DD'),
        holidayName: values.holidayName,
        isOffday: values.isOffday,
        holidayType: values.holidayType,
        year: values.year,
        remark: values.remark,
      };
      if (editingHoliday) {
        await updateHoliday(editingHoliday.id, data);
        message.success('更新成功');
      } else {
        await createHoliday(data);
        message.success('创建成功');
      }
      setHolidayModalVisible(false);
      fetchHolidayList();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error(editingHoliday ? '更新失败' : '创建失败');
    }
  };

  // 批量新增节假日
  const handleBatchAddHoliday = () => {
    const defaultYear = holidayYear ?? availableYears[0] ?? new Date().getFullYear();
    setBatchHolidays([{ holidayDate: dayjs(), holidayName: '', isOffday: true, holidayType: 'legal', year: defaultYear }]);
    setBatchHolidayModalVisible(true);
  };

  const addBatchHolidayRow = () => {
    const defaultYear = holidayYear ?? availableYears[0] ?? new Date().getFullYear();
    setBatchHolidays([
      ...batchHolidays,
      { holidayDate: dayjs(), holidayName: '', isOffday: true, holidayType: 'legal', year: defaultYear },
    ]);
  };

  const removeBatchHolidayRow = (index: number) => {
    const newRows = batchHolidays.filter((_, i) => i !== index);
    setBatchHolidays(newRows);
  };

  const updateBatchHolidayRow = (index: number, field: string, value: any) => {
    const newRows = [...batchHolidays];
    newRows[index] = { ...newRows[index], [field]: value };
    setBatchHolidays(newRows);
  };

  const handleBatchHolidaySubmit = async () => {
    for (let i = 0; i < batchHolidays.length; i++) {
      const row = batchHolidays[i];
      if (!row.holidayName) {
        message.error(`第 ${i + 1} 行：请输入节假日名称`);
        return;
      }
    }

    try {
      const data = batchHolidays.map((row) => ({
        holidayDate: row.holidayDate.format('YYYY-MM-DD'),
        holidayName: row.holidayName,
        isOffday: row.isOffday,
        holidayType: row.holidayType as 'legal' | 'workday' | 'custom',
        year: row.year,
        remark: row.remark,
      }));
      await batchCreateHolidays(data as any);
      message.success(`成功创建 ${data.length} 条节假日记录`);
      setBatchHolidayModalVisible(false);
      fetchHolidayList();
    } catch (error) {
      message.error('批量创建失败');
    }
  };

  // Excel 导入
  const handleExcelImport: UploadProps['customRequest'] = async (options) => {
    const { file, onSuccess, onError } = options;
    try {
      const fileObj = file as File;
      const data = await fileObj.arrayBuffer();
      const workbook = XLSX.read(data);
      const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
      const jsonData = XLSX.utils.sheet_to_json(firstSheet, { header: 1 }) as any[][];

      if (jsonData.length < 2) {
        message.error('Excel文件为空或格式不正确');
        return;
      }

      const holidays: Array<{
        holidayDate: string;
        holidayName: string;
        isOffday: boolean;
        holidayType: 'legal' | 'workday' | 'custom';
        year: number;
        remark?: string;
      }> = [];

      for (let i = 1; i < jsonData.length; i++) {
        const row = jsonData[i];
        if (!row[0] && !row[1]) continue;

        const dateStr = row[0];
        const name = row[1];
        const typeStr = row[2];
        const isOffdayStr = row[3];
        const year = row[4] || new Date().getFullYear();
        const remark = row[5];

        let formattedDate: string;
        if (typeof dateStr === 'number') {
          const date = XLSX.SSF.parse_date_code(dateStr);
          formattedDate = `${date.y}-${String(date.m).padStart(2, '0')}-${String(date.d).padStart(2, '0')}`;
        } else {
          const parsed = dayjs(dateStr);
          if (!parsed.isValid()) {
            message.error(`第 ${i + 1} 行：日期格式不正确 - ${dateStr}`);
            return;
          }
          formattedDate = parsed.format('YYYY-MM-DD');
        }

        let holidayType: 'legal' | 'workday' | 'custom' = 'legal';
        if (typeof typeStr === 'string') {
          const typeLower = typeStr.toLowerCase();
          if (typeLower.includes('法定') || typeLower === 'legal') {
            holidayType = 'legal';
          } else if (typeLower.includes('调休') || typeLower.includes('工作') || typeLower === 'workday') {
            holidayType = 'workday';
          } else if (typeLower.includes('自定义') || typeLower === 'custom') {
            holidayType = 'custom';
          }
        }

        let isOffday = true;
        if (typeof isOffdayStr === 'boolean') {
          isOffday = isOffdayStr;
        } else if (typeof isOffdayStr === 'string' || typeof isOffdayStr === 'number') {
          const str = String(isOffdayStr).toLowerCase();
          isOffday = str === '是' || str === 'true' || str === '1' || str === '休息';
        }

        holidays.push({
          holidayDate: formattedDate,
          holidayName: name || '节假日',
          isOffday,
          holidayType,
          year: Number(year),
          remark,
        });
      }

      if (holidays.length === 0) {
        message.error('没有有效的节假日数据');
        return;
      }

      await batchCreateHolidays(holidays as any);
      message.success(`成功导入 ${holidays.length} 条节假日记录`);
      fetchHolidayList();
      fetchAvailableYears();

      onSuccess?.(holidays);
    } catch (error: any) {
      console.error('Excel导入失败:', error);
      message.error(`导入失败: ${error.message || '未知错误'}`);
      onError?.(error as Error);
    }
  };

  // 下载模板
  const downloadTemplate = () => {
    const template = [
      ['日期', '名称', '类型', '是否休息', '年份', '备注'],
    ];
    const ws = XLSX.utils.aoa_to_sheet(template);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '节假日模板');
    XLSX.writeFile(wb, '节假日导入模板.xlsx');
  };

  // ==================== 值班配置操作 ====================

  const handleSaveConfig = async () => {
    try {
      setConfigSaving(true);
      const values = await configForm.validateFields();
      const data = {
        reminderEnabled: values.reminderEnabled,
        reminderTime: values.reminderTime.format('HH:mm'),
        reminderChannels: Array.isArray(values.reminderChannels) ? values.reminderChannels.join(',') : values.reminderChannels,
        beforeReminderMinutes: values.beforeReminderMinutes,
      };
      await updateDutyConfig(data);
      message.success('配置保存成功');
      fetchConfig();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('配置保存失败');
    } finally {
      setConfigSaving(false);
    }
  };

  // ==================== 工具函数 ====================

  const getWeekRangeText = () => {
    const weekEnd = currentWeekStart.endOf('week');
    return `${currentWeekStart.format('YYYY年MM月DD日')} - ${weekEnd.format('YYYY年MM月DD日')}`;
  };

  const getWeekDays = () => {
    const days = [];
    for (let i = 0; i < 7; i++) {
      days.push(currentWeekStart.add(i, 'day'));
    }
    return days;
  };

  const getWeekdayText = (day: Dayjs) => {
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    return weekdays[day.day()];
  };

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

  // ==================== 表格列定义 ====================

  const scheduleColumns: ColumnsType<DutySchedule> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (scheduleCurrent - 1) * schedulePageSize + index + 1,
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
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: DutySchedule) => (
        <Popconfirm title="确定要删除这条排班吗？" onConfirm={() => handleDeleteSchedule(record.id)} okText="确定" cancelText="取消">
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  // ==================== 渲染 ====================

  return (
    <div style={{ padding: '16px' }}>
      <Collapse
        defaultActiveKey={['weekly', 'schedules']}
        style={{ background: 'transparent' }}
      >
        {/* ==================== 值班周视图 ==================== */}
        <Panel
          header={
            <Space>
              <CalendarOutlined />
              <span style={{ fontSize: '16px', fontWeight: 500 }}>值班周历</span>
            </Space>
          }
          key="weekly"
          extra={
            <Space>
              <Button size="small" icon={<LeftOutlined />} onClick={handlePrevWeek}>
                上一周
              </Button>
              <Button size="small" onClick={handleTodayWeek}>
                本周
              </Button>
              <Button size="small" icon={<RightOutlined />} onClick={handleNextWeek}>
                下一周
              </Button>
            </Space>
          }
        >
          <Card bordered={false} style={{ marginBottom: 0 }}>
            <div style={{ marginBottom: '16px' }}>
              <Text strong style={{ fontSize: '14px' }}>{getWeekRangeText()}</Text>
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
                      <div style={{ marginBottom: '6px' }}>
                        <Text
                          strong
                          style={{
                            fontSize: '13px',
                            color: isToday ? '#1890ff' : undefined,
                          }}
                        >
                          {getWeekdayText(day)}
                        </Text>
                        <br />
                        <Text style={{ fontSize: '12px', color: '#999' }}>
                          {day.format('MM-DD')}
                        </Text>
                      </div>
                      <Space direction="vertical" size="small" style={{ width: '100%' }}>
                        {members.length > 0 ? (
                          members.map((member, index) => (
                            <Badge
                              key={index}
                              color={getDutyTypeBadgeColor(member.dutyType)}
                              text={
                                <span
                                  style={{
                                    fontSize: '12px',
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
                          <Text style={{ fontSize: '12px', color: '#ccc' }}>无值班</Text>
                        )}
                      </Space>
                    </Card>
                  </Col>
                );
              })}
            </Row>
            <div style={{ marginTop: '12px', display: 'flex', gap: '16px', fontSize: '12px' }}>
              <Badge color="blue" text={<Text type="secondary">工作日</Text>} />
              <Badge color="orange" text={<Text type="secondary">周末</Text>} />
              <Badge color="red" text={<Text type="secondary">节假日</Text>} />
            </div>
          </Card>
        </Panel>

        {/* ==================== 排班列表 ==================== */}
        <Panel
          header={
            <Space>
              <ClockCircleOutlined />
              <span style={{ fontSize: '16px', fontWeight: 500 }}>排班列表</span>
            </Space>
          }
          key="schedules"
          extra={
            <Space>
              <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setGenerateModalVisible(true)}>
                生成排班
              </Button>
              <Button size="small" icon={<SwapOutlined />} onClick={() => setSwapModalVisible(true)}>
                调班
              </Button>
              <Button size="small" icon={<EditOutlined />} onClick={() => setManualModalVisible(true)}>
                手动排班
              </Button>
              {selectedRowKeys.length > 0 && (
                <Popconfirm
                  title="确定要批量删除选中的排班记录吗？"
                  onConfirm={handleBatchDeleteSchedule}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button size="small" danger icon={<DeleteOutlined />}>
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                </Popconfirm>
              )}
            </Space>
          }
        >
          <Card bordered={false} style={{ marginBottom: 0 }}>
            {/* 搜索表单 */}
            <Form form={scheduleForm} layout="inline" style={{ marginBottom: 16 }}>
              <Form.Item name="poolId">
                <Select placeholder="值班池" allowClear style={{ width: 140 }}>
                  {pools.map((pool) => (
                    <Option key={pool.id} value={pool.id}>{pool.poolName}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="userId">
                <Select placeholder="值班人员" allowClear style={{ width: 120 }}>
                  {users.map((user) => (
                    <Option key={user.id} value={user.id}>{user.nickName || user.username}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="dutyType">
                <Select placeholder="值班类型" allowClear style={{ width: 120 }}>
                  <Option value="weekday">工作日</Option>
                  <Option value="weekend">周末</Option>
                  <Option value="holiday">节假日</Option>
                </Select>
              </Form.Item>
              <Form.Item name="dateRange">
                <RangePicker style={{ width: 240 }} />
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" icon={<SearchOutlined />} onClick={() => fetchScheduleList(1, schedulePageSize)}>
                    查询
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => { scheduleForm.resetFields(); fetchScheduleList(1, schedulePageSize); }}>
                    重置
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            <Table
              rowKey="id"
              columns={scheduleColumns}
              dataSource={schedules}
              loading={scheduleLoading}
              scroll={{ x: 1100 }}
              rowSelection={{
                selectedRowKeys,
                onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys as string[]),
              }}
              pagination={{
                current: scheduleCurrent,
                pageSize: schedulePageSize,
                total: scheduleTotal,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (t) => `共 ${t} 条`,
                pageSizeOptions: ['10', '20', '50', '100'],
                onChange: (page, pageSize) => {
                  fetchScheduleList(page, pageSize);
                },
              }}
            />
          </Card>
        </Panel>

        {/* ==================== 节假日管理 ==================== */}
        <Panel
          header={
            <Space>
              <CalendarOutlined />
              <span style={{ fontSize: '16px', fontWeight: 500 }}>节假日管理</span>
            </Space>
          }
          key="holidays"
        >
          <Card bordered={false} style={{ marginBottom: 0 }}>
            {/* 年份选择和操作按钮 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <Space>
                <Text>年份：</Text>
                <Select
                  value={holidayYear}
                  onChange={(y) => fetchHolidayList(y)}
                  style={{ width: 120 }}
                >
                  {availableYears.map((y) => (
                    <Option key={y} value={y}>{y}年</Option>
                  ))}
                </Select>
              </Space>
              <Space>
                <Button icon={<PlusOutlined />} onClick={handleAddHoliday}>
                  新增
                </Button>
                <Button icon={<PlusOutlined />} onClick={handleBatchAddHoliday}>
                  批量新增
                </Button>
                <Button icon={<FileExcelOutlined />} onClick={downloadTemplate}>
                  下载模板
                </Button>
                <Upload
                  accept=".xlsx,.xls"
                  showUploadList={false}
                  customRequest={handleExcelImport}
                >
                  <Button icon={<UploadOutlined />} type="primary">
                    导入Excel
                  </Button>
                </Upload>
                <Button icon={<ReloadOutlined />} onClick={() => fetchHolidayList()} loading={holidayLoading}>
                  刷新
                </Button>
              </Space>
            </div>

            <Table
              rowKey="id"
              dataSource={holidays}
              loading={holidayLoading}
              pagination={{
                pageSize: 20,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (t) => `共 ${t} 条`,
                pageSizeOptions: ['20', '50', '100'],
              }}
              columns={[
                {
                  title: '日期',
                  dataIndex: 'holidayDate',
                  key: 'holidayDate',
                  width: 120,
                  render: (date: string) => dayjs(date).format('YYYY-MM-DD'),
                },
                {
                  title: '名称',
                  dataIndex: 'holidayName',
                  key: 'holidayName',
                },
                {
                  title: '类型',
                  dataIndex: 'holidayType',
                  key: 'holidayType',
                  width: 100,
                  render: (type: string) => {
                    const colorMap: Record<string, string> = {
                      legal: 'red',
                      workday: 'orange',
                      custom: 'blue',
                    };
                    const textMap: Record<string, string> = {
                      legal: '法定节假日',
                      workday: '调休工作日',
                      custom: '自定义',
                    };
                    return <Tag color={colorMap[type]}>{textMap[type]}</Tag>;
                  },
                },
                {
                  title: '是否休息',
                  dataIndex: 'isOffday',
                  key: 'isOffday',
                  width: 100,
                  render: (isOffday: boolean) => (
                    <Tag color={isOffday ? 'green' : 'default'}>{isOffday ? '休息' : '工作'}</Tag>
                  ),
                },
                {
                  title: '年份',
                  dataIndex: 'year',
                  key: 'year',
                  width: 80,
                },
                {
                  title: '备注',
                  dataIndex: 'remark',
                  key: 'remark',
                  ellipsis: true,
                },
                {
                  title: '操作',
                  key: 'action',
                  width: 150,
                  fixed: 'right',
                  render: (_: unknown, record: Holiday) => (
                    <Space size="small">
                      <Button type="link" size="small" onClick={() => handleEditHoliday(record)}>编辑</Button>
                      <Popconfirm title="确定删除?" onConfirm={() => handleDeleteHoliday(record.id)} okText="确定" cancelText="取消">
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
              size="small"
            />
          </Card>
        </Panel>

        {/* ==================== 值班配置 ==================== */}
        <Panel
          header={
            <Space>
              <SettingOutlined />
              <span style={{ fontSize: '16px', fontWeight: 500 }}>值班配置</span>
            </Space>
          }
          key="config"
        >
          <Card bordered={false} style={{ marginBottom: 0 }}>
            <Row gutter={24}>
              <Col span={12}>
                <Card title="当前配置" size="small" style={{ height: '100%' }}>
                  {config ? (
                    <Descriptions column={1} bordered size="small">
                      <Descriptions.Item label="提醒状态">
                        {config.reminderEnabled ? (
                          <Tag color="green" icon={<BellOutlined />}>已启用</Tag>
                        ) : (
                          <Tag color="red">已禁用</Tag>
                        )}
                      </Descriptions.Item>
                      <Descriptions.Item label="提醒时间">{config.reminderTime}</Descriptions.Item>
                      <Descriptions.Item label="提醒渠道">
                        {config.reminderChannels.split(',').map((ch) => {
                          const map: Record<string, string> = {
                            websocket: '站内通知',
                            email: '邮件',
                            sms: '短信',
                          };
                          return <Tag key={ch}>{map[ch] || ch}</Tag>;
                        })}
                      </Descriptions.Item>
                      <Descriptions.Item label="提前提醒">
                        {config.beforeReminderMinutes ? `${config.beforeReminderMinutes}分钟` : '当天提醒'}
                      </Descriptions.Item>
                    </Descriptions>
                  ) : (
                    <div style={{ textAlign: 'center', padding: '20px' }}>
                      <Spin spinning={configLoading} />
                    </div>
                  )}
                </Card>
              </Col>
              <Col span={12}>
                <Card title="修改配置" size="small" style={{ height: '100%' }}>
                  <Form form={configForm} layout="vertical">
                    <Form.Item label="启用提醒" name="reminderEnabled" valuePropName="checked">
                      <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                    </Form.Item>
                    <Form.Item label="提醒时间" name="reminderTime" rules={[{ required: true, message: '请选择时间' }]}>
                      <Input type="time" style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item label="提醒渠道" name="reminderChannels" rules={[{ required: true, message: '请选择渠道' }]}>
                      <Select mode="multiple" placeholder="选择渠道">
                        <Option value="websocket">站内通知</Option>
                        <Option value="email" disabled>邮件（未开放）</Option>
                        <Option value="sms" disabled>短信（未开放）</Option>
                      </Select>
                    </Form.Item>
                    <Form.Item label="提前提醒(分钟)" name="beforeReminderMinutes">
                      <InputNumber placeholder="当天提醒请留空" style={{ width: '100%' }} min={0} />
                    </Form.Item>
                    <Form.Item>
                      <Button type="primary" icon={<SaveOutlined />} onClick={handleSaveConfig} loading={configSaving} block>
                        保存配置
                      </Button>
                    </Form.Item>
                  </Form>
                </Card>
              </Col>
            </Row>
          </Card>
        </Panel>
      </Collapse>

      {/* 生成排班弹窗 */}
      <Modal
        title="生成排班"
        open={generateModalVisible}
        onOk={handleGenerate}
        onCancel={() => setGenerateModalVisible(false)}
        destroyOnClose
      >
        <Form form={generateForm} layout="vertical" preserve={false}>
          <Form.Item name="poolId" label="值班池" rules={[{ required: true, message: '请选择值班池' }]}>
            <Select placeholder="请选择值班池">
              {pools.map((pool) => (
                <Option key={pool.id} value={pool.id}>{pool.poolName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="dateRange" label="日期范围" rules={[{ required: true, message: '请选择日期范围' }]}>
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="dutyType" label="值班类型" rules={[{ required: true, message: '请选择值班类型' }]} initialValue="weekday">
            <Select placeholder="请选择值班类型">
              <Option value="weekday">工作日值班</Option>
              <Option value="weekend">周末值班</Option>
              <Option value="holiday">节假日值班</Option>
            </Select>
          </Form.Item>
          <Form.Item name="clearExists" label="清除已有排班" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Alert message="系统将根据值班池中的成员顺序，按轮询方式自动生成排班记录。只对符合所选值班类型的日期进行排班。" type="info" showIcon />
        </Form>
      </Modal>

      {/* 调班弹窗 */}
      <Modal
        title="调班"
        open={swapModalVisible}
        onOk={handleSwap}
        onCancel={() => setSwapModalVisible(false)}
        destroyOnClose
        width={500}
      >
        <Form form={swapForm} layout="vertical" preserve={false}>
          <Form.Item name="fromScheduleId" label="原排班记录" rules={[{ required: true, message: '请选择' }]}>
            <Select placeholder="请选择" showSearch optionFilterProp="label">
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
          <Form.Item name="toScheduleId" label="目标排班记录" rules={[{ required: true, message: '请选择' }]}>
            <Select placeholder="请选择" showSearch optionFilterProp="label">
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
            <TextArea rows={3} placeholder="请输入调班原因" />
          </Form.Item>
          <Alert message="选择两条排班记录进行互换，两位值班人员将交换各自的值班日期。" type="info" showIcon />
        </Form>
      </Modal>

      {/* 手动排班弹窗 */}
      <Modal
        title="手动排班"
        open={manualModalVisible}
        onOk={handleManual}
        onCancel={() => setManualModalVisible(false)}
        destroyOnClose
      >
        <Form form={manualForm} layout="vertical" preserve={false}>
          <Form.Item name="poolId" label="值班池" rules={[{ required: true, message: '请选择值班池' }]}>
            <Select placeholder="请选择值班池">
              {pools.map((pool) => (
                <Option key={pool.id} value={pool.id}>{pool.poolName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="dutyDate" label="值班日期" rules={[{ required: true, message: '请选择日期' }]}>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="dutyType" label="值班类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select placeholder="请选择值班类型">
              <Option value="weekday">工作日</Option>
              <Option value="weekend">周末</Option>
              <Option value="holiday">节假日</Option>
            </Select>
          </Form.Item>
          <Form.Item name="userIds" label="值班人员" rules={[{ required: true, message: '请选择人员' }]}>
            <Select mode="multiple" placeholder="请选择值班人员" optionLabelProp="label">
              {users.map((user) => (
                <Option key={user.id} value={user.id} label={user.nickName || user.username}>
                  {user.username} - {user.nickName}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="reason" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 节假日新增/编辑弹窗 */}
      <Modal
        title={editingHoliday ? '编辑节假日' : '新增节假日'}
        open={holidayModalVisible}
        onOk={handleHolidayModalOk}
        onCancel={() => setHolidayModalVisible(false)}
        destroyOnClose
      >
        <Form form={holidayEditForm} layout="vertical" preserve={false}>
          <Form.Item name="holidayDate" label="日期" rules={[{ required: true, message: '请选择日期' }]}>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="holidayName" label="节假日名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="请输入节假日名称" />
          </Form.Item>
          <Form.Item name="holidayType" label="类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select placeholder="请选择类型">
              <Option value="legal">法定节假日</Option>
              <Option value="workday">调休工作日</Option>
              <Option value="custom">自定义</Option>
            </Select>
          </Form.Item>
          <Form.Item name="isOffday" label="是否休息日" valuePropName="checked">
            <Switch checkedChildren="休息" unCheckedChildren="工作" />
          </Form.Item>
          <Form.Item name="year" label="年份" rules={[{ required: true, message: '请输入年份' }]}>
            <InputNumber min={2020} max={2030} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 批量新增节假日弹窗 */}
      <Modal
        title="批量新增节假日"
        open={batchHolidayModalVisible}
        onOk={handleBatchHolidaySubmit}
        onCancel={() => setBatchHolidayModalVisible(false)}
        width={700}
        destroyOnClose
      >
        <div className="mb-3">
          <Button type="dashed" onClick={addBatchHolidayRow} block icon={<PlusOutlined />}>
            添加一行
          </Button>
        </div>
        <div className="overflow-x-auto" style={{ maxHeight: '300px', overflow: 'auto' }}>
          <table className="w-full border-collapse" style={{ fontSize: '12px' }}>
            <thead className="bg-gray-50 sticky top-0">
              <tr>
                <th className="border p-2">日期</th>
                <th className="border p-2">名称</th>
                <th className="border p-2">类型</th>
                <th className="border p-2">是否休息</th>
                <th className="border p-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {batchHolidays.map((row, index) => (
                <tr key={index}>
                  <td className="border p-2">
                    <DatePicker
                      value={row.holidayDate}
                      onChange={(date) => updateBatchHolidayRow(index, 'holidayDate', date)}
                      style={{ width: '100%' }}
                      size="small"
                    />
                  </td>
                  <td className="border p-2">
                    <Input
                      value={row.holidayName}
                      onChange={(e) => updateBatchHolidayRow(index, 'holidayName', e.target.value)}
                      placeholder="节假日名称"
                      size="small"
                    />
                  </td>
                  <td className="border p-2">
                    <Select
                      value={row.holidayType}
                      onChange={(v) => updateBatchHolidayRow(index, 'holidayType', v)}
                      style={{ width: '100%' }}
                      size="small"
                    >
                      <Option value="legal">法定节假日</Option>
                      <Option value="workday">调休工作日</Option>
                      <Option value="custom">自定义</Option>
                    </Select>
                  </td>
                  <td className="border p-2 text-center">
                    <Switch
                      checked={row.isOffday}
                      onChange={(v) => updateBatchHolidayRow(index, 'isOffday', v)}
                      checkedChildren="休息"
                      unCheckedChildren="工作"
                      size="small"
                    />
                  </td>
                  <td className="border p-2 text-center">
                    <Button type="link" danger size="small" onClick={() => removeBatchHolidayRow(index)}>
                      删除
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Modal>
    </div>
  );
};

export default DutyManagementPage;

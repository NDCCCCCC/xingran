import { useState, useEffect, useCallback } from 'react';
import { Tabs, Modal, Form, Select, DatePicker, message, Button } from 'antd';
import { LeftOutlined, RightOutlined, CalendarOutlined, SettingOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import * as XLSX from 'xlsx';
import { getDutyPoolList, getUserList } from '@/lib/dutyApi';
import type { DutyPool, SimpleUser, Holiday } from '@/lib/dutyApi';
import {
  WeeklyView,
  ScheduleManagement,
  HolidayManagement,
  DutyConfig,
  GenerateScheduleModal,
} from './components';
import { useScheduleData, useHolidayData, useDutyConfig } from './hooks';

const { RangePicker } = DatePicker;
const { Option } = Select;

// ==================== 主组件 ====================

export default function DutyManagement() {
  // ==================== 排班数据 ====================
  const scheduleData = useScheduleData();

  // ==================== 节假日数据 ====================
  const holidayData = useHolidayData();

  // ==================== 值班配置 ====================
  const dutyConfig = useDutyConfig();

  // ==================== 其他状态 ====================
  const [activeTab, setActiveTab] = useState('weekly');
  const [pools, setPools] = useState<DutyPool[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);

  // 模态框状态
  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [swapModalVisible, setSwapModalVisible] = useState(false);
  const [manualModalVisible, setManualModalVisible] = useState(false);
  const [holidayModalVisible, setHolidayModalVisible] = useState(false);
  const [holidayBatchModalVisible, setHolidayBatchModalVisible] = useState(false);

  // 调班表单
  const [swapForm] = Form.useForm();
  // 手动排班表单
  const [manualForm] = Form.useForm();
  // 节假日表单
  const [holidayForm] = Form.useForm();

  // ==================== 初始化 ====================

  useEffect(() => {
    fetchPools();
    fetchUsers();
    scheduleData.fetchWeeklyDuty(scheduleData.currentWeekStart);
    holidayData.fetchYears();
    dutyConfig.fetch();
  }, []);

  // ==================== 数据获取 ====================

  const fetchPools = async () => {
    try {
      const result = await getDutyPoolList({ current: 1, pageSize: 100 });
      setPools(result.data?.list || []);
    } catch (error) {
      message.error('获取值班池列表失败');
    }
  };

  const fetchUsers = async () => {
    try {
      const result = await getUserList({ current: 1, pageSize: 1000 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取用户列表失败', error);
    }
  };

  // ==================== 排班操作 ====================

  interface ScheduleSearchParams {
    poolId?: string;
    userId?: string;
    dutyType?: string;
    dateRange?: [dayjs.Dayjs, dayjs.Dayjs];
  }

  const handleScheduleSearch = useCallback((values: ScheduleSearchParams) => {
    const { poolId, userId, dutyType, dateRange } = values;
    scheduleData.fetchList(1, scheduleData.pageSize, {
      poolId,
      userId,
      dutyType,
      startDate: dateRange?.[0]?.format('YYYY-MM-DD'),
      endDate: dateRange?.[1]?.format('YYYY-MM-DD'),
    });
  }, [scheduleData]);

  const handleScheduleReset = useCallback(() => {
    scheduleData.fetchList(1, scheduleData.pageSize);
  }, [scheduleData]);

  const handleGenerateClick = useCallback(() => {
    setGenerateModalVisible(true);
  }, []);

  interface GenerateScheduleValues {
    poolId: string;
    startDate: string;
    endDate: string;
    dutyType: string;
    clearExists?: boolean;
  }

  const handleGenerateOk = useCallback(async (values: GenerateScheduleValues) => {
    return await scheduleData.generate(values);
  }, [scheduleData]);

  const handleSwapClick = useCallback(() => {
    scheduleData.fetchAllSchedules();
    setSwapModalVisible(true);
  }, [scheduleData]);

  const handleSwapOk = useCallback(async () => {
    try {
      const values = await swapForm.validateFields();
      await scheduleData.swap(values);
      setSwapModalVisible(false);
      swapForm.resetFields();
    } catch (error) {
      // 表单验证失败
    }
  }, [scheduleData, swapForm]);

  const handleManualClick = useCallback(() => {
    setManualModalVisible(true);
  }, []);

  const handleManualOk = useCallback(async () => {
    try {
      const values = await manualForm.validateFields();
      const success = await scheduleData.manual({
        poolId: values.poolId,
        dutyDate: values.dutyDate.format('YYYY-MM-DD'),
        dutyType: values.dutyType,
        userIds: values.userIds,
        reason: values.reason,
      });
      if (success) {
        setManualModalVisible(false);
        manualForm.resetFields();
      }
    } catch (error) {
      // 表单验证失败
    }
  }, [scheduleData, manualForm]);

  // ==================== 节假日操作 ====================

  const handleHolidayYearChange = useCallback((year: number) => {
    holidayData.setHolidayYear(year);
    holidayData.fetchList(year);
  }, [holidayData]);

  const handleHolidayAdd = useCallback(() => {
    setHolidayModalVisible(true);
  }, []);

  const handleHolidayBatchAdd = useCallback(() => {
    setHolidayBatchModalVisible(true);
  }, []);

  const handleHolidayEdit = useCallback((record: Holiday) => {
    holidayForm.setFieldsValue({
      ...record,
      holidayDate: dayjs(record.holidayDate),
    });
    setHolidayModalVisible(true);
  }, [holidayForm]);

  const handleHolidayDelete = useCallback(async (id: string) => {
    await holidayData.deleteOne(id);
  }, [holidayData]);

  const handleHolidayModalOk = useCallback(async () => {
    try {
      const values = await holidayForm.validateFields();
      const data: Omit<Holiday, 'id' | 'createdAt' | 'updatedAt' | 'createdBy'> = {
        holidayDate: values.holidayDate.format('YYYY-MM-DD'),
        holidayName: values.holidayName,
        isOffday: values.isOffday ?? true,
        holidayType: (values.holidayType || 'custom') as 'legal' | 'workday' | 'custom',
        year: values.holidayDate.year(),
        remark: values.remark,
      };

      const id = holidayForm.getFieldValue('id');
      if (id) {
        await holidayData.update(id, data);
      } else {
        await holidayData.create(data);
      }
      setHolidayModalVisible(false);
      holidayForm.resetFields();
    } catch (error) {
      // 表单验证失败
    }
  }, [holidayData, holidayForm]);

  const handleDownloadTemplate = useCallback(() => {
    const template = [
      {
        '日期(YYYY-MM-DD)': '2024-01-01',
        '名称': '元旦',
        '类型(legal/workday/custom)': 'legal',
        '是否休息(true/false)': 'true',
        '备注': '法定节假日',
      },
    ];
    const worksheet = XLSX.utils.json_to_sheet(template);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, '节假日模板');
    XLSX.writeFile(workbook, '节假日导入模板.xlsx');
    message.success('模板下载成功');
  }, []);

  interface HolidayExcelRow {
    '日期(YYYY-MM-DD)'?: string;
    '日期'?: string;
    '名称'?: string;
    '节假日名称'?: string;
    '类型(legal/workday/custom)'?: string;
    '类型'?: string;
    '是否休息(true/false)'?: boolean;
    '备注'?: string;
  }

  interface ImportOptions {
    file: File;
    onProgress?: (event: { percent: number }) => void;
    onSuccess?: (response?: unknown) => void;
    onError?: (error: Error) => void;
  }

  const isValidHolidayType = (value: string): value is 'legal' | 'workday' | 'custom' => {
    return ['legal', 'workday', 'custom'].includes(value);
  };

  const handleImport = useCallback(async (options: ImportOptions) => {
    const { file, onProgress, onSuccess, onError } = options;
    const reader = new FileReader();

    reader.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        const percent = Math.round((e.loaded / e.total) * 100);
        onProgress({ percent });
      }
    };

    reader.onload = async (e) => {
      try {
        const data = e.target?.result;
        if (!data) {
          throw new Error('文件读取失败');
        }
        const workbook = XLSX.read(data, { type: 'binary' });
        const sheetName = workbook.SheetNames[0];
        const worksheet = workbook.Sheets[sheetName];
        const jsonData = XLSX.utils.sheet_to_json<HolidayExcelRow>(worksheet);

        if (jsonData.length === 0) {
          throw new Error('Excel 文件为空');
        }

        const holidays: Omit<Holiday, 'id' | 'createdAt' | 'updatedAt' | 'createdBy'>[] = [];

        for (let i = 0; i < jsonData.length; i++) {
          const row = jsonData[i];
          const rowNum = i + 2; // Excel 行号（从1开始，第1行是表头）

          // 验证日期
          const dateStr = row['日期(YYYY-MM-DD)'] || row['日期'];
          if (!dateStr) {
            throw new Error(`第 ${rowNum} 行：日期字段不能为空`);
          }
          if (!dayjs(dateStr).isValid()) {
            throw new Error(`第 ${rowNum} 行：日期格式不正确 (${dateStr})`);
          }

          // 验证名称
          const holidayName = row['名称'] || row['节假日名称'];
          if (!holidayName) {
            throw new Error(`第 ${rowNum} 行：名称字段不能为空`);
          }

          // 验证类型
          const holidayTypeRaw = row['类型(legal/workday/custom)'] || row['类型'] || 'custom';
          if (!isValidHolidayType(holidayTypeRaw)) {
            throw new Error(`第 ${rowNum} 行：类型值不正确 (${holidayTypeRaw})，应为 legal、workday 或 custom`);
          }

          holidays.push({
            holidayDate: dateStr,
            holidayName,
            isOffday: row['是否休息(true/false)'] !== false,
            holidayType: holidayTypeRaw,
            year: dayjs(dateStr).year(),
            remark: row['备注'] || '',
          });
        }

        await holidayData.batchCreate(holidays);
        onSuccess?.(undefined);
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : '导入失败';
        message.error(errorMsg);
        onError?.(error instanceof Error ? error : new Error('导入失败'));
      }
    };

    reader.onerror = () => {
      message.error('文件读取失败');
      onError?.(new Error('文件读取失败'));
    };

    reader.readAsBinaryString(file);
  }, [holidayData]);

  // ==================== 配置保存 ====================

  interface DutyConfigValues {
    reminderEnabled: boolean;
    reminderTime: dayjs.Dayjs;
    reminderChannels: string[];
    beforeReminderMinutes?: number;
  }

  const handleConfigSave = useCallback(async (values: DutyConfigValues) => {
    return await dutyConfig.save(values);
  }, [dutyConfig]);

  // ==================== 渲染 ====================

  return (
    <div>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'weekly',
            label: (
              <span>
                <CalendarOutlined />
                周视图
              </span>
            ),
            children: (
              <>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                  <Button icon={<LeftOutlined />} onClick={scheduleData.prevWeek}>
                    上一周
                  </Button>
                  <Button onClick={scheduleData.todayWeek}>
                    今天
                  </Button>
                  <Button icon={<RightOutlined />} onClick={scheduleData.nextWeek}>
                    下一周
                  </Button>
                </div>
                <WeeklyView
                  currentWeekStart={scheduleData.currentWeekStart}
                  weeklyDutyData={scheduleData.weeklyDutyData}
                />
              </>
            ),
          },
          {
            key: 'schedule',
            label: '排班管理',
            children: (
              <ScheduleManagement
                schedules={scheduleData.schedules}
                loading={scheduleData.loading}
                total={scheduleData.total}
                current={scheduleData.current}
                pageSize={scheduleData.pageSize}
                selectedRowKeys={scheduleData.selectedRowKeys}
                pools={pools}
                users={users}
                onSearch={handleScheduleSearch}
                onReset={handleScheduleReset}
                onPageChange={(page, size) => scheduleData.fetchList(page, size)}
                onSelectedChange={(keys) => scheduleData.setSelectedRowKeys(keys)}
                onDelete={(id) => scheduleData.deleteOne(id)}
                onBatchDelete={() => scheduleData.batchDelete(scheduleData.selectedRowKeys)}
                onGenerateClick={handleGenerateClick}
                onSwapClick={handleSwapClick}
                onManualClick={handleManualClick}
              />
            ),
          },
          {
            key: 'holiday',
            label: '节假日管理',
            children: (
              <HolidayManagement
                holidays={holidayData.holidays}
                loading={holidayData.loading}
                holidayYear={holidayData.holidayYear}
                availableYears={holidayData.availableYears}
                onYearChange={handleHolidayYearChange}
                onRefresh={() => holidayData.fetchList()}
                onAdd={handleHolidayAdd}
                onBatchAdd={handleHolidayBatchAdd}
                onEdit={handleHolidayEdit}
                onDelete={handleHolidayDelete}
                onImport={handleImport}
                onDownloadTemplate={handleDownloadTemplate}
              />
            ),
          },
          {
            key: 'config',
            label: (
              <span>
                <SettingOutlined />
                值班配置
              </span>
            ),
            children: (
              <DutyConfig
                config={dutyConfig.config}
                loading={dutyConfig.loading}
                saving={dutyConfig.saving}
                onSave={handleConfigSave}
              />
            ),
          },
        ]}
      />

      {/* 生成排班模态框 */}
      <GenerateScheduleModal
        visible={generateModalVisible}
        onCancel={() => setGenerateModalVisible(false)}
        onOk={handleGenerateOk}
        pools={pools}
      />

      {/* 调班模态框 */}
      <Modal
        title="调班"
        open={swapModalVisible}
        onOk={handleSwapOk}
        onCancel={() => setSwapModalVisible(false)}
        destroyOnHidden
      >
        <Form form={swapForm} layout="vertical">
          <Form.Item name="fromScheduleId" label="原排班" rules={[{ required: true }]}>
            <Select placeholder="选择原排班">
              {scheduleData.allSchedules.map((s) => (
                <Option key={s.id} value={s.id}>
                  {dayjs(s.scheduleDate).format('YYYY-MM-DD')} - {s.user?.nickName || s.user?.username}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="toScheduleId" label="目标排班" rules={[{ required: true }]}>
            <Select placeholder="选择目标排班">
              {scheduleData.allSchedules.map((s) => (
                <Option key={s.id} value={s.id}>
                  {dayjs(s.scheduleDate).format('YYYY-MM-DD')} - {s.user?.nickName || s.user?.username}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="reason" label="调班原因" rules={[{ required: true, message: '请输入调班原因' }]}>
            <Select placeholder="选择原因">
              <Option value="临时有事">临时有事</Option>
              <Option value="身体原因">身体原因</Option>
              <Option value="工作原因">工作原因</Option>
              <Option value="其他">其他</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 手动排班模态框 */}
      <Modal
        title="手动排班"
        open={manualModalVisible}
        onOk={handleManualOk}
        onCancel={() => setManualModalVisible(false)}
        destroyOnHidden
        width={500}
      >
        <Form form={manualForm} layout="vertical">
          <Form.Item name="poolId" label="值班池" rules={[{ required: true, message: '请选择值班池' }]}>
            <Select placeholder="选择值班池">
              {pools.map((pool) => (
                <Option key={pool.id} value={pool.id}>{pool.poolName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="dutyDate" label="值班日期" rules={[{ required: true, message: '请选择日期' }]}>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="dutyType" label="值班类型" rules={[{ required: true }]} initialValue="weekday">
            <Select>
              <Option value="weekday">工作日</Option>
              <Option value="weekend">周末</Option>
              <Option value="holiday">节假日</Option>
            </Select>
          </Form.Item>
          <Form.Item name="userIds" label="值班人员" rules={[{ required: true, message: '请选择人员' }]}>
            <Select mode="multiple" placeholder="选择人员">
              {users.map((user) => (
                <Option key={user.id} value={user.id}>{user.nickName || user.username}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="reason" label="备注">
            <Select placeholder="选择原因（可选）" allowClear>
              <Option value="临时排班">临时排班</Option>
              <Option value="替班">替班</Option>
              <Option value="补充排班">补充排班</Option>
              <Option value="其他">其他</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 节假日模态框 */}
      <Modal
        title="节假日"
        open={holidayModalVisible}
        onOk={handleHolidayModalOk}
        onCancel={() => setHolidayModalVisible(false)}
        destroyOnHidden
      >
        <Form form={holidayForm} layout="vertical">
          <Form.Item name="holidayDate" label="日期" rules={[{ required: true, message: '请选择日期' }]}>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="holidayName" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Select placeholder="选择或输入">
              <Option value="元旦">元旦</Option>
              <Option value="春节">春节</Option>
              <Option value="清明节">清明节</Option>
              <Option value="劳动节">劳动节</Option>
              <Option value="端午节">端午节</Option>
              <Option value="中秋节">中秋节</Option>
              <Option value="国庆节">国庆节</Option>
            </Select>
          </Form.Item>
          <Form.Item name="holidayType" label="类型" rules={[{ required: true }]} initialValue="custom">
            <Select>
              <Option value="legal">法定节假日</Option>
              <Option value="workday">调休工作日</Option>
              <Option value="custom">自定义</Option>
            </Select>
          </Form.Item>
          <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
            <Select>
              <Option value={true}>休息</Option>
              <Option value={false}>工作</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Select placeholder="选择备注（可选）" allowClear>
              <Option value="放假">放假</Option>
              <Option value="调休">调休</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 批量添加节假日模态框 */}
      <Modal
        title="批量添加节假日"
        open={holidayBatchModalVisible}
        onOk={async () => {
          try {
            const values = await holidayForm.validateFields();
            const { dateRange, holidayName, holidayType, isOffday } = values;

            const startDate = dateRange[0];
            const endDate = dateRange[1];
            const holidays = [];
            let current = startDate;

            // 限制日期范围最多90天，防止用户误操作创建过多数据
            const maxDays = 90;
            const daysDiff = endDate.diff(startDate, 'day') + 1;
            if (daysDiff > maxDays) {
              message.error(`日期范围不能超过 ${maxDays} 天（当前选择 ${daysDiff} 天）`);
              return;
            }

            while (current.isBefore(endDate) || current.isSame(endDate, 'day')) {
              holidays.push({
                holidayDate: current.format('YYYY-MM-DD'),
                holidayName,
                isOffday: isOffday ?? true,
                holidayType: holidayType || 'custom',
                year: current.year(),
                remark: values.remark,
              });
              current = current.add(1, 'day');
            }

            await holidayData.batchCreate(holidays);
            setHolidayBatchModalVisible(false);
            holidayForm.resetFields();
          } catch (error) {
            // 表单验证失败
          }
        }}
        onCancel={() => setHolidayBatchModalVisible(false)}
        destroyOnHidden
      >
        <Form form={holidayForm} layout="vertical">
          <Form.Item name="dateRange" label="日期范围" rules={[{ required: true, message: '请选择日期范围' }]} extra="最多可选择90天">
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="holidayName" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Select placeholder="选择假期名称">
              <Option value="春节">春节</Option>
              <Option value="国庆节">国庆节</Option>
            </Select>
          </Form.Item>
          <Form.Item name="holidayType" label="类型" rules={[{ required: true }]} initialValue="legal">
            <Select>
              <Option value="legal">法定节假日</Option>
              <Option value="custom">自定义</Option>
            </Select>
          </Form.Item>
          <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
            <Select>
              <Option value={true}>休息</Option>
              <Option value={false}>工作</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

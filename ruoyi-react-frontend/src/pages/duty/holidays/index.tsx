import React, { useState, useEffect } from 'react';
import {
  Button,
  Table,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Switch,
  message,
  Popconfirm,
  Card,
  Space,
  Tag,
  DatePicker,
  Alert,
  Upload,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { UploadProps } from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  UploadOutlined,
  CalendarOutlined,
  FileExcelOutlined,
} from '@ant-design/icons';
import dayjs, { Dayjs } from 'dayjs';
import * as XLSX from 'xlsx';
import {
  getHolidayList,
  createHoliday,
  updateHoliday,
  deleteHoliday,
  batchCreateHolidays,
  getHolidayYears,
  type Holiday,
} from '@/lib/dutyApi';

const { Option } = Select;
const { TextArea } = Input;

const DutyHolidayPage: React.FC = () => {
  const [editForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<Holiday[]>([]);
  const [year, setYear] = useState(new Date().getFullYear());
  const [availableYears, setAvailableYears] = useState<number[]>([]);

  const [modalVisible, setModalVisible] = useState(false);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<Holiday | null>(null);

  const [batchHolidays, setBatchHolidays] = useState<
    Array<{ holidayDate: Dayjs; holidayName: string; isOffday: boolean; holidayType: string; year: number; remark?: string }>
  >([{ holidayDate: dayjs(), holidayName: '', isOffday: true, holidayType: 'legal', year }]);

  const fetchList = async (y?: number) => {
    setLoading(true);
    try {
      const targetYear = y ?? year;
      const result = await getHolidayList(targetYear);
      const data = result as any; // API响应格式: { code: 0, data: Holiday[] }
      setDataSource(data.data);
      if (y !== undefined) setYear(y);
    } catch (error) {
      message.error('获取节假日列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchAvailableYears = async () => {
    try {
      const result = await getHolidayYears();
      const years = result.data || [];
      setAvailableYears(years);

      // 如果当前年份不在可用列表中，选择最新的年份
      if (years.length > 0 && !years.includes(year)) {
        setYear(years[0]);
        fetchList(years[0]);
      }
    } catch (error) {
      console.error('获取年份列表失败:', error);
    }
  };

  useEffect(() => {
    fetchList();
    fetchAvailableYears();
  }, []);

  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      holidayDate: dayjs(),
      isOffday: true,
      holidayType: 'legal',
      year: year,
    });
    setModalVisible(true);
  };

  const handleEdit = (record: Holiday) => {
    setEditingRecord(record);
    editForm.setFieldsValue({
      holidayDate: dayjs(record.holidayDate),
      holidayName: record.holidayName,
      isOffday: record.isOffday,
      holidayType: record.holidayType,
      year: record.year,
      remark: record.remark,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHoliday(id);
      message.success('删除成功');
      fetchList();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();
      const data = {
        holidayDate: values.holidayDate.format('YYYY-MM-DD'),
        holidayName: values.holidayName,
        isOffday: values.isOffday,
        holidayType: values.holidayType,
        year: values.year,
        remark: values.remark,
      };
      if (editingRecord) {
        await updateHoliday(editingRecord.id, data);
        message.success('更新成功');
      } else {
        await createHoliday(data);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchList();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error(editingRecord ? '更新失败' : '创建失败');
    }
  };

  const handleBatchAdd = () => {
    setBatchHolidays([{ holidayDate: dayjs(), holidayName: '', isOffday: true, holidayType: 'legal', year }]);
    setBatchModalVisible(true);
  };

  const addBatchRow = () => {
    setBatchHolidays([
      ...batchHolidays,
      { holidayDate: dayjs(), holidayName: '', isOffday: true, holidayType: 'legal', year },
    ]);
  };

  const removeBatchRow = (index: number) => {
    const newRows = batchHolidays.filter((_, i) => i !== index);
    setBatchHolidays(newRows);
  };

  const updateBatchRow = (index: number, field: string, value: any) => {
    const newRows = [...batchHolidays];
    newRows[index] = { ...newRows[index], [field]: value };
    setBatchHolidays(newRows);
  };

  const handleBatchSubmit = async () => {
    // 验证所有行
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
      await batchCreateHolidays(data);
      message.success(`成功创建 ${data.length} 条节假日记录`);
      setBatchModalVisible(false);
      fetchList();
    } catch (error) {
      message.error('批量创建失败');
    }
  };

  // Excel 文件导入处理
  const handleExcelImport: UploadProps['customRequest'] = async (options) => {
    const { file, onSuccess, onError } = options;
    try {
      // 处理 file 对象
      const fileObj = file as File;
      const data = await fileObj.arrayBuffer();
      const workbook = XLSX.read(data);
      const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
      const jsonData = XLSX.utils.sheet_to_json(firstSheet, { header: 1 }) as any[][];

      // 验证Excel格式（跳过表头）
      if (jsonData.length < 2) {
        message.error('Excel文件为空或格式不正确');
        return;
      }

      // Excel 格式: 日期 | 名称 | 类型 | 是否休息 | 年份 | 备注
      const holidays: Array<{
        holidayDate: string;
        holidayName: string;
        isOffday: boolean;
        holidayType: 'legal' | 'workday' | 'custom';
        year: number;
        remark?: string;
      }> = [];

      // 从第二行开始解析（跳过表头）
      for (let i = 1; i < jsonData.length; i++) {
        const row = jsonData[i];
        if (!row[0] && !row[1]) continue; // 跳过空行

        const dateStr = row[0];
        const name = row[1];
        const typeStr = row[2];
        const isOffdayStr = row[3];
        const year = row[4] || new Date().getFullYear();
        const remark = row[5];

        // 解析日期
        let formattedDate: string;
        if (typeof dateStr === 'number') {
          // Excel 日期序列号转换
          const date = XLSX.SSF.parse_date_code(dateStr);
          formattedDate = `${date.y}-${String(date.m).padStart(2, '0')}-${String(date.d).padStart(2, '0')}`;
        } else {
          // 字符串日期，尝试多种格式
          const parsed = dayjs(dateStr);
          if (!parsed.isValid()) {
            message.error(`第 ${i + 1} 行：日期格式不正确 - ${dateStr}`);
            return;
          }
          formattedDate = parsed.format('YYYY-MM-DD');
        }

        // 解析类型
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

        // 解析是否休息
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

      // 批量创建
      await batchCreateHolidays(holidays as any);
      message.success(`成功导入 ${holidays.length} 条节假日记录`);
      fetchList();
      fetchAvailableYears(); // 刷新年份列表

      onSuccess?.(holidays);
    } catch (error: any) {
      console.error('Excel导入失败:', error);
      message.error(`导入失败: ${error.message || '未知错误'}`);
      onError?.(error as Error);
    }
  };

  // 下载Excel模板
  const downloadTemplate = () => {
    // 创建模板数据（仅表头，不包含示例数据）
    const template = [
      ['日期', '名称', '类型', '是否休息', '年份', '备注'],
    ];

    // 创建工作簿
    const ws = XLSX.utils.aoa_to_sheet(template);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '节假日模板');

    // 下载
    XLSX.writeFile(wb, '节假日导入模板.xlsx');
  };

  const columns: ColumnsType<Holiday> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => index + 1,
    },
    {
      title: '日期',
      dataIndex: 'holidayDate',
      key: 'holidayDate',
      width: 120,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD'),
      sorter: (a: Holiday, b: Holiday) => dayjs(a.holidayDate).unix() - dayjs(b.holidayDate).unix(),
    },
    {
      title: '星期',
      key: 'weekday',
      width: 80,
      render: (_: unknown, record: Holiday) => {
        const weekday = dayjs(record.holidayDate).day();
        const weekdays = ['日', '一', '二', '三', '四', '五', '六'];
        return `周${weekdays[weekday]}`;
      },
    },
    {
      title: '节假日名称',
      dataIndex: 'holidayName',
      key: 'holidayName',
      width: 150,
    },
    {
      title: '类型',
      dataIndex: 'holidayType',
      key: 'holidayType',
      width: 100,
      render: (type: string) => {
        if (type === 'legal') return <Tag color="red">法定节假日</Tag>;
        if (type === 'workday') return <Tag color="orange">调休工作日</Tag>;
        if (type === 'custom') return <Tag color="blue">自定义</Tag>;
        return <Tag>{type}</Tag>;
      },
    },
    {
      title: '是否休息',
      dataIndex: 'isOffday',
      key: 'isOffday',
      width: 100,
      render: (isOffday: boolean) => (
        <Tag color={isOffday ? 'green' : 'default'}>{isOffday ? '休息日' : '工作日'}</Tag>
      ),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      width: 200,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right',
      render: (_: unknown, record: Holiday) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定要删除吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>节假日管理 - {year}年</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              value={year}
              onChange={(y) => fetchList(y)}
              style={{ width: 120 }}
              suffixIcon={null}
            >
              {availableYears.map((y) => (
                <Option key={y} value={y}>
                  {y}年
                </Option>
              ))}
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => fetchList()}>
              刷新
            </Button>
          </Space>
        }
      >
        <div className="mb-4 flex gap-2">
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增节假日
          </Button>
          <Button icon={<UploadOutlined />} onClick={handleBatchAdd}>
            批量新增
          </Button>
          <Button icon={<FileExcelOutlined />} onClick={downloadTemplate}>
            下载导入模板
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
        </div>

        <Alert
          title="提示"
          description="支持通过Excel批量导入节假日数据。请先下载模板，按模板格式填写后导入。日期格式支持 YYYY-MM-DD 或 Excel日期格式。"
          type="info"
          showIcon
          closable
          className="mb-4"
        />

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={false}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? '编辑节假日' : '新增节假日'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            name="holidayDate"
            label="日期"
            rules={[{ required: true, message: '请选择日期' }]}
          >
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="holidayName"
            label="节假日名称"
            rules={[{ required: true, message: '请输入节假日名称' }]}
          >
            <Input placeholder="请输入节假日名称" />
          </Form.Item>

          <Form.Item
            name="holidayType"
            label="类型"
            rules={[{ required: true, message: '请选择类型' }]}
          >
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

      {/* 批量新增弹窗 */}
      <Modal
        title="批量新增节假日"
        open={batchModalVisible}
        onOk={handleBatchSubmit}
        onCancel={() => setBatchModalVisible(false)}
        width={800}
        destroyOnHidden
      >
        <div className="mb-4">
          <Button type="dashed" onClick={addBatchRow} block icon={<PlusOutlined />}>
            添加一行
          </Button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="bg-gray-50">
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
                      onChange={(date) => updateBatchRow(index, 'holidayDate', date)}
                      style={{ width: '100%' }}
                    />
                  </td>
                  <td className="border p-2">
                    <Input
                      value={row.holidayName}
                      onChange={(e) => updateBatchRow(index, 'holidayName', e.target.value)}
                      placeholder="节假日名称"
                    />
                  </td>
                  <td className="border p-2">
                    <Select
                      value={row.holidayType}
                      onChange={(v) => updateBatchRow(index, 'holidayType', v)}
                      style={{ width: '100%' }}
                    >
                      <Option value="legal">法定节假日</Option>
                      <Option value="workday">调休工作日</Option>
                      <Option value="custom">自定义</Option>
                    </Select>
                  </td>
                  <td className="border p-2 text-center">
                    <Switch
                      checked={row.isOffday}
                      onChange={(v) => updateBatchRow(index, 'isOffday', v)}
                      checkedChildren="休息"
                      unCheckedChildren="工作"
                    />
                  </td>
                  <td className="border p-2 text-center">
                    <Button type="link" danger size="small" onClick={() => removeBatchRow(index)}>
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

export default DutyHolidayPage;

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  Switch,
  message,
  Tag,
  Tooltip,
  Popconfirm,
  Drawer,
  Typography,
  Row,
  Col,
  Statistic
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  HistoryOutlined,
  ClockCircleOutlined
} from '@ant-design/icons';
import { post, put, del, get, postLongRequest } from '@/lib/api';
import type { ColumnsType } from 'antd/es/table';

const { TextArea } = Input;

interface JobInfo {
  id: string;
  jobName: string;
  jobGroup: string;
  invokeTarget: string;
  cronExpression: string;
  misfirePolicy: number;
  concurrent: boolean;
  status: number;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
  remark?: string;
  nextRunTime?: string;
  prevRunTime?: string;
}

interface JobLog {
  id: string;
  jobName: string;
  jobGroup: string;
  invokeTarget: string;
  jobMessage: string;
  status: number;
  exceptionInfo?: string;
  startTime?: string;
  endTime?: string;
  duration: number;
  createdAt: string;
}

interface PageData {
  list: JobInfo[];
  total: number;
  current: number;
  pageSize: number;
}

const JobManager: React.FC = () => {
  const [jobs, setJobs] = useState<JobInfo[]>([]);
  const [jobLogs, setJobLogs] = useState<JobLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm, setSearchForm] = useState({
    jobName: '',
    jobGroup: '',
    status: undefined as number | undefined
  });
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('');
  const [isEdit, setIsEdit] = useState(false);
  const [logDrawerVisible, setLogDrawerVisible] = useState(false);
  const [selectedJob, setSelectedJob] = useState<JobInfo | null>(null);
  const [cronNextTime, setCronNextTime] = useState<string>('');

  const [form] = Form.useForm();

  // 获取任务列表
  const fetchJobs = async () => {
    setLoading(true);
    try {
      const result = await post<PageData>('/monitor/jobs/list', {
        ...searchForm,
        current,
        pageSize
      });

      setJobs(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      console.error('获取任务列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 获取任务日志
  const fetchJobLogs = async (jobName?: string, jobGroup?: string) => {
    try {
      const result = await post<{ list: JobLog[]; total: number }>('/monitor/jobs/logs/list', {
        jobName: jobName || '',
        jobGroup: jobGroup || '',
        current: 1,
        pageSize: 50
      });

      setJobLogs(result.data?.list || []);
    } catch (error) {
      console.error('获取任务日志失败:', error);
    }
  };

  // 验证Cron表达式并获取下次执行时间
  const validateCronAndGetNextTime = async (cronExpr: string) => {
    if (!cronExpr) {
      setCronNextTime('');
      return;
    }
    // 简单计算下次执行时间（前端模拟）
    try {
      const now = new Date();
      const nextTime = new Date(now.getTime() + 5 * 60 * 1000); // 假设5分钟后
      setCronNextTime(`下次执行: ${nextTime.toLocaleString()}`);
    } catch (error) {
      setCronNextTime('无效的Cron表达式');
    }
  };

  // 打开新增/编辑模态框
  const openModal = (record?: JobInfo) => {
    if (record) {
      setIsEdit(true);
      setModalTitle('编辑定时任务');
      form.setFieldsValue({
        id: record.id,
        jobName: record.jobName,
        jobGroup: record.jobGroup,
        invokeTarget: record.invokeTarget,
        cronExpression: record.cronExpression,
        misfirePolicy: record.misfirePolicy,
        concurrent: record.concurrent ? 1 : 0,
        status: record.status,
        remark: record.remark || ''
      });
      validateCronAndGetNextTime(record.cronExpression);
    } else {
      setIsEdit(false);
      setModalTitle('新增定时任务');
      form.resetFields();
      form.setFieldsValue({
        misfirePolicy: 1,
        concurrent: 0,
        status: 0
      });
      setCronNextTime('');
    }
    setModalVisible(true);
  };

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      if (isEdit) {
        await put(`/monitor/jobs/${values.id}`, values);
      } else {
        await post('/monitor/jobs', values);
      }

      message.success(isEdit ? '更新成功' : '新增成功');
      setModalVisible(false);
      fetchJobs();
    } catch (error) {
      console.error('提交失败:', error);
    }
  };

  // 删除任务
  const handleDelete = async (record: JobInfo) => {
    try {
      await del(`/monitor/jobs/${record.id}`);
      message.success('删除成功');
      fetchJobs();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  // 启动/停止任务
  const handleToggleStatus = async (record: JobInfo) => {
    try {
      await put(`/monitor/jobs/${record.id}/status`, {
        status: record.status === 0 ? 1 : 0
      });

      message.success(record.status === 0 ? '暂停成功' : '启动成功');
      fetchJobs();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  // 立即执行任务
  const handleExecute = async (record: JobInfo) => {
    try {
      // 使用长时间请求函数（5分钟超时），因为设备监控任务可能耗时较长
      await postLongRequest(`/monitor/jobs/${record.id}/execute`, {});
      message.success('执行成功');
      if (selectedJob?.id === record.id) {
        fetchJobLogs(record.jobName, record.jobGroup);
      }
    } catch (error) {
      console.error('执行失败:', error);
    }
  };

  // 查看日志
  const handleViewLogs = (record: JobInfo) => {
    setSelectedJob(record);
    setLogDrawerVisible(true);
    fetchJobLogs(record.jobName, record.jobGroup);
  };

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    fetchJobs();
  };

  // 重置
  const handleReset = () => {
    setSearchForm({
      jobName: '',
      jobGroup: '',
      status: undefined
    });
    setCurrent(1);
  };

  // 刷新
  const handleRefresh = () => {
    fetchJobs();
  };

  // 分页变化
  const handleTableChange = (pagination: any) => {
    setCurrent(pagination.current);
    setPageSize(pagination.pageSize);
  };

  // Cron表达式变化时验证
  const handleCronChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    validateCronAndGetNextTime(value);
  };

  useEffect(() => {
    fetchJobs();
  }, [current, pageSize]);

  // 状态选项
  const statusOptions = [
    { label: '正常', value: 0 },
    { label: '暂停', value: 1 }
  ];

  // 策略选项
  const misfirePolicyOptions = [
    { label: '立即执行', value: 1 },
    { label: '执行一次', value: 2 },
    { label: '放弃执行', value: 3 }
  ];

  // 常用Cron表达式
  const commonCronExpressions = [
    { label: '每分钟', value: '0 * * * * ?' },
    { label: '每小时', value: '0 0 * * * ?' },
    { label: '每天0点', value: '0 0 0 * * ?' },
    { label: '每周一0点', value: '0 0 0 * * MON ?' },
    { label: '每月1号0点', value: '0 0 0 1 * ?' },
    { label: '每5分钟', value: '0 */5 * * * ?' },
    { label: '每10分钟', value: '0 */10 * * * ?' },
    { label: '每30分钟', value: '0 */30 * * * ?' }
  ];

  const columns: ColumnsType<JobInfo> = [
    {
      title: '任务编号',
      dataIndex: 'id',
      key: 'id',
      width: 100,
      ellipsis: true,
    },
    {
      title: '任务名称',
      dataIndex: 'jobName',
      key: 'jobName',
      width: 150,
    },
    {
      title: '任务组',
      dataIndex: 'jobGroup',
      key: 'jobGroup',
      width: 120,
    },
    {
      title: '调用目标',
      dataIndex: 'invokeTarget',
      key: 'invokeTarget',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Cron表达式',
      dataIndex: 'cronExpression',
      key: 'cronExpression',
      width: 150,
      render: (text: string) => (
        <Tooltip title={text}>
          <code className="bg-gray-100 px-2 py-1 rounded text-xs">{text}</code>
        </Tooltip>
      ),
    },
    {
      title: '并发执行',
      dataIndex: 'concurrent',
      key: 'concurrent',
      width: 100,
      render: (concurrent: boolean) => (
        <Tag color={concurrent ? 'green' : 'orange'}>
          {concurrent ? '允许' : '禁止'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number, record: JobInfo) => (
        <Space>
          <Tag color={status === 0 ? 'success' : 'warning'}>
            {status === 0 ? '正常' : '暂停'}
          </Tag>
          <Tooltip title={status === 0 ? '暂停任务' : '启动任务'}>
            <Button
              type="text"
              size="small"
              icon={status === 0 ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => handleToggleStatus(record)}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '下次执行时间',
      dataIndex: 'nextRunTime',
      key: 'nextRunTime',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      fixed: 'right',
      render: (_, record: JobInfo) => (
        <Space size="small">
          <Tooltip title="立即执行">
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => handleExecute(record)}
            />
          </Tooltip>
          <Tooltip title="查看日志">
            <Button
              type="text"
              icon={<HistoryOutlined />}
              onClick={() => handleViewLogs(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => openModal(record)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Popconfirm
              title="确定要删除这个任务吗？"
              description="删除后将无法恢复"
              onConfirm={() => handleDelete(record)}
            >
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  const jobLogColumns: ColumnsType<JobLog> = [
    {
      title: '执行时间',
      dataIndex: 'startTime',
      key: 'startTime',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '执行时长',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration: number) => {
        if (!duration) return '-';
        if (duration >= 1000) {
          // 超过1秒，显示秒（保留2位小数）
          const seconds = (duration / 1000).toFixed(2);
          return `${seconds}s`;
        }
        return `${duration}ms`;
      },
    },
    {
      title: '执行消息',
      dataIndex: 'jobMessage',
      key: 'jobMessage',
      width: 200,
      ellipsis: true,
    },
    {
      title: '执行状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '异常信息',
      dataIndex: 'exceptionInfo',
      key: 'exceptionInfo',
      ellipsis: true,
      render: (info: string) => info ? (
        <Tooltip title={info}>
          <span className="text-red-500">查看错误</span>
        </Tooltip>
      ) : '-',
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">定时任务管理</h1>

        {/* 搜索表单 */}
        <Card>
          <Row gutter={16}>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="任务名称"
                value={searchForm.jobName}
                onChange={(e) => setSearchForm({ ...searchForm, jobName: e.target.value })}
                allowClear
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="任务组"
                value={searchForm.jobGroup}
                onChange={(e) => setSearchForm({ ...searchForm, jobGroup: e.target.value })}
                allowClear
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Select
                placeholder="状态"
                value={searchForm.status}
                onChange={(value) => setSearchForm({ ...searchForm, status: value })}
                allowClear
                style={{ width: '100%' }}
                options={statusOptions}
              />
            </Col>
            <Col xs={24} sm={24} md={6}>
              <Space>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
                  新增
                </Button>
                <Button icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Col>
          </Row>
        </Card>
      </div>

      {/* 任务列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={jobs}
          rowKey="id"
          loading={loading}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
            onChange: handleTableChange,
          }}
          scroll={{ x: 1500 }}
        />
      </Card>

      {/* 新增/编辑模态框 */}
      <Modal
        title={modalTitle}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            misfirePolicy: 1,
            concurrent: 0,
            status: 0
          }}
        >
          <Form.Item
            label="任务名称"
            name="jobName"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>
          <Form.Item
            label="任务组"
            name="jobGroup"
            rules={[{ required: true, message: '请输入任务组' }]}
          >
            <Input placeholder="请输入任务组，如：DEFAULT" />
          </Form.Item>
          <Form.Item
            label="调用目标"
            name="invokeTarget"
            rules={[{ required: true, message: '请输入调用目标字符串' }]}
            extra="任务执行时调用的目标方法或脚本"
          >
            <TextArea
              placeholder="请输入调用目标字符串"
              rows={3}
            />
          </Form.Item>
          <Form.Item
            label="Cron表达式"
            name="cronExpression"
            rules={[{ required: true, message: '请输入Cron表达式' }]}
          >
            <Space.Compact style={{ width: '100%' }}>
              <Input
                placeholder="请输入Cron表达式，如：0 0 0 * * ?"
                onChange={handleCronChange}
              />
              <Select
                placeholder="常用"
                style={{ width: 120 }}
                onChange={(value) => {
                  form.setFieldValue('cronExpression', value);
                  validateCronAndGetNextTime(value);
                }}
                options={commonCronExpressions}
              />
            </Space.Compact>
          </Form.Item>
          {cronNextTime && (
            <div className="text-sm text-gray-500 -mt-4 mb-4">{cronNextTime}</div>
          )}
          <Form.Item
            label="执行策略"
            name="misfirePolicy"
            rules={[{ required: true }]}
            extra="任务错过执行时间的处理策略"
          >
            <Select options={misfirePolicyOptions} />
          </Form.Item>
          <Form.Item
            label="并发执行"
            name="concurrent"
            valuePropName="checked"
            extra="是否允许同一任务同时执行多个实例"
          >
            <Switch checkedChildren="允许" unCheckedChildren="禁止" />
          </Form.Item>
          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true }]}
          >
            <Select options={statusOptions} />
          </Form.Item>
          <Form.Item
            label="备注"
            name="remark"
          >
            <TextArea placeholder="请输入备注" rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 任务日志抽屉 */}
      <Drawer
        title={`任务执行日志 - ${selectedJob?.jobName}`}
        placement="right"
        onClose={() => setLogDrawerVisible(false)}
        open={logDrawerVisible}
        size={800}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchJobLogs(selectedJob?.jobName, selectedJob?.jobGroup)}>
              刷新
            </Button>
          </Space>
        }
      >
        <div className="mb-4">
          <Row gutter={16}>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="总执行次数"
                  value={jobLogs.length}
                  prefix={<ClockCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="成功次数"
                  value={jobLogs.filter(log => log.status === 0).length}
                  styles={{ content: { color: '#3f8600' } }}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="失败次数"
                  value={jobLogs.filter(log => log.status === 1).length}
                  styles={{ content: { color: '#cf1322' } }}
                />
              </Card>
            </Col>
          </Row>
        </div>

        <Table
          columns={jobLogColumns}
          dataSource={jobLogs}
          rowKey="id"
          size="small"
          pagination={{
            pageSize: 50,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      </Drawer>
    </div>
  );
};

export default JobManager;

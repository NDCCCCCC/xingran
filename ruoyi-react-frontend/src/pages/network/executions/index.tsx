import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Card,
  Steps,
  Progress,
  Tag,
  Drawer,
  Row,
  Col,
  Statistic,
  Popconfirm,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlayCircleOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  EyeOutlined,
  StopOutlined,
  CodeOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import type { NetworkDevice, ConfigTemplate, ConfigExecution, ConfigExecutionDetail } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;
const { Step } = Steps;

const ConfigExecutionPage: React.FC = () => {
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [templates, setTemplates] = useState<ConfigTemplate[]>([]);
  const [executions, setExecutions] = useState<ConfigExecution[]>([]);
  const [loading, setLoading] = useState(false);
  const [execLoading, setExecLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [execTotal, setExecTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [execCurrent, setExecCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [executeModalVisible, setExecuteModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [executeForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [currentExecution, setCurrentExecution] = useState<ConfigExecution | null>(null);
  const [executionDetails, setExecutionDetails] = useState<ConfigExecutionDetail[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<ConfigTemplate | null>(null);
  const [renderedContent, setRenderedContent] = useState('');
  const [variableModalVisible, setVariableModalVisible] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    pending: 0,
    running: 0,
    success: 0,
    failed: 0,
  });

  // 加载设备列表
  const loadDevices = async () => {
    try {
      const result = await post('/network/devices/list', {
        current: 1,
        pageSize: 1000,
        status: 0, // 只显示在线设备
      });
      setDevices(result.data.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    }
  };

  // 加载模板列表
  const loadTemplates = async () => {
    try {
      const result = await post('/network/templates/list', {
        current: 1,
        pageSize: 1000,
        templateType: 'config',
      });
      setTemplates(result.data.list || []);
    } catch (error) {
      console.error('加载模板列表失败:', error);
    }
  };

  // 加载执行记录列表
  const loadExecutions = async (params: any = {}) => {
    setExecLoading(true);
    try {
      const result = await post('/network/executions/list', {
        current: params.current || execCurrent,
        pageSize: params.pageSize || pageSize,
      });
      setExecutions(result.data.list);
      setExecTotal(result.data.total);
      // 更新统计数据
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载执行记录失败:', error);
    } finally {
      setExecLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/executions/list', { current: 1, pageSize: 10000 });
      const allExecutions = result.data.list || [];
      updateStatistics(allExecutions);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (executionList: ConfigExecution[]) => {
    const pendingCount = executionList.filter(e => e.status === 'pending').length;
    const runningCount = executionList.filter(e => e.status === 'running').length;
    const successCount = executionList.filter(e => e.status === 'success').length;
    const failedCount = executionList.filter(e => e.status === 'failed').length;
    setStatistics({
      total: executionList.length,
      pending: pendingCount,
      running: runningCount,
      success: successCount,
      failed: failedCount,
    });
  };

  // 加载执行明细
  const loadExecutionDetails = async (executionId: string) => {
    try {
      const result = await post(`/network/executions/${executionId}`, {});
      setCurrentExecution(result.data);
      setExecutionDetails(result.data.details || []);
    } catch (error) {
      message.error('加载执行明细失败');
    }
  };

  // 模板选择变化
  const handleTemplateChange = (templateId: string) => {
    const template = templates.find(t => t.id === templateId);
    if (template) {
      setSelectedTemplate(template);
      // 解析模板变量
      if (template.variables && Object.keys(template.variables).length > 0) {
        setVariableModalVisible(true);
      }
    }
  };

  // 通过模板执行配置
  const handleExecuteByTemplate = async () => {
    try {
      const values = await executeForm.validateFields();
      const data = {
        executionName: values.executionName,
        templateId: values.templateId,
        deviceIds: selectedRowKeys as string[],
        templateVariables: values.templateVariables || {},
      };
      await post('/network/executions/template/execute', data);
      message.success('配置执行任务已创建');
      setExecuteModalVisible(false);
      executeForm.resetFields();
      setSelectedRowKeys([]);
      setSelectedTemplate(null);
      loadExecutions();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('创建执行任务失败');
    }
  };

  // 取消执行
  const handleCancelExecution = async (id: string) => {
    try {
      await post(`/network/executions/${id}/cancel`, {});
      message.success('取消成功');
      loadExecutions();
    } catch (error) {
      message.error('取消失败');
    }
  };

  // 查看执行详情
  const handleViewDetail = (record: ConfigExecution) => {
    loadExecutionDetails(record.id);
    setDetailDrawerVisible(true);
  };

  // 打开执行模态框
  const openExecuteModal = () => {
    loadDevices();
    loadTemplates();
    setSelectedRowKeys([]);
    setSelectedTemplate(null);
    executeForm.resetFields();
    setExecuteModalVisible(true);
  };

  // 执行状态选项
  const statusOptions = [
    { label: '待执行', value: 'pending' },
    { label: '执行中', value: 'running' },
    { label: '成功', value: 'success' },
    { label: '失败', value: 'failed' },
  ];

  // 设备表格列
  const deviceColumns: ColumnsType<NetworkDevice> = [
    {
      title: '设备名称',
      dataIndex: 'deviceName',
      key: 'deviceName',
    },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress' },
    { title: '厂商', dataIndex: 'vendor', key: 'vendor' },
    { title: '型号', dataIndex: 'model', key: 'model' },
  ];

  // 执行记录表格列
  const executionColumns: ColumnsType<ConfigExecution> = [
    { title: '任务名称', dataIndex: 'executionName', key: 'executionName', width: 200 },
    { title: '模板名称', dataIndex: 'templateName', key: 'templateName', width: 150 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config: Record<string, { color: string; icon: React.ReactNode; text: string }> = {
          pending: { color: 'default', icon: <ClockCircleOutlined />, text: '待执行' },
          running: { color: 'processing', icon: <ClockCircleOutlined />, text: '执行中' },
          success: { color: 'success', icon: <CheckCircleOutlined />, text: '成功' },
          failed: { color: 'error', icon: <CloseCircleOutlined />, text: '失败' },
        };
        const { color, icon, text } = config[status] || config.pending;
        return <Tag color={color} icon={icon}>{text}</Tag>;
      },
    },
    { title: '设备总数', dataIndex: 'totalDevices', key: 'totalDevices', width: 100 },
    { title: '成功数', dataIndex: 'successCount', key: 'successCount', width: 80 },
    { title: '失败数', dataIndex: 'failureCount', key: 'failureCount', width: 80 },
    {
      title: '进度',
      key: 'progress',
      width: 150,
      render: (_, record) => {
        const percent = record.totalDevices > 0
          ? Math.round(((record.successCount + record.failureCount) / record.totalDevices) * 100)
          : 0;
        return <Progress percent={percent} size="small" status={record.status === 'failed' ? 'exception' : undefined} />;
      },
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 180,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleViewDetail(record)}>查看</Button>
          {(record.status === 'pending' || record.status === 'running') && (
            <Popconfirm title="确认取消执行?" onConfirm={() => handleCancelExecution(record.id)}>
              <Button type="link" size="small" danger icon={<StopOutlined />}>取消</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // 执行明细表格列
  const detailColumns: ColumnsType<ConfigExecutionDetail> = [
    { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName', width: 150 },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress', width: 140 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config: Record<string, { color: string; text: string }> = {
          pending: { color: 'default', text: '待执行' },
          running: { color: 'processing', text: '执行中' },
          success: { color: 'success', text: '成功' },
          failed: { color: 'error', text: '失败' },
        };
        const { color, text } = config[status] || config.pending;
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '输出',
      key: 'output',
      width: 100,
      render: (_, record) => (
        record.outputReceived ? (
          <Button type="link" size="small" onClick={() => {
            Modal.info({
              title: '配置输出',
              width: 800,
              content: (
                <pre style={{ maxHeight: 500, overflow: 'auto', background: '#f5f5f5', padding: 12, whiteSpace: 'pre-wrap' }}>
                  {record.outputReceived}
                </pre>
              ),
            });
          }}>查看</Button>
        ) : '-'
      ),
    },
    { title: '开始时间', dataIndex: 'startedAt', key: 'startedAt', width: 180 },
    { title: '完成时间', dataIndex: 'completedAt', key: 'completedAt', width: 180 },
    { title: '耗时(秒)', dataIndex: 'duration', key: 'duration', width: 100 },
    {
      title: '错误信息',
      dataIndex: 'errorMessage',
      key: 'errorMessage',
      ellipsis: true,
      render: (msg: string) => msg ? (
        <Tooltip title={msg}>
          <span style={{ color: '#ff4d4f' }}>{msg}</span>
        </Tooltip>
      ) : '-',
    },
  ];

  useEffect(() => {
    loadExecutions();
    loadStatistics();
  }, [execCurrent, pageSize]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic
              title="执行总数"
              value={statistics.total}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="待执行"
              value={statistics.pending}
              styles={{ content: { color: '#999' } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="成功"
              value={statistics.success}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: '#ff4d4f' } }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }} onChange={() => { loadExecutions(); loadStatistics(); }}>
                {statusOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button icon={<SearchOutlined />} onClick={() => { loadExecutions(); loadStatistics(); }}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadExecutions(); loadStatistics(); }}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={openExecuteModal}>
              执行配置模板
            </Button>
          </Space>
        </div>
      </Card>

      {/* 执行记录表格 */}
      <Card>
        <Table
          columns={executionColumns}
          dataSource={executions}
          loading={execLoading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={{
            current: execCurrent,
            pageSize,
            total: execTotal,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => { setExecCurrent(page); setPageSize(size || 10); },
          }}
        />
      </Card>

      {/* 执行配置模态框 */}
      <Modal
        title="执行配置模板"
        open={executeModalVisible}
        onOk={handleExecuteByTemplate}
        onCancel={() => setExecuteModalVisible(false)}
        width={900}
        okButtonProps={{ disabled: selectedRowKeys.length === 0 || !selectedTemplate }}
      >
        <Form form={executeForm} layout="vertical">
          <Form.Item
            name="executionName"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>

          <Form.Item name="templateId" label="选择模板" rules={[{ required: true, message: '请选择模板' }]}>
            <Select
              placeholder="请选择配置模板"
              showSearch
              optionFilterProp="children"
              onChange={handleTemplateChange}
            >
              {templates.map(t => (
                <Option key={t.id} value={t.id}>
                  {t.templateName} ({t.templateCode})
                </Option>
              ))}
            </Select>
          </Form.Item>

          {selectedTemplate && (
            <Card size="small" style={{ marginBottom: 16 }}>
              <div><strong>模板预览:</strong></div>
              <pre style={{ maxHeight: 200, overflow: 'auto', background: '#fafafa', padding: 12, marginTop: 8 }}>
                {selectedTemplate.templateContent}
              </pre>
            </Card>
          )}

          <Form.Item label="选择目标设备" required>
            <Table
              rowSelection={{
                selectedRowKeys,
                onChange: setSelectedRowKeys,
                type: 'checkbox',
              }}
              columns={deviceColumns}
              dataSource={devices}
              rowKey="id"
              pagination={false}
              size="small"
              scroll={{ y: 200 }}
            />
            <div style={{ marginTop: 8, color: '#999' }}>
              已选择 {selectedRowKeys.length} 台设备
            </div>
          </Form.Item>

          <Form.Item
            name="timeout"
            label="超时时间(秒)"
            initialValue={300}
            rules={[{ required: true }]}
          >
            <Input type="number" min={10} max={3600} style={{ width: 200 }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 变量输入模态框 */}
      <Modal
        title="模板变量"
        open={variableModalVisible}
        onOk={() => setVariableModalVisible(false)}
        onCancel={() => setVariableModalVisible(false)}
        width={600}
      >
        <Form form={executeForm} layout="vertical">
          {selectedTemplate?.variables && Object.entries(selectedTemplate.variables).map(([key, info]: [string, any]) => (
            <Form.Item
              key={key}
              name={['templateVariables', key]}
              label={key}
              extra={info?.description || ''}
            >
              <Input placeholder={`默认值: ${info?.default || '-'}`} />
            </Form.Item>
          ))}
        </Form>
      </Modal>

      {/* 执行明细抽屉 */}
      <Drawer
        title={`执行明细 - ${currentExecution?.executionName || ''}`}
        placement="right"
        width={1000}
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
      >
        {currentExecution && (
          <div>
            <Card size="small" style={{ marginBottom: 16 }}>
              <Steps
                current={currentExecution.status === 'pending' ? 0 : currentExecution.status === 'running' ? 1 : 2}
                status={currentExecution.status === 'failed' ? 'error' : undefined}
                size="small"
              >
                <Step title="待执行" />
                <Step title="执行中" />
                <Step title="已完成" />
              </Steps>
              <div style={{ marginTop: 16 }}>
                <Progress
                  percent={currentExecution.totalDevices > 0
                    ? Math.round(((currentExecution.successCount + currentExecution.failureCount) / currentExecution.totalDevices) * 100)
                    : 0}
                  success={{ successCount: currentExecution.successCount }}
                  status={currentExecution.status === 'failed' ? 'exception' : undefined}
                />
                <div style={{ marginTop: 8, display: 'flex', gap: 24 }}>
                  <span>总设备: {currentExecution.totalDevices}</span>
                  <span style={{ color: '#52c41a' }}>成功: {currentExecution.successCount}</span>
                  <span style={{ color: '#ff4d4f' }}>失败: {currentExecution.failureCount}</span>
                </div>
              </div>
            </Card>

            <Table
              columns={detailColumns}
              dataSource={executionDetails}
              rowKey="id"
              scroll={{ x: 1200 }}
              pagination={false}
              size="small"
            />
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default ConfigExecutionPage;

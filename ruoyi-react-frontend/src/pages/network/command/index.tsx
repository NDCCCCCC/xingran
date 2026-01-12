import { useState, useEffect } from 'react';
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
  ThunderboltOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  EyeOutlined,
  StopOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import type { NetworkDevice, ConfigExecution, ConfigExecutionDetail } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

import type { FC } from 'react';

const CommandDispatch: FC = () => {
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [executions, setExecutions] = useState<ConfigExecution[]>([]);
  const [execLoading, setExecLoading] = useState(false);
  const [execTotal, setExecTotal] = useState(0);
  const [execCurrent, setExecCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [dispatchModalVisible, setDispatchModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [dispatchForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [currentExecution, setCurrentExecution] = useState<ConfigExecution | null>(null);
  const [executionDetails, setExecutionDetails] = useState<ConfigExecutionDetail[]>([]);

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

  // 加载执行记录列表
  const loadExecutions = async (params: any = {}) => {
    setExecLoading(true);
    try {
      const result = await post('/network/command/list', {
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
      const result = await post('/network/command/list', { current: 1, pageSize: 10000 });
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
      const result = await post(`/network/command/${executionId}`, {});
      setCurrentExecution(result.data);
      setExecutionDetails(result.data.details || []);
    } catch (error) {
      message.error('加载执行明细失败');
    }
  };

  // 快速命令分发
  const handleQuickCommand = async () => {
    try {
      const values = await dispatchForm.validateFields();
      await post('/network/command/quick', values);
      message.success('命令已分发，请查看执行进度');
      setDispatchModalVisible(false);
      dispatchForm.resetFields();
      setSelectedRowKeys([]);
      loadExecutions();
      loadStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('分发失败');
    }
  };

  // 查看执行详情
  const handleViewDetail = (record: ConfigExecution) => {
    loadExecutionDetails(record.id);
    setDetailDrawerVisible(true);
  };

  // 取消执行
  const handleCancelExecution = async (id: string) => {
    try {
      await post(`/network/command/${id}/cancel`, {});
      message.success('取消成功');
      loadExecutions();
      loadStatistics();
    } catch (error) {
      message.error('取消失败');
    }
  };

  // 打开命令分发模态框
  const openDispatchModal = () => {
    loadDevices();
    setSelectedRowKeys([]);
    dispatchForm.resetFields();
    setDispatchModalVisible(true);
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
  ];

  // 执行记录表格列
  const executionColumns: ColumnsType<ConfigExecution> = [
    { title: '任务名称', dataIndex: 'executionName', key: 'executionName', width: 200 },
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
      title: '命令内容',
      dataIndex: 'commandSent',
      key: 'commandSent',
      ellipsis: true,
      render: (command: string) => command || '-',
    },
    { title: '开始时间', dataIndex: 'startedAt', key: 'startedAt', width: 180 },
    { title: '完成时间', dataIndex: 'completedAt', key: 'completedAt', width: 180 },
    { title: '耗时(秒)', dataIndex: 'duration', key: 'duration', width: 100 },
    {
      title: '输出',
      key: 'output',
      width: 100,
      render: (_, record) => (
        record.outputReceived ? (
          <Button type="link" size="small" onClick={() => {
            Modal.info({
              title: '命令输出',
              width: 700,
              content: (
                <pre style={{ maxHeight: 400, overflow: 'auto', background: '#f5f5f5', padding: 12 }}>
                  {record.outputReceived}
                </pre>
              ),
            });
          }}>查看</Button>
        ) : '-'
      ),
    },
    {
      title: '错误信息',
      dataIndex: 'errorMessage',
      key: 'errorMessage',
      width: 150,
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
        <Col span={5}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
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

      {/* 主要内容 */}
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Form layout="inline">
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
                <Button type="primary" icon={<ThunderboltOutlined />} onClick={openDispatchModal}>
                  快速命令
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </div>

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

      {/* 命令分发模态框 */}
      <Modal
        title="快速命令分发"
        open={dispatchModalVisible}
        onOk={handleQuickCommand}
        onCancel={() => setDispatchModalVisible(false)}
        width={900}
        okButtonProps={{ disabled: selectedRowKeys.length === 0 }}
      >
        <Form form={dispatchForm} layout="vertical">
          <Form.Item label="选择设备" required>
            <Table
              rowSelection={{
                selectedRowKeys,
                onChange: (keys) => setSelectedRowKeys(keys as string[]),
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
            name="commandContent"
            label="命令内容"
            rules={[{ required: true, message: '请输入命令内容' }]}
            extra="每行一条命令，支持标准网络设备命令"
          >
            <TextArea
              rows={8}
              placeholder={`示例：
display version
display interface
display ip routing-table

!
show version
show running-config
!`}
            />
          </Form.Item>

          <Form.Item name="timeout" label="超时时间(秒)" initialValue={300} rules={[{ required: true }]}>
            <Input type="number" min={10} max={3600} />
          </Form.Item>
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
                items={[
                  { title: '待执行' },
                  { title: '执行中' },
                  { title: '已完成' },
                ]}
              />
              <div style={{ marginTop: 16 }}>
                <Progress
                  percent={currentExecution.totalDevices > 0
                    ? Math.round(((currentExecution.successCount + currentExecution.failureCount) / currentExecution.totalDevices) * 100)
                    : 0}
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

export default CommandDispatch;

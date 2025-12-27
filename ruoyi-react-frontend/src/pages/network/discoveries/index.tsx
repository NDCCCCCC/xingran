import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Switch,
  message,
  Card,
  Tag,
  Progress,
  Popconfirm,
  Row,
  Col,
  Statistic,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  EyeOutlined,
  DeleteOutlined,
  ImportOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { DeviceDiscovery, Department, NetworkDevice } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

// IPRange 接口定义
interface IPRange {
  startIP: string;
  endIP: string;
}

const DeviceDiscoveryPage: React.FC = () => {
  const [discoveries, setDiscoveries] = useState<DeviceDiscovery[]>([]);
  const [discoveredDevices, setDiscoveredDevices] = useState<NetworkDevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [resultModalVisible, setResultModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [currentDiscovery, setCurrentDiscovery] = useState<DeviceDiscovery | null>(null);
  const [pollingTimer, setPollingTimer] = useState<NodeJS.Timeout | null>(null);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    pending: 0,
    running: 0,
    completed: 0,
    failed: 0,
    totalDevices: 0,
  });

  // 加载发现任务列表
  const loadDiscoveries = async (params: any = {}) => {
    setLoading(true);
    try {
      const result = await post('/network/discoveries/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
      });
      setDiscoveries(result.data.list);
      setTotal(result.data.total);
      // 更新统计数据
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载发现任务失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/discoveries/list', { current: 1, pageSize: 10000 });
      const allDiscoveries = result.data.list || [];
      updateStatistics(allDiscoveries);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (discoveryList: DeviceDiscovery[]) => {
    const pendingCount = discoveryList.filter(d => d.status === 'pending').length;
    const runningCount = discoveryList.filter(d => d.status === 'running').length;
    const completedCount = discoveryList.filter(d => d.status === 'completed').length;
    const failedCount = discoveryList.filter(d => d.status === 'failed').length;
    const totalDevices = discoveryList.reduce((sum, d) => sum + (d.discoveredCount || 0), 0);
    setStatistics({
      total: discoveryList.length,
      pending: pendingCount,
      running: runningCount,
      completed: completedCount,
      failed: failedCount,
      totalDevices,
    });
  };

  // 加载部门列表
  const loadDepartments = async () => {
    try {
      const result = await post('/system/departments/list', { current: 1, pageSize: 1000 });
      setDepartments(result.data.list || []);
    } catch (error) {
      console.error('加载部门列表失败:', error);
    }
  };

  // 解析IP范围字符串为IPRange数组
  const parseIPRanges = (text: string): IPRange[] => {
    const ranges: IPRange[] = [];
    const lines = text.split('\n').filter(line => line.trim());

    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed.includes('-')) {
        // 格式: 192.168.1.1-192.168.1.100
        const [start, end] = trimmed.split('-').map(s => s.trim());
        ranges.push({ startIP: start, endIP: end });
      } else if (trimmed.includes('/')) {
        // 格式: 192.168.1.0/24 - 简化处理，转换为起始和结束IP
        const [ip, cidr] = trimmed.split('/');
        const prefix = parseInt(cidr);
        const parts = ip.split('.').map(Number);

        if (prefix >= 24) {
          // /24 网络
          const startIP = `${parts[0]}.${parts[1]}.${parts[2]}.1`;
          const endIP = `${parts[0]}.${parts[1]}.${parts[2]}.254`;
          ranges.push({ startIP, endIP });
        } else if (prefix >= 16) {
          // /16 网络
          const startIP = `${parts[0]}.${parts[1]}.0.1`;
          const endIP = `${parts[0]}.${parts[1]}.255.254`;
          ranges.push({ startIP, endIP });
        } else {
          // 其他情况，使用整个IP范围作为示例
          ranges.push({ startIP: ip, endIP: ip });
        }
      } else {
        // 单个IP
        ranges.push({ startIP: trimmed, endIP: trimmed });
      }
    }

    return ranges;
  };

  // 创建发现任务
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();

      // 解析IP范围
      const ipRangesText = values.ipRanges || '';
      const ipRanges = parseIPRanges(ipRangesText);

      if (ipRanges.length === 0) {
        message.error('请输入有效的IP范围');
        return;
      }

      // 构建请求数据
      const requestData = {
        taskName: values.taskName,
        discoveryType: values.discoveryType,
        ipRanges: ipRanges,
        snmpCommunity: values.snmpCommunity || '',
        snmpPort: values.snmpPort || 161,
        groupId: values.groupId || null,
        autoImport: values.autoImport || false,
      };

      await post('/network/discoveries/create', requestData);
      message.success('发现任务已创建');
      setModalVisible(false);
      form.resetFields();
      loadDiscoveries();
      loadStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      console.error('创建任务失败:', error);
      message.error('创建任务失败');
    }
  };

  // 删除任务
  const handleDelete = async (id: string) => {
    try {
      await post(`/network/discoveries/${id}/delete`, {});
      message.success('删除成功');
      loadDiscoveries();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 查看发现结果
  const handleViewResult = async (record: DeviceDiscovery) => {
    setCurrentDiscovery(record);
    try {
      const result = await post(`/network/discoveries/${record.id}/results`, {});
      setDiscoveredDevices(result.data.devices || []);
      setResultModalVisible(true);
    } catch (error) {
      message.error('获取结果失败');
    }
  };

  // 导入发现的设备
  const handleImport = async () => {
    if (!currentDiscovery) return;
    try {
      await post(`/network/discoveries/${currentDiscovery.id}/import`, {});
      message.success('导入成功');
      setResultModalVisible(false);
      loadDiscoveries();
    } catch (error) {
      message.error('导入失败');
    }
  };

  // 打开创建模态框
  const openModal = () => {
    form.resetFields();
    form.setFieldsValue({ snmpPort: 161, discoveryType: 'snmp' });
    loadDepartments();
    setModalVisible(true);
  };

  // 轮询进行中的任务
  const startPolling = () => {
    if (pollingTimer) clearInterval(pollingTimer);
    const timer = setInterval(() => {
      const runningTasks = discoveries.filter(d => d.status === 'running');
      if (runningTasks.length === 0) {
        clearInterval(timer);
        setPollingTimer(null);
        return;
      }
      loadDiscoveries();
    }, 3000);
    setPollingTimer(timer);
  };

  // 发现类型选项
  const discoveryTypeOptions = [
    { label: 'SNMP扫描', value: 'snmp' },
    { label: '网络扫描', value: 'scan' },
  ];

  // 发现状态选项
  const statusOptions = [
    { label: '待执行', value: 'pending' },
    { label: '扫描中', value: 'running' },
    { label: '已完成', value: 'completed' },
    { label: '失败', value: 'failed' },
  ];

  // 发现任务表格列
  const discoveryColumns: ColumnsType<DeviceDiscovery> = [
    { title: '任务名称', dataIndex: 'taskName', key: 'taskName', width: 200 },
    {
      title: '发现类型',
      dataIndex: 'discoveryType',
      key: 'discoveryType',
      width: 100,
      render: (type: string) => {
        const option = discoveryTypeOptions.find(o => o.value === type);
        return <Tag color="blue">{option?.label}</Tag>;
      },
    },
    {
      title: 'IP范围',
      dataIndex: 'ipRanges',
      key: 'ipRanges',
      width: 200,
      render: (ranges: string[]) => ranges.join(', '),
    },
    { title: '总IP数', dataIndex: 'totalIPs', key: 'totalIPs', width: 100 },
    { title: '发现数', dataIndex: 'discoveredCount', key: 'discoveredCount', width: 100 },
    {
      title: '进度',
      key: 'progress',
      width: 150,
      render: (_, record) => {
        if (record.status === 'completed') {
          return <Progress percent={100} size="small" />;
        }
        if (record.status === 'running') {
          const percent = record.totalIPs > 0
            ? Math.round((record.discoveredCount / record.totalIPs) * 100)
            : 0;
          return <Progress percent={percent} size="small" status="active" />;
        }
        return '-';
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config: Record<string, { color: string; text: string }> = {
          pending: { color: 'default', text: '待执行' },
          running: { color: 'processing', text: '扫描中' },
          completed: { color: 'success', text: '已完成' },
          failed: { color: 'error', text: '失败' },
        };
        const { color, text } = config[status] || config.pending;
        return <Tag color={color}>{text}</Tag>;
      },
    },
    { title: '开始时间', dataIndex: 'startedAt', key: 'startedAt', width: 180 },
    { title: '完成时间', dataIndex: 'completedAt', key: 'completedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 180,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleViewResult(record)} disabled={record.status === 'pending'}>
            查看结果
          </Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 发现结果表格列
  const deviceColumns: ColumnsType<NetworkDevice> = [
    { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName' },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress' },
    { title: '厂商', dataIndex: 'vendor', key: 'vendor' },
    { title: '型号', dataIndex: 'model', key: 'model' },
    { title: '设备类型', dataIndex: 'deviceType', key: 'deviceType' },
  ];

  useEffect(() => {
    loadDiscoveries();
    loadStatistics();
    return () => {
      if (pollingTimer) clearInterval(pollingTimer);
    };
  }, [current, pageSize]);

  useEffect(() => {
    // 检查是否有运行中的任务，如果有则开始轮询
    const runningTasks = discoveries.filter(d => d.status === 'running');
    if (runningTasks.length > 0 && !pollingTimer) {
      startPolling();
    }
  }, [discoveries]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card>
            <Statistic
              title="任务总数"
              value={statistics.total}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
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
              title="扫描中"
              value={statistics.running}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="已完成"
              value={statistics.completed}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: '#ff4d4f' } }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="发现设备"
              value={statistics.totalDevices}
              prefix={<ImportOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }} onChange={() => { loadDiscoveries(); loadStatistics(); }}>
                {statusOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button icon={<SearchOutlined />} onClick={() => { loadDiscoveries(); loadStatistics(); }}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadDiscoveries(); loadStatistics(); }}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={openModal}>
              创建发现任务
            </Button>
          </Space>
        </div>
      </Card>

      {/* 发现任务表格 */}
      <Card>
        <Table
          columns={discoveryColumns}
          dataSource={discoveries}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); },
          }}
        />
      </Card>

      {/* 创建发现任务模态框 */}
      <Modal
        title="创建设备发现任务"
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => setModalVisible(false)}
        width={700}
      >
        <Form form={form} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="taskName" label="任务名称" rules={[{ required: true, message: '请输入任务名称' }]}>
            <Input placeholder="请输入任务名称" />
          </Form.Item>

          <Form.Item name="discoveryType" label="发现类型" rules={[{ required: true }]}>
            <Select>
              {discoveryTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="ipRanges" label="IP范围" rules={[{ required: true, message: '请输入IP范围' }]} extra="每行一个IP范围，支持CIDR格式">
            <TextArea
              rows={4}
              placeholder={`示例：
192.168.1.0/24
192.168.2.1-192.168.2.100
10.0.0.1-10.0.0.50`}
            />
          </Form.Item>

          {Form.useWatch('discoveryType', form) === 'snmp' && (
            <>
              <Form.Item name="snmpCommunity" label="SNMP Community" rules={[{ required: true, message: '请输入SNMP Community' }]}>
                <Input placeholder="请输入SNMP Community，如：public" />
              </Form.Item>
              <Form.Item name="snmpPort" label="SNMP端口" rules={[{ required: true }]}>
                <InputNumber min={1} max={65535} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}

          <Form.Item name="groupId" label="导入部门">
            <Select placeholder="请选择部门" allowClear>
              {departments.map(dept => (
                <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="autoImport" label="自动导入" valuePropName="checked" extra="自动导入发现的设备到设备列表">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 发现结果模态框 */}
      <Modal
        title={`发现结果 - ${currentDiscovery?.taskName || ''}`}
        open={resultModalVisible}
        onCancel={() => setResultModalVisible(false)}
        width={900}
        footer={[
          <Button key="close" onClick={() => setResultModalVisible(false)}>关闭</Button>,
          <Button
            key="import"
            type="primary"
            icon={<ImportOutlined />}
            onClick={handleImport}
            disabled={discoveredDevices.length === 0}
          >
            导入设备 ({discoveredDevices.length})
          </Button>,
        ]}
      >
        <div style={{ marginBottom: 16 }}>
          <Space>
            <span>总IP数: {currentDiscovery?.totalIPs || 0}</span>
            <span>发现设备: {discoveredDevices.length}</span>
          </Space>
        </div>

        <Table
          columns={deviceColumns}
          dataSource={discoveredDevices}
          rowKey={(record) => record.ipAddress}
          pagination={false}
          size="small"
          scroll={{ y: 400 }}
        />
      </Modal>
    </div>
  );
};

export default DeviceDiscoveryPage;

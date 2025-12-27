import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  message,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Layout,
  Divider,
  Alert,
  Spin,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import type { NetworkDevice, Department, AuthCredential } from '@/types';
import { post } from '@/lib/api';
import DeptTree from '@/components/DeptTree';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';

const { Option } = Select;
const { TextArea } = Input;
const { Sider, Content } = Layout;

// 时间格式化函数
const formatDateTime = (dateTime: string | undefined) => {
  if (!dateTime) return '-';
  return dayjs(dateTime).format('YYYY-MM-DD HH:mm:ss');
};

const DeviceManagement: React.FC = () => {
  const navigate = useNavigate();
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [quickCreateModalVisible, setQuickCreateModalVisible] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [quickCreateForm] = Form.useForm();
  const [editingDevice, setEditingDevice] = useState<NetworkDevice | null>(null);
  const [viewingDevice, setViewingDevice] = useState<NetworkDevice | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [credentials, setCredentials] = useState<AuthCredential[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<string>('');

  // 快速创建相关状态
  const [probing, setProbing] = useState(false);
  const [probeResult, setProbeResult] = useState<any>(null);
  const [creating, setCreating] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    online: 0,
    offline: 0,
    unknown: 0,
  });

  // 加载设备列表
  const loadDevices = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/network/devices/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
        ...params,
      });
      setDevices(result.data.list);
      setTotal(result.data.total);
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 部门树选择处理
  const handleDeptSelect = (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
      loadDevices({ deptId: selectedKeys[0] });
    } else {
      setSelectedDeptId('');
      loadDevices();
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/devices/list', { current: 1, pageSize: 10000 });
      const allDevices = result.data.list || [];
      updateStatistics(allDevices);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (deviceList: NetworkDevice[]) => {
    const onlineCount = deviceList.filter(d => d.status === 0).length;
    const offlineCount = deviceList.filter(d => d.status === 1).length;
    const unknownCount = deviceList.filter(d => d.status === 2).length;
    setStatistics({
      total: deviceList.length,
      online: onlineCount,
      offline: offlineCount,
      unknown: unknownCount,
    });
  };

  // 加载部门列表
  const loadDepartments = async () => {
    try {
      const result = await post('/system/departments/list');
      setDepartments(result.data || []);
    } catch (error) {
      console.error('加载部门列表失败:', error);
      setDepartments([]);
    }
  };

  // 渲染树形部门选项
  const renderDeptTreeOptions = (departments: any[]) => {
    if (!departments || departments.length === 0) {
      return [<Option key="" value="">暂无部门数据</Option>];
    }

    const renderOptions = (nodes: any[], level = 0): React.ReactElement[] => {
      const options: React.ReactElement[] = [];

      nodes.forEach(node => {
        options.push(
          <Option key={node.id} value={node.id}>
            {'　'.repeat(level * 2)}{node.deptName}
          </Option>
        );

        if (node.children && node.children.length > 0) {
          options.push(...renderOptions(node.children, level + 1));
        }
      });

      return options;
    };

    return renderOptions(departments);
  };

  // 加载授权凭证列表
  const loadCredentials = async () => {
    try {
      const result = await post('/network/credentials/list', { current: 1, pageSize: 1000 });
      setCredentials(result.data.list || []);
    } catch (error) {
      console.error('加载授权凭证列表失败:', error);
    }
  };

  // 探测设备
  const handleProbe = async () => {
    try {
      const values = await quickCreateForm.validateFields(['ipAddress', 'credentialId']);
      setProbing(true);
      setProbeResult(null);

      // 只传递 IP 地址和凭证 ID，SNMP 配置从凭证中自动获取
      const result = await post('/network/devices/discover', {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
      });

      if (result.data?.success) {
        setProbeResult(result.data);
        message.success('设备探测成功');
      } else {
        message.error(result.data?.message || '探测失败');
      }
    } catch (error: any) {
      if (error.errorFields) {
        message.warning('请先填写 IP 地址和授权凭证');
      } else {
        message.error('探测失败');
      }
    } finally {
      setProbing(false);
    }
  };

  // 快速创建设备
  const handleQuickCreate = async () => {
    try {
      const values = await quickCreateForm.validateFields();
      setCreating(true);

      await post('/network/devices/quick-create', {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
        snmpPort: values.snmpPort || 161,
        communities: values.communities || [],
        deptId: values.deptId || undefined,
        location: values.location || '',
        description: values.description || '',
      });

      message.success('设备创建成功');
      setQuickCreateModalVisible(false);
      quickCreateForm.resetFields();
      setProbeResult(null);

      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error(error?.data?.message || '创建失败');
    } finally {
      setCreating(false);
    }
  };

  // 创建/更新设备
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingDevice) {
        await post(`/network/devices/${editingDevice.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/network/devices', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      editForm.resetFields();
      setEditingDevice(null);
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败');
    }
  };

  // 删除设备
  const handleDelete = async (id: string) => {
    try {
      await post(`/network/devices/${id}/delete`, {});
      message.success('删除成功');
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除设备
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/network/devices/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: NetworkDevice) => {
    if (record) {
      setEditingDevice(record);
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
        credentialId: record.credentialId || undefined,
      });
    } else {
      setEditingDevice(null);
      editForm.resetFields();
      editForm.setFieldsValue({ port: 22, snmpPort: 161, status: 0 });
    }
    setModalVisible(true);
    loadDepartments();
    loadCredentials();
  };

  // 打开快速创建模态框
  const openQuickCreateModal = () => {
    quickCreateForm.resetFields();
    setProbeResult(null);
    setQuickCreateModalVisible(true);
    loadDepartments();
    loadCredentials();
  };

  // 打开详情模态框
  const openDetailModal = async (record: NetworkDevice) => {
    setViewingDevice(record);
    setDetailModalVisible(true);
  };

  // 搜索
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        searchParams[key] = value;
      }
    });
    if (selectedDeptId) {
      searchParams.deptId = selectedDeptId;
    }
    loadDevices(searchParams);
  };

  // 重置
  const handleReset = () => {
    searchForm.resetFields();
    if (selectedDeptId) {
      loadDevices({ deptId: selectedDeptId });
    } else {
      loadDevices();
    }
  };

  // 刷新
  const handleRefresh = () => {
    if (selectedDeptId) {
      loadDevices({ deptId: selectedDeptId });
    } else {
      loadDevices();
    }
    loadStatistics();
  };

  // 设备厂商选项
  const vendorOptions = [
    { label: 'Huawei', value: 'huawei' },
    { label: 'H3C', value: 'h3c' },
    { label: 'Ruijie', value: 'ruijie' },
    { label: 'Maipu', value: 'maipu' },
  ];

  // 设备类型选项
  const deviceTypeOptions = [
    { label: '路由器', value: 'router' },
    { label: '交换机', value: 'switch' },
    { label: '防火墙', value: 'firewall' },
    { label: '无线AP', value: 'ap' },
    { label: '负载均衡器', value: 'loadbalancer' },
  ];

  // 设备状态选项
  const statusOptions = [
    { label: '在线', value: 0 },
    { label: '离线', value: 1 },
    { label: '未知', value: 2 },
  ];

  // 表格列
  const columns: ColumnsType<NetworkDevice> = [
    {
      title: '设备名称',
      dataIndex: 'deviceName',
      key: 'deviceName',
      width: 200,
      render: (deviceName: string, record: NetworkDevice) => (
        <a
          onClick={() => navigate(`/network/ports?deviceId=${record.id}`)}
          style={{ color: '#1890ff', cursor: 'pointer' }}
        >
          {deviceName}
        </a>
      ),
    },
    {
      title: '设备类型',
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 120,
      render: (deviceType: string) => {
        const option = deviceTypeOptions.find(o => o.value === deviceType);
        return <Tag color="blue">{option?.label}</Tag>;
      },
    },
    {
      title: '厂商',
      dataIndex: 'vendor',
      key: 'vendor',
      width: 100,
      render: (vendor: string) => {
        const option = vendorOptions.find(o => o.value === vendor);
        return <Tag color="geekblue">{option?.label}</Tag>;
      },
    },
    { title: '型号', dataIndex: 'model', key: 'model', width: 150 },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress', width: 140 },
    { title: '端口', dataIndex: 'port', key: 'port', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: number) => {
        const option = statusOptions.find(o => o.value === status);
        const color = status === 0 ? 'success' : status === 1 ? 'error' : 'default';
        return <Tag color={color}>{option?.label}</Tag>;
      },
    },
    { title: '部门', dataIndex: 'deptName', key: 'deptName', width: 180 },
    {
      title: '最后连接',
      dataIndex: 'lastSeenAt',
      key: 'lastSeenAt',
      width: 180,
      render: (lastSeenAt: string) => formatDateTime(lastSeenAt),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (createdAt: string) => formatDateTime(createdAt),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 180,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => openDetailModal(record)}>详情</Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>编辑</Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadDevices();
    loadStatistics();
  }, [current, pageSize]);

  return (
    <Layout style={{ background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      {/* 左侧部门树 */}
      <Sider width={280} style={{ background: '#fff', padding: '16px', borderRight: '1px solid #f0f0f0' }}>
        <DeptTree
          onSelect={(selectedKeys) => handleDeptSelect(selectedKeys)}
          selectedKeys={selectedDeptId ? [selectedDeptId] : []}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ padding: '16px', background: '#f0f2f5' }}>
        <div>
          {/* 统计卡片 */}
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="设备总数"
                  value={statistics.total}
                  prefix={<CloudServerOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="在线设备"
                  value={statistics.online}
                  styles={{ content: { color: '#3f8600' } }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="离线设备"
                  value={statistics.offline}
                  styles={{ content: { color: '#cf1322' } }}
                  prefix={<CloseCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="未知状态"
                  value={statistics.unknown}
                  styles={{ content: { color: '#faad14' } }}
                  prefix={<WarningOutlined />}
                />
              </Card>
            </Col>
          </Row>

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
              <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item name="deviceName" label="设备名称">
                  <Input placeholder="请输入设备名称" allowClear style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="ipAddress" label="IP地址">
                  <Input placeholder="请输入IP地址" allowClear style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="vendor" label="厂商">
                  <Select placeholder="请选择厂商" allowClear style={{ width: 120 }}>
                    {vendorOptions.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item name="deviceType" label="设备类型">
                  <Select placeholder="请选择设备类型" allowClear style={{ width: 130 }}>
                    {deviceTypeOptions.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" allowClear style={{ width: 100 }}>
                    {statusOptions.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                <Button type="primary" icon={<ThunderboltOutlined />} onClick={openQuickCreateModal}>快速创建</Button>
                <Button type="default" icon={<PlusOutlined />} onClick={() => openModal()}>手动新增</Button>
                <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
              </Space>
            </div>
          </Card>

          {/* 设备表格 */}
          <Card>
            <Table
              rowSelection={{
                selectedRowKeys,
                onChange: setSelectedRowKeys,
              }}
              columns={columns}
              dataSource={devices}
              loading={loading}
              rowKey="id"
              scroll={{ x: 1600 }}
              pagination={{
                current,
                pageSize,
                total,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total) => `共 ${total} 条`,
                onChange: (page, size) => {
                  setCurrent(page);
                  setPageSize(size || 10);
                  const params: any = { current: page, pageSize: size };
                  if (selectedDeptId) {
                    params.deptId = selectedDeptId;
                  }
                  loadDevices(params);
                },
              }}
            />
          </Card>
        </div>
      </Content>

      {/* 快速创建模态框 */}
      <Modal
        title="快速创建设备"
        open={quickCreateModalVisible}
        onOk={handleQuickCreate}
        onCancel={() => { setQuickCreateModalVisible(false); quickCreateForm.resetFields(); setProbeResult(null); }}
        width={700}
        okText="创建设备"
        confirmLoading={creating}
      >
        <Alert
          message="快速创建说明"
          description="只需输入 IP 地址和授权凭证，系统会自动使用凭证中配置的 SNMP Communities 探测获取设备信息并创建设备。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={quickCreateForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="ipAddress" label="IP地址" rules={[{ required: true, message: '请输入IP地址' }]}>
            <Input placeholder="请输入设备IP地址" />
          </Form.Item>
          <Form.Item name="credentialId" label="授权凭证" rules={[{ required: true, message: '请选择授权凭证' }]}>
            <Select placeholder="请选择授权凭证">
              {credentials.map(cred => (
                <Option key={cred.id} value={cred.id}>{cred.credentialName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label=" ">
            <Col offset={6}>
              <Space>
                <Button type="primary" icon={<ApiOutlined />} onClick={handleProbe} loading={probing}>
                  探测设备
                </Button>
                <Button onClick={() => setProbeResult(null)}>清除结果</Button>
              </Space>
            </Col>
          </Form.Item>

          {probeResult && (
            <>
              <Divider orientation="left">探测结果</Divider>
              <Alert
                message="探测成功"
                type="success"
                style={{ marginBottom: 16 }}
              />
              <Row gutter={16}>
                <Col span={12}>
                  <Card size="small" title="设备信息">
                    <p><strong>设备名称:</strong> {probeResult.deviceName || '-'}</p>
                    <p><strong>设备类型:</strong> {probeResult.deviceType || '-'}</p>
                    <p><strong>厂商:</strong> {probeResult.vendor || '-'}</p>
                    <p><strong>型号:</strong> {probeResult.model || '-'}</p>
                  </Card>
                </Col>
                <Col span={12}>
                  <Card size="small" title="系统信息">
                    <p><strong>SysName:</strong> {probeResult.sysName || '-'}</p>
                    <p><strong>SysDescr:</strong> {probeResult.sysDescr?.substring(0, 50) || '-'}{probeResult.sysDescr?.length > 50 ? '...' : ''}</p>
                  </Card>
                </Col>
              </Row>
            </>
          )}

          <Divider orientation="left">其他信息（可选）</Divider>
          <Form.Item name="deptId" label="所属部门">
            <Select placeholder="请选择部门" allowClear>
              {renderDeptTreeOptions(departments)}
            </Select>
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入设备位置" />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={2} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 手动新增/编辑模态框 */}
      <Modal
        title={editingDevice ? '编辑设备' : '手动新增设备'}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); editForm.resetFields(); setEditingDevice(null); }}
        width={700}
      >
        <Alert
          message="手动新增说明"
          description="需要手动填写所有设备信息。如果只想输入IP地址自动获取信息，请使用「快速创建」功能。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="deviceName" label="设备名称" rules={[{ required: true, message: '请输入设备名称' }]}>
            <Input placeholder="请输入设备名称" />
          </Form.Item>
          <Form.Item name="deviceType" label="设备类型" rules={[{ required: true, message: '请选择设备类型' }]}>
            <Select placeholder="请选择设备类型">
              {deviceTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="vendor" label="厂商" rules={[{ required: true, message: '请选择厂商' }]}>
            <Select placeholder="请选择厂商">
              {vendorOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="model" label="型号">
            <Input placeholder="请输入设备型号" />
          </Form.Item>
          <Form.Item name="ipAddress" label="IP地址" rules={[{ required: true, message: '请输入IP地址' }]}>
            <Input placeholder="请输入IP地址" />
          </Form.Item>
          <Form.Item name="port" label="SSH端口" rules={[{ required: true, message: '请输入SSH端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="snmpPort" label="SNMP端口" rules={[{ required: true, message: '请输入SNMP端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="credentialId" label="授权凭证">
            <Select placeholder="请选择授权凭证" allowClear>
              {credentials.map(cred => (
                <Option key={cred.id} value={cred.id}>{cred.credentialName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="deptId" label="所属部门">
            <Select placeholder="请选择部门" allowClear>
              {renderDeptTreeOptions(departments)}
            </Select>
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入位置" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              {statusOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 设备详情模态框 */}
      <Modal
        title="设备详细信息"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>关闭</Button>,
        ]}
        width={800}
      >
        {viewingDevice && (
          <div>
            <Divider orientation="left">基本信息</Divider>
            <Row gutter={16}>
              <Col span={12}>
                <p><strong>设备名称:</strong> {viewingDevice.deviceName}</p>
                <p><strong>设备类型:</strong> {
                  deviceTypeOptions.find(o => o.value === viewingDevice.deviceType)?.label || '-'
                }</p>
                <p><strong>厂商:</strong> {
                  vendorOptions.find(o => o.value === viewingDevice.vendor)?.label || '-'
                }</p>
                <p><strong>型号:</strong> {viewingDevice.model || '-'}</p>
                <p><strong>IP地址:</strong> {viewingDevice.ipAddress}</p>
                <p><strong>SSH端口:</strong> {viewingDevice.port}</p>
                <p><strong>SNMP端口:</strong> {viewingDevice.snmpPort}</p>
              </Col>
              <Col span={12}>
                <p><strong>状态:</strong> {
                  statusOptions.find(o => o.value === viewingDevice.status)?.label || '-'
                }</p>
                <p><strong>所属部门:</strong> {viewingDevice.deptName || '-'}</p>
                <p><strong>位置:</strong> {viewingDevice.location || '-'}</p>
                <p><strong>授权凭证:</strong> {viewingDevice.credentialName || '-'}</p>
                <p><strong>最后连接:</strong> {viewingDevice.lastSeenAt || '-'}</p>
                <p><strong>创建时间:</strong> {viewingDevice.createdAt}</p>
              </Col>
            </Row>

            <Divider orientation="left">设备详细信息（SSH采集）</Divider>
            <Row gutter={16}>
              <Col span={12}>
                <p><strong>型号:</strong> {viewingDevice.model || '-'}</p>
                <p><strong>序列号:</strong> {viewingDevice.serialNumber || '-'}</p>
              </Col>
              <Col span={12}>
                <p><strong>软件版本:</strong> {viewingDevice.softwareVersion || '-'}</p>
                <p><strong>运行时间:</strong> {viewingDevice.uptime || '-'}</p>
              </Col>
            </Row>

            {viewingDevice.description && (
              <>
                <Divider orientation="left">备注</Divider>
                <p>{viewingDevice.description}</p>
              </>
            )}
          </div>
        )}
      </Modal>
    </Layout>
  );
};

export default DeviceManagement;

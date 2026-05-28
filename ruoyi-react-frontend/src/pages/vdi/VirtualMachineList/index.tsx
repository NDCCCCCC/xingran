import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  message,
  Popconfirm,
  Select,
  Input,
  Card,
  Tooltip,
  Form,
  Slider,
} from 'antd';
import {
  ReloadOutlined,
  PlayCircleOutlined,
  StopOutlined,
  SyncOutlined,
  DeleteOutlined,
  PlusOutlined,
  EditOutlined,
  SettingOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { vmApi, vdiServerApi } from '@/lib/vdiApi';
import type { VirtualMachine, VMListParams, CreateVMRequest, VDIServer, VDIResourceGroup } from '@/types/vdi';
import type { ColumnsType } from 'antd/es/table';

const VirtualMachineList: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [vms, setVMs] = useState<VirtualMachine[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [filters, setFilters] = useState<Partial<VMListParams>>({});
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [configIPModalVisible, setConfigIPModalVisible] = useState(false);
  const [renameModalVisible, setRenameModalVisible] = useState(false);
  const [bindUserModalVisible, setBindUserModalVisible] = useState(false);
  const [selectedVM, setSelectedVM] = useState<VirtualMachine | null>(null);
  const [form] = Form.useForm();
  const [vdiServers, setVdiServers] = useState<VDIServer[]>([]);
  const [resourceGroups, setResourceGroups] = useState<VDIResourceGroup[]>([]);

  // 加载虚拟机列表
  const loadVMs = async () => {
    setLoading(true);
    try {
      const params: VMListParams = {
        current,
        pageSize,
        ...filters,
      };
      const result = await vmApi.list(params);
      setVMs(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      message.error('加载虚拟机列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVMs();
  }, [current, pageSize, filters]);

  // Watch form field changes for cascading dropdowns and auto-name
  const selectedServerId = Form.useWatch('vdi_server_id', form);
  const selectedResourceId = Form.useWatch('resource_id', form);

  // Load resource groups when VDI server is selected
  useEffect(() => {
    if (selectedServerId && createModalVisible) {
      vmApi.listResourceGroups(selectedServerId).then(result => {
        setResourceGroups(result.data || []);
      }).catch(() => {
        setResourceGroups([]);
      });
    } else {
      setResourceGroups([]);
    }
  }, [selectedServerId, createModalVisible]);

  // Auto-generate name when resource group is selected
  useEffect(() => {
    if (selectedResourceId) {
      const group = resourceGroups.find(g => g.resource_group_id === selectedResourceId);
      if (group) {
        const suffix = form.getFieldValue('name_suffix') || '';
        form.setFieldsValue({ name: group.name + (suffix ? '-' + suffix : '') });
      }
    }
  }, [selectedResourceId, resourceGroups, form]);

  // Update name when suffix changes
  const handleSuffixChange = useCallback((suffix: string) => {
    const selectedResource = form.getFieldValue('resource_id');
    const group = resourceGroups.find(g => g.resource_group_id === selectedResource);
    if (group) {
      form.setFieldsValue({ name: group.name + (suffix ? '-' + suffix : '') });
    }
  }, [resourceGroups, form]);

  // Open create modal and load dropdown data
  const openCreateModal = async () => {
    setCreateModalVisible(true);
    form.resetFields();
    try {
      const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
      setVdiServers(serverResult.data?.list || []);
    } catch (e) {
      // ignore
    }
  };

  // 创建虚拟机（调用 VDI API）
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const data: CreateVMRequest = {
        ...values,
      };
      await vmApi.create(data);
      message.success('虚拟机创建成功，VDI API 调用完成');
      setCreateModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('创建虚拟机失败');
    } finally {
      setLoading(false);
    }
  };

  // 操作虚拟机（调用 VDI API）
  const handleOperate = async (action: string, vmIds?: string[]) => {
    const ids = vmIds || selectedRowKeys as string[];
    if (ids.length === 0) {
      message.warning('请选择要操作的虚拟机');
      return;
    }

    setLoading(true);
    try {
      await vmApi.batchOperate(ids, action);
      message.success(`${action === 'start' ? '开机' : action === 'stop' ? '关机' : '重启'}操作已提交，VDI API 调用成功`);
      setSelectedRowKeys([]);
      loadVMs();
    } catch (error) {
      message.error('操作失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 删除虚拟机（调用 VDI API）
  const handleDelete = async (id: string) => {
    setLoading(true);
    try {
      await vmApi.delete(id);
      message.success('删除成功，VDI API 调用完成');
      loadVMs();
    } catch (error) {
      message.error('删除失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 同步虚拟机（调用 VDI API）
  const handleSync = async (id: string) => {
    setLoading(true);
    try {
      await vmApi.sync(id);
      message.success('同步成功，VDI 状态已更新');
      loadVMs();
    } catch (error) {
      message.error('同步失败');
    } finally {
      setLoading(false);
    }
  };

  // 配置 IP（调用 VDI API）
  const handleConfigIP = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.configIP([{
        vm_id: selectedVM!.vm_id,
        ip_address: values.ip_address,
        netmask: values.netmask,
        gateway: values.gateway,
      }]);
      message.success('IP 配置成功，VDI API 调用完成');
      setConfigIPModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('配置 IP 失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 重命名（调用 VDI API）
  const handleRename = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.rename(selectedVM!.id, { new_name: values.new_name });
      message.success('重命名成功，VDI API 调用完成');
      setRenameModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('重命名失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 绑定用户（调用 VDI API）
  const handleBindUser = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.bindUser(selectedVM!.id, { user_id: values.user_id });
      message.success('用户绑定成功，VDI API 调用完成');
      setBindUserModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('绑定用户失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义（包含所有操作按钮）
  const columns: ColumnsType<VirtualMachine> = [
    {
      title: '虚拟机 ID',
      dataIndex: 'vm_id',
      key: 'vm_id',
      width: 120,
      ellipsis: true,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (text: string, record) => (
        <a
          onClick={() => navigate(`/vdi/vm/${record.id}`)}
          style={{ color: '#1890ff', textDecoration: 'underline' }}
        >
          {text}
        </a>
      ),
    },
    {
      title: '虚拟机状态',
      dataIndex: 'power_state',
      key: 'power_state',
      width: 100,
      render: (state: string) => {
        const config = {
          pending: { color: 'blue', text: '等待Agent' },
          stopped: { color: 'red', text: '关机' },
          suspended: { color: 'orange', text: '挂起' },
          in_use: { color: 'lime', text: '正常使用' },
        };
        const { color, text } = config[state as keyof typeof config] || { color: 'default', text: state };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => {
        return status === 0 ? (
          <Tag color="green">正常</Tag>
        ) : (
          <Tag color="red">停用</Tag>
        );
      },
    },
    {
      title: 'IP 地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 120,
      render: (ip: string, record: any) => {
        const ipType = record.ip_type || 'DHCP';
        return ip ? (
          <Tooltip title={`类型: ${ipType === 'STATIC' ? '静态IP' : 'DHCP'}${record.subnet_mask ? `\n子网掩码: ${record.subnet_mask}` : ''}${record.default_gateway ? `\n网关: ${record.default_gateway}` : ''}${record.name_server ? `\nDNS: ${record.name_server}` : ''}`}>
            <span>{ip} ({ipType === 'STATIC' ? '静态' : 'DHCP'})</span>
          </Tooltip>
        ) : '-';
      },
    },
    {
      title: '操作系统',
      dataIndex: 'os_type',
      key: 'os_type',
      width: 120,
    },
    {
      title: 'CPU/内存/磁盘',
      key: 'resources',
      width: 200,
      render: (_, record) => {
        const cpuNumber = record.cpu_number || 0;
        const cpuCore = record.cpu_core || 0;
        const cpuPer = record.cpu_per || 0;
        const memoryMB = parseInt(String(record.memory || '0'), 10);
        const memoryPer = record.memory_per || 0;
        const diskMB = parseInt(String(record.disk || '0'), 10);
        const diskPer = record.disk_per || 0;

        // 转换为 GB
        const memoryGB = (memoryMB / 1024).toFixed(1);
        const diskGB = (diskMB / 1024).toFixed(1);

        // 构建CPU显示字符串
        let cpuDisplay = '';
        if (cpuNumber > 0 && cpuCore > 0) {
          cpuDisplay = `${cpuNumber}颗 × ${cpuCore}核`;
          if (cpuPer > 0) {
            cpuDisplay += ` (${cpuPer}%)`;
          }
        } else if (cpuNumber > 0) {
          cpuDisplay = `${cpuNumber}核`;
        } else {
          cpuDisplay = '-';
        }

        // 构建内存显示字符串
        let memoryDisplay = `${memoryGB}GB`;
        if (memoryPer > 0) {
          memoryDisplay += ` (${memoryPer}%)`;
        }

        // 构建磁盘显示字符串
        let diskDisplay = `${diskGB}GB`;
        if (diskPer > 0) {
          diskDisplay += ` (${diskPer}%)`;
        }

        return `${cpuDisplay} / ${memoryDisplay} / ${diskDisplay}`;
      },
    },
    {
      title: '绑定用户',
      dataIndex: 'bound_user_name',
      key: 'bound_user_name',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: '最后同步',
      dataIndex: 'last_sync_at',
      key: 'last_sync_at',
      width: 160,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 300,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="开机">
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => handleOperate('start', [record.id])}
              disabled={record.power_state === 'running'}
            />
          </Tooltip>
          <Tooltip title="关机">
            <Button
              type="text"
              icon={<StopOutlined />}
              onClick={() => handleOperate('stop', [record.id])}
              disabled={record.power_state === 'stopped'}
            />
          </Tooltip>
          <Tooltip title="重启">
            <Button
              type="text"
              icon={<SyncOutlined />}
              onClick={() => handleOperate('restart', [record.id])}
              disabled={record.power_state !== 'running'}
            />
          </Tooltip>
          <Tooltip title="同步">
            <Button
              type="text"
              icon={<SyncOutlined />}
              onClick={() => handleSync(record.id)}
            />
          </Tooltip>
          <Tooltip title="配置 IP">
            <Button
              type="text"
              icon={<SettingOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setConfigIPModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="重命名">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setRenameModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="绑定用户">
            <Button
              type="text"
              icon={<UserAddOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setBindUserModalVisible(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个虚拟机吗？此操作将调用 VDI API 删除虚拟机。"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        {/* 工具栏 */}
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建虚拟机
          </Button>
          <Input.Search
            placeholder="搜索虚拟机名称"
            style={{ width: 200 }}
            onSearch={(value) => setFilters({ ...filters, name: value })}
          />
          <Select
            placeholder="虚拟机状态"
            style={{ width: 120 }}
            allowClear
            onChange={(value) => setFilters({ ...filters, powerState: value })}
          >
            <Select.Option value="pending">等待Agent</Select.Option>
            <Select.Option value="stopped">关机</Select.Option>
            <Select.Option value="suspended">挂起</Select.Option>
            <Select.Option value="in_use">正常使用</Select.Option>
          </Select>
          <Button icon={<ReloadOutlined />} onClick={loadVMs}>
            刷新
          </Button>
        </Space>

        {/* 批量操作 */}
        {selectedRowKeys.length > 0 && (
          <Space>
            <span>已选择 {selectedRowKeys.length} 项</span>
            <Button onClick={() => handleOperate('start')}>批量开机</Button>
            <Button onClick={() => handleOperate('stop')}>批量关机</Button>
            <Button onClick={() => handleOperate('restart')}>批量重启</Button>
          </Space>
        )}

        {/* 表格 */}
        <Table
          columns={columns}
          dataSource={vms}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1400 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
            },
          }}
        />
      </Space>

      {/* 创建虚拟机模态框 */}
      <Modal
        title="创建虚拟机"
        open={createModalVisible}
        onOk={handleCreate}
        onCancel={() => setCreateModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="VDI 服务器" name="vdi_server_id" rules={[{ required: true, message: '请选择VDI服务器' }]}>
            <Select placeholder="请选择VDI服务器" loading={vdiServers.length === 0}>
              {vdiServers.filter(s => s.status === 0).map(server => (
                <Select.Option key={server.id} value={server.id}>
                  {server.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item label="资源组" name="resource_id" rules={[{ required: true, message: '请选择资源组' }]}>
            <Select placeholder={selectedServerId ? "请选择资源组" : "请先选择VDI服务器"} disabled={!selectedServerId}>
              {resourceGroups.map(group => (
                <Select.Option key={group.resource_group_id} value={group.resource_group_id}>
                  {group.name} {group.type ? `(${group.type})` : ''}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item label="虚拟机名称" name="name" rules={[{ required: true, message: '请输入虚拟机名称' }]}>
            <Input placeholder="选择资源组后自动生成" />
          </Form.Item>
          <Form.Item label="名称后缀（可选）">
            <Input
              placeholder="自定义后缀，例如：user01"
              onChange={(e) => handleSuffixChange(e.target.value)}
            />
          </Form.Item>

          <Form.Item label="CPU 颗数" name="cpu_number" initialValue={1}>
            <Slider min={1} max={16} marks={{ 1: '1', 4: '4', 8: '8', 16: '16' }} />
          </Form.Item>

          <Form.Item label="每颗 CPU 核数" name="cpu_core" initialValue={4}>
            <Slider min={1} max={32} marks={{ 1: '1', 8: '8', 16: '16', 32: '32' }} />
          </Form.Item>

          <Form.Item label="内存" name="memory" initialValue={4096}>
            <Slider
              min={512}
              max={65536}
              step={512}
              marks={{ 512: '0.5GB', 4096: '4GB', 8192: '8GB', 16384: '16GB', 32768: '32GB', 65536: '64GB' }}
              tooltip={{ formatter: (v) => v ? `${(v / 1024).toFixed(1)} GB` : '' }}
            />
          </Form.Item>

          <Form.Item label="磁盘" name="disk" initialValue={60}>
            <Slider
              min={20}
              max={500}
              step={10}
              marks={{ 20: '20GB', 100: '100GB', 200: '200GB', 500: '500GB' }}
              tooltip={{ formatter: (v) => v ? `${v} GB` : '' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 配置 IP 模态框 */}
      <Modal
        title="配置 IP 地址"
        open={configIPModalVisible}
        onOk={handleConfigIP}
        onCancel={() => setConfigIPModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="IP 地址" name="ip_address" rules={[{ required: true }]}>
            <Input placeholder="192.168.1.100" />
          </Form.Item>
          <Form.Item label="子网掩码" name="netmask" rules={[{ required: true }]}>
            <Input placeholder="255.255.255.0" />
          </Form.Item>
          <Form.Item label="网关" name="gateway" rules={[{ required: true }]}>
            <Input placeholder="192.168.1.1" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 重命名模态框 */}
      <Modal
        title="重命名虚拟机"
        open={renameModalVisible}
        onOk={handleRename}
        onCancel={() => setRenameModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="新名称" name="new_name" rules={[{ required: true }]}>
            <Input placeholder="新虚拟机名称" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 绑定用户模态框 */}
      <Modal
        title="绑定用户"
        open={bindUserModalVisible}
        onOk={handleBindUser}
        onCancel={() => setBindUserModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="用户 ID" name="user_id" rules={[{ required: true }]}>
            <Input placeholder="用户 ID" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default VirtualMachineList;
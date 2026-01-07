import React, { useState, useEffect } from 'react';
import {
  Button,
  Table,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  Space,
  message,
  Popconfirm,
  Card,
  Tag,
  Tooltip,
  Progress
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getADConfigList,
  createADConfig,
  updateADConfig,
  deleteADConfig,
  testADConnection,
  syncADData,
  type ADConfig,
  type ADSyncResult
} from '@/lib/adDomainApi';

const ADConfigPage: React.FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState<ADConfig | null>(null);

  const [testingConfig, setTestingConfig] = useState<string | null>(null);
  const [syncingConfig, setSyncingConfig] = useState<string | null>(null);
  const [syncProgress, setSyncProgress] = useState<{ [key: string]: ADSyncResult }>({});

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const res = await getADConfigList({ current: 1, pageSize: 100 });
      if (res.code === 0) {
        setConfigs(res.data?.list ?? []);
      }
    } catch (error) {
      message.error('获取AD配置失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingConfig(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      serverPort: 389,
      useSsl: false,
      useTls: false,
      syncEnabled: true,
      syncInterval: 3600,
    });
    setModalVisible(true);
  };

  const handleEdit = (config: ADConfig) => {
    setEditingConfig(config);
    editForm.setFieldsValue({
      ...config,
      adminPassword: undefined, // 不回填密码
    });
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await editForm.validateFields();

      if (editingConfig) {
        const updateData = {
          ...values,
          adminPassword: values.adminPassword || undefined, // 如果没有输入密码则不更新
        };
        await updateADConfig(editingConfig.id, updateData);
        message.success('更新成功');
      } else {
        await createADConfig(values);
        message.success('创建成功');
      }

      setModalVisible(false);
      fetchConfigs();
    } catch (error: any) {
      message.error(error.message || '操作失败');
    }
  };

  const handleTest = async (configId: string) => {
    setTestingConfig(configId);
    try {
      await testADConnection(configId);
      message.success('连接测试成功');
    } catch (error: any) {
      message.error('连接测试失败: ' + error.message);
    } finally {
      setTestingConfig(null);
    }
  };

  const handleSync = async (configId: string) => {
    setSyncingConfig(configId);
    try {
      const res = await syncADData(configId, 'full');
      if (res.code === 0 && res.data?.result) {
        const result = res.data.result;
        message.success(`同步成功: OU=${result.ouCount}, 用户组=${result.groupCount}, 用户=${result.userCount}`);
        setSyncProgress(prev => ({ ...prev, [configId]: result }));
        fetchConfigs();
      }
    } catch (error: any) {
      message.error('同步失败: ' + error.message);
    } finally {
      setSyncingConfig(null);
    }
  };

  const handleDelete = async (configId: string) => {
    try {
      await deleteADConfig(configId);
      message.success('删除成功');
      fetchConfigs();
    } catch (error: any) {
      message.error('删除失败: ' + error.message);
    }
  };

  const columns: ColumnsType<ADConfig> = [
    {
      title: '配置名称',
      dataIndex: 'configName',
      key: 'configName',
    },
    {
      title: '服务器地址',
      dataIndex: 'serverAddress',
      key: 'serverAddress',
      render: (text: string, record: ADConfig) => `${text}:${record.serverPort}`,
    },
    {
      title: '域名',
      dataIndex: 'domainName',
      key: 'domainName',
    },
    {
      title: '基础DN',
      dataIndex: 'baseDn',
      key: 'baseDn',
      ellipsis: true,
    },
    {
      title: '安全连接',
      key: 'security',
      render: (_: any, record: ADConfig) => (
        <Space>
          {record.useSsl && <Tag color="blue">SSL</Tag>}
          {record.useTls && <Tag color="green">TLS</Tag>}
          {!record.useSsl && !record.useTls && <Tag>Plain</Tag>}
        </Space>
      ),
    },
    {
      title: '自动同步',
      key: 'syncEnabled',
      render: (_: any, record: ADConfig) => (
        <Tag color={record.syncEnabled ? 'success' : 'default'}>
          {record.syncEnabled ? `是 (${record.syncInterval}s)` : '否'}
        </Tag>
      ),
    },
    {
      title: '最后同步',
      dataIndex: 'lastSyncAt',
      key: 'lastSyncAt',
      render: (text: string | null) => text ? new Date(text).toLocaleString('zh-CN') : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'default'}>
          {status === 0 ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      title: '同步结果',
      key: 'syncResult',
      render: (_: any, record: ADConfig) => {
        const result = syncProgress[record.id];
        if (!result) return '-';
        return (
          <Space direction="vertical" size="small">
            <span>OU: {result.ouCount}</span>
            <span>用户组: {result.groupCount}</span>
            <span>用户: {result.userCount}</span>
          </Space>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      render: (_: any, record: ADConfig) => (
        <Space size="small">
          <Tooltip title="测试连接">
            <Button
              type="link"
              size="small"
              icon={<CheckCircleOutlined />}
              loading={testingConfig === record.id}
              onClick={() => handleTest(record.id)}
            />
          </Tooltip>
          <Tooltip title="同步数据">
            <Button
              type="link"
              size="small"
              icon={<SyncOutlined />}
              loading={syncingConfig === record.id}
              onClick={() => handleSync(record.id)}
            />
          </Tooltip>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Popconfirm
            title="确定删除此配置吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新增AD配置
          </Button>
          <Button icon={<ReloadOutlined />} onClick={fetchConfigs}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={configs}
        loading={loading}
        rowKey="id"
        pagination={false}
      />

      <Modal
        title={editingConfig ? '编辑AD配置' : '新增AD配置'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            label="配置名称"
            name="configName"
            rules={[{ required: true, message: '请输入配置名称' }]}
          >
            <Input placeholder="如: 公司主AD域" />
          </Form.Item>

          <Form.Item
            label="服务器地址"
            name="serverAddress"
            rules={[{ required: true, message: '请输入AD服务器地址' }]}
          >
            <Input placeholder="如: ad.example.com 或 192.168.1.100" />
          </Form.Item>

          <Form.Item
            label="端口"
            name="serverPort"
            rules={[{ required: true, message: '请输入端口' }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            label="域名"
            name="domainName"
            rules={[{ required: true, message: '请输入域名' }]}
          >
            <Input placeholder="如: example.com" />
          </Form.Item>

          <Form.Item
            label="基础DN"
            name="baseDn"
            rules={[{ required: true, message: '请输入基础DN' }]}
          >
            <Input placeholder="如: DC=example,DC=com" />
          </Form.Item>

          <Form.Item
            label="管理员用户名"
            name="adminUsername"
            rules={[{ required: true, message: '请输入管理员用户名' }]}
          >
            <Input placeholder="如: Administrator" />
          </Form.Item>

          <Form.Item
            label="管理员密码"
            name="adminPassword"
            rules={editingConfig ? [] : [{ required: true, message: '请输入管理员密码' }]}
          >
            <Input.Password placeholder={editingConfig ? "留空表示不修改" : "请输入管理员密码"} />
          </Form.Item>

          <Form.Item label="使用SSL (LDAPS)" name="useSsl" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item label="使用TLS (StartTLS)" name="useTls" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item label="启用自动同步" name="syncEnabled" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item
            label="同步间隔(秒)"
            name="syncInterval"
          >
            <InputNumber min={60} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default ADConfigPage;

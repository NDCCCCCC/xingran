import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Select,
  Input,
  Space,
  Button,
  Tag,
  Form,
  Modal,
  message,
  Switch,
  Popconfirm
} from 'antd';
import {
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  SwapOutlined,
  EditOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getADUserList,
  updateADUser,
  moveADUser,
  enableADUser,
  disableADUser,
  getADConfigList,
  getADOUTree,
  type ADUser,
  type ADConfig,
  type ADOUNode
} from '@/lib/adDomainApi';

const ADUserPage: React.FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [selectedConfig, setSelectedConfig] = useState<string>('');

  const [users, setUsers] = useState<ADUser[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<ADUser | null>(null);

  const [moveModalVisible, setMoveModalVisible] = useState(false);
  const [movingUser, setMovingUser] = useState<ADUser | null>(null);

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    try {
      const res = await getADConfigList({ current: 1, pageSize: 100, status: 0 });
      if (res.code === 0 && res.data?.list?.length > 0) {
        setConfigs(res.data.list);
        if (!selectedConfig) {
          setSelectedConfig(res.data.list[0].id);
        }
      }
    } catch (error) {
      message.error('获取AD配置失败');
    }
  };

  const fetchOUTree = async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADOUTree(selectedConfig);
      if (res.code === 0) {
        setOUTree(res.data ?? []);
      }
    } catch (error) {
      console.error('获取OU树失败', error);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
      fetchUsers();
    }
  }, [selectedConfig, current, pageSize]);

  const fetchUsers = async () => {
    if (!selectedConfig) return;

    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const res = await getADUserList({
        configId: selectedConfig,
        username: values.username,
        isEnabled: values.isEnabled,
        current,
        pageSize,
      });
      if (res.code === 0) {
        setUsers(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      }
    } catch (error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrent(1);
    fetchUsers();
  };

  const handleEdit = (user: ADUser) => {
    setEditingUser(user);
    editForm.setFieldsValue({
      displayName: user.displayName,
      email: user.email,
      phone: user.phone,
      mobile: user.mobile,
      title: user.title,
      department: user.department,
      description: user.description,
    });
    setEditModalVisible(true);
  };

  const handleUpdateUser = async () => {
    if (!editingUser || !selectedConfig) return;

    try {
      const values = await editForm.validateFields();
      await updateADUser(editingUser.id, selectedConfig, values);
      message.success('更新成功');
      setEditModalVisible(false);
      fetchUsers();
    } catch (error: any) {
      message.error(error.message || '更新失败');
    }
  };

  const handleMove = (user: ADUser) => {
    setMovingUser(user);
    setMoveModalVisible(true);
  };

  const handleMoveConfirm = async (ouDn: string) => {
    if (!movingUser || !selectedConfig) return;

    try {
      await moveADUser(movingUser.id, selectedConfig, ouDn);
      message.success('移动成功');
      setMoveModalVisible(false);
      fetchUsers();
    } catch (error: any) {
      message.error(error.message || '移动失败');
    }
  };

  const handleEnable = async (user: ADUser) => {
    if (!selectedConfig) return;

    try {
      if (user.isEnabled) {
        await disableADUser(user.id, selectedConfig);
        message.success('禁用成功');
      } else {
        await enableADUser(user.id, selectedConfig);
        message.success('启用成功');
      }
      fetchUsers();
    } catch (error: any) {
      message.error(error.message || '操作失败');
    }
  };

  // 构建扁平化的OU选项列表
  const buildOUOptions = (nodes: ADOUNode[], level = 0): Array<{ value: string; label: string }> => {
    const options: Array<{ value: string; label: string }> = [];
    const prefix = '　'.repeat(level);

    for (const node of nodes) {
      options.push({
        value: node.dn,
        label: `${prefix}${node.name}`,
      });
      if (node.children && node.children.length > 0) {
        options.push(...buildOUOptions(node.children, level + 1));
      }
    }
    return options;
  };

  const columns: ColumnsType<ADUser> = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      fixed: 'left',
      width: 150,
    },
    {
      title: '显示名',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 150,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      width: 200,
      ellipsis: true,
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
      width: 150,
    },
    {
      title: '职位',
      dataIndex: 'title',
      key: 'title',
      width: 120,
    },
    {
      title: '办公电话',
      dataIndex: 'phone',
      key: 'phone',
      width: 130,
    },
    {
      title: '手机',
      dataIndex: 'mobile',
      key: 'mobile',
      width: 130,
    },
    {
      title: '状态',
      key: 'status',
      width: 200,
      render: (_: any, user: ADUser) => (
        <Space size="small">
          <Tag color={user.isEnabled ? 'success' : 'default'}>
            {user.isEnabled ? '启用' : '禁用'}
          </Tag>
          {user.isLocked && <Tag color="error">锁定</Tag>}
          {user.passwordExpired && <Tag color="warning">密码过期</Tag>}
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 250,
      render: (_: any, user: ADUser) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={user.isEnabled ? <StopOutlined /> : <CheckCircleOutlined />}
            onClick={() => handleEnable(user)}
          >
            {user.isEnabled ? '禁用' : '启用'}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<SwapOutlined />}
            onClick={() => handleMove(user)}
          >
            移动
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(user)}
          >
            编辑
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 搜索栏 */}
        <Form form={form} layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={setSelectedConfig}
              options={configs.map(c => ({ label: c.configName, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="username">
            <Input placeholder="用户名" allowClear style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="isEnabled">
            <Select placeholder="状态" allowClear style={{ width: 120 }}>
              <Select.Option value={true}>启用</Select.Option>
              <Select.Option value={false}>禁用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={fetchUsers}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {/* 用户列表 */}
        <Table
          columns={columns}
          dataSource={users}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size || 10);
            },
          }}
        />
      </Space>

      {/* 编辑用户弹窗 */}
      <Modal
        title="编辑用户"
        open={editModalVisible}
        onOk={handleUpdateUser}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item label="显示名" name="displayName">
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <Input />
          </Form.Item>
          <Form.Item label="办公电话" name="phone">
            <Input />
          </Form.Item>
          <Form.Item label="手机" name="mobile">
            <Input />
          </Form.Item>
          <Form.Item label="职位" name="title">
            <Input />
          </Form.Item>
          <Form.Item label="部门" name="department">
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 移动用户弹窗 */}
      <Modal
        title={`移动用户 - ${movingUser?.username}`}
        open={moveModalVisible}
        onCancel={() => setMoveModalVisible(false)}
        footer={null}
        width={500}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>选择目标OU：</div>
          <Select
            style={{ width: '100%' }}
            placeholder="请选择目标OU"
            options={buildOUOptions(ouTree)}
            onChange={(value) => {
              handleMoveConfirm(value);
            }}
          />
        </Space>
      </Modal>
    </Card>
  );
};

export default ADUserPage;

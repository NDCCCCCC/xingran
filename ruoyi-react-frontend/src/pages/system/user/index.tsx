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
  Popconfirm,
  Tag,
  Avatar,
  Card,
  Row,
  Col,
  Statistic,
  Layout,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  UserOutlined,
  TeamOutlined,
  LockOutlined,
  UnlockOutlined,
  KeyOutlined,
} from '@ant-design/icons';
import type { User } from '@/types';
import { post } from '@/lib/api';
import DeptTree from '@/components/DeptTree';

const { Option } = Select;
const { Sider, Content } = Layout;

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [resetPasswordModalVisible, setResetPasswordModalVisible] = useState(false);
  const [resetPasswordForm] = Form.useForm();
  const [resettingUser, setResettingUser] = useState<User | null>(null);
  const [selectedDeptId, setSelectedDeptId] = useState<string>('');

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 角色数据
  const [roles, setRoles] = useState<any[]>([]);

  // 部门数据
  const [departments, setDepartments] = useState<any[]>([]);

  // 加载用户列表
  const loadUsers = async (params: any = {}) => {
    setLoading(true);
    try {
      const result = await post('/system/users/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params,
      });
      setUsers(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载用户列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      // 从服务器获取统计数据
      const result = await post('/system/users/list', { current: 1, pageSize: 1000 });
      const allUsers = result.data.list || [];
      const activeCount = allUsers.filter(u => u.status === 0).length;
      const inactiveCount = allUsers.filter(u => u.status === 1).length;
      setStatistics({
        total: allUsers.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      console.error('加载统计数据失败', error);
      // 如果接口失败，使用当前用户数据统计
      const activeCount = users.filter(u => u.status === 0).length;
      const inactiveCount = users.filter(u => u.status === 1).length;
      setStatistics({
        total: users.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    }
  };

  // 加载角色列表
  const loadRoles = async () => {
    try {
      const result = await post('/system/roles/all');
      setRoles(result.data || []);
    } catch (error) {
      console.error('加载角色列表失败', error);
    }
  };

  // 加载部门列表
  const loadDepartments = async () => {
    try {
      const result = await post('/system/departments/list');
      // 直接存储树形结构数据
      setDepartments(result.data || []);
    } catch (error) {
      console.error('加载部门列表失败', error);
      setDepartments([]);
    }
  };

  // 渲染树形部门选项
  const renderDeptTreeOptions = (departments: any[]) => {
    if (!departments || departments.length === 0) {
      return [<Option key="" value="">暂无部门数据</Option>];
    }

    // 递归渲染树形选项
    const renderOptions = (nodes: any[], level = 0): React.ReactElement[] => {
      const options: React.ReactElement[] = [];

      nodes.forEach(node => {
        options.push(
          <Option key={node.id} value={node.id}>
            {'　'.repeat(level * 2)}{node.deptName}
          </Option>
        );

        // 递归渲染子部门
        if (node.children && node.children.length > 0) {
          options.push(...renderOptions(node.children, level + 1));
        }
      });

      return options;
    };

    return renderOptions(departments);
  };

  useEffect(() => {
    loadUsers();
    loadStatistics();
    loadRoles();
    loadDepartments();
  }, []);

  // 搜索
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();

    // 确保状态值是数字类型
    const searchParams: any = {};
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          // 将状态值转换为数字
          searchParams[key] = Number(value);
        } else {
          searchParams[key] = value;
        }
      }
    });

    // 如果有选中的部门，添加部门ID到搜索参数
    if (selectedDeptId) {
      searchParams.deptId = selectedDeptId;
    }

    loadUsers(searchParams);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    setSelectedDeptId('');
    loadUsers();
  };

  // 刷新
  const handleRefresh = () => {
    loadUsers();
    loadStatistics();
  };

  // 新增用户
  const handleAdd = () => {
    setEditingUser(null);
    editForm.resetFields();
    // 设置默认值
    const defaultValues: any = {
      gender: 0,
      status: 0,
    };
    // 设置第一个部门为默认值
    if (departments.length > 0) {
      defaultValues.deptId = departments[0].id;
    }
    // 设置第一个角色为默认值
    if (roles.length > 0) {
      defaultValues.roleIds = [roles[0].id];
    }
    editForm.setFieldsValue(defaultValues);
    setEditModalVisible(true);
  };

  // 编辑用户
  const handleEdit = (record: User) => {
    setEditingUser(record);
    setEditModalVisible(true);
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && editingUser) {
      // Modal 打开时设置表单值
      editForm.setFieldsValue(editingUser);
    }
  };

  // 删除用户
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/users/${id}/delete`);
      message.success('删除成功');
      loadUsers();
      loadStatistics();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的用户');
      return;
    }

    try {
      await post('/system/users/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadUsers();
      loadStatistics();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  // 更新用户状态
  const handleUpdateStatus = async (id: string, status: number) => {
    try {
      await post(`/system/users/${id}/status`, { status });
      message.success(status === 0 ? '启用成功' : '禁用成功');
      loadUsers();
      loadStatistics();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  // 保存用户
  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();

      // 确保 gender 和 status 是数字类型
      if (values.gender !== undefined) {
        values.gender = Number(values.gender);
      }
      if (values.status !== undefined) {
        values.status = Number(values.status);
      }

      const url = editingUser
        ? `/system/users/${editingUser.id}/update`
        : '/system/users';

      await post(url, values);
      message.success(editingUser ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadUsers();
      loadStatistics();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  // 重置密码
  const handleResetPassword = (user: User) => {
    setResettingUser(user);
    resetPasswordForm.resetFields();
    setResetPasswordModalVisible(true);
  };

  // 保存重置密码
  const handleSaveResetPassword = async () => {
    try {
      const values = await resetPasswordForm.validateFields();

      await post(`/system/users/${resettingUser!.id}/reset-password`, {
        password: values.password,
      });

      message.success('密码重置成功');
      setResetPasswordModalVisible(false);
    } catch (error) {
      console.error('重置密码失败:', error);
    }
  };

  // 部门树选择处理
  const handleDeptSelect = (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
      // 重新加载用户列表，筛选该部门的用户
      loadUsers({ deptId: selectedKeys[0] });
    } else {
      setSelectedDeptId('');
      // 清空部门筛选，重新加载所有用户
      loadUsers();
    }
  };

  // 表格列定义
  const columns: ColumnsType<User> = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text, record) => (
        <Space>
          <Avatar src={record.avatar} icon={<UserOutlined />} />
          {text}
        </Space>
      ),
    },
    {
      title: '昵称',
      dataIndex: 'nickname',
      key: 'nickname',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '手机号',
      dataIndex: 'phone',
      key: 'phone',
    },
    {
      title: '性别',
      dataIndex: 'gender',
      key: 'gender',
      render: (gender) => {
        switch (gender) {
          case 0: return '男';
          case 1: return '女';
          case 2: return '保密';
          default: return '保密';
        }
      },
    },
    {
      title: '部门',
      dataIndex: 'deptName',
      key: 'deptName',
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      render: (roles: string[]) => {
        // 如果用户有角色，显示角色名称，否则显示无角色
        if (!roles || roles.length === 0) {
          return <span style={{ color: '#999' }}>无角色</span>;
        }
        return (
          <Space size="small" wrap>
            {roles.map((roleName, index) => (
              <Tag key={index} color="blue">
                {roleName}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (text) => new Date(text).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            icon={record.status === 1 ? <LockOutlined /> : <UnlockOutlined />}
            onClick={() => handleUpdateStatus(record.id, record.status === 1 ? 0 : 1)}
          >
            {record.status === 0 ? '禁用' : '启用'}
          </Button>
          <Button
            type="link"
            icon={<KeyOutlined />}
            onClick={() => handleResetPassword(record)}
          >
            重置密码
          </Button>
          <Popconfirm
            title="确定要删除这个用户吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

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
            <Col span={8}>
              <Card>
                <Statistic
                  title="总用户数"
                  value={statistics.total}
                  prefix={<TeamOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="正常用户"
                  value={statistics.active}
                  styles={{ content: { color: '#3f8600' } }}
                  prefix={<UnlockOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="禁用用户"
                  value={statistics.inactive}
                  styles={{ content: { color: '#cf1322' } }}
                  prefix={<LockOutlined />}
                />
              </Card>
            </Col>
          </Row>

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
              <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item name="username" label="用户名">
                  <Input placeholder="请输入用户名" />
                </Form.Item>
                <Form.Item name="nickname" label="昵称">
                  <Input placeholder="请输入昵称" />
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                    <Option value="0">启用</Option>
                    <Option value="1">禁用</Option>
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                      刷新
                    </Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                  新增用户
                </Button>
                <Button
                  danger
                  onClick={handleBatchDelete}
                  disabled={selectedRowKeys.length === 0}
                >
                  批量删除
                </Button>
              </Space>
            </div>
          </Card>

          {/* 用户表格 */}
          <Card>
            <Table
              columns={columns}
              dataSource={users}
              rowKey="id"
              loading={loading}
              pagination={{
                current,
                pageSize,
                total,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total) => `共 ${total} 条`,
                onChange: (page, size) => {
                  setCurrent(page);
                  setPageSize(size);
                  const params: any = { current: page, pageSize: size };
                  if (selectedDeptId) {
                    params.deptId = selectedDeptId;
                  }
                  loadUsers(params);
                },
              }}
              rowSelection={{
                selectedRowKeys,
                onChange: setSelectedRowKeys,
              }}
            />
          </Card>
        </div>
      </Content>

      {/* 编辑弹窗 */}
      <Modal
        title={editingUser ? '编辑用户' : '新增用户'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input disabled={!!editingUser} />
          </Form.Item>
          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password />
            </Form.Item>
          )}
          <Form.Item
            name="nickname"
            label="昵称"
            rules={[{ required: true, message: '请输入昵称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="phone" label="手机号">
            <Input />
          </Form.Item>
          <Form.Item name="gender" label="性别" initialValue={0}>
            <Select>
              <Option value={0}>男</Option>
              <Option value={1}>女</Option>
              <Option value={2}>保密</Option>
            </Select>
          </Form.Item>
          <Form.Item name="deptId" label="部门">
            <Select placeholder="请选择部门" allowClear>
              {renderDeptTreeOptions(departments)}
            </Select>
          </Form.Item>
          <Form.Item name="roleIds" label="角色">
            <Select
              mode="multiple"
              placeholder="请选择角色"
              allowClear
              optionFilterProp="children"
              showSearch
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
            >
              {roles.map(role => (
                <Option key={role.id} value={role.id}>
                  {role.roleName}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>启用</Option>
              <Option value={1}>禁用</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码弹窗 */}
      <Modal
        title={`重置密码 - ${resettingUser?.username || ''}`}
        open={resetPasswordModalVisible}
        onOk={handleSaveResetPassword}
        onCancel={() => setResetPasswordModalVisible(false)}
        width={400}
      >
        <Form form={resetPasswordForm} layout="vertical">
          <Form.Item
            name="password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度不能少于6位' },
              { max: 20, message: '密码长度不能超过20位' },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入密码" />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default UserManagement;
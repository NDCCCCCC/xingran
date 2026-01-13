import { useState, useEffect, useCallback } from 'react';
import type { FC, Key, ReactElement } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tag,
  Avatar,
  Card,
  Row,
  Col,
  Statistic,
  Layout,
  Alert,
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
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;
const { Sider, Content } = Layout;

const UserManagement: FC = () => {
  // 自定义状态（不能迁移到 useTableManager）
  const [resetPasswordModalVisible, setResetPasswordModalVisible] = useState(false);
  const [resetPasswordForm] = Form.useForm();
  const [resettingUser, setResettingUser] = useState<User | null>(null);
  const [selectedDeptId, setSelectedDeptId] = useState<string>('');
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [roles, setRoles] = useState<any[]>([]);
  const [departments, setDepartments] = useState<any[]>([]);

  // 数据加载函数（使用 useCallback 保持引用稳定）
  const fetchUsers = useCallback(async (params: any) => {
    const result = await post('/system/users/list', params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  // 使用 useTableManager 管理表格状态
  const {
    loading,
    data: users,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingUser,
    setCurrent,
    setPageSize,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadUsers,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<User>(fetchUsers, { pageSize: 10 });

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post('/system/users/list', { current: 1, pageSize: 1000 });
      const allUsers = result.data.list || [];
      const activeCount = allUsers.filter((u: any) => u.status === 0).length;
      const inactiveCount = allUsers.filter((u: any) => u.status === 1).length;
      setStatistics({
        total: allUsers.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    try {
      const result = await post('/system/roles/all');
      setRoles(result.data || []);
    } catch (error) {
      handleApiError(error, '加载角色列表', false);
    }
  }, []);

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post('/system/departments/list');
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
      setDepartments([]);
    }
  }, []);

  // 渲染树形部门选项
  const renderDeptTreeOptions = (departments: any[]) => {
    if (!departments || departments.length === 0) {
      return [<Option key="" value="">暂无部门数据</Option>];
    }

    // 递归渲染树形选项
    const renderOptions = (nodes: any[], level = 0): ReactElement[] => {
      const options: ReactElement[] = [];

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

  // 初始化加载（只在组件挂载时执行一次）
  useEffect(() => {
    loadUsers();
    loadStatistics();
    loadRoles();
    loadDepartments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

  // 新增用户（覆盖 useTableManager 的 handleAdd）
  const handleAddUser = () => {
    handleAdd();
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
      handleSuccess('删除');
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '删除');
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }

    try {
      await post('/system/users/batch-delete', { ids: selectedRowKeys });
      handleSuccess('批量删除');
      resetSelection();
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  };

  // 更新用户状态
  const handleUpdateStatus = async (id: string, status: number) => {
    try {
      await post(`/system/users/${id}/status`, { status });
      handleSuccess(status === 0 ? '启用' : '禁用');
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '操作');
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
      handleSuccess(editingUser ? '更新' : '创建');
      setEditModalVisible(false);
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '操作');
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

      handleSuccess('密码重置');
      setResetPasswordModalVisible(false);
    } catch (error) {
      handleApiError(error, '重置密码');
    }
  };

  // 部门树选择处理
  const handleDeptSelect = (selectedKeys: Key[]) => {
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
      title: '工号',
      dataIndex: 'employeeNo',
      key: 'employeeNo',
      render: (text) => text || '-',
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
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'status',
            label: record.status === 0 ? '禁用' : '启用',
            icon: record.status === 1 ? <LockOutlined /> : <UnlockOutlined />,
            onClick: () => handleUpdateStatus(record.id, record.status === 1 ? 0 : 1),
          },
          {
            key: 'reset-password',
            label: '重置密码',
            icon: <KeyOutlined />,
            onClick: () => handleResetPassword(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个用户吗？"
                onConfirm={() => handleDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <Layout style={{ display: 'flex', alignItems: 'stretch' }}>
      {/* 左侧部门树 */}
      <Sider width={360} className="dept-list-sider" style={{ background: '#fff', padding: '0 16px 16px 0' }}>
        <DeptTree
          onSelect={(selectedKeys) => handleDeptSelect(selectedKeys)}
          selectedKeys={selectedDeptId ? [selectedDeptId] : []}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ padding: 0, background: '#fff' }}>
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
                {selectedRowKeys.length > 0 && (
                  <Button
                    danger
                    icon={<DeleteOutlined />}
                    onClick={handleBatchDelete}
                  >
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                )}
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAddUser}>
                  新增用户
                </Button>
              </Space>
            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                message={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个用户，
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setSelectedRowKeys([])}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
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
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input disabled={!!editingUser} className="user-form-input" />
          </Form.Item>
          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password className="user-form-input" />
            </Form.Item>
          )}
          <Form.Item
            name="nickname"
            label="昵称"
            rules={[{ required: true, message: '请输入昵称' }]}
          >
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="employeeNo" label="工号">
            <Input placeholder="请输入工号" className="user-form-input" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="phone" label="手机号">
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="gender" label="性别" initialValue={0}>
            <Select className="user-form-input">
              <Option value={0}>男</Option>
              <Option value={1}>女</Option>
              <Option value={2}>保密</Option>
            </Select>
          </Form.Item>
          <Form.Item name="deptId" label="部门">
            <Select placeholder="请选择部门" allowClear className="user-form-input">
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
              className="user-form-input"
            >
              {roles.map(role => (
                <Option key={role.id} value={role.id}>
                  {role.roleName}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select className="user-form-input">
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
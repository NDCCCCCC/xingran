import { useState, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  TreeSelect,
  message,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  ApartmentOutlined,
  TeamOutlined,
  PhoneOutlined,
  MailOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Department } from '@/types';
import { post, get } from '@/lib/api';

const { Option } = Select;

// 部门用户类型
interface DeptUser {
  id: string;
  username: string;
  nickname?: string;
  phone?: string;
  email?: string;
}

const DepartmentManagement: FC = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [parentOptions, setParentOptions] = useState<any[]>([]);
  const [deptUsers, setDeptUsers] = useState<DeptUser[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    topLevel: 0,
    subLevel: 0,
  });

  // 状态标签映射
  const getStatusTag = (status: number) => {
    return (
      <Tag color={status === 0 ? 'success' : 'error'}>
        {status === 0 ? '正常' : '停用'}
      </Tag>
    );
  };

  // 加载部门列表
  const loadDepartments = async (searchParams?: any) => {
    setLoading(true);
    try {
      const result = await post('/system/departments/list', searchParams || {});
      const deptData = result.data || [];
      setDepartments(deptData);

      const flatList = flattenTreeToList(deptData);

      // 组建父级选项（树形结构）
      const options = transformToParentTreeOptions(deptData);
      setParentOptions(options);

      // 统计数据
      const stats = {
        total: flatList.length,
        topLevel: flatList.filter(d => !d.parentId).length,
        subLevel: flatList.filter(d => d.parentId).length,
      };
      setStatistics(stats);
    } catch (error) {
      console.error('加载部门列表失败:', error);
      // 设置默认值
      setDepartments([]);
      setParentOptions([{ label: '顶级部门', value: '', key: '' }]);
      setStatistics({ total: 0, topLevel: 0, subLevel: 0 });
    } finally {
      setLoading(false);
    }
  };

  // 扁平化树形数据
  const flattenTreeToList = (tree: any[]): any[] => {
    const result: any[] = [];
    if (!tree || !Array.isArray(tree)) {
      return result;
    }
    const traverse = (nodes: any[]) => {
      if (!nodes || !Array.isArray(nodes)) {
        return;
      }
      nodes.forEach(node => {
        if (node) {
          result.push(node);
          if (node.children && node.children.length > 0) {
            traverse(node.children);
          }
        }
      });
    };
    traverse(tree);
    return result;
  };

  // 转换为 TreeSelect 所需的树形结构
  const transformToParentTreeOptions = (data: any[]): any[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }

    return data.map(item => ({
      title: item.deptName,
      value: item.id,
      key: item.id,
      children: item.children && item.children.length > 0 ? transformToParentTreeOptions(item.children) : undefined,
    }));
  };

  // 渲染树形表格数据
  const renderTreeData = (data: Department[]): any[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }
    return data.map(item => ({
      ...item,
      key: item.id,
      children: item.children && item.children.length > 0 ? renderTreeData(item.children) : undefined,
    }));
  };

  useEffect(() => {
    loadDepartments();
  }, []);

  // 搜索
  const handleSearch = async () => {
    const values = await searchForm.validateFields();
    loadDepartments(values);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    loadDepartments();
  };

  // 刷新
  const handleRefresh = () => {
    loadDepartments();
  };

  // 新增部门
  const handleAdd = () => {
    setEditingDept(null);
    editForm.resetFields();
    setDeptUsers([]);
    setEditModalVisible(true);
  };

  // 编辑部门
  const handleEdit = (record: Department) => {
    setEditingDept(record);
    setEditModalVisible(true);
    // 加载该部门的用户列表
    loadDeptUsers(record.id);
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && editingDept) {
      // Modal 打开时设置表单值
      editForm.setFieldsValue(editingDept);
    }
  };

  // 加载部门用户列表
  const loadDeptUsers = async (deptId: string) => {
    setLoadingUsers(true);
    try {
      const result = await get(`/system/departments/${deptId}/users`);
      setDeptUsers(result.data || []);
    } catch (error) {
      console.error('加载部门用户失败:', error);
      setDeptUsers([]);
    } finally {
      setLoadingUsers(false);
    }
  };

  // 上级部门改变时加载用户列表
  const handleParentChange = (value: string) => {
    if (value) {
      loadDeptUsers(value);
    } else {
      setDeptUsers([]);
    }
  };

  // 负责人改变时自动填充电话和邮箱
  const handleLeaderChange = (userId: string) => {
    const selectedUser = deptUsers.find(u => u.id === userId);
    if (selectedUser) {
      editForm.setFieldsValue({
        phone: selectedUser.phone || '',
        email: selectedUser.email || '',
      });
    }
  };

  // 删除部门
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/departments/${id}/delete`);
      message.success('删除成功');
      loadDepartments();
    } catch (error) {
      console.error('删除部门失败:', error);
    }
  };

  // 保存部门
  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();

      // 处理电话和邮箱字段：如果没有选择负责人，则不发送这些字段
      const submitData: any = {
        deptName: values.deptName,
        parentId: values.parentId || null,
        orderNum: values.orderNum,
        status: values.status,
        remark: values.remark || '',
      };

      // 只有选择了负责人时才发送 leader、phone、email
      if (values.leader) {
        submitData.leader = values.leader;
        submitData.phone = values.phone || '';
        submitData.email = values.email || '';
      }

      if (editingDept) {
        // 更新时需要添加 id 字段
        await post(`/system/departments/${editingDept.id}/update`, {
          ...submitData,
          id: editingDept.id,
        });
      } else {
        await post('/system/departments', submitData);
      }

      message.success(editingDept ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadDepartments();
    } catch (error) {
      console.error('保存部门失败:', error);
    }
  };

  // 表格行样式：不可访问的部门显示为灰色
  const getRowClassName = (record: Department) => {
    if (record.accessible === false) {
      return 'dept-inaccessible-row';
    }
    return '';
  };

  // 表格列定义
  const columns: ColumnsType<Department> = [
    {
      title: '部门名称',
      dataIndex: 'deptName',
      key: 'deptName',
      render: (text, record) => (
        <Space style={{ opacity: record.accessible === false ? 0.5 : 1 }}>
          <ApartmentOutlined />
          {text}
          {record.accessible === false && (
            <Tag color="default" style={{ fontSize: '12px', marginLeft: 4 }}>无权限</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '负责人',
      dataIndex: 'leader',
      key: 'leader',
      render: (_, record) => {
        if (record.leaderName && record.leaderUsername) {
          return `${record.leaderName}（${record.leaderUsername}）`;
        }
        return '-';
      },
    },
    {
      title: '联系电话',
      dataIndex: 'phone',
      key: 'phone',
      render: (phone) => phone ? (
        <Space>
          <PhoneOutlined />
          {phone}
        </Space>
      ) : '-',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      render: (email) => email ? (
        <Space>
          <MailOutlined />
          {email}
        </Space>
      ) : '-',
    },
    {
      title: '显示顺序',
      dataIndex: 'orderNum',
      key: 'orderNum',
    },
    {
      title: '外部机构',
      dataIndex: 'isExternalOrg',
      key: 'isExternalOrg',
      render: (isExternalOrg: number) => (
        <Tag color={isExternalOrg === 1 ? 'blue' : 'default'}>
          {isExternalOrg === 1 ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => getStatusTag(status),
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
            disabled={record.accessible === false}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个部门吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
            disabled={record.accessible === false}
          >
            <Button type="link" danger icon={<DeleteOutlined />} disabled={record.accessible === false}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="总部门数"
              value={statistics.total}
              prefix={<ApartmentOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="顶级部门"
              value={statistics.topLevel}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="子部门"
              value={statistics.subLevel}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApartmentOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deptName" label="部门名称">
              <Input placeholder="请输入部门名称" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value="0">正常</Option>
                <Option value="1">停用</Option>
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
              新增部门
            </Button>
          </Space>
        </div>
      </Card>

      {/* 部门表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={renderTreeData(departments)}
          rowKey="id"
          loading={loading}
          pagination={false}
          rowClassName={(record) => getRowClassName(record)}
          expandable={{
            defaultExpandAllRows: true,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingDept ? '编辑部门' : '新增部门'}
        open={editModalVisible}
        onOk={handleSave}
        onCancel={() => setEditModalVisible(false)}
        afterOpenChange={handleModalOpenChange}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="parentId"
            label="上级部门"
            initialValue=""
          >
            <TreeSelect
              style={{ width: '100%' }}
              placeholder="请选择上级部门"
              treeData={[{ title: '顶级部门', value: '', key: '' }, ...parentOptions]}
              allowClear
              showSearch
              treeDefaultExpandAll
              treeLine={{ showLeafIcon: false }}
              onChange={handleParentChange}
            />
          </Form.Item>
          <Form.Item
            name="deptName"
            label="部门名称"
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="orderNum"
            label="显示顺序"
            rules={[{ required: true, message: '请输入显示顺序' }]}
            initialValue={0}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="leader" label="负责人">
            <Select
              placeholder="请选择负责人"
              loading={loadingUsers}
              showSearch
              allowClear
              optionFilterProp="label"
              onChange={handleLeaderChange}
              notFoundContent={loadingUsers ? '加载中...' : '请先选择上级部门'}
            >
              {deptUsers.length > 0 && (
                <Option
                  key={deptUsers[0].id}
                  value={deptUsers[0].id}
                  label={`${deptUsers[0].nickname || deptUsers[0].username} (${deptUsers[0].username})`}
                >
                  <Space>
                    <UserOutlined />
                    {deptUsers[0].nickname || deptUsers[0].username}
                    <span style={{ color: '#999' }}>({deptUsers[0].username})</span>
                  </Space>
                </Option>
              )}
              {deptUsers.slice(1).map(user => (
                <Option
                  key={user.id}
                  value={user.id}
                  label={`${user.nickname || user.username} (${user.username})`}
                >
                  <Space>
                    <UserOutlined />
                    {user.nickname || user.username}
                    <span style={{ color: '#999' }}>({user.username})</span>
                  </Space>
                </Option>
              ))}
            </Select>
          </Form.Item>
          {/* 隐藏字段：电话和邮箱，通过选择负责人自动填充 */}
          <Form.Item name="phone" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="email" hidden>
            <Input />
          </Form.Item>
          <Form.Item
            name="isExternalOrg"
            label="外部机构"
            initialValue={0}
            tooltip="标记为外部机构的部门可以作为楼宇的所属机构"
          >
            <Select>
              <Option value={0}>否</Option>
              <Option value={1}>是</Option>
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DepartmentManagement;
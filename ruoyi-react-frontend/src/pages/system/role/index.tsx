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
  Card,
  Row,
  Col,
  Statistic,
  Tree,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Role } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;

const RoleManagement: React.FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [menuTreeVisible, setMenuTreeVisible] = useState(false);
  const [menuTree, setMenuTree] = useState<any[]>([]);
  const [selectedMenuIds, setSelectedMenuIds] = useState<string[]>([]);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 加载角色列表
  const loadRoles = async (params: any = {}) => {
    setLoading(true);
    try {
      const result = await post('/system/roles/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params,
      });
      setRoles(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载角色列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      // 从服务器获取统计数据
      const result = await post('/system/roles/list', { current: 1, pageSize: 1000 });
      const allRoles = result.data.list || [];
      const activeCount = allRoles.filter(r => r.status === 0).length;
      const inactiveCount = allRoles.filter(r => r.status === 1).length;
      setStatistics({
        total: allRoles.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      console.error('加载统计数据失败', error);
      // 如果接口失败，使用当前角色数据统计
      const activeCount = roles.filter(r => r.status === 0).length;
      const inactiveCount = roles.filter(r => r.status === 1).length;
      setStatistics({
        total: roles.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    }
  };

  // 加载菜单树
  const loadMenuTree = async () => {
    try {
      const result = await post('/system/menus/tree-select');
      // 处理树形数据，使其符合Antd Tree的数据格式
      const processTreeData = (nodes: any[]): any[] => {
        return nodes.map((node: any) => ({
          key: node.key || node.id,
          title: node.title || node.menuName,
          value: node.value || node.id,
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setMenuTree(processTreeData(result.data));
    } catch (error) {
      console.error('加载菜单树失败:', error);
    }
  };

  useEffect(() => {
    loadRoles();
    loadStatistics();
  }, []);

  // 搜索
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    loadRoles(values);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    loadRoles();
  };

  // 刷新
  const handleRefresh = () => {
    loadRoles();
    loadStatistics();
  };

  // 新增角色
  const handleAdd = () => {
    setEditingRole(null);
    editForm.resetFields();
    setEditModalVisible(true);
  };

  // 编辑角色
  const handleEdit = (record: Role) => {
    setEditingRole(record);
    editForm.setFieldsValue(record);
    setEditModalVisible(true);
  };

  // 删除角色
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/roles/${id}/delete`);
      message.success('删除成功');
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('删除角色失败:', error);
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的角色');
      return;
    }

    try {
      await post('/system/roles/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  // 更新角色状态
  const handleUpdateStatus = async (id: string, status: number) => {
    try {
      await post(`/system/roles/${id}/status`, { status });
      message.success(status === 0 ? '启用成功' : '停用成功');
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('更新角色状态失败:', error);
    }
  };

  // 保存角色
  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();

      // 添加缺失的字段并确保数据类型正确
      const roleData = {
        roleName: values.roleName,
        roleKey: values.roleKey,
        roleSort: parseInt(values.roleSort) || 0,
        dataScope: values.dataScope || 1, // 默认数据范围：全部数据
        menuCheckStrictly: values.menuCheckStrictly !== undefined ? values.menuCheckStrictly : true,
        deptCheckStrictly: values.deptCheckStrictly !== undefined ? values.deptCheckStrictly : true,
        status: parseInt(values.status) || 0,
        remark: values.remark,
        menuIds: values.menuIds || [],
        deptIds: values.deptIds || [],
      };

      if (editingRole) {
        await post(`/system/roles/${editingRole.id}/update`, roleData);
      } else {
        await post('/system/roles', roleData);
      }

      message.success(editingRole ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('保存角色失败:', error);
    }
  };

  // 分配菜单权限
  const handleAssignMenu = async (role: Role) => {
    setEditingRole(role);
    await loadMenuTree();

    // 加载角色已选中的菜单
    try {
      const result = await post(`/system/menus/role-menu-tree-select/${role.id}`);
      setSelectedMenuIds(result.data.checkedKeys || []);
    } catch (error) {
      console.error('加载角色菜单权限失败:', error);
    }

    setMenuTreeVisible(true);
  };

  // 保存菜单权限
  const handleSaveMenu = async () => {
    if (!editingRole) return;

    try {
      await post(`/system/roles/${editingRole.id}/menus`, {
        menuIds: selectedMenuIds,
      });
      message.success('菜单权限分配成功');
      setMenuTreeVisible(false);
    } catch (error) {
      console.error('保存菜单权限失败:', error);
    }
  };

  // 表格列定义
  const columns: ColumnsType<Role> = [
    {
      title: '角色名称',
      dataIndex: 'roleName',
      key: 'roleName',
      render: (text) => (
        <Space>
          <SafetyCertificateOutlined />
          {text}
        </Space>
      ),
    },
    {
      title: '权限字符',
      dataIndex: 'roleKey',
      key: 'roleKey',
      render: (text) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: '显示顺序',
      dataIndex: 'roleSort',
      key: 'roleSort',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '正常' : '停用'}
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
            icon={<SettingOutlined />}
            onClick={() => handleAssignMenu(record)}
          >
            菜单权限
          </Button>
          <Button
            type="link"
            icon={record.status === 0 ? <DeleteOutlined /> : <SettingOutlined />}
            onClick={() => handleUpdateStatus(record.id, record.status === 0 ? 1 : 0)}
          >
            {record.status === 0 ? '停用' : '启用'}
          </Button>
          <Popconfirm
            title="确定要删除这个角色吗？"
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
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="总角色数"
              value={statistics.total}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="正常角色"
              value={statistics.active}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<SettingOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用角色"
              value={statistics.inactive}
              styles={{ content: { color: '#cf1322' } }}
              prefix={<DeleteOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单 */}
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
          <Form.Item name="roleName" label="角色名称">
            <Input placeholder="请输入角色名称" />
          </Form.Item>
          <Form.Item name="roleKey" label="权限字符">
            <Input placeholder="请输入权限字符" />
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
      </Card>

      {/* 操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增角色
          </Button>
          <Button
            danger
            onClick={handleBatchDelete}
            disabled={selectedRowKeys.length === 0}
          >
            批量删除
          </Button>
        </Space>
      </Card>

      {/* 角色表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={roles}
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
              loadRoles({ current: page, pageSize: size });
            },
          }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingRole ? '编辑角色' : '新增角色'}
        open={editModalVisible}
        onOk={handleSave}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="roleName"
            label="角色名称"
            rules={[{ required: true, message: '请输入角色名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="roleKey"
            label="权限字符"
            rules={[{ required: true, message: '请输入权限字符' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="roleSort"
            label="显示顺序"
            rules={[{ required: true, message: '请输入显示顺序' }]}
          >
            <Input type="number" />
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

      {/* 菜单权限弹窗 */}
      <Modal
        title="分配菜单权限"
        open={menuTreeVisible}
        onOk={handleSaveMenu}
        onCancel={() => setMenuTreeVisible(false)}
        width={600}
      >
        <div style={{ marginBottom: 16 }}>
          <span>角色：{editingRole?.roleName}</span>
        </div>
        <Tree
          checkable
          treeData={menuTree}
          checkedKeys={selectedMenuIds}
          onCheck={(checkedKeys) => {
            setSelectedMenuIds(checkedKeys as string[]);
          }}
        />
      </Modal>
    </div>
  );
};

export default RoleManagement;
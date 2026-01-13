import { useState, useEffect } from 'react';
import type { FC } from 'react';
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
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { TreeDataNode } from 'antd';
import type { Role } from '@/types';
import type { Key } from 'antd/es/table/interface';
import { post } from '@/lib/api';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;

const RoleManagement: FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);

  // 新增状态：菜单树、部门树、数据范围
  const [menuTree, setMenuTree] = useState<TreeDataNode[]>([]);
  const [deptTree, setDeptTree] = useState<TreeDataNode[]>([]);
  const [currentDataScope, setCurrentDataScope] = useState<number>(1);
  // 待设置的表单数据
  const [pendingFormData, setPendingFormData] = useState<any>(null);
  // 菜单和部门选中状态
  const [checkedMenuKeys, setCheckedMenuKeys] = useState<Key[]>([]);
  const [checkedDeptKeys, setCheckedDeptKeys] = useState<Key[]>([]);

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
      const processTreeData = (nodes: any[]): TreeDataNode[] => {
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

  // 加载部门树
  const loadDeptTree = async () => {
    try {
      const result = await post('/system/departments/tree-select');
      const processTreeData = (nodes: any[]): TreeDataNode[] => {
        return nodes.map((node: any) => ({
          key: node.key || node.id,
          title: node.title || node.deptName,
          value: node.value || node.id,
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setDeptTree(processTreeData(result.data));
    } catch (error) {
      console.error('加载部门树失败:', error);
    }
  };

  // 加载角色的菜单权限
  const loadRoleMenus = async (roleId: string): Promise<string[]> => {
    try {
      const result = await post(`/system/menus/role-menu-tree-select/${roleId}`);
      return result.data.checkedKeys || [];
    } catch (error) {
      console.error('加载角色菜单权限失败:', error);
      return [];
    }
  };

  // 加载角色的部门权限
  const loadRoleDepts = async (roleId: string): Promise<string[]> => {
    try {
      const result = await post(`/system/departments/role-dept-tree-select/${roleId}`);
      return result.data.checkedKeys || [];
    } catch (error) {
      console.error('加载角色部门权限失败:', error);
      return [];
    }
  };

  useEffect(() => {
    loadRoles();
    loadStatistics();
    loadMenuTree();
    loadDeptTree();
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
    editForm.setFieldsValue({
      dataScope: 1,
      status: 0,
    });
    setCurrentDataScope(1);
    setCheckedMenuKeys([]);
    setCheckedDeptKeys([]);
    setEditModalVisible(true);
  };

  // 编辑角色
  const handleEdit = async (record: Role) => {
    setEditingRole(record);

    // 加载角色的菜单和部门权限
    const [menuIds, deptIds] = await Promise.all([
      loadRoleMenus(record.id),
      loadRoleDepts(record.id),
    ]);

    // 设置选中的菜单和部门
    setCheckedMenuKeys(menuIds);
    setCheckedDeptKeys(deptIds);

    // 保存待设置的表单数据
    setPendingFormData({
      ...record,
      menuIds,
      deptIds,
    });
    setCurrentDataScope(record.dataScope || 1);
    setEditModalVisible(true);
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && pendingFormData) {
      // Modal 打开时设置表单值
      editForm.setFieldsValue(pendingFormData);
      setPendingFormData(null);
    }
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

      const roleData = {
        roleName: values.roleName,
        roleKey: values.roleKey,
        roleSort: parseInt(values.roleSort) || 0,
        dataScope: values.dataScope || 1,
        menuCheckStrictly: true,
        deptCheckStrictly: true,
        status: parseInt(values.status) || 0,
        remark: values.remark,
        menuIds: values.menuIds || [],
        deptIds: values.deptIds || [],
      };

      if (editingRole) {
        await post(`/system/roles/${editingRole.id}/update`, {
          ...roleData,
          id: editingRole.id,
        });
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

  // 数据范围变更时更新状态
  const handleDataScopeChange = (value: number) => {
    setCurrentDataScope(value);
    // 如果不是自定义数据权限，清空部门权限选择
    if (value !== 2) {
      editForm.setFieldsValue({ deptIds: [] });
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
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'toggle-status',
            label: record.status === 0 ? '停用' : '启用',
            icon: record.status === 0 ? <DeleteOutlined /> : <SafetyCertificateOutlined />,
            onClick: () => handleUpdateStatus(record.id, record.status === 0 ? 1 : 0),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个角色吗？"
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

  // 数据范围选项
  const dataScopeOptions = [
    { label: '全部数据', value: 1 },
    { label: '自定义数据', value: 2 },
    { label: '本部门数据', value: 3 },
    { label: '本部门及子部门数据', value: 4 },
    { label: '仅本人数据', value: 5 },
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
              prefix={<SafetyCertificateOutlined />}
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

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
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
        </div>
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
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={800}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      >
        <Form form={editForm} layout="vertical">
          {/* 基本信息 */}
          <Card title="基本信息" size="small" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="roleName"
                  label="角色名称"
                  rules={[{ required: true, message: '请输入角色名称' }]}
                >
                  <Input />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="roleKey"
                  label="权限字符"
                  rules={[{ required: true, message: '请输入权限字符' }]}
                >
                  <Input />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="roleSort"
                  label="显示顺序"
                  rules={[{ required: true, message: '请输入显示顺序' }]}
                  initialValue={0}
                >
                  <Input type="number" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="status"
                  label="状态"
                  initialValue={0}
                >
                  <Select>
                    <Option value={0}>正常</Option>
                    <Option value={1}>停用</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="remark" label="备注">
              <Input.TextArea rows={2} />
            </Form.Item>
          </Card>

          {/* 菜单权限和数据权限横向排列 */}
          <Row gutter={16}>
            {/* 左侧：菜单权限 */}
            <Col span={12}>
              <Card title="菜单权限" size="small">
                <Form.Item
                  name="menuIds"
                  label="权限菜单"
                  tooltip="选择角色可以访问的菜单（选择父菜单时自动选择子菜单）"
                >
                  <Tree
                    checkable
                    checkedKeys={checkedMenuKeys}
                    onCheck={(keys) => {
                      const keyArray = (keys as unknown) as Key[];
                      setCheckedMenuKeys(keyArray);
                      editForm.setFieldsValue({ menuIds: keyArray });
                    }}
                    treeData={menuTree}
                    placeholder="请选择菜单"
                    style={{ background: '#f5f5f5', padding: '8px', borderRadius: '4px', maxHeight: '400px', overflowY: 'auto' }}
                  />
                </Form.Item>
              </Card>
            </Col>

            {/* 右侧：数据权限 */}
            <Col span={12}>
              <Card title="数据权限" size="small">
                <Form.Item
                  name="dataScope"
                  label="数据范围"
                  initialValue={1}
                >
                  <Select onChange={handleDataScopeChange}>
                    {dataScopeOptions.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>

                {currentDataScope === 2 && (
                  <Form.Item
                    name="deptIds"
                    label="部门权限"
                    tooltip="选择角色可以访问的部门数据（选择父部门时自动选择子部门）"
                  >
                    <Tree
                      checkable
                      checkedKeys={checkedDeptKeys}
                      onCheck={(keys) => {
                        const keyArray = (keys as unknown) as Key[];
                        setCheckedDeptKeys(keyArray);
                        editForm.setFieldsValue({ deptIds: keyArray });
                      }}
                      treeData={deptTree}
                      placeholder="请选择部门"
                      style={{ background: '#f5f5f5', padding: '8px', borderRadius: '4px', maxHeight: '400px', overflowY: 'auto' }}
                    />
                  </Form.Item>
                )}
              </Card>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
};

export default RoleManagement;

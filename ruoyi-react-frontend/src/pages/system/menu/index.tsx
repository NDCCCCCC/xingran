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
  InputNumber,
  Switch,
  TreeSelect,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  MenuOutlined,
  FolderOutlined,
  FileOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Menu } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;

const MenuManagement: React.FC = () => {
  const [menus, setMenus] = useState<Menu[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingMenu, setEditingMenu] = useState<Menu | null>(null);
  const [parentOptions, setParentOptions] = useState<any[]>([]);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    directories: 0,
    menus: 0,
    buttons: 0,
  });

  // 菜单类型图标映射
  const getMenuIcon = (menuType: string) => {
    switch (menuType) {
      case 'M':
        return <FolderOutlined />;
      case 'C':
        return <FileOutlined />;
      case 'F':
        return <AppstoreOutlined />;
      default:
        return <MenuOutlined />;
    }
  };

  // 菜单类型标签
  const getMenuTypeTag = (menuType: string) => {
    switch (menuType) {
      case 'M':
        return <Tag color="blue">目录</Tag>;
      case 'C':
        return <Tag color="green">菜单</Tag>;
      case 'F':
        return <Tag color="orange">按钮</Tag>;
      default:
        return <Tag>未知</Tag>;
    }
  };

  // 加载菜单列表
  const loadMenus = async (searchParams?: any) => {
    setLoading(true);
    try {
      const result = await post('/system/menus/list', searchParams || {});
      const menuData = result.data || [];
      // 处理树形数据
      const flatMenus = flattenTree(menuData);
      setMenus(menuData);
      setParentOptions(buildParentOptions(menuData));

      // 统计数据
      const stats = {
        total: flatMenus.length,
        directories: flatMenus.filter(m => m.menuType === 'M').length,
        menus: flatMenus.filter(m => m.menuType === 'C').length,
        buttons: flatMenus.filter(m => m.menuType === 'F').length,
      };
      setStatistics(stats);
    } catch (error) {
      console.error('加载菜单列表失败:', error);
      // 设置默认值
      setMenus([]);
      setParentOptions([]);
      setStatistics({ total: 0, directories: 0, menus: 0, buttons: 0 });
    } finally {
      setLoading(false);
    }
  };

  // 扁平化树形数据
  const flattenTree = (tree: any[]): any[] => {
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

  // 构建父级选项
  const buildParentOptions = (tree: any[]): any[] => {
    const options: any[] = [{ title: '顶级菜单', value: '', key: '' }];
    if (!tree || !Array.isArray(tree)) {
      return options;
    }
    const traverse = (nodes: any[], parentTitle = '') => {
      if (!nodes || !Array.isArray(nodes)) {
        return;
      }
      nodes.forEach(node => {
        if (node) {
          options.push({
            title: `${parentTitle}${node.menuName}`,
            value: node.id,
            key: node.id,
          });
          if (node.children && node.children.length > 0) {
            traverse(node.children, `${parentTitle}${node.menuName} / `);
          }
        }
      });
    };
    traverse(tree);
    return options;
  };

  useEffect(() => {
    loadMenus();
  }, []);

  // 搜索
  const handleSearch = async () => {
    const values = await searchForm.validateFields();
    loadMenus(values);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    loadMenus();
  };

  // 刷新
  const handleRefresh = () => {
    loadMenus();
  };

  // 新增菜单
  const handleAdd = () => {
    setEditingMenu(null);
    editForm.resetFields();
    // 设置默认值
    editForm.setFieldsValue({
      parentId: '',
      menuType: 'M',
      orderNum: 0,
      status: true,
    });
    setEditModalVisible(true);
  };

  // 编辑菜单
  const handleEdit = (record: Menu) => {
    setEditingMenu(record);
    setEditModalVisible(true);
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && editingMenu) {
      // Modal 打开时设置表单值
      // 转换状态值：0->true(显示), 1->false(隐藏)
      editForm.setFieldsValue({
        ...editingMenu,
        status: editingMenu.status === 0
      });
    }
  };

  // 删除菜单
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/menus/${id}/delete`);
      message.success('删除成功');
      loadMenus();
    } catch (error) {
      console.error('删除菜单失败:', error);
    }
  };

  // 保存菜单
  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();

      // 转换Switch状态值：true->0(显示), false->1(隐藏)
      if (typeof values.status === 'boolean') {
        values.status = values.status ? 0 : 1;
      }

      if (editingMenu) {
        await post(`/system/menus/${editingMenu.id}/update`, {
          ...values,
          id: editingMenu.id,
        });
      } else {
        await post('/system/menus', values);
      }

      message.success(editingMenu ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadMenus();
    } catch (error) {
      console.error('保存菜单失败:', error);
    }
  };

  // 表格列定义
  const columns: ColumnsType<Menu> = [
    {
      title: '菜单名称',
      dataIndex: 'menuName',
      key: 'menuName',
      render: (text, record) => (
        <Space>
          {getMenuIcon(record.menuType)}
          {text}
        </Space>
      ),
    },
    {
      title: '菜单类型',
      dataIndex: 'menuType',
      key: 'menuType',
      render: (menuType) => getMenuTypeTag(menuType),
    },
    {
      title: '权限标识',
      dataIndex: 'perms',
      key: 'perms',
      render: (perms) => perms && <Tag color="cyan">{perms}</Tag>,
    },
    {
      title: '路由地址',
      dataIndex: 'path',
      key: 'path',
    },
    {
      title: '组件路径',
      dataIndex: 'component',
      key: 'component',
    },
    {
      title: '显示顺序',
      dataIndex: 'orderNum',
      key: 'orderNum',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '显示' : '隐藏'}
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
          <Popconfirm
            title="确定要删除这个菜单吗？"
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

  // 渲染树形表格数据
  const renderTreeData = (data: Menu[]): any[] => {
    return data.map(item => ({
      ...item,
      key: item.id,
      children: item.children && item.children.length > 0 ? renderTreeData(item.children) : undefined,
    }));
  };

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总菜单数"
              value={statistics.total}
              prefix={<MenuOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="目录数量"
              value={statistics.directories}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<FolderOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="菜单数量"
              value={statistics.menus}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<FileOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="按钮数量"
              value={statistics.buttons}
              styles={{ content: { color: '#fa8c16' } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="menuName" label="菜单名称">
              <Input placeholder="请输入菜单名称" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value="0">显示</Option>
                <Option value="1">隐藏</Option>
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
              新增菜单
            </Button>
          </Space>
        </div>
      </Card>

      {/* 菜单表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={renderTreeData(menus)}
          rowKey="id"
          loading={loading}
          pagination={false}
          expandable={{
            defaultExpandAllRows: true,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingMenu ? '编辑菜单' : '新增菜单'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="parentId"
                label="上级菜单"
              >
                <TreeSelect
                  style={{ width: '100%' }}
                  dropdownStyle={{ maxHeight: 400, overflow: 'auto' }}
                  treeData={parentOptions}
                  placeholder="请选择上级菜单"
                  treeDefaultExpandAll
                  allowClear
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="menuType"
                label="菜单类型"
                rules={[{ required: true, message: '请选择菜单类型' }]}
              >
                <Select placeholder="请选择菜单类型">
                  <Option value="M">目录</Option>
                  <Option value="C">菜单</Option>
                  <Option value="F">按钮</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="menuName"
                label="菜单名称"
                rules={[{ required: true, message: '请输入菜单名称' }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="orderNum"
                label="显示顺序"
                rules={[{ required: true, message: '请输入显示顺序' }]}
              >
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="path" label="路由地址">
                <Input placeholder="请输入路由地址" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="component" label="组件路径">
                <Input placeholder="请输入组件路径" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="perms" label="权限标识">
                <Input placeholder="请输入权限标识" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="icon" label="菜单图标">
                <Input placeholder="请输入菜单图标" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="status" label="显示状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="显示" unCheckedChildren="隐藏" />
          </Form.Item>

          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default MenuManagement;
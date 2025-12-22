import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
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
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Department } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;

const DepartmentManagement: React.FC = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [parentOptions, setParentOptions] = useState<any[]>([]);

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
      const result = await post('/system/depts/list', searchParams || {});
      const deptData = result.data || [];
      setDepartments(deptData);

      // 组建父级选项
      const flatList = flattenTreeToList(deptData);
      const options = flatList.map(dept => ({
        title: dept.deptName,
        value: dept.id,
        key: dept.id,
      }));
      options.unshift({ title: '顶级部门', value: '', key: '' });
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
      setParentOptions([{ title: '顶级部门', value: '', key: '' }]);
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
    setEditModalVisible(true);
  };

  // 编辑部门
  const handleEdit = (record: Department) => {
    setEditingDept(record);
    editForm.setFieldsValue(record);
    setEditModalVisible(true);
  };

  // 删除部门
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/depts/${id}/delete`);
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

      if (editingDept) {
        await post(`/system/depts/${editingDept.id}/update`, values);
      } else {
        await post('/system/depts', values);
      }

      message.success(editingDept ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadDepartments();
    } catch (error) {
      console.error('保存部门失败:', error);
    }
  };

  // 表格列定义
  const columns: ColumnsType<Department> = [
    {
      title: '部门名称',
      dataIndex: 'deptName',
      key: 'deptName',
      render: (text) => (
        <Space>
          <ApartmentOutlined />
          {text}
        </Space>
      ),
    },
    {
      title: '负责人',
      dataIndex: 'leader',
      key: 'leader',
      render: (leader) => leader || '-',
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
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个部门吗？"
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

      {/* 搜索表单 */}
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
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
      </Card>

      {/* 操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增部门
          </Button>
        </Space>
      </Card>

      {/* 部门表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={renderTreeData(departments)}
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
        title={editingDept ? '编辑部门' : '新增部门'}
        open={editModalVisible}
        onOk={handleSave}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="parentId"
            label="上级部门"
          >
            <Select
              style={{ width: '100%' }}
              placeholder="请选择上级部门"
              options={parentOptions}
              allowClear
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
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="leader" label="负责人">
            <Input placeholder="请输入负责人" />
          </Form.Item>
          <Form.Item name="phone" label="联系电话">
            <Input placeholder="请输入联系电话" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="请输入邮箱" />
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
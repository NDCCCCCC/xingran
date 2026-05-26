import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  message,
  Popconfirm,
  Tag
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { DeptGroupMapping, ListMappingsRequest } from '@/types/ad-group';
import { post } from '@/lib/api';

const { Option } = Select;

const GroupMapping: React.FC = () => {
  const [mappings, setMappings] = useState<DeptGroupMapping[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);

  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [currentMapping, setCurrentMapping] = useState<DeptGroupMapping | null>(null);

  const [form] = Form.useForm();

  // 获取映射列表
  const fetchMappings = async (params: ListMappingsRequest = {}) => {
    setLoading(true);
    try {
      const result = await post('/api/v1/ad/groups/mappings', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params
      });
      setMappings(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      message.error('获取映射列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMappings();
  }, []);

  // 创建映射
  const handleCreate = async (values: any) => {
    try {
      await post('/api/v1/ad/groups/mappings', values);
      message.success('创建成功');
      setCreateModalVisible(false);
      form.resetFields();
      fetchMappings();
    } catch (error) {
      message.error('创建失败');
    }
  };

  // 更新映射
  const handleUpdate = async (values: any) => {
    if (!currentMapping) return;
    try {
      await post(`/api/v1/ad/groups/mappings/${currentMapping.id}/update`, values);
      message.success('更新成功');
      setEditModalVisible(false);
      setCurrentMapping(null);
      fetchMappings();
    } catch (error) {
      message.error('更新失败');
    }
  };

  // 删除映射
  const handleDelete = async (id: string) => {
    try {
      await post(`/api/v1/ad/groups/mappings/${id}/delete`);
      message.success('删除成功');
      fetchMappings();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const columns = [
    { title: '部门名称', dataIndex: 'deptName', key: 'deptName' },
    { title: 'AD组名称', dataIndex: 'adGroupName', key: 'adGroupName' },
    {
      title: '映射类型',
      dataIndex: 'mappingType',
      key: 'mappingType',
      render: (type: string) => (
        <Tag color={type === 'auto' ? 'blue' : 'green'}>
          {type === 'auto' ? '自动映射' : '手动映射'}
        </Tag>
      )
    },
    {
      title: '状态',
      dataIndex: 'mappingStatus',
      key: 'mappingStatus',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>
          {status === 'active' ? '活跃' : '停用'}
        </Tag>
      )
    },
    {
      title: '同步启用',
      dataIndex: 'syncEnabled',
      key: 'syncEnabled',
      render: (enabled: boolean) => enabled ? '是' : '否'
    },
    { title: '成员数', dataIndex: 'memberCount', key: 'memberCount' },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: DeptGroupMapping) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setCurrentMapping(record);
              setEditModalVisible(true);
            }}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个映射吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div>
      <Card
        title="部门-组映射管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModalVisible(true)}
          >
            新建映射
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={mappings}
          rowKey="id"
          loading={loading}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
              fetchMappings({ current: page, pageSize: size });
            }
          }}
        />
      </Card>

      {/* 创建映射对话框 */}
      <Modal
        title="新建映射"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        onOk={() => form.submit()}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
        >
          <Form.Item
            name="deptId"
            label="部门"
            rules={[{ required: true, message: '请选择部门' }]}
          >
            <Select placeholder="请选择部门">
              {/* TODO: 从API获取部门列表 */}
              <Option value="dept-1">技术部</Option>
              <Option value="dept-2">市场部</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="adGroupId"
            label="AD组"
            rules={[{ required: true, message: '请输入AD组DN' }]}
          >
            <Input placeholder="请输入AD组DN" />
          </Form.Item>

          <Form.Item
            name="adConfigId"
            label="AD配置"
            rules={[{ required: true, message: '请选择AD配置' }]}
          >
            <Select placeholder="请选择AD配置">
              {/* TODO: 从API获取AD配置列表 */}
              <Option value="config-1">AD配置1</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="mappingType"
            label="映射类型"
            initialValue="manual"
          >
            <Select>
              <Option value="manual">手动映射</Option>
              <Option value="auto">自动映射</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="syncEnabled"
            label="启用同步"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑映射对话框 */}
      <Modal
        title="编辑映射"
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false);
          setCurrentMapping(null);
        }}
        onOk={() => form.submit()}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleUpdate}
          initialValues={currentMapping || undefined}
        >
          <Form.Item
            name="mappingStatus"
            label="映射状态"
          >
            <Select>
              <Option value="active">活跃</Option>
              <Option value="inactive">停用</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="syncEnabled"
            label="启用同步"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default GroupMapping;
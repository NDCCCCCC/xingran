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
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { Post } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

const PostManagement: React.FC = () => {
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [postForm] = Form.useForm();
  const [editingPost, setEditingPost] = useState<Post | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 加载岗位列表
  const loadPosts = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/posts/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setPosts(result.data.list || []);
      setTotal(result.data.total || 0);
    } catch (error) {
      console.error('加载岗位列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 创建岗位
  const handleCreate = async () => {
    try {
      const values = await postForm.validateFields();
      if (editingPost) {
        await post(`/system/posts/${editingPost.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/posts', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      postForm.resetFields();
      setEditingPost(null);
      loadPosts();
    } catch (error: any) {
      if (error.errorFields) {
        return; // 表单验证错误
      }
      message.error('操作失败');
    }
  };

  // 删除岗位
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/posts/${id}/delete`, {});
      message.success('删除成功');
      loadPosts();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除岗位
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/posts/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadPosts();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Post) => {
    if (record) {
      setEditingPost(record);
      postForm.setFieldsValue(record);
    } else {
      setEditingPost(null);
      postForm.resetFields();
      postForm.setFieldsValue({ postSort: 0, status: 0 });
    }
    setModalVisible(true);
  };

  // 表格列定义
  const columns: ColumnsType<Post> = [
    { title: '岗位编码', dataIndex: 'postCode', key: 'postCode', width: 150 },
    { title: '岗位名称', dataIndex: 'postName', key: 'postName', width: 150 },
    { title: '显示排序', dataIndex: 'postSort', key: 'postSort', width: 100 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '正常' : '停用'}
        </Tag>
      ),
    },
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>
            编辑
          </Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadPosts();
  }, [current, pageSize]);

  return (
    <div>
      {/* 搜索表单 */}
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="postCode" label="岗位编码">
          <Input placeholder="请输入岗位编码" allowClear style={{ width: 150 }} />
        </Form.Item>
        <Form.Item name="postName" label="岗位名称">
          <Input placeholder="请输入岗位名称" allowClear style={{ width: 150 }} />
        </Form.Item>
        <Form.Item name="status" label="状态">
          <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
            <Option value={0}>正常</Option>
            <Option value={1}>停用</Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => loadPosts()}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadPosts(); }}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      {/* 操作按钮 */}
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新增
          </Button>
          <Button
            danger
            icon={<DeleteOutlined />}
            onClick={handleBatchDelete}
            disabled={selectedRowKeys.length === 0}
          >
            批量删除
          </Button>
        </Space>
      </div>

      {/* 数据表格 */}
      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={posts}
        loading={loading}
        rowKey="id"
        scroll={{ x: 1200 }}
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
          onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); },
        }}
      />

      {/* 编辑模态框 */}
      <Modal
        title={editingPost ? '编辑岗位' : '新增岗位'}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); postForm.resetFields(); setEditingPost(null); }}
        width={600}
      >
        <Form form={postForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="postCode" label="岗位编码" rules={[{ required: true, message: '请输入岗位编码' }]}>
            <Input placeholder="请输入岗位编码" disabled={!!editingPost} />
          </Form.Item>
          <Form.Item name="postName" label="岗位名称" rules={[{ required: true, message: '请输入岗位名称' }]}>
            <Input placeholder="请输入岗位名称" />
          </Form.Item>
          <Form.Item name="postSort" label="显示排序" rules={[{ required: true, message: '请输入显示排序' }]}>
            <InputNumber min={0} style={{ width: '100%' }} placeholder="请输入显示排序" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
            <Select placeholder="请选择状态">
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PostManagement;

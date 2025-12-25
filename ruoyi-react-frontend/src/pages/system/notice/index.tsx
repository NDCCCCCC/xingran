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
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { Notice } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

const NoticeManagement: React.FC = () => {
  const [notices, setNotices] = useState<Notice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingNotice, setEditingNotice] = useState<Notice | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 加载通知公告列表
  const loadNotices = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/notices/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setNotices(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载通知公告失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 创建通知公告
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingNotice) {
        await post(`/system/notices/${editingNotice.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/notices', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      editForm.resetFields();
      setEditingNotice(null);
      loadNotices();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败');
    }
  };

  // 删除通知公告
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/notices/${id}/delete`, {});
      message.success('删除成功');
      loadNotices();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除通知公告
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/notices/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadNotices();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Notice) => {
    if (record) {
      setEditingNotice(record);
      editForm.setFieldsValue(record);
    } else {
      setEditingNotice(null);
      editForm.resetFields();
      editForm.setFieldsValue({ noticeType: '1', status: 0 });
    }
    setModalVisible(true);
  };

  // 表格列
  const columns: ColumnsType<Notice> = [
    { title: '公告标题', dataIndex: 'noticeTitle', key: 'noticeTitle', ellipsis: true },
    {
      title: '公告类型',
      dataIndex: 'noticeType',
      key: 'noticeType',
      render: (noticeType: string) => (
        <Tag color={noticeType === '1' ? 'blue' : 'orange'}>{noticeType === '1' ? '公告' : '警告'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'default'}>{status === 0 ? '正常' : '关闭'}</Tag>
      ),
    },
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>编辑</Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadNotices();
  }, [current, pageSize]);

  return (
    <div>
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="noticeTitle" label="公告标题">
          <Input placeholder="请输入公告标题" allowClear />
        </Form.Item>
        <Form.Item name="noticeType" label="公告类型">
          <Select placeholder="请选择公告类型" allowClear style={{ width: 120 }}>
            <Option value="1">公告</Option>
            <Option value="2">警告</Option>
          </Select>
        </Form.Item>
        <Form.Item name="createTime" label="创建时间">
          <Input placeholder="请选择创建时间" allowClear />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => loadNotices()}>查询</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadNotices(); }}>重置</Button>
          </Space>
        </Form.Item>
      </Form>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增</Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
        </Space>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={notices}
        loading={loading}
        rowKey="id"
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
        title={editingNotice ? '编辑通知公告' : '新增通知公告'}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); editForm.resetFields(); setEditingNotice(null); }}
        width={800}
      >
        <Form form={editForm} labelCol={{ span: 4 }} wrapperCol={{ span: 18 }}>
          <Form.Item name="noticeTitle" label="公告标题" rules={[{ required: true, message: '请输入公告标题' }]}>
            <Input placeholder="请输入公告标题" />
          </Form.Item>
          <Form.Item name="noticeType" label="公告类型" rules={[{ required: true }]}>
            <Select>
              <Option value="1">公告</Option>
              <Option value="2">警告</Option>
            </Select>
          </Form.Item>
          <Form.Item name="noticeContent" label="公告内容" rules={[{ required: true, message: '请输入公告内容' }]}>
            <TextArea rows={8} placeholder="请输入公告内容" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>关闭</Option>
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

export default NoticeManagement;

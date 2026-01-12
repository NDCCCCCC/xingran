import { useEffect } from 'react';
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
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn, createActionColumn } from '@/utils/tableHelpers';

const { Option } = Select;
const { TextArea } = Input;

const PostManagement: FC = () => {
  const {
    loading,
    data: posts,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm: postForm,
    editModalVisible: modalVisible,
    editingItem: editingPost,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingPost,
    setCurrent,
    setPageSize,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadPosts,
    resetSelection,
  } = useTableManager<Post>(
    async (params) => {
      const result = await post('/system/posts/list', params);
      return { list: result.data.list || [], total: result.data.total || 0 };
    },
    { pageSize: 10 }
  );

  // 创建岗位
  const handleCreate = async () => {
    try {
      const values = await postForm.validateFields();
      if (editingPost) {
        await post(`/system/posts/${editingPost.id}/update`, values);
        handleSuccess('更新');
      } else {
        await post('/system/posts', values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadPosts();
    } catch (error: any) {
      if (error.errorFields) {
        return; // 表单验证错误
      }
      handleApiError(error, '操作');
    }
  };

  // 删除岗位
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/posts/${id}/delete`, {});
      handleSuccess('删除');
      loadPosts();
    } catch (error) {
      handleApiError(error, '删除');
    }
  };

  // 批量删除岗位
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    try {
      await post('/system/posts/batch-delete', { ids: selectedRowKeys });
      handleSuccess('批量删除');
      resetSelection();
      loadPosts();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Post) => {
    if (record) {
      handleEdit(record);
      postForm.setFieldsValue(record);
    } else {
      handleAdd();
      postForm.setFieldsValue({ postSort: 0, status: 0 });
    }
  };

  // 表格列定义
  const columns: ColumnsType<Post> = [
    { title: '岗位编码', dataIndex: 'postCode', key: 'postCode', width: 150 },
    { title: '岗位名称', dataIndex: 'postName', key: 'postName', width: 150 },
    { title: '显示排序', dataIndex: 'postSort', key: 'postSort', width: 100 },
    createStatusColumn('status', { width: 100 }),
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    createActionColumn(
      (record) => openModal(record),
      (record) => handleDelete(record.id)
    ),
  ];

  useEffect(() => {
    loadPosts();
  }, [current, pageSize, loadPosts]);

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
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
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
          onChange: (page, size) => {
            setCurrent(page);
            setPageSize(size || 10);
          },
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

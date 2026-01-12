import type { FC } from 'react';
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
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { Config } from '@/types';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { withErrorHandling } from '@/utils/errorHandler';

const { Option } = Select;
const { TextArea } = Input;

const ConfigManagement: FC = () => {
  const {
    loading,
    data: configs,
    total,
    current,
    pageSize,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingConfig,
    setEditModalVisible,
    setEditingItem,
    loadData: loadConfigs,
    handleSearch,
    handleReset,
    handleRefresh,
    handleAdd,
    handleEdit,
    handleModalClose,
    setCurrent,
    setPageSize,
  } = useTableManager<Config>(
    async (params) => {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/configs/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      return result.data;
    }
  );

  // 打开编辑模态框
  const openModal = (record?: Config) => {
    if (record) {
      setEditingItem(record);
      editForm.setFieldsValue(record);
    } else {
      setEditingItem(null);
      editForm.resetFields();
      editForm.setFieldsValue({ configType: 'N', isSystem: 0 });
    }
    setEditModalVisible(true);
  };

  // 创建参数配置
  const handleCreate = async () => {
    await withErrorHandling(
      async () => {
        const values = await editForm.validateFields();
        if (editingConfig) {
          await post(`/system/configs/${editingConfig.id}/update`, values);
          return '更新成功';
        } else {
          await post('/system/configs', values);
          return '创建成功';
        }
      },
      {
        onSuccess: () => {
          handleModalClose();
          loadConfigs();
        },
      }
    );
  };

  // 删除参数配置
  const handleDelete = async (id: string, isSystem: number) => {
    if (isSystem === 1) {
      message.warning('系统内置参数不能删除');
      return;
    }
    await withErrorHandling(
      async () => {
        await post(`/system/configs/${id}/delete`, {});
        return '删除成功';
      },
      {
        onSuccess: () => loadConfigs(),
      }
    );
  };

  // 批量删除参数配置
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    await withErrorHandling(
      async () => {
        await post('/system/configs/batch-delete', { ids: selectedRowKeys });
        return '批量删除成功';
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadConfigs();
        },
      }
    );
  };

  // 刷新缓存
  const handleRefreshCache = async () => {
    await withErrorHandling(
      async () => {
        await post('/system/configs/refresh-cache', {});
        return '刷新缓存成功';
      },
      {}
    );
  };

  // 表格列
  const columns: ColumnsType<Config> = [
    { title: '参数名称', dataIndex: 'configName', key: 'configName' },
    { title: '参数键名', dataIndex: 'configKey', key: 'configKey' },
    { title: '参数键值', dataIndex: 'configValue', key: 'configValue', ellipsis: true },
    {
      title: '系统内置',
      dataIndex: 'isSystem',
      key: 'isSystem',
      render: (isSystem: number) => (
        <Tag color={isSystem === 1 ? 'blue' : 'default'}>{isSystem === 1 ? '是' : '否'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'configType',
      key: 'configType',
      render: (configType: string) => (
        <Tag color={configType === 'Y' ? 'success' : 'default'}>{configType === 'Y' ? '是' : '否'}</Tag>
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
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm
            title="确认删除?"
            onConfirm={() => handleDelete(record.id, record.isSystem)}
            disabled={record.isSystem === 1}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} disabled={record.isSystem === 1}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="configName" label="参数名称">
          <Input placeholder="请输入参数名称" allowClear />
        </Form.Item>
        <Form.Item name="configKey" label="参数键名">
          <Input placeholder="请输入参数键名" allowClear />
        </Form.Item>
        <Form.Item name="configType" label="状态">
          <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
            <Option value="Y">是</Option>
            <Option value="N">否</Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
          </Space>
        </Form.Item>
      </Form>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新增</Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
          <Button icon={<ReloadOutlined />} onClick={handleRefreshCache}>刷新缓存</Button>
        </Space>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
          getCheckboxProps: (record) => ({
            disabled: record.isSystem === 1,
          }),
        }}
        columns={columns}
        dataSource={configs}
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
        title={editingConfig ? '编辑参数配置' : '新增参数配置'}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={handleModalClose}
        width={600}
      >
        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="configName" label="参数名称" rules={[{ required: true, message: '请输入参数名称' }]}>
            <Input placeholder="请输入参数名称" />
          </Form.Item>
          <Form.Item name="configKey" label="参数键名" rules={[{ required: true, message: '请输入参数键名' }]} extra="建议格式：sys.user.initPassword">
            <Input placeholder="请输入参数键名" disabled={!!editingConfig && editingConfig.isSystem === 1} />
          </Form.Item>
          <Form.Item name="configValue" label="参数键值" rules={[{ required: true, message: '请输入参数键值' }]}>
            <Input placeholder="请输入参数键值" />
          </Form.Item>
          <Form.Item name="configType" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value="Y">是</Option>
              <Option value="N">否</Option>
            </Select>
          </Form.Item>
          <Form.Item name="isSystem" label="系统内置" rules={[{ required: true }]} extra="系统内置参数不可删除">
            <Select disabled={!!editingConfig}>
              <Option value={0}>否</Option>
              <Option value={1}>是</Option>
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

export default ConfigManagement;

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
import type { Config } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

const ConfigManagement: React.FC = () => {
  const [configs, setConfigs] = useState<Config[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<Config | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 加载参数配置列表
  const loadConfigs = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/configs/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setConfigs(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载参数配置失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 创建参数配置
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingConfig) {
        await post(`/system/configs/${editingConfig.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/configs', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      editForm.resetFields();
      setEditingConfig(null);
      loadConfigs();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败');
    }
  };

  // 删除参数配置
  const handleDelete = async (id: string, isSystem: number) => {
    if (isSystem === 1) {
      message.warning('系统内置参数不能删除');
      return;
    }
    try {
      await post(`/system/configs/${id}/delete`, {});
      message.success('删除成功');
      loadConfigs();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除参数配置
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/configs/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadConfigs();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 刷新缓存
  const handleRefreshCache = async () => {
    try {
      await post('/system/configs/refresh-cache', {});
      message.success('刷新缓存成功');
    } catch (error) {
      message.error('刷新缓存失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Config) => {
    if (record) {
      setEditingConfig(record);
      editForm.setFieldsValue(record);
    } else {
      setEditingConfig(null);
      editForm.resetFields();
      editForm.setFieldsValue({ configType: 'N', isSystem: 0 });
    }
    setModalVisible(true);
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
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>编辑</Button>
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

  useEffect(() => {
    loadConfigs();
  }, [current, pageSize]);

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
            <Button type="primary" icon={<SearchOutlined />} onClick={() => loadConfigs()}>查询</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadConfigs(); }}>重置</Button>
          </Space>
        </Form.Item>
      </Form>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增</Button>
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
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); editForm.resetFields(); setEditingConfig(null); }}
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

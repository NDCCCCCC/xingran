import { useState, useEffect } from 'react';
import type { FC, Key } from 'react';
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
  Tabs,
  InputNumber,
  Switch,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { DictType, DictData } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TabPane } = Tabs;
const { TextArea } = Input;

const DictManagement: FC = () => {
  const [dictTypes, setDictTypes] = useState<DictType[]>([]);
  const [dictDataList, setDictDataList] = useState<DictData[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [dataSearchForm] = Form.useForm();
  const [typeModalVisible, setTypeModalVisible] = useState(false);
  const [dataModalVisible, setDataModalVisible] = useState(false);
  const [typeForm] = Form.useForm();
  const [dataForm] = Form.useForm();
  const [editingType, setEditingType] = useState<DictType | null>(null);
  const [editingData, setEditingData] = useState<DictData | null>(null);
  const [selectedType, setSelectedType] = useState<string>('');
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [activeTab, setActiveTab] = useState('type');

  // 加载字典类型列表
  const loadDictTypes = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/dicts/types/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setDictTypes(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载字典类型失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载字典数据列表
  const loadDictData = async (params: any = {}) => {
    if (!selectedType) {
      message.warning('请先选择字典类型');
      return;
    }
    setLoading(true);
    try {
      const values = dataSearchForm.getFieldsValue();
      const result = await post('/system/dicts/data/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        dictType: selectedType,
        ...values,
      });
      setDictDataList(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载字典数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 创建字典类型
  const handleCreateType = async () => {
    try {
      const values = await typeForm.validateFields();
      if (editingType) {
        await post(`/system/dicts/types/${editingType.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/dicts/types', values);
        message.success('创建成功');
      }
      setTypeModalVisible(false);
      typeForm.resetFields();
      setEditingType(null);
      loadDictTypes();
    } catch (error: any) {
      if (error.errorFields) {
        return; // 表单验证错误
      }
      message.error('操作失败');
    }
  };

  // 删除字典类型
  const handleDeleteType = async (id: string) => {
    try {
      await post(`/system/dicts/types/${id}/delete`, {});
      message.success('删除成功');
      loadDictTypes();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除字典类型
  const handleBatchDeleteType = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/dicts/types/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadDictTypes();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 创建字典数据
  const handleCreateData = async () => {
    try {
      const values = await dataForm.validateFields();
      if (editingData) {
        await post(`/system/dicts/data/${editingData.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/dicts/data', { ...values, dictType: selectedType });
        message.success('创建成功');
      }
      setDataModalVisible(false);
      dataForm.resetFields();
      setEditingData(null);
      loadDictData();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败');
    }
  };

  // 删除字典数据
  const handleDeleteData = async (id: string) => {
    try {
      await post(`/system/dicts/data/${id}/delete`, {});
      message.success('删除成功');
      loadDictData();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除字典数据
  const handleBatchDeleteData = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/dicts/data/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadDictData();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 刷新缓存
  const handleRefreshCache = async () => {
    try {
      await post('/system/dicts/refresh-cache', {});
      message.success('刷新缓存成功');
    } catch (error) {
      message.error('刷新缓存失败');
    }
  };

  // 打开编辑字典类型模态框
  const openTypeModal = (record?: DictType) => {
    if (record) {
      setEditingType(record);
      typeForm.setFieldsValue(record);
    } else {
      setEditingType(null);
      typeForm.resetFields();
      typeForm.setFieldsValue({ status: 0 });
    }
    setTypeModalVisible(true);
  };

  // 打开编辑字典数据模态框
  const openDataModal = (record?: DictData) => {
    if (!selectedType) {
      message.warning('请先选择字典类型');
      return;
    }
    if (record) {
      setEditingData(record);
      dataForm.setFieldsValue(record);
    } else {
      setEditingData(null);
      dataForm.resetFields();
      dataForm.setFieldsValue({ dictSort: 0, status: 0, isDefault: false });
    }
    setDataModalVisible(true);
  };

  // 字典类型表格列
  const typeColumns: ColumnsType<DictType> = [
    { title: '字典名称', dataIndex: 'dictName', key: 'dictName' },
    { title: '字典类型', dataIndex: 'dictType', key: 'dictType' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: number) => (
      <Tag color={status === 0 ? 'success' : 'error'}>{status === 0 ? '启用' : '禁用'}</Tag>
    )},
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openTypeModal(record)}>编辑</Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDeleteType(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 字典数据表格列
  const dataColumns: ColumnsType<DictData> = [
    { title: '字典标签', dataIndex: 'dictLabel', key: 'dictLabel' },
    { title: '字典键值', dataIndex: 'dictValue', key: 'dictValue' },
    { title: '字典排序', dataIndex: 'dictSort', key: 'dictSort' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: number) => (
      <Tag color={status === 0 ? 'success' : 'error'}>{status === 0 ? '启用' : '禁用'}</Tag>
    )},
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openDataModal(record)}>编辑</Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDeleteData(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    if (activeTab === 'type') {
      loadDictTypes();
    }
  }, [activeTab, current, pageSize]);

  useEffect(() => {
    if (activeTab === 'data' && selectedType) {
      loadDictData();
    }
  }, [activeTab, selectedType, current, pageSize]);

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="字典类型" key="type">
          <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
            <Form.Item name="dictName" label="字典名称">
              <Input placeholder="请输入字典名称" allowClear />
            </Form.Item>
            <Form.Item name="dictType" label="字典类型">
              <Input placeholder="请输入字典类型" allowClear />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
                <Option value={0}>启用</Option>
                <Option value={1}>禁用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={() => loadDictTypes()}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadDictTypes(); }}>重置</Button>
              </Space>
            </Form.Item>
          </Form>

          <div style={{ marginBottom: 16 }}>
            <Space>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openTypeModal()}>新增</Button>
              <Button danger icon={<DeleteOutlined />} onClick={handleBatchDeleteType} disabled={selectedRowKeys.length === 0}>批量删除</Button>
              <Button icon={<ReloadOutlined />} onClick={handleRefreshCache}>刷新缓存</Button>
            </Space>
          </div>

          <Table
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
            }}
            columns={typeColumns}
            dataSource={dictTypes}
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
        </TabPane>

        <TabPane tab="字典数据" key="data">
          <Form form={dataSearchForm} layout="inline" style={{ marginBottom: 16 }}>
            <Form.Item label="字典类型">
              <Select placeholder="请选择字典类型" value={selectedType} onChange={setSelectedType} style={{ width: 200 }}>
                {dictTypes.filter(t => t.status === 0).map(t => (
                  <Option key={t.id} value={t.dictType}>{t.dictName}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="dictLabel" label="字典标签">
              <Input placeholder="请输入字典标签" allowClear />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
                <Option value={0}>启用</Option>
                <Option value={1}>禁用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={() => loadDictData()}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { dataSearchForm.resetFields(); loadDictData(); }}>重置</Button>
              </Space>
            </Form.Item>
          </Form>

          <div style={{ marginBottom: 16 }}>
            <Space>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openDataModal()} disabled={!selectedType}>新增</Button>
              <Button danger icon={<DeleteOutlined />} onClick={handleBatchDeleteData} disabled={selectedRowKeys.length === 0}>批量删除</Button>
            </Space>
          </div>

          <Table
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
            }}
            columns={dataColumns}
            dataSource={dictDataList}
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
        </TabPane>
      </Tabs>

      {/* 字典类型编辑模态框 */}
      <Modal
        title={editingType ? '编辑字典类型' : '新增字典类型'}
        open={typeModalVisible}
        onOk={handleCreateType}
        onCancel={() => { setTypeModalVisible(false); typeForm.resetFields(); setEditingType(null); }}
        width={600}
      >
        <Form form={typeForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="dictName" label="字典名称" rules={[{ required: true, message: '请输入字典名称' }]}>
            <Input placeholder="请输入字典名称" />
          </Form.Item>
          <Form.Item name="dictType" label="字典类型" rules={[{ required: true, message: '请输入字典类型' }]}>
            <Input placeholder="请输入字典类型" disabled={!!editingType} />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value={0}>启用</Option>
              <Option value={1}>禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 字典数据编辑模态框 */}
      <Modal
        title={editingData ? '编辑字典数据' : '新增字典数据'}
        open={dataModalVisible}
        onOk={handleCreateData}
        onCancel={() => { setDataModalVisible(false); dataForm.resetFields(); setEditingData(null); }}
        width={600}
      >
        <Form form={dataForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="dictSort" label="字典排序" rules={[{ required: true, message: '请输入字典排序' }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="dictLabel" label="字典标签" rules={[{ required: true, message: '请输入字典标签' }]}>
            <Input placeholder="请输入字典标签" />
          </Form.Item>
          <Form.Item name="dictValue" label="字典键值" rules={[{ required: true, message: '请输入字典键值' }]}>
            <Input placeholder="请输入字典键值" />
          </Form.Item>
          <Form.Item name="cssClass" label="CSS类名">
            <Input placeholder="请输入CSS类名" />
          </Form.Item>
          <Form.Item name="listClass" label="列表类名">
            <Input placeholder="请输入列表类名" />
          </Form.Item>
          <Form.Item name="isDefault" label="是否默认" valuePropName="checked">
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value={0}>启用</Option>
              <Option value={1}>禁用</Option>
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

export default DictManagement;

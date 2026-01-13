import { useState, useEffect, useCallback } from 'react';
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
  Card,
  Row,
  Col,
  Statistic,
  Alert,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from '@ant-design/icons';
import type { DictType, DictData } from '@/types';
import { post } from '@/lib/api';
import ActionButtons from '@/components/shared/ActionButtons';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';

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
  const [typeStatistics, setTypeStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [dataStatistics, setDataStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

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

  // 加载字典类型统计数据
  const loadTypeStatistics = useCallback(async () => {
    try {
      const result = await post('/system/dicts/types/list', { current: 1, pageSize: 1000 });
      const allTypes = result.data.list || [];
      const activeCount = allTypes.filter((t: any) => t.status === 0).length;
      const inactiveCount = allTypes.filter((t: any) => t.status === 1).length;
      setTypeStatistics({
        total: allTypes.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  // 加载字典数据统计数据
  const loadDataStatistics = useCallback(async () => {
    if (!selectedType) {
      setDataStatistics({ total: 0, active: 0, inactive: 0 });
      return;
    }
    try {
      const result = await post('/system/dicts/data/list', { current: 1, pageSize: 1000, dictType: selectedType });
      const allData = result.data.list || [];
      const activeCount = allData.filter((d: any) => d.status === 0).length;
      const inactiveCount = allData.filter((d: any) => d.status === 1).length;
      setDataStatistics({
        total: allData.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, [selectedType]);

  // 创建字典类型
  const handleCreateType = async () => {
    try {
      const values = await typeForm.validateFields();
      if (editingType) {
        await post(`/system/dicts/types/${editingType.id}/update`, values);
        handleSuccess('更新');
      } else {
        await post('/system/dicts/types', values);
        handleSuccess('创建');
      }
      setTypeModalVisible(false);
      typeForm.resetFields();
      setEditingType(null);
      loadDictTypes();
      loadTypeStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return; // 表单验证错误
      }
      handleApiError(error, '操作');
    }
  };

  // 删除字典类型
  const handleDeleteType = async (id: string) => {
    try {
      await post(`/system/dicts/types/${id}/delete`, {});
      handleSuccess('删除');
      loadDictTypes();
      loadTypeStatistics();
    } catch (error) {
      handleApiError(error, '删除');
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
      handleSuccess('批量删除');
      setSelectedRowKeys([]);
      loadDictTypes();
      loadTypeStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  };

  // 创建字典数据
  const handleCreateData = async () => {
    try {
      const values = await dataForm.validateFields();
      if (editingData) {
        await post(`/system/dicts/data/${editingData.id}/update`, values);
        handleSuccess('更新');
      } else {
        await post('/system/dicts/data', { ...values, dictType: selectedType });
        handleSuccess('创建');
      }
      setDataModalVisible(false);
      dataForm.resetFields();
      setEditingData(null);
      loadDictData();
      loadDataStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      handleApiError(error, '操作');
    }
  };

  // 删除字典数据
  const handleDeleteData = async (id: string) => {
    try {
      await post(`/system/dicts/data/${id}/delete`, {});
      handleSuccess('删除');
      loadDictData();
      loadDataStatistics();
    } catch (error) {
      handleApiError(error, '删除');
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
      handleSuccess('批量删除');
      setSelectedRowKeys([]);
      loadDictData();
      loadDataStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
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
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => openTypeModal(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm title="确认删除?" onConfirm={() => handleDeleteType(record.id)}>
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
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => openDataModal(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm title="确认删除?" onConfirm={() => handleDeleteData(record.id)}>
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

  useEffect(() => {
    if (activeTab === 'type') {
      loadDictTypes();
      loadTypeStatistics();
    }
  }, [activeTab, current, pageSize, loadTypeStatistics]);

  useEffect(() => {
    if (activeTab === 'data' && selectedType) {
      loadDictData();
      loadDataStatistics();
    }
  }, [activeTab, selectedType, current, pageSize, loadDataStatistics]);

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="字典类型" key="type">
          {/* 统计卡片 - 只在总数大于10时显示 */}
          {typeStatistics.total > 10 && (
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="总类型数"
                    value={typeStatistics.total}
                    prefix={<CheckCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="正常类型"
                    value={typeStatistics.active}
                    styles={{ content: { color: '#3f8600' } }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="停用类型"
                    value={typeStatistics.inactive}
                    styles={{ content: { color: '#cf1322' } }}
                    prefix={<StopOutlined />}
                  />
                </Card>
              </Col>
            </Row>
          )}

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
              <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item name="dictName" label="字典名称">
                  <Input placeholder="请输入字典名称" allowClear className="user-form-input" style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="dictType" label="字典类型">
                  <Input placeholder="请输入字典类型" allowClear className="user-form-input" style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
                    <Option value={0}>启用</Option>
                    <Option value={1}>禁用</Option>
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={() => loadDictTypes()}>
                      搜索
                    </Button>
                    <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadDictTypes(); loadTypeStatistics(); }}>
                      刷新
                    </Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                {selectedRowKeys.length > 0 && (
                  <Button
                    danger
                    icon={<DeleteOutlined />}
                    onClick={handleBatchDeleteType}
                  >
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                )}
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openTypeModal()}>
                  新增类型
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefreshCache}>
                  刷新缓存
                </Button>
              </Space>
            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                message={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个字典类型，
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setSelectedRowKeys([])}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </Card>

          {/* 数据表格 */}
          <Card>
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
                onChange: (page, size) => {
                  setCurrent(page);
                  setPageSize(size || 10);
                  loadDictTypes({ current: page, pageSize: size || 10 });
                },
              }}
            />
          </Card>
        </TabPane>

        <TabPane tab="字典数据" key="data">
          {/* 统计卡片 - 只在总数大于10时显示 */}
          {dataStatistics.total > 10 && (
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="总数据数"
                    value={dataStatistics.total}
                    prefix={<CheckCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="正常数据"
                    value={dataStatistics.active}
                    styles={{ content: { color: '#3f8600' } }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card>
                  <Statistic
                    title="停用数据"
                    value={dataStatistics.inactive}
                    styles={{ content: { color: '#cf1322' } }}
                    prefix={<StopOutlined />}
                  />
                </Card>
              </Col>
            </Row>
          )}

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
              <Form form={dataSearchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item label="字典类型">
                  <Select placeholder="请选择字典类型" value={selectedType} onChange={setSelectedType} style={{ width: 200 }}>
                    {dictTypes.filter(t => t.status === 0).map(t => (
                      <Option key={t.id} value={t.dictType}>{t.dictName}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item name="dictLabel" label="字典标签">
                  <Input placeholder="请输入字典标签" allowClear className="user-form-input" style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
                    <Option value={0}>启用</Option>
                    <Option value={1}>禁用</Option>
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={() => loadDictData()}>
                      搜索
                    </Button>
                    <Button icon={<ReloadOutlined />} onClick={() => { dataSearchForm.resetFields(); loadDictData(); loadDataStatistics(); }}>
                      刷新
                    </Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                {selectedRowKeys.length > 0 && (
                  <Button
                    danger
                    icon={<DeleteOutlined />}
                    onClick={handleBatchDeleteData}
                  >
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                )}
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openDataModal()} disabled={!selectedType}>
                  新增数据
                </Button>
              </Space>
            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                message={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个字典数据，
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setSelectedRowKeys([])}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </Card>

          {/* 数据表格 */}
          <Card>
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
                onChange: (page, size) => {
                  setCurrent(page);
                  setPageSize(size || 10);
                  loadDictData({ current: page, pageSize: size || 10 });
                },
              }}
            />
          </Card>
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
        <Form form={typeForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item name="dictName" label="字典名称" rules={[{ required: true, message: '请输入字典名称' }]}>
            <Input placeholder="请输入字典名称" className="user-form-input" />
          </Form.Item>
          <Form.Item name="dictType" label="字典类型" rules={[{ required: true, message: '请输入字典类型' }]}>
            <Input placeholder="请输入字典类型" disabled={!!editingType} className="user-form-input" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input">
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
        <Form form={dataForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item name="dictSort" label="字典排序" rules={[{ required: true, message: '请输入字典排序' }]}>
            <InputNumber min={0} style={{ width: '100%' }} placeholder="请输入字典排序" />
          </Form.Item>
          <Form.Item name="dictLabel" label="字典标签" rules={[{ required: true, message: '请输入字典标签' }]}>
            <Input placeholder="请输入字典标签" className="user-form-input" />
          </Form.Item>
          <Form.Item name="dictValue" label="字典键值" rules={[{ required: true, message: '请输入字典键值' }]}>
            <Input placeholder="请输入字典键值" className="user-form-input" />
          </Form.Item>
          <Form.Item name="cssClass" label="CSS类名">
            <Input placeholder="请输入CSS类名" className="user-form-input" />
          </Form.Item>
          <Form.Item name="listClass" label="列表类名">
            <Input placeholder="请输入列表类名" className="user-form-input" />
          </Form.Item>
          <Form.Item name="isDefault" label="是否默认" valuePropName="checked">
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input">
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

import type { FC } from 'react';
import { useState, useCallback, useEffect } from 'react';
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
  Card,
  Row,
  Col,
  Statistic,
  Alert,
  message,
  Switch,
  Divider,
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
  LockOutlined,
  UnlockOutlined,
  SafetyOutlined,
} from '@ant-design/icons';
import type { Config } from '@/types';
import { formatDateTime } from '@/utils/datetime';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess, isFormValidationError } from '@/utils/errorHandler';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;
const { TextArea } = Input;

const ConfigManagement: FC = () => {
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 请求加密状态
  const [encryptionEnabled, setEncryptionEnabled] = useState<boolean>(true);
  const [encryptionConfig, setEncryptionConfig] = useState<Config | null>(null);
  const [encryptionLoading, setEncryptionLoading] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: configs,
    total,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingConfig,
    loadData: loadConfigs,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
  } = useTableManager<Config>(
    async (params): Promise<{ list: Config[]; total: number }> => {
      const values = searchForm.getFieldsValue() as Record<string, unknown>;
      const result = await post<{ list: Config[]; total: number }>('/system/configs/list', {
        current: params.current || paginationProps.current,
        pageSize: params.pageSize || paginationProps.pageSize,
        ...values,
      });
      return result.data || { list: [], total: 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  // 加载请求加密配置
  const loadEncryptionConfig = useCallback(async () => {
    try {
      const result = await post<Config>('/system/configs/getByKey', { configKey: 'sys.request.encryption.enabled' });
      const config = result.data;
      if (config) {
        setEncryptionConfig(config);
        const isEnabled = config.configValue === 'true' || config.configValue === '1';
        setEncryptionEnabled(isEnabled);
      }
    } catch (error) {
      // 如果配置不存在，使用默认值 true
      setEncryptionEnabled(true);
    }
  }, []);

  // 切换请求加密状态
  const handleToggleEncryption = async (enabled: boolean) => {
    setEncryptionLoading(true);
    try {
      if (encryptionConfig) {
        // 更新现有配置
        await post(`/system/configs/${encryptionConfig.id}/update`, {
          configValue: enabled ? 'true' : 'false',
        });
      } else {
        // 创建新配置（不应该发生，因为迁移已创建）
        await post('/system/configs', {
          configName: '请求加密开关',
          configKey: 'sys.request.encryption.enabled',
          configValue: enabled ? 'true' : 'false',
          configType: 'Y',
          isSystem: 1,
          remark: '控制请求体加密功能的启停。true=启用加密，false=禁用加密',
        });
      }

      // 刷新加密配置缓存
      await post('/system/configs/refresh-cache', {});

      setEncryptionEnabled(enabled);
      message.success(`请求加密已${enabled ? '启用' : '禁用'}，配置将在 30 秒内生效`);

      // 重新加载配置数据
      loadConfigs();
    } catch (error) {
      handleApiError(error, '切换请求加密状态');
      // 恢复开关状态
      setEncryptionEnabled(!enabled);
    } finally {
      setEncryptionLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{ list: Config[] }>('/system/configs/list', { current: 1, pageSize: 1000 });
      const allConfigs = result.data?.list || [];
      const activeCount = allConfigs.filter((c: Config) => c.configType === 'Y').length;
      const inactiveCount = allConfigs.filter((c: Config) => c.configType === 'N').length;
      setStatistics({
        total: allConfigs.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  useEffect(() => {
    loadConfigs();
    loadStatistics();
    loadEncryptionConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 创建参数配置
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingConfig) {
        // 更新时只发送必要的字段，移除不可修改的字段
        const { configKey, isSystem, ...updateData } = values;
        await post(`/system/configs/${editingConfig.id}/update`, updateData);
        handleSuccess('更新');
      } else {
        await post('/system/configs', values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadConfigs();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return; // 表单验证错误
      }
      handleApiError(error, '操作');
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
      handleSuccess('删除');
      loadConfigs();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '删除');
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
      handleSuccess('批量删除');
      setSelectedRowKeys([]);
      loadConfigs();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  };

  // 刷新缓存
  const handleRefreshCache = async () => {
    try {
      await post('/system/configs/refresh-cache', {});
      handleSuccess('刷新缓存');
    } catch (error) {
      handleApiError(error, '刷新缓存');
    }
  };

  // 表格列
  const columns: ColumnsType<Config> = [
    {
      title: '参数名称',
      dataIndex: 'configName',
      key: 'configName',
      width: 200,
      minWidth: 180,
      render: (configName: string, record: Config) => {
        const isEncryptionConfig = record.configKey === 'sys.request.encryption.enabled';
        return (
          <Space>
            {configName}
            {isEncryptionConfig && (
              <Tag color="purple" icon={<SafetyOutlined />}>
                加密开关
              </Tag>
            )}
          </Space>
        );
      },
    },
    { title: '参数键名', dataIndex: 'configKey', key: 'configKey', width: 220, minWidth: 200 },
    { title: '参数键值', dataIndex: 'configValue', key: 'configValue', width: 180, minWidth: 150, ellipsis: true },
    {
      title: '系统内置',
      dataIndex: 'isSystem',
      key: 'isSystem',
      width: 100,
      minWidth: 90,
      align: 'center' as const,
      render: (isSystem: number) => (
        <Tag color={isSystem === 1 ? 'blue' : 'default'}>{isSystem === 1 ? '是' : '否'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'configType',
      key: 'configType',
      width: 80,
      minWidth: 70,
      align: 'center' as const,
      render: (configType: string) => (
        <Tag color={configType === 'Y' ? 'success' : 'default'}>{configType === 'Y' ? '是' : '否'}</Tag>
      ),
    },
    { title: '备注', dataIndex: 'remark', key: 'remark', width: 150, minWidth: 120, ellipsis: true },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      minWidth: 130,
      fixed: 'right' as const,
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            disabled: record.isSystem === 1,
            render: () => (
              <Popconfirm
                title="确认删除?"
                onConfirm={() => handleDelete(record.id, record.isSystem)}
              >
                <Button type="link" icon={<DeleteOutlined />} style={{ color: '#ff4d4f' }} size="small" disabled={record.isSystem === 1}>
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

  return (
    <div>
      {/* 请求加密状态卡片 */}
      <Card
        style={{
          marginBottom: 16,
          border: encryptionEnabled ? '1px solid #52c41a' : '1px solid #ff4d4f',
          background: encryptionEnabled ? 'linear-gradient(135deg, #f6ffed 0%, #ffffff 100%)' : 'linear-gradient(135deg, #fff2f0 0%, #ffffff 100%)',
        }}
      >
        <Row gutter={16} align="middle">
          <Col span={16}>
            <Space direction="vertical" size={4}>
              <Space>
                {encryptionEnabled ? (
                  <SafetyOutlined style={{ fontSize: 24, color: '#52c41a' }} />
                ) : (
                  <UnlockOutlined style={{ fontSize: 24, color: '#ff4d4f' }} />
                )}
                <span style={{ fontSize: 16, fontWeight: 500 }}>
                  请求加密功能
                </span>
                <Tag color={encryptionEnabled ? 'success' : 'error'}>
                  {encryptionEnabled ? '已启用' : '已禁用'}
                </Tag>
              </Space>
              <div style={{ color: '#666', fontSize: 14 }}>
                {encryptionEnabled
                  ? '所有 POST/PUT/PATCH 请求将被强制使用 SM2+SM4 加密传输'
                  : '警告：请求未加密，敏感数据可能被拦截'}
              </div>
            </Space>
          </Col>
          <Col span={8} style={{ textAlign: 'right' }}>
            <Popconfirm
              title={encryptionEnabled ? '确认禁用请求加密？' : '确认启用请求加密？'}
              description={
                encryptionEnabled
                  ? '禁用后，所有请求将不再加密，仅用于调试环境'
                  : '启用后，所有请求必须使用 SM2+SM4 加密，否则将被拒绝'
              }
              onConfirm={() => handleToggleEncryption(!encryptionEnabled)}
              okText="确认"
              cancelText="取消"
            >
              <Button
                type={encryptionEnabled ? 'default' : 'primary'}
                danger={!encryptionEnabled}
                icon={encryptionEnabled ? <UnlockOutlined /> : <LockOutlined />}
                loading={encryptionLoading}
              >
                {encryptionEnabled ? '禁用加密' : '启用加密'}
              </Button>
            </Popconfirm>
          </Col>
        </Row>
      </Card>

      {/* 统计卡片 - 只在总数大于10时显示 */}
      {statistics.total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card>
              <Statistic
                title="总配置数"
                value={statistics.total}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="启用配置"
                value={statistics.active}
                styles={{ content: { color: '#3f8600' } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="禁用配置"
                value={statistics.inactive}
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
            <Form.Item name="configName" label="参数名称">
              <Input placeholder="请输入参数名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="configKey" label="参数键名">
              <Input placeholder="请输入参数键名" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="configType" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
                <Option value="Y">是</Option>
                <Option value="N">否</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadConfigs(); loadStatistics(); }}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                icon={<DeleteOutlined />} style={{ color: '#ff4d4f' }}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增配置
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
                已选择 <strong>{selectedRowKeys.length}</strong> 个参数配置，
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
            getCheckboxProps: (record) => ({
              disabled: record.isSystem === 1,
            }),
          }}
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={(pagination) => {
            const newPage = pagination.current ?? 1;
            const newPageSize = pagination.pageSize ?? 10;
            setCurrent(newPage);
            setPageSize(newPageSize);
            loadConfigs({
              current: newPage,
              pageSize: newPageSize,
            });
          }}
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={editingConfig ? '编辑参数配置' : '新增参数配置'}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={handleModalClose}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item name="configName" label="参数名称" rules={[{ required: true, message: '请输入参数名称' }]}>
            <Input placeholder="请输入参数名称" className="user-form-input" />
          </Form.Item>
          <Form.Item name="configKey" label="参数键名" rules={[{ required: true, message: '请输入参数键名' }]} extra="建议格式：sys.user.initPassword">
            <Input placeholder="请输入参数键名" disabled={!!editingConfig && editingConfig.isSystem === 1} className="user-form-input" />
          </Form.Item>
          <Form.Item name="configValue" label="参数键值" rules={[{ required: true, message: '请输入参数键值' }]}>
            <Input placeholder="请输入参数键值" className="user-form-input" />
          </Form.Item>
          <Form.Item name="configType" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input">
              <Option value="Y">是</Option>
              <Option value="N">否</Option>
            </Select>
          </Form.Item>
          <Form.Item name="isSystem" label="系统内置" rules={[{ required: true }]} extra="系统内置参数不可删除">
            <Select disabled={!!editingConfig} className="user-form-input">
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


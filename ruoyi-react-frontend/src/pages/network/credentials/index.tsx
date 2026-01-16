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
  Card,
  Row,
  Col,
  Statistic,
  Alert,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  KeyOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import type { AuthCredential, BaseResponse, PageResponse } from '@/types';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { withErrorHandling } from '@/utils/errorHandler';
import { useState, useEffect } from 'react';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;
const { TextArea } = Input;

const CredentialManagement: FC = () => {
  const [showPassword, setShowPassword] = useState(false);
  const [showEnablePassword, setShowEnablePassword] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    ssh: 0,
    telnet: 0,
  });

  const {
    loading,
    data: credentials,
    total,
    current,
    pageSize,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingCredential,
    setEditModalVisible,
    setEditingItem,
    loadData: loadCredentials,
    handleSearch,
    handleReset,
    handleRefresh,
    handleAdd,
    handleEdit,
    handleModalClose,
    setCurrent,
    setPageSize,
  } = useTableManager<AuthCredential>(
    async (params) => {
      const values = searchForm.getFieldsValue();
      // 映射搜索字段名到API字段名
      const searchParams: any = {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
      };
      if (values.searchCredentialName) {
        searchParams.credentialName = values.searchCredentialName;
      }
      if (values.searchProtocolType) {
        searchParams.protocolType = values.searchProtocolType;
      }
      const result = await post<BaseResponse<PageResponse<AuthCredential>>>('/network/credentials/list', searchParams);
      // 更新统计数据
      updateStatistics(result.data?.list || []);
      return result.data;
    }
  );

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post<BaseResponse<PageResponse<AuthCredential>>>('/network/credentials/list', { current: 1, pageSize: 10000 });
      const allCredentials = result.data?.list || [];
      updateStatistics(allCredentials);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (credentialList: AuthCredential[]) => {
    const sshCount = credentialList.filter(c => c.protocolType === 'ssh').length;
    const telnetCount = credentialList.filter(c => c.protocolType === 'telnet').length;
    setStatistics({
      total: credentialList.length,
      ssh: sshCount,
      telnet: telnetCount,
    });
  };

  useEffect(() => {
    loadCredentials();
    loadStatistics();
  }, []);

  // 创建/更新凭证
  const handleCreate = async () => {
    await withErrorHandling(
      async () => {
        const values = await editForm.validateFields();
        // 移除确认密码字段
        const { confirmPassword, ...data } = values as any;
        if (editingCredential) {
          await post(`/network/credentials/${editingCredential.id}/update`, data);
          return '更新成功';
        } else {
          await post('/network/credentials', data);
          return '创建成功';
        }
      },
      {
        onSuccess: () => {
          handleModalClose();
          setShowPassword(false);
          setShowEnablePassword(false);
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 删除凭证
  const handleDelete = async (id: string) => {
    await withErrorHandling(
      async () => {
        await post(`/network/credentials/${id}/delete`, {});
        return '删除成功';
      },
      {
        onSuccess: () => {
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 批量删除凭证
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    await withErrorHandling(
      async () => {
        await post('/network/credentials/batch-delete', { ids: selectedRowKeys });
        return '批量删除成功';
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 设置为默认凭证
  const handleSetDefault = async (id: string) => {
    await withErrorHandling(
      async () => {
        await post(`/network/credentials/${id}/set-default`, {});
        return '设置成功';
      },
      {
        onSuccess: () => {
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 打开编辑模态框
  const openModal = (record?: AuthCredential) => {
    if (record) {
      setEditingItem(record);
      editForm.setFieldsValue({
        ...record,
        // 确保 snmpVersion 有默认值
        snmpVersion: record.snmpVersion || 'v2c',
      });
    } else {
      setEditingItem(null);
      editForm.resetFields();
      editForm.setFieldsValue({
        protocolType: 'ssh',
        snmpVersion: 'v2c',
      });
    }
    setEditModalVisible(true);
    setShowPassword(false);
    setShowEnablePassword(false);
  };

  // 协议类型选项
  const protocolTypeOptions = [
    { label: 'SSH', value: 'ssh' },
    { label: 'Telnet', value: 'telnet' },
  ];

  // SNMP版本选项
  const snmpVersionOptions = [
    { label: 'v1', value: 'v1' },
    { label: 'v2c', value: 'v2c' },
    { label: 'v3', value: 'v3' },
  ];

  // 表格列
  const columns: ColumnsType<AuthCredential> = [
    { title: '凭证名称', dataIndex: 'credentialName', key: 'credentialName', width: 180 },
    {
      title: '协议类型',
      dataIndex: 'protocolType',
      key: 'protocolType',
      width: 100,
      render: (protocolType: string) => (
        <Tag color="blue">{protocolType?.toUpperCase()}</Tag>
      ),
    },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
    {
      title: '密码',
      dataIndex: 'password',
      key: 'password',
      width: 100,
      render: (password: string) => password ? '******' : '-',
    },
    {
      title: 'SNMP Communities',
      dataIndex: 'snmpCommunities',
      key: 'snmpCommunities',
      width: 200,
      render: (snmpCommunities: string[]) => (
        snmpCommunities && snmpCommunities.length > 0
          ? snmpCommunities.map((c, i) => <Tag key={i}>{c}</Tag>)
          : '-'
      ),
    },
    {
      title: 'SNMP版本',
      dataIndex: 'snmpVersion',
      key: 'snmpVersion',
      width: 100,
      render: (snmpVersion: string) => snmpVersion?.toUpperCase() || '-',
    },
    {
      title: '默认凭证',
      dataIndex: 'isDefault',
      key: 'isDefault',
      width: 100,
      render: (isDefault: boolean) => (
        isDefault ? <Tag color="gold">默认</Tag> : <Tag>普通</Tag>
      ),
    },
    { title: '备注', dataIndex: 'description', key: 'description', width: 150, ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => {
        const actions = [
          {
            key: 'set-default',
            label: '设为默认',
            onClick: () => handleSetDefault(record.id),
            hidden: record.isDefault,
          },
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
            render: () => (
              <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
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

  // 监听协议类型变化，动态显示特权密码字段
  const protocolType = Form.useWatch('protocolType', editForm);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="凭证总数"
              value={statistics.total}
              prefix={<KeyOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="SSH凭证"
              value={statistics.ssh}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Telnet凭证"
              value={statistics.telnet}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="searchCredentialName" label="凭证名称">
              <Input placeholder="请输入凭证名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="searchProtocolType" label="协议类型">
              <Select placeholder="请选择协议类型" allowClear className="user-form-input" style={{ width: 120 }}>
                {protocolTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新增</Button>
            <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
          </Space>
        </div>
      </Card>

      {/* 凭证表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={credentials}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1400 }}
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
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={editingCredential ? '编辑授权凭证' : '新增授权凭证'}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={() => { handleModalClose(); setShowPassword(false); setShowEnablePassword(false); }}
        width={600}
      >
        <Alert
          message="提示"
          description="请选择协议类型（SSH 或 Telnet），并填写对应的配置信息。一个凭证可以同时包含 SSH/Telnet 配置和 SNMP 配置。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="credentialName" label="凭证名称" rules={[{ required: true, message: '请输入凭证名称' }]}>
            <Input placeholder="请输入凭证名称" />
          </Form.Item>
          <Form.Item name="protocolType" label="协议类型" rules={[{ required: true, message: '请选择协议类型' }]}>
            <Select placeholder="请选择协议类型 (SSH 或 Telnet)">
              {protocolTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          {/* SSH/Telnet 字段 - 始终显示 */}
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: !editingCredential, message: '请输入密码' }]}
            extra={editingCredential ? '留空表示不修改密码' : ''}
          >
            <Input.Password
              placeholder="请输入密码"
              visibilityToggle={{ visible: showPassword, onVisibleChange: setShowPassword }}
            />
          </Form.Item>
          {protocolType === 'ssh' && (
            <Form.Item name="enablePassword" label="特权密码">
              <Input.Password
                placeholder="请输入特权密码（可选）"
                visibilityToggle={{ visible: showEnablePassword, onVisibleChange: setShowEnablePassword }}
              />
            </Form.Item>
          )}

          {/* SNMP 字段 - 可选 */}
          <Form.Item
            name="snmpCommunities"
            label="SNMP Communities"
            extra="用于设备状态检查和自动发现，可留空"
          >
            <Select
              mode="tags"
              placeholder="请输入 SNMP Community，可添加多个"
              tokenSeparators={[',']}
            >
            </Select>
          </Form.Item>
          <Form.Item name="snmpVersion" label="SNMP版本" rules={[{ required: true }]}>
            <Select>
              {snmpVersionOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="description" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default CredentialManagement;

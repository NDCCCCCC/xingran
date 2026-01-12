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
  Card,
  Row,
  Col,
  Statistic,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  FileTextOutlined,
  CopyOutlined,
  EyeOutlined,
  AppstoreOutlined,
  SettingOutlined,
  FundOutlined,
} from '@ant-design/icons';
import type { ConfigTemplate } from '@/types';
import { post } from '@/lib/api';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;
const { TextArea } = Input;

const TemplateManagement: FC = () => {
  const [templates, setTemplates] = useState<ConfigTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [variablesModalVisible, setVariablesModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingTemplate, setEditingTemplate] = useState<ConfigTemplate | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [previewContent, setPreviewContent] = useState('');
  const [templateVariables, setTemplateVariables] = useState<Record<string, any>>({});

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    system: 0,
    custom: 0,
    init: 0,
  });

  // 统一错误处理
  const handleApiError = (error: unknown, defaultMessage: string) => {
    if (error && typeof error === 'object' && 'message' in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  };

  const handleSuccess = (msg: string) => {
    message.success(msg);
  };

  // 加载模板列表
  const loadTemplates = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/network/templates/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setTemplates(result.data.list);
      setTotal(result.data.total);
      // 更新统计数据
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载模板列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/templates/list', { current: 1, pageSize: 10000 });
      const allTemplates = result.data.list || [];
      updateStatistics(allTemplates);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (templateList: ConfigTemplate[]) => {
    const systemCount = templateList.filter(t => t.isSystem).length;
    const customCount = templateList.filter(t => !t.isSystem).length;
    const initCount = templateList.filter(t => t.templateType === 'init').length;
    setStatistics({
      total: templateList.length,
      system: systemCount,
      custom: customCount,
      init: initCount,
    });
  };

  // 创建/更新模板
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      const data = {
        ...values,
        variables: values.variables ? JSON.parse(values.variables) : undefined,
      };
      if (editingTemplate) {
        await post(`/network/templates/${editingTemplate.id}/update`, data);
        handleSuccess('更新成功');
      } else {
        await post('/network/templates', data);
        handleSuccess('创建成功');
      }
      setModalVisible(false);
      editForm.resetFields();
      setEditingTemplate(null);
      loadTemplates();
      loadStatistics();
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) {
        return;
      }
      handleApiError(error, '操作失败');
    }
  };

  // 删除模板
  const handleDelete = async (id: string) => {
    try {
      await post(`/network/templates/${id}/delete`, {});
      handleSuccess('删除成功');
      loadTemplates();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '删除失败');
    }
  };

  // 批量删除模板
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/network/templates/batch-delete', { ids: selectedRowKeys });
      handleSuccess('批量删除成功');
      setSelectedRowKeys([]);
      loadTemplates();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '批量删除失败');
    }
  };

  // 预览模板
  const handlePreview = async (id: string) => {
    try {
      const result = await post(`/network/templates/${id}/preview`, {});
      setPreviewContent(result.data.content || '');
      setPreviewVisible(true);
    } catch (error) {
      handleApiError(error, '预览失败');
    }
  };

  // 克隆模板
  const handleClone = async (id: string) => {
    try {
      await post(`/network/templates/${id}/clone`, {});
      handleSuccess('克隆成功');
      loadTemplates();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '克隆失败');
    }
  };

  // 获取模板变量
  const handleGetVariables = async (id: string) => {
    try {
      const result = await post(`/network/templates/${id}/variables`, {});
      setTemplateVariables(result.data.variables || {});
      setVariablesModalVisible(true);
    } catch (error) {
      handleApiError(error, '获取变量失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: ConfigTemplate) => {
    if (record) {
      setEditingTemplate(record);
      editForm.setFieldsValue({
        ...record,
        variables: record.variables ? JSON.stringify(record.variables, null, 2) : undefined,
      });
    } else {
      setEditingTemplate(null);
      editForm.resetFields();
      editForm.setFieldsValue({ templateType: 'config' });
    }
    setModalVisible(true);
  };

  // 搜索
  const handleSearch = () => {
    loadTemplates();
  };

  // 重置
  const handleReset = () => {
    searchForm.resetFields();
    loadTemplates();
  };

  // 刷新
  const handleRefresh = () => {
    loadTemplates();
    loadStatistics();
  };

  // 厂商选项
  const vendorOptions = [
    { label: 'Huawei', value: 'huawei' },
    { label: 'H3C', value: 'h3c' },
    { label: 'Ruijie', value: 'ruijie' },
    { label: 'Maipu', value: 'maipu' },
  ];

  // 设备类型选项
  const deviceTypeOptions = [
    { label: '路由器', value: 'router' },
    { label: '交换机', value: 'switch' },
    { label: '防火墙', value: 'firewall' },
    { label: '无线AP', value: 'ap' },
    { label: '负载均衡器', value: 'loadbalancer' },
  ];

  // 模板类型选项
  const templateTypeOptions = [
    { label: '初始化配置', value: 'init' },
    { label: '业务配置', value: 'config' },
    { label: '备份配置', value: 'backup' },
  ];

  // 表格列
  const columns: ColumnsType<ConfigTemplate> = [
    { title: '模板名称', dataIndex: 'templateName', key: 'templateName', width: 180 },
    { title: '模板编码', dataIndex: 'templateCode', key: 'templateCode', width: 150 },
    {
      title: '模板类型',
      dataIndex: 'templateType',
      key: 'templateType',
      width: 120,
      render: (templateType: string) => {
        const option = templateTypeOptions.find(o => o.value === templateType);
        return <Tag color="blue">{option?.label}</Tag>;
      },
    },
    {
      title: '适用厂商',
      dataIndex: 'vendor',
      key: 'vendor',
      width: 100,
      render: (vendor: string) => {
        if (!vendor) return <Tag>通用</Tag>;
        const option = vendorOptions.find(o => o.value === vendor);
        return <Tag color="geekblue">{option?.label}</Tag>;
      },
    },
    {
      title: '适用设备',
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 120,
      render: (deviceType: string) => {
        if (!deviceType) return '-';
        const option = deviceTypeOptions.find(o => o.value === deviceType);
        return <Tag color="purple">{option?.label}</Tag>;
      },
    },
    {
      title: '系统模板',
      dataIndex: 'isSystem',
      key: 'isSystem',
      width: 100,
      render: (isSystem: boolean) => (
        isSystem ? <Tag color="gold">系统</Tag> : <Tag>自定义</Tag>
      ),
    },
    { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
    { title: '备注', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: 'preview',
            label: '预览',
            icon: <EyeOutlined />,
            onClick: () => handlePreview(record.id),
          },
          {
            key: 'clone',
            label: '克隆',
            icon: <CopyOutlined />,
            onClick: () => handleClone(record.id),
          },
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => openModal(record),
          },
          ...(!record.isSystem ? [{
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
          }] : []),
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  useEffect(() => {
    loadTemplates();
    loadStatistics();
  }, [current, pageSize]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="模板总数"
              value={statistics.total}
              prefix={<FileTextOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="系统模板"
              value={statistics.system}
              styles={{ content: { color: '#faad14' } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自定义模板"
              value={statistics.custom}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<SettingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="初始化模板"
              value={statistics.init}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<FundOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="templateName" label="模板名称">
              <Input placeholder="请输入模板名称" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="templateCode" label="模板编码">
              <Input placeholder="请输入模板编码" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="templateType" label="模板类型">
              <Select placeholder="请选择模板类型" allowClear style={{ width: 130 }}>
                {templateTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="vendor" label="厂商">
              <Select placeholder="请选择厂商" allowClear style={{ width: 120 }}>
                {vendorOptions.map(opt => (
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
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增</Button>
            <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
          </Space>
        </div>
      </Card>

      {/* 模板表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={templates}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
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
        title={editingTemplate ? '编辑配置模板' : '新增配置模板'}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); editForm.resetFields(); setEditingTemplate(null); }}
        width={900}
      >
        <Form form={editForm} labelCol={{ span: 4 }} wrapperCol={{ span: 18 }}>
          <Form.Item name="templateName" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input placeholder="请输入模板名称" />
          </Form.Item>
          <Form.Item name="templateCode" label="模板编码" rules={[{ required: true, message: '请输入模板编码' }]}>
            <Input placeholder="请输入模板编码（英文，用于API调用）" disabled={!!editingTemplate} />
          </Form.Item>
          <Form.Item name="templateType" label="模板类型" rules={[{ required: true }]}>
            <Select>
              {templateTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Space size="large">
            <Form.Item name="vendor" label="适用厂商">
              <Select placeholder="通用" allowClear style={{ width: 150 }}>
                {vendorOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="deviceType" label="适用设备">
              <Select placeholder="通用" allowClear style={{ width: 150 }}>
                {deviceTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
          <Form.Item name="templateContent" label="模板内容" rules={[{ required: true, message: '请输入模板内容' }]} extra="使用 {{.变量名}} 语法定义变量">
            <TextArea rows={15} placeholder={`示例配置模板：
interface {{.InterfaceName}}
 description {{.Description}}
 ip address {{.IPAddress}} {{.SubnetMask}}

!
vlan {{.VLANID}}
 name {{.VLANName}}
!
`} />
          </Form.Item>
          <Form.Item name="variables" label="变量定义" extra="JSON格式，定义变量的默认值和描述">
            <TextArea rows={6} placeholder={`示例：
{
  "InterfaceName": {"default": "GigabitEthernet0/0/1", "description": "接口名称"},
  "Description": {"default": "Uplink", "description": "接口描述"},
  "IPAddress": {"default": "192.168.1.1", "description": "IP地址"},
  "SubnetMask": {"default": "255.255.255.0", "description": "子网掩码"}
}`} />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={2} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 预览模态框 */}
      <Modal
        title="模板预览"
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={[
          <Button key="close" onClick={() => setPreviewVisible(false)}>关闭</Button>,
        ]}
        width={800}
      >
        <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, maxHeight: 500, overflow: 'auto' }}>
          {previewContent}
        </pre>
      </Modal>

      {/* 变量查看模态框 */}
      <Modal
        title="模板变量"
        open={variablesModalVisible}
        onCancel={() => setVariablesModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setVariablesModalVisible(false)}>关闭</Button>,
        ]}
        width={600}
      >
        {Object.keys(templateVariables).length > 0 ? (
          <Table
            dataSource={Object.entries(templateVariables).map(([key, value]: [string, any]) => ({
              key,
              description: value?.description || '-',
              defaultValue: value?.default || '-',
            }))}
            columns={[
              { title: '变量名', dataIndex: 'key', key: 'key' },
              { title: '描述', dataIndex: 'description', key: 'description' },
              { title: '默认值', dataIndex: 'defaultValue', key: 'defaultValue' },
            ]}
            pagination={false}
            size="small"
          />
        ) : (
          <p style={{ textAlign: 'center', color: '#999' }}>此模板没有定义变量</p>
        )}
      </Modal>
    </div>
  );
};

export default TemplateManagement;

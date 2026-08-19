import { useState, useEffect } from "react";
import type { FC } from "react";
import {
  App,
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Tag,
  InputNumber,
  Tabs,
  Radio,
  Empty,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ApiOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { formatDateTime } from "@/utils/datetime";
import {
  getAPINotificationConfigList,
  createAPINotificationConfig,
  updateAPINotificationConfig,
  deleteAPINotificationConfig,
  APIConfigTypes,
  AuthTypes,
  type APINotificationConfig,
  type APINotificationConfigCreateRequest,
  type APINotificationConfigUpdateRequest,
  type APIConfigType,
  type AuthType,
} from "@/lib/notificationConfigApi";
import { usePagination } from "@/hooks/usePagination";
import { isFormValidationError } from "@/utils/errorHandler";

const { Option } = Select;
const { TextArea } = Input;

const APIConfigPage: FC = () => {
  const { message } = App.useApp();
  // 列表相关状态
  const [configs, setConfigs] = useState<APINotificationConfig[]>([]);
  const [loading, setLoading] = useState(false);

  // 搜索筛选（configType + 名称 + 状态）
  const [searchForm] = Form.useForm();
  const [searchName, setSearchName] = useState<string>("");
  const [configTypeFilter, setConfigTypeFilter] = useState<APIConfigType | undefined>(undefined);
  const [statusFilter, setStatusFilter] = useState<number | undefined>(undefined);

  // 统计卡轻请求
  const [totalCount, setTotalCount] = useState(0);
  const [enabledCount, setEnabledCount] = useState(0);
  const [disabledCount, setDisabledCount] = useState(0);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<APINotificationConfig | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 统计卡轻请求：status 0/1 各一次只取 total（API 配置 0=启用/1=停用）
  const loadStatistics = async () => {
    try {
      const [allRes, enabledRes, disabledRes] = await Promise.all([
        getAPINotificationConfigList({ page: 1, pageSize: 1 }),
        getAPINotificationConfigList({ page: 1, pageSize: 1, status: 0 }),
        getAPINotificationConfigList({ page: 1, pageSize: 1, status: 1 }),
      ]);
      const data = allRes as { data: { total: number } };
      const eData = enabledRes as { data: { total: number } };
      const dData = disabledRes as { data: { total: number } };
      setTotalCount(data.data.total);
      setEnabledCount(eData.data.total);
      setDisabledCount(dData.data.total);
    } catch (error) {
      console.error("加载API配置统计失败:", error);
    }
  };

  // 加载API通知配置列表
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const result = (await getAPINotificationConfigList({
        page: paginationProps.current,
        pageSize: paginationProps.pageSize,
        configType: configTypeFilter,
        status: statusFilter,
      })) as { data: { list: APINotificationConfig[]; total: number } };
      setConfigs(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error("加载API配置失败:", error);
      message.error("加载API配置失败，请刷新重试");
    } finally {
      setLoading(false);
    }
  };

  // 打开编辑模态框
  const openModal = (record?: APINotificationConfig) => {
    if (record) {
      setEditingConfig(record);
      editForm.setFieldsValue({
        ...record,
      });
    } else {
      setEditingConfig(null);
      editForm.resetFields();
      editForm.setFieldsValue({
        apiMethod: "POST",
        authType: AuthTypes.NONE,
        retryCount: 3,
        timeout: 30,
        isDefault: false,
        status: 0,
      });
    }
    setModalVisible(true);
  };

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await editForm.validateFields();
      setSubmitting(true);

      if (editingConfig) {
        const updateData: APINotificationConfigUpdateRequest = { ...values };
        await updateAPINotificationConfig(editingConfig.id, updateData);
        message.success("更新成功");
      } else {
        const createData: APINotificationConfigCreateRequest = { ...values };
        await createAPINotificationConfig(createData);
        message.success("创建成功");
      }

      setModalVisible(false);
      editForm.resetFields();
      setEditingConfig(null);
      loadConfigs();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      message.error("操作失败: " + ((error as Error).message || "未知错误"));
    } finally {
      setSubmitting(false);
    }
  };

  // 删除配置
  const handleDelete = async (id: string) => {
    try {
      await deleteAPINotificationConfig(id);
      message.success("删除成功");
      loadConfigs();
      loadStatistics();
    } catch (_error) {
      message.error("删除失败");
    }
  };

  // 搜索处理
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    setSearchName(values.configName || "");
    setConfigTypeFilter(values.configType);
    setStatusFilter(values.status);
    setCurrent(1);
  };

  // 重置搜索
  const handleResetSearch = () => {
    searchForm.resetFields();
    setSearchName("");
    setConfigTypeFilter(undefined);
    setStatusFilter(undefined);
    setCurrent(1);
  };

  // 启用占比
  const enabledPct = totalCount > 0 ? Math.round((enabledCount / totalCount) * 100) : 0;

  // 类型 Tag 标签映射（统一走 xr-tag 中性 — 移除 preset cyan/purple/orange）
  const configTypeLabelMap: Record<APIConfigType, string> = {
    [APIConfigTypes.SMS]: "短信",
    [APIConfigTypes.WEBHOOK]: "Webhook",
    [APIConfigTypes.PUSH]: "推送",
  };

  // 认证方式 label
  const authTypeLabelMap: Record<AuthType, string> = {
    [AuthTypes.NONE]: "无",
    [AuthTypes.BASIC]: "Basic",
    [AuthTypes.BEARER]: "Bearer",
    [AuthTypes.APIKEY]: "API Key",
  };

  // 表格列定义
  const columns: ColumnsType<APINotificationConfig> = [
    {
      title: "配置名称",
      dataIndex: "configName",
      key: "configName",
      width: 180,
      render: (text: string, record) => (
        <Space size={4}>
          <span className="xr-cell-id">{text}</span>
          {record.isDefault && <span className="xr-tag xr-tag-gold">默认</span>}
        </Space>
      ),
    },
    {
      title: "类型",
      dataIndex: "configType",
      key: "configType",
      width: 100,
      render: (type: APIConfigType) => (
        <span className="xr-tag">{configTypeLabelMap[type] || type}</span>
      ),
    },
    {
      title: "API地址",
      dataIndex: "apiUrl",
      key: "apiUrl",
      ellipsis: true,
    },
    {
      title: "请求方法",
      dataIndex: "apiMethod",
      key: "apiMethod",
      width: 80,
    },
    {
      title: "认证方式",
      dataIndex: "authType",
      key: "authType",
      width: 100,
      render: (type: AuthType) => <span>{authTypeLabelMap[type] || type}</span>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      render: (status: number) => (
        <span className={`xr-tag ${status === 0 ? "xr-tag-green" : ""}`}>
          {status === 0 ? "正常" : "停用"}
        </span>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (date: string) => <span className="xr-cell-time">{formatDateTime(date)}</span>,
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 150,
      render: (_, record: APINotificationConfig) => (
        <div className="xr-row-ops">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            className="xr-op-danger"
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: "删除 API 配置？",
                content: "删除后不可恢复，使用该配置的通知推送将失败。",
                okText: "删除",
                okButtonProps: { danger: true },
                cancelText: "取消",
                onOk: () => handleDelete(record.id),
              });
            }}
          >
            删除
          </Button>
        </div>
      ),
    },
  ];

  useEffect(() => {
    loadConfigs();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize, configTypeFilter, statusFilter]);

  return (
    <>
      {/* 统计卡行：3 卡 — 总配置数 / 启用 / 停用（不加第 4 卡——configType 分布由工具栏筛选承载） */}
      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-label">总配置数</div>
          <div className="stat-value">{totalCount}</div>
          <div className="stat-trend">全部推送渠道</div>
        </div>
        <div className="stat-card sc-green">
          <div className="stat-label">启用</div>
          <div className="stat-value">{enabledCount}</div>
          <div className="stat-trend">占比 {enabledPct}%</div>
        </div>
        <div className="stat-card sc-gray">
          <div className="stat-label">停用</div>
          <div className="stat-value">{disabledCount}</div>
          <div className="stat-trend">可随时恢复</div>
        </div>
      </div>

      {/* 工具栏卡：配置名称 + configType 筛选 + 状态筛选 + 新增配置 */}
      <Card style={{ marginBottom: 14 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="configName" label="配置名称">
              <Input placeholder="按配置名称搜索" allowClear style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="configType" label="配置类型">
              <Select
                placeholder="全部类型"
                style={{ width: 140 }}
                allowClear
                onSearch={() => {}}
              >
                <Option value={APIConfigTypes.SMS}>短信</Option>
                <Option value={APIConfigTypes.WEBHOOK}>Webhook</Option>
                <Option value={APIConfigTypes.PUSH}>推送</Option>
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                placeholder="全部状态"
                style={{ width: 140 }}
                allowClear
                onSearch={() => {}}
              >
                <Option value={0}>正常</Option>
                <Option value={1}>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button onClick={handleResetSearch}>重置</Button>
                <Button type="primary" onClick={handleSearch}>
                  搜索
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新增配置
          </Button>
        </div>
      </Card>

      {/* 表格卡：双层纸感表格（.xr-table-zebra 绿灰表头 + 斑马纹） */}
      <Card>
        <Table
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1200 }}
          pagination={paginationProps}
          className="xr-table-zebra"
          size="middle"
          onChange={(pagination) => {
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
          }}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <Space direction="vertical" size={4}>
                    <span style={{ color: "var(--theme-text-primary)", fontWeight: 500 }}>
                      暂无 API 通知配置
                    </span>
                    <span style={{ color: "var(--theme-text-secondary)", fontSize: 12 }}>
                      新增 Webhook / 短信 / 推送渠道配置，用于系统通知推送
                    </span>
                  </Space>
                }
              />
            ),
          }}
        />
      </Card>

      {/* 编辑模态框 — D-09: 宽度 800 不变；Modal 内 headers/template/auth Tabs 原样保留 */}
      <Modal
        title={editingConfig ? "编辑API配置" : "新增API配置"}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingConfig(null);
        }}
        confirmLoading={submitting}
        width={800}
      >
        <Form form={editForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item
            name="configName"
            label="配置名称"
            rules={[{ required: true, message: "请输入配置名称" }]}
          >
            <Input placeholder="请输入配置名称，如：阿里云短信" />
          </Form.Item>

          <Form.Item
            name="configType"
            label="配置类型"
            rules={[{ required: true, message: "请选择配置类型" }]}
          >
            <Select placeholder="请选择配置类型" onSearch={() => {}}>
              <Option value={APIConfigTypes.SMS}>短信</Option>
              <Option value={APIConfigTypes.WEBHOOK}>Webhook</Option>
              <Option value={APIConfigTypes.PUSH}>推送</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="apiUrl"
            label="API地址"
            rules={[{ required: true, message: "请输入API地址" }]}
          >
            <Input placeholder="请输入API调用地址" />
          </Form.Item>

          <Form.Item name="apiMethod" label="请求方法" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio value="GET">GET</Radio>
              <Radio value="POST">POST</Radio>
            </Radio.Group>
          </Form.Item>

          <Tabs
            defaultActiveKey="headers"
            items={[
              {
                key: "headers",
                label: "请求头",
                children: (
                  <Form.Item name="headers" label="请求头配置">
                    <TextArea
                      rows={4}
                      placeholder='JSON格式，如：{"Content-Type": "application/json"}'
                    />
                  </Form.Item>
                ),
              },
              {
                key: "template",
                label: "模板内容",
                children: (
                  <Form.Item name="templateBody" label="模板内容">
                    <TextArea
                      rows={6}
                      placeholder='支持变量：{{title}}、{{content}}、{{recipients}}等
示例：{"title": "{{title}}", "content": "{{content}}"}'
                    />
                  </Form.Item>
                ),
              },
              {
                key: "auth",
                label: "认证配置",
                children: (
                  <>
                    <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
                      <Select onSearch={() => {}}>
                        <Option value={AuthTypes.NONE}>无需认证</Option>
                        <Option value={AuthTypes.BASIC}>Basic Auth</Option>
                        <Option value={AuthTypes.BEARER}>Bearer Token</Option>
                        <Option value={AuthTypes.APIKEY}>API Key</Option>
                      </Select>
                    </Form.Item>
                    <Form.Item
                      noStyle
                      shouldUpdate={(prev, curr) => prev.authType !== curr.authType}
                    >
                      {({ getFieldValue }) => {
                        const authType = getFieldValue("authType");
                        if (authType === AuthTypes.BASIC) {
                          return (
                            <>
                              <Form.Item name={["authConfig", "username"]} label="用户名">
                                <Input placeholder="请输入用户名" />
                              </Form.Item>
                              <Form.Item name={["authConfig", "password"]} label="密码">
                                <Input.Password placeholder="请输入密码" />
                              </Form.Item>
                            </>
                          );
                        }
                        if (authType === AuthTypes.BEARER) {
                          return (
                            <Form.Item name={["authConfig", "token"]} label="Token">
                              <Input placeholder="请输入Bearer Token" />
                            </Form.Item>
                          );
                        }
                        if (authType === AuthTypes.APIKEY) {
                          return (
                            <>
                              <Form.Item name={["authConfig", "key"]} label="Key名称">
                                <Input placeholder="如：X-API-Key" />
                              </Form.Item>
                              <Form.Item name={["authConfig", "value"]} label="Key值">
                                <Input placeholder="请输入API Key的值" />
                              </Form.Item>
                            </>
                          );
                        }
                        return null;
                      }}
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />

          <Form.Item name="retryCount" label="重试次数" initialValue={3}>
            <InputNumber min={0} max={10} className="w-full" />
          </Form.Item>

          <Form.Item name="timeout" label="超时时间(秒)" initialValue={30}>
            <InputNumber min={1} max={300} className="w-full" />
          </Form.Item>

          <Form.Item name="isDefault" label="设为默认" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item name="status" label="状态" initialValue={0}>
            <Select onSearch={() => {}}>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>

          <Form.Item name="remark" label="备注">
            <TextArea rows={2} placeholder="请输入备注信息" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default APIConfigPage;
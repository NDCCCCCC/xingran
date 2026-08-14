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
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
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
  const [configTypeFilter, setConfigTypeFilter] = useState<APIConfigType | undefined>();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<APINotificationConfig | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 加载API通知配置列表
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const result = (await getAPINotificationConfigList({
        page: paginationProps.current,
        pageSize: paginationProps.pageSize,
        configType: configTypeFilter,
      })) as { data: { list: APINotificationConfig[]; total: number } };
      setConfigs(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error("加载API配置失败:", error);
      message.error("加载API配置失败");
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
    } catch (_error) {
      message.error("删除失败");
    }
  };

  // 表格列定义
  const columns: ColumnsType<APINotificationConfig> = [
    {
      title: "配置名称",
      dataIndex: "configName",
      key: "configName",
      render: (text: string, record) => (
        <Space>
          <span>{text}</span>
          {record.isDefault && <Tag color="blue">默认</Tag>}
        </Space>
      ),
    },
    {
      title: "类型",
      dataIndex: "configType",
      key: "configType",
      render: (type: APIConfigType) => {
        const config = {
          [APIConfigTypes.SMS]: { color: "cyan", label: "短信" },
          [APIConfigTypes.WEBHOOK]: { color: "purple", label: "Webhook" },
          [APIConfigTypes.PUSH]: { color: "orange", label: "推送" },
        }[type];
        return <Tag color={config?.color}>{config?.label}</Tag>;
      },
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
      render: (type: AuthType) => {
        const config = {
          [AuthTypes.NONE]: { label: "无" },
          [AuthTypes.BASIC]: { label: "Basic" },
          [AuthTypes.BEARER]: { label: "Bearer" },
          [AuthTypes.APIKEY]: { label: "API Key" },
        }[type];
        return <span>{config?.label}</span>;
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "default"}>{status === 0 ? "正常" : "停用"}</Tag>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 150,
      render: (_, record: APINotificationConfig) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openModal(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: "确认删除?",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadConfigs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize, configTypeFilter]);

  return (
    <div className="p-6">
      <Card
        title={
          <Space>
            <ApiOutlined />
            <span>API通知配置</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              placeholder="筛选类型"
              allowClear
              style={{ width: 120 }}
              onChange={(value) => setConfigTypeFilter(value)}
              onSearch={() => {}}
            >
              <Option value={APIConfigTypes.SMS}>短信</Option>
              <Option value={APIConfigTypes.WEBHOOK}>Webhook</Option>
              <Option value={APIConfigTypes.PUSH}>推送</Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={loadConfigs}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
              新增配置
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1200 }}
          pagination={paginationProps}
          onChange={(pagination) => {
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            loadConfigs();
          }}
        />
      </Card>

      {/* 编辑模态框 */}
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
    </div>
  );
};

export default APIConfigPage;

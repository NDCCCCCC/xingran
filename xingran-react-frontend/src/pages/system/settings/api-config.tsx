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
  Switch,
  InputNumber,
  Tabs,
  Radio,
  Empty,
} from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
// Phase 71 (W2)：裸 Input/Select/TextArea → 契约组件。
// 跳过项：<Input.Password>（成员表达式，不在本波 selector 命中面）→ 保留 antd Input 导入。
// 配置类型 / 状态 / 认证方式均为固定枚举 children → 改写 options 形态；状态类禁搜。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
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
import { isFormValidationError } from "@/utils/errorHandler";
import { confirmDelete } from "@/components/shared/confirmDelete";

// 配置类型枚举（形态分类器产物：children <Option> → options）
const CONFIG_TYPE_OPTIONS = [
  { value: APIConfigTypes.SMS, label: "短信" },
  { value: APIConfigTypes.WEBHOOK, label: "Webhook" },
  { value: APIConfigTypes.PUSH, label: "推送" },
];

// 状态枚举（形态分类器产物）
const STATUS_ENUM_OPTIONS = [
  { value: 0, label: "正常" },
  { value: 1, label: "停用" },
];

// 认证方式枚举（形态分类器产物）
const AUTH_TYPE_OPTIONS = [
  { value: AuthTypes.NONE, label: "无需认证" },
  { value: AuthTypes.BASIC, label: "Basic Auth" },
  { value: AuthTypes.BEARER, label: "Bearer Token" },
  { value: AuthTypes.APIKEY, label: "API Key" },
];

const APIConfigPage: FC = () => {
  const { message } = App.useApp();
  // 列表相关状态
  const [configs, setConfigs] = useState<APINotificationConfig[]>([]);
  const [loading, setLoading] = useState(false);

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<APINotificationConfig | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 加载API通知配置列表（设置页平铺：配置条数为个位数，一次拉全量、不翻页）
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const result = (await getAPINotificationConfigList({ page: 1, pageSize: 100 })) as {
        data: { list: APINotificationConfig[]; total: number };
      };
      setConfigs(result.data.list);
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
            className="xr-op-danger"
            icon={<DeleteOutlined />}
            onClick={() => {
              confirmDelete({
                target: `API 配置 ${record.configName}`,
                onDelete: () => handleDelete(record.id),
              });
            }}
          >
            删除
          </Button>
        </div>
      ),
    },
  ];

  // 设置页平铺：无筛选/分页依赖，挂载时拉一次全量即可
  useEffect(() => {
    void loadConfigs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <>
      {/* 表格卡：双层纸感表格（.xr-table-zebra 绿灰表头 + 斑马纹）
          设置页平铺：统计卡与筛选栏已移除（配置条数为个位数），新增入口收进卡片标题右侧 */}
      <Card
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新增配置
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1200 }}
          pagination={false}
          className="xr-table-zebra"
          size="medium"
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <Space orientation="vertical" size={4}>
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
        cancelButtonProps={{ disabled: submitting }}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingConfig(null);
        }}
        width={800}
      >
        <Form form={editForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item
            name="configName"
            label="配置名称"
            rules={[{ required: true, message: "请输入配置名称" }]}
          >
            <AppInput placeholder="请输入配置名称，如：阿里云短信" />
          </Form.Item>

          <Form.Item
            name="configType"
            label="配置类型"
            rules={[{ required: true, message: "请选择配置类型" }]}
          >
            <AppSelect
              allowClear
              placeholder="请选择配置类型"
              showSearch={false}
              options={CONFIG_TYPE_OPTIONS}
            />
          </Form.Item>

          <Form.Item
            name="apiUrl"
            label="API地址"
            rules={[{ required: true, message: "请输入API地址" }]}
          >
            <AppInput placeholder="请输入API调用地址" />
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
                    <AppTextArea
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
                    <AppTextArea
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
                      <AppSelect allowClear showSearch={false} options={AUTH_TYPE_OPTIONS} />
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
                                <AppInput placeholder="请输入用户名" />
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
                              <AppInput placeholder="请输入Bearer Token" />
                            </Form.Item>
                          );
                        }
                        if (authType === AuthTypes.APIKEY) {
                          return (
                            <>
                              <Form.Item name={["authConfig", "key"]} label="Key名称">
                                <AppInput placeholder="如：X-API-Key" />
                              </Form.Item>
                              <Form.Item name={["authConfig", "value"]} label="Key值">
                                <AppInput placeholder="请输入API Key的值" />
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
            <AppSelect allowClear showSearch={false} options={STATUS_ENUM_OPTIONS} />
          </Form.Item>

          <Form.Item name="remark" label="备注">
            <AppTextArea rows={2} placeholder="请输入备注信息" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default APIConfigPage;

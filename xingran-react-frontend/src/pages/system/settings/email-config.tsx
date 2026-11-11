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
  Empty,
} from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined } from "@ant-design/icons";
// Phase 71 (W2)：裸 Input/Select/TextArea → 契约组件。
// 跳过项：<Input.Password>（成员表达式，不在本波 selector 命中面）→ 保留 antd Input 导入。
// 状态枚举 children → options + 禁搜。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { ColumnsType } from "antd/es/table";
import { formatDateTime } from "@/utils/datetime";
import {
  getEmailConfigList,
  createEmailConfig,
  updateEmailConfig,
  deleteEmailConfig,
  testEmailConfig,
  type EmailConfig,
  type EmailConfigCreateRequest,
  type EmailConfigUpdateRequest,
} from "@/lib/notificationConfigApi";
import { isFormValidationError } from "@/utils/errorHandler";
import { confirmDelete } from "@/components/shared/confirmDelete";

// 状态枚举（形态分类器产物：children <Option> → options）
const STATUS_ENUM_OPTIONS = [
  { value: 0, label: "正常" },
  { value: 1, label: "停用" },
];

const EmailConfigPage: FC = () => {
  const { message } = App.useApp();
  // 列表相关状态
  const [configs, setConfigs] = useState<EmailConfig[]>([]);
  const [loading, setLoading] = useState(false);

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<EmailConfig | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 测试模态框
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [testForm] = Form.useForm();
  const [testing, setTesting] = useState(false);

  // 加载邮箱配置列表（设置页平铺：配置条数为个位数，一次拉全量、不翻页）
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const result = (await getEmailConfigList({ page: 1, pageSize: 100 })) as {
        data: { list: EmailConfig[]; total: number };
      };
      setConfigs(result.data.list);
    } catch (error) {
      console.error("加载邮箱配置失败:", error);
      message.error("加载邮箱配置失败，请刷新重试");
    } finally {
      setLoading(false);
    }
  };

  // 打开编辑模态框
  const openModal = (record?: EmailConfig) => {
    if (record) {
      setEditingConfig(record);
      editForm.setFieldsValue({
        ...record,
      });
    } else {
      setEditingConfig(null);
      editForm.resetFields();
      editForm.setFieldsValue({
        port: 587,
        useSsl: true,
        useStartTls: true,
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
        const updateData: EmailConfigUpdateRequest = { ...values };
        // 如果密码为空或未修改，不发送密码字段（脱敏：password === ******）
        if (!values.password || values.password === "******") {
          delete updateData.password;
        }
        await updateEmailConfig(editingConfig.id, updateData);
        message.success("更新成功");
      } else {
        const createData: EmailConfigCreateRequest = { ...values };
        await createEmailConfig(createData);
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
      await deleteEmailConfig(id);
      message.success("删除成功");
      loadConfigs();
    } catch (_error) {
      message.error("删除失败");
    }
  };

  // 打开测试模态框
  const openTestModal = (config: EmailConfig) => {
    testForm.setFieldsValue({
      configId: config.id,
      testTo: config.fromEmail || config.username,
    });
    setTestModalVisible(true);
  };

  // 测试发送
  const handleTest = async () => {
    try {
      const values = await testForm.validateFields();
      setTesting(true);

      // 调用测试API
      await testEmailConfig(values.configId, values.testTo);
      message.success("测试邮件已发送，请查收");
      setTestModalVisible(false);
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      message.error("测试失败: " + ((error as Error).message || "未知错误"));
    } finally {
      setTesting(false);
    }
  };

  // 表格列定义
  const columns: ColumnsType<EmailConfig> = [
    {
      title: "配置名称",
      dataIndex: "configName",
      key: "configName",
      width: 160,
      render: (text: string, record) => (
        <Space size={4}>
          <span className="xr-cell-id">{text}</span>
          {record.isDefault && <span className="xr-tag xr-tag-gold">默认</span>}
        </Space>
      ),
    },
    {
      title: "SMTP服务器",
      key: "server",
      width: 200,
      render: (_, record) => (
        <span>
          {record.host}:{record.port}
          {record.useSsl && (
            <span className="xr-tag" style={{ marginLeft: 8 }}>
              SSL
            </span>
          )}
        </span>
      ),
    },
    {
      title: "发件人",
      dataIndex: "username",
      key: "username",
      width: 200,
    },
    {
      title: "发件名称",
      dataIndex: "fromName",
      key: "fromName",
      width: 120,
      render: (text) => text || "-",
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
      render: (text: string) => <span className="xr-cell-time">{formatDateTime(text)}</span>,
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 200,
      render: (_, record: EmailConfig) => (
        <div className="xr-row-ops">
          <Button
            type="link"
            size="small"
            icon={<SendOutlined />}
            onClick={() => openTestModal(record)}
          >
            测试
          </Button>
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
                target: `邮件配置 ${record.configName}`,
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
                      暂无邮箱配置
                    </span>
                    <span style={{ color: "var(--theme-text-secondary)", fontSize: 12 }}>
                      新增第一个邮箱服务器配置，用于发送通知邮件
                    </span>
                  </Space>
                }
              />
            ),
          }}
        />
      </Card>

      {/* 编辑模态框 — D-09: 宽度 700 不变 */}
      <Modal
        title={editingConfig ? "编辑邮箱配置" : "新增邮箱配置"}
        open={modalVisible}
        onOk={handleSubmit}
        cancelButtonProps={{ disabled: submitting }}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingConfig(null);
        }}
        width={700}
      >
        <Form form={editForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item
            name="configName"
            label="配置名称"
            rules={[{ required: true, message: "请输入配置名称" }]}
          >
            <AppInput placeholder="请输入配置名称，如：企业邮箱" />
          </Form.Item>

          <Form.Item
            name="host"
            label="SMTP服务器"
            rules={[{ required: true, message: "请输入SMTP服务器地址" }]}
          >
            <AppInput placeholder="如：smtp.example.com" />
          </Form.Item>

          <Form.Item name="port" label="端口" rules={[{ required: true, message: "请输入端口" }]}>
            <InputNumber min={1} max={65535} className="w-full" placeholder="默认587" />
          </Form.Item>

          <Form.Item
            name="username"
            label="邮箱账号"
            rules={[{ required: true, message: "请输入邮箱账号" }]}
          >
            <AppInput placeholder="请输入发件人邮箱账号" />
          </Form.Item>

          <Form.Item
            name="password"
            label="邮箱密码"
            rules={editingConfig ? [] : [{ required: true, message: "请输入邮箱密码或授权码" }]}
          >
            <Input.Password placeholder="请输入邮箱密码或授权码（留空则不修改）" />
          </Form.Item>

          <Form.Item name="fromName" label="发件名称">
            <AppInput placeholder="发件人显示名称" />
          </Form.Item>

          <Form.Item name="fromEmail" label="发件邮箱">
            <AppInput placeholder="发件人显示邮箱（可不同于登录邮箱）" />
          </Form.Item>

          <Form.Item name="useSsl" label="启用SSL" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item
            name="useStartTls"
            label="启用STARTTLS"
            valuePropName="checked"
            tooltip="当不使用SSL时，可选择是否使用STARTTLS升级为加密连接"
          >
            <Switch />
          </Form.Item>

          <Form.Item name="isDefault" label="设为默认" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item name="status" label="状态" initialValue={0}>
            <AppSelect allowClear showSearch={false} options={STATUS_ENUM_OPTIONS} />
          </Form.Item>

          <Form.Item name="remark" label="备注">
            <AppTextArea rows={3} placeholder="请输入备注信息" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 测试模态框 — D-09: 结构不变 */}
      <Modal
        title="测试邮件发送"
        open={testModalVisible}
        onOk={handleTest}
        cancelButtonProps={{ disabled: testing }}
        onCancel={() => setTestModalVisible(false)}
      >
        <Form form={testForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item name="configId" hidden>
            <AppInput />
          </Form.Item>
          <Form.Item
            name="testTo"
            label="收件人"
            rules={[{ required: true, message: "请输入收件人邮箱" }]}
          >
            <AppInput placeholder="请输入测试邮件的收件人邮箱" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default EmailConfigPage;

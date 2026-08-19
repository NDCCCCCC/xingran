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
  Empty,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  MailOutlined,
  SendOutlined,
} from "@ant-design/icons";
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
import { usePagination } from "@/hooks/usePagination";
import { isFormValidationError } from "@/utils/errorHandler";

const { Option } = Select;
const { TextArea } = Input;

const EmailConfigPage: FC = () => {
  const { message } = App.useApp();
  // 列表相关状态
  const [configs, setConfigs] = useState<EmailConfig[]>([]);
  const [loading, setLoading] = useState(false);

  // 统计卡轻请求（status 0/1 各一次只取 total，零后端改动 — email 0=启用/1=停用）
  const [totalCount, setTotalCount] = useState(0);
  const [enabledCount, setEnabledCount] = useState(0);
  const [disabledCount, setDisabledCount] = useState(0);

  // 搜索表单
  const [searchForm] = Form.useForm();
  const [searchName, setSearchName] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<number | undefined>(undefined);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingConfig, setEditingConfig] = useState<EmailConfig | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 测试模态框
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [testForm] = Form.useForm();
  const [testing, setTesting] = useState(false);

  // 统计卡轻请求：status 0/1 各一次只取 total
  const loadStatistics = async () => {
    try {
      const [allRes, enabledRes, disabledRes] = await Promise.all([
        getEmailConfigList({ page: 1, pageSize: 1 }),
        getEmailConfigList({ page: 1, pageSize: 1, status: 0 }),
        getEmailConfigList({ page: 1, pageSize: 1, status: 1 }),
      ]);
      const data = allRes as { data: { total: number } };
      const eData = enabledRes as { data: { total: number } };
      const dData = disabledRes as { data: { total: number } };
      setTotalCount(data.data.total);
      setEnabledCount(eData.data.total);
      setDisabledCount(dData.data.total);
    } catch (error) {
      console.error("加载邮箱配置统计失败:", error);
    }
  };

  // 加载邮箱配置列表
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const result = (await getEmailConfigList({
        page: paginationProps.current,
        pageSize: paginationProps.pageSize,
        status: statusFilter,
      })) as { data: { list: EmailConfig[]; total: number } };
      setConfigs(result.data.list);
      setTotal(result.data.total);
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
      await deleteEmailConfig(id);
      message.success("删除成功");
      loadConfigs();
      loadStatistics();
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

  // 搜索处理
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    setSearchName(values.configName || "");
    setStatusFilter(values.status);
    setCurrent(1);
  };

  // 重置搜索
  const handleResetSearch = () => {
    searchForm.resetFields();
    setSearchName("");
    setStatusFilter(undefined);
    setCurrent(1);
  };

  // 启用占比 = enabled/total * 100（保留一位小数向下取整百分比）
  const enabledPct = totalCount > 0 ? Math.round((enabledCount / totalCount) * 100) : 0;

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
          {record.useSsl && <span className="xr-tag" style={{ marginLeft: 8 }}>SSL</span>}
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
          <Button type="link" size="small" icon={<SendOutlined />} onClick={() => openTestModal(record)}>
            测试
          </Button>
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
                title: "删除邮箱配置？",
                content: "删除后不可恢复，使用该配置的通知邮件将发送失败。",
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
  }, [paginationProps.current, paginationProps.pageSize, statusFilter]);

  return (
    <>
      {/* 统计卡行：3 卡 — 总配置数 / 启用 / 停用 */}
      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-label">总配置数</div>
          <div className="stat-value">{totalCount}</div>
          <div className="stat-trend">全部通知渠道</div>
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

      {/* 工具栏卡：搜索 / 状态筛选 / 新增配置 */}
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
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingConfig(null);
        }}
        confirmLoading={submitting}
        width={700}
      >
        <Form form={editForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item
            name="configName"
            label="配置名称"
            rules={[{ required: true, message: "请输入配置名称" }]}
          >
            <Input placeholder="请输入配置名称，如：企业邮箱" />
          </Form.Item>

          <Form.Item
            name="host"
            label="SMTP服务器"
            rules={[{ required: true, message: "请输入SMTP服务器地址" }]}
          >
            <Input placeholder="如：smtp.example.com" />
          </Form.Item>

          <Form.Item name="port" label="端口" rules={[{ required: true, message: "请输入端口" }]}>
            <InputNumber min={1} max={65535} className="w-full" placeholder="默认587" />
          </Form.Item>

          <Form.Item
            name="username"
            label="邮箱账号"
            rules={[{ required: true, message: "请输入邮箱账号" }]}
          >
            <Input placeholder="请输入发件人邮箱账号" />
          </Form.Item>

          <Form.Item
            name="password"
            label="邮箱密码"
            rules={editingConfig ? [] : [{ required: true, message: "请输入邮箱密码或授权码" }]}
          >
            <Input.Password placeholder="请输入邮箱密码或授权码（留空则不修改）" />
          </Form.Item>

          <Form.Item name="fromName" label="发件名称">
            <Input placeholder="发件人显示名称" />
          </Form.Item>

          <Form.Item name="fromEmail" label="发件邮箱">
            <Input placeholder="发件人显示邮箱（可不同于登录邮箱）" />
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
            <Select onSearch={() => {}}>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>

          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注信息" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 测试模态框 — D-09: 结构不变 */}
      <Modal
        title="测试邮件发送"
        open={testModalVisible}
        onOk={handleTest}
        onCancel={() => setTestModalVisible(false)}
        confirmLoading={testing}
      >
        <Form form={testForm} labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item name="configId" hidden>
            <Input />
          </Form.Item>
          <Form.Item
            name="testTo"
            label="收件人"
            rules={[{ required: true, message: "请输入收件人邮箱" }]}
          >
            <Input placeholder="请输入测试邮件的收件人邮箱" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default EmailConfigPage;
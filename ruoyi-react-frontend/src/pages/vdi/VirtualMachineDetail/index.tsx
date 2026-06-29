import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Tabs,
  Button,
  Space,
  Tag,
  Table,
  Card,
  Descriptions,
  App,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tooltip,
} from "antd";
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  PlusOutlined,
  DeleteOutlined,
  KeyOutlined,
} from "@ant-design/icons";
import { vmApi } from "@/lib/vdiApi";
import type { VirtualMachine, VMAccount } from "@/types/vdi";
import type { ColumnsType } from "antd/es/table";

const VirtualMachineDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [vm, setVM] = useState<VirtualMachine | null>(null);
  const [accounts, setAccounts] = useState<VMAccount[]>([]);
  const [activeTab, setActiveTab] = useState("overview");
  const [createAccountModalVisible, setCreateAccountModalVisible] = useState(false);
  const [resetPasswordModalVisible, setResetPasswordModalVisible] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<VMAccount | null>(null);
  const [form] = Form.useForm();

  // 加载虚拟机详情
  const loadVMDetail = async () => {
    setLoading(true);
    try {
      const result = await vmApi.get(id!);
      if (result.data) {
        setVM(result.data);
      }
    } catch (error) {
      message.error("加载虚拟机详情失败");
    } finally {
      setLoading(false);
    }
  };

  // 加载账号列表
  const loadAccounts = async () => {
    setLoading(true);
    try {
      const result = await vmApi.listAccounts(id!);
      setAccounts(result.data?.list || []);
    } catch (error) {
      message.error("加载账号列表失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (id) {
      loadVMDetail();
      if (activeTab === "accounts") {
        loadAccounts();
      }
    }
  }, [id, activeTab]);

  // 创建账号
  const handleCreateAccount = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.createAccount(id!, {
        username: values.username,
        password: values.password,
        os_type: values.os_type || "Windows",
        is_admin: values.is_admin || false,
      });
      message.success("账号创建成功");
      setCreateAccountModalVisible(false);
      form.resetFields();
      loadAccounts();
    } catch (error) {
      message.error("创建账号失败");
    } finally {
      setLoading(false);
    }
  };

  // 重置密码
  const handleResetPassword = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.resetAccountPassword(id!, selectedAccount!.id, {
        new_password: values.new_password,
      });
      message.success("密码重置成功");
      setResetPasswordModalVisible(false);
      form.resetFields();
      loadAccounts();
    } catch (error) {
      message.error("密码重置失败");
    } finally {
      setLoading(false);
    }
  };

  // 删除账号
  const handleDeleteAccount = async (accountId: string) => {
    setLoading(true);
    try {
      await vmApi.deleteAccount(id!, accountId);
      message.success("账号删除成功");
      loadAccounts();
    } catch (error) {
      message.error("删除账号失败");
    } finally {
      setLoading(false);
    }
  };

  // 账号列表表格列定义
  const accountColumns: ColumnsType<VMAccount> = [
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
    },
    {
      title: "操作系统",
      dataIndex: "os_type",
      key: "os_type",
      render: (os: string) => (
        <Tag color={os === "Windows" ? "blue" : "green"}>{os}</Tag>
      ),
    },
    {
      title: "管理员",
      dataIndex: "is_admin",
      key: "is_admin",
      render: (isAdmin: boolean) => (
        <Tag color={isAdmin ? "red" : "default"}>{isAdmin ? "是" : "否"}</Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "is_enabled",
      key: "is_enabled",
      render: (enabled: boolean) => (
        <Tag color={enabled ? "success" : "default"}>
          {enabled ? "启用" : "禁用"}
        </Tag>
      ),
    },
    {
      title: "同步状态",
      dataIndex: "sync_status",
      key: "sync_status",
      render: (status: string) => {
        const config = {
          synced: { color: "success", text: "已同步" },
          pending: { color: "warning", text: "待同步" },
          failed: { color: "error", text: "同步失败" },
        };
        const { color, text } = config[status as keyof typeof config] || { color: "default", text: status };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: "操作",
      key: "action",
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="重置密码">
            <Button
              type="text"
              icon={<KeyOutlined />}
              onClick={() => {
                setSelectedAccount(record);
                setResetPasswordModalVisible(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个账号吗？"
            onConfirm={() => handleDeleteAccount(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (!vm) {
    return <div>加载中...</div>;
  }

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        {/* 头部 */}
        <Space>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate("/vdi/vm")}
          >
            返回列表
          </Button>
          <Button icon={<ReloadOutlined />} onClick={loadVMDetail}>
            刷新
          </Button>
        </Space>

        {/* 标签页 */}
        <Tabs
          activeKey={activeTab}
          onChange={(key) => setActiveTab(key)}
        >
          {/* 概览标签页 */}
          <Tabs.TabPane tab="概览" key="overview">
            <Descriptions title="虚拟机信息" bordered column={2}>
              <Descriptions.Item label="虚拟机 ID">{vm.vm_id}</Descriptions.Item>
              <Descriptions.Item label="名称">{vm.name}</Descriptions.Item>
              <Descriptions.Item label="电源状态">
                <Tag color={
                  vm.power_state === "in_use" ? "success" :
                  vm.power_state === "stopped" ? "error" : "warning"
                }>
                  {vm.power_state === "in_use" ? "运行中" :
                   vm.power_state === "stopped" ? "已关机" : "已休眠"}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="IP 地址">{vm.ip_address || "-"}</Descriptions.Item>
              <Descriptions.Item label="操作系统">{vm.os_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="资源规格">
                {vm.cpu_number || 0}核 / {vm.memory || 0}MB / {vm.disk || 0}GB
              </Descriptions.Item>
              <Descriptions.Item label="绑定用户">{vm.bound_user_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="最后同步">
                {vm.last_sync_at ? new Date(vm.last_sync_at).toLocaleString("zh-CN") : "-"}
              </Descriptions.Item>
            </Descriptions>
          </Tabs.TabPane>

          {/* 账号管理标签页 */}
          <Tabs.TabPane tab="账号管理" key="accounts">
            <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
              <Space>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setCreateAccountModalVisible(true)}
                >
                  创建账号
                </Button>
                <Button icon={<ReloadOutlined />} onClick={loadAccounts}>
                  刷新
                </Button>
              </Space>

              <Table
                columns={accountColumns}
                dataSource={accounts}
                rowKey="id"
                loading={loading}
                pagination={false}
              />
            </Space>
          </Tabs.TabPane>

          {/* 操作记录标签页 */}
          <Tabs.TabPane tab="操作记录" key="operations">
            <div style={{ padding: "20px", textAlign: "center", color: "var(--theme-text-tertiary, #999)" }}>
              操作记录功能（未来实现）
            </div>
          </Tabs.TabPane>

          {/* 监控标签页 */}
          <Tabs.TabPane tab="监控" key="monitor">
            <div style={{ padding: "20px", textAlign: "center", color: "var(--theme-text-tertiary, #999)" }}>
              监控数据功能（未来实现）
            </div>
          </Tabs.TabPane>
        </Tabs>
      </Space>

      {/* 创建账号模态框 */}
      <Modal
        title="创建 VM 账号"
        open={createAccountModalVisible}
        onOk={handleCreateAccount}
        onCancel={() => setCreateAccountModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="用户名"
            name="username"
            rules={[{ required: true, message: "请输入用户名" }]}
          >
            <Input placeholder="用户名" />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[
              { required: true, message: "请输入密码" },
              { min: 8, message: "密码至少 8 位" },
            ]}
          >
            <Input.Password placeholder="密码" />
          </Form.Item>
          <Form.Item
            label="操作系统类型"
            name="os_type"
            initialValue="Windows"
          >
            <Select onSearch={() => {}}>
              <Select.Option value="Windows">Windows</Select.Option>
              <Select.Option value="Linux">Linux</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            label="管理员权限"
            name="is_admin"
            valuePropName="checked"
            initialValue={false}
          >
            <Select onSearch={() => {}}>
              <Select.Option value={false}>普通用户</Select.Option>
              <Select.Option value={true}>管理员</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码模态框 */}
      <Modal
        title="重置密码"
        open={resetPasswordModalVisible}
        onOk={handleResetPassword}
        onCancel={() => setResetPasswordModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 8, message: "密码至少 8 位" },
            ]}
          >
            <Input.Password placeholder="新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default VirtualMachineDetail;

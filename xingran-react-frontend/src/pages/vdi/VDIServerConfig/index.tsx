import React, { useState, useEffect } from "react";
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  App,
  Popconfirm,
  Input,
  Card,
  Form,
  InputNumber,
  Switch,
} from "antd";
import {
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ApiOutlined,
} from "@ant-design/icons";
import { vdiServerApi } from "@/lib/vdiApi";
import type { VDIServer, VDIServerConfig } from "@/types/vdi";
import type { ColumnsType } from "antd/es/table";

const VDIServerConfig: React.FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [servers, setServers] = useState<VDIServer[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalMode, setModalMode] = useState<"create" | "edit">("create");
  const [selectedServer, setSelectedServer] = useState<VDIServer | null>(null);
  const [form] = Form.useForm();

  // 加载服务器列表
  const loadServers = async () => {
    setLoading(true);
    try {
      const result = await vdiServerApi.list({ current, pageSize });
      setServers(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (_error) {
      message.error("加载 VDI 服务器列表失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadServers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current, pageSize]);

  // 创建服务器
  const handleCreate = () => {
    setModalMode("create");
    setSelectedServer(null);
    form.resetFields();
    form.setFieldsValue({
      tenant_id: 1,
      status: true, // Switch期望布尔值：true=启用(正常), false=停用
    });
    setModalVisible(true);
  };

  // 编辑服务器
  const handleEdit = (server: VDIServer) => {
    setModalMode("edit");
    setSelectedServer(server);
    form.setFieldsValue({
      name: server.name,
      endpoint: server.endpoint,
      username: server.username,
      tenant_id: server.tenant_id,
      status: server.status === 0, // 转换为布尔值：0=true(启用), 1=false(停用)
    });
    setModalVisible(true);
  };

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (modalMode === "create") {
        const data: VDIServerConfig = {
          name: values.name,
          endpoint: values.endpoint,
          username: values.username,
          password: values.password,
          tenant_id: values.tenant_id,
          status: values.status ? 0 : 1,
        };
        await vdiServerApi.create(data);
        message.success("VDI 服务器创建成功");
      } else {
        const data: Partial<VDIServerConfig> = {
          name: values.name,
          endpoint: values.endpoint,
          username: values.username,
          tenant_id: values.tenant_id,
          status: values.status ? 0 : 1,
        };
        if (values.password) {
          data.password = values.password;
        }
        await vdiServerApi.update(selectedServer!.id, data);
        message.success("VDI 服务器更新成功");
      }

      setModalVisible(false);
      form.resetFields();
      loadServers();
    } catch (_error) {
      message.error(modalMode === "create" ? "创建服务器失败" : "更新服务器失败");
    } finally {
      setLoading(false);
    }
  };

  // 删除服务器
  const handleDelete = async (id: string) => {
    setLoading(true);
    try {
      await vdiServerApi.delete(id);
      message.success("删除成功");
      loadServers();
    } catch (_error) {
      message.error("删除失败");
    } finally {
      setLoading(false);
    }
  };

  // 测试连接
  const handleTestConnection = async (id: string) => {
    setLoading(true);
    try {
      await vdiServerApi.testConnection(id);
      message.success("连接测试成功");
    } catch (_error) {
      message.error("连接测试失败");
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns: ColumnsType<VDIServer> = [
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      width: 150,
    },
    {
      title: "服务端点",
      dataIndex: "endpoint",
      key: "endpoint",
      width: 250,
      ellipsis: true,
    },
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
      width: 120,
    },
    {
      title: "租户 ID",
      dataIndex: "tenant_id",
      key: "tenant_id",
      width: 100,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "default"}>
          {status === 0 ? "正常" : "停用"}
        </Tag>
      ),
    },
    {
      title: "Token 过期时间",
      dataIndex: "token_expiry",
      key: "token_expiry",
      width: 160,
      render: (time: string) => time ? new Date(time).toLocaleString("zh-CN") : "-",
    },
    {
      title: "最后同步",
      dataIndex: "lastSyncTime",
      key: "lastSyncTime",
      width: 160,
      render: (time: string) => time ? new Date(time).toLocaleString("zh-CN") : "-",
    },
    {
      title: "操作",
      key: "action",
      width: 250,
      fixed: "right",
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            icon={<ApiOutlined />}
            onClick={() => handleTestConnection(record.id)}
          >
            测试连接
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个 VDI 服务器吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        {/* 工具栏 */}
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            添加服务器
          </Button>
          <Button icon={<ReloadOutlined />} onClick={loadServers}>
            刷新
          </Button>
        </Space>

        {/* 表格 */}
        <Table
          columns={columns}
          dataSource={servers}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
            },
          }}
        />
      </Space>

      {/* 创建/编辑模态框 */}
      <Modal
        title={modalMode === "create" ? "添加 VDI 服务器" : "编辑 VDI 服务器"}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="服务器名称"
            name="name"
            rules={[{ required: true, message: "请输入服务器名称" }]}
          >
            <Input placeholder="例如：生产环境 VDI" />
          </Form.Item>
          <Form.Item
            label="服务端点"
            name="endpoint"
            rules={[
              { required: true, message: "请输入服务端点" },
              { type: "url", message: "请输入有效的 URL" },
            ]}
          >
            <Input placeholder="https://vdi.example.com" />
          </Form.Item>
          <Form.Item
            label="用户名"
            name="username"
            rules={[{ required: true, message: "请输入用户名" }]}
          >
            <Input placeholder="VDI 管理员用户名" />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={modalMode === "create" ? [{ required: true, message: "请输入密码" }] : []}
          >
            <Input.Password placeholder={modalMode === "create" ? "VDI 管理员密码" : "留空表示不修改密码"} />
          </Form.Item>
          <Form.Item
            label="租户 ID"
            name="tenant_id"
            rules={[{ required: true, message: "请输入租户 ID" }]}
          >
            <InputNumber min={1} style={{ width: "100%" }} placeholder="默认为 1" />
          </Form.Item>
          <Form.Item
            label="状态"
            name="status"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="正常" unCheckedChildren="停用" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default VDIServerConfig;

/**
 * API 密钥管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  App,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Tag,
  Card,
  Row,
  Col,
  Tooltip,
  Popconfirm,
  Alert,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CopyOutlined,
  EyeOutlined,
  KeyOutlined,
  LockOutlined,
  UnlockOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import {
  listAPIKeys,
  createAPIKey,
  updateAPIKey,
  deleteAPIKey,
  toggleAPIKeyStatus,
} from "@/api/apikey";
import type {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  APIKeyListParams,
} from "@/types/apikey";
import type { PageData } from "@/types/apikey";
import { formatDateTime } from "@/utils/datetime";
import LogsModal from "./LogsModal";

const { Option } = Select;
const { TextArea } = Input;

// ==================== 常量定义 ====================

const SCOPE_OPTIONS = [
  { label: "读取", value: "read" },
  { label: "写入", value: "write" },
  { label: "管理", value: "admin" },
];

const SCOPE_COLORS: Record<string, string> = {
  read: "blue",
  write: "orange",
  admin: "red",
};

// ==================== 工具函数 ====================

/**
 * 密钥脱敏显示 - 仅显示前12位
 */
function maskKey(key: string): string {
  if (!key || key.length <= 12) {
    return key;
  }
  return `${key.slice(0, 12)}...`;
}

/**
 * 复制文本到剪贴板
 */
async function copyToClipboard(
  text: string,
  message: { success: (msg: string) => void; error: (msg: string) => void }
): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
    return true;
  } catch (error) {
    message.error("复制失败，请手动复制");
    return false;
  }
}

/**
 * 格式化作用域标签
 */
function renderScopeTags(scopes: string[]) {
  return (
    <Space size="small" wrap>
      {scopes.map((scope) => (
        <Tag key={scope} color={SCOPE_COLORS[scope] || "default"}>
          {scope}
        </Tag>
      ))}
    </Space>
  );
}

// ==================== 主组件 ====================

const APIKeyManagement: FC = () => {
  const { message } = App.useApp();
  // ==================== 状态管理 ====================
  const [dataSource, setDataSource] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  // Modal 状态
  const [modalVisible, setModalVisible] = useState<boolean>(false);
  const [modalType, setModalType] = useState<"create" | "edit">("create");
  const [editingRecord, setEditingRecord] = useState<APIKey | null>(null);
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  // 日志 Modal 状态
  const [logsModalVisible, setLogsModalVisible] = useState<boolean>(false);
  const [selectedKeyId, setSelectedKeyId] = useState<string>("");

  // 搜索筛选状态
  const [searchKeyword, setSearchKeyword] = useState<string>("");
  const location = useLocation();
  const [filterStatus, setFilterStatus] = usePersistedStateController<boolean | undefined>({
    keyPrefix: location.pathname,
    keySuffix: "filterStatus",
    defaultValue: undefined,
  });
  const [filterScope, setFilterScope] = usePersistedStateController<string | undefined>({
    keyPrefix: location.pathname,
    keySuffix: "filterScope",
    defaultValue: undefined,
  });

  // 服务端排序状态（field 对应后端 apiKeyAllowedSortFields 白名单 key）
  const [sortField, setSortField] = useState<string>("");
  const [sortOrder, setSortOrder] = useState<"ascend" | "descend" | null>(null);

  // Form 实例
  const [form] = Form.useForm();

  // ==================== 数据加载 ====================

  /**
   * 稳定的查询参数 - 防止 useEffect 无限循环
   */
  const queryParams = useMemo(() => {
    return {
      current: pagination.current,
      pageSize: pagination.pageSize,
      ...(searchKeyword && { keyword: searchKeyword }),
      ...(filterStatus !== undefined && { status: filterStatus }),
      ...(filterScope && { scope: filterScope }),
      // 服务端排序透传（避坑：详见 memory server-sort-loadfunc-param-drop）
      ...(sortField && { orderByColumn: sortField, isAsc: sortOrder === "ascend" }),
    };
  }, [
    pagination.current,
    pagination.pageSize,
    searchKeyword,
    filterStatus,
    filterScope,
    sortField,
    sortOrder,
  ]);

  /**
   * 加载 API 密钥列表
   */
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listAPIKeys(queryParams);
      setDataSource(result.data?.list || []);
      setPagination((prev) => ({
        ...prev,
        total: result.data?.total || 0,
      }));
    } catch (error) {
      console.error("加载 API 密钥列表失败:", error);
      message.error("加载数据失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, [queryParams]);

  /**
   * 初始化加载数据
   */
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // ==================== 事件处理 ====================

  /**
   * 打开创建密钥 Modal
   */
  const handleAdd = useCallback(() => {
    setModalType("create");
    setEditingRecord(null);
    setCreatedKey(null);
    form.resetFields();
    form.setFieldsValue({
      inheritPerms: true,
      scopes: ["read"],
    });
    setModalVisible(true);
  }, [form]);

  /**
   * 打开编辑密钥 Modal
   */
  const handleEdit = useCallback(
    (record: APIKey) => {
      setModalType("edit");
      setEditingRecord(record);
      setCreatedKey(null);
      form.setFieldsValue({
        name: record.name,
        description: record.description,
        scopes: record.scopes,
        inheritPerms: record.inheritPerms,
        ipWhitelist: record.ipWhitelist?.join(", "),
      });
      setModalVisible(true);
    },
    [form]
  );

  /**
   * 查看密钥详情
   */
  const handleView = useCallback((record: APIKey) => {
    Modal.info({
      title: "密钥详情",
      width: 600,
      content: (
        <div style={{ marginTop: 16 }}>
          <p>
            <strong>名称：</strong>
            {record.name}
          </p>
          <p>
            <strong>密钥：</strong>
            {maskKey(record.key)}
          </p>
          <p>
            <strong>作用域：</strong>
            {renderScopeTags(record.scopes)}
          </p>
          <p>
            <strong>继承权限：</strong>
            {record.inheritPerms ? "是" : "否"}
          </p>
          {record.ipWhitelist && record.ipWhitelist.length > 0 && (
            <p>
              <strong>IP 白名单：</strong>
              {record.ipWhitelist.join(", ")}
            </p>
          )}
          {record.description && (
            <p>
              <strong>描述：</strong>
              {record.description}
            </p>
          )}
          <p>
            <strong>过期时间：</strong>
            {record.expiresAt ? formatDateTime(record.expiresAt) : "永不过期"}
          </p>
          <p>
            <strong>最后使用：</strong>
            {record.lastUsedAt ? formatDateTime(record.lastUsedAt) : "未使用"}
          </p>
          <p>
            <strong>创建时间：</strong>
            {formatDateTime(record.createdAt)}
          </p>
        </div>
      ),
    });
  }, []);

  /**
   * 删除密钥
   */
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteAPIKey(id);
        message.success("删除成功");
        fetchData();
      } catch (error) {
        console.error("删除 API 密钥失败:", error);
        message.error("删除失败，请稍后重试");
      }
    },
    [fetchData]
  );

  /**
   * 切换密钥状态
   */
  const handleToggleStatus = useCallback(
    async (record: APIKey) => {
      try {
        await toggleAPIKeyStatus(record.id);
        message.success(`${record.isActive ? "禁用" : "启用"}成功`);
        fetchData();
      } catch (error) {
        console.error("切换状态失败:", error);
        message.error("操作失败，请稍后重试");
      }
    },
    [fetchData]
  );

  /**
   * 提交表单（创建或更新）
   */
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();

      // 处理 IP 白名单
      let ipWhitelist: string[] | undefined;
      if (values.ipWhitelist) {
        ipWhitelist = values.ipWhitelist
          .split(",")
          .map((s: string) => s.trim())
          .filter((s: string) => s.length > 0);
      }

      if (modalType === "create") {
        // 创建密钥
        const createData: CreateAPIKeyRequest = {
          name: values.name,
          description: values.description,
          scopes: values.scopes,
          inheritPerms: values.inheritPerms,
          ipWhitelist: ipWhitelist,
          expiresAt: values.expiresAt ? values.expiresAt.toISOString() : undefined,
        };

        const result = await createAPIKey(createData);
        setCreatedKey(result.data?.key || null);
        message.success("创建成功");
      } else {
        // 更新密钥
        const updateData: UpdateAPIKeyRequest = {
          name: values.name,
          description: values.description,
          scopes: values.scopes,
          inheritPerms: values.inheritPerms,
          ipWhitelist: ipWhitelist,
        };

        await updateAPIKey(editingRecord!.id, updateData);
        message.success("更新成功");
      }

      setModalVisible(false);
      fetchData();
    } catch (error) {
      console.error("提交失败:", error);
      if (error instanceof Error) {
        message.error(`操作失败: ${error.message}`);
      } else {
        message.error("操作失败，请稍后重试");
      }
    }
  }, [form, modalType, editingRecord, fetchData]);

  /**
   * 搜索处理
   */
  const handleSearch = useCallback(() => {
    setPagination((prev) => ({ ...prev, current: 1 }));
  }, []);

  /**
   * 重置搜索
   */
  const handleReset = useCallback(() => {
    setSearchKeyword("");
    setFilterStatus(undefined);
    setFilterScope(undefined);
    setPagination((prev) => ({ ...prev, current: 1 }));
  }, []);

  /**
   * 刷新数据
   */
  const handleRefresh = useCallback(() => {
    fetchData();
  }, [fetchData]);

  /**
   * 打开日志 Modal
   */
  const handleViewLogs = useCallback((record: APIKey) => {
    setSelectedKeyId(record.id);
    setLogsModalVisible(true);
  }, []);

  /**
   * 分页 + 排序变化（antd Table onChange 统一入口）
   */
  const handleTableChange = useCallback((newPagination: any, _filters: any, sorter: any) => {
    setPagination({
      current: newPagination.current || 1,
      pageSize: newPagination.pageSize || 10,
      total: newPagination.total || 0,
    });
    // 排序：用 local const 持有新值，规避 React 18 setState 异步时序（handleTableChange 触发的
    // fetchData 读 state 仍为旧值——详见 commit 7ab1189 ad-domain 同类坑）。
    // queryParams 已监听 sortField/sortOrder，setState 后 useEffect 会自动重拉。
    const field = sorter?.field || "";
    const order = sorter?.order ?? null;
    setSortField(field);
    setSortOrder(order);
  }, []);

  // ==================== 表格列定义 ====================

  const columns: ColumnsType<APIKey> = useMemo(
    () => [
      {
        title: "名称",
        dataIndex: "name",
        key: "name",
        width: 150,
        sorter: true,
        sortOrder: sortField === "name" ? sortOrder : undefined,
        render: (text, record) => (
          <Space>
            <KeyOutlined />
            {text}
          </Space>
        ),
      },
      {
        title: "密钥",
        dataIndex: "key",
        key: "key",
        width: 200,
        render: (text) => (
          <Space>
            <code style={{ fontSize: 12 }}>{maskKey(text)}</code>
            <Tooltip title="复制密钥">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(text, message)}
              />
            </Tooltip>
          </Space>
        ),
      },
      {
        title: "作用域",
        dataIndex: "scopes",
        key: "scopes",
        width: 150,
        render: (scopes: string[]) => renderScopeTags(scopes),
      },
      {
        title: "继承权限",
        dataIndex: "inheritPerms",
        key: "inheritPerms",
        width: 100,
        align: "center" as const,
        render: (inherit: boolean) => (
          <Tag color={inherit ? "green" : "default"}>{inherit ? "是" : "否"}</Tag>
        ),
      },
      {
        title: "IP 白名单",
        dataIndex: "ipWhitelist",
        key: "ipWhitelist",
        width: 150,
        ellipsis: true,
        render: (whitelist: string[]) => {
          if (!whitelist || whitelist.length === 0) {
            return <span style={{ color: "var(--theme-text-tertiary, #999)" }}>-</span>;
          }
          return (
            <Tooltip title={whitelist.join(", ")}>
              <span>
                {whitelist[0]}
                {whitelist.length > 1 ? "..." : ""}
              </span>
            </Tooltip>
          );
        },
      },
      {
        title: "状态",
        dataIndex: "isActive",
        key: "isActive",
        width: 80,
        align: "center" as const,
        sorter: true,
        sortOrder: sortField === "isActive" ? sortOrder : undefined,
        render: (active: boolean) => (
          <Tag color={active ? "green" : "red"}>{active ? "启用" : "禁用"}</Tag>
        ),
      },
      {
        title: "过期时间",
        dataIndex: "expiresAt",
        key: "expiresAt",
        width: 160,
        sorter: true,
        sortOrder: sortField === "expiresAt" ? sortOrder : undefined,
        render: (text) => {
          if (!text) {
            return <span style={{ color: "var(--theme-text-tertiary, #999)" }}>永不过期</span>;
          }
          const date = new Date(text);
          const now = new Date();
          const isExpired = date < now;
          return (
            <span style={{ color: isExpired ? "#ff4d4f" : undefined }}>{formatDateTime(text)}</span>
          );
        },
      },
      {
        title: "最后使用",
        dataIndex: "lastUsedAt",
        key: "lastUsedAt",
        width: 160,
        sorter: true,
        sortOrder: sortField === "lastUsedAt" ? sortOrder : undefined,
        render: (text) => {
          if (!text) {
            return <span style={{ color: "var(--theme-text-tertiary, #999)" }}>未使用</span>;
          }
          return formatDateTime(text);
        },
      },
      {
        title: "操作",
        key: "action",
        width: 220,
        fixed: "right" as const,
        render: (_, record) => (
          <Space size="small">
            <Tooltip title="查看详情">
              <Button
                type="text"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => handleView(record)}
              />
            </Tooltip>
            <Tooltip title="编辑">
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => handleEdit(record)}
              />
            </Tooltip>
            <Tooltip title={record.isActive ? "禁用" : "启用"}>
              <Button
                type="text"
                size="small"
                icon={record.isActive ? <LockOutlined /> : <UnlockOutlined />}
                onClick={() => handleToggleStatus(record)}
              />
            </Tooltip>
            <Tooltip title="使用日志">
              <Button
                type="text"
                size="small"
                icon={<FileTextOutlined />}
                onClick={() => handleViewLogs(record)}
              />
            </Tooltip>
            <Popconfirm
              title="确定要删除这个密钥吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="确定"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Tooltip title="删除">
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Tooltip>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [handleView, handleEdit, handleToggleStatus, handleViewLogs, handleDelete, sortField, sortOrder]
  );

  // ==================== 渲染 ====================

  return (
    <div>
      {/* 页面标题和操作栏 */}
      <Card style={{ marginBottom: 16 }}>
        <Row gutter={16} justify="space-between" align="middle">
          <Col>
            <Space>
              <Input
                placeholder="搜索名称或密钥"
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                onPressEnter={handleSearch}
                style={{ width: 200 }}
                allowClear
              />
              <Select
                placeholder="状态筛选"
                value={filterStatus}
                onChange={setFilterStatus}
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
              >
                <Option value={true}>启用</Option>
                <Option value={false}>禁用</Option>
              </Select>
              <Select
                placeholder="作用域筛选"
                value={filterScope}
                onChange={setFilterScope}
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
              >
                {SCOPE_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                刷新
              </Button>
            </Space>
          </Col>
          <Col>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              创建密钥
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 密钥列表表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={dataSource}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          onChange={handleTableChange}
          scroll={{ x: 1400 }}
        />
      </Card>

      {/* 创建/编辑 Modal */}
      <Modal
        title={modalType === "create" ? "创建 API 密钥" : "编辑 API 密钥"}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnHidden
      >
        {createdKey ? (
          <Alert
            title="创建成功"
            description={
              <div>
                <p>请妥善保管您的 API 密钥，完整密钥仅显示一次：</p>
                <div style={{ marginTop: 8, padding: 12, background: "#f5f5f5", borderRadius: 4 }}>
                  <code style={{ fontSize: 14, wordBreak: "break-all" }}>{createdKey}</code>
                </div>
                <Button
                  type="primary"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => copyToClipboard(createdKey, message)}
                  style={{ marginTop: 12 }}
                >
                  复制密钥
                </Button>
              </div>
            }
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
          />
        ) : null}

        <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item
            name="name"
            label="密钥名称"
            rules={[
              { required: true, message: "请输入密钥名称" },
              { max: 100, message: "名称长度不能超过100个字符" },
            ]}
          >
            <Input placeholder="请输入密钥名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: "描述长度不能超过500个字符" }]}
          >
            <TextArea rows={3} placeholder="请输入描述（可选）" />
          </Form.Item>

          <Form.Item
            name="scopes"
            label="作用域"
            rules={[{ required: true, message: "请选择作用域" }]}
          >
            <Select
              mode="multiple"
              placeholder="请选择作用域"
              options={SCOPE_OPTIONS}
              onSearch={() => {}}
            />
          </Form.Item>

          <Form.Item name="inheritPerms" label="继承权限" valuePropName="checked">
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>

          <Form.Item
            name="ipWhitelist"
            label="IP 白名单"
            extra="支持单个 IP 或 CIDR 格式，多个用逗号分隔，留空表示不限制"
          >
            {/* eslint-disable-next-line no-restricted-syntax -- placeholder hint, not an actual server URL */}
            <Input placeholder="例如: 192.168.1.100, 10.0.0.0/24" />
          </Form.Item>

          {modalType === "create" && (
            <Form.Item name="expiresAt" label="过期时间">
              <Input type="datetime-local" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 使用日志 Modal */}
      <LogsModal
        visible={logsModalVisible}
        apiKeyId={selectedKeyId}
        onClose={() => setLogsModalVisible(false)}
      />
    </div>
  );
};

export default APIKeyManagement;

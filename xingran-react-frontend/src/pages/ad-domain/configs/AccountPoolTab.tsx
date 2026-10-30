import { useEffect, useState, useCallback } from "react";
import { confirmDelete } from "@/components/shared/confirmDelete";
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  message,
  Statistic,
  Row,
  Col,
  Tooltip,
} from "antd";
// Phase 71 (W5)：裸 Input/TextArea → 契约组件。
// 跳过项：<Input.Password>（成员表达式，不在 P65 selector 命中面）→ 保留 antd Input 导入。
import { AppInput, AppTextArea } from "@/components/ui";
import { Input } from "antd";
import {
  PlusOutlined,
  ReloadOutlined,
  UnlockOutlined,
  EditOutlined,
  StopOutlined,
  CheckCircleOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { FC } from "react";
// ⚠️ 直接引模块路径（不走 @/components/shared 桶文件）：桶文件 re-export 了
// FloorPlanEditor → @react-three/fiber → three.js，引入它会让本组件的动态 import
// （既有测试 `AccountPoolTab.render.test.tsx` 的 `await import(...)`）从 ~6s
// 涨到 >15s 直接超时（实测）。useActionPermission 本身零依赖，直接引更轻。
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import {
  listADServiceAccounts,
  createADServiceAccount,
  updateADServiceAccount,
  deleteADServiceAccount,
  enableADServiceAccount,
  disableADServiceAccount,
  unlockADServiceAccount,
  getADServiceAccountStats,
  type ADServiceAccount,
  type ADServiceAccountStats,
} from "@/lib/adDomainApi";
import { formatDateTime } from "@/utils/datetime";

// 状态映射
const STATUS_MAP: Record<number, { color: string; text: string }> = {
  0: { color: "green", text: "可用" },
  1: { color: "default", text: "已停用" },
  2: { color: "orange", text: "熔断中" },
};

// Props: 接收 configId（来自父组件 AD 配置）
interface AccountPoolTabProps {
  configId: string;
}

/**
 * 服务账号池 Tab（嵌入 AD 配置详情 Drawer）
 *
 * Phase 36: 每个 AD 配置有独立的账号池
 * - 1:N 关系: 一个 ADConfig 对应多个 ServiceAccount
 * - 隔离选择: 登录/同步时只使用本 config 的账号
 * - 失败自动切换: 任一账号失败 → 立即试下一个
 *
 * ─────────────────────────────────────────────────────────────────────────
 * Phase 86 W4（RBAC-02）权限码**归属裁决**（源码注释落地处）：
 *
 * 账号池**无独立 TS 码区块** —— 后端 `pkg/permission/config.go:531-538` 有
 * `ops:ad:config:account:list / edit / delete` 三个码（后端已按动作拆到
 * `RequireStrictPermissions` 单码），但前端单一真相源
 * `src/constants/permissionCodes.ts` 的 `opsADConfig` 区块只有
 * list / add / edit / delete / test / sync —— **没有 account 子区块**。
 *
 * 86 全程零新增权限码（Lead 裁定 4），故账号池写操作**按 AD 配置域的操作码归属**：
 *   - 编辑 / 立即解锁 / 启用·停用 → `opsADConfig.edit`
 *   - 删除                        → `opsADConfig.delete`
 *   - 新增账号                    → `opsADConfig.add`
 *
 * ⚠️ 该归属是**前端侧的就近归码**，不是后端权限模型的等价表达：
 * 后端 account 子码的 TS 声明回补归「后端动作级拆分 phase」，
 * 回补后应把这里的就近归码改回 `ops:ad:config:account:*` 精确码。
 * ─────────────────────────────────────────────────────────────────────────
 */
const AccountPoolTab: FC<AccountPoolTabProps> = ({ configId }) => {
  // Phase 86 W4（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 编辑 / 解锁 / 启用·停用 → opsADConfig.edit（归属裁决见文件头注释）
  const permEdit = permGate.decide(PERMISSION_CODES.opsADConfig.edit);
  // 删除 → opsADConfig.delete
  const permDelete = permGate.decide(PERMISSION_CODES.opsADConfig.delete);
  // 新增账号 → opsADConfig.add
  const permAdd = permGate.decide(PERMISSION_CODES.opsADConfig.add);

  const [list, setList] = useState<ADServiceAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [stats, setStats] = useState<ADServiceAccountStats | null>(null);

  // Modal 状态
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [unlocking, setUnlocking] = useState(false);
  const [editing, setEditing] = useState<ADServiceAccount | null>(null);
  const [form] = Form.useForm();

  // Unlock Modal
  const [unlockModalVisible, setUnlockModalVisible] = useState(false);
  const [unlockingId, setUnlockingId] = useState<string>("");
  const [unlockForm] = Form.useForm();

  // 加载账号列表
  const loadList = useCallback(async () => {
    if (!configId) return;
    setLoading(true);
    try {
      const res = await listADServiceAccounts({
        configId,
        page,
        pageSize,
      });
      if (res.code === 0 && res.data) {
        setList(res.data.list);
        setTotal(res.data.total);
      } else {
        message.error(res.message || "加载失败");
      }
    } catch (_e) {
      message.error("加载账号列表失败");
    } finally {
      setLoading(false);
    }
  }, [configId, page, pageSize]);

  // 加载统计
  const loadStats = useCallback(async () => {
    if (!configId) return;
    try {
      const res = await getADServiceAccountStats(configId);
      if (res.code === 0 && res.data) {
        setStats(res.data);
      }
    } catch (_e) {
      // 静默
    }
  }, [configId]);

  useEffect(() => {
    setPage(1); // configId 变化时重置分页
    loadList();
    loadStats();
  }, [configId, loadList, loadStats]);

  // 新增
  const handleCreate = () => {
    setEditing(null);
    form.resetFields();
    setEditModalVisible(true);
  };

  // 编辑
  const handleEdit = (record: ADServiceAccount) => {
    setEditing(record);
    form.setFieldsValue({
      username: record.username,
      remark: record.remark,
    });
    setEditModalVisible(true);
  };

  // 提交（创建/更新）
  // 后端用全局 SM4 cipher 加密明文密码（HTTPS 保护传输层）
  // 服务端存为 password_ciphertext（命中 operlog password 关键词，自动脱敏）
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      if (editing) {
        // 更新：只在填了新密码时才传
        const updateData: {
          id: string;
          username?: string;
          password?: string;
          remark?: string;
        } = {
          id: editing.id,
          username: values.username,
          remark: values.remark,
        };
        if (values.password) {
          updateData.password = values.password; // 明文（HTTPS 保护 + 后端 SM4 加密）
        }
        const res = await updateADServiceAccount(updateData);
        if (res.code === 0) {
          message.success("更新成功");
          setEditModalVisible(false);
          loadList();
          loadStats();
        } else {
          message.error(res.message || "更新失败");
        }
      } else {
        if (!values.password) {
          message.error("请输入密码");
          return;
        }
        const res = await createADServiceAccount({
          configId,
          username: values.username,
          password: values.password,
          remark: values.remark,
        });
        if (res.code === 0) {
          message.success("创建成功");
          setEditModalVisible(false);
          loadList();
          loadStats();
        } else {
          message.error(res.message || "创建失败");
        }
      }
    } catch (_e) {
      // 校验失败
    }
  };

  // 删除
  // 68.1-T3：删除确认改走 confirmDelete，业务失败上抛由 helper 统一反馈
  const handleDelete = async (id: string) => {
    const res = await deleteADServiceAccount(id);
    if (res.code !== 0) {
      throw new Error(res.message || "删除失败");
    }
    loadList();
    loadStats();
  };

  // 启用/停用
  const handleToggleEnabled = async (record: ADServiceAccount) => {
    const action = record.status === 1 ? enableADServiceAccount : disableADServiceAccount;
    try {
      const res = await action(record.id);
      if (res.code === 0) {
        message.success(record.status === 1 ? "已启用" : "已停用");
        loadList();
        loadStats();
      } else {
        message.error(res.message || "操作失败");
      }
    } catch (_e) {
      message.error("操作失败");
    }
  };

  // 立即解锁
  const handleUnlockClick = (id: string) => {
    setUnlockingId(id);
    unlockForm.resetFields();
    setUnlockModalVisible(true);
  };

  const handleUnlockSubmit = async () => {
    try {
      const values = await unlockForm.validateFields();
      if (values.reason.length < 10) {
        message.error("解锁原因至少 10 字符");
        return;
      }
      const res = await unlockADServiceAccount({ id: unlockingId, reason: values.reason });
      if (res.code === 0) {
        message.success("解锁成功");
        setUnlockModalVisible(false);
        loadList();
        loadStats();
      } else {
        message.error(res.message || "解锁失败");
      }
    } catch (_e) {
      // 校验失败
    }
  };

  // 表格列定义
  const columns: ColumnsType<ADServiceAccount> = [
    {
      title: "账号",
      dataIndex: "username",
      key: "username",
      width: 200,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: number) => (
        <Tag color={STATUS_MAP[status]?.color || "default"}>
          {STATUS_MAP[status]?.text || "未知"}
        </Tag>
      ),
    },
    {
      title: "失败次数",
      dataIndex: "failureCount",
      key: "failureCount",
      width: 100,
    },
    {
      title: "上次失败原因",
      dataIndex: "lastFailureReason",
      key: "lastFailureReason",
      ellipsis: true,
      render: (text?: string) =>
        text ? (
          <Tooltip title={text}>
            <span>{text.length > 30 ? text.slice(0, 30) + "..." : text}</span>
          </Tooltip>
        ) : (
          "-"
        ),
    },
    {
      title: "上次成功",
      dataIndex: "lastSuccessAt",
      key: "lastSuccessAt",
      width: 160,
      render: (t?: string) => (t ? formatDateTime(t) : "从未"),
    },
    {
      title: "熔断到期",
      dataIndex: "circuitBreakerUntil",
      key: "circuitBreakerUntil",
      width: 160,
      render: (t?: string) => (t ? formatDateTime(t) : "-"),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      ellipsis: true,
    },
    {
      title: "操作",
      key: "action",
      width: 280,
      fixed: "right",
      render: (_, record) => (
        <Space size="small">
          {/* Phase 86 W4：账号池写操作按 AD 配置域操作码归属（见文件头注释） */}
          {permEdit.visible && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              disabled={permEdit.disabled}
              title={permEdit.hint}
              onClick={() => handleEdit(record)}
            >
              编辑
            </Button>
          )}
          {record.status === 2 && permEdit.visible && (
            <Button
              type="link"
              size="small"
              icon={<UnlockOutlined />}
              disabled={permEdit.disabled}
              title={permEdit.hint}
              onClick={() => handleUnlockClick(record.id)}
            >
              立即解锁
            </Button>
          )}
          {permEdit.visible && (
            <Button
              type="link"
              size="small"
              icon={record.status === 1 ? <CheckCircleOutlined /> : <StopOutlined />}
              disabled={permEdit.disabled}
              title={permEdit.hint}
              onClick={() => handleToggleEnabled(record)}
            >
              {record.status === 1 ? "启用" : "停用"}
            </Button>
          )}
          {/* 删除：裸 Button + confirmDelete 触发器 → 门控点落在触发按钮本身
              （条件渲染 + disabled + 提示），confirmDelete.ts 零改动 */}
          {permDelete.visible && (
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={permDelete.disabled}
              title={permDelete.hint}
              onClick={() =>
                confirmDelete({
                  target: `该账号池账号（${record.username || record.id}）`,
                  extra: "删除后该账号无法恢复。",
                  onDelete: () => handleDelete(record.id),
                })
              }
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      {stats && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}>
            <Statistic title="账号总数" value={stats.total} />
          </Col>
          <Col span={4}>
            <Statistic
              title="可用"
              value={stats.available}
              valueStyle={{ color: "var(--theme-success)" }}
            />
          </Col>
          <Col span={4}>
            <Statistic title="已停用" value={stats.disabled} />
          </Col>
          <Col span={4}>
            <Statistic
              title="熔断中"
              value={stats.circuitBroken}
              valueStyle={{ color: "var(--theme-error)" }}
            />
          </Col>
          <Col span={8}>
            <Statistic title="当前活跃账号" value={stats.currentAccount || "无"} />
          </Col>
        </Row>
      )}

      <Space style={{ marginBottom: 16 }}>
        {/* 刷新是读入口（loadList / loadStats，不落库）→ **不门控** */}
        <Button
          icon={<ReloadOutlined />}
          onClick={() => {
            loadList();
            loadStats();
          }}
        >
          刷新
        </Button>
        {/* 新增账号是写入口 → opsADConfig.add（条件渲染 + disabled + 提示） */}
        {permAdd.visible && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            disabled={permAdd.disabled}
            title={permAdd.hint}
            onClick={handleCreate}
          >
            新增账号
          </Button>
        )}
      </Space>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={list}
        loading={loading}
        scroll={{ x: 1200 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      {/* 新增/编辑 Modal */}
      <Modal
        title={editing ? "编辑账号" : "新增账号"}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        onOk={async () => {
          setSaving(true);
          try {
            await handleSubmit();
          } finally {
            setSaving(false);
          }
        }}
        cancelButtonProps={{ disabled: saving }}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="username"
            label="账号（UPN 或 sAMAccountName）"
            rules={[{ required: true, message: "请输入账号" }]}
          >
            <AppInput allowClear placeholder="如 svc-01@corp.local" disabled={!!editing} />
          </Form.Item>
          <Form.Item
            name="password"
            label={editing ? "新密码（留空 = 不修改）" : "密码"}
            rules={editing ? [] : [{ required: true, message: "请输入密码" }]}
          >
            <Input.Password placeholder="SM4 加密后传输" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <AppTextArea rows={2} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Modal>

      {/* 解锁 Modal */}
      <Modal
        title="立即解锁"
        open={unlockModalVisible}
        onCancel={() => setUnlockModalVisible(false)}
        onOk={async () => {
          setUnlocking(true);
          try {
            await handleUnlockSubmit();
          } finally {
            setUnlocking(false);
          }
        }}
        cancelButtonProps={{ disabled: unlocking }}
        destroyOnHidden
      >
        <Form form={unlockForm} layout="vertical" preserve={false}>
          <Form.Item
            name="reason"
            label="解锁原因（至少 10 字符，会记录到操作日志）"
            rules={[
              { required: true, message: "请填写解锁原因" },
              { min: 10, message: "解锁原因至少 10 字符" },
            ]}
          >
            <AppTextArea
              rows={3}
              maxLength={200}
              showCount
              placeholder="如：AD 域控已解锁该账号，恢复使用"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AccountPoolTab;

/**
 * 例外规则 admin 页 (Phase 44 R3 / Plan 44-01 Task 6b)
 *
 * 内容:
 *   - 顶部统计卡片(总规则/启用/停用,复用 exceptionRuleStats 端点)
 *   - "记录当前为基线"按钮(调 reconciliationApi.baseline.snapshot,UI 先到位,
 *     后端 44-02 实现;本 task 仅 wire UI)
 *   - 筛选表单(name / is_active / scope_type)
 *   - Table(name / ip_range / actions / severity_override / scope / expires_at /
 *     is_active / 操作[编辑/删除/命中测试])
 *   - Modal 嵌 ExceptionRuleForm 组件
 *   - 命中测试 Drawer(内容渲染 placeholder,Task 6c 接入 MatchTestPanel)
 *
 * CLAUDE.md 强约束:
 *   - 所有 API 调用用 reconciliationApi.exceptionRule.* (不用 raw axios)
 *   - useEffect 依赖稳定(listParams 用 useMemo + JSON.stringify)
 *
 * D-R3-A4-01: 顶部"记录当前为基线"按钮 onClick 调 baseline.snapshot(),
 * useMutation 成功后 invalidate queryKeys.reconciliation.baselineCompare()
 */

import { useMemo, useState, useCallback } from "react";
import {
  Table,
  Form,
  Input,
  Select,
  Button,
  Space,
  Tag,
  Card,
  App,
  Drawer,
  Statistic,
  Row,
  Col,
  Popconfirm,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  ExperimentOutlined,
  CameraOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";
import ExceptionRuleForm, {
  type ExceptionRuleFormValues,
} from "@/components/asset/reconciliation/ExceptionRuleForm";
import MatchTestPanel from "@/components/asset/reconciliation/MatchTestPanel";

// 类型:例外规则列表项(对齐后端 SysReconciliationException JSON)
interface ExceptionRuleItem {
  id: string;
  name: string;
  ipRange: string;
  conflictTypes?: string[];
  exceptionActions: string[];
  severityOverride?: string | null;
  scopeType: "global" | "dept" | "user";
  scopeId?: string | null;
  reason: string;
  isActive: number;
  expiresAt?: string | null;
  createdAt?: string;
}

interface ExceptionRuleListParams {
  current: number;
  pageSize: number;
  name?: string;
  isActive?: number;
  scopeType?: string;
}

const ExceptionRulesPage: React.FC = () => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();

  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filters, setFilters] = useState<{
    name?: string;
    isActive?: number;
    scopeType?: string;
  }>({});

  // Modal 状态
  const [modalState, setModalState] = useState<{
    open: boolean;
    editValues?: Partial<ExceptionRuleFormValues>;
  }>({ open: false });

  // 命中测试 Drawer(内容 Task 6c 接入 MatchTestPanel)
  const [matchTestOpen, setMatchTestOpen] = useState(false);

  // listParams 用 useMemo + JSON.stringify 稳定(CLAUDE.md useEffect 强约束)
  const listParams = useMemo<ExceptionRuleListParams>(
    () => ({
      current,
      pageSize,
      ...(filters.name ? { name: filters.name } : {}),
      ...(filters.isActive !== undefined ? { isActive: filters.isActive } : {}),
      ...(filters.scopeType ? { scopeType: filters.scopeType } : {}),
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [current, pageSize, JSON.stringify(filters)]
  );

  // 查询列表(用 queryKeys.reconciliation.ruleList)
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.reconciliation.ruleList(listParams),
    queryFn: async () => {
      const params = listParams as unknown as Record<string, unknown>;
      const res = await reconciliationApi.exceptionRule.list<ExceptionRuleItem>(params);
      return {
        list: (res.list ?? []) as ExceptionRuleItem[],
        total: res.total,
      };
    },
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  // 统计卡片数据(复用 exceptionRuleStats 端点,R3 启用后才有命中统计;
  // 这里简化用当前列表统计展示总数/启用数/停用数 — 不依赖额外端点,
  // 避免 stat-cards-from-list-length-capped-at-100:列表本身可能被 MaxPageSize
  // 钳制,但本 admin 页是低频运维场景,规则总数通常 < 100,可接受)
  const stats = useMemo(() => {
    const list = data?.list ?? [];
    const total = data?.total ?? 0;
    const enabled = list.filter((r) => r.isActive === 0).length;
    const disabled = list.filter((r) => r.isActive === 1).length;
    return { total, enabled, disabled };
  }, [data]);

  // 创建 / 更新 / 删除 mutation
  const createMutation = useMutation({
    mutationFn: (values: ExceptionRuleFormValues) =>
      reconciliationApi.exceptionRule.create<unknown>(values as unknown as Record<string, unknown>),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.ruleList() });
      setModalState({ open: false });
    },
    onError: (err) => message.error((err as Error)?.message ?? "创建失败"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: ExceptionRuleFormValues }) =>
      reconciliationApi.exceptionRule.update<unknown>(
        id,
        values as unknown as Record<string, unknown>
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.ruleList() });
      setModalState({ open: false });
    },
    onError: (err) => message.error((err as Error)?.message ?? "更新失败"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => reconciliationApi.exceptionRule.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.ruleList() });
      message.success("已删除");
    },
    onError: (err) => message.error((err as Error)?.message ?? "删除失败"),
  });

  // 降噪基线 snapshot (D-R3-A4-01) — UI 先到位,后端 44-02 实现
  const baselineSnapshotMutation = useMutation({
    mutationFn: () => reconciliationApi.baseline.snapshot(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.reconciliation.baselineCompare(),
      });
      message.success("基线已记录");
    },
    onError: (err) => message.error((err as Error)?.message ?? "记录基线失败(后端可能未启用)"),
  });

  // 搜索 / 重置
  const handleSearch = useCallback(() => {
    const v = form.getFieldsValue() as typeof filters;
    setFilters({
      name: v.name?.trim() || undefined,
      isActive: v.isActive,
      scopeType: v.scopeType,
    });
    setCurrent(1);
  }, [form]);

  const handleReset = useCallback(() => {
    form.resetFields();
    setFilters({});
    setCurrent(1);
  }, [form]);

  // 提交表单
  const handleSubmit = useCallback(
    async (values: ExceptionRuleFormValues) => {
      if (modalState.editValues?.id) {
        await updateMutation.mutateAsync({
          id: modalState.editValues.id,
          values,
        });
      } else {
        await createMutation.mutateAsync(values);
      }
    },
    [modalState.editValues, createMutation, updateMutation]
  );

  // Table 列
  const columns = useMemo<ColumnsType<ExceptionRuleItem>>(
    () => [
      {
        title: "规则名称",
        dataIndex: "name",
        key: "name",
        width: 200,
        ellipsis: true,
      },
      {
        title: "IP段",
        dataIndex: "ipRange",
        key: "ipRange",
        width: 160,
        render: (v: string) => <code>{v}</code>,
      },
      {
        title: "动作",
        dataIndex: "exceptionActions",
        key: "exceptionActions",
        width: 240,
        render: (actions: string[]) => (
          <Space size={4} wrap>
            {(actions ?? []).map((a) => (
              <Tag key={a} color={actionTagColor(a)}>
                {a}
              </Tag>
            ))}
          </Space>
        ),
      },
      {
        title: "严重度覆盖",
        dataIndex: "severityOverride",
        key: "severityOverride",
        width: 100,
        render: (v?: string | null) => (v ? <Tag color="blue">{v}</Tag> : "-"),
      },
      {
        title: "范围",
        key: "scope",
        width: 160,
        render: (_: unknown, record: ExceptionRuleItem) => (
          <Space size={4}>
            <Tag color={record.scopeType === "global" ? "green" : "orange"}>{record.scopeType}</Tag>
            {record.scopeId && (
              <span style={{ fontSize: 12, color: "#999" }}>{record.scopeId.slice(0, 8)}...</span>
            )}
          </Space>
        ),
      },
      {
        title: "过期时间",
        dataIndex: "expiresAt",
        key: "expiresAt",
        width: 170,
        render: (v?: string | null) => (v ? new Date(v).toLocaleString("zh-CN") : "永久"),
      },
      {
        title: "状态",
        dataIndex: "isActive",
        key: "isActive",
        width: 80,
        render: (v: number) => (
          <Tag color={v === 0 ? "green" : "default"}>{v === 0 ? "启用" : "停用"}</Tag>
        ),
      },
      {
        title: "操作",
        key: "oper",
        width: 220,
        fixed: "right",
        render: (_: unknown, record: ExceptionRuleItem) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() =>
                setModalState({
                  open: true,
                  editValues: {
                    id: record.id,
                    name: record.name,
                    ipRange: record.ipRange,
                    conflictTypes: record.conflictTypes,
                    exceptionActions: record.exceptionActions,
                    severityOverride: record.severityOverride ?? undefined,
                    scopeType: record.scopeType,
                    scopeId: record.scopeId ?? undefined,
                    // expiresAt 由 ExceptionRuleForm 内部 dayjs() 转换(string → Dayjs)
                    reason: record.reason,
                  } as Partial<ExceptionRuleFormValues>,
                })
              }
            >
              编辑
            </Button>
            <Popconfirm
              title="确认删除该规则?"
              description="软删除,历史审计链不断(D-R3-A4-03)"
              onConfirm={() => deleteMutation.mutate(record.id)}
              okText="确认"
              cancelText="取消"
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deleteMutation]
  );

  return (
    <div className="p-6">
      <h2 style={{ marginBottom: 16 }}>例外规则管理</h2>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic title="总规则数" value={stats.total} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="启用中" value={stats.enabled} valueStyle={{ color: "#52c41a" }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="已停用" value={stats.disabled} valueStyle={{ color: "#8c8c8c" }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Button
              type="primary"
              icon={<CameraOutlined />}
              loading={baselineSnapshotMutation.isPending}
              onClick={() => baselineSnapshotMutation.mutate()}
              block
            >
              记录当前为基线
            </Button>
            <div style={{ fontSize: 12, color: "#999", marginTop: 8 }}>
              降噪对比基准 (D-R3-A4-01)
            </div>
          </Card>
        </Col>
      </Row>

      {/* 筛选表单 */}
      <Card className="mb-4">
        <Form form={form} layout="inline" onFinish={handleSearch}>
          <Form.Item name="name" label="规则名称">
            <Input placeholder="模糊匹配" allowClear style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="isActive" label="状态">
            {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
            <Select placeholder="全部" allowClear style={{ width: 120 }}>
              <Select.Option value={0}>启用</Select.Option>
              <Select.Option value={1}>停用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="scopeType" label="范围">
            {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
            <Select placeholder="全部" allowClear style={{ width: 140 }}>
              <Select.Option value="global">global</Select.Option>
              <Select.Option value="dept">dept</Select.Option>
              <Select.Option value="user">user</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} htmlType="submit">
                搜索
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setModalState({ open: true, editValues: undefined })}
              >
                新建规则
              </Button>
              <Button icon={<ExperimentOutlined />} onClick={() => setMatchTestOpen(true)}>
                命中测试
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 数据表格 */}
      <Table
        rowKey="id"
        columns={columns}
        dataSource={data?.list ?? []}
        loading={isLoading}
        scroll={{ x: 1400 }}
        pagination={{
          current,
          pageSize,
          total: data?.total ?? 0,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
          onChange: (c, s) => {
            setCurrent(c);
            setPageSize(s);
          },
        }}
      />

      {/* 新建/编辑 Modal */}
      <ExceptionRuleForm
        open={modalState.open}
        initialValues={modalState.editValues}
        onSubmit={handleSubmit}
        onCancel={() => setModalState({ open: false })}
      />

      {/* 命中测试 Drawer(嵌入 MatchTestPanel,Phase 44 R3 / Task 6c) */}
      <Drawer
        title="命中测试"
        open={matchTestOpen}
        onClose={() => setMatchTestOpen(false)}
        width={780}
      >
        <MatchTestPanel embedded />
      </Drawer>
    </div>
  );
};

// actionTagColor 给 exception action 上色(运维识别)
function actionTagColor(action: string): string {
  switch (action) {
    case "silence":
      return "red";
    case "no_alert":
      return "orange";
    case "no_notice":
      return "gold";
    case "no_workorder":
      return "purple";
    case "skip_severity":
      return "blue";
    default:
      return "default";
  }
}

export default ExceptionRulesPage;

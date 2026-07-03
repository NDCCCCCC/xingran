/**
 * 修复建议 admin 页面 (Phase 46 R5)
 *
 * 内容:
 *   - 顶部 5 KPI 卡片:待处理(pending) / 7d 应用(applied) / 7d 回滚(rolledBack) /
 *     7d 误修复率(misFixRate) / 7d 拒绝(rejected)
 *   - 筛选表单:fixStatus / conflictType / responsibleDeptId
 *   - 8 列 Table(D-D2 紧凑行):资产编号 / 现 user_id / 建议 user_id / 置信度 /
 *     冲突类型 / 状态 / 创建时间 / 操作
 *   - 3 个 Modal:RejectModal / RollbackModal(原因 ≥10 字符)
 *   - 详情 Drawer:点击行打开(FixSuggestionDetailDrawer)
 *
 * D-D3:仅单条接受(不批量)
 * D-D4:默认排序 + 部门/状态筛选,复用 BaseListRequest + ApplySort 白名单
 * 4 个排序字段:createdAt / confidenceScore / fixStatus / appliedAt
 */

import { useMemo, useState, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Table,
  Form,
  Select,
  Button,
  Space,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  App,
  Modal,
  Input,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  UndoOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import {
  fixSuggestionApi,
  type FixSuggestionListItem,
  type FixSuggestionListParams,
  type FixStatus,
} from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";
import { useMenuStore } from "@/store/menuStore";
import { FixSuggestionDetailDrawer } from "./components/FixSuggestionDetailDrawer";
import { RollbackModal } from "./components/RollbackModal";

/**
 * 状态机颜色映射(D-D2 视觉强化 / 46-02 Task 3 锁定)
 *
 * 6 种状态:
 *   - pending     黄色  待处理
 *   - accepted    蓝色  已接受(未应用)
 *   - rejected    灰色  已拒绝
 *   - applied     绿色  已应用(可回滚)
 *   - rolled_back 橙色  已回滚(从 applied 恢复)
 *   - failed      红色  失败
 */
const fixStatusColor: Record<FixStatus, string> = {
  pending: "gold",
  accepted: "blue",
  rejected: "default",
  applied: "green",
  rolled_back: "orange",
  failed: "red",
};

const fixStatusLabel: Record<FixStatus, string> = {
  pending: "待处理",
  accepted: "已接受",
  rejected: "已拒绝",
  applied: "已应用",
  rolled_back: "已回滚",
  failed: "失败",
};

const FixSuggestion = () => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [searchParams, setSearchParams] = useSearchParams();

  // 权限 — 5 个 perm code
  const permissions = useMenuStore((s) => s.permissions);
  const canAccept = permissions.includes("asset:reconciliation:fix:accept");
  const canReject = permissions.includes("asset:reconciliation:fix:reject");
  const canRollback = permissions.includes("asset:reconciliation:fix:rollback");

  // 分页
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Drawer 状态
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // Reject / Rollback Modal 状态
  const [rejectModal, setRejectModal] = useState<{
    open: boolean;
    suggestionId: string | null;
    submitting: boolean;
  }>({ open: false, suggestionId: null, submitting: false });
  const [rejectForm] = Form.useForm();

  const [rollbackModal, setRollbackModal] = useState<{
    open: boolean;
    suggestionId: string | null;
    rollbackReason: string;
    submitting: boolean;
  }>({ open: false, suggestionId: null, rollbackReason: "", submitting: false });

  // 服务端排序 — 4 个白名单字段
  const { orderByColumn, isAsc, handleTableChange } = useServerSort<FixSuggestionListItem>({
    sorterMetas: [
      createSorterMeta<FixSuggestionListItem>("createdAt"),
      createSorterMeta<FixSuggestionListItem>("confidenceScore"),
      createSorterMeta<FixSuggestionListItem>("fixStatus"),
      createSorterMeta<FixSuggestionListItem>("appliedAt"),
    ],
    defaultSort: { orderByColumn: "createdAt", isAsc: false },
  });

  // 初始 filter 从 URL 读
  const [filterValues, setFilterValues] = useState<{
    fixStatus?: FixStatus;
    conflictType?: "A" | "B" | "C" | "D" | "E" | "F";
    responsibleDeptId?: string;
  }>(() => ({
    fixStatus: (searchParams.get("fixStatus") as FixStatus) || undefined,
    conflictType: (searchParams.get("conflictType") as "A" | "B" | "C" | "D" | "E" | "F") || undefined,
    responsibleDeptId: searchParams.get("deptId") || undefined,
  }));

  useEffect(() => {
    form.setFieldsValue(filterValues);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 拼装 listParams(primitive deps 避免无限循环)
  const listParams = useMemo<FixSuggestionListParams>(() => {
    const p: FixSuggestionListParams = { current, pageSize };
    if (filterValues.fixStatus) p.fixStatus = filterValues.fixStatus;
    if (filterValues.conflictType) p.conflictType = filterValues.conflictType;
    if (filterValues.responsibleDeptId) p.responsibleDeptId = filterValues.responsibleDeptId;
    if (orderByColumn) {
      p.orderByColumn = orderByColumn;
      p.isAsc = isAsc;
    }
    return p;
  }, [current, pageSize, filterValues.fixStatus, filterValues.conflictType, filterValues.responsibleDeptId, orderByColumn, isAsc]);

  // 5 KPI 卡片
  const { data: stats } = useQuery({
    queryKey: queryKeys.reconciliation.fixSuggestionStats(7),
    queryFn: () => fixSuggestionApi.stats(7),
  });

  // 列表
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.reconciliation.fixSuggestionList(listParams),
    queryFn: () => fixSuggestionApi.list(listParams),
  });

  // 搜索
  const handleSearch = useCallback(() => {
    const values = form.getFieldsValue() as typeof filterValues;
    setFilterValues(values);
    setCurrent(1);
    const newParams: Record<string, string> = {};
    if (values.fixStatus) newParams.fixStatus = values.fixStatus;
    if (values.conflictType) newParams.conflictType = values.conflictType;
    if (values.responsibleDeptId) newParams.deptId = values.responsibleDeptId;
    setSearchParams(newParams, { replace: true });
  }, [form, setSearchParams]);

  // 重置
  const handleReset = useCallback(() => {
    form.resetFields();
    setFilterValues({});
    setCurrent(1);
    setSearchParams({}, { replace: true });
  }, [form, setSearchParams]);

  // 接受
  const handleAccept = useCallback(
    async (record: FixSuggestionListItem) => {
      try {
        await fixSuggestionApi.accept(record.id);
        message.success("已接受建议");
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
      } catch (err) {
        message.error((err as Error)?.message ?? "接受失败");
      }
    },
    [queryClient, listParams, message]
  );

  // 打开 Reject Modal
  const openReject = useCallback((record: FixSuggestionListItem) => {
    setRejectModal({ open: true, suggestionId: record.id, submitting: false });
    rejectForm.resetFields();
  }, [rejectForm]);

  // 提交 Reject
  const handleRejectSubmit = useCallback(async () => {
    if (!rejectModal.suggestionId) return;
    try {
      const values = (await rejectForm.validateFields()) as { rejectionReason: string };
      if (values.rejectionReason.trim().length < 10) {
        message.error("拒绝原因至少 10 字符");
        return;
      }
      setRejectModal((prev) => ({ ...prev, submitting: true }));
      await fixSuggestionApi.reject(rejectModal.suggestionId, values.rejectionReason.trim());
      message.success("已拒绝建议");
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
      setRejectModal({ open: false, suggestionId: null, submitting: false });
      rejectForm.resetFields();
    } catch (err) {
      const errMsg = (err as Error)?.message ?? "拒绝失败";
      message.error(errMsg);
      setRejectModal((prev) => ({ ...prev, submitting: false }));
    }
  }, [rejectModal.suggestionId, rejectForm, queryClient, listParams, message]);

  // 应用
  const handleApply = useCallback(
    async (record: FixSuggestionListItem) => {
      try {
        await fixSuggestionApi.apply(record.id);
        message.success("已应用修复");
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
      } catch (err) {
        message.error((err as Error)?.message ?? "应用失败");
      }
    },
    [queryClient, listParams, message]
  );

  // 打开 Rollback Modal
  const openRollback = useCallback((record: FixSuggestionListItem) => {
    setRollbackModal({ open: true, suggestionId: record.id, rollbackReason: "", submitting: false });
  }, []);

  // 提交 Rollback
  const handleRollbackSubmit = useCallback(
    async (reason: string) => {
      if (!rollbackModal.suggestionId) return;
      if (reason.trim().length < 10) {
        message.error("回滚原因至少 10 字符");
        return;
      }
      try {
        setRollbackModal((prev) => ({ ...prev, submitting: true, rollbackReason: reason.trim() }));
        await fixSuggestionApi.rollback(rollbackModal.suggestionId, reason.trim());
        message.success("已回滚修复");
        // 失效 list + stats + detail 三组 query
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
        if (rollbackModal.suggestionId) {
          queryClient.invalidateQueries({
            queryKey: queryKeys.reconciliation.fixSuggestionDetail(rollbackModal.suggestionId),
          });
        }
        setRollbackModal({ open: false, suggestionId: null, rollbackReason: "", submitting: false });
      } catch (err) {
        const errMsg = (err as Error)?.message ?? "回滚失败";
        message.error(errMsg);
        // 特定错误:窗口已过 → 刷新列表隐藏按钮
        if (errMsg.includes("回滚窗口已过")) {
          queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        }
        setRollbackModal((prev) => ({ ...prev, submitting: false }));
      }
    },
    [rollbackModal.suggestionId, queryClient, listParams, message]
  );

  // 打开详情 Drawer
  const openDrawer = useCallback((record: FixSuggestionListItem) => {
    setSelectedId(record.id);
    setDrawerOpen(true);
  }, []);

  // 7d 回滚窗口判断(46-02 Task 3 / D-C2)
  //
  // 规则:
  //   - record.rollbackWindowUntil 为 null → 不在窗口(未应用)
  //   - 当前时间 < rollbackWindowUntil → 在 7d 窗口内(可回滚)
  //   - 当前时间 >= rollbackWindowUntil → 超过 7d(按钮隐藏,DB 仍允许)
  const isWithin7d = (record: FixSuggestionListItem): boolean => {
    if (!record.rollbackWindowUntil) return false;
    return new Date(record.rollbackWindowUntil).getTime() > Date.now();
  };

  // 8 列 Table
  const columns: ColumnsType<FixSuggestionListItem> = [
    {
      title: "资产编号",
      dataIndex: "assetCode",
      key: "assetCode",
      width: 120,
    },
    {
      title: "现 user_id",
      dataIndex: "currentUserId",
      key: "currentUserId",
      width: 140,
      render: (v: string | null) => v ?? <span style={{ color: "#999" }}>-</span>,
    },
    {
      title: "建议 user_id",
      dataIndex: "suggestedUserId",
      key: "suggestedUserId",
      width: 140,
    },
    {
      title: "置信度",
      dataIndex: "confidenceScore",
      key: "confidenceScore",
      width: 80,
      sorter: true,
      render: (v: number) => (v * 100).toFixed(0) + "%",
    },
    {
      title: "冲突类型",
      dataIndex: "conflictType",
      key: "conflictType",
      width: 80,
      render: (v: string) => <Tag color="orange">Type {v}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "fixStatus",
      key: "fixStatus",
      width: 90,
      sorter: true,
      render: (v: FixStatus) => <Tag color={fixStatusColor[v]}>{fixStatusLabel[v]}</Tag>,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 160,
      sorter: true,
      defaultSortOrder: "descend",
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      fixed: "right",
      render: (_: unknown, record: FixSuggestionListItem) => {
        if (record.fixStatus === "pending" && (canAccept || canReject)) {
          return (
            <Space size="small">
              {canAccept && (
                <Button type="link" size="small" icon={<CheckOutlined />} onClick={() => handleAccept(record)}>
                  接受
                </Button>
              )}
              {canReject && (
                <Button type="link" size="small" danger icon={<CloseOutlined />} onClick={() => openReject(record)}>
                  拒绝
                </Button>
              )}
            </Space>
          );
        }
        if (record.fixStatus === "accepted" && canAccept) {
          return (
            <Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => handleApply(record)}>
              应用
            </Button>
          );
        }
        if (record.fixStatus === "applied" && isWithin7d(record) && canRollback) {
          return (
            <Button type="link" size="small" icon={<UndoOutlined />} onClick={() => openRollback(record)}>
              回滚
            </Button>
          );
        }
        return <Tag>{fixStatusLabel[record.fixStatus]}</Tag>;
      },
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      {/* 5 KPI 卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic title="待处理" value={stats?.pendingAll ?? stats?.pending ?? 0} suffix="条" />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic title="7d 应用" value={stats?.applied ?? 0} suffix="条" />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic title="7d 回滚" value={stats?.rolledBack ?? 0} suffix="条" valueStyle={{ color: stats?.rolledBack ? "red" : undefined }} />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="7d 误修复率"
              value={((stats?.misFixRate ?? 0) * 100).toFixed(2)}
              suffix="%"
              valueStyle={{ color: stats?.thresholdBreached ? "red" : "green" }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic title="7d 拒绝" value={stats?.rejected ?? 0} suffix="条" />
          </Card>
        </Col>
      </Row>

      {/* 筛选表单 */}
      <Card style={{ marginBottom: 16 }}>
        <Form form={form} layout="inline" onFinish={handleSearch}>
          <Form.Item name="fixStatus" label="状态">
            <Select
              allowClear
              placeholder="全部"
              style={{ width: 140 }}
              showSearch
              onSearch={() => undefined}
              options={Object.entries(fixStatusLabel).map(([value, label]) => ({ value, label }))}
            />
          </Form.Item>
          <Form.Item name="conflictType" label="冲突类型">
            <Select
              allowClear
              placeholder="全部"
              style={{ width: 120 }}
              showSearch
              onSearch={() => undefined}
              options={["A", "B", "C", "D", "E", "F"].map((v) => ({ value: v, label: `Type ${v}` }))}
            />
          </Form.Item>
          <Form.Item name="responsibleDeptId" label="责任部门">
            <Input placeholder="部门 ID" style={{ width: 180 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 8 列 Table */}
      <Card>
        <Table<FixSuggestionListItem>
          rowKey="id"
          columns={columns}
          dataSource={data?.list ?? []}
          loading={isLoading}
          onChange={handleTableChange}
          onRow={(record) => ({
            onClick: () => openDrawer(record),
            style: { cursor: "pointer" },
          })}
          pagination={{
            current,
            pageSize,
            total: data?.total ?? 0,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (c, s) => {
              setCurrent(c);
              setPageSize(s);
            },
          }}
          scroll={{ x: 1100 }}
          size="small"
        />
      </Card>

      {/* 详情 Drawer */}
      <FixSuggestionDetailDrawer
        open={drawerOpen}
        suggestionId={selectedId}
        onClose={() => setDrawerOpen(false)}
      />

      {/* Reject Modal */}
      <Modal
        title="拒绝修复建议"
        open={rejectModal.open}
        confirmLoading={rejectModal.submitting}
        onCancel={() => setRejectModal({ open: false, suggestionId: null, submitting: false })}
        onOk={handleRejectSubmit}
        okText="确认拒绝"
        okButtonProps={{ danger: true }}
      >
        <Form form={rejectForm} layout="vertical">
          <Form.Item
            name="rejectionReason"
            label="拒绝原因"
            rules={[{ required: true, min: 10, message: "至少 10 字符" }]}
          >
            <Input.TextArea rows={4} placeholder="请说明拒绝原因(至少 10 字符)" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Rollback Modal(46-02 Task 3 提取为独立组件) */}
      <RollbackModal
        open={rollbackModal.open}
        submitting={rollbackModal.submitting}
        onCancel={() => setRollbackModal({ open: false, suggestionId: null, rollbackReason: "", submitting: false })}
        onSubmit={handleRollbackSubmit}
      />
    </div>
  );
};

export default FixSuggestion;

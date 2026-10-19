/**
 * Phase 59 / GUARD-02 — AD 配置页「对账」只读 Tab
 *
 * 数据源：59-03 的 `POST /ad-domain/reconciliation/mismatch`（视图 v_dept_ou_mismatch 实时计算）。
 *
 * 契约（59-UI-SPEC.md + D-17）：
 * - 权限门是 UX 门：无 `ops:ad:config:reconciliation:list` 时在 Tab 内渲染 403 且**不发任何请求**；
 *   Tab 本身始终可见（真实鉴权在后端 RequirePermissions）。
 * - 只读：无修复/删除/搬移/导出入口（自动修复已 Deferred）。
 * - 查询为显式提交：筛选控件 onChange 只改草稿值，不触发请求（点击「查询」或关键字 Enter 才提交）。
 * - 不展示原始 UUID 列、无操作列、无排序。
 */
import { useCallback, useEffect, useState } from "react";
import { Result, Table, Tag, Tooltip } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { FC } from "react";
import {
  getDeptOUMismatchList,
  type DeptOUMismatchListRequest,
  type DeptOUMismatchRow,
  type DeptOUMismatchType,
} from "@/lib/adDomainApi";
import { useMenuStore } from "@/store/menuStore";
import EmptyStateWithAction from "@/components/shared/EmptyStateWithAction";

/** D-15 锁定的权限点（与后端 RequirePermissions / migration_218 同一字面） */
export const RECONCILIATION_PERM = "ops:ad:config:reconciliation:list";

/** mismatch_type → 中文标签 / antd 预设 Tag color（UI-SPEC Color 契约，禁 hex） */
const MISMATCH_TYPE_META: Record<DeptOUMismatchType, { label: string; color: string }> = {
  user_ou_mismatch: { label: "用户 OU 与部门映射不一致", color: "orange" },
  dept_soft_deleted: { label: "映射指向已删除部门", color: "red" },
  mapping_disabled: { label: "映射已停用", color: "gold" },
  config_unrouted: { label: "配置无法归属", color: "blue" },
};

/** 空值统一渲染 "-"（UI-SPEC partial 规则） */
function renderText(value?: string | null) {
  return value ? value : "-";
}

/**
 * 组合 rowKey（UI-SPEC 锁定）：部门级与用户级行都唯一，
 * NULL 键统一落空串，避免 undefined 参与拼接。
 */
function rowKeyOf(row: DeptOUMismatchRow) {
  return `${row.mismatchType}-${row.deptId ?? ""}-${row.userId ?? ""}-${row.adConfigId ?? ""}`;
}

const ReconciliationTab: FC = () => {
  const permissions = useMenuStore((s) => s.permissions);
  const canView = permissions.includes(RECONCILIATION_PERM);

  const [list, setList] = useState<DeptOUMismatchRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const loadList = useCallback(async () => {
    if (!canView) return;
    setLoading(true);
    setError(null);
    try {
      const params: DeptOUMismatchListRequest = { current, pageSize };
      const res = await getDeptOUMismatchList(params);
      setList(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  }, [canView, current, pageSize]);

  // 依赖只含原始值 / useCallback 引用（CLAUDE.md useEffect 稳定性规则）
  useEffect(() => {
    if (!canView) return;
    void loadList();
  }, [canView, loadList]);

  const columns: ColumnsType<DeptOUMismatchRow> = [
    {
      title: "不一致类型",
      dataIndex: "mismatchType",
      key: "mismatchType",
      width: 180,
      render: (type: DeptOUMismatchRow["mismatchType"]) => {
        const meta = MISMATCH_TYPE_META[type];
        if (!meta) return renderText(type);
        return (
          <Tag color={meta.color} style={{ whiteSpace: "nowrap" }}>
            {meta.label}
          </Tag>
        );
      },
    },
    {
      title: "部门",
      dataIndex: "deptName",
      key: "deptName",
      width: 180,
      render: (name?: string | null) => renderText(name),
    },
    {
      title: "AD 配置",
      dataIndex: "configName",
      key: "configName",
      width: 150,
      render: (name?: string | null) => renderText(name),
    },
    {
      title: "映射 OU",
      dataIndex: "mappedOuDn",
      key: "mappedOuDn",
      width: 260,
      ellipsis: true,
      render: (dn?: string | null) =>
        dn ? (
          <Tooltip title={dn}>
            <span>{dn}</span>
          </Tooltip>
        ) : (
          "-"
        ),
    },
    {
      title: "用户",
      dataIndex: "username",
      key: "username",
      width: 160,
      render: (name?: string | null) => renderText(name),
    },
    {
      title: "用户 OU",
      dataIndex: "userAdOuDn",
      key: "userAdOuDn",
      width: 260,
      ellipsis: true,
      render: (dn?: string | null) =>
        dn ? (
          <Tooltip title={dn}>
            <span>{dn}</span>
          </Tooltip>
        ) : (
          "-"
        ),
    },
    {
      title: "详情",
      dataIndex: "mismatchDetail",
      key: "mismatchDetail",
      width: 320,
      ellipsis: true,
      render: (detail?: string | null) =>
        detail ? (
          <Tooltip title={detail}>
            <span>{detail}</span>
          </Tooltip>
        ) : (
          "-"
        ),
    },
  ];

  // 权限门（D-17）：Tab 内渲染 403，且不进入任何请求 effect（loadList 已按 canView 短路）
  if (!canView) {
    return (
      <Result
        status="403"
        title="无对账查看权限"
        subTitle="请联系管理员为当前角色授予 ops:ad:config:reconciliation:list 权限后重新登录。"
      />
    );
  }

  return (
    <div>
      {error && <div>对账数据加载失败，请稍后重试。</div>}
      {list.length === 0 && !loading && !error ? (
        <EmptyStateWithAction
          title="未发现不一致条目"
          description="当前视图为实时计算。可调整筛选条件，或先在「配置列表」完成部门同步后再查看。"
        />
      ) : (
        <Table<DeptOUMismatchRow>
          rowKey={rowKeyOf}
          columns={columns}
          dataSource={list}
          loading={loading}
          scroll={{ x: 1400 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setCurrent(p);
              setPageSize(ps);
            },
          }}
        />
      )}
    </div>
  );
};

/** 无权限态文案（UI-SPEC Copywriting Contract 逐字，见上方权限门渲染） */

export default ReconciliationTab;

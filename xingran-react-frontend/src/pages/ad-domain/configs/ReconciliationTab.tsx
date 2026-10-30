/**
 * Phase 59 / GUARD-02 — AD 配置页「对账」只读 Tab
 *
 * 数据源：59-03 的 `POST /ad-domain/reconciliation/mismatch`（视图 v_dept_ou_mismatch 实时计算）。
 *
 * 契约（59-UI-SPEC.md + D-17）：
 * - 权限门是 UX 门：无 `ops:ad:config:reconciliation:list` 时在 Tab 内渲染 403 且**不发任何请求**；
 *   Tab 本身始终可见（真实鉴权在后端 RequirePermissions）。
 * - 只读：无修复/删除/搬移/导出入口（自动修复已 Deferred）。
 * - 查询为显式提交：筛选控件 onChange 只改草稿值（存于 Form），不触发请求；
 *   点击「查询」或关键字 Enter 才提交；「重置」清空后回到第一页立即查询。
 * - 翻页/改页大小保持当前已提交筛选条件（已提交值在 loadList 依赖里）。
 * - 状态面：失败 → ErrorAlertWithRetry；空 → EmptyStateWithAction（不渲染空表头）；加载 → Table loading。
 * - 不展示原始 UUID 列、无操作列、无排序。
 */
import { useCallback, useEffect, useState } from "react";
// Phase 71 (W5)：裸 Input/Select → 契约组件
import { AppInput, AppSelect } from "@/components/ui";
import { Button, Form, Result, Space, Table, Tag, Tooltip } from "antd";
import type { FormInstance } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { FC } from "react";
import {
  getDeptOUMismatchList,
  type DeptOUMismatchListRequest,
  type DeptOUMismatchRow,
  type DeptOUMismatchType,
} from "@/lib/adDomainApi";
import { useMenuStore } from "@/store/menuStore";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { SimpleDept } from "@/lib/dutyApi";
import DepartmentTreeSelect from "@/components/shared/DepartmentTreeSelect";
import EmptyStateWithAction from "@/components/shared/EmptyStateWithAction";
import ErrorAlertWithRetry from "@/components/shared/ErrorAlertWithRetry";
// 本 Tab 专属样式：空态标题字重 600 !important 覆盖共享组件的 inline 16px/500（见 css 文件头注释）
import "./ReconciliationTab.css";
// v1.24 P82：MISMATCH_TYPE_META / rowKeyOf 拆至 constants.ts / rowKey.ts（react-refresh 边界）
import { MISMATCH_TYPE_META } from "./constants";
import { rowKeyOf } from "./rowKey";

/** D-15 锁定的权限点（与后端 RequirePermissions / migration_218 同一字面） */
export const RECONCILIATION_PERM = "ops:ad:config:reconciliation:list";

/** 筛选下拉选项（与 MISMATCH_TYPE_META 单一事实源派生） */
const MISMATCH_TYPE_OPTIONS = (Object.keys(MISMATCH_TYPE_META) as DeptOUMismatchType[]).map(
  (value) => ({ value, label: MISMATCH_TYPE_META[value].label })
);

/** 空部门树常量：避免内联 [] 造成每次渲染新引用（rc-tree motion bail 陷阱，参照 DeptTree #185） */
const EMPTY_DEPTS: SimpleDept[] = [];

/** 空值统一渲染 "-"（UI-SPEC partial 规则） */
function renderText(value?: string | null) {
  return value ? value : "-";
}

export interface ReconciliationTabProps {
  /** 注入受控 Form 实例（测试 seam；缺省用组件内部实例） */
  form?: FormInstance;
}

const ReconciliationTab: FC<ReconciliationTabProps> = ({ form: injectedForm }) => {
  const permissions = useMenuStore((s) => s.permissions);
  const canView = permissions.includes(RECONCILIATION_PERM);

  const [internalForm] = Form.useForm<{ mismatchType?: DeptOUMismatchType; deptId?: string; keyword?: string }>();
  const form = injectedForm ?? internalForm;

  const deptTree = useDeptTree();

  const [list, setList] = useState<DeptOUMismatchRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 已提交筛选（仅「查询 / 重置 / Enter」改变；原始值才能进依赖）
  const [mismatchType, setMismatchType] = useState<DeptOUMismatchType | "">("");
  const [deptId, setDeptId] = useState("");
  const [keyword, setKeyword] = useState("");

  // 参数化请求：事件驱动的唯一直连通道（相同筛选重复点击查询也重拉）
  const fetchList = useCallback(
    async (params: DeptOUMismatchListRequest) => {
      if (!canView) return;
      setLoading(true);
      setError(null);
      try {
        const res = await getDeptOUMismatchList(params);
        setList(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      } catch (e) {
        setError(e instanceof Error ? e : new Error(String(e)));
      } finally {
        setLoading(false);
      }
    },
    [canView]
  );

  const loadList = useCallback(async () => {
    const params: DeptOUMismatchListRequest = { current, pageSize };
    if (mismatchType) params.mismatchType = mismatchType;
    if (deptId) params.deptId = deptId;
    if (keyword) params.keyword = keyword;
    await fetchList(params);
  }, [current, pageSize, mismatchType, deptId, keyword, fetchList]);

  // 依赖只含 canView 与 loadList（后者依赖原始值）——CLAUDE.md useEffect 稳定性规则
  useEffect(() => {
    if (!canView) return;
    void loadList();
  }, [canView, loadList]);

  // 事件驱动：筛选未变化时 state 不动、effect 不重跑，直接经 fetchList 重拉
  // （保持「相同筛选下重复点击查询也要重新拉取」的既有语义，替代原 queryTick 机制）
  const handleQuery = () => {
    const values = form.getFieldsValue();
    const next = {
      mismatchType: (values.mismatchType ?? "") as DeptOUMismatchType | "",
      deptId: values.deptId ?? "",
      keyword: (values.keyword ?? "").trim(),
    };
    const unchanged =
      next.mismatchType === mismatchType &&
      next.deptId === deptId &&
      next.keyword === keyword &&
      current === 1;
    setMismatchType(next.mismatchType);
    setDeptId(next.deptId);
    setKeyword(next.keyword);
    setCurrent(1);
    if (unchanged) {
      const params: DeptOUMismatchListRequest = { current: 1, pageSize };
      if (next.mismatchType) params.mismatchType = next.mismatchType;
      if (next.deptId) params.deptId = next.deptId;
      if (next.keyword) params.keyword = next.keyword;
      void fetchList(params);
    }
  };

  const handleReset = () => {
    form.resetFields();
    const unchanged = mismatchType === "" && deptId === "" && keyword === "" && current === 1;
    setMismatchType("");
    setDeptId("");
    setKeyword("");
    setCurrent(1);
    if (unchanged) {
      void fetchList({ current: 1, pageSize });
    }
  };

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
      render: (name: string | null | undefined, record: DeptOUMismatchRow) => {
        if (!name && !record.deptDeletedAt) return "-";
        return (
          <span>
            {name ?? "-"}
            {record.deptDeletedAt ? <Tag color="red">已删除</Tag> : null}
          </span>
        );
      },
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
      <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
        <Space size="medium" wrap>
          <Form.Item name="mismatchType" label="不一致类型">
            {/* Phase 71 W5：迁 AppSelect——原 antd 单选默认禁搜，显式 showSearch={false}；
                契约要求显式 onSearch，补空函数不改变行为 */}
            <AppSelect
              placeholder="全部类型"
              allowClear
              style={{ width: 220 }}
              showSearch={false}
              onSearch={() => {}}
              options={MISMATCH_TYPE_OPTIONS}
            />
          </Form.Item>
          <Form.Item name="deptId" label="部门">
            <DepartmentTreeSelect
              departments={deptTree.data ?? EMPTY_DEPTS}
              placeholder="全部部门"
              allowClear
              style={{ width: 240 }}
              loading={deptTree.isLoading}
            />
          </Form.Item>
          <Form.Item name="keyword">
            <AppInput
              placeholder="搜索用户名 / 部门名称 / OU DN"
              allowClear
              style={{ width: 260 }}
              onPressEnter={handleQuery}
            />
          </Form.Item>
          <Button type="primary" onClick={handleQuery} loading={loading}>
            查询
          </Button>
          <Button onClick={handleReset} loading={loading}>
            重置
          </Button>
        </Space>
      </Form>

      {error ? (
        <ErrorAlertWithRetry
          error={error}
          description="对账数据加载失败，请稍后重试。"
          onRetry={loadList}
        />
      ) : list.length === 0 && !loading ? (
        <div className="reconciliation-tab-empty">
          <EmptyStateWithAction
            title="未发现不一致条目"
            description="当前视图为实时计算。可调整筛选条件，或先在「配置列表」完成部门同步后再查看。"
          />
        </div>
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

export default ReconciliationTab;

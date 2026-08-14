/**
 * 异常列表 admin 页面 (Phase 42 R1)
 *
 * 内容:
 *   - 顶部 4 字段筛选表单(conflict_type / severity / asset_code / detected_at RangePicker)
 *   - 9 列 Table:
 *       1. detected_at        (默认 DESC,可排序)
 *       2. conflict_type      (useDict 显示 label)
 *       3. severity           (useDict 显示 label,critical 红)
 *       4. asset_code         (从 JOIN 取)
 *       5. asset_ip           (INET,INET Display)
 *       6. physical_username   (从 reconciliation_normalized MV)
 *       7. responsible_username (从 sys_user JOIN)
 *       8. exception_rule_id  (R3 命中才显示,本 plan 隐藏)
 *       9. operlog_btn         (查看日志 /monitor/operlog?bizId={id} 只读链接)
 *   - 分页 + showSizeChanger + showTotal
 *
 * D-18: R1 不上"标记已解决"按钮(只读)
 * D-05: 用 useSearchParams 读 URL query string 初始化筛选(Dashboard 跳过来 URL 已带 ?type=X)
 *
 * useExceptionList hook 内部用 useMemo + JSON.stringify 稳定 params,
 * 配合 keepPreviousData 翻页不闪烁。
 */

import { useMemo, useState, useEffect, useCallback } from "react";
import { Link, useSearchParams } from "react-router-dom";
import dayjs, { type Dayjs } from "dayjs";
import {
  Table,
  Form,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Tag,
  Card,
  Empty,
  App,
  Modal,
  Switch,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { TablePaginationConfig, FilterValue, SorterResult } from "antd/es/table/interface";
import { SearchOutlined, ReloadOutlined, LinkOutlined, CheckOutlined } from "@ant-design/icons";
import { useQueryClient } from "@tanstack/react-query";
import { useExceptionList } from "@/hooks/useExceptionList";
import { useDict } from "@/hooks/useDict";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import { createSorterMeta, createSorter } from "@/utils/tableHelpers";
import {
  reconciliationApi,
  type ExceptionListItem,
  type ExceptionListParams,
} from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";
import { useMenuStore } from "@/store/menuStore";

const { RangePicker } = DatePicker;

/**
 * 字典 → dictValue->label 映射 helper
 */
function buildDictLabelMap(
  items: { dictValue: string; dictLabel: string; listClass?: string }[] | undefined
): {
  labels: Record<string, string>;
  listClass: Record<string, string | undefined>;
} {
  const labels: Record<string, string> = {};
  const listClass: Record<string, string | undefined> = {};
  (items ?? []).forEach((item) => {
    labels[item.dictValue] = item.dictLabel;
    listClass[item.dictValue] = item.listClass;
  });
  return { labels, listClass };
}

const Exceptions = () => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [searchParams, setSearchParams] = useSearchParams();
  // 权限(D-08):复用 menuStore.permissions(established pattern,见 MACHistoryPage)
  const permissions = useMenuStore((s) => s.permissions);
  const canResolve = permissions.includes("asset:reconciliation:resolve");

  // 当前分页
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Phase 43 R2 / D-A4-04:标记已解决 Modal 状态
  const [resolveModal, setResolveModal] = useState<{
    open: boolean;
    exceptionId: string | null;
    note: string;
    submitting: boolean;
  }>({ open: false, exceptionId: null, note: "", submitting: false });
  const [resolveForm] = Form.useForm();

  // 服务端排序
  const { orderByColumn, isAsc, sortOrder, handleTableChange, resetSort } =
    useServerSort<ExceptionListItem>({
      sorterMetas: [
        createSorterMeta<ExceptionListItem>("detectedAt", "date"),
        createSorterMeta<ExceptionListItem>("conflictType"),
        createSorterMeta<ExceptionListItem>("severity"),
        createSorterMeta<ExceptionListItem>("assetCode"),
      ],
      defaultSort: { orderByColumn: "detectedAt", isAsc: false },
    });

  // 列级 sortOrder：只对当前排序列返回方向，其余 undefined。
  // 修"高亮恒落第一列"——sortOrder 是全局单值，必须按 dataIndex 派发到对应列。
  const getColumnSortOrder = useCallback(
    (field: string): "ascend" | "descend" | null | undefined => {
      if (orderByColumn !== String(field)) return undefined;
      return sortOrder;
    },
    [orderByColumn, sortOrder]
  );

  // 字典 — conflict_type / severity
  const conflictTypeDict = useDict("asset_reconciliation_conflict_type");
  const severityDict = useDict("asset_reconciliation_severity");

  const conflictTypeMap = useMemo(
    () => buildDictLabelMap(conflictTypeDict.data).labels,
    [conflictTypeDict.data]
  );
  const conflictTypeListClass = useMemo(
    () => buildDictLabelMap(conflictTypeDict.data).listClass,
    [conflictTypeDict.data]
  );
  const severityMap = useMemo(
    () => buildDictLabelMap(severityDict.data).labels,
    [severityDict.data]
  );

  // 初始 filter 从 URL 读(D-05 双向打通)
  const [filterValues, setFilterValues] = useState<{
    conflictType?: string;
    severity?: string;
    assetCode?: string;
    detectedAt?: [Dayjs, Dayjs];
    showSilenced?: boolean;
  }>(() => {
    const type = searchParams.get("type") || undefined;
    const severity = searchParams.get("severity") || undefined;
    const from = searchParams.get("from");
    const to = searchParams.get("to");
    let range: [Dayjs, Dayjs] | undefined;
    if (from && to) {
      const f = dayjs(from);
      const t = dayjs(to);
      if (f.isValid() && t.isValid()) {
        range = [f, t];
      }
    }
    return { conflictType: type, severity, detectedAt: range };
  });

  // 同步 URL → 表单初值
  useEffect(() => {
    form.setFieldsValue(filterValues);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 拼装 listParams
  const listParams = useMemo<ExceptionListParams>(() => {
    const p: ExceptionListParams = { current, pageSize };
    if (filterValues.conflictType) p.conflictType = filterValues.conflictType;
    if (filterValues.severity) p.severity = filterValues.severity;
    if (filterValues.assetCode) p.assetCode = filterValues.assetCode;
    if (filterValues.detectedAt && filterValues.detectedAt.length === 2) {
      p.detectedFrom = filterValues.detectedAt[0].startOf("day").toISOString();
      p.detectedTo = filterValues.detectedAt[1].endOf("day").toISOString();
    }
    // R3 / D-R3-A1-01 — silence 默认隐藏,开关打开时透传 showSilenced
    if (filterValues.showSilenced) p.showSilenced = true;
    if (orderByColumn) {
      p.orderByColumn = orderByColumn;
      p.isAsc = isAsc;
    }
    return p;
  }, [current, pageSize, filterValues, orderByColumn, isAsc]);

  const { data, isLoading, isError } = useExceptionList(listParams);

  // 搜索按钮
  const handleSearch = useCallback(() => {
    const values = form.getFieldsValue() as typeof filterValues;
    setFilterValues(values);
    setCurrent(1);
    // 同步到 URL
    const newParams: Record<string, string> = {};
    if (values.conflictType) newParams.type = values.conflictType;
    if (values.severity) newParams.severity = values.severity;
    if (values.assetCode) newParams.assetCode = values.assetCode;
    if (values.detectedAt && values.detectedAt.length === 2) {
      newParams.from = values.detectedAt[0].format("YYYY-MM-DD");
      newParams.to = values.detectedAt[1].format("YYYY-MM-DD");
    }
    setSearchParams(newParams, { replace: true });
  }, [form, setSearchParams]);

  // 重置按钮
  const handleReset = useCallback(() => {
    form.resetFields();
    setFilterValues({});
    setCurrent(1);
    resetSort();
    setSearchParams({}, { replace: true });
  }, [form, resetSort, setSearchParams]);

  // 9 列
  const columns: ColumnsType<ExceptionListItem> = useMemo(
    () => [
      {
        title: "检测时间",
        dataIndex: "detectedAt",
        key: "detectedAt",
        width: 170,
        sorter: createSorter<ExceptionListItem>("detectedAt", "date"),
        sortOrder: getColumnSortOrder("detectedAt"),
        render: (val: string) => (val ? dayjs(val).format("YYYY-MM-DD HH:mm:ss") : "-"),
      },
      {
        title: "冲突类型",
        dataIndex: "conflictType",
        key: "conflictType",
        width: 110,
        sorter: createSorter<ExceptionListItem>("conflictType"),
        sortOrder: getColumnSortOrder("conflictType"),
        render: (val: string) => {
          const label = conflictTypeMap[val] || val;
          const cls = conflictTypeListClass[val];
          return <Tag color={cls || "default"}>{label}</Tag>;
        },
      },
      {
        title: "严重级别",
        dataIndex: "severity",
        key: "severity",
        width: 100,
        sorter: createSorter<ExceptionListItem>("severity"),
        sortOrder: getColumnSortOrder("severity"),
        render: (val: string) => {
          const label = severityMap[val] || val;
          const isCritical = val === "critical";
          return <Tag color={isCritical ? "red" : "default"}>{label}</Tag>;
        },
      },
      {
        title: "资产编号",
        dataIndex: "assetCode",
        key: "assetCode",
        width: 140,
        ellipsis: true,
        sorter: createSorter<ExceptionListItem>("assetCode"),
        sortOrder: getColumnSortOrder("assetCode"),
      },
      {
        title: "资产IP",
        dataIndex: "assetIpDisplay",
        key: "assetIpDisplay",
        width: 130,
      },
      {
        title: "物理使用人",
        dataIndex: "physicalUsername",
        key: "physicalUsername",
        width: 120,
        render: (val: string) => val || "-",
      },
      {
        title: "责任人",
        dataIndex: "responsibleUsername",
        key: "responsibleUsername",
        width: 120,
        render: (val: string) => val || "-",
      },
      {
        title: "例外规则",
        dataIndex: "exceptionRuleId",
        key: "exceptionRuleId",
        width: 130,
        // D-18: R1 例外规则 UI 尚未启用,这里仅展示 id(命中 R3 例外才非空)
        render: (val: string | null) => (val ? <Tag color="purple">{val}</Tag> : "-"),
      },
      {
        // Phase 43 R2 / D-A4-04:标记已解决按钮列
        // 权限控制:无 asset:reconciliation:resolve perm 时不渲染(对齐 T-43-11 mitigation)
        // 已 resolved(异常 resolvedAt 非空)的记录 disabled,避免重复 resolve
        title: "解决",
        key: "resolve_btn",
        width: 110,
        fixed: "right",
        render: (_: unknown, record: ExceptionListItem) => {
          if (!canResolve) return null;
          // resolvedAt 字段在 list 返回里通常为 null 或时间戳;空/null 视为未解决
          const resolved =
            record.resolvedAt !== null &&
            record.resolvedAt !== undefined &&
            record.resolvedAt !== "";
          return (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              disabled={resolved}
              onClick={() => {
                setResolveModal({
                  open: true,
                  exceptionId: record.id,
                  note: "",
                  submitting: false,
                });
                resolveForm.resetFields();
              }}
            >
              {resolved ? "已解决" : "标记已解决"}
            </Button>
          );
        },
      },
      {
        title: "操作",
        key: "operlog_btn",
        width: 110,
        fixed: "right",
        render: (_: unknown, _record: ExceptionListItem) => (
          // R1 范围: 后端 sys_oper_log 表无 biz_id 列,无法按 reconciliation.id 过滤。
          // 链接到操作日志页(去掉 bizId),运维按操作时间/模块人工查找。
          // R2 接入 WebSocket 推送 + 转工单后,oper_log 会写入 business_type=RECONCILIATION
          // + 关联 record.id,届时再加 ?bizId=... 精确跳转。
          //
          // 路径说明: 前端路由是动态从 sys_menu 菜单生成。
          // 父菜单"系统监控" path='monitor',子菜单"日志管理" path='logs'。
          // routeGenerator.resolvePath 拼接成 'monitor/logs'(注意:不是 /logs,也不是后端
          // API 路径 /api/v1/monitor/oper-logs)。React Router 注册的实际 path 是
          // 'monitor/logs',直接 navigate '/monitor/oper-logs' 或 '/logs' 都会 fallback
          // 到 /dashboard。
          //
          // 不要加 target="_blank": SPA 新 tab 启动时 auth 状态与原 tab 不一致
          // (无 cookie 跨 tab 共享,某些 race condition 触发 refresh 失败 → redirect
          // /login)。用 SPA 内导航(同 tab 跳转)最稳,用户可点浏览器后退回异常页。
          <Link to="/monitor/logs">
            <Space size={4}>
              <LinkOutlined />
              查看日志
            </Space>
          </Link>
        ),
      },
    ],
    [
      getColumnSortOrder,
      conflictTypeMap,
      conflictTypeListClass,
      severityMap,
      canResolve,
      resolveForm,
    ]
  );

  // 提交"标记已解决" Modal
  //
  // 流程:
  //   1. validate Form(resolutionNote 可选,但提供就 trim 校验)
  //   2. 调 reconciliationApi.exceptionResolve(id, { resolutionNote })
  //   3. 成功后 message.success + queryClient.invalidateQueries(reconciliation)
  //   4. 关闭 Modal,清空状态
  //   5. 失败 message.error + 保留 Modal
  //
  // 后端 handler 调 operlog.Record(OperTypeUpdate) → 写 sys_oper_log(WORKORDER-02)
  const handleResolveSubmit = useCallback(async () => {
    if (!resolveModal.exceptionId) return;
    try {
      const values = await resolveForm.validateFields();
      setResolveModal((prev) => ({ ...prev, submitting: true }));
      await reconciliationApi.exceptionResolve(resolveModal.exceptionId, {
        resolutionNote: values.resolutionNote?.trim() || undefined,
      });
      message.success("已标记为已解决");
      // 触发 dashboard + 异常列表 query 重新拉取(7d 静默期也自动生效)
      queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });
      setResolveModal({ open: false, exceptionId: null, note: "", submitting: false });
      resolveForm.resetFields();
    } catch (err) {
      // 后端返回 400 "该异常已标记为已解决" 时,同步 invalidate 让 UI 刷新
      const errMsg = (err as Error)?.message ?? "标记失败";
      if (errMsg.includes("已解决")) {
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });
      }
      message.error(errMsg);
      setResolveModal((prev) => ({ ...prev, submitting: false }));
    }
  }, [resolveModal.exceptionId, resolveForm, message, queryClient]);

  // Table.onChange 集成分页 + 排序(per useTableManager 模式)
  const onTableChange = useCallback(
    (
      pagination: TablePaginationConfig,
      _filters: Record<string, FilterValue | null>,
      sorter: SorterResult<ExceptionListItem> | SorterResult<ExceptionListItem>[]
    ) => {
      handleTableChange(pagination, _filters, sorter);
      const newPage = pagination.current ?? 1;
      const newSize = pagination.pageSize ?? pageSize;
      setCurrent(newPage);
      setPageSize(newSize);
      // 同步新排序到下一次 query
      const { orderByColumn: newOrderBy, isAsc: newIsAsc } = resolveSorter(sorter, [
        createSorterMeta<ExceptionListItem>("detectedAt", "date"),
        createSorterMeta<ExceptionListItem>("conflictType"),
        createSorterMeta<ExceptionListItem>("severity"),
        createSorterMeta<ExceptionListItem>("assetCode"),
      ]);
      if (newOrderBy) {
        message.success(`已按 ${newOrderBy} ${newIsAsc ? "升序" : "降序"} 排序`);
      } else {
        message.info("已清空排序");
      }
    },
    [handleTableChange, pageSize, message]
  );

  // 错误兜底
  if (isError && !isLoading) {
    return (
      <div className="p-6">
        <Empty description="异常列表加载失败,请稍后重试" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <h2 style={{ marginBottom: 16 }}>异常列表</h2>

      {/* 筛选表单 */}
      <Card className="mb-4">
        <Form form={form} layout="inline" onFinish={handleSearch}>
          <Form.Item name="conflictType" label="冲突类型">
            {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
            <Select
              placeholder="请选择"
              allowClear
              style={{ width: 160 }}
              loading={conflictTypeDict.isLoading}
            >
              {conflictTypeDict.data?.map((it) => (
                <Select.Option key={it.dictValue} value={it.dictValue}>
                  {it.dictLabel}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="severity" label="严重级别">
            {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
            <Select
              placeholder="请选择"
              allowClear
              style={{ width: 140 }}
              loading={severityDict.isLoading}
            >
              {severityDict.data?.map((it) => (
                <Select.Option key={it.dictValue} value={it.dictValue}>
                  {it.dictLabel}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="assetCode" label="资产编号">
            <Input placeholder="请输入资产编号" allowClear style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="detectedAt" label="检测时间">
            <RangePicker style={{ width: 280 }} placeholder={["开始日期", "结束日期"]} />
          </Form.Item>
          {/* R3 / D-R3-A1-01 — 显示已静默(silence)记录开关,默认 false 隐藏 */}
          <Form.Item name="showSilenced" label="显示已静默" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} htmlType="submit">
                搜索
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
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
        scroll={{ x: 1380 }}
        pagination={{
          current,
          pageSize,
          total: data?.total ?? 0,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={onTableChange}
      />

      {/* Phase 43 R2 / D-A4-04:标记已解决 Modal */}
      <Modal
        title="标记异常已解决"
        open={resolveModal.open}
        onOk={handleResolveSubmit}
        confirmLoading={resolveModal.submitting}
        onCancel={() => {
          if (resolveModal.submitting) return;
          setResolveModal({ open: false, exceptionId: null, note: "", submitting: false });
          resolveForm.resetFields();
        }}
        okText="确认标记"
        cancelText="取消"
        destroyOnHidden
      >
        <Form form={resolveForm} layout="vertical" preserve={false}>
          <Form.Item
            label="解决说明(可选)"
            name="resolutionNote"
            extra="请简要说明解决方式,如:已修正责任人 / 已同步 AD 域账号 等"
          >
            <Input.TextArea
              rows={4}
              maxLength={500}
              showCount
              placeholder="例如:已将 sys_user.user_id 同步到 ops_asset.responsible_user_id"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Exceptions;

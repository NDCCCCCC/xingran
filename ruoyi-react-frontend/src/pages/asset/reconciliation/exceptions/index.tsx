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
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type {
  TablePaginationConfig,
  FilterValue,
  SorterResult,
} from "antd/es/table/interface";
import { SearchOutlined, ReloadOutlined, LinkOutlined } from "@ant-design/icons";
import { useExceptionList } from "@/hooks/useExceptionList";
import { useDict } from "@/hooks/useDict";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import { createSorterMeta, createSorter } from "@/utils/tableHelpers";
import type { ExceptionListItem, ExceptionListParams } from "@/lib/assetApi";

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
  const [form] = Form.useForm();
  const [searchParams, setSearchParams] = useSearchParams();

  // 当前分页
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

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
        sortOrder: sortOrder === null ? undefined : sortOrder,
        render: (val: string) => (val ? dayjs(val).format("YYYY-MM-DD HH:mm:ss") : "-"),
      },
      {
        title: "冲突类型",
        dataIndex: "conflictType",
        key: "conflictType",
        width: 110,
        sorter: createSorter<ExceptionListItem>("conflictType"),
        sortOrder: sortOrder === null ? undefined : sortOrder,
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
        sortOrder: sortOrder === null ? undefined : sortOrder,
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
        sortOrder: sortOrder === null ? undefined : sortOrder,
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
        title: "操作",
        key: "operlog_btn",
        width: 110,
        fixed: "right",
        render: (_: unknown, record: ExceptionListItem) => (
          <Link
            to={`/monitor/operlog?bizId=${encodeURIComponent(record.id)}`}
            target="_blank"
            rel="noopener"
          >
            <Space size={4}>
              <LinkOutlined />
              查看日志
            </Space>
          </Link>
        ),
      },
    ],
    [sortOrder, conflictTypeMap, conflictTypeListClass, severityMap]
  );

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
      const { orderByColumn: newOrderBy, isAsc: newIsAsc } = resolveSorter(
        sorter,
        [
          createSorterMeta<ExceptionListItem>("detectedAt", "date"),
          createSorterMeta<ExceptionListItem>("conflictType"),
          createSorterMeta<ExceptionListItem>("severity"),
          createSorterMeta<ExceptionListItem>("assetCode"),
        ]
      );
      if (newOrderBy) {
        message.success(
          `已按 ${newOrderBy} ${newIsAsc ? "升序" : "降序"} 排序`
        );
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
            <RangePicker
              style={{ width: 280 }}
              placeholder={["开始日期", "结束日期"]}
            />
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
        scroll={{ x: 1280 }}
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
    </div>
  );
};

export default Exceptions;

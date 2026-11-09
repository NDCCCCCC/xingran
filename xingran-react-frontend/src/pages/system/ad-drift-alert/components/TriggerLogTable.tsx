/**
 * 同步漂移告警 — 触发记录表（Phase 144 ALRT-03）
 *
 * 用既有 PageTable 契约组件（内建 loading / 空态 / 错误态 三态 + 分页 showTotal）。
 * 空态接 EmptyStateWithAction；错误态由 PageTable 内建 ErrorAlertWithRetry 统一出口。
 *
 * 依赖稳定性（CLAUDE.md 防无限循环）：
 *   - request / columns / searchFields / extraFilters 全部 useCallback / useMemo 稳定；
 *   - 禁止把对象字面量直接写进 useEffect 依赖；
 *   - extraFilters 只承载 refreshToken（primitive 经 useMemo 包成稳定对象引用），
 *     refreshToken 变化 → PageTable.applyFilters 回第 1 页重载，作为「立即检测 /
 *     WS 新告警」的刷新信号；request 内部显式丢弃该辅助字段，不污染后端入参。
 */
import { useCallback, useMemo } from "react";
import type { FC } from "react";
import { Tag } from "antd";
import dayjs from "dayjs";
import type { ColumnsType } from "antd/es/table";
import PageTable from "@/components/shared/PageTable";
import EmptyStateWithAction from "@/components/shared/EmptyStateWithAction";
import type { SearchField } from "@/components/shared/SearchForm";
import {
  listTriggerLogs,
  type ADDriftTriggerLog,
  type ADDriftTriggerLogParams,
} from "@/lib/adDriftAlertApi";

export interface TriggerLogTableProps {
  /** 刷新信号：index.tsx 在「立即检测成功」/「收到 WS 新告警」时 +1 */
  refreshToken: number;
}

const STATUS_OPTIONS: SearchField[] = [
  {
    type: "select",
    name: "status",
    label: "状态",
    placeholder: "全部状态",
    options: [
      { label: "全部", value: "all" },
      { label: "检测到漂移", value: "drifted" },
      { label: "无漂移", value: "normal" },
      { label: "被节流", value: "throttled" },
    ],
  },
  {
    type: "input",
    name: "startTime",
    label: "起始时间",
    placeholder: "YYYY-MM-DD HH:mm:ss",
  },
  {
    type: "input",
    name: "endTime",
    label: "结束时间",
    placeholder: "YYYY-MM-DD HH:mm:ss",
  },
];

/** 时间格式化：dayjs 容错，空值显示 "-" */
function formatLogTime(value?: string): string {
  if (!value) return "-";
  const d = dayjs(value);
  return d.isValid() ? d.format("YYYY-MM-DD HH:mm:ss") : value;
}

const TriggerLogTable: FC<TriggerLogTableProps> = ({ refreshToken }) => {
  // 稳定引用：PageTable 的 request 变化会让 useTableManager.loadData 重建，
  // 进而可能触发重复加载；用 useCallback 空依赖锁定模块级导入的函数引用。
  const request = useCallback(async (params: Record<string, unknown>) => {
    const p = params as Partial<ADDriftTriggerLogParams> & Record<string, unknown>;
    // 显式构造业务入参，丢弃 PageTable 注入的 _reload / 排序等辅助字段
    const payload: ADDriftTriggerLogParams = {
      current: typeof p.current === "number" ? p.current : 1, // current 从 1
      pageSize: typeof p.pageSize === "number" ? p.pageSize : 10,
    };
    if (typeof p.startTime === "string" && p.startTime) payload.startTime = p.startTime;
    if (typeof p.endTime === "string" && p.endTime) payload.endTime = p.endTime;
    if (
      p.status === "all" ||
      p.status === "drifted" ||
      p.status === "normal" ||
      p.status === "throttled"
    ) {
      payload.status = p.status;
    }
    const page = await listTriggerLogs(payload);
    return { list: page?.list ?? [], total: page?.total ?? 0 };
  }, []);

  const columns = useMemo<ColumnsType<ADDriftTriggerLog>>(
    () => [
      {
        title: "触发时间",
        dataIndex: "createdAt",
        key: "createdAt",
        width: 190,
        render: (v: string) => formatLogTime(v),
      },
      {
        title: "来源",
        dataIndex: "source",
        key: "source",
        width: 100,
        render: (v?: string) => (v === "manual" ? "手动" : v === "cron" ? "定时" : (v ?? "-")),
      },
      {
        title: "检测结果",
        dataIndex: "drifted",
        key: "drifted",
        width: 120,
        render: (drifted: boolean, record: ADDriftTriggerLog) =>
          record.throttled ? (
            <Tag color="default">被节流</Tag>
          ) : drifted ? (
            <Tag color="error">检测到漂移</Tag>
          ) : (
            <Tag color="success">无漂移</Tag>
          ),
      },
      {
        title: "成员数 / 组数",
        key: "counts",
        width: 140,
        render: (_, r: ADDriftTriggerLog) => `${r.memberCount ?? "-"} / ${r.distinctGroups ?? "-"}`,
      },
      {
        title: "漂移量",
        dataIndex: "delta",
        key: "delta",
        render: (v?: string) => v ?? "-",
      },
      {
        title: "操作人",
        dataIndex: "operator",
        key: "operator",
        width: 120,
        render: (v?: string) => v ?? "-",
      },
      {
        title: "结果",
        dataIndex: "result",
        key: "result",
        ellipsis: true,
        render: (v?: string) => v ?? "-",
      },
    ],
    []
  );

  const searchFields = useMemo<SearchField[]>(() => STATUS_OPTIONS, []);

  // 刷新信号：useMemo 稳定对象引用，仅当 refreshToken（primitive）变化时重建，
  // 触发 PageTable.applyFilters 回第 1 页重载。禁止每次 render 新建对象字面量。
  const extraFilters = useMemo<Record<string, unknown>>(
    () => ({ _reload: refreshToken }),
    [refreshToken]
  );

  const emptyAction = useMemo(
    () => (
      <EmptyStateWithAction
        title="暂无触发记录"
        description="执行一次「立即检测」或等待定时任务运行后，记录将出现在这里"
      />
    ),
    []
  );

  // PageTable 的 rowKey 契约是 GetRowKey<T>（函数），不是字段名字符串。
  const rowKey = useCallback((record: ADDriftTriggerLog) => record.id, []);

  return (
    <PageTable<ADDriftTriggerLog>
      request={request}
      columns={columns}
      rowKey={rowKey}
      pageSize={10}
      searchFields={searchFields}
      extraFilters={extraFilters}
      emptyAction={emptyAction}
    />
  );
};

export default TriggerLogTable;

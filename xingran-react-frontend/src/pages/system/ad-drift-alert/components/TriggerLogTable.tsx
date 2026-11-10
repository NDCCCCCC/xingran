/**
 * 同步漂移告警 — 触发记录表（Phase 144 ALRT-03，B-1 / v1.56 契约对齐）
 *
 * 用既有 PageTable 契约组件（内建 loading / 空态 / 错误态 三态 + 分页 showTotal）。
 * 空态接 EmptyStateWithAction；错误态由 PageTable 内建 ErrorAlertWithRetry 统一出口。
 *
 * B-1 / v1.56：
 *   - 与后端 DriftAlertTriggerLog 逐字段对齐（此前按虚构业务字段渲染，全部列落空）；
 *   - 删除后端不存在的「时间/状态」假筛选（ListLogs 只接 current/pageSize）；
 *   - 新增「已读」列 + 「标记已读」行操作，复用 sys_notice_read 按用户已读；
 *   - 标记成功后通过 onMarkRead 回调复用父组件 refreshToken 通道刷新，零新增 state。
 *
 * 依赖稳定性（CLAUDE.md 防无限循环）：
 *   - request / columns / rowActions / extraFilters 全部 useCallback / useMemo 稳定；
 *   - 禁止把对象字面量直接写进 useEffect 依赖；
 *   - extraFilters 只承载 refreshToken（primitive 经 useMemo 包成稳定对象引用），
 *     refreshToken 变化 → PageTable.applyFilters 回第 1 页重载。
 */
import { useCallback, useMemo } from "react";
import type { FC } from "react";
import { Tag } from "antd";
import dayjs from "dayjs";
import type { ColumnsType } from "antd/es/table";
import PageTable from "@/components/shared/PageTable";
import EmptyStateWithAction from "@/components/shared/EmptyStateWithAction";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import {
  listTriggerLogs,
  markTriggerLogRead,
  type ADDriftTriggerLog,
  type ADDriftTriggerLogParams,
} from "@/lib/adDriftAlertApi";

export interface TriggerLogTableProps {
  /** 刷新信号：index.tsx 在「立即检测成功」/「收到 WS 新告警」/「标记已读」时 +1 */
  refreshToken: number;
  /** 标记已读成功回调：复用父组件 bumpLogs（refreshToken 通道） */
  onMarkRead?: () => void;
}

/** 时间格式化：dayjs 容错，空值显示 "-" */
function formatLogTime(value?: string): string {
  if (!value) return "-";
  const d = dayjs(value);
  return d.isValid() ? d.format("YYYY-MM-DD HH:mm:ss") : value;
}

/** 解析后端 jsonResult（诊断 JSON，损坏时软降级为空对象） */
function parseDriftPayload(raw?: string): {
  memberCount?: number;
  distinctGroups?: number;
  memberDelta?: number;
  groupDelta?: number;
} {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw) as {
      memberCount?: number;
      distinctGroups?: number;
      memberDelta?: number;
      groupDelta?: number;
    };
    return parsed ?? {};
  } catch {
    return {};
  }
}

const TriggerLogTable: FC<TriggerLogTableProps> = ({ refreshToken, onMarkRead }) => {
  // 稳定引用：PageTable 的 request 变化会让 useTableManager.loadData 重建，
  // 进而可能触发重复加载；用 useCallback 空依赖锁定模块级导入的函数引用。
  const request = useCallback(async (params: Record<string, unknown>) => {
    // 后端只接 current/pageSize；显式构造业务入参，丢弃 PageTable 注入的
    // _reload / 排序等辅助字段，也不发送后端不支持的假筛选参数。
    const payload: ADDriftTriggerLogParams = {
      current: typeof params.current === "number" ? params.current : 1,
      pageSize: typeof params.pageSize === "number" ? params.pageSize : 10,
    };
    const page = await listTriggerLogs(payload);
    return { list: page?.list ?? [], total: page?.total ?? 0 };
  }, []);

  const handleMarkRead = useCallback(
    async (logId: string) => {
      try {
        await markTriggerLogRead(logId);
        handleSuccess("标记已读");
        // 复用父组件 refreshToken 通道：extraFilters 引用变化 → PageTable 回第 1 页重载
        onMarkRead?.();
      } catch (error) {
        // 唯一错误出口
        handleApiError(error, "标记已读");
      }
    },
    [onMarkRead]
  );

  const rowActions = useCallback(
    (record: ADDriftTriggerLog) => [
      {
        key: "markRead",
        label: record.isRead ? "已读" : "标记已读",
        // 后端按 logId 反查 JsonResult.noticeId；无关联通知 / 已读时禁用，避免 400
        disabled: record.isRead || !record.noticeId,
        permission: PERMISSION_CODES.adDriftAlert.list,
        onClick: () => void handleMarkRead(record.id),
      },
    ],
    [handleMarkRead]
  );

  const columns = useMemo<ColumnsType<ADDriftTriggerLog>>(
    () => [
      {
        title: "触发时间",
        dataIndex: "operTime",
        key: "operTime",
        width: 190,
        render: (v: string) => formatLogTime(v),
      },
      {
        title: "标题",
        dataIndex: "title",
        key: "title",
        ellipsis: true,
        render: (v?: string) => v ?? "-",
      },
      {
        title: "成员数 / 组数",
        key: "counts",
        width: 150,
        render: (_: unknown, r: ADDriftTriggerLog) => {
          const p = parseDriftPayload(r.jsonResult);
          return `${p.memberCount ?? "-"} / ${p.distinctGroups ?? "-"}`;
        },
      },
      {
        title: "漂移量（成员 / 组）",
        key: "delta",
        width: 150,
        render: (_: unknown, r: ADDriftTriggerLog) => {
          const p = parseDriftPayload(r.jsonResult);
          return `${p.memberDelta ?? "-"} / ${p.groupDelta ?? "-"}`;
        },
      },
      {
        title: "日志状态",
        dataIndex: "status",
        key: "status",
        width: 100,
        render: (v: number) =>
          v === 0 ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>,
      },
      {
        title: "已读",
        dataIndex: "isRead",
        key: "isRead",
        width: 90,
        render: (v: boolean, r: ADDriftTriggerLog) =>
          !r.noticeId ? (
            <Tag color="default">无关联通知</Tag>
          ) : v ? (
            <Tag color="default">已读</Tag>
          ) : (
            <Tag color="blue">未读</Tag>
          ),
      },
    ],
    []
  );

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
      rowActions={rowActions}
      pageSize={10}
      extraFilters={extraFilters}
      emptyAction={emptyAction}
    />
  );
};

export default TriggerLogTable;

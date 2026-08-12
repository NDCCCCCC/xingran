/**
 * ReconciliationTimeline — Phase 45 R4 / D-A3-01/02
 *
 * 抽屉"历史变更"Tab 内容:已解决冲突记录 Timeline。
 *
 * 锁定决策 (UI-SPEC):
 *   - antd Timeline mode="left"
 *   - 每项:冲突类型色点 + 检出时间 + 解决时间 + 解决人 + 说明
 *   - raw_snapshot 不展开(backend-only audit data, D-A3-02)
 *   - 空态:"该资产暂无已解决的冲突记录。"
 *   - 加载态:Skeleton
 *
 * 现阶段 Timeline 数据由父组件(Drawer)通过 props 注入(本文件为纯展示组件)。
 * 实际数据流:由 Plan 02 接入 sys_data_reconciliation WHERE resolved_at IS NOT NULL
 *   ORDER BY resolved_at DESC 的实时查询,Plan 01 仅固定组件契约。
 */
import React from "react";
import { Timeline, Skeleton, Empty, Tag } from "antd";
import dayjs from "dayjs";
import { HealthBadge } from "./HealthBadge";

export interface TimelineRecord {
  id: string;
  conflictType: string;
  detectedAt: string;
  resolvedAt: string;
  resolvedByUsername: string;
  resolutionNote?: string;
}

export interface ReconciliationTimelineProps {
  records: TimelineRecord[];
  loading: boolean;
}

const CONFLICT_TYPE_LABELS: Record<string, string> = {
  A: "正常",
  B: "物理有/责任人无",
  C: "物理有/责任人不一致",
  D: "物理无/责任人有",
  E: "三路均无",
  F: "AD 单独不一致",
};

const TimelineInner: React.FC<ReconciliationTimelineProps> = ({ records, loading }) => {
  if (loading) {
    return <Skeleton active paragraph={{ rows: 3 }} />;
  }

  if (!records || records.length === 0) {
    return <Empty description="该资产暂无已解决的冲突记录。" />;
  }

  return (
    <Timeline mode="left">
      {records.map((r) => (
        <Timeline.Item
          key={r.id}
          dot={<HealthBadge assetId={r.id} conflictType={r.conflictType} onClick={() => {}} />}
        >
          <div>
            <Tag color="default">{CONFLICT_TYPE_LABELS[r.conflictType] ?? r.conflictType}</Tag>
            <span style={{ marginLeft: 8 }}>
              检出: {dayjs(r.detectedAt).format("YYYY-MM-DD HH:mm")}
            </span>
          </div>
          <div style={{ marginTop: 4 }}>
            由 {r.resolvedByUsername} 于 {dayjs(r.resolvedAt).format("YYYY-MM-DD HH:mm")} 解决
          </div>
          <div style={{ marginTop: 4, fontSize: 12, color: "#6b7280" }}>
            说明: {r.resolutionNote || "(无)"}
          </div>
        </Timeline.Item>
      ))}
    </Timeline>
  );
};

export const ReconciliationTimeline = React.memo(TimelineInner);
ReconciliationTimeline.displayName = "ReconciliationTimeline";

/**
 * ReconciliationDrawer — Phase 45 R4 / D-A1-02
 *
 * 冲突详情抽屉(3 Tabs):冲突摘要 / 历史变更 / 例外规则。
 *
 * 锁定决策 (UI-SPEC):
 *   - 780px 宽
 *   - 动态 title:资产行点击 → "资产对账详情 — {assetCode}";工位 expand → "工位对账详情"
 *   - extra 区域含"申请例外"按钮
 *   - Tabs activeKey + onChange tabBarGutter=24
 *   - useReconciliationVisibility() === false → render null
 *
 * 数据源:
 *   - Tab 1:useAssetHealth (从父级 useWorkstationHealth cache 切片,无 N+1)
 *   - Tab 2:ReconciliationTimeline(Plan 02 接入)
 *   - Tab 3:ExceptionMatchList(Plan 02 接入)
 */
import React, { useState } from "react";
import { Drawer, Tabs, Tag, Descriptions, Button, Skeleton, Empty } from "antd";
import { ReconciliationTimeline, type TimelineRecord } from "./ReconciliationTimeline";
import { ExceptionMatchList, type ExceptionRuleItem } from "./ExceptionMatchList";
import { useAssetHealth } from "./hooks/useAssetHealth";
import { useReconciliationVisibility } from "./hooks/useReconciliationVisibility";

export type DrawerTabKey = "summary" | "timeline" | "exception";

export interface ReconciliationDrawerProps {
  open: boolean;
  onClose: () => void;
  selectedAssetId: string | null;
  workstationId: string | null;
  assetCode?: string;
  workstationName?: string;
  activeTab: DrawerTabKey;
  onTabChange: (key: DrawerTabKey) => void;
  onApplyException: () => void;
}

const SEVERITY_COLOR: Record<string, string> = {
  low: "default",
  medium: "warning",
  high: "error",
  critical: "error",
};

const DrawerInner: React.FC<ReconciliationDrawerProps> = ({
  open,
  onClose,
  selectedAssetId,
  workstationId,
  assetCode,
  workstationName,
  activeTab,
  onTabChange,
  onApplyException,
}) => {
  const visible = useReconciliationVisibility();
  const [timeline] = useState<TimelineRecord[]>([]);
  const [timelineLoading] = useState(false);
  const [rules] = useState<ExceptionRuleItem[]>([]);
  const [rulesLoading] = useState(false);

  const asset = useAssetHealth(selectedAssetId, workstationId ?? "");

  // Plan 01:timeline / rules 实际数据由 Plan 02 接入(实时查询 sys_data_reconciliation /
  //   sys_reconciliation_exception)。当前 local state 保留供 Plan 02 填充。

  if (!visible) {
    return null;
  }

  const title = assetCode
    ? `资产对账详情 — ${assetCode}`
    : workstationName
      ? `工位对账详情 — ${workstationName}`
      : "对账详情";

  const summaryTab = (
    <div>
      {asset ? (
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="资产编号">{asset.assetCode}</Descriptions.Item>
          <Descriptions.Item label="冲突类型">
            {asset.conflictType ? (
              <Tag color="default">{asset.conflictType}</Tag>
            ) : (
              <Tag color="success">健康</Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="严重程度">
            {asset.severity ? (
              <Tag color={SEVERITY_COLOR[asset.severity] ?? "default"}>{asset.severity}</Tag>
            ) : (
              "-"
            )}
          </Descriptions.Item>
          <Descriptions.Item label="IP">{asset.ip ?? "-"}</Descriptions.Item>
          <Descriptions.Item label="置信度">
            {asset.confidenceScore?.toFixed(2) ?? "-"}
          </Descriptions.Item>
          {asset.appliedActions && asset.appliedActions.length > 0 && (
            <Descriptions.Item label="已应用 Actions">
              {asset.appliedActions.map((a) => (
                <Tag key={a}>{a}</Tag>
              ))}
            </Descriptions.Item>
          )}
        </Descriptions>
      ) : selectedAssetId ? (
        <Skeleton active />
      ) : (
        <Empty description="请选择一个资产查看详情" />
      )}
    </div>
  );

  const timelineTab = (
    <ReconciliationTimeline records={timeline} loading={timelineLoading} />
  );

  const exceptionTab = (
    <ExceptionMatchList
      rules={rules}
      loading={rulesLoading}
      onCreateRule={onApplyException}
    />
  );

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      width={780}
      extra={
        <Button type="primary" onClick={onApplyException}>
          申请例外
        </Button>
      }
      destroyOnHidden
    >
      <Tabs
        activeKey={activeTab}
        onChange={(k) => onTabChange(k as DrawerTabKey)}
        tabBarGutter={24}
        items={[
          { key: "summary", label: "冲突摘要", children: summaryTab },
          { key: "timeline", label: "历史变更", children: timelineTab },
          { key: "exception", label: "例外规则", children: exceptionTab },
        ]}
      />
    </Drawer>
  );
};

export const ReconciliationDrawer = React.memo(DrawerInner);
ReconciliationDrawer.displayName = "ReconciliationDrawer";

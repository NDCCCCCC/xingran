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
 *
 * Phase 45 R4 / Plan 02 (SC9): 申请例外按钮携带 workstationId + assetIp + conflictType
 * 三个 query 参数,跳转后由 /asset/reconciliation/exception-rules/new 页面 useSearchParams 解析
 */
import React, { useCallback, useState } from "react";
import { Drawer, Tabs, Tag, Descriptions, Button, Skeleton, Empty, message } from "antd";
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
  /**
   * 抽屉顶部"申请例外"按钮回调(由父级 page 注入,默认走本组件内的内联实现)。
   * Plan 02 改动: 该参数变为可选;若未提供,本组件自动 navigate 到
   * /asset/reconciliation/exception-rules/new 携带 workstationId + assetIp + conflictType
   * query 参数。
   */
  onApplyException?: () => void;
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

  // 🆕 Plan 02 / SC9: 申请例外按钮 — 携带 workstationId + assetIp + conflictType
  // 三参数(优先用父级 onApplyException 回调,否则用本组件内联 navigate)
  // 注意:useCallback 必须在所有 early return 之前(react-hooks/rules-of-hooks)
  // deps 使用 asset 对象引用,React compiler 会基于 asset 引用变化自动推断
  const handleApplyException = useCallback(() => {
    if (onApplyException) {
      onApplyException();
      return;
    }
    const params = new URLSearchParams();
    if (workstationId) {
      params.set("workstationId", workstationId);
    }
    if (selectedAssetId) {
      params.set("assetId", selectedAssetId);
    }
    if (asset?.ip) {
      params.set("assetIp", asset.ip);
    }
    if (asset?.conflictType) {
      params.set("conflictType", asset.conflictType);
    }
    const url = `/asset/reconciliation/exception-rules/new?${params.toString()}`;
    window.open(url, "_blank");
    message.success("已跳转到例外规则新建页,IP 与冲突类型已预填。");
  }, [onApplyException, workstationId, selectedAssetId, asset]);

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

  // 🆕 Plan 02: 例外规则 Tab 传入 assetIp + conflictType 让空态"去创建"按钮携带预填
  const exceptionTab = (
    <ExceptionMatchList
      rules={rules}
      loading={rulesLoading}
      assetIp={asset?.ip}
      conflictType={asset?.conflictType}
      onCreateRule={onApplyException}
    />
  );

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      size="large"
      extra={
        <Button type="primary" onClick={handleApplyException}>
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

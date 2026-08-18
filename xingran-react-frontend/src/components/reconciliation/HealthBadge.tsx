/**
 * HealthBadge — Phase 45 R4 / D-A2-04
 *
 * 行内对账健康徽标(8px 圆点 + Tooltip)。
 *
 * 锁定决策 (UI-SPEC):
 *   - 6 类型 A-F 不同颜色(A=绿/list_class 映射)
 *   - 8px diameter 圆点(24px touch target via cell 32px row)
 *   - Tooltip mouseEnterDelay=1s
 *   - 点击 → onClick(assetId, conflictType) 打开 drawer
 *   - useReconciliationVisibility() === false → render "-" 占位(D-A1-03)
 *   - a11y:有冲突时 role=button + tabIndex=0;无冲突 role=img
 */
import React, { useMemo } from "react";
import { Tooltip } from "antd";
import { useDict } from "@/hooks/useDict";
import { useReconciliationVisibility } from "./hooks/useReconciliationVisibility";

export interface HealthBadgeProps {
  assetId: string;
  conflictType: string | null;
  onClick: (assetId: string, conflictType: string) => void;
}

const CONFLICT_TYPE_TOOLTIPS: Record<string, string> = {
  A: "物理有/责任人有且一致",
  B: "物理有/责任人无",
  C: "物理有/责任人不一致(高危)",
  D: "物理无/责任人有",
  E: "三路数据均未检测到该资产(疑似幽灵资产)",
  F: "仅 AD managed_by 与系统登记不一致(AD 已知不可靠)",
};

const LIST_CLASS_TO_HEX: Record<string, string> = {
  success: "#2d8949",
  warning: "#b07a20",
  error: "#ba3630",
  default: "#d4d4d8",
  processing: "#156031",
};

const HealthBadgeInner: React.FC<HealthBadgeProps> = ({ assetId, conflictType, onClick }) => {
  const visible = useReconciliationVisibility();
  const { data: conflictTypeDict } = useDict("asset_reconciliation_conflict_type");

  const dotColor = useMemo(() => {
    if (!conflictType) return LIST_CLASS_TO_HEX.success; // 健康(A/null)→ 绿
    const entry = conflictTypeDict?.find((d) => d.dictValue === conflictType);
    const listClass = entry?.listClass ?? "default";
    return LIST_CLASS_TO_HEX[listClass] ?? LIST_CLASS_TO_HEX.default;
  }, [conflictType, conflictTypeDict]);

  // 静默降级(D-A1-03)
  if (!visible) {
    return <>-</>;
  }

  const hasConflict = Boolean(conflictType);
  const tooltipText = conflictType ? `${CONFLICT_TYPE_TOOLTIPS[conflictType] ?? ""}` : "健康";

  const handleClick = () => {
    if (conflictType) onClick(assetId, conflictType);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && conflictType) {
      onClick(assetId, conflictType);
    }
  };

  const dot = (
    <span
      role={hasConflict ? "button" : "img"}
      tabIndex={hasConflict ? 0 : -1}
      aria-label={tooltipText}
      onClick={hasConflict ? handleClick : undefined}
      onKeyDown={hasConflict ? handleKeyDown : undefined}
      style={{
        display: "inline-block",
        width: 8,
        height: 8,
        borderRadius: "50%",
        backgroundColor: dotColor,
        cursor: hasConflict ? "pointer" : "default",
        verticalAlign: "middle",
      }}
    />
  );

  if (!hasConflict) {
    return dot;
  }

  return (
    <Tooltip title={tooltipText} mouseEnterDelay={1}>
      {dot}
    </Tooltip>
  );
};

export const HealthBadge = React.memo(HealthBadgeInner);
HealthBadge.displayName = "HealthBadge";

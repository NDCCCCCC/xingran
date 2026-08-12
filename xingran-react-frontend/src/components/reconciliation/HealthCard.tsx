/**
 * HealthCard — Phase 45 R4 / D-A1-01
 *
 * 工位对账健康度(单行紧凑版,/gsd-fast 2026-06-30)。
 * 放置于工位 expand 展开区顶部(参 UI-SPEC "Anchor" + D-A1-01)。
 *
 * 显示(单行,无 Card 标题/趋势 chart/5 KPI grid):
 *   [score/100(颜色按分数段)] 正常 N · 漂移 N · 冲突 N [· 无数据 N] [· 例外 N] [申请例外 →]
 *
 * 得分颜色:≥80 绿 / 60-79 黄 / <60 红
 * useReconciliationVisibility() === false → render null
 * useEffect/useQuery deps 仅 primitive workstationId
 */
import React from "react";
import { Empty, Skeleton, Result, Button } from "antd";
import { useWorkstationHealth } from "./hooks/useWorkstationHealth";
import { useReconciliationVisibility } from "./hooks/useReconciliationVisibility";

export interface HealthCardProps {
  workstationId: string;
  onApplyException?: () => void;
}

const SCORE_BAND_COLOR = (score: number): string => {
  if (score >= 80) return "#22c55e";
  if (score >= 60) return "#f59e0b";
  return "#ef4444";
};

const HealthCardInner: React.FC<HealthCardProps> = ({ workstationId, onApplyException }) => {
  const visible = useReconciliationVisibility();
  const { data, isLoading, isError, refetch } = useWorkstationHealth(workstationId);

  
  // 静默降级(D-A1-03)
  if (!visible) {
    return null;
  }

  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 2 }} />;
  }

  if (isError) {
    return (
      <Result
        status="error"
        title="健康度加载失败"
        subTitle="请稍后重试或联系运维"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  if (!data) {
    return <Empty description="暂无数据" />;
  }

  // 空态(/gsd-fast: 同步压成单行内联,去掉 Card 标题/Empty 图标撑出的空白)
  if (data.healthScore.total === 0) {
    return (
      <div style={{ padding: "4px 8px", marginBottom: 8, fontSize: 12, color: "#999" }}>
        对账健康度:该工位暂无关联资产。
      </div>
    );
  }

  const { healthScore } = data;
  const scoreColor = SCORE_BAND_COLOR(healthScore.score);

  // /gsd-fast (2026-06-30): 单行紧凑版 — 彩色 score · 内联 KPI 摘要 · 申请例外。
  // 原 Card 标题/score 大数字/trend chart/5 KPI grid 占位过高,expand 后行高跳变。
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "4px 8px",
        marginBottom: 8,
        fontSize: 12,
        color: "#666",
      }}
    >
      <span style={{ color: scoreColor, fontWeight: 700, fontSize: 16 }}>
        {healthScore.score}
        <span style={{ fontSize: 11, color: "#999", fontWeight: 400 }}> /100</span>
      </span>
      <span>
        正常 {healthScore.normal} · 漂移 {healthScore.drift} · 冲突 {healthScore.conflict}
        {healthScore.noData > 0 ? ` · 无数据 ${healthScore.noData}` : ""}
        {healthScore.exceptionHit > 0 ? ` · 例外 ${healthScore.exceptionHit}` : ""}
      </span>
      {onApplyException && (
        <Button
          type="link"
          size="small"
          style={{ marginLeft: "auto", padding: 0, height: "auto" }}
          onClick={onApplyException}
        >
          申请例外
        </Button>
      )}
    </div>
  );
};

export const HealthCard = React.memo(HealthCardInner);
HealthCard.displayName = "HealthCard";

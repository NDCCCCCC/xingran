/**
 * HealthCard — Phase 45 R4 / D-A1-01
 *
 * 工位对账健康度卡片(5 KPI + 趋势 mini chart + 得分)。
 * 放置于工位 expand 展开区顶部(参 UI-SPEC "Anchor" + D-A1-01)。
 *
 * 锁定决策 (UI-SPEC):
 *   - antd Card title="对账健康度" + "本周" tag + score + 两个链接
 *   - 5 KPI grid (5 Col span=4)
 *   - 趋势 mini chart:echarts-for-react,height=56,单 line series,无坐标轴
 *   - 得分颜色:≥80 绿 / 60-79 黄 / <60 红
 *   - useReconciliationVisibility() === false → render null
 *   - useEffect/useQuery deps 仅 primitive workstationId
 */
import React, { useMemo } from "react";
import { Card, Col, Row, Statistic, Tag, Skeleton, Empty, Result, Button } from "antd";
import ReactECharts from "echarts-for-react";
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

  const trendOption = useMemo(() => {
    const points = (data?.healthScore?.trend ?? []).map((t: { date: string; openCount: number }) => [t.date, t.openCount]);
    return {
      grid: { left: 0, right: 0, top: 4, bottom: 4 },
      xAxis: { type: "category", show: false, data: points.map((p: (string | number)[]) => p[0]) },
      yAxis: { type: "value", show: false },
      tooltip: { show: false },
      series: [
        {
          type: "line",
          data: points.map((p: (string | number)[]) => p[1]),
          smooth: true,
          symbol: "none",
          lineStyle: { color: data ? SCORE_BAND_COLOR(data.healthScore.score) : "#22c55e", width: 2 },
          areaStyle: { color: data ? SCORE_BAND_COLOR(data.healthScore.score) : "#22c55e", opacity: 0.1 },
        },
      ],
    };
  }, [data]);

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

  // 空态
  if (data.healthScore.total === 0) {
    return (
      <Card
        size="small"
        title="对账健康度"
        extra={<Tag color="blue">本周</Tag>}
        style={{ marginBottom: 16 }}
      >
        <Empty description="该工位暂无关联资产。" />
      </Card>
    );
  }

  const { healthScore } = data;
  const scoreColor = SCORE_BAND_COLOR(healthScore.score);

  return (
    <Card
      size="small"
      title="对账健康度"
      extra={
        <Tag color="blue" style={{ marginRight: 8 }}>
          本周
        </Tag>
      }
      style={{ marginBottom: 16 }}
    >
      <Row gutter={16} align="middle" style={{ marginBottom: 12 }}>
        <Col flex="80px">
          <Statistic
            value={healthScore.score}
            suffix="/100"
            valueStyle={{ color: scoreColor, fontSize: 28, fontWeight: 700 }}
          />
        </Col>
        <Col flex="auto">
          <ReactECharts
            option={trendOption}
            style={{ height: 56, width: "100%" }}
            opts={{ renderer: "svg" }}
            notMerge
            lazyUpdate
          />
        </Col>
        {onApplyException && (
          <Col flex="100px" style={{ textAlign: "right" }}>
            <Button type="link" onClick={onApplyException}>
              申请例外
            </Button>
          </Col>
        )}
      </Row>
      <Row gutter={16}>
        <Col span={4} xs={12} sm={8} md={6} lg={4}>
          <Statistic title="正常" value={healthScore.normal} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
        </Col>
        <Col span={4} xs={12} sm={8} md={6} lg={4}>
          <Statistic title="漂移" value={healthScore.drift} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
        </Col>
        <Col span={4} xs={12} sm={8} md={6} lg={4}>
          <Statistic title="冲突" value={healthScore.conflict} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
        </Col>
        <Col span={4} xs={12} sm={8} md={6} lg={4}>
          <Statistic title="无数据" value={healthScore.noData} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
        </Col>
        <Col span={4} xs={12} sm={8} md={6} lg={4}>
          <Statistic title="例外命中" value={healthScore.exceptionHit} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
        </Col>
      </Row>
    </Card>
  );
};

export const HealthCard = React.memo(HealthCardInner);
HealthCard.displayName = "HealthCard";

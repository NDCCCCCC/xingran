/**
 * 楼层统计卡片组件
 *
 * UI 统一整改：内部渲染改为全站统一增强版 <StatCards> 范本，
 * 替代旧 antd <Row><Col><Card><Statistic>> 结构。
 * 对外 props（statistics / show）与文案保持不变。
 */
import type { FC } from "react";
import StatCards from "@/components/shared/StatCards";

interface FloorStatistics {
  total: number;
  active: number;
  inactive: number;
}

interface FloorStatisticsCardsProps {
  statistics: FloorStatistics;
  show: boolean;
}

export const FloorStatisticsCards: FC<FloorStatisticsCardsProps> = ({ statistics, show }) => {
  return (
    <StatCards
      show={show}
      items={[
        { label: "总楼层数", value: statistics.total },
        { label: "正常楼层", value: statistics.active, tone: "green" },
        { label: "停用楼层", value: statistics.inactive, tone: "danger" },
      ]}
    />
  );
};

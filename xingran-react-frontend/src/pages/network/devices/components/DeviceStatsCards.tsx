/**
 * Phase 67-T3: 设备统计卡片（原 index.tsx :684-727 纯搬迁）
 * 全站 UI 统一整改：改用共享 StatCards 范本（.stat-cards/.stat-card）。
 * 保留原条件渲染语义：statistics.total > 10 才展示（存量行为，不扩不缩）。
 */
import { StatCards } from "@/components/shared";

export interface DeviceStats {
  total: number;
  online: number;
  offline: number;
  unknown: number;
}

interface DeviceStatsCardsProps {
  statistics: DeviceStats;
}

const DeviceStatsCards: React.FC<DeviceStatsCardsProps> = ({ statistics }) => (
  <StatCards
    show={statistics.total > 10}
    items={[
      { label: "设备总数", value: statistics.total, tone: "default" },
      { label: "在线设备", value: statistics.online, tone: "green" },
      { label: "离线设备", value: statistics.offline, tone: "danger" },
      { label: "未知状态", value: statistics.unknown, tone: "gray" },
    ]}
  />
);

export default DeviceStatsCards;

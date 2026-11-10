import type { NoticeStatistics } from "../hooks/useNoticeStatistics";
import { StatCards } from "@/components/shared";

interface NoticeStatisticsCardProps {
  statistics: NoticeStatistics;
}

/**
 * 通知统计卡片组件
 * 显示总公告数、已发布、草稿/撤回、定时发布数量；统一走 <StatCards>
 * 只在总数大于 10 时显示（show 条件）
 */
export const NoticeStatisticsCard: React.FC<NoticeStatisticsCardProps> = ({ statistics }) => {
  return (
    <StatCards
      show={statistics.total > 10}
      style={{ marginBottom: 16 }}
      items={[
        { label: "总公告数", value: statistics.total },
        { label: "已发布", value: statistics.published, tone: "green" },
        { label: "草稿/撤回", value: statistics.draft, tone: "gold" },
        { label: "定时发布", value: statistics.scheduled },
      ]}
    />
  );
};

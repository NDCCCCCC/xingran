import { Drawer, Spin } from "antd";
import type { NoticeStatistics as NoticeStatisticsType } from "@/types/notice";
import { StatCards } from "@/components/shared";

interface StatisticsDrawerProps {
  visible: boolean;
  onClose: () => void;
  loading: boolean;
  statistics: NoticeStatisticsType | null;
}

/**
 * 统计详情抽屉组件
 * 显示单个通知的详细阅读统计信息；统计卡统一走 <StatCards>，保持 Drawer 结构
 */
export const StatisticsDrawer: React.FC<StatisticsDrawerProps> = ({
  visible,
  onClose,
  loading,
  statistics,
}) => {
  return (
    <Drawer title="阅读统计" placement="right" size="default" open={visible} onClose={onClose}>
      {loading ? (
        <div className="flex justify-center py-8">
          <Spin />
        </div>
      ) : statistics ? (
        <StatCards
          items={[
            { label: "目标用户数", value: statistics.totalTargets },
            { label: "已读数", value: `${statistics.readCount} 人`, tone: "green" },
            { label: "未读数", value: `${statistics.unreadCount} 人`, tone: "gray" },
            {
              label: "阅读率",
              value: `${Number(statistics.readRate).toFixed(1)}%`,
              tone: statistics.readRate >= 50 ? "green" : "danger",
            },
          ]}
        />
      ) : (
        <div className="text-center text-gray-500 py-8">暂无统计数据</div>
      )}
    </Drawer>
  );
};

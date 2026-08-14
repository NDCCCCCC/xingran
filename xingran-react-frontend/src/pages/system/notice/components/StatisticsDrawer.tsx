import { Drawer, Card, Statistic, Spin } from "antd";
import type { NoticeStatistics as NoticeStatisticsType } from "@/types/notice";

interface StatisticsDrawerProps {
  visible: boolean;
  onClose: () => void;
  loading: boolean;
  statistics: NoticeStatisticsType | null;
}

/**
 * 统计详情抽屉组件
 * 显示单个通知的详细阅读统计信息
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
        <div className="space-y-6">
          <Card>
            <Statistic title="目标用户数" value={statistics.totalTargets} />
          </Card>
          <Card>
            <Statistic title="已读数" value={statistics.readCount} suffix="人" />
          </Card>
          <Card>
            <Statistic title="未读数" value={statistics.unreadCount} suffix="人" />
          </Card>
          <Card>
            <Statistic
              title="阅读率"
              value={statistics.readRate}
              precision={1}
              suffix="%"
              styles={{
                content: {
                  color:
                    statistics.readRate >= 50
                      ? "var(--theme-success, #3f8600)"
                      : "var(--theme-error, #cf1322)",
                },
              }}
            />
          </Card>
        </div>
      ) : (
        <div className="text-center text-gray-500 py-8">暂无统计数据</div>
      )}
    </Drawer>
  );
};

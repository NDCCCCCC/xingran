import { Card, Row, Col, Statistic } from "antd";
import { CheckCircleOutlined, StopOutlined, SearchOutlined } from "@ant-design/icons";
import type { NoticeStatistics } from "../hooks/useNoticeStatistics";

interface NoticeStatisticsCardProps {
  statistics: NoticeStatistics;
}

/**
 * 通知统计卡片组件
 * 显示总公告数、已发布、草稿/撤回、定时发布数量
 */
export const NoticeStatisticsCard: React.FC<NoticeStatisticsCardProps> = ({ statistics }) => {
  // 只在总数大于10时显示
  if (statistics.total <= 10) {
    return null;
  }

  return (
    <Row gutter={16} style={{ marginBottom: 16 }}>
      <Col span={6}>
        <Card>
          <Statistic
            title="总公告数"
            value={statistics.total}
            prefix={<CheckCircleOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="已发布"
            value={statistics.published}
            styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
            prefix={<CheckCircleOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="草稿/撤回"
            value={statistics.draft}
            styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
            prefix={<StopOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="定时发布"
            value={statistics.scheduled}
            styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
            prefix={<SearchOutlined />}
          />
        </Card>
      </Col>
    </Row>
  );
};

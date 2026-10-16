/**
 * 楼层统计卡片组件
 */

import type { FC } from "react";
import { Row, Col, Card, Statistic } from "antd";
import { CheckCircleOutlined, StopOutlined } from "@ant-design/icons";

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
  if (!show) {
    return null;
  }

  return (
    <Row gutter={16} style={{ marginBottom: 16 }}>
      <Col span={8}>
        <Card>
          <Statistic title="总楼层数" value={statistics.total} prefix={<CheckCircleOutlined />} />
        </Card>
      </Col>
      <Col span={8}>
        <Card>
          <Statistic
            title="正常楼层"
            value={statistics.active}
            styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
            prefix={<CheckCircleOutlined />}
          />
        </Card>
      </Col>
      <Col span={8}>
        <Card>
          <Statistic
            title="停用楼层"
            value={statistics.inactive}
            styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
            prefix={<StopOutlined />}
          />
        </Card>
      </Col>
    </Row>
  );
};

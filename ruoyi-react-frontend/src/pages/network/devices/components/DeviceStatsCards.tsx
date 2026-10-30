/**
 * Phase 67-T3: 设备统计卡片（原 index.tsx :684-727 纯搬迁）
 * 保留原条件渲染语义：statistics.total > 10 才展示（存量行为，不扩不缩）。
 */
import { Card, Col, Row, Statistic } from "antd";
import {
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
} from "@ant-design/icons";

export interface DeviceStats {
  total: number;
  online: number;
  offline: number;
  unknown: number;
}

interface DeviceStatsCardsProps {
  statistics: DeviceStats;
}

const DeviceStatsCards: React.FC<DeviceStatsCardsProps> = ({ statistics }) => {
  if (statistics.total <= 10) return null;
  return (
    <Row gutter={16} style={{ marginBottom: 16 }}>
      <Col span={6}>
        <Card>
          <Statistic title="设备总数" value={statistics.total} prefix={<CloudServerOutlined />} />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="在线设备"
            value={statistics.online}
            styles={{ content: { color: "var(--theme-success)" } }}
            prefix={<CheckCircleOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="离线设备"
            value={statistics.offline}
            styles={{ content: { color: "var(--theme-error)" } }}
            prefix={<CloseCircleOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="未知状态"
            value={statistics.unknown}
            styles={{ content: { color: "var(--theme-warning)" } }}
            prefix={<WarningOutlined />}
          />
        </Card>
      </Col>
    </Row>
  );
};

export default DeviceStatsCards;

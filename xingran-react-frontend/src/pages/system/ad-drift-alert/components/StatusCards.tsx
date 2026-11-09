/**
 * 同步漂移告警 — 状态卡（Phase 144 ALRT-03）
 *
 * 展示：累计告警次数 / 当前基线 / 上次告警 / 容差 / 节流窗口 /
 * 各通道开关（enabled / noticeEnabled / wsEnabled / webhookEnabled）。
 *
 * 只读展示组件：数据由 index.tsx 的 getStatus() 注入，自身不发请求、不订阅 WS。
 * loading 时用 antd Card 的 loading 骨架，避免出现「0」误读为真实值。
 */
import { Card, Col, Row, Statistic, Tag, Tooltip } from "antd";
import {
  AlertOutlined,
  ApiOutlined,
  BellOutlined,
  CloudSyncOutlined,
  FieldNumberOutlined,
  InfoCircleOutlined,
} from "@ant-design/icons";
import type { FC, ReactNode } from "react";
import type { ADDriftAlertStatus } from "@/lib/adDriftAlertApi";
import { formatDateTime } from "@/utils/datetime";

export interface StatusCardsProps {
  status: ADDriftAlertStatus | null;
  loading?: boolean;
}

/** 通道开关徽标：on=绿色启用 / off=默认灰色停用 */
const ChannelTag: FC<{ label: string; on: boolean; icon: ReactNode }> = ({ label, on, icon }) => (
  <Tooltip title={on ? "已启用" : "已停用"}>
    <Tag color={on ? "success" : "default"} icon={icon} style={{ marginInlineEnd: 4 }}>
      {label}
    </Tag>
  </Tooltip>
);

const StatusCards: FC<StatusCardsProps> = ({ status, loading = false }) => {
  const baselineText =
    status?.baseline === null || status?.baseline === undefined || status?.baseline === ""
      ? "-"
      : String(status.baseline);

  return (
    <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
      <Col xs={24} sm={12} lg={6}>
        <Card loading={loading} size="small">
          <Statistic
            title="累计告警次数"
            value={status?.alertCount ?? 0}
            prefix={<AlertOutlined />}
            valueStyle={{ color: (status?.alertCount ?? 0) > 0 ? "var(--theme-error)" : undefined }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card loading={loading} size="small">
          <Statistic
            title="当前基线（成员数,组数）"
            value={baselineText}
            prefix={<FieldNumberOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card loading={loading} size="small">
          <Statistic
            title="容差 / 节流"
            value={`${status?.tolerance ?? 0} / ${status?.throttleMinutes ?? 0}min`}
            prefix={<InfoCircleOutlined />}
          />
          <div style={{ marginTop: 4, fontSize: 12, color: "var(--theme-text-tertiary)" }}>
            上次告警：
            {status?.lastAlertAt ? formatDateTime(status.lastAlertAt) : "无"}
            {status?.lastAlertDrifted ? "（检测到漂移）" : ""}
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card loading={loading} size="small" title="告警通道">
          <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
            <ChannelTag label="总开关" on={status?.enabled ?? false} icon={<CloudSyncOutlined />} />
            <ChannelTag
              label="站内通知"
              on={status?.noticeEnabled ?? false}
              icon={<BellOutlined />}
            />
            <ChannelTag label="WS 实时" on={status?.wsEnabled ?? false} icon={<ApiOutlined />} />
            <ChannelTag
              label="Webhook"
              on={status?.webhookEnabled ?? false}
              icon={<CloudSyncOutlined />}
            />
          </div>
        </Card>
      </Col>
    </Row>
  );
};

export default StatusCards;

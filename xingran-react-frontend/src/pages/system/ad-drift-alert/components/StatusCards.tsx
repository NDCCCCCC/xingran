/**
 * 同步漂移告警 — 状态卡（Phase 144 ALRT-03）
 *
 * 展示：累计告警次数 / 当前基线 / 上次告警 / 容差 / 节流窗口 /
 * 各通道开关（enabled / noticeEnabled / wsEnabled / webhookEnabled）。
 *
 * 只读展示组件：数据由 index.tsx 的 getStatus() 注入，自身不发请求、不订阅 WS。
 * 数值统计卡统一走 <StatCards>；通道开关 Tag 组属于另一类展示，保留在其后。
 * loading 时用 show 隐藏数值卡，避免把「0」误读为真实值。
 */
import { Card, Tag, Tooltip } from "antd";
import { ApiOutlined, BellOutlined, CloudSyncOutlined } from "@ant-design/icons";
import type { FC, ReactNode } from "react";
import type { ADDriftAlertStatus } from "@/lib/adDriftAlertApi";
import { formatDateTime } from "@/utils/datetime";
import { StatCards } from "@/components/shared";

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

  const lastAlertText = status?.lastAlertAt
    ? `${formatDateTime(status.lastAlertAt)}${status?.lastAlertDrifted ? "（检测到漂移）" : ""}`
    : "无";

  return (
    <div style={{ marginBottom: 16 }}>
      {/* 数值统计卡：统一走 <StatCards>；loading 时不显示，避免 0 被误读为真实值 */}
      <StatCards
        show={!loading}
        items={[
          {
            label: "累计告警次数",
            value: status?.alertCount ?? 0,
            tone: (status?.alertCount ?? 0) > 0 ? "danger" : "default",
          },
          { label: "当前基线（成员数,组数）", value: baselineText },
          {
            label: "容差 / 节流",
            value: `${status?.tolerance ?? 0} / ${status?.throttleMinutes ?? 0}min`,
            trend: `上次告警：${lastAlertText}`,
          },
        ]}
      />

      {/* 通道开关：Tag 徽标组属于另一类展示，保留原功能与骨架 */}
      <Card loading={loading} size="small" title="告警通道" style={{ marginTop: 16 }}>
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
    </div>
  );
};

export default StatusCards;

/**
 * MAC 事件时间线组件(Phase 14-01 引入)
 *
 * 跨页复用:列表页"展开行" / 轨迹页"右侧侧边栏" / 设备详情页"该设备 MAC 事件"
 * 颜色与图标体系与 MACTrajectoryChart.tsx:24-29 完全一致(D-10 锁定)
 *   appeared     → #52c41a (绿色) + PlusCircleOutlined
 *   disappeared → #ff4d4f (红色) + MinusCircleOutlined
 *   moved        → #faad14 (黄色) + SwapOutlined
 *   vlan_changed → #1890ff (蓝色) + TagOutlined
 *
 * 事件项点击后跳 `/network/mac/trajectory?mac=...&startTime=...&endTime=...[&deviceId=...]`(D-11 锁定)
 */

import React from 'react';
import { Timeline, Empty, Skeleton, Tag, Card, Space, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { getMACEvents } from '@/lib/api/networkApi';
import type { MACHistoryRecord } from '@/lib/api/networkApi';
import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL, EVENT_TAG_COLOR, type MACEventType } from './macEventMeta';
import { ErrorAlertWithRetry } from '@/components/shared';

export interface MACEventsTimelineProps {
  /** MAC 地址(必填) */
  mac: string;
  /** 时间范围起点(ISO 字符串) */
  startTime: string;
  /** 时间范围终点(ISO 字符串) */
  endTime: string;
  /** 设备 ID(可选,点击事件项跳轨迹页时携带) */
  deviceId?: string;
}

/**
 * 渲染单条事件
 */
const TimelineItem: React.FC<{
  event: MACHistoryRecord;
  onClick: () => void;
}> = ({ event, onClick }) => {
  const eventType = (event.eventType ?? 'appeared') as MACEventType;
  const Icon = EVENT_ICON[eventType];
  const color = EVENT_COLORS[eventType];
  const label = EVENT_LABEL[eventType];

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onClick();
        }
      }}
      style={{ cursor: 'pointer', padding: '4px 8px', borderRadius: 4 }}
    >
      <Space direction="vertical" size={2} style={{ width: '100%' }}>
        <Space size={8} wrap>
          <Typography.Text strong>
            {dayjs(event.firstSeen).format('YYYY-MM-DD HH:mm')}
          </Typography.Text>
          <Tag color={EVENT_TAG_COLOR[eventType]}>{label}</Tag>
          {event.vlanId != null && <Tag> VLAN {event.vlanId} </Tag>}
        </Space>
        <Space size={4} wrap>
          <Typography.Text type="secondary">设备:</Typography.Text>
          <Typography.Text>{event.deviceNameSnapshot || event.deviceId}</Typography.Text>
          <Typography.Text type="secondary">| 端口:</Typography.Text>
          <Typography.Text>{event.interfaceName}</Typography.Text>
        </Space>
      </Space>
    </div>
  );
};

const MACEventsTimeline: React.FC<MACEventsTimelineProps> = ({
  mac,
  startTime,
  endTime,
  deviceId,
}) => {
  const navigate = useNavigate();

  const { data: events, isLoading, error, refetch } = useQuery({
    queryKey: ['macEvents', mac, startTime, endTime],
    queryFn: () => getMACEvents(mac, startTime, endTime),
    enabled: !!mac,
    staleTime: 60 * 1000,
  });

  // 错误
  if (error) {
    return (
      <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
        <ErrorAlertWithRetry error={error as Error} onRetry={() => { void refetch(); }} />
      </Card>
    );
  }

  // 加载
  if (isLoading) {
    return (
      <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
        <Skeleton active paragraph={{ rows: 4 }} />
      </Card>
    );
  }

  // 空数据
  if (!events || events.length === 0) {
    return (
      <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
        <Empty description="该 MAC 在此时间范围内无事件" />
      </Card>
    );
  }

  return (
    <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
      <Timeline
        mode="left"
        items={events.map((event) => {
          const eventType = (event.eventType ?? 'appeared') as MACEventType;
          const Icon = EVENT_ICON[eventType];
          const color = EVENT_COLORS[eventType];
          return {
            color,
            dot: React.createElement(Icon, {
              style: { color, fontSize: 16 },
            }),
            children: (
              <TimelineItem
                event={event}
                onClick={() => {
                  const search = new URLSearchParams();
                  search.set('mac', mac);
                  search.set('startTime', startTime);
                  search.set('endTime', endTime);
                  if (deviceId) search.set('deviceId', deviceId);
                  navigate(`/network/mac/trajectory?${search.toString()}`);
                }}
              />
            ),
          };
        })}
      />
    </Card>
  );
};

export default MACEventsTimeline;

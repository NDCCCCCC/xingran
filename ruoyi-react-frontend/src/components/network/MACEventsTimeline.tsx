/**
 * MAC 事件时间线组件(Phase 14-01 引入,2026-06-30 quick 改造)
 *
 * 当前用法: MAC 地址列表页(/network/mac)的右侧 Drawer 内显示。
 *
 * 2026-06-30 quick 改造:
 * - 把 getMACEvents 改为分页版,支持"加载更多"按钮(busy MAC 数据量大时分页拉)
 * - 本组件维护累积 events 列表 + currentPage + hasMore
 * - 抽屉空间有限,> 100 事件时分页拉,避免一次性渲染卡顿
 *
 * 颜色与图标体系(D-10 锁定):
 *   appeared     → #52c41a (绿色) + PlusCircleOutlined
 *   disappeared → #ff4d4f (红色) + MinusCircleOutlined
 *   moved        → #faad14 (黄色) + SwapOutlined
 *   vlan_changed → #1890ff (蓝色) + TagOutlined
 */

import React, { useState, useCallback, useMemo, useEffect } from "react";
import { Timeline, Empty, Skeleton, Tag, Card, Space, Typography, Button } from "antd";
import { useQuery } from "@tanstack/react-query";
import { DownOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { getMACEvents } from "@/lib/api/networkApi";
import type { MACHistoryRecord } from "@/lib/api/networkApi";
import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL, EVENT_TAG_COLOR, type MACEventType } from "./macEventMeta";
import { ErrorAlertWithRetry } from "@/components/shared";

export interface MACEventsTimelineProps {
  /** MAC 地址(必填) */
  mac: string;
  /** 时间范围起点(ISO 字符串) */
  startTime: string;
  /** 时间范围终点(ISO 字符串) */
  endTime: string;
  /** 单页条数(默认 100) */
  pageSize?: number;
  /** 设备 ID(可选, R5 物理链路场景传入, 当前未在组件内使用, 仅透传给未来扩展) */
  deviceId?: string;
}

const TimelineItem: React.FC<{
  event: MACHistoryRecord;
}> = ({ event }) => {
  const eventType = (event.eventType ?? "appeared") as MACEventType;
  const label = EVENT_LABEL[eventType];

  return (
    <div style={{ padding: "4px 8px", borderRadius: 4 }}>
      <Space direction="vertical" size={2} style={{ width: "100%" }}>
        <Space size={8} wrap>
          <Typography.Text strong>
            {dayjs(event.firstSeen).format("YYYY-MM-DD HH:mm")}
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
  pageSize = 100,
}) => {
  // 累积 events 列表 + 分页状态(2026-06-30 quick:替换原 useQuery 单次全量)
  const [allEvents, setAllEvents] = useState<MACHistoryRecord[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [total, setTotal] = useState(0);
  const [loadingMore, setLoadingMore] = useState(false);

  // mac/startTime/endTime 变化时重置(抽屉开关场景)
  useEffect(() => {
    setAllEvents([]);
    setCurrentPage(1);
    setHasMore(false);
    setTotal(0);
  }, [mac, startTime, endTime]);

  // 首页查询(交给 useQuery 处理 loading / error / refetch)
  const { data: firstPage, isLoading, error, refetch } = useQuery({
    queryKey: ["macEvents", mac, startTime, endTime, 1],
    queryFn: () => getMACEvents(mac, startTime, endTime, { current: 1, pageSize }),
    enabled: !!mac,
    staleTime: 60 * 1000,
  });

  // 同步首页结果到累积状态
  useEffect(() => {
    if (firstPage) {
      setAllEvents(firstPage.list);
      setTotal(firstPage.total);
      setHasMore(firstPage.hasMore);
      setCurrentPage(1);
    }
  }, [firstPage]);

  // 加载更多
  const handleLoadMore = useCallback(async () => {
    if (loadingMore || !hasMore) return;
    const nextPage = currentPage + 1;
    setLoadingMore(true);
    try {
      const result = await getMACEvents(mac, startTime, endTime, { current: nextPage, pageSize });
      setAllEvents((prev) => [...prev, ...result.list]);
      setHasMore(result.hasMore);
      setCurrentPage(nextPage);
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, hasMore, currentPage, mac, startTime, endTime, pageSize]);

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
  if (allEvents.length === 0) {
    return (
      <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
        <Empty description="该 MAC 在此时间范围内无事件" />
      </Card>
    );
  }

  return (
    <Card
      size="small"
      title={
        <Space size={4}>
          <span>MAC 事件时间线 — {mac}</span>
          {total > pageSize && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              (已显示 {allEvents.length}/{total})
            </Typography.Text>
          )}
        </Space>
      }
      bordered={false}
    >
      <Timeline
        mode="left"
        items={allEvents.map((event) => {
          const eventType = (event.eventType ?? "appeared") as MACEventType;
          const Icon = EVENT_ICON[eventType];
          const color = EVENT_COLORS[eventType];
          return {
            color,
            dot: React.createElement(Icon, {
              style: { color, fontSize: 16 },
            }),
            children: <TimelineItem event={event} />,
          };
        })}
      />
      {hasMore && (
        <div style={{ textAlign: "center", marginTop: 12 }}>
          <Button
            type="link"
            icon={<DownOutlined />}
            onClick={handleLoadMore}
            loading={loadingMore}
          >
            加载更多 (还剩 {total - allEvents.length} 条)
          </Button>
        </div>
      )}
    </Card>
  );
};

export default MACEventsTimeline;

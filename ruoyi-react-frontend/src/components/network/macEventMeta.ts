/**
 * MAC 事件类型元数据(D-10 锁定单一事实源)
 * - 颜色:appeared=green,disappeared=red,moved=gold,vlan_changed=blue
 * - 图标:PlusCircleOutlined / MinusCircleOutlined / SwapOutlined / TagOutlined
 * - 中文标签:出现 / 消失 / 迁移 / VLAN 变更
 *
 * 被以下文件 import:
 * - components/network/MACEventsTimeline.tsx
 * - pages/network/mac/history/MACHistoryPage.tsx
 * - pages/network/mac/index.tsx(列表页 Drawer,2026-06-30 quick)
 */
import {
  PlusCircleOutlined,
  MinusCircleOutlined,
  SwapOutlined,
  TagOutlined,
} from "@ant-design/icons";
import type { ComponentType, CSSProperties } from "react";

export type MACEventType = "appeared" | "disappeared" | "moved" | "vlan_changed";

export const EVENT_COLORS: Record<MACEventType, string> = {
  appeared: "var(--theme-success, #2d8949)",
  disappeared: "#ba3630",
  moved: "var(--theme-warning, #b07a20)",
  vlan_changed: "var(--theme-info, #337ab0)",
};

export const EVENT_ICON: Record<MACEventType, ComponentType<{ style?: CSSProperties }>> = {
  appeared: PlusCircleOutlined,
  disappeared: MinusCircleOutlined,
  moved: SwapOutlined,
  vlan_changed: TagOutlined,
};

export const EVENT_LABEL: Record<MACEventType, string> = {
  appeared: "出现",
  disappeared: "消失",
  moved: "迁移",
  vlan_changed: "VLAN 变更",
};

/** AntD Tag color (与 ECharts hex 兼容) */
export const EVENT_TAG_COLOR: Record<MACEventType, string> = {
  appeared: "green",
  disappeared: "red",
  moved: "gold",
  vlan_changed: "blue",
};

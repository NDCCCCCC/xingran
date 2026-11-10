/**
 * Phase 67-T4: 缓存监控概览卡（自 index.tsx :489-564 纯搬迁，零 API 修改）
 * 注：裸 hex 色值（#3f8600/#337ab0）按 UI-SPEC B 类豁免原样保留——只搬不增删。
 * UI 统一整改：顶部 4 卡概览整体替换为 <StatCards>；内联色值改语义 tone，
 * 丢弃 prefix 图标；第 3/4 卡的复合内容（双命中率、状态 Tag）用 ReactNode value 原样保留。
 */
import type { FC } from "react";
import { Tag } from "antd";
import { StatCards } from "@/components/shared";
import type { CacheMonitor } from "../types";
import { formatMemorySize } from "../utils";

interface MonitorOverviewProps {
  monitor: CacheMonitor | null;
}

const MonitorOverview: FC<MonitorOverviewProps> = ({ monitor }) => (
  <StatCards
    className="mb-6"
    items={[
      {
        label: "L1 内存缓存键数",
        value: monitor?.l1?.stats?.keyCount || 0,
        trend: `内存 ${formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}`,
      },
      {
        label: "L2 Redis缓存键数",
        value: monitor?.l2?.stats?.keyCount || 0,
        trend: `内存 ${formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}`,
      },
      {
        label: "综合命中率",
        value: (
          <div className="flex flex-col gap-1 text-base leading-snug">
            <span>L1 {(monitor?.l1?.stats?.hitRate || 0).toFixed(1)}%</span>
            <span>L2 {(monitor?.l2?.stats?.hitRate || 0).toFixed(1)}%</span>
          </div>
        ),
        tone: "green",
      },
      {
        label: "缓存状态",
        value: (
          <div className="flex flex-col gap-1 text-base">
            <div className="flex justify-between items-center">
              <span>L1状态:</span>
              <Tag color={monitor?.l1?.status?.connected ? "success" : "error"}>
                {monitor?.l1?.status?.connected ? "正常" : "异常"}
              </Tag>
            </div>
            <div className="flex justify-between items-center">
              <span>L2状态:</span>
              <Tag color={monitor?.l2?.status?.connected ? "success" : "error"}>
                {monitor?.l2?.status?.connected ? "正常" : "异常"}
              </Tag>
            </div>
          </div>
        ),
      },
    ]}
  />
);

export default MonitorOverview;

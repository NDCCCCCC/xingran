/**
 * Phase 67-T4: 缓存统计信息模态框（自 index.tsx :833-969 纯搬迁，零 API 修改）
 * 注：裸 hex 色值按 B 类豁免原样保留；Tabs「缓存统计」面板内容为本 Modal 的子集
 * （键数量/命中/未命中/命中率/内存使用均覆盖），迁移时 Tabs 移除、入口保留「统计信息」按钮。
 * UI 统一整改：弹窗内 L1 / L2 / 综合对比三组 <Row><Col><Card><Statistic> 各改成一个
 * <StatCards>，保持 Modal 结构不变；内联色值改语义 tone，丢弃 precision/suffix 的样式表达。
 */
import type { FC } from "react";
import { Divider, Modal, Tag } from "antd";
import { StatCards } from "@/components/shared";
import type { CacheMonitor } from "../types";
import { formatMemorySize } from "../utils";

interface CacheStatsModalProps {
  open: boolean;
  onClose: () => void;
  monitor: CacheMonitor | null;
}

const CacheStatsModal: FC<CacheStatsModalProps> = ({ open, onClose, monitor }) => (
  <Modal
    title="缓存统计信息"
    mask={{ closable: false }}
    keyboard={false}
    open={open}
    onCancel={onClose}
    footer={null}
    width={900}
  >
    <div className="mb-2">
      <Tag color="blue">L1</Tag> 内存缓存统计
    </div>
    <StatCards
      items={[
        { label: "键数量", value: monitor?.l1?.stats?.keyCount || 0 },
        {
          label: "命中次数",
          value: monitor?.l1?.stats?.hitCount || 0,
          tone: "green",
        },
        {
          label: "未命中次数",
          value: monitor?.l1?.stats?.missCount || 0,
          tone: "danger",
        },
        {
          label: "命中率",
          value: `${(monitor?.l1?.stats?.hitRate || 0).toFixed(2)}%`,
          tone: "default",
        },
        {
          label: "内存使用",
          value: formatMemorySize(monitor?.l1?.stats?.usedMemory || 0),
        },
      ]}
    />

    <Divider />

    <div className="mb-2">
      <Tag color="green">L2</Tag> Redis缓存统计
    </div>
    <StatCards
      items={[
        { label: "键数量", value: monitor?.l2?.stats?.keyCount || 0 },
        {
          label: "命中次数",
          value: monitor?.l2?.stats?.hitCount || 0,
          tone: "green",
        },
        {
          label: "未命中次数",
          value: monitor?.l2?.stats?.missCount || 0,
          tone: "danger",
        },
        {
          label: "命中率",
          value: `${(monitor?.l2?.stats?.hitRate || 0).toFixed(2)}%`,
          tone: "default",
        },
        {
          label: "内存使用",
          value: formatMemorySize(monitor?.l2?.stats?.usedMemory || 0),
        },
      ]}
    />

    <Divider />

    <div className="mb-2">综合对比</div>
    <StatCards
      items={[
        {
          label: "总键数(L1+L2)",
          value: (monitor?.l1?.stats?.keyCount || 0) + (monitor?.l2?.stats?.keyCount || 0),
        },
        {
          label: "总内存使用",
          value: formatMemorySize(
            (monitor?.l1?.stats?.usedMemory || 0) + (monitor?.l2?.stats?.usedMemory || 0)
          ),
        },
        {
          label: "L1缓存命中率",
          value: `${(monitor?.l1?.stats?.hitRate || 0).toFixed(2)}%`,
        },
      ]}
    />
  </Modal>
);

export default CacheStatsModal;

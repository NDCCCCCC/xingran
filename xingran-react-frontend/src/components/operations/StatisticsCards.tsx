/**
 * 统计卡片组件（运营管理页面共用）
 *
 * UI 统一整改：内部渲染改为全站统一的增强版 <StatCards> 范本
 * （系统管理 user/settings 页基准：左 3px 色条 + label/value/trend 三段
 *  + 底边占比轨 ratio + 可点下钻 onClick），由 shared/StatCards 承载，
 * 替代旧 antd <Row><Col><Card><Statistic>> 结构。
 *
 * 对外 props 契约保持不变（items / show / columns / style），6 个引用页无需改动；
 * - items[].title → label，items[].value → value；
 * - items[].prefix / valueStyle / styles.content.color 为旧 Statistic 用法，
 *   新范本不含图标与内联染色，统一按语义色条（tone）表达，见下方 inferTone；
 *   故这些字段仅保留类型兼容、不再影响渲染。
 */
import type { FC } from "react";
import type { ReactNode } from "react";
import StatCards from "@/components/shared/StatCards";
import type { StatCardTone } from "@/components/shared/StatCards";

export interface StatisticItem {
  title: string;
  value: number;
  prefix?: ReactNode;
  styles?: { content?: { color?: string } };
  /** @deprecated 请改用 `styles.content` */
  valueStyle?: React.CSSProperties;
}

interface StatisticsCardsProps {
  /** 统计数据项 */
  items: StatisticItem[];
  /** 是否显示（默认在 total > 10 时显示） */
  show?: boolean;
  /** 每行显示的列数（新范本为响应式 grid，此参数仅为兼容旧调用保留） */
  columns?: number;
  /** 自定义样式 */
  style?: React.CSSProperties;
}

/**
 * 把旧的内联 color 变量映射到 .stat-card 色条语义：
 * --theme-success → green / --theme-error → danger / --theme-warning → gold
 * --theme-text-tertiary / --theme-text-secondary / --theme-info → gray
 * 其余 → default（品牌主色）。
 */
function inferTone(color?: string): StatCardTone {
  if (!color) return "default";
  if (color.includes("success")) return "green";
  if (color.includes("error") || color.includes("danger")) return "danger";
  if (color.includes("warning")) return "gold";
  if (color.includes("tertiary") || color.includes("secondary") || color.includes("--theme-info"))
    return "gray";
  return "default";
}

/**
 * 运营管理页面通用的统计卡片组
 */
export const StatisticsCards: FC<StatisticsCardsProps> = ({ items, show = true, style }) => {
  if (!show) return null;

  return (
    <StatCards
      show={show}
      style={style}
      items={items.map((item) => ({
        label: item.title,
        value: item.value,
        tone: inferTone(item.styles?.content?.color ?? item.valueStyle?.color),
      }))}
    />
  );
};

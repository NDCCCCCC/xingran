/**
 * 全站统一统计卡组件（范本 = 系统管理 user/settings 的 .stat-cards/.stat-card）。
 *
 * 设计基线（src/index.css 统计卡组区块）：
 * - 容器 .stat-cards：grid auto-fit minmax(170px,1fr) + 12px gap
 * - 卡片 .stat-card：白卡 + 左 3px 色条（.sc-green/.sc-gray/.sc-gold/.sc-danger）
 * - 三段文本：.stat-label / .stat-value / .stat-trend
 * - 底边 2px 占比轨：由 `ratio` 注入 CSS 变量 `--ratio`（色条从装饰变数据）
 *
 * 用于替代各业务模块里散落的 antd <Row><Col><Card><Statistic>> 写法，
 * 保证「汇总数据布局 / 配色」全站统一到系统管理新范本。
 * 纯展示、无 antd Statistic 依赖；颜色语义一律走色条 class，不内联染色。
 */
import type { CSSProperties, FC, KeyboardEvent, ReactNode } from "react";

/** 色条语义：对应 .sc-* 修饰类 */
export type StatCardTone = "default" | "green" | "gray" | "gold" | "danger";

export interface StatCardItem {
  /** 指标名称（如「设备总数」） */
  label: string;
  /** 指标值（数字或自定义节点，如「无」/ 带单位字符串） */
  value: number | string | ReactNode;
  /** 辅助说明行：只放「比例 / 增量 / 口径」，不要塞功能说明类文案 */
  trend?: string;
  /** 色条语义，默认品牌主色；成功/中性/警示/危险 分别对应绿/灰/金/红 */
  tone?: StatCardTone;
  /** 0–1：驱动底边占比轨；不传则隐藏（宽度 0），不影响三段结构 */
  ratio?: number;
  /** 传入即整卡可点（下钻筛选），同时启用 hover 反馈与键盘可达 */
  onClick?: () => void;
  /** 悬停说明；也可承接「不受筛选影响」这类口径提示 */
  hint?: string;
}

export interface StatCardsProps {
  items: StatCardItem[];
  /** 是否显示（供 `total > 10 才展示` 这类存量条件渲染复用） */
  show?: boolean;
  /** 额外的容器样式（如自定义 marginBottom） */
  style?: React.CSSProperties;
  /** 额外的容器 className */
  className?: string;
}

const toneClass: Record<StatCardTone, string> = {
  default: "",
  green: "sc-green",
  gray: "sc-gray",
  gold: "sc-gold",
  danger: "sc-danger",
};

/** 把任意输入夹到 [0,1]，并抹掉浮点尾差（CSS 宽度计算用） */
const clampRatio = (ratio?: number): number => {
  if (ratio === undefined || Number.isNaN(ratio)) return 0;
  return Math.round(Math.min(1, Math.max(0, ratio)) * 1000) / 1000;
};

const StatCards: FC<StatCardsProps> = ({ items, show = true, style, className }) => {
  if (!show) return null;
  return (
    <div className={["stat-cards", className].filter(Boolean).join(" ")} style={style}>
      {items.map((item, index) => {
        const clickable = typeof item.onClick === "function";
        const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
          if (event.key !== "Enter" && event.key !== " ") return;
          event.preventDefault();
          item.onClick?.();
        };
        return (
          <div
            key={`${item.label}-${index}`}
            className={[
              "stat-card",
              toneClass[item.tone ?? "default"],
              clickable ? "is-clickable" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            style={{ "--ratio": clampRatio(item.ratio) } as CSSProperties}
            title={item.hint}
            {...(clickable
              ? {
                  role: "button" as const,
                  tabIndex: 0,
                  onClick: item.onClick,
                  onKeyDown: handleKeyDown,
                }
              : {})}
          >
            <div className="stat-label">{item.label}</div>
            <div className="stat-value">{item.value}</div>
            {item.trend !== undefined && <div className="stat-trend">{item.trend}</div>}
          </div>
        );
      })}
    </div>
  );
};

export default StatCards;

/**
 * 统计卡片组件
 * 用于运营管理页面的统计数据展示
 */

import type { FC } from "react";
import { Row, Col, Card, Statistic } from "antd";
import type { ReactNode } from "react";

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
  /** 每行显示的列数 */
  columns?: number;
  /** 自定义样式 */
  style?: React.CSSProperties;
}

/**
 * 运营管理页面通用的统计卡片组
 */
export const StatisticsCards: FC<StatisticsCardsProps> = ({
  items,
  show = true,
  columns = items.length,
  style,
}) => {
  if (!show) return null;

  const span = 24 / columns;

  return (
    <Row gutter={16} style={{ marginBottom: 16, ...style }}>
      {items.map((item, index) => (
        <Col key={index} span={span}>
          <Card>
            <Statistic
              title={item.title}
              value={item.value}
              prefix={item.prefix}
              styles={{
                content: {
                  ...(item.valueStyle ?? {}),
                  ...(item.styles?.content ?? {}),
                },
              }}
            />
          </Card>
        </Col>
      ))}
    </Row>
  );
};

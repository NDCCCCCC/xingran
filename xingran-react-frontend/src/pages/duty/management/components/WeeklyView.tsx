import React from "react";
import { Card, Row, Col, Space, Badge, Typography } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import type { MonthlyDutyMember } from "@/lib/dutyApi";

const { Text } = Typography;

interface WeeklyViewProps {
  currentWeekStart: Dayjs;
  weeklyDutyData: Record<string, MonthlyDutyMember[]>;
}

export const WeeklyView: React.FC<WeeklyViewProps> = ({ currentWeekStart, weeklyDutyData }) => {
  const getWeekRangeText = () => {
    const weekEnd = currentWeekStart.endOf("week");
    return `${currentWeekStart.format("YYYY年MM月DD日")} - ${weekEnd.format("YYYY年MM月DD日")}`;
  };

  const getWeekDays = () => {
    const days = [];
    for (let i = 0; i < 7; i++) {
      days.push(currentWeekStart.add(i, "day"));
    }
    return days;
  };

  const getWeekdayText = (day: Dayjs) => {
    const weekdays = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
    return weekdays[day.day()];
  };

  const getDutyTypeBadgeColor = (type: string) => {
    switch (type) {
      case "weekday":
        return "blue";
      case "weekend":
        return "orange";
      case "holiday":
        return "red";
      default:
        // 开发环境下警告未知的 dutyType (仅在控制台可见)
        if (
          type &&
          typeof window !== "undefined" &&
          (window as Window & { __DEV__?: boolean }).__DEV__
        ) {
          console.warn(`[WeeklyView] 未知的 dutyType 值: "${type}"`);
        }
        return "default";
    }
  };

  return (
    <Card variant="borderless">
      <div style={{ marginBottom: "16px" }}>
        <Text strong style={{ fontSize: "14px" }}>
          {getWeekRangeText()}
        </Text>
      </div>
      <Row gutter={[8, 8]}>
        {getWeekDays().map((day) => {
          const dateStr = day.format("YYYY-MM-DD");
          const members = weeklyDutyData[dateStr] || [];
          const isToday = day.format("YYYY-MM-DD") === dayjs().format("YYYY-MM-DD");

          return (
            <Col key={dateStr} span={3}>
              <Card
                size="small"
                style={{
                  height: "120px",
                  borderColor: isToday ? "var(--theme-info, #337ab0)" : undefined,
                  backgroundColor: isToday ? "#e6f7ff" : undefined,
                }}
                styles={{ body: { padding: "8px", height: "100%", overflow: "auto" } }}
              >
                <div style={{ display: "flex", alignItems: "flex-start", height: "100%" }}>
                  <div style={{ marginRight: "8px", minWidth: "50px" }}>
                    <Text
                      strong
                      style={{
                        fontSize: "13px",
                        color: isToday ? "var(--theme-info, #337ab0)" : undefined,
                        display: "block",
                      }}
                    >
                      {getWeekdayText(day)}
                    </Text>
                    <Text
                      style={{
                        fontSize: "12px",
                        color: "var(--theme-text-tertiary, #999)",
                        display: "block",
                      }}
                    >
                      {day.format("MM-DD")}
                    </Text>
                  </div>
                  <div style={{ flex: 1 }}>
                    {members.length > 0 ? (
                      <Space size="small" wrap>
                        {members.map((member, index) => (
                          <Badge
                            key={index}
                            color={getDutyTypeBadgeColor(member.dutyType)}
                            text={
                              <span
                                style={{
                                  fontSize: "12px",
                                  whiteSpace: "nowrap",
                                }}
                              >
                                {member.username}
                              </span>
                            }
                          />
                        ))}
                      </Space>
                    ) : (
                      <Text style={{ fontSize: "12px", color: "#ccc" }}>无值班</Text>
                    )}
                  </div>
                </div>
              </Card>
            </Col>
          );
        })}
      </Row>
      <div style={{ marginTop: "12px", display: "flex", gap: "16px", fontSize: "12px" }}>
        <Badge color="blue" text={<Text type="secondary">工作日</Text>} />
        <Badge color="orange" text={<Text type="secondary">周末</Text>} />
        <Badge color="red" text={<Text type="secondary">节假日</Text>} />
      </div>
    </Card>
  );
};

export default WeeklyView;

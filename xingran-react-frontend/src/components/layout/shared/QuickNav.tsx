/**
 * 快速导航组件
 * 提供常用功能的快速入口
 */

import type { FC, ReactNode } from "react";
import { Card, Row, Col } from "antd";
import { useNavigate } from "react-router-dom";
import { DASHBOARD, MONITOR_DASHBOARD } from "@/constants/routes";

interface QuickNavItem {
  title: string;
  icon: ReactNode;
  path: string;
  color: string;
}

const quickNavItems: QuickNavItem[] = [
  { title: "仪表盘", icon: "📊", path: DASHBOARD, color: "var(--theme-info, #156031)" },
  { title: "用户管理", icon: "👥", path: "/system/user", color: "#10b981" },
  { title: "菜单管理", icon: "📋", path: "/system/menu", color: "#b07a20" },
  { title: "系统监控", icon: "📈", path: MONITOR_DASHBOARD, color: "#ba3630" },
  {
    title: "设备管理",
    icon: "🌐",
    path: "/network/devices",
    color: "var(--theme-purple, #c09058)",
  },
  { title: "我的值班", icon: "📅", path: "/duty/my-duty", color: "#ec4899" },
];

const QuickNav: FC = () => {
  const navigate = useNavigate();

  return (
    <div style={{ marginBottom: "16px" }}>
      <Row gutter={[12, 12]}>
        {quickNavItems.map((item) => (
          <Col key={item.path} xs={12} sm={8} md={6} lg={4}>
            <Card
              hoverable
              onClick={() => navigate(item.path)}
              style={{
                textAlign: "center",
                borderRadius: "var(--theme-radius-lg)",
                border: "1px solid var(--theme-border-primary)",
                background: "var(--theme-bg-surface)",
                transition: "all var(--theme-transition-base)",
                cursor: "pointer",
              }}
              styles={{
                body: {
                  padding: "12px",
                },
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = "translateY(-2px)";
                e.currentTarget.style.boxShadow = "var(--theme-shadow-md)";
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = "translateY(0)";
                e.currentTarget.style.boxShadow = "none";
              }}
            >
              <div style={{ fontSize: "24px", marginBottom: "4px" }}>{item.icon}</div>
              <div
                style={{
                  fontSize: "12px",
                  fontWeight: 500,
                  color: "var(--theme-text-secondary)",
                }}
              >
                {item.title}
              </div>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
};

export default QuickNav;

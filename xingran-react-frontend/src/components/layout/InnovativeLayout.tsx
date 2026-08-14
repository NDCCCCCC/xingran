/**
 * 创新布局组件
 * 创新导航方式 + 模块化面板 + 沉浸式体验
 */

import { useState } from "react";
import { Card, Button, Tooltip, Space, Divider } from "antd";
import { useNavigate } from "react-router-dom";
import ThemeSwitcher from "@/design-system/components/ThemeSwitcher";
import LayoutSwitcher from "@/design-system/components/LayoutSwitcher";
import DensitySwitcher from "@/design-system/components/DensitySwitcher";
import ColorSwitcher from "@/design-system/components/ColorSwitcher";
import { useAuthStore } from "@/store/authStore";
import { DASHBOARD, MONITOR_DASHBOARD, LOGIN } from "@/constants/routes";
import {
  DashboardOutlined,
  UserOutlined,
  SettingOutlined,
  MonitorOutlined,
  CloudServerOutlined,
  CalendarOutlined,
  FileTextOutlined,
  ApiOutlined,
  LogoutOutlined,
} from "@ant-design/icons";
import QuickNav from "./shared/QuickNav";

import type { FC, ReactNode } from "react";

interface InnovativeLayoutProps {
  children: ReactNode;
}

interface SpaceNav {
  key: string;
  title: string;
  icon: ReactNode;
  path: string;
  color: string;
  description: string;
}

const SPACE_NAVS: SpaceNav[] = [
  {
    key: "dashboard",
    title: "仪表盘",
    icon: <DashboardOutlined />,
    path: DASHBOARD,
    color: "var(--theme-info, #3b82f6)",
    description: "数据概览与统计",
  },
  {
    key: "system",
    title: "系统管理",
    icon: <SettingOutlined />,
    path: "/system/user",
    color: "#10b981",
    description: "用户、角色、菜单",
  },
  {
    key: "monitor",
    title: "系统监控",
    icon: <MonitorOutlined />,
    path: MONITOR_DASHBOARD,
    color: "#f59e0b",
    description: "服务器、日志、任务",
  },
  {
    key: "network",
    title: "网络设备",
    icon: <CloudServerOutlined />,
    path: "/network/devices",
    color: "var(--theme-purple, #8b5cf6)",
    description: "设备、凭证、配置",
  },
  {
    key: "duty",
    title: "值班管理",
    icon: <CalendarOutlined />,
    path: "/duty/my-duty",
    color: "#ec4899",
    description: "排班、节假日、池",
  },
  {
    key: "workorder",
    title: "工单管理",
    icon: <FileTextOutlined />,
    path: "/workorder/orders",
    color: "#ef4444",
    description: "工单、分类、统计",
  },
  {
    key: "notice",
    title: "消息通知",
    icon: <ApiOutlined />,
    path: "/user/my-notices",
    color: "#14b8a6",
    description: "通知、消息、提醒",
  },
  {
    key: "profile",
    title: "个人中心",
    icon: <UserOutlined />,
    path: "/user/profile",
    color: "#6366f1",
    description: "个人信息、设置",
  },
  {
    key: "settings",
    title: "系统设置",
    icon: <SettingOutlined />,
    path: "/user/settings",
    color: "var(--theme-purple, #8b5cf6)",
    description: "配置、偏好、主题",
  },
];

const createSpaceIconStyles = (isActive: boolean, color: string) => ({
  width: "48px",
  height: "48px",
  borderRadius: "12px",
  display: "flex" as const,
  alignItems: "center",
  justifyContent: "center",
  cursor: "pointer",
  background: isActive ? color : "transparent",
  color: isActive ? "#fff" : "var(--theme-text-secondary)",
  transition: "all var(--theme-transition-base)",
  fontSize: "20px",
});

const InnovativeLayout: FC<InnovativeLayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [activeSpace, setActiveSpace] = useState<string>("dashboard");

  const handleSpaceClick = (space: SpaceNav) => {
    setActiveSpace(space.key);
    navigate(space.path);
  };

  const handleLogout = () => {
    logout();
    window.location.href = LOGIN;
  };

  return (
    <div
      className="h-screen flex"
      style={{
        background: "var(--theme-bg-secondary)",
        transition: "all var(--theme-transition-slow)",
      }}
    >
      <div
        style={{
          width: "80px",
          background: "var(--theme-bg-surface)",
          borderRight: "1px solid var(--theme-border-primary)",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          paddingTop: "16px",
          paddingBottom: "16px",
          gap: "12px",
          transition: "all var(--theme-transition-base)",
        }}
      >
        <div
          style={{
            fontSize: "24px",
            fontWeight: "bold",
            marginBottom: "16px",
            background:
              "linear-gradient(135deg, var(--theme-primary-500, #3b82f6) 0%, var(--theme-primary-600, #2563eb) 100%)",
            WebkitBackgroundClip: "text",
            WebkitTextFillColor: "transparent",
            backgroundClip: "text",
          }}
        >
          数智
        </div>

        {SPACE_NAVS.map((space) => (
          <Tooltip key={space.key} title={space.description} placement="right">
            <div
              onClick={() => handleSpaceClick(space)}
              style={createSpaceIconStyles(activeSpace === space.key, space.color)}
              onMouseEnter={(e) => {
                if (activeSpace !== space.key) {
                  e.currentTarget.style.background = space.color + "20";
                  e.currentTarget.style.transform = "scale(1.1)";
                }
              }}
              onMouseLeave={(e) => {
                if (activeSpace !== space.key) {
                  e.currentTarget.style.background = "transparent";
                  e.currentTarget.style.transform = "scale(1)";
                }
              }}
            >
              {space.icon}
            </div>
          </Tooltip>
        ))}
      </div>

      <div
        className="flex-1 overflow-auto"
        style={{
          padding: "24px",
          maxWidth: "1600px",
          margin: "0 auto",
          width: "100%",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "16px",
            padding: "12px 16px",
            borderRadius: "var(--theme-radius-lg)",
            background: "var(--theme-bg-surface)",
            border: "1px solid var(--theme-border-primary)",
            boxShadow: "var(--theme-shadow-sm)",
            transition: "all var(--theme-transition-base)",
          }}
        >
          <Space size="middle">
            <span
              style={{
                fontSize: "14px",
                fontWeight: 500,
                color: "var(--theme-text-secondary)",
              }}
            >
              XingRan
            </span>
          </Space>

          <Space size="middle" separator={<Divider orientation="vertical" style={{ margin: 0 }} />}>
            <ThemeSwitcher />
            <LayoutSwitcher />
            <DensitySwitcher />
            <ColorSwitcher />
            <span
              style={{
                fontSize: "14px",
                color: "var(--theme-text-secondary)",
              }}
            >
              {user?.nickname || user?.username}
            </span>
            <Tooltip title="退出登录">
              <Button
                type="text"
                icon={<LogoutOutlined />}
                onClick={handleLogout}
                style={{
                  color: "var(--theme-text-secondary)",
                  transition: "all var(--theme-transition-base)",
                }}
              />
            </Tooltip>
          </Space>
        </div>

        <QuickNav />

        <Card
          style={{
            borderRadius: "var(--theme-radius-xl)",
            border: "1px solid var(--theme-border-primary)",
            background: "var(--theme-bg-surface)",
            boxShadow: "var(--theme-shadow-sm)",
            transition: "all var(--theme-transition-base)",
          }}
          styles={{
            body: {
              padding: "24px",
            },
          }}
        >
          {children}
        </Card>
      </div>
    </div>
  );
};

export default InnovativeLayout;

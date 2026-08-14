import { Layout, Avatar, Dropdown, Space } from "antd";
import { UserOutlined, LogoutOutlined, SettingOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/store/authStore";
import { useWebSocket } from "@/hooks/useWebSocket";
import NotificationBell from "@/components/NotificationBell";
import GlobalSearch from "@/components/shared/GlobalSearch";
import { USER_PROFILE, USER_SETTINGS, LOGIN } from "@/constants/routes";
import type { MenuProps } from "antd";
import type { FC } from "react";
import { AVATAR_BORDER_OPACITY, HEADER_Z_INDEX } from "./header.constants";
import "@/design-system/themes/theme-styles.css";

const { Header: AntHeader } = Layout;

const Header: FC = () => {
  const { user, logout } = useAuthStore();
  const navigate = useNavigate();

  // 初始化 WebSocket 连接
  // useWebSocket 内部已处理只连接一次的逻辑（通过模块级单例 + useEffect + subscribe）
  // WebSocket 连接成功后会自动获取未读数量
  useWebSocket();

  const handleLogout = async () => {
    try {
      await logout();
      // 等待 React 状态更新完成
      await new Promise((resolve) => setTimeout(resolve, 0));
      window.location.href = LOGIN;
    } catch (error) {
      console.error("登出失败:", error);
      // 即使登出失败，也跳转到登录页
      window.location.href = LOGIN;
    }
  };

  const userMenuItems: MenuProps["items"] = [
    {
      key: "profile",
      label: "个人中心",
      icon: <UserOutlined />,
      onClick: () => navigate(USER_PROFILE),
    },
    {
      key: "settings",
      label: "系统设置",
      icon: <SettingOutlined />,
      onClick: () => navigate(USER_SETTINGS),
    },
    {
      type: "divider",
    },
    {
      key: "logout",
      label: "退出登录",
      icon: <LogoutOutlined />,
      onClick: handleLogout,
    },
  ];

  return (
    <AntHeader
      className="layout-header flex items-center justify-between px-6 shadow-lg"
      style={{
        position: "relative",
        zIndex: HEADER_Z_INDEX,
      }}
    >
      {/* 左侧留空，保持布局平衡 */}
      <div className="flex items-center gap-4"></div>

      <Space size="middle">
        {/* 通知铃铛 */}
        <NotificationBell />

        <span className="font-medium" style={{ color: "var(--theme-text-secondary)" }}>
          欢迎，{user?.nickname || user?.username}
        </span>

        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar
            size="large"
            icon={<UserOutlined />}
            src={user?.avatar}
            className="cursor-pointer shadow-md border-2"
            style={{
              background:
                "linear-gradient(135deg, var(--theme-brand, #3b82f6) 0%, var(--theme-brand-dark, #2563eb) 100%)",
              color: "var(--theme-text-inverse)",
              borderColor: `rgba(255, 255, 255, ${AVATAR_BORDER_OPACITY})`,
              transition: "var(--theme-transition-base)",
            }}
          />
        </Dropdown>
      </Space>

      {/* 全局搜索组件 */}
      <GlobalSearch />
    </AntHeader>
  );
};

export default Header;

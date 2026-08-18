import { useMemo } from "react";
import { Layout, Avatar, Dropdown, Space } from "antd";
import { UserOutlined, LogoutOutlined, SettingOutlined, DownOutlined } from "@ant-design/icons";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { useAuthStore } from "@/store/authStore";
import { useWebSocket } from "@/hooks/useWebSocket";
import { routeConfigManager } from "@/router/routeConfigManager";
import NotificationBell from "@/components/NotificationBell";
import GlobalSearch from "@/components/shared/GlobalSearch";
import { USER_PROFILE, USER_SETTINGS, LOGIN, DASHBOARD } from "@/constants/routes";
import type { MenuProps } from "antd";
import type { FC } from "react";
import { HEADER_Z_INDEX } from "./header.constants";

const { Header: AntHeader } = Layout;

const Header: FC = () => {
  const { user, logout } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

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

  // 面包屑（原型格式：「首页 / 父级」，不含当前页）
  const breadcrumbItems = useMemo(() => {
    if (!routeConfigManager.isInitialized()) {
      return [];
    }
    const chain = routeConfigManager.buildBreadcrumb(location.pathname);
    // 去掉末项（当前页）；链为空（如首页自身）时只显示「首页」
    return chain.slice(0, -1);
  }, [location.pathname]);

  return (
    <AntHeader
      className="layout-header flex items-center justify-between px-6 shadow-lg"
      style={{
        position: "relative",
        zIndex: HEADER_Z_INDEX,
      }}
    >
      {/* 左侧：面包屑（原型 header 左区） */}
      <nav className="header-breadcrumb" aria-label="面包屑">
        <Link to={DASHBOARD}>首页</Link>
        {breadcrumbItems.map((item) => (
          <span key={item.path} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
            <span className="sep">/</span>
            <span>{item.title}</span>
          </span>
        ))}
      </nav>

      <Space size="middle">
        {/* 全局搜索触发器（原型 .search-trigger 260px，铃铛左侧） */}
        <GlobalSearch />

        {/* 通知铃铛 */}
        <NotificationBell />

        {/* 用户胶囊（原型 .header-user：avatar + 名字 + caret） */}
        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <div className="header-user-pill">
            <Avatar size={30} icon={<UserOutlined />} src={user?.avatar} />
            <span className="uname">{user?.nickname || user?.username}</span>
            <DownOutlined className="ucaret" />
          </div>
        </Dropdown>
      </Space>
    </AntHeader>
  );
};

export default Header;

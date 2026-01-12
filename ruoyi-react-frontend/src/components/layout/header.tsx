import React, { useEffect } from 'react';
import { Layout, Avatar, Dropdown, Space, Divider } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useNoticeStore } from '@/store/noticeStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { getUnreadCount } from '@/lib/noticeApi';
import { useLayout } from '@/store/layoutStore';
import NotificationBell from '@/components/NotificationBell';
import ColorSwitcher from '@/design-system/components/ColorSwitcher';
import GlobalSearch from '@/components/shared/GlobalSearch';
import type { MenuProps } from 'antd';

const { Header: AntHeader } = Layout;

const Header: React.FC = () => {
  const { user, logout } = useAuthStore();
  const { setUnreadCount } = useNoticeStore();
  const { sidebarCollapsed, toggleSidebar, layout } = useLayout();
  const navigate = useNavigate();

  // 初始化 WebSocket 连接
  useWebSocket();

  // 获取未读数量
  useEffect(() => {
    const fetchUnreadCount = async () => {
      try {
        const response = await getUnreadCount();
        setUnreadCount(response.data?.count || 0);
      } catch (error) {
        console.error('获取未读数量失败:', error);
      }
    };

    fetchUnreadCount();
  }, [setUnreadCount]);

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      label: '个人中心',
      icon: <UserOutlined />,
      onClick: () => navigate('/profile'),
    },
    {
      key: 'settings',
      label: '系统设置',
      icon: <SettingOutlined />,
      onClick: () => navigate('/settings'),
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      label: '退出登录',
      icon: <LogoutOutlined />,
      onClick: handleLogout,
    },
  ];

  return (
    <AntHeader
      className="flex items-center justify-between px-6 shadow-lg"
      style={{
        background: 'var(--theme-bg-surface)',
        color: 'var(--theme-text-primary)',
        borderBottom: '1px solid var(--theme-border-primary)',
        transition: 'var(--theme-transition-base)',
      }}
    >
      <div className="flex items-center gap-4">
        {/* 侧边栏切换按钮（仅经典布局和混合布局显示） */}
        {(layout === 'classic' || layout === 'hybrid') && (
          <button
            onClick={toggleSidebar}
            className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            style={{
              color: 'var(--theme-text-secondary)',
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
            }}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 20 20"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M3 5h14M3 10h10M3 15h6"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
            </svg>
          </button>
        )}
        <h1
          className="text-xl font-bold"
          style={{ color: 'var(--theme-text-primary)' }}
        >
          产险湖北数智化平台
        </h1>
      </div>

      <Space size="middle">
        {/* 颜色选择器 */}
        <ColorSwitcher />

        <Divider orientation="vertical" style={{ height: 24 }} />

        {/* 通知铃铛 */}
        <NotificationBell />

        <span className="font-medium" style={{ color: 'var(--theme-text-secondary)' }}>
          欢迎，{user?.nickname || user?.username}
        </span>

        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar
            size="large"
            icon={<UserOutlined />}
            src={user?.avatar}
            className="cursor-pointer shadow-md border-2"
            style={{
              background: 'linear-gradient(135deg, var(--theme-primary-500, #3b82f6) 0%, var(--theme-primary-600, #2563eb) 100%)',
              color: 'var(--theme-text-inverse)',
              borderColor: 'rgba(255, 255, 255, 0.3)',
              transition: 'var(--theme-transition-base)',
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
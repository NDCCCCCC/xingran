import { Layout, Avatar, Dropdown, Space, Divider } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useNoticeStore } from '@/store/noticeStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useLayout } from '@/store/layoutStore';
import NotificationBell from '@/components/NotificationBell';
import ColorSwitcher from '@/design-system/components/ColorSwitcher';
import GlobalSearch from '@/components/shared/GlobalSearch';
import type { MenuProps } from 'antd';

const { Header: AntHeader } = Layout;

import type { FC } from 'react';

const Header: FC = () => {
  const { user, logout } = useAuthStore();
  const unreadCount = useNoticeStore(state => state.unreadCount);
  const { toggleSidebar, layout } = useLayout();
  const navigate = useNavigate();

  // 初始化 WebSocket 连接
  // useWebSocket 内部已处理只连接一次的逻辑（通过模块级单例 + useEffect + subscribe）
  // WebSocket 连接成功后会自动获取未读数量
  useWebSocket();

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
        position: 'relative',
        zIndex: 10,
      }}
    >
      {/* 左侧留空，保持布局平衡 */}
      <div className="flex items-center gap-4">
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
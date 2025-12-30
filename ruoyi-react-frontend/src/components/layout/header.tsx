import React, { useEffect } from 'react';
import { Layout, Avatar, Dropdown, Space } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useNoticeStore } from '@/store/noticeStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { getUnreadCount } from '@/lib/noticeApi';
import NotificationBell from '@/components/NotificationBell';
import type { MenuProps } from 'antd';

const { Header: AntHeader } = Layout;

const Header: React.FC = () => {
  const { user, logout } = useAuthStore();
  const { setUnreadCount } = useNoticeStore();
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
    <AntHeader className="flex items-center justify-between px-6 bg-gradient-to-r from-blue-600 to-blue-700 shadow-lg">
      <div className="text-xl font-bold text-white">
        {/* <span className="text-yellow-300">数智化</span>平台 */}
      </div>

      <Space size="middle">
        {/* 通知铃铛 */}
        <NotificationBell />

        <span className="text-white font-medium">
          欢迎，{user?.nickname || user?.username}
        </span>

        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar
            size="large"
            icon={<UserOutlined />}
            src={user?.avatar}
            className="cursor-pointer bg-gradient-to-br from-blue-400 to-blue-500 hover:from-blue-500 hover:to-blue-600 transition-all duration-300 shadow-md border-2 border-white/30"
          />
        </Dropdown>
      </Space>
    </AntHeader>
  );
};

export default Header;
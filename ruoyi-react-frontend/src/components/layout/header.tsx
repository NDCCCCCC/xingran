import React from 'react';
import { Layout, Avatar, Dropdown, Space, Button } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useAuthStore } from '@/store/authStore';
import type { MenuProps } from 'antd';

const { Header: AntHeader } = Layout;

const Header: React.FC = () => {
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      label: '个人中心',
      icon: <UserOutlined />,
    },
    {
      key: 'settings',
      label: '系统设置',
      icon: <SettingOutlined />,
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
    <AntHeader className="flex items-center justify-between px-6 bg-white border-b border-gray-200 shadow-sm">
      <div className="text-xl font-bold text-gray-800">
        <span className="text-blue-600">RuoYi</span> Next
      </div>

      <Space size="middle">
        <span className="text-gray-600 font-medium">
          欢迎，{user?.nickname || user?.username}
        </span>

        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar
            size="large"
            icon={<UserOutlined />}
            src={user?.avatar}
            className="cursor-pointer bg-gradient-to-br from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 transition-all duration-300 shadow-md"
          />
        </Dropdown>
      </Space>
    </AntHeader>
  );
};

export default Header;
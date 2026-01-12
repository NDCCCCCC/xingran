/**
 * 创新布局组件
 * 创新导航方式 + 模块化面板 + 沉浸式体验
 */

import React, { useState } from 'react';
import { Layout as AntLayout, Card, Row, Col, Button, Tooltip, Space, Divider } from 'antd';
import { useNavigate } from 'react-router-dom';
import ThemeSwitcher from '@/design-system/components/ThemeSwitcher';
import LayoutSwitcher from '@/design-system/components/LayoutSwitcher';
import DensitySwitcher from '@/design-system/components/DensitySwitcher';
import ColorSwitcher from '@/design-system/components/ColorSwitcher';
import { useAuthStore } from '@/store/authStore';
import {
  DashboardOutlined,
  UserOutlined,
  SettingOutlined,
  MonitorOutlined,
  CloudServerOutlined,
  CalendarOutlined,
  FileTextOutlined,
  ApiOutlined,
  AppstoreOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import QuickNav from './shared/QuickNav';

const { Content } = AntLayout;

interface InnovativeLayoutProps {
  children: React.ReactNode;
}

interface SpaceNav {
  key: string;
  title: string;
  icon: React.ReactNode;
  path: string;
  color: string;
  description: string;
}

const spaceNavs: SpaceNav[] = [
  {
    key: 'dashboard',
    title: '仪表盘',
    icon: <DashboardOutlined />,
    path: '/dashboard',
    color: '#3b82f6',
    description: '数据概览与统计',
  },
  {
    key: 'system',
    title: '系统管理',
    icon: <SettingOutlined />,
    path: '/system/user',
    color: '#10b981',
    description: '用户、角色、菜单',
  },
  {
    key: 'monitor',
    title: '系统监控',
    icon: <MonitorOutlined />,
    path: '/monitor/dashboard',
    color: '#f59e0b',
    description: '服务器、日志、任务',
  },
  {
    key: 'network',
    title: '网络设备',
    icon: <CloudServerOutlined />,
    path: '/network/devices',
    color: '#8b5cf6',
    description: '设备、凭证、配置',
  },
  {
    key: 'duty',
    title: '值班管理',
    icon: <CalendarOutlined />,
    path: '/duty/my-duty',
    color: '#ec4899',
    description: '排班、节假日、池',
  },
  {
    key: 'workorder',
    title: '工单管理',
    icon: <FileTextOutlined />,
    path: '/workorder/orders',
    color: '#ef4444',
    description: '工单、分类、统计',
  },
  {
    key: 'notice',
    title: '消息通知',
    icon: <ApiOutlined />,
    path: '/my-notices',
    color: '#14b8a6',
    description: '通知、消息、提醒',
  },
  {
    key: 'profile',
    title: '个人中心',
    icon: <UserOutlined />,
    path: '/profile',
    color: '#6366f1',
    description: '个人信息、设置',
  },
  {
    key: 'settings',
    title: '系统设置',
    icon: <SettingOutlined />,
    path: '/settings',
    color: '#8b5cf6',
    description: '配置、偏好、主题',
  },
];

const InnovativeLayout: React.FC<InnovativeLayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [activeSpace, setActiveSpace] = useState<string>('dashboard');

  const handleSpaceClick = (space: SpaceNav) => {
    setActiveSpace(space.key);
    navigate(space.path);
  };

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  return (
    <div
      className="h-screen flex"
      style={{
        background: 'var(--theme-bg-secondary)',
        transition: 'all var(--theme-transition-slow)',
      }}
    >
      {/* 左侧空间导航 */}
      <div
        style={{
          width: '80px',
          background: 'var(--theme-bg-surface)',
          borderRight: '1px solid var(--theme-border-primary)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          paddingY: '16px',
          gap: '12px',
          transition: 'all var(--theme-transition-base)',
        }}
      >
        <div
          style={{
            fontSize: '24px',
            fontWeight: 'bold',
            marginBottom: '16px',
            background: 'linear-gradient(135deg, var(--theme-primary-500, #3b82f6) 0%, var(--theme-primary-600, #2563eb) 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text',
          }}
        >
          数智
        </div>

        {spaceNavs.map((space) => (
          <Tooltip key={space.key} title={space.description} placement="right">
            <div
              onClick={() => handleSpaceClick(space)}
              style={{
                width: '48px',
                height: '48px',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                background: activeSpace === space.key ? space.color : 'transparent',
                color: activeSpace === space.key ? '#fff' : 'var(--theme-text-secondary)',
                transition: 'all var(--theme-transition-base)',
                fontSize: '20px',
              }}
              onMouseEnter={(e) => {
                if (activeSpace !== space.key) {
                  e.currentTarget.style.background = space.color + '20';
                  e.currentTarget.style.transform = 'scale(1.1)';
                }
              }}
              onMouseLeave={(e) => {
                if (activeSpace !== space.key) {
                  e.currentTarget.style.background = 'transparent';
                  e.currentTarget.style.transform = 'scale(1)';
                }
              }}
            >
              {space.icon}
            </div>
          </Tooltip>
        ))}
      </div>

      {/* 主内容区 */}
      <div
        className="flex-1 overflow-auto"
        style={{
          padding: '24px',
          maxWidth: '1600px',
          margin: '0 auto',
          width: '100%',
        }}
      >
        {/* 控制栏 */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '16px',
            padding: '12px 16px',
            borderRadius: 'var(--theme-radius-lg)',
            background: 'var(--theme-bg-surface)',
            border: '1px solid var(--theme-border-primary)',
            boxShadow: 'var(--theme-shadow-sm)',
            transition: 'all var(--theme-transition-base)',
          }}
        >
          <Space size="middle">
            <span
              style={{
                fontSize: '14px',
                fontWeight: 500,
                color: 'var(--theme-text-secondary)',
              }}
            >
              产险湖北数智化平台
            </span>
          </Space>

          <Space size="middle" separator={<Divider orientation="vertical" style={{ margin: 0 }} />}>
            {/* 主题切换器 */}
            <ThemeSwitcher />

            {/* 布局切换器 */}
            <LayoutSwitcher />

            {/* 密度模式切换器 */}
            <DensitySwitcher />

            {/* 颜色切换器 */}
            <ColorSwitcher />

            <span
              style={{
                fontSize: '14px',
                color: 'var(--theme-text-secondary)',
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
                  color: 'var(--theme-text-secondary)',
                  transition: 'all var(--theme-transition-base)',
                }}
              />
            </Tooltip>
          </Space>
        </div>

        {/* 快速导航 */}
        <QuickNav />

        {/* 内容卡片 */}
        <Card
          style={{
            borderRadius: 'var(--theme-radius-xl)',
            border: '1px solid var(--theme-border-primary)',
            background: 'var(--theme-bg-surface)',
            boxShadow: 'var(--theme-shadow-sm)',
            transition: 'all var(--theme-transition-base)',
          }}
          styles={{
            body: {
              padding: '24px',
            }
          }}
        >
          {children}
        </Card>
      </div>
    </div>
  );
};

export default InnovativeLayout;

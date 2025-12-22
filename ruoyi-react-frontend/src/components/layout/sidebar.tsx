import React, { useState } from 'react';
import { Layout, Menu } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  MenuOutlined,
  SettingOutlined,
  FileTextOutlined,
  BellOutlined,
  MonitorOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';

const { Sider } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

function getItem(
  label: React.ReactNode,
  key: React.Key,
  icon?: React.ReactNode,
  children?: MenuItem[],
): MenuItem {
  return {
    key,
    icon,
    children,
    label,
  } as MenuItem;
}

const menuItems: MenuItem[] = [
  getItem('仪表盘', '/dashboard', <DashboardOutlined />),
  getItem('系统管理', '/system', <SettingOutlined />, [
    getItem('用户管理', '/system/user', <UserOutlined />),
    getItem('角色管理', '/system/role', <TeamOutlined />),
    getItem('菜单管理', '/system/menu', <MenuOutlined />),
    getItem('部门管理', '/system/dept', <TeamOutlined />),
    getItem('岗位管理', '/system/posts', <UserOutlined />),
    getItem('字典管理', '/system/dicts', <FileTextOutlined />),
    getItem('参数配置', '/system/configs', <SettingOutlined />),
    getItem('通知公告', '/system/notices', <BellOutlined />),
  ]),
  getItem('系统监控', '/monitor', <MonitorOutlined />, [
    getItem('在线用户', '/monitor/online', <UserOutlined />),
    getItem('操作日志', '/monitor/operlog', <FileTextOutlined />),
    getItem('登录日志', '/monitor/loginlog', <FileTextOutlined />),
  ]),
];

const Sidebar: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    navigate(e.key);
  };

  // 获取当前选中的菜单项
  const getSelectedKeys = () => {
    return [location.pathname];
  };

  // 获取当前展开的菜单项
  const getOpenKeys = () => {
    const pathSnippets = location.pathname.split('/').filter((i) => i);
    if (pathSnippets.length >= 2) {
      return [`/${pathSnippets[0]}/${pathSnippets[1]}`];
    }
    return [];
  };

  return (
    <Sider
      collapsible
      collapsed={collapsed}
      onCollapse={(value) => setCollapsed(value)}
      className="bg-gradient-to-b from-gray-900 to-gray-800 border-r border-gray-700 shadow-xl"
      width={256}
      trigger={null}
    >
      <div className="h-16 flex items-center justify-center border-b border-gray-700">
        <h1
          className={`text-xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent ${
            collapsed ? 'hidden' : 'block'
          }`}
        >
          RuoYi Next
        </h1>
        {collapsed && (
          <span className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent">R</span>
        )}
      </div>

      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={getSelectedKeys()}
        defaultOpenKeys={getOpenKeys()}
        items={menuItems}
        onClick={handleMenuClick}
        className="border-r-0 px-2 py-4"
        style={{
          background: 'transparent',
          fontSize: '14px'
        }}
      />
    </Sider>
  );
};

export default Sidebar;
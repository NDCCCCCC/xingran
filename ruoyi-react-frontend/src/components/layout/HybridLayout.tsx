/**
 * 混合式布局组件
 * 可折叠侧边栏 + 多标签页系统
 */

import { useEffect, useCallback, useState } from 'react';
import { Layout as AntLayout } from 'antd';
import { useLocation } from 'react-router-dom';
import { useLayout } from '@/store/layoutStore';
import { useTabs } from '@/store/tabsStore';
import { getAllUserMenus } from '@/lib/menuApi';
import Header from './header';
import Sidebar from './sidebar';
import TabBar from './shared/TabBar';
import type { Menu as MenuType } from '@/types';
import { getSpecialPathTitle, matchDynamicRouteTitle } from '@/constants/pageTitles';

const { Content } = AntLayout;

import type { FC, ReactNode } from 'react';

interface HybridLayoutProps {
  children: ReactNode;
}

const HybridLayout: FC<HybridLayoutProps> = ({ children }) => {
  const { sidebarCollapsed } = useLayout();
  const { addTab } = useTabs();
  const [allMenus, setAllMenus] = useState<MenuType[]>([]); // 完整菜单数据（包含隐藏菜单，用于标签标题）
  const location = useLocation();

  // 获取完整菜单数据（包括隐藏菜单，用于标签标题）
  useEffect(() => {
    const fetchAllMenus = async () => {
      try {
        const all = await getAllUserMenus();
        setAllMenus(all);
      } catch (error) {
        console.error('获取完整菜单失败:', error);
      }
    };
    fetchAllMenus();
  }, []);

  // 根据路径从菜单数据中获取标题
  const getTitleByPath = useCallback((path: string): string => {
    // 优先级 1: 检查特殊路径映射
    const specialTitle = getSpecialPathTitle(path);
    if (specialTitle) {
      return specialTitle;
    }

    // 优先级 2: 检查动态路由模式（详情页）
    const dynamicTitle = matchDynamicRouteTitle(path);
    if (dynamicTitle) {
      return dynamicTitle;
    }

    // 优先级 3: 在完整菜单数据中查找对应的菜单名称（包括隐藏菜单）
    // 递归查找函数
    const findMenuByPath = (menuList: MenuType[], targetPath: string): string => {
      // 规范化目标路径（去掉前导斜杠）
      const normalizedTargetPath = targetPath.startsWith('/') ? targetPath.slice(1) : targetPath;

      for (const menu of menuList) {
        if (menu.menuType === 'F') continue; // 跳过按钮类型，但包含隐藏菜单

        // 获取菜单路径
        const menuPath = menu.path || '';

        // 规范化菜单路径
        const normalizedMenuPath = menuPath.startsWith('/') ? menuPath.slice(1) : menuPath;

        // 精确匹配菜单路径
        if (normalizedMenuPath === normalizedTargetPath) {
          return menu.menuName;
        }

        // 递归查找子菜单
        if (menu.children && menu.children.length > 0) {
          const found = findMenuByPath(menu.children, targetPath);
          if (found) return found;
        }
      }
      return '';
    };

    // 使用菜单数据查找标题
    const menuTitle = findMenuByPath(allMenus, path);
    if (menuTitle) {
      return menuTitle;
    }

    // 优先级 4: 未找到则返回路径的最后一段作为标题
    const segments = path.split('/').filter(Boolean);
    return segments[segments.length - 1] || '页面';
  }, [allMenus]);

  // 自动添加标签页
  useEffect(() => {
    // 排除登录页
    if (location.pathname !== '/login') {
      const isDashboard = location.pathname === '/dashboard' || location.pathname === '/';

      addTab({
        key: location.pathname,
        title: getTitleByPath(location.pathname),
        path: location.pathname,
        closable: !isDashboard,  // 仪表盘不可关闭
        pinned: isDashboard,     // 仪表盘为固定标签
      });
    }
  }, [location.pathname, addTab, getTitleByPath]);

  return (
    <AntLayout className="h-screen" style={{ background: 'var(--theme-bg-secondary)' }}>
      <Sidebar />
      <AntLayout className="h-full" style={{ background: 'var(--theme-bg-secondary)', position: 'relative' }}>
        <Header />
        <TabBar />
        <Content
          className="overflow-auto"
          style={{
            padding: sidebarCollapsed ? '16px' : '24px',
            transition: 'padding var(--theme-transition-base)',
          }}
        >
          <div
            style={{
              background: 'var(--theme-bg-surface)',
              borderRadius: 'var(--theme-radius-lg)',
              padding: '24px',
              minHeight: '100%',
              transition: 'all var(--theme-transition-base)',
            }}
          >
            {children}
          </div>
        </Content>
      </AntLayout>
    </AntLayout>
  );
};

export default HybridLayout;

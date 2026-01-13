/**
 * 混合式布局组件
 * 可折叠侧边栏 + 多标签页系统
 */

import { useEffect, useCallback } from 'react';
import { Layout as AntLayout } from 'antd';
import { useLocation } from 'react-router-dom';
import { useLayout } from '@/store/layoutStore';
import { useTabs } from '@/store/tabsStore';
import { useMenuStore } from '@/store/menuStore';
import Header from './header';
import Sidebar from './sidebar';
import TabBar from './shared/TabBar';
import type { Menu as MenuType } from '@/types';

const { Content } = AntLayout;

import type { FC, ReactNode } from 'react';

interface HybridLayoutProps {
  children: ReactNode;
}

const HybridLayout: FC<HybridLayoutProps> = ({ children }) => {
  const { sidebarCollapsed } = useLayout();
  const { addTab } = useTabs();
  const { menus } = useMenuStore();
  const location = useLocation();

  // 根据路径从菜单数据中获取标题
  const getTitleByPath = useCallback((path: string): string => {
    // 特殊路径处理（优先处理）
    const specialPathMap: Record<string, string> = {
      '/dashboard': '仪表盘',
      '/': '首页',
    };

    if (specialPathMap[path]) {
      return specialPathMap[path];
    }

    // 在菜单数据中查找对应的菜单名称
    // 递归查找函数，需要与sidebar.tsx中的convertToMenuItem使用相同的路径构建逻辑
    const findMenuByPath = (menuList: MenuType[], targetPath: string, parentPath: string = ''): string => {
      for (const menu of menuList) {
        if (menu.menuType === 'F' || menu.visible !== 1) continue;

        // 构建完整路径（与 sidebar.tsx convertToMenuItem 逻辑完全一致）
        let menuPath = menu.path || '';
        if (menuPath && !menuPath.startsWith('/') && parentPath) {
          if (!menuPath.startsWith(parentPath + '/')) {
            menuPath = `${parentPath}/${menuPath}`;
          }
        }

        // 规范化路径用于比较（统一格式）
        const normalizeForCompare = (p: string): string => {
          if (!p) return '';
          return p.startsWith('/') ? p.slice(1) : p;
        };

        const normalizedMenuPath = normalizeForCompare(menuPath);
        const normalizedTargetPath = normalizeForCompare(targetPath);

        // 匹配所有类型的菜单项（包括目录M和菜单C）
        // 精确匹配：menuPath === targetPath 或 /menuPath === targetPath
        if (normalizedMenuPath === normalizedTargetPath ||
            menuPath === targetPath ||
            (menuPath && `/${menuPath}` === targetPath)) {
          return menu.menuName;
        }

        // 递归查找子菜单，传递构建后的路径作为parentPath
        if (menu.children && menu.children.length > 0) {
          const found = findMenuByPath(menu.children, targetPath, menuPath);
          if (found) return found;
        }
      }
      return '';
    };

    // 尝试多种路径格式查找
    const pathVariants = [
      path,                           // 原始路径: /system/user
      path.startsWith('/') ? path.slice(1) : path,  // 去掉前导斜杠: system/user
    ];

    for (const pathVariant of pathVariants) {
      const menuTitle = findMenuByPath(menus, pathVariant);
      if (menuTitle) {
        return menuTitle;
      }
    }

    // 未找到则返回路径的最后一段作为标题
    const segments = path.split('/').filter(Boolean);
    return segments[segments.length - 1] || '页面';
  }, [menus]);

  // 自动添加标签页
  useEffect(() => {
    // 排除登录页
    if (location.pathname !== '/login') {
      addTab({
        key: location.pathname,
        title: getTitleByPath(location.pathname),
        path: location.pathname,
        closable: true,
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

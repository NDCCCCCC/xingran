import { useMemo } from 'react';
import type { FC } from 'react';
import { Breadcrumb } from 'antd';
import { useLocation, Link } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import type { Menu } from '@/types';

// 递归查找菜单路径对应的菜单项
const findMenuByPath = (menus: Menu[], path: string): Menu | null => {
  for (const menu of menus) {
    // 匹配路径
    if (menu.path === path || menu.path === `/${path}`) {
      return menu;
    }
    // 递归查找子菜单
    if (menu.children && menu.children.length > 0) {
      const found = findMenuByPath(menu.children, path);
      if (found) return found;
    }
  }
  return null;
};

// 构建面包屑路径
const buildBreadcrumbPath = (menus: Menu[], pathSnippets: string[]): Array<{ path: string; name: string }> => {
  const result: Array<{ path: string; name: string }> = [];
  const dashboardMenu = menus.find(m => m.path === '/dashboard');

  // 处理仪表盘
  if (pathSnippets[0] === 'dashboard') {
    result.push({ path: '/dashboard', name: dashboardMenu?.menuName || '仪表盘' });
    return result;
  }

  // 处理系统管理模块
  if (pathSnippets[0] === 'system') {
    const systemMenu = menus.find(m => m.path === '/system');
    result.push({ path: '/system', name: systemMenu?.menuName || '系统管理' });

    // 查找子菜单
    if (pathSnippets[1]) {
      const childPath = `/system/${pathSnippets[1]}`;
      const childMenu = findMenuByPath(menus, childPath);
      if (childMenu) {
        result.push({ path: childPath, name: childMenu.menuName });
      }
    }
  }

  // 处理系统监控模块
  if (pathSnippets[0] === 'monitor') {
    const monitorMenu = menus.find(m => m.path === '/monitor');
    result.push({ path: '/monitor', name: monitorMenu?.menuName || '系统监控' });

    // 查找子菜单
    if (pathSnippets[1]) {
      const childPath = `/monitor/${pathSnippets[1]}`;
      const childMenu = findMenuByPath(menus, childPath);
      if (childMenu) {
        result.push({ path: childPath, name: childMenu.menuName });
      }
    }
  }

  return result;
};

const BreadcrumbComponent: FC = () => {
  const location = useLocation();
  const { menus } = useMenuStore();
  const pathSnippets = location.pathname.split('/').filter((i) => i);

  const breadcrumbItems = useMemo(() => {
    const breadcrumbPath = buildBreadcrumbPath(menus, pathSnippets);

    return breadcrumbPath.map((item, index) => ({
      key: item.path,
      title: index === breadcrumbPath.length - 1 ? (
        <span>{item.name}</span>
      ) : (
        <Link to={item.path}>{item.name}</Link>
      ),
    }));
  }, [menus, pathSnippets]);

  // 如果没有面包屑项，不显示
  if (breadcrumbItems.length === 0) {
    return null;
  }

  return (
    <div
      className="px-6 py-3 border-b"
      style={{
        background: 'var(--theme-bg-surface)',
        borderColor: 'var(--theme-border-primary)',
      }}
    >
      <Breadcrumb
        items={breadcrumbItems}
        className="text-sm"
        style={{
          color: 'var(--theme-text-secondary)',
        }}
      />
    </div>
  );
};

export default BreadcrumbComponent;
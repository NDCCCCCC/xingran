import { useState, useEffect, useMemo, useRef } from 'react';
import { Layout, Menu, Spin, message } from 'antd';
import type { MenuProps } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import { useLayoutStore } from '@/store/layoutStore';
import type { Menu as MenuType } from '@/types';
import {
  buildMenuPathMap,
  isSameTopLevelMenu,
  getMenuLevel,
  isSecondLevelMenu,
  isTopLevelMenu,
} from './sidebar-helper';
import { getIconComponent } from '@/utils/iconUtils';

const { Sider } = Layout;

type MenuItem = NonNullable<MenuProps['items']>[number];

// ========================================
// 获取图标组件 - 使用 iconUtils 中的函数
// ========================================
// 直接从后端 API 获取的图标名称（如 UserOutlined、SettingOutlined 等）
// 通过 iconUtils 中的 getIconComponent 函数转换为 React 组件
function getIcon(iconName?: string | null): React.ReactNode {
  return getIconComponent(iconName);
}

// 将后端菜单数据转换为 Ant Design Menu 组件所需的格式
function convertToMenuItem(menu: MenuType, parentPath: string = ''): MenuItem {
  const icon = getIcon(menu.icon);

  // 只显示目录(M)和菜单(C)类型，不显示按钮(F)类型
  if (menu.menuType === 'F') {
    return null as any;
  }

  // 隐藏的菜单（visible=0）不显示
  if (menu.visible !== 1) {
    return null as any;
  }

  const isDisabled = menu.status !== 0; // 停用的菜单

  // 构建完整路径（与 buildMenuPathMap 逻辑一致）
  let menuPath = menu.path || '';
  if (menuPath && !menuPath.startsWith('/') && parentPath) {
    if (!menuPath.startsWith(parentPath + '/')) {
      menuPath = `${parentPath}/${menuPath}`;
    }
  }

  // 生成菜单key（与 buildMenuPathMap 逻辑一致）
  const menuKey = menuPath || menu.id;

  // 如果有子菜单，递归转换
  if (menu.children && menu.children.length > 0) {
    const validChildren = menu.children
      .filter(child => child.menuType !== 'F' && child.visible === 1)
      .map(child => convertToMenuItem(child, menuPath))
      .filter((item): item is MenuItem => item !== null);

    // 如果有正常子菜单，或者是父级目录，都显示
    if (validChildren.length > 0 || menu.menuType === 'M') {
      return {
        key: menuKey,
        icon,
        label: menu.menuName,
        children: validChildren.length > 0 ? validChildren : undefined,
        disabled: isDisabled,
        style: isDisabled ? { opacity: 0.5 } : undefined,
      };
    }
  }

  // 显示所有可见的菜单（包括停用的）
  return {
    key: menuKey,
    icon,
    label: menu.menuName,
    disabled: isDisabled,
    style: isDisabled ? { opacity: 0.5 } : undefined,
  };
}

const Sidebar = () => {
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const expectedOpenKeysRef = useRef<string[] | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { menus, loading, fetchMenus } = useMenuStore();
  const { sidebarCollapsed, toggleSidebar } = useLayoutStore();

  // 组件加载时获取菜单
  useEffect(() => {
    if (menus.length === 0) {
      fetchMenus();
    }
  }, []);

  // 将后端菜单数据转换为 Ant Design Menu 组件所需的格式
  const menuItems: MenuItem[] = useMemo(() => {
    return menus
      .filter(menu => menu.menuType !== 'F' && menu.visible === 1)
      .map(menu => convertToMenuItem(menu))
      .filter((item): item is MenuItem => item !== null);
  }, [menus]);

  // ========================================
  // 新增：构建菜单路径映射表
  // ========================================
  const menuPathMap = useMemo(() => {
    return buildMenuPathMap(menus);
  }, [JSON.stringify(menus)]);

  // ========================================
  // 辅助函数：构建完整的父级菜单链
  // ========================================
  /**
   * 构建完整的父级菜单链
   * 确保展开到当前菜单时，所有父级菜单也都展开
   *
   * 注意：只返回父级菜单的 key，不包含当前页面路径
   * 因为当前页面可能是不可展开的菜单项（menuType === 'C'）
   */
  function buildParentKeyChain(
    currentPath: string,
    menuPathMap: Map<string, any>
  ): string[] {
    const normalizePath = (p: string): string => {
      return p.startsWith('/') ? p.slice(1) : p;
    };

    const normalizedPathname = normalizePath(currentPath);
    const menuInfo = menuPathMap.get(normalizedPathname);

    if (!menuInfo) {
      return [];
    }

    const result: string[] = [];

    // 不添加当前菜单的 key，只添加父级菜单
    // result.push(normalizedPathname);

    // 如果是二级菜单，添加其父级（一级）菜单
    if (menuInfo.level === 2 && menuInfo.topLevel) {
      result.push(menuInfo.topLevel);
    }

    // 如果是三级菜单，添加其父级（二级）菜单和祖父级（一级）菜单
    if (menuInfo.level === 3) {
      if (menuInfo.secondLevel) {
        result.push(menuInfo.secondLevel); // 二级
      }
      if (menuInfo.topLevel) {
        result.push(menuInfo.topLevel); // 一级
      }
    }

    return result;
  }

  // ========================================
  // 路由变化时更新展开的菜单（防止循环刷新的关键）
  // ========================================
  useEffect(() => {
    if (!menuPathMap.size || !location.pathname) return;

    // 构建应该展开的父级菜单链
    const parentKeys = buildParentKeyChain(location.pathname, menuPathMap);

    // 设置预期值，标记这是导航触发的变化
    expectedOpenKeysRef.current = parentKeys;
    setOpenKeys(parentKeys);

    // 清除预期值标记（给足够的时间让状态更新完成）
    setTimeout(() => {
      expectedOpenKeysRef.current = null;
    }, 100);
  }, [location.pathname, menuPathMap]);

  // ========================================
  // 菜单点击处理
  // ========================================
  const handleMenuClick: MenuProps['onClick'] = (e) => {
    // 在菜单数据中查找对应的菜单
    const findMenuByPath = (menuList: MenuType[], key: string, parentPath: string = ''): MenuType | null => {
      for (const menu of menuList) {
        let menuPath = menu.path || '';
        if (menuPath && !menuPath.startsWith('/') && parentPath) {
          if (!menuPath.startsWith(parentPath + '/')) {
            menuPath = `${parentPath}/${menuPath}`;
          }
        }

        const menuKey = menuPath || menu.id;

        // 检查是否匹配
        if ((menuKey === key || menu.id === key) && menu.visible === 1) {
          return menu;
        }

        // 递归查找子菜单
        if (menu.children) {
          const found = findMenuByPath(menu.children, key, menuPath);
          if (found) return found;
        }
      }
      return null;
    };

    const clickedMenu = findMenuByPath(menus, e.key);

    // 检查菜单是否停用
    if (clickedMenu && clickedMenu.status !== 0) {
      message.warning('该功能已停用，暂时无法访问');
      return;
    }

    // 只有菜单类型（C）才执行导航，目录类型（M）不导航
    if (clickedMenu && clickedMenu.menuType === 'C') {
      // 确保路径以 / 开头
      const path = e.key.startsWith('/') ? e.key : `/${e.key}`;
      navigate(path);
    }
  };

  // ========================================
  // 处理菜单展开/收起 - 手风琴效果（支持一级和二级菜单）
  // ========================================
  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    // 检查是否是导航触发的预期变化（防止循环刷新）
    if (expectedOpenKeysRef.current) {
      const expected = expectedOpenKeysRef.current;
      const isExpectedChange =
        keys.length === expected.length &&
        keys.every(k => expected.includes(k));

      if (isExpectedChange) {
        setOpenKeys(keys);
        return;
      }
    }

    // 找出新展开的菜单（keys 中有但 openKeys 中没有的）
    const latestOpenKey = keys.find((key) => !openKeys.includes(key));

    if (latestOpenKey) {
      const menuLevel = getMenuLevel(latestOpenKey, menuPathMap);

      if (menuLevel === 1 || latestOpenKey.startsWith('top_')) {
        // ========================================
        // 一级菜单：应用手风琴效果
        // ========================================
        // 1. 保留新展开的一级菜单
        // 2. 保留所有非一级菜单（二级和三级菜单）
        // 3. 移除其他所有一级菜单
        const nonTopLevelKeys = keys.filter(key => !isTopLevelMenu(key, menuPathMap));
        const result = [latestOpenKey, ...nonTopLevelKeys];
        setOpenKeys(result);

      } else if (menuLevel === 2) {
        // ========================================
        // 二级菜单：应用手风琴效果（仅在同一一级菜单下）
        // ========================================
        // 1. 找出当前二级菜单所属的一级菜单
        const currentTopLevel = menuPathMap.get(latestOpenKey)?.topLevel;

        if (!currentTopLevel) {
          setOpenKeys([...keys]);
          return;
        }

        // 2. 保留所有一级菜单
        const topLevelKeys = keys.filter(key => isTopLevelMenu(key, menuPathMap));

        // 3. 找出所有与新展开的二级菜单属于同一一级菜单的其他已展开二级菜单
        const sameTopLevelSecondKeys = openKeys.filter(key => {
          if (!isSecondLevelMenu(key, menuPathMap)) return false;
          return isSameTopLevelMenu(key, latestOpenKey, menuPathMap);
        });

        // 4. 需要收起的这些同级二级菜单
        const keysToClose = sameTopLevelSecondKeys;

        // 5. 构建新的openKeys：
        //    - 保留所有一级菜单
        //    - 保留不属于同一一级菜单的其他二级菜单
        //    - 保留新展开的二级菜单
        //    - 保留所有三级菜单
        //    - 移除需要收起的同级二级菜单
        const result = [
          ...topLevelKeys,
          ...keys.filter(key =>
            !keysToClose.includes(key) &&
            (isSecondLevelMenu(key, menuPathMap) || key === latestOpenKey)
          )
        ];
        setOpenKeys(result);

      } else if (menuLevel === 3) {
        // ========================================
        // 三级菜单：不应用手风琴效果
        // ========================================
        setOpenKeys([...keys]);

      } else {
        // ========================================
        // 未知层级：直接应用（降级处理）
        // ========================================
        setOpenKeys([...keys]);
      }
    } else {
      // ========================================
      // 收起操作：直接应用
      // ========================================
      setOpenKeys([...keys]);
    }
  };

  // 获取当前选中的菜单项
  // 注意：菜单 key 可能不带前导斜杠（如 ops/server-rooms）
  // 而 location.pathname 带前导斜杠（如 /ops/server-rooms）
  // 需要返回匹配菜单 key 的格式
  const getSelectedKeys = () => {
    // 去掉前导斜杠，与菜单 key 保持一致
    const normalizedPath = location.pathname.startsWith('/')
      ? location.pathname.slice(1)
      : location.pathname;

    // 返回两种格式，确保能匹配到
    return [location.pathname, normalizedPath];
  };

  return (
    <Sider
      collapsible
      collapsed={sidebarCollapsed}
      onCollapse={toggleSidebar}
      width={280}
      trigger={null}
      style={{
        background: 'var(--theme-bg-secondary)',
        borderRight: '1px solid var(--theme-border-primary)',
        transition: 'all var(--theme-transition-base)',
      }}
    >
      <div
        className="h-16 flex items-center relative"
        style={{
          borderBottom: '1px solid var(--theme-border-primary)',
          paddingLeft: sidebarCollapsed ? '0' : '56px',
          transition: 'padding-left var(--theme-transition-base)',
        }}
      >
        {/* 收缩按钮 */}
        <button
          onClick={toggleSidebar}
          className="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors absolute"
          style={{
            color: 'var(--theme-text-secondary)',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            left: '16px',
          }}
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 20 20"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            style={{
              transition: 'transform var(--theme-transition-base)',
              transform: sidebarCollapsed ? 'rotate(180deg)' : 'rotate(0deg)',
            }}
          >
            <path
              d="M3 5h14M3 10h10M3 15h6"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            />
          </svg>
        </button>

        <h1
          className={`text-xl font-bold ${
            sidebarCollapsed ? 'hidden' : 'block'
          }`}
          style={{
            background: 'linear-gradient(135deg, var(--theme-primary-500, #3b82f6) 0%, var(--theme-primary-600, #2563eb) 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text',
          }}
        >
          数字化平台
        </h1>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin size="large" />
        </div>
      ) : (
        <Menu
          mode="inline"
          selectedKeys={getSelectedKeys()}
          openKeys={openKeys}
          onOpenChange={handleOpenChange}
          items={menuItems}
          onClick={handleMenuClick}
          className="border-r-0 px-2 py-4"
          style={{
            background: 'transparent',
            fontSize: '14px',
            color: 'var(--theme-text-primary)',
          }}
        />
      )}
    </Sider>
  );
};

export default Sidebar;

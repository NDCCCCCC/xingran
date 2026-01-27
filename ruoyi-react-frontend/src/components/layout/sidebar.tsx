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

const getIcon = (iconName?: string | null): React.ReactNode => {
  return getIconComponent(iconName);
};

const convertToMenuItem = (menu: MenuType, parentPath: string = ''): MenuItem => {
  const icon = getIcon(menu.icon);

  if (menu.menuType === 'F' || menu.visible !== 1) {
    return null as any;
  }

  const isDisabled = menu.status !== 0;
  let menuPath = menu.path || '';

  if (menuPath && !menuPath.startsWith('/') && parentPath) {
    if (!menuPath.startsWith(parentPath + '/')) {
      menuPath = `${parentPath}/${menuPath}`;
    }
  }

  const menuKey = menu.id;

  if (menu.children && menu.children.length > 0) {
    const validChildren = menu.children
      .filter(child => child.menuType !== 'F' && child.visible === 1)
      .map(child => convertToMenuItem(child, menuPath))
      .filter((item): item is MenuItem => item !== null);

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

  return {
    key: menuKey,
    icon,
    label: menu.menuName,
    disabled: isDisabled,
    style: isDisabled ? { opacity: 0.5 } : undefined,
  };
};

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
    // 在菜单数据中查找对应的菜单（通过 ID）
    const findMenuById = (menuList: MenuType[], id: string): MenuType | null => {
      for (const menu of menuList) {
        if (menu.id === id) {
          return menu;
        }
        // 递归查找子菜单
        if (menu.children) {
          const found = findMenuById(menu.children, id);
          if (found) return found;
        }
      }
      return null;
    };

    const clickedMenu = findMenuById(menus, e.key);

    // 检查菜单是否停用
    if (clickedMenu && clickedMenu.status !== 0) {
      message.warning('该功能已停用，暂时无法访问');
      return;
    }

    // 只有菜单类型（C）才执行导航，目录类型（M）不导航
    if (clickedMenu && clickedMenu.menuType === 'C') {
      // 构建完整的导航路径（考虑父路径）
      const buildFullPath = (menu: MenuType, parentPath: string = ''): string => {
        const menuPath = menu.path || '';
        if (!menuPath) return parentPath;

        // 如果已经是绝对路径，直接返回
        if (menuPath.startsWith('/')) {
          return menuPath;
        }

        // 如果没有父路径，返回菜单路径
        if (!parentPath) {
          return `/${menuPath}`;
        }

        // 拼接父路径和菜单路径
        return `${parentPath}/${menuPath}`;
      };

      // 从 menuPathMap 中查找该菜单的完整路径
      const menuInfo = menuPathMap.get(clickedMenu.id);

      // 调试日志
      console.log('[Sidebar MenuClick] 菜单点击:', {
        menuName: clickedMenu.menuName,
        menuId: clickedMenu.id,
        menuPath: clickedMenu.path,
        menuInfo: menuInfo,
        fullPathFromMap: menuInfo?.fullPath,
        fallbackPath: buildFullPath(clickedMenu),
      });

      const fullPath = menuInfo?.fullPath || buildFullPath(clickedMenu);

      // 确保路径以 / 开头
      const navigationPath = fullPath.startsWith('/') ? fullPath : `/${fullPath}`;
      console.log('[Sidebar MenuClick] 导航到:', navigationPath);
      navigate(navigationPath);
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
  // 使用 menuPathMap 来查找，因为其中包含了正确构建的 fullPath
  const getSelectedKeys = () => {
    if (!location.pathname) return [];

    // 尝试多种路径格式匹配
    const pathsToTry = [
      location.pathname,                    // /system/user-center
      location.pathname.startsWith('/') ? location.pathname.slice(1) : location.pathname,  // system/user-center
    ];

    for (const path of pathsToTry) {
      const menuInfo = menuPathMap.get(path);
      if (menuInfo) {
        // 通过 menuPathMap 的 key 反向查找菜单 ID
        // menuPathMap 中存储了 path -> menuInfo 的映射
        // 但我们需要返回菜单 ID（即菜单的 key）
        // 所以需要从 menus 中查找对应的菜单

        // 遍历 menus 找到 fullPath 匹配的菜单
        const findMenuByFullPath = (menuList: MenuType[], fullPath: string): string | null => {
          for (const menu of menuList) {
            if (menu.menuType === 'F') continue;

            // 检查当前菜单的路径
            const menuPath = menu.path || '';
            let computedPath = menuPath;

            // 如果是子菜单，需要拼接父路径
            const buildFullPath = (m: MenuType, parent: string = ''): string => {
              const p = m.path || '';
              if (!p) return parent;
              if (p.startsWith('/')) return p.slice(1); // 去掉前导斜杠
              if (!parent) return p;
              return p.startsWith(parent + '/') ? p : `${parent}/${p}`;
            };

            computedPath = buildFullPath(menu, '');

            // 递归处理子菜单
            if (menu.children) {
              for (const child of menu.children) {
                const childFullPath = buildFullPath(child, computedPath);
                if (childFullPath === fullPath || child.path === fullPath) {
                  return child.id;
                }

                // 检查三级菜单
                if (child.children) {
                  for (const grandChild of child.children) {
                    const grandChildFullPath = buildFullPath(grandChild, childFullPath);
                    if (grandChildFullPath === fullPath || grandChild.path === fullPath) {
                      return grandChild.id;
                    }
                  }
                }
              }
            }

            // 检查一级菜单
            if (computedPath === fullPath || menuPath === fullPath) {
              return menu.id;
            }
          }
          return null;
        };

        const menuId = findMenuByFullPath(menus, path);
        if (menuId) return [menuId];
      }
    }

    return [];
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

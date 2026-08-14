import { useState, useEffect, useMemo, useRef } from "react";
import { Layout, Menu, Spin, App } from "antd";
import type { MenuProps } from "antd";
import { useNavigate, useLocation } from "react-router-dom";
import { useMenuStore } from "@/store/menuStore";
import { useLayoutStore } from "@/store/layoutStore";
import type { Menu as MenuType } from "@/types";
import {
  buildMenuPathMap,
  isSameTopLevelMenu,
  getMenuLevel,
  isSecondLevelMenu,
  isTopLevelMenu,
} from "./sidebar-helper";
import { getIconComponent } from "@/utils/iconUtils";
import { buildFullPath, findMenuById, findMenuByFullPath } from "./sidebar.utils";
import {
  DEFAULT_SIDEBAR_WIDTH,
  HEADER_HEIGHT,
  COLLAPSE_BUTTON_LEFT,
  MENU_FONT_SIZE,
  NAVIGATION_DELAY,
} from "./sidebar.constants";
import "@/design-system/themes/theme-styles.css";

const { Sider } = Layout;

type MenuItem = NonNullable<MenuProps["items"]>[number];

const getIcon = (iconName?: string | null): React.ReactNode => getIconComponent(iconName);

const convertToMenuItem = (menu: MenuType, parentPath: string = ""): MenuItem => {
  const icon = getIcon(menu.icon);

  if (menu.menuType === "F" || menu.visible !== 1) {
    return null;
  }

  const isDisabled = menu.status !== 0;
  let menuPath = menu.path || "";

  if (menuPath && !menuPath.startsWith("/") && parentPath) {
    if (!menuPath.startsWith(parentPath + "/")) {
      menuPath = `${parentPath}/${menuPath}`;
    }
  }

  const menuKey = menu.id;

  if (menu.children && menu.children.length > 0) {
    const validChildren = menu.children
      .filter((child) => child.menuType !== "F" && child.visible === 1)
      .map((child) => convertToMenuItem(child, menuPath))
      .filter((item): item is MenuItem => item !== null);

    if (validChildren.length > 0 || menu.menuType === "M") {
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
  const { message } = App.useApp();
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
      .filter((menu) => menu.menuType !== "F" && menu.visible === 1)
      .map((menu) => convertToMenuItem(menu))
      .filter((item): item is MenuItem => item !== null);
  }, [menus]);

  // 构建菜单路径映射表
  const menuPathMap = useMemo(() => buildMenuPathMap(menus), [menus]);

  /**
   * 构建完整的父级菜单链
   * 只返回父级菜单的 key，不包含当前页面路径
   * 因为当前页面可能是不可展开的菜单项（menuType === 'C'）
   */
  function buildParentKeyChain(
    currentPath: string,
    menuPathMap: Map<string, { level: number; topLevel?: string; secondLevel?: string }>
  ): string[] {
    const normalizePath = (p: string): string => (p.startsWith("/") ? p.slice(1) : p);
    const normalizedPathname = normalizePath(currentPath);
    const menuInfo = menuPathMap.get(normalizedPathname);

    if (!menuInfo) {
      return [];
    }

    const result: string[] = [];

    // 二级菜单：添加其父级（一级）菜单
    if (menuInfo.level === 2 && menuInfo.topLevel) {
      result.push(menuInfo.topLevel);
    }

    // 三级菜单：添加其父级（二级）菜单和祖父级（一级）菜单
    if (menuInfo.level === 3) {
      if (menuInfo.secondLevel) {
        result.push(menuInfo.secondLevel);
      }
      if (menuInfo.topLevel) {
        result.push(menuInfo.topLevel);
      }
    }

    return result;
  }

  // 路由变化时更新展开的菜单（防止循环刷新）
  useEffect(() => {
    if (!menuPathMap.size || !location.pathname) return;

    const parentKeys = buildParentKeyChain(location.pathname, menuPathMap);

    expectedOpenKeysRef.current = parentKeys;
    setOpenKeys(parentKeys);

    setTimeout(() => {
      expectedOpenKeysRef.current = null;
    }, NAVIGATION_DELAY);
  }, [location.pathname, menuPathMap]);

  // 菜单点击处理
  const handleMenuClick: MenuProps["onClick"] = (e) => {
    const clickedMenu = findMenuById(menus, e.key);

    if (clickedMenu && clickedMenu.status !== 0) {
      message.warning("该功能已停用，暂时无法访问");
      return;
    }

    // 只有菜单类型（C）才执行导航，目录类型（M）不导航
    if (clickedMenu && clickedMenu.menuType === "C") {
      const menuInfo = menuPathMap.get(clickedMenu.id);
      const fullPath = menuInfo?.fullPath || buildFullPath(clickedMenu);
      const navigationPath = fullPath.startsWith("/") ? fullPath : `/${fullPath}`;
      navigate(navigationPath);
    }
  };

  // 处理菜单展开/收起 - 手风琴效果（支持一级和二级菜单）
  const handleOpenChange: MenuProps["onOpenChange"] = (keys) => {
    // 检查是否是导航触发的预期变化（防止循环刷新）
    if (expectedOpenKeysRef.current) {
      const expected = expectedOpenKeysRef.current;
      const isExpectedChange =
        keys.length === expected.length && keys.every((k) => expected.includes(k));

      if (isExpectedChange) {
        setOpenKeys(keys);
        return;
      }
    }

    const latestOpenKey = keys.find((key) => !openKeys.includes(key));

    if (!latestOpenKey) {
      setOpenKeys([...keys]);
      return;
    }

    const menuLevel = getMenuLevel(latestOpenKey, menuPathMap);

    switch (menuLevel) {
      case 1:
      case undefined: {
        // 一级菜单或以 top_ 开头的菜单：应用手风琴效果
        // 保留新展开的一级菜单和所有非一级菜单，移除其他一级菜单
        const nonTopLevelKeys = keys.filter((key) => !isTopLevelMenu(key, menuPathMap));
        setOpenKeys([latestOpenKey, ...nonTopLevelKeys]);
        break;
      }

      case 2: {
        // 二级菜单：应用手风琴效果（仅在同一一级菜单下）
        const currentTopLevel = menuPathMap.get(latestOpenKey)?.topLevel;

        if (!currentTopLevel) {
          setOpenKeys([...keys]);
          return;
        }

        // 保留所有一级菜单
        const topLevelKeys = keys.filter((key) => isTopLevelMenu(key, menuPathMap));

        // 找出所有与新展开的二级菜单属于同一一级菜单的其他已展开二级菜单
        const sameTopLevelSecondKeys = openKeys.filter((key) => {
          if (!isSecondLevelMenu(key, menuPathMap)) return false;
          return isSameTopLevelMenu(key, latestOpenKey, menuPathMap);
        });

        // 构建新的openKeys
        const result = [
          ...topLevelKeys,
          ...keys.filter(
            (key) =>
              !sameTopLevelSecondKeys.includes(key) &&
              (isSecondLevelMenu(key, menuPathMap) || key === latestOpenKey)
          ),
        ];
        setOpenKeys(result);
        break;
      }

      case 3:
        // 三级菜单：不应用手风琴效果
        setOpenKeys([...keys]);
        break;

      default:
        // 未知层级：直接应用
        setOpenKeys([...keys]);
    }
  };

  // 获取当前选中的菜单项
  const getSelectedKeys = () => {
    if (!location.pathname) return [];

    const pathsToTry = [
      location.pathname,
      location.pathname.startsWith("/") ? location.pathname.slice(1) : location.pathname,
    ];

    for (const path of pathsToTry) {
      const menuInfo = menuPathMap.get(path);
      if (menuInfo) {
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
      width={DEFAULT_SIDEBAR_WIDTH}
      trigger={null}
      className="layout-sidebar"
      style={{
        background: "#ffffff",
      }}
    >
      <div
        className="flex items-center relative"
        style={{
          height: `${HEADER_HEIGHT}px`,
          borderBottom: "1px solid var(--theme-border-primary)",
          paddingLeft: sidebarCollapsed ? "0" : "56px",
          transition: "padding-left var(--theme-transition-base)",
        }}
      >
        {/* 收缩按钮 */}
        <button
          onClick={toggleSidebar}
          className="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors absolute"
          style={{
            color: "var(--theme-text-secondary)",
            background: "transparent",
            border: "none",
            cursor: "pointer",
            left: `${COLLAPSE_BUTTON_LEFT}px`,
          }}
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 20 20"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            style={{
              transition: "transform var(--theme-transition-base)",
              transform: sidebarCollapsed ? "rotate(180deg)" : "rotate(0deg)",
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
          className="text-xl font-bold"
          style={{
            backgroundImage:
              "linear-gradient(135deg, var(--theme-brand, #3b82f6) 0%, #ffffff 100%)",
            WebkitBackgroundClip: "text",
            WebkitTextFillColor: "transparent",
            backgroundClip: "text",
            color: "transparent",
            display: sidebarCollapsed ? "none" : "inline-block",
          }}
        >
          星苒
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
            background: "transparent",
            fontSize: `${MENU_FONT_SIZE}px`,
            color: "var(--theme-text-primary)",
          }}
        />
      )}
    </Sider>
  );
};

export default Sidebar;

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Layout, Menu, Spin, message } from 'antd';
import {
  HomeOutlined,
  MenuOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import type { Menu as MenuType } from '@/types';
import {
  buildMenuPathMap,
  isSameTopLevelMenu,
  getMenuLevel,
  isThirdLevelMenu,
  isSecondLevelMenu,
  isTopLevelMenu,
} from './sidebar-helper';
import { getIconComponent } from '@/utils/iconUtils';

const { Sider } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

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

const Sidebar: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const expectedOpenKeysRef = useRef<string[] | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { menus, loading, fetchMenus } = useMenuStore();

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
      console.log('[buildParentKeyChain] 未找到菜单:', normalizedPathname);
      return [];
    }

    console.log('[buildParentKeyChain] 菜单信息:', {
      key: normalizedPathname,
      level: menuInfo.level,
      topLevel: menuInfo.topLevel,
      secondLevel: menuInfo.secondLevel
    });

    const result: string[] = [];

    // 不添加当前菜单的 key，只添加父级菜单
    // result.push(normalizedPathname);

    // 如果是二级菜单，添加其父级（一级）菜单
    if (menuInfo.level === 2 && menuInfo.topLevel) {
      result.push(menuInfo.topLevel);
      console.log('[buildParentKeyChain] 添加一级菜单:', menuInfo.topLevel);
    }

    // 如果是三级菜单，添加其父级（二级）菜单和祖父级（一级）菜单
    if (menuInfo.level === 3) {
      if (menuInfo.secondLevel) {
        result.push(menuInfo.secondLevel); // 二级
        console.log('[buildParentKeyChain] 添加二级菜单:', menuInfo.secondLevel);
      }
      if (menuInfo.topLevel) {
        result.push(menuInfo.topLevel); // 一级
        console.log('[buildParentKeyChain] 添加一级菜单:', menuInfo.topLevel);
      }
    }

    console.log('[buildParentKeyChain] 最终结果:', result);
    return result;
  }

  // 当路由变化时，自动展开包含当前页面的父菜单
  useEffect(() => {
    console.log('=== 路由变化 useEffect ===');
    console.log('当前路径:', location.pathname);
    console.log('当前 openKeys:', openKeys);

    // 不要对 dashboard 或根路径做处理
    if (location.pathname === '/' || location.pathname === '/dashboard') {
      setOpenKeys([]);
      return;
    }

    const parentKeyChain = buildParentKeyChain(location.pathname, menuPathMap);
    console.log('父级菜单链:', parentKeyChain);

    if (parentKeyChain.length > 0) {
      console.log('设置 openKeys 为:', parentKeyChain);
      setOpenKeys(parentKeyChain);
    } else {
      console.log('父级菜单链为空，清空 openKeys');
      setOpenKeys([]);
    }
  }, [location.pathname, JSON.stringify(menus)]);

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    console.log('菜单点击:', e.key, '当前 openKeys:', openKeys);

    // 查找被点击的菜单
    const findMenuByPath = (menuList: MenuType[], key: string, parentPath: string = ''): MenuType | null => {
      for (const menu of menuList) {
        // 构建完整路径（与 convertToMenuItem 逻辑一致）
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
    console.log('点击的菜单:', clickedMenu?.menuName, '类型:', clickedMenu?.menuType);

    // 检查菜单是否停用
    if (clickedMenu && clickedMenu.status !== 0) {
      message.warning('该功能已停用，暂时无法访问');
      return;
    }

    // 只有菜单类型（C）才执行导航，目录类型（M）不导航
    if (clickedMenu && clickedMenu.menuType === 'C') {
      // 确保路径以 / 开头
      const path = e.key.startsWith('/') ? e.key : `/${e.key}`;
      console.log('导航到:', path);
      navigate(path);
    }
  };

  // 处理菜单展开/收起 - 手风琴效果
  // 处理菜单展开/收起 - 改进版手风琴效果（支持二级菜单手风琴）
  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    console.log('=== 菜单展开/收起事件 ===');
    console.log('新 keys:', keys);
    console.log('当前 openKeys:', openKeys);
    console.log('预期 keys:', expectedOpenKeysRef.current);

    // 检查是否是导航触发的预期变化
    if (expectedOpenKeysRef.current) {
      const expected = expectedOpenKeysRef.current;
      const isExpectedChange =
        keys.length === expected.length &&
        keys.every(k => expected.includes(k));

      if (isExpectedChange) {
        console.log('这是导航触发的预期变化，直接应用，跳过手风琴逻辑');
        setOpenKeys(keys);
        return;
      }
    }

    // 找出新展开的菜单（keys 中有但 openKeys 中没有的）
    const latestOpenKey = keys.find((key) => !openKeys.includes(key));
    console.log('latestOpenKey:', latestOpenKey);

    if (latestOpenKey) {
      const menuLevel = getMenuLevel(latestOpenKey, menuPathMap);
      console.log('菜单层级:', menuLevel, 'menuPathMap中:', menuPathMap.get(latestOpenKey));

      if (menuLevel === 1 || latestOpenKey.startsWith('top_')) {
        // ========================================
        // 一级菜单：应用手风琴效果
        // ========================================
        console.log('>>> 一级菜单手风琴效果');
        // 1. 保留新展开的一级菜单
        // 2. 保留所有非一级菜单（二级和三级菜单）
        // 3. 移除其他所有一级菜单
        const nonTopLevelKeys = keys.filter(key => !isTopLevelMenu(key, menuPathMap));
        const result = [latestOpenKey, ...nonTopLevelKeys];
        console.log('一级菜单手风琴最终结果:', result);
        setOpenKeys(result);

      } else if (menuLevel === 2) {
        // ========================================
        // 二级菜单：应用手风琴效果（仅在同一一级菜单下）
        // ========================================
        console.log('>>> 二级菜单手风琴效果');
        // 1. 找出当前二级菜单所属的一级菜单
        const currentTopLevel = menuPathMap.get(latestOpenKey)?.topLevel;

        if (!currentTopLevel) {
          console.warn('无法确定二级菜单的所属一级菜单，直接应用');
          setOpenKeys([...keys]);
          return;
        }

        console.log('当前二级菜单所属一级菜单:', currentTopLevel);

        // 2. 保留所有一级菜单
        const topLevelKeys = keys.filter(key => isTopLevelMenu(key, menuPathMap));
        console.log('所有一级菜单:', topLevelKeys);

        // 3. 找出所有与新展开的二级菜单属于同一一级菜单的其他已展开二级菜单
        const sameTopLevelSecondKeys = openKeys.filter(key => {
          if (!isSecondLevelMenu(key, menuPathMap)) return false;
          return isSameTopLevelMenu(key, latestOpenKey, menuPathMap);
        });
        console.log('同属一个一级菜单的其他二级菜单:', sameTopLevelSecondKeys);

        // 4. 需要收起的这些同级二级菜单
        const keysToClose = sameTopLevelSecondKeys;
        console.log('需要收起的同级二级菜单:', keysToClose);

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
        console.log('二级菜单手风琴最终结果:', result);
        setOpenKeys(result);

      } else if (menuLevel === 3) {
        // ========================================
        // 三级菜单：不应用手风琴效果
        // ========================================
        console.log('>>> 三级菜单：不应用手风琴效果，直接应用');
        setOpenKeys([...keys]);

      } else {
        // ========================================
        // 未知层级：直接应用（降级处理）
        // ========================================
        console.warn('未知菜单层级，直接应用:', latestOpenKey);
        setOpenKeys([...keys]);
      }
    } else {
      // ========================================
      // 收起操作：直接应用
      // ========================================
      console.log('收起操作，直接应用:', keys);
      setOpenKeys([...keys]);
    }
  };

  // 获取当前选中的菜单项
  const getSelectedKeys = () => {
    return [location.pathname];
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
          产险湖北数智化平台
        </h1>
        {collapsed && (
          <span className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent">R</span>
        )}
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin size="large" />
        </div>
      ) : (
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={getSelectedKeys()}
          openKeys={openKeys}
          onOpenChange={handleOpenChange}
          items={menuItems}
          onClick={handleMenuClick}
          className="border-r-0 px-2 py-4"
          style={{
            background: 'transparent',
            fontSize: '14px'
          }}
        />
      )}
    </Sider>
  );
};

export default Sidebar;

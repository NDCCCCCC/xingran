import React, { useState, useEffect, useMemo } from 'react';
import { Layout, Menu, Spin, message } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  MenuOutlined,
  SettingOutlined,
  FileTextOutlined,
  BellOutlined,
  MonitorOutlined,
  DatabaseOutlined,
  BarChartOutlined,
  HistoryOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  FormOutlined,
  AppstoreOutlined,
  HomeOutlined,
  LaptopOutlined,
  CodeOutlined,
  BugOutlined,
  SafetyOutlined,
  ToolOutlined,
  FolderOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  PlusOutlined,
  MinusOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  QuestionCircleOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  WarningOutlined,
  LockOutlined,
  UnlockOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  CopyOutlined,
  SyncOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  PoweroffOutlined,
  DownloadOutlined,
  UploadOutlined,
  ImportOutlined,
  ExportOutlined,
  SaveOutlined,
  UndoOutlined,
  RedoOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  UpOutlined,
  DownOutlined,
  LeftOutlined,
  RightOutlined,
  CaretUpOutlined,
  CaretDownOutlined,
  CaretLeftOutlined,
  CaretRightOutlined,
  ToTopOutlined,
  BorderOutlined,
  AreaChartOutlined,
  PieChartOutlined,
  DotChartOutlined,
  LineChartOutlined,
  FundOutlined,
  SlidersOutlined,
  WindowsOutlined,
  ChromeOutlined,
  GithubOutlined,
  WeiboOutlined,
  TwitterOutlined,
  WechatOutlined,
  SkypeOutlined,
  QqOutlined,
  GitlabOutlined,
  LinkedinOutlined,
  DropboxOutlined,
  FacebookOutlined,
  AmazonOutlined,
  GoogleOutlined,
  InstagramOutlined,
  YahooOutlined,
  BehanceOutlined,
  FacebookFilled,
  YoutubeOutlined,
  ZhihuOutlined,
  DingdingOutlined,
  IdcardOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import type { Menu as MenuType } from '@/types';

const { Sider } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

// 图标映射表
const iconMap: Record<string, React.ReactNode> = {
  dashboard: <DashboardOutlined />,
  user: <UserOutlined />,
  users: <TeamOutlined />,
  menu: <MenuOutlined />,
  setting: <SettingOutlined />,
  settings: <SettingOutlined />,
  file: <FileTextOutlined />,
  files: <FileTextOutlined />,
  document: <FileTextOutlined />,
  bell: <BellOutlined />,
  notification: <BellOutlined />,
  monitor: <MonitorOutlined />,
  server: <CloudServerOutlined />,
  database: <DatabaseOutlined />,
  chart: <BarChartOutlined />,
  barchart: <BarChartOutlined />,
  history: <HistoryOutlined />,
  log: <HistoryOutlined />,
  clock: <ClockCircleOutlined />,
  job: <ClockCircleOutlined />,
  cache: <DatabaseOutlined />,
  home: <HomeOutlined />,
  system: <SettingOutlined />,
  idcard: <IdcardOutlined />,
  edit: <EditOutlined />,
  delete: <DeleteOutlined />,
  search: <SearchOutlined />,
  add: <PlusOutlined />,
  remove: <MinusOutlined />,
  check: <CheckCircleOutlined />,
  close: <CloseCircleOutlined />,
  question: <QuestionCircleOutlined />,
  exclamation: <ExclamationCircleOutlined />,
  info: <InfoCircleOutlined />,
  warning: <WarningOutlined />,
  lock: <LockOutlined />,
  unlock: <UnlockOutlined />,
  eye: <EyeOutlined />,
  'eye-invisible': <EyeInvisibleOutlined />,
  copy: <CopyOutlined />,
  sync: <SyncOutlined />,
  reload: <ReloadOutlined />,
  play: <PlayCircleOutlined />,
  pause: <PauseCircleOutlined />,
  stop: <StopOutlined />,
  poweroff: <PoweroffOutlined />,
  download: <DownloadOutlined />,
  upload: <UploadOutlined />,
  import: <ImportOutlined />,
  export: <ExportOutlined />,
  save: <SaveOutlined />,
  undo: <UndoOutlined />,
  redo: <RedoOutlined />,
  arrowLeft: <ArrowLeftOutlined />,
  arrowRight: <ArrowRightOutlined />,
  arrowUp: <ArrowUpOutlined />,
  arrowDown: <ArrowDownOutlined />,
  left: <LeftOutlined />,
  right: <RightOutlined />,
  up: <UpOutlined />,
  down: <DownOutlined />,
  appstore: <AppstoreOutlined />,
  laptop: <LaptopOutlined />,
  code: <CodeOutlined />,
  bug: <BugOutlined />,
  safety: <SafetyOutlined />,
  tool: <ToolOutlined />,
  folder: <FolderOutlined />,
  form: <FormOutlined />,
  areaChart: <AreaChartOutlined />,
  pieChart: <PieChartOutlined />,
  lineChart: <LineChartOutlined />,
};

// 完整的 Ant Design 图标名映射到 iconMap 的键名
const fullIconNameMap: Record<string, string> = {
  // 单词图标
  'UserOutlined': 'user',
  'TeamOutlined': 'users',
  'MenuOutlined': 'menu',
  'SettingOutlined': 'setting',
  'FileTextOutlined': 'file',
  'BellOutlined': 'bell',
  'MonitorOutlined': 'monitor',
  'DatabaseOutlined': 'database',
  'HistoryOutlined': 'history',
  'ClockCircleOutlined': 'clock',
  'HomeOutlined': 'home',
  'FolderOutlined': 'folder',
  'LaptopOutlined': 'laptop',
  'DashboardOutlined': 'dashboard',
  'DashboardFilled': 'dashboard',
  'SettingFilled': 'setting',
  'IdcardOutlined': 'idcard',

  // 复合词图标 - 需要特殊映射
  'CloudServerOutlined': 'server',
  'ApartmentOutlined': 'users',
  'DesktopOutlined': 'laptop',
  'BookOutlined': 'file',
};

// 获取图标组件
function getIcon(iconName?: string | null): React.ReactNode {
  if (!iconName) return undefined;

  // 首先尝试直接从 iconMap 获取
  if (iconMap[iconName]) {
    return iconMap[iconName];
  }

  // 尝试从完整图标名映射中获取
  const mappedName = fullIconNameMap[iconName];
  if (mappedName && iconMap[mappedName]) {
    return iconMap[mappedName];
  }

  return undefined;
}

// 将后端菜单数据转换为 Ant Design Menu 组件所需的格式
function convertToMenuItem(menu: MenuType): MenuItem {
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

  // 如果有子菜单，递归转换
  if (menu.children && menu.children.length > 0) {
    const validChildren = menu.children
      .filter(child => child.menuType !== 'F' && child.visible === 1)
      .map(convertToMenuItem)
      .filter((item): item is MenuItem => item !== null);

    // 如果有正常子菜单，或者是父级目录，都显示
    if (validChildren.length > 0 || menu.menuType === 'M') {
      return {
        key: menu.path || menu.id,
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
    key: menu.path || menu.id,
    icon,
    label: menu.menuName,
    disabled: isDisabled,
    style: isDisabled ? { opacity: 0.5 } : undefined,
  };
}

const Sidebar: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [openKeys, setOpenKeys] = useState<string[]>([]);
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
      .map(convertToMenuItem)
      .filter((item): item is MenuItem => item !== null);
  }, [menus]);

  // 根据当前路径查找父菜单的 key
  const findParentKey = (pathname: string): string | null => {
    const findMenuByPath = (menuList: MenuType[], path: string): { menu: MenuType | null; parentKey: string | null } => {
      for (const menu of menuList) {
        // 检查子菜单是否匹配当前路径
        if (menu.children && menu.children.length > 0) {
          for (const child of menu.children) {
            if (child.path === path) {
              // 找到匹配的子菜单，返回父菜单的 key（id 或 path）
              return { menu, parentKey: menu.path || menu.id };
            }
            // 递归检查更深层的子菜单
            const result = findMenuByPath([child], path);
            if (result.menu) {
              return { menu: result.menu, parentKey: menu.path || menu.id };
            }
          }
        }
      }
      return { menu: null, parentKey: null };
    };

    const result = findMenuByPath(menus, pathname);
    return result.parentKey;
  };

  // 当路由变化时，自动展开包含当前页面的父菜单
  useEffect(() => {
    // 不要对 dashboard 或根路径做处理
    if (location.pathname === '/' || location.pathname === '/dashboard') {
      setOpenKeys([]);
      return;
    }

    const parentKey = findParentKey(location.pathname);
    if (parentKey) {
      setOpenKeys([parentKey]);
    } else {
      // 如果找不到父菜单，保持当前展开状态或清空
      setOpenKeys([]);
    }
  }, [location.pathname, menus]);

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    // 查找被点击的菜单
    const findMenuByPath = (menuList: MenuType[], key: string): MenuType | null => {
      for (const menu of menuList) {
        if ((menu.path === key || menu.id === key) && menu.visible === 1) {
          return menu;
        }
        if (menu.children) {
          const found = findMenuByPath(menu.children, key);
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

    navigate(e.key);
  };

  // 处理菜单展开/收起 - 手风琴效果
  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    // 只保留最新展开的菜单
    const latestOpenKey = keys.find((key) => !openKeys.includes(key));
    if (latestOpenKey) {
      setOpenKeys([latestOpenKey]);
    } else {
      // 如果没有新的展开菜单，说明是收起操作
      setOpenKeys(keys);
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
          RuoYi Next
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

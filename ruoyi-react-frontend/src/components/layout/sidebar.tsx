import React, { useState, useEffect, useMemo } from 'react';
import { Layout, Menu, Spin } from 'antd';
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

// 获取图标组件
function getIcon(iconName?: string | null): React.ReactNode {
  if (!iconName) return undefined;
  return iconMap[iconName] || undefined;
}

// 将后端菜单数据转换为 Ant Design Menu 组件所需的格式
function convertToMenuItem(menu: MenuType): MenuItem {
  const icon = getIcon(menu.icon);

  // 只显示目录(M)和菜单(C)类型，不显示按钮(F)类型
  if (menu.menuType === 'F') {
    return null as any;
  }

  // 如果有子菜单，递归转换
  if (menu.children && menu.children.length > 0) {
    const validChildren = menu.children
      .filter(child => child.menuType !== 'F' && child.visible === 1 && child.status === 0)
      .map(convertToMenuItem)
      .filter((item): item is MenuItem => item !== null);

    if (validChildren.length > 0) {
      return {
        key: menu.path || menu.id,
        icon,
        label: menu.menuName,
        children: validChildren,
      };
    }
  }

  // 只显示可见且状态正常的菜单
  if (menu.visible === 1 && menu.status === 0) {
    return {
      key: menu.path || menu.id,
      icon,
      label: menu.menuName,
    };
  }

  return null as any;
}

const Sidebar: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
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
      .filter(menu => menu.menuType !== 'F' && menu.visible === 1 && menu.status === 0)
      .map(convertToMenuItem)
      .filter((item): item is MenuItem => item !== null);
  }, [menus]);

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

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin size="large" />
        </div>
      ) : (
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
      )}
    </Sider>
  );
};

export default Sidebar;

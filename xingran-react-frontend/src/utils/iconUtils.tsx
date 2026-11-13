/**
 * 图标工具函数
 * 用于管理 Ant Design 图标的映射和获取
 *
 * Performance note (Vercel audit Sprint 1 Fix #3):
 * All icons are imported via named imports into a static registry.
 * This avoids `import * as Icons from "@ant-design/icons"` which
 * pulls ~1 MB of the entire icon library into the initial bundle.
 */

import {
  DashboardOutlined,
  HomeOutlined,
  AppstoreOutlined,
  MenuOutlined,
  SettingOutlined,
  ToolOutlined,
  BulbOutlined,
  AlertOutlined,
  UserOutlined,
  TeamOutlined,
  IdcardOutlined,
  SolutionOutlined,
  LoginOutlined,
  LogoutOutlined,
  KeyOutlined,
  LockOutlined,
  FileTextOutlined,
  FileOutlined,
  FolderOutlined,
  FolderOpenOutlined,
  CopyOutlined,
  SnippetsOutlined,
  BookOutlined,
  DatabaseOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  UpOutlined,
  DownOutlined,
  LeftOutlined,
  RightOutlined,
  CaretUpOutlined,
  CaretDownOutlined,
  CaretLeftOutlined,
  CaretRightOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  FastForwardOutlined,
  FastBackwardOutlined,
  StepForwardOutlined,
  BellOutlined,
  MessageOutlined,
  MailOutlined,
  WechatOutlined,
  QqOutlined,
  DingdingOutlined,
  WeiboOutlined,
  GithubOutlined,
  CloudOutlined,
  CloudServerOutlined,
  LaptopOutlined,
  DesktopOutlined,
  MobileOutlined,
  TabletOutlined,
  AreaChartOutlined,
  BarChartOutlined,
  LineChartOutlined,
  PieChartOutlined,
  DotChartOutlined,
  FundOutlined,
  SlidersOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
  CalendarOutlined,
  FieldTimeOutlined,
  HourglassOutlined,
  MonitorOutlined,
  BugOutlined,
  CodeOutlined,
  ApiOutlined,
  ConsoleSqlOutlined,
  NodeIndexOutlined,
  SafetyOutlined,
  SecurityScanOutlined,
  UnlockOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  WifiOutlined,
  ApartmentOutlined,
  ClusterOutlined,
  ShareAltOutlined,
  StarOutlined,
  HeartOutlined,
  LikeOutlined,
  DislikeOutlined,
  FlagOutlined,
  TagOutlined,
  // 菜单种子在用的图标（menu_catalog_seed.sql）——此前未进静态注册表，
  // 导致对应菜单项无图标渲染（AntD Menu 无 .ant-menu-item-icon 容器，
  // 文字缩进比同级有图标项左移约 24px，同级菜单排版参差）。
  // 注：DotChartOutlined / FundOutlined 已在上方 import，避免重复声明。
  FormOutlined,
  BuildOutlined,
  ControlOutlined,
  EnvironmentOutlined,
  RobotOutlined,
  SyncOutlined,
  UnorderedListOutlined,
  CloudDownloadOutlined,
} from "@ant-design/icons";
import type { FC } from "react";

// Static registry — all icons from iconCategories, plus SearchOutlined (used directly)
// Using FC<any> to accommodate ForwardRefExoticComponent (antd icons) which is
// not assignable to ComponentType<unknown> but is to FC<any>（antd 图标类型边界契约，
// unknown 会破坏全部 antd 图标赋值——登记保留）
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- antd 图标组件类型边界（见上注释）
export const STATIC_ICON_REGISTRY: Record<string, FC<any>> = {
  DashboardOutlined,
  HomeOutlined,
  AppstoreOutlined,
  MenuOutlined,
  SettingOutlined,
  ToolOutlined,
  BulbOutlined,
  AlertOutlined,
  UserOutlined,
  TeamOutlined,
  IdcardOutlined,
  SolutionOutlined,
  LoginOutlined,
  LogoutOutlined,
  KeyOutlined,
  LockOutlined,
  FileTextOutlined,
  FileOutlined,
  FolderOutlined,
  FolderOpenOutlined,
  CopyOutlined,
  SnippetsOutlined,
  BookOutlined,
  DatabaseOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  UpOutlined,
  DownOutlined,
  LeftOutlined,
  RightOutlined,
  CaretUpOutlined,
  CaretDownOutlined,
  CaretLeftOutlined,
  CaretRightOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  FastForwardOutlined,
  FastBackwardOutlined,
  StepForwardOutlined,
  BellOutlined,
  MessageOutlined,
  MailOutlined,
  WechatOutlined,
  QqOutlined,
  DingdingOutlined,
  WeiboOutlined,
  GithubOutlined,
  CloudOutlined,
  CloudServerOutlined,
  LaptopOutlined,
  DesktopOutlined,
  MobileOutlined,
  TabletOutlined,
  AreaChartOutlined,
  BarChartOutlined,
  LineChartOutlined,
  PieChartOutlined,
  DotChartOutlined,
  FundOutlined,
  SlidersOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
  CalendarOutlined,
  FieldTimeOutlined,
  HourglassOutlined,
  MonitorOutlined,
  BugOutlined,
  CodeOutlined,
  ApiOutlined,
  ConsoleSqlOutlined,
  NodeIndexOutlined,
  SafetyOutlined,
  SecurityScanOutlined,
  UnlockOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  WifiOutlined,
  ApartmentOutlined,
  ClusterOutlined,
  ShareAltOutlined,
  StarOutlined,
  HeartOutlined,
  LikeOutlined,
  DislikeOutlined,
  FlagOutlined,
  TagOutlined,
  // 菜单种子在用图标（补注册表缺失项，详见 import 段注释）
  // 注：DotChartOutlined / FundOutlined 已在注册表上方，避免重复键。
  FormOutlined,
  BuildOutlined,
  ControlOutlined,
  EnvironmentOutlined,
  RobotOutlined,
  SyncOutlined,
  UnorderedListOutlined,
  CloudDownloadOutlined,
};

// ========================================
// 图标分类（用于图标选择器）
// ========================================

export const iconCategories = {
  基础: [
    "DashboardOutlined",
    "HomeOutlined",
    "AppstoreOutlined",
    "MenuOutlined",
    "SettingOutlined",
    "ToolOutlined",
    "BulbOutlined",
    "AlertOutlined",
  ],
  用户: [
    "UserOutlined",
    "TeamOutlined",
    "IdcardOutlined",
    "SolutionOutlined",
    "LoginOutlined",
    "LogoutOutlined",
    "KeyOutlined",
    "LockOutlined",
  ],
  文件: [
    "FileTextOutlined",
    "FileOutlined",
    "FolderOutlined",
    "FolderOpenOutlined",
    "CopyOutlined",
    "SnippetsOutlined",
    "BookOutlined",
    "DatabaseOutlined",
  ],
  操作: [
    "EditOutlined",
    "DeleteOutlined",
    "SearchOutlined",
    "ReloadOutlined",
    "PlusOutlined",
    "MinusOutlined",
    "CheckCircleOutlined",
    "CloseCircleOutlined",
  ],
  方向: [
    "ArrowUpOutlined",
    "ArrowDownOutlined",
    "ArrowLeftOutlined",
    "ArrowRightOutlined",
    "UpOutlined",
    "DownOutlined",
    "LeftOutlined",
    "RightOutlined",
    "CaretUpOutlined",
    "CaretDownOutlined",
    "CaretLeftOutlined",
    "CaretRightOutlined",
  ],
  媒体: [
    "PlayCircleOutlined",
    "PauseCircleOutlined",
    "StopOutlined",
    "FastForwardOutlined",
    "FastBackwardOutlined",
    "StepForwardOutlined",
  ],
  通信: [
    "BellOutlined",
    "MessageOutlined",
    "MailOutlined",
    "WechatOutlined",
    "QqOutlined",
    "DingdingOutlined",
    "WeiboOutlined",
    "GithubOutlined",
  ],
  云服务: [
    "CloudOutlined",
    "CloudServerOutlined",
    "LaptopOutlined",
    "DesktopOutlined",
    "MobileOutlined",
    "TabletOutlined",
  ],
  数据: [
    "DatabaseOutlined",
    "AreaChartOutlined",
    "BarChartOutlined",
    "LineChartOutlined",
    "PieChartOutlined",
    "DotChartOutlined",
    "FundOutlined",
    "SlidersOutlined",
  ],
  时间: [
    "ClockCircleOutlined",
    "HistoryOutlined",
    "CalendarOutlined",
    "FieldTimeOutlined",
    "HourglassOutlined",
  ],
  系统: [
    "MonitorOutlined",
    "SettingOutlined",
    "BugOutlined",
    "CodeOutlined",
    "ApiOutlined",
    "ConsoleSqlOutlined",
    "DatabaseOutlined",
    "NodeIndexOutlined",
  ],
  安全: [
    "SafetyOutlined",
    "SecurityScanOutlined",
    "LockOutlined",
    "UnlockOutlined",
    "EyeOutlined",
    "EyeInvisibleOutlined",
  ],
  网络: [
    "WifiOutlined",
    "ApartmentOutlined",
    "ClusterOutlined",
    "NodeIndexOutlined",
    "ShareAltOutlined",
    "ApiOutlined",
  ],
  其他: [
    "StarOutlined",
    "HeartOutlined",
    "LikeOutlined",
    "DislikeOutlined",
    "FlagOutlined",
    "TagOutlined",
    // 菜单种子在用图标（此前未进注册表/分类，见 STATIC_ICON_REGISTRY 注释）
    "FormOutlined",
    "BuildOutlined",
    "ControlOutlined",
    "EnvironmentOutlined",
    "RobotOutlined",
    "SyncOutlined",
    "UnorderedListOutlined",
    "CloudDownloadOutlined",
  ],
};

// ========================================
// 图标名称到中文描述的映射
// ========================================

export const iconDescriptionMap: Record<string, string> = {
  // 基础图标
  DashboardOutlined: "仪表盘",
  HomeOutlined: "首页",
  AppstoreOutlined: "应用网格",
  MenuOutlined: "菜单",
  SettingOutlined: "设置",
  Settings: "设置",
  ToolOutlined: "工具",
  BulbOutlined: "灯泡",
  AlertOutlined: "提醒",

  // 用户相关
  UserOutlined: "用户",
  TeamOutlined: "团队",
  IdcardOutlined: "身份证",
  SolutionOutlined: "解决方案",
  LoginOutlined: "登录",
  LogoutOutlined: "登出",
  KeyOutlined: "密钥",
  LockOutlined: "锁定",

  // 文件相关
  FileTextOutlined: "文档",
  FileOutlined: "文件",
  FolderOutlined: "文件夹",
  FolderOpenOutlined: "打开文件夹",
  CopyOutlined: "复制",
  SnippetsOutlined: "片段",
  BookOutlined: "书本",
  DatabaseOutlined: "数据库",

  // 操作相关
  EditOutlined: "编辑",
  DeleteOutlined: "删除",
  SearchOutlined: "搜索",
  ReloadOutlined: "刷新",
  PlusOutlined: "添加",
  MinusOutlined: "减少",
  CheckCircleOutlined: "成功",
  CloseCircleOutlined: "失败",

  // 方向相关
  ArrowUpOutlined: "向上箭头",
  ArrowDownOutlined: "向下箭头",
  ArrowLeftOutlined: "向左箭头",
  ArrowRightOutlined: "向右箭头",
  UpOutlined: "向上",
  DownOutlined: "向下",
  LeftOutlined: "向左",
  RightOutlined: "向右",

  // 媒体相关
  PlayCircleOutlined: "播放",
  PauseCircleOutlined: "暂停",
  StopOutlined: "停止",

  // 通信相关
  BellOutlined: "通知",
  MessageOutlined: "消息",
  MailOutlined: "邮件",

  // 云服务
  CloudOutlined: "云",
  CloudServerOutlined: "云服务器",
  LaptopOutlined: "笔记本",
  DesktopOutlined: "台式机",

  // 数据相关
  AreaChartOutlined: "面积图",
  BarChartOutlined: "柱状图",
  LineChartOutlined: "折线图",
  PieChartOutlined: "饼图",

  // 时间相关
  ClockCircleOutlined: "时钟",
  HistoryOutlined: "历史",
  CalendarOutlined: "日历",

  // 系统相关
  MonitorOutlined: "监控",
  BugOutlined: "Bug",
  CodeOutlined: "代码",

  // 安全相关
  SafetyOutlined: "安全",
  SecurityScanOutlined: "安全扫描",
  UnlockOutlined: "解锁",
  EyeOutlined: "眼睛",
  EyeInvisibleOutlined: "隐藏眼睛",

  // 网络相关
  WifiOutlined: "WiFi",
  ApartmentOutlined: "组织架构",
  ClusterOutlined: "集群",
  ShareAltOutlined: "分享",

  // 其他
  StarOutlined: "星标",
  HeartOutlined: "心",
  LikeOutlined: "赞",
  DislikeOutlined: "踩",
  FlagOutlined: "旗帜",
  TagOutlined: "标签",
};

// ========================================
// 完整图标名映射（向后兼容）
// ========================================

export const fullIconNameMap: Record<string, string> = {
  // 单词图标
  UserOutlined: "user",
  TeamOutlined: "users",
  MenuOutlined: "menu",
  SettingOutlined: "setting",
  SettingFilled: "setting",
  FileTextOutlined: "file",
  BellOutlined: "bell",
  MonitorOutlined: "monitor",
  DatabaseOutlined: "database",
  HistoryOutlined: "history",
  ClockCircleOutlined: "clock",
  HomeOutlined: "home",
  FolderOutlined: "folder",
  LaptopOutlined: "laptop",
  DashboardOutlined: "dashboard",
  DashboardFilled: "dashboard",
  IdcardOutlined: "idcard",

  // 网络设备管理相关图标
  CloudOutlined: "cloud",
  KeyOutlined: "key",
  ApartmentOutlined: "apartment",

  // 复合词图标 - 需要特殊映射
  CloudServerOutlined: "server",
  DesktopOutlined: "laptop",
  BookOutlined: "file",
};

// ========================================
// 获取图标组件
// ========================================

/**
 * 根据图标名称获取对应的 React 组件
 * @param iconName - 图标名称（支持简短名称和完整图标名）
 * @returns React 图标组件或 undefined
 */
export function getIconComponent(iconName?: string | null): React.ReactNode {
  if (!iconName) return undefined;

  // 1. Static registry lookup (all icons in iconCategories)
  const staticIcon = STATIC_ICON_REGISTRY[iconName];
  if (staticIcon) {
    const Icon = staticIcon;
    return <Icon />;
  }

  // 2. fullIconNameMap reverse-lookup into static registry
  const mappedName = fullIconNameMap[iconName];
  if (mappedName) {
    for (const key of Object.keys(STATIC_ICON_REGISTRY)) {
      if (
        key.toLowerCase().includes(mappedName.toLowerCase()) ||
        key.toLowerCase() === mappedName + "outlined"
      ) {
        const Icon = STATIC_ICON_REGISTRY[key];
        return <Icon />;
      }
    }
  }

  return undefined;
}

/**
 * 侧边栏专用：图标缺失时返回**透明占位图标**，保证同级菜单项的文字左缩进一致。
 *
 * 背景：Ant Design Menu 的图标占位靠 `.ant-menu-item-icon` 容器实现——
 * 有 icon 时该容器存在（宽 16px + 右间距），无 icon 时整个容器不渲染，
 * 导致同层级中「有图标项」的文字比「无图标项」右移约 24px，形成参差。
 * 侧边栏菜单同级并排时（运维管理 → 工单系统 / 值班管理 / 知识库文章）该错位
 * 视觉上尤其明显。
 *
 * 方案：统一渲染一个不可见（visibility:hidden）的占位图标，保留占位宽度，
 * 不引入任何可见的额外图标语义。
 *
 * @param iconName - 图标名称
 * @returns 图标节点；无图标时返回透明占位
 */
export function getSidebarIconComponent(iconName?: string | null): React.ReactNode {
  const icon = getIconComponent(iconName);
  if (icon) return icon;
  // 占位用 .anticon 类：AntD 既有规则 `.ant-menu-item .anticon` 已定义
  // min-width/margin-right（图标位宽度），内联 visibility:hidden 使其不可见。
  // 这样不新增任何 CSS 规则即可让同级菜单文字左缩进一致。
  return (
    <span
      className="anticon sidebar-icon-placeholder"
      aria-hidden="true"
      style={{ visibility: "hidden" }}
    />
  );
}

// ========================================
// 获取所有图标列表（扁平化）
// ========================================

/**
 * 获取所有图标的扁平化列表（用于图标选择器）
 * @returns 所有图标名称的数组
 */
export function getAllIcons(): string[] {
  const allIcons: string[] = [];
  for (const category of Object.values(iconCategories)) {
    allIcons.push(...category);
  }
  // 去重
  return Array.from(new Set(allIcons));
}

// ========================================
// 搜索图标
// ========================================

/**
 * 根据关键词搜索图标
 * @param keyword - 搜索关键词
 * @returns 匹配的图标名称数组
 */
export function searchIcons(keyword: string): string[] {
  if (!keyword) return getAllIcons();

  const lowerKeyword = keyword.toLowerCase();
  const allIcons = getAllIcons();

  return allIcons.filter((iconName) => {
    // 匹配图标名称
    if (iconName.toLowerCase().includes(lowerKeyword)) {
      return true;
    }
    // 匹配中文描述
    const description = iconDescriptionMap[iconName];
    if (description && description.includes(keyword)) {
      return true;
    }
    return false;
  });
}

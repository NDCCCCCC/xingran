/**
 * 图标工具函数
 * 用于管理 Ant Design 图标的映射和获取
 */

import * as Icons from "@ant-design/icons";
import type { ComponentType } from "react";

// 动态图标访问的类型定义
type IconName = keyof typeof Icons;
type IconComponentMap = Record<string, ComponentType<unknown>>;

// 过滤掉非组件的导出（如 createFromIconfontCN）
const iconComponentKeys = Object.keys(Icons).filter(
  (key) =>
    typeof (Icons as Record<string, unknown>)[key] === "function" || key !== "createFromIconfontCN"
);

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
    "TimerOutlined",
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
    "ShieldOutlined",
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
  ShieldOutlined: "盾牌",
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

  // 获取所有导出的图标组件（过滤掉工具函数）
  const iconKeys = Object.keys(Icons).filter(
    (key) => key !== "createFromIconfontCN" && key !== "default"
  );

  // 尝试直接匹配
  if (iconKeys.includes(iconName)) {
    const IconComponent = (Icons as unknown as IconComponentMap)[iconName];
    if (IconComponent) {
      return <IconComponent />;
    }
  }

  // 尝试通过 fullIconNameMap 映射
  const mappedName = fullIconNameMap[iconName];
  if (mappedName) {
    // 尝试找到对应的完整图标名
    for (const key of iconKeys) {
      if (
        key.toLowerCase().includes(mappedName.toLowerCase()) ||
        key.toLowerCase() === mappedName + "outlined"
      ) {
        const IconComponent = (Icons as unknown as IconComponentMap)[key];
        if (IconComponent) {
          return <IconComponent />;
        }
      }
    }
  }

  return undefined;
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

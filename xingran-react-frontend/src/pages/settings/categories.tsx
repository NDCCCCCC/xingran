/**
 * 用户设置分类注册表（v1.24 P82 自 index.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，categories 常量数组同文件会破坏 Fast Refresh 边界；
 * 卡片组件再拆至 SettingCards.tsx —— 常量注册表文件不得含组件定义）
 */

import { BgColorsOutlined, DatabaseOutlined, LayoutOutlined } from "@ant-design/icons";
import type { SettingsCategory } from "@/design-system/components/SettingsShell";
import { AppearanceSettingsCard, LayoutSettingsCard, DataSettingsCard } from "./SettingCards";

// ---------- 分类注册表（模块级常量：CLAUDE.md useEffect 依赖稳定纪律） ----------

export const userSettingsCategories: SettingsCategory[] = [
  {
    key: "appearance",
    label: "界面设置",
    icon: <BgColorsOutlined />,
    content: <AppearanceSettingsCard />,
    maxWidth: 760,
  },
  {
    key: "layout",
    label: "布局设置",
    icon: <LayoutOutlined />,
    content: <LayoutSettingsCard />,
    maxWidth: 760,
  },
  {
    key: "data",
    label: "数据设置",
    icon: <DatabaseOutlined />,
    content: <DataSettingsCard />,
    maxWidth: 760,
  },
];

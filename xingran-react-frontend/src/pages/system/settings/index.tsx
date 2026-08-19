/**
 * 系统设置页面（Phase 70 · D-01/D-11 目录合并版）
 * System Settings Page (SettingsShell instance)
 *
 * 结构：
 *   - SettingsShell 共用骨架（与用户设置页同构），左导航 = 邮箱/API/验证码 三分类，
 *     `?cat=` URL 参数驱动（D-03），默认 email。
 *   - 三分类内容为表格/网格类：不限宽撑满（D-02），无 maxWidth。
 *   - cat 值从旧版的 "captcha-background" 收敛为 "captcha"（UI-SPEC L-1）。
 *
 * 合并说明（D-11）：
 *   - 本文件原为三分类子页的 barrel re-export，70-05/70-04/70-03 已把三个子页
 *     （email-config / api-config / captcha-background）改造为 v16 形态并保留在本目录，
 *     本文件改为页面入口后旧 Tabs 壳 src/pages/system/settings-page/ 随之删除，
 *     sys_menu component 由 Migrate209 从 system/settings-page/index 更新为
 *     system/settings/index（path 字段不动，路由 URL 保持 /system/settings-page）。
 *   - 旧 usePersistedStateController activeTab 持久化被 SettingsShell 的
 *     useSearchParams 唯一真相源取代（D-03/D-12）。
 */

import type { FC } from "react";
import { ApiOutlined, MailOutlined, PictureOutlined } from "@ant-design/icons";
import { SettingsShell, type SettingsCategory } from "@/design-system/components/SettingsShell";
import EmailConfigPage from "./email-config";
import APIConfigPage from "./api-config";
import CaptchaBackgroundSettingsPage from "./captcha-background";

// ---------- 分类注册表（模块级常量：CLAUDE.md useEffect 依赖稳定纪律） ----------

export const systemSettingsCategories: SettingsCategory[] = [
  {
    key: "email",
    label: "邮箱配置",
    icon: <MailOutlined />,
    content: <EmailConfigPage />,
  },
  {
    key: "api",
    label: "API配置",
    icon: <ApiOutlined />,
    content: <APIConfigPage />,
  },
  {
    key: "captcha",
    label: "验证码背景图",
    icon: <PictureOutlined />,
    content: <CaptchaBackgroundSettingsPage />,
  },
];

// ---------- 页面入口 ----------

const SystemSettingsPage: FC = () => {
  return <SettingsShell categories={systemSettingsCategories} defaultCat="email" />;
};

export default SystemSettingsPage;

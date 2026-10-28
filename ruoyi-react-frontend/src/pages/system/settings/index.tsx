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
 *   - 旧 sessionStorage activeTab 持久化被 SettingsShell 的
 *     useSearchParams 唯一真相源取代（D-03/D-12）。
 */

import type { FC } from "react";
import { SettingsShell } from "@/design-system/components/SettingsShell";
// 分类注册表自 v1.24 P82 拆至 ./categories（react-refresh 组件文件只导组件）
import { systemSettingsCategories } from "./categories";

// ---------- 页面入口 ----------

const SystemSettingsPage: FC = () => {
  return <SettingsShell categories={systemSettingsCategories} defaultCat="email" />;
};

export default SystemSettingsPage;

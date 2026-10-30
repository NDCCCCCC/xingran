/**
 * 系统设置分类注册表（v1.24 P82 自 index.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，categories 常量数组同文件会破坏 Fast Refresh 边界）
 *
 * 左导航 = 邮箱/API/验证码 三分类，内容为表格/网格类：不限宽撑满（D-02），无 maxWidth。
 */

import { ApiOutlined, MailOutlined, PictureOutlined } from "@ant-design/icons";
import type { SettingsCategory } from "@/design-system/components/SettingsShell";
import EmailConfigPage from "./email-config";
import APIConfigPage from "./api-config";
import CaptchaBackgroundSettingsPage from "./captcha-background";

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

/**
 * 系统设置分类注册表（v1.24 P82 自 index.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，categories 常量数组同文件会破坏 Fast Refresh 边界）
 *
 * v1.56 目录化：设置页改为「左目录 + 右侧全部分类纵向平铺 + 锚点跳转」单页，
 * 分类从 3 个扩展为 5 个 —— 原独立菜单页「经理同步」（规则 + 机构两页合并）与
 * 「数据备份与恢复」并入本页成为分类区块。原两处菜单由 migration_241 隐藏。
 *
 * 权限显隐：
 *   - 经理同步：无 `ops:manager:rule:list` 时整个目录项与区块隐藏（任务硬要求）。
 *   - 备份与恢复：**不设** permission —— 该菜单的 F 按钮码（system:backup:list/export/restore）
 *     在存量库中未必授给超管角色（migration_240 刻意不写 sys_role_menu，超管靠后端旁路），
 *     若按 F 码隐藏整块，超管会**彻底失去备份功能入口**。区块常驻、按钮级仍按
 *     export/restore 门控（见 backup-restore.tsx）。
 *   - 邮箱 / API / 验证码背景图：无独立权限码（都挂在「系统设置」C 菜单 system:config:list 下），
 *     与改造前一致常驻显示。
 */

import {
  ApiOutlined,
  CloudDownloadOutlined,
  MailOutlined,
  PictureOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { SettingsCategory } from "@/design-system/components/SettingsShell";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import EmailConfigPage from "./email-config";
import APIConfigPage from "./api-config";
import CaptchaBackgroundSettingsPage from "./captcha-background";
import ManagerSyncSettings from "./manager-sync";
import BackupRestoreSettings from "./backup-restore";

export const systemSettingsCategories: SettingsCategory[] = [
  {
    key: "email",
    label: "邮箱配置",
    icon: <MailOutlined />,
    content: <EmailConfigPage />,
    description: "SMTP 通知邮件服务器配置，支持多套配置与发信测试。",
  },
  {
    key: "api",
    label: "API通知配置",
    icon: <ApiOutlined />,
    content: <APIConfigPage />,
    description: "短信 / Webhook / 推送等通知渠道的接口凭据配置。",
  },
  {
    key: "captcha",
    label: "验证码背景图",
    icon: <PictureOutlined />,
    content: <CaptchaBackgroundSettingsPage />,
    description: "登录验证码背景图素材库，支持启用禁用与抽取使用。",
  },
  {
    key: "manager",
    label: "经理同步",
    icon: <TeamOutlined />,
    content: <ManagerSyncSettings />,
    description: "部门联系人 / 岗位到 AD manager 属性的同步机构定义与规则维护。",
    permission: PERMISSION_CODES.opsManagerRule.list,
  },
  {
    key: "backup",
    label: "备份与恢复",
    icon: <CloudDownloadOutlined />,
    content: <BackupRestoreSettings />,
    description: "数据库全量备份、备份计划与导入恢复。",
  },
];

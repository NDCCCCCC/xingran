/**
 * 数据备份与恢复（旧独立菜单页 → 薄壳）。
 *
 * 页面实现已整体迁入系统设置页的「备份与恢复」分类区块：
 *   src/pages/system/settings/backup-restore.tsx
 *
 * 本文件仅保留 re-export，使旧路由（sys_menu component = system/backup/index）与
 * 既有引用仍可打开同一实现；菜单入口将隐藏，统一从「系统设置」进入。
 */
import BackupRestoreSettings from "@/pages/system/settings/backup-restore";

export default BackupRestoreSettings;

/**
 * 数据备份与恢复页面内部共享类型与展示常量。
 * 仅承载本页 Tabs / 异步任务 / 文案映射，业务实体类型一律取自 @/types/backup。
 */
import type { Plan } from "@/types/backup";

/** 两个 Tab 的 key（配合 usePersistedStateController 记忆 activeTab） */
export type BackupTabKey = "list" | "plan";

/** 触发异步任务的动作类型 */
export type AsyncTaskKind = "export" | "restore";

/** 父页面轮询所需的任务描述（taskId 稳定，避免 useEffect 反复重建定时器） */
export interface AsyncTask {
  taskId: string;
  kind: AsyncTaskKind;
}

/**
 * 敏感字段脱敏占位符。
 * 与后端约定：回填展示 "******"；提交时若仍为该值，前端改写为 ""（空串），
 * 后端收到 "" 表示「保留原值、不覆盖」（后端 SavePlan 亦做保留）。
 */
export const SENSITIVE_PLACEHOLDER = "******";
export const SENSITIVE_KEEP_ORIGINAL = "";

/** 计划表单初始值（GET /plan 失败或首次进入时的兜底） */
export const DEFAULT_PLAN: Plan = {
  enabled: false,
  cron: "0 0 2 * * *",
  retention: 7,
  storage: "local",
  oss: { endpoint: "", bucket: "", accessKey: "", secretKey: "", useSSL: true },
  ftp: { host: "", port: 21, user: "", password: "", remoteDir: "/" },
  smb: { path: "" },
};

/** 存储后端下拉（固定 4 项，禁搜） */
export const STORAGE_OPTIONS = [
  { value: "local", label: "本地磁盘" },
  { value: "oss", label: "OSS / S3 对象存储" },
  { value: "ftp", label: "FTP 服务器" },
  { value: "smb", label: "SMB 共享（OS 已挂载目录）" },
] as const;

/** 备份来源文案 */
export const SOURCE_TEXT: Record<string, string> = {
  manual: "手动备份",
  cron: "定时备份",
  pre_restore: "恢复前备份",
};

/** 记录状态文案与标签色 */
export const STATUS_TEXT: Record<string, string> = {
  ready: "就绪",
  remote_only: "仅远端",
  deleted: "已删除",
};
export const STATUS_TAG_COLOR: Record<string, string> = {
  ready: "success",
  remote_only: "processing",
  deleted: "default",
};

/** 存储后端文案（列表列展示） */
export const STORAGE_TEXT: Record<string, string> = {
  local: "本地",
  oss: "OSS",
  ftp: "FTP",
  smb: "SMB",
};

/** 异步任务状态文案与标签色（四态状态机） */
export const TASK_STATUS_TEXT: Record<string, string> = {
  pending: "等待中",
  running: "执行中",
  success: "成功",
  failed: "失败",
};
export const TASK_STATUS_COLOR: Record<string, string> = {
  pending: "default",
  running: "processing",
  success: "success",
  failed: "error",
};

/** 二次确认正文（破坏性操作，必须显式告知覆盖语义与自动恢复前备份） */
export const RESTORE_CONFIRM_TEXT =
  "恢复将覆盖现有全部数据、期间不可写入、恢复后需重新登录，操作前系统会自动生成一份恢复前备份。确认使用 {target} 进行恢复吗？";

/**
 * SQLite 整库恢复的离线执行入口提示。
 *
 * 依据 dbBackupApi.capabilities() 返回的 dbType === "sqlite" 常驻显示（列表页 + RestoreModal），
 * 不能只在失败后把命令塞进 errorMessage。SQLite 在线服务的连接池持有库文件句柄，
 * 整库替换必须停服后由离线 CLI 完成，在线端点只会把产物落到 `.pending_restore`
 * 并降级报错（见后端 restorer.go）。命令里的 `<sqlite-db-path>` / `<pending-file>`
 * 为占位符，真实路径以后端失败信息为准。
 */
export const SQLITE_OFFLINE_NOTICE =
  "SQLite 整库恢复需停服离线执行：在线端点只生成 .pending_restore 产物，不会替换运行中的库。请停服后执行离线恢复命令，再启动服务并重新登录。";

export const SQLITE_OFFLINE_COMMAND =
  "go run ./scripts/ops/db-backup-restore.go -db <sqlite-db-path> -pending <pending-file>";

/** 备份包含全库敏感数据，下载前二次提示 */
export const DOWNLOAD_SENSITIVE_CONFIRM_TEXT =
  "备份包含全库敏感数据（用户口令散列、个人信息、业务数据等），下载即等同于导出整库明文。请确认已获得授权并妥善保管，确认下载 {target} 吗？";

/**
 * 「需重新登录」常驻提示（后端无全局会话失效入口的最小实现）。
 * 后端整库恢复会替换 sys_user/sys_menu/sys_role 等鉴权表，进程内菜单/权限缓存
 * 会被清理，但已签发的 JWT 无法由服务端统一作废，故要求所有已登录会话重新登录。
 */
export const RESTORE_RELOGIN_NOTICE =
  "整库恢复会替换用户/角色/菜单等鉴权数据，恢复后所有已登录会话需重新登录以重建会话与权限缓存。";

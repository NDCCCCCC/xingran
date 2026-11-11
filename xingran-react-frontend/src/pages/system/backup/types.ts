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
  "恢复将覆盖现有全部数据，操作前系统会自动生成一份恢复前备份。确认使用 {target} 进行恢复吗？";

/**
 * 数据备份与恢复模块类型定义。
 * 字段与 .planning/quick/261110-tbq-oss-ftp-smb-cron/261110-tbq-CONTRACT.md
 * 第 3/4/5 节保持一致（camelCase；时间字段统一按 string 承接后端 JSON 时间串）。
 */

/** 存储后端类型，对应后端 dbbackup.StorageType */
export type StorageType = "local" | "oss" | "ftp" | "smb";

/** OSS / S3 兼容对象存储配置 */
export interface OSSConfig {
  endpoint: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
  useSSL: boolean;
}

/** FTP 配置 */
export interface FTPConfig {
  host: string;
  port: number;
  user: string;
  password: string;
  remoteDir: string;
}

/** SMB 配置：path 为部署机 OS 已挂载共享后的本地目录绝对路径 */
export interface SMBConfig {
  path: string;
}

/** 备份计划与存储配置（对应后端 dbbackup.Plan） */
export interface Plan {
  enabled: boolean;
  cron: string;
  retention: number;
  storage: StorageType;
  oss: OSSConfig;
  ftp: FTPConfig;
  smb: SMBConfig;
}

/** 备份记录视图（对应后端 dbbackup.BackupRecord） */
export interface BackupRecord {
  id: string;
  fileName: string;
  filePath: string;
  storageType: string;
  remotePath: string;
  dbType: string;
  fileSize: number;
  sha256: string;
  schemaVersion: string;
  tableCount: number;
  source: string;
  status: string;
  triggeredBy: string;
  remark: string;
  createdAt: string;
}

/** 异步任务状态视图（对应后端 dbbackup.TaskView） */
export interface TaskView {
  id: string;
  taskType: string;
  backupId: string;
  status: string;
  progress: number;
  stage: string;
  errorMessage: string;
  startedAt: string | null;
  completedAt: string | null;
  createdAt: string;
}

/** 能力预检（GET /capabilities 返回，对应后端 dbbackup.Capabilities） */
export interface Capabilities {
  dbType: string;
  pgDumpAvailable: boolean;
  pgRestoreAvailable: boolean;
  pgDumpPath: string;
  pgRestorePath: string;
  hint: string;
}

/** 统计卡数据（statistics 端点为 map，按 tolerant 形态声明） */
export interface BackupStatistics {
  total?: number;
  success?: number;
  usedSpace?: number;
  [key: string]: unknown;
}

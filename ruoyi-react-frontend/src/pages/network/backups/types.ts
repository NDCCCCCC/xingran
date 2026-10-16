/**
 * 配置备份类型定义
 */

import type { ConfigBackup } from "@/types";

// 差异行类型
export interface DiffLine {
  type: "same" | "removed" | "added" | "empty";
  content: string;
  lineNum?: number;
}

// 差异对比结果
export interface DiffResult {
  leftContent: string;
  rightContent: string;
  leftLines: DiffLine[];
  rightLines: DiffLine[];
  oldVersion: string;
  newVersion: string;
}

// 设备备份分组
export interface DeviceBackupGroup {
  deviceId: string;
  deviceName: string;
  ipAddress: string;
  backups: ConfigBackup[];
  latestBackup: ConfigBackup;
  backupCount: number;
  autoCount: number;
  manualCount: number;
}

// 统计数据
export interface BackupStatistics {
  total: number;
  auto: number;
  manual: number;
  devices: number;
}

// 恢复任务状态（与后端 models.RestoreTaskStatus 四态对齐，Phase 93 D-34）
export type RestoreTaskStatus = "pending" | "running" | "success" | "failed";

// 配置恢复任务（Phase 93 异步恢复；字段与后端 ConfigRestoreTask model json tag 对齐）
export interface ConfigRestoreTask {
  id: string;
  deviceId: string;
  backupId: string;
  status: RestoreTaskStatus;
  totalLines?: number;
  sentLines?: number;
  failedLine?: string;
  /** JSON 字符串：{ totalLines, sentLines, failedLine?, hashMatched, restoredHash? } */
  resultJson?: string;
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

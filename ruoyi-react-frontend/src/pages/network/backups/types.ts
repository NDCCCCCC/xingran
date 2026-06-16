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

/**
 * 配置备份工具函数
 */

import type { ConfigBackup } from "@/types";
import type { DiffResult, DeviceBackupGroup, DiffLine } from "./types";

/**
 * 计算两个文本的差异（使用简化的 LCS 算法）
 */
export function computeDiff(content1: string, content2: string): DiffResult {
  const lines1 = content1.split("\n");
  const lines2 = content2.split("\n");

  // 使用动态规划计算最长公共子序列
  const m = lines1.length;
  const n = lines2.length;
  const dp: number[][] = Array(m + 1)
    .fill(0)
    .map(() => Array(n + 1).fill(0));

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (lines1[i - 1] === lines2[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
      }
    }
  }

  // 回溯生成差异
  const leftLines: DiffLine[] = [];
  const rightLines: DiffLine[] = [];
  let i = m,
    j = n;

  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && lines1[i - 1] === lines2[j - 1]) {
      // 相同的行
      leftLines.unshift({ type: "same", content: lines1[i - 1], lineNum: i });
      rightLines.unshift({ type: "same", content: lines2[j - 1], lineNum: j });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      // 新增的行
      leftLines.unshift({ type: "empty", content: "", lineNum: undefined });
      rightLines.unshift({ type: "added", content: lines2[j - 1], lineNum: j });
      j--;
    } else {
      // 删除的行
      leftLines.unshift({ type: "removed", content: lines1[i - 1], lineNum: i });
      rightLines.unshift({ type: "empty", content: "", lineNum: undefined });
      i--;
    }
  }

  return {
    leftContent: content1,
    rightContent: content2,
    leftLines,
    rightLines,
    oldVersion: "",
    newVersion: "",
  };
}

/**
 * 将备份列表按设备分组
 */
export function groupBackupsByDevice(backups: ConfigBackup[]): DeviceBackupGroup[] {
  const groupMap = new Map<string, ConfigBackup[]>();

  // 按设备ID分组
  backups.forEach((backup) => {
    if (!groupMap.has(backup.deviceId)) {
      groupMap.set(backup.deviceId, []);
    }
    groupMap.get(backup.deviceId)!.push(backup);
  });

  // 转换为分组数组
  const groups: DeviceBackupGroup[] = [];
  groupMap.forEach((deviceBackups, deviceId) => {
    // 按创建时间倒序排序
    deviceBackups.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

    const latestBackup = deviceBackups[0];
    const autoCount = deviceBackups.filter((b) => b.backupType === "auto").length;
    const manualCount = deviceBackups.filter((b) => b.backupType === "manual").length;

    groups.push({
      deviceId,
      deviceName: latestBackup.deviceName,
      ipAddress: latestBackup.ipAddress,
      backups: deviceBackups,
      latestBackup,
      backupCount: deviceBackups.length,
      autoCount,
      manualCount,
    });
  });

  return groups;
}

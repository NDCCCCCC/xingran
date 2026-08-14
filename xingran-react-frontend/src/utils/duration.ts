/**
 * 秒转人类可读时长(项目级工具,补充 datetime.ts 仅有日期/时间格式化的不足)
 *
 * 规则:
 * - 0 / 负数 / NaN / null / undefined -> "0s"(瞬时场景,与 Gantt minWidth 兜底视觉一致)
 * - < 60s   -> "Ns"  (如 "45s")
 * - < 60m   -> "Nm"  (如 "15m")
 * - < 24h   -> "Nh"  (如 "42h")
 * - >= 24h  -> "Nd Nh" (如 "3d 4h")
 *
 * 注意:src/pages/monitor/job/utils.tsx 的 formatDuration 是毫秒级(定时任务执行耗时),
 * 本函数是秒级(MAC 停留时长),职责分离。
 */
export function formatDurationSeconds(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds <= 0) return "0s";
  const sec = Math.floor(seconds);
  const days = Math.floor(sec / 86400);
  const hours = Math.floor((sec % 86400) / 3600);
  const minutes = Math.floor((sec % 3600) / 60);

  if (days > 0) return `${days}d ${hours}h`; // 151200 -> "1d 18h"
  if (hours > 0) return `${hours}h`; // 3600 -> "1h"
  if (minutes > 0) return `${minutes}m`; // 900 -> "15m"
  return `${sec % 60}s`; // 45 -> "45s"
}

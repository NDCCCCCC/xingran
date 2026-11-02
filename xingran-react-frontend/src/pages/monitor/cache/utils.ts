/**
 * Cache 工具函数
 */

export { formatDateTime } from "@/utils/datetime";

export function formatMemorySize(bytes: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

export function formatTTL(seconds: number): string {
  if (seconds <= 0) return "永久";

  const units = [
    { name: "秒", value: 60 },
    { name: "分钟", value: 60 },
    { name: "小时", value: 24 },
    { name: "天", value: Infinity },
  ];

  let value = seconds;
  let unitIndex = 0;

  while (unitIndex < units.length - 1 && value >= units[unitIndex].value) {
    value /= units[unitIndex].value;
    unitIndex++;
  }

  if (value < 1 && unitIndex > 0) {
    value *= units[unitIndex - 1].value;
    unitIndex--;
  }

  const formatted = Number.isInteger(value) ? value.toString() : value.toFixed(2);
  return `${formatted}${units[unitIndex].name}`;
}

export function exportCacheAsJson(data: unknown[]): void {
  const dataStr = JSON.stringify(data, null, 2);
  const dataUri = "data:application/json;charset=utf-8," + encodeURIComponent(dataStr);
  const exportFileDefaultName = `cache_export_${new Date().toISOString().slice(0, 19).replace(/:/g, "-")}.json`;

  const linkElement = document.createElement("a");
  linkElement.setAttribute("href", dataUri);
  linkElement.setAttribute("download", exportFileDefaultName);
  linkElement.click();
}

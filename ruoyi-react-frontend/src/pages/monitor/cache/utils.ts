/**
 * Cache 工具函数
 */

/**
 * 格式化内存大小
 */
export function formatMemorySize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

/**
 * 格式化TTL时间
 */
export function formatTTL(seconds: number): string {
  if (seconds <= 0) {
    return '永久';
  }

  const units = [
    { name: '秒', value: 60 },
    { name: '分钟', value: 60 },
    { name: '小时', value: 24 },
    { name: '天', value: Infinity }
  ];

  let value = seconds;
  let unitIndex = 0;

  while (unitIndex < units.length - 1 && value >= units[unitIndex].value) {
    value /= units[unitIndex].value;
    unitIndex++;
  }

  // 如果小于1，显示为更小的单位
  if (value < 1 && unitIndex > 0) {
    value *= units[unitIndex - 1].value;
    unitIndex--;
  }

  // 格式化：如果整数就不显示小数，否则最多显示2位小数
  const formatted = Number.isInteger(value) ? value.toString() : value.toFixed(2);

  return `${formatted}${units[unitIndex].name}`;
}

/**
 * 格式化时间戳
 */
export function formatDateTime(time: string): string {
  return time ? new Date(time).toLocaleString() : '-';
}

/**
 * 导出缓存数据为 JSON 文件
 */
export function exportCacheAsJson(data: unknown[], searchForm: Record<string, unknown>): void {
  const dataStr = JSON.stringify(data, null, 2);
  const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);
  const exportFileDefaultName = `cache_export_${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;

  const linkElement = document.createElement('a');
  linkElement.setAttribute('href', dataUri);
  linkElement.setAttribute('download', exportFileDefaultName);
  linkElement.click();
}

/**
 * CAD 坐标工具函数
 * 坐标取整到 0.01，防止 SVG 亚像素抖动
 */

/** 坐标取整到 0.01 */
export function snapCoord(v: number): number {
  return Math.round(v * 100) / 100;
}

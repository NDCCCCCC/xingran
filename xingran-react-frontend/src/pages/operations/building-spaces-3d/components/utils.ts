/**
 * 3D 楼宇视图组件共享工具函数
 */

import type { BuildingItem } from "./types";
import { MARKER_COLORS, STATUS_TEXT } from "./constants";
import type { BMapPoint } from "@/types/baidu-map";
import type { DictItem } from "@/hooks/useDict";
import { buildDictLabelIndex, resolveDictLabel } from "@/components/reconciliation/dictLabels";

/**
 * Base64 编码（正确处理 Unicode 字符）
 */
export function toBase64(str: string): string {
  try {
    return btoa(unescape(encodeURIComponent(str)));
  } catch {
    return btoa(str);
  }
}

/**
 * 获取楼宇标记颜色
 */
export function getBuildingMarkerColors(status: number) {
  const isStopped = status === 1;
  return {
    main: isStopped ? MARKER_COLORS.STOPPED.MAIN : MARKER_COLORS.NORMAL.MAIN,
    border: isStopped ? MARKER_COLORS.STOPPED.BORDER : MARKER_COLORS.NORMAL.BORDER,
    shadow: isStopped ? MARKER_COLORS.STOPPED.SHADOW : MARKER_COLORS.NORMAL.SHADOW,
  };
}

/**
 * 获取楼宇状态文本
 */
export function getBuildingStatusText(status: number): string {
  return status === 0 ? STATUS_TEXT.BUILDING.NORMAL : STATUS_TEXT.BUILDING.STOPPED;
}

/**
 * 获取楼层状态文本
 */
export function getFloorStatusText(status: number): string {
  return status === 0 ? STATUS_TEXT.FLOOR.NORMAL : STATUS_TEXT.FLOOR.STOPPED;
}

/**
 * 获取工位状态文本
 */
export function getWorkstationStatusText(status: number): string {
  switch (status) {
    case 0:
      return STATUS_TEXT.WORKSTATION.AVAILABLE;
    case 1:
      return STATUS_TEXT.WORKSTATION.OCCUPIED;
    case 2:
      return STATUS_TEXT.WORKSTATION.MAINTENANCE;
    default:
      return "未知";
  }
}

/**
 * 工位类型文案静态回退映射。
 * 工位类型是数据字典 ops_workstation_type，管理员可新增类型（如 3=虚拟工位）；
 * 字典为空 / 接口异常 / 未命中时沿用本映射，保证字典接入前后展示行为一致。
 */
const WORKSTATION_TYPE_FALLBACK_LABELS: Record<string, string> = {
  "0": STATUS_TEXT.WORKSTATION.FIXED,
  "1": STATUS_TEXT.WORKSTATION.FLEXIBLE,
  "2": STATUS_TEXT.WORKSTATION.MANAGEMENT,
};

/**
 * 获取工位类型文本（字典驱动，三级回退）。
 * 1) 字典命中（dictValue → dictLabel）优先；
 * 2) 字典未命中降级静态映射（固定 / 灵活 / 管理）；
 * 3) 两者皆无保留原兜底文案「未知」（不吞成空串）。
 * 字典数据由上层 React 组件 useDict("ops_workstation_type") 后透传；
 * 不传 dictItems 时等价于原静态 switch（既有调用 / 测试零变更）。
 * @param type 类型值
 * @param dictItems 可选字典项数组
 * @returns 类型文本
 */
export function getWorkstationTypeText(type: number, dictItems?: ReadonlyArray<DictItem>): string {
  const { labels } = buildDictLabelIndex(dictItems ? [...dictItems] : undefined);
  const resolved = resolveDictLabel(String(type), labels, WORKSTATION_TYPE_FALLBACK_LABELS);
  // resolveDictLabel 未命中时回原始值，这里还原本模块既有的「未知」兜底文案
  return resolved === String(type) ? "未知" : resolved;
}

/**
 * 获取工位颜色（3D 场景）
 * 颜色按 status / type 数值静态取色，与字典文案解耦；新增工位类型由
 * ops_workstation_type 字典覆盖展示文案，不新增色值。
 */
export function getWorkstationColor(workstation: { status: number; type: number }): number {
  if (workstation.status === 1) return 0xd32f2f; // 占用 - 深红色
  if (workstation.status === 2) return 0xf57c00; // 维护 - 深橙色
  if (workstation.type === 1) return 0x7b1fa2; // 灵活 - 深紫色
  if (workstation.type === 2) return 0x13c2c2; // 管理 - 深青色
  return 0x388e3c; // 空闲固定 - 深绿色
}

/**
 * 获取楼层颜色（3D 场景）
 */
export function getFloorColor(floor: { status: number; workstationCount: number }): number {
  if (floor.status === 1) return 0xd32f2f; // 停用 - 深红色
  if (floor.workstationCount === 0) return 0xf57c00; // 无工位 - 深橙色
  const occupiedRatio = floor.workstationCount > 0 ? 0.3 : 0;
  return occupiedRatio > 0.7 ? 0x388e3c : 0x1976d2; // 绿色或蓝色
}

/**
 * 生成楼宇信息窗口 HTML 内容
 */
export function generateBuildingInfoHTML(building: BuildingItem): string {
  const isStopped = building.status === 1;
  const colors = getBuildingMarkerColors(building.status);
  const statusText = getBuildingStatusText(building.status);

  return `
    <div style="padding: 12px; min-width: 260px;">
      <h3 style="margin: 0 0 12px 0; font-size: 16px; color: var(--theme-info, #337ab0); border-bottom: 2px solid #337ab0; padding-bottom: 8px;">
        ${building.name}
      </h3>
      <div style="font-size: 13px; color: var(--theme-text-tertiary, #666); line-height: 1.8;">
        <div>📌 ${building.address || "暂无地址"}</div>
        <div style="margin-top: 8px;">
          <span style="display: inline-block; padding: 2px 10px; background: ${isStopped ? "#f5f5f5" : "#fff2e8"}; color: ${colors.main}; border-radius: 12px; font-size: 12px; font-weight: 500;">
            ${isStopped ? "⏸ " + statusText : "✓ " + statusText}
          </span>
          <span style="margin-left: 12px;">🏢 ${building.floorCount || 0} 层</span>
          <span style="margin-left: 12px;">🪑 ${building.workstationCount || 0} 工位</span>
        </div>
      </div>
      <div style="margin-top: 12px; text-align: right;">
        <button
          onclick="window.viewBuildingDetails('${building.id}')"
          style="
            padding: 6px 18px;
            background: linear-gradient(135deg, #337ab0, #096dd9);
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 500;
            box-shadow: 0 2px 4px rgba(51, 122, 176, 0.3);
          "
        >
          查看详情
        </button>
      </div>
    </div>
  `;
}

/**
 * 计算两点之间的像素距离
 */
export function calculatePixelDistance(
  p1: { x: number; y: number },
  p2: { x: number; y: number }
): number {
  return Math.sqrt(Math.pow(p1.x - p2.x, 2) + Math.pow(p1.y - p2.y, 2));
}

/**
 * 生成聚类图标 SVG
 */
export function generateClusterIconSVG(count: number, size: number): string {
  return `
    <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
      <defs>
        <filter id="clusterShadow" x="-50%" y="-50%" width="200%" height="200%">
          <feDropShadow dx="0" dy="2" stdDeviation="3" flood-color="rgba(186, 54, 48, 0.5)" flood-opacity="0.7"/>
        </filter>
      </defs>
      <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 1}" fill="rgba(186, 54, 48, 0.1)"/>
      <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 4}" fill="#ba3630" stroke="#fff" stroke-width="3" filter="url(#clusterShadow)"/>
      <text x="${size / 2}" y="${size / 2 + 8}" text-anchor="middle" fill="white" font-size="${size / 3}" font-weight="bold">${count}</text>
    </svg>
  `;
}

/**
 * 生成楼宇标记 SVG
 */
export function generateBuildingMarkerSVG(
  buildingName: string,
  colors: { main: string; border: string; shadow: string }
): string {
  const label = buildingName ? buildingName.slice(0, 2) : "楼宇";

  return `
    <svg xmlns="http://www.w3.org/2000/svg" width="42" height="48" viewBox="0 0 42 48">
      <defs>
        <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
          <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="${colors.shadow}" flood-opacity="0.6"/>
        </filter>
      </defs>
      <path d="M21 4 L21 4 Q21 4 21 4 C9 4 4 14 4 22 C4 32 21 44 21 44 C21 44 38 32 38 22 C38 14 33 4 21 4 Z"
        fill="${colors.main}" stroke="${colors.border}" stroke-width="3" filter="url(#shadow)"/>
      <circle cx="21" cy="20" r="8" fill="white" opacity="0.3"/>
      <text x="21" y="25" text-anchor="middle" fill="white" font-size="11" font-weight="bold">${label}</text>
    </svg>
  `;
}

/**
 * 将边界字符串转换为地图点数组
 */
export function parseBoundaryPoints<T extends new (lng: number, lat: number) => BMapPoint>(
  boundaryStr: string,
  PointClass: T
): BMapPoint[] {
  return boundaryStr.split(";").map((point) => {
    const [lng, lat] = point.split(",").map(Number);
    return new PointClass(lng, lat);
  });
}

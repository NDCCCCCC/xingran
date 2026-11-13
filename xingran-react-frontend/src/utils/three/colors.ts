import type { DictItem } from "@/hooks/useDict";
import { buildDictLabelIndex, resolveDictLabel } from "@/components/reconciliation/dictLabels";

// 颜色方案
// 值冻结自 legacy 品牌色导出（Phase 65 · THEME-01 删除前原样内联），
// 3D 接品牌令牌属 VIS-01 (v1.23+)，本 phase 3D 视觉零变化。
const legacyBrandColors = {
  primary: "#2563eb",
  secondary: "#a855f7",
  success: "#22c55e",
  warning: "#f59e0b",
  error: "#ef4444",
  info: "#3b82f6",
} as const;

/**
 * 工位状态颜色（同步系统主题颜色）
 */
export const WORKSTATION_STATUS_COLORS = {
  0: {
    main: legacyBrandColors.success, // 绿色 - 空闲
    label: "空闲",
    bg: "#f6ffed",
    border: "#b7eb8f",
  },
  1: {
    main: legacyBrandColors.error, // 红色 - 占用/禁用
    label: "占用",
    bg: "#fff1f0",
    border: "#ffccc7",
  },
  2: {
    main: legacyBrandColors.warning, // 橙色 - 维护
    label: "维护",
    bg: "#fffbe6",
    border: "#ffe58f",
  },
} as const;

/**
 * 工位类型颜色（同步系统主题颜色）
 * label 仅作为字典空态 / 接口异常时的静态回退；运行时展示文案以
 * ops_workstation_type 字典为准（见 resolveWorkstationTypeLabel 与
 * getWorkstationTypeColor 的 dictItems 参数），管理员新增类型不会被写死文案挡住。
 */
export const WORKSTATION_TYPE_COLORS = {
  0: {
    main: legacyBrandColors.primary, // 蓝色 - 固定
    label: "固定",
    bg: "#e6f7ff",
    border: "#91d5ff",
  },
  1: {
    main: legacyBrandColors.secondary, // 紫色 - 灵活
    label: "灵活",
    bg: "#f9f0ff",
    border: "#d3adf7",
  },
  2: {
    main: legacyBrandColors.info, // 青色 - 管理
    label: "管理",
    bg: "#e6fffb",
    border: "#87e8de",
  },
} as const;

/**
 * 工位类型文案静态回退（取自 WORKSTATION_TYPE_COLORS.label）。
 * 与 dictLabels 的三级回退口径一致：字典 → 本映射 → 原始值。
 */
export const WORKSTATION_TYPE_LABELS: Record<string, string> = {
  "0": WORKSTATION_TYPE_COLORS[0].label,
  "1": WORKSTATION_TYPE_COLORS[1].label,
  "2": WORKSTATION_TYPE_COLORS[2].label,
};

/**
 * 地图标记颜色
 */
export const MAP_MARKER_COLORS = {
  city: {
    main: "#1890ff", // 蓝色 - 城市标记
    bg: "#e6f7ff",
    border: "#91d5ff",
  },
  building: {
    normal: "#52c41a", // 绿色 - 正常楼宇
    stopped: "#9e9e9e", // 灰色 - 停用
    bg: "#f6ffed",
    border: "#b7eb8f",
  },
  selected: {
    main: "#ff4d4f", // 红色 - 选中
    bg: "#fff1f0",
    border: "#ffccc7",
  },
} as const;

/**
 * 楼宇 3D 模型颜色
 */
export const BUILDING_3D_COLORS = {
  normal: 0x667eea, // 正常
  stopped: 0x9e9e9e, // 停用
  hover: 0x764ba2, // 悬停
  selected: 0xff6b6b, // 选中
} as const;

/**
 * 获取工位状态颜色
 */
export function getWorkstationStatusColor(status: number) {
  return (
    WORKSTATION_STATUS_COLORS[status as keyof typeof WORKSTATION_STATUS_COLORS] ||
    WORKSTATION_STATUS_COLORS[0]
  );
}

/**
 * 解析工位类型展示文案（字典驱动，三级回退）。
 * 1) ops_workstation_type 字典命中（dictValue → dictLabel）优先；
 * 2) 未命中降级 WORKSTATION_TYPE_LABELS 静态映射；
 * 3) 两者皆无显示原始值（不吞成空串）。
 * @param type 类型值
 * @param dictItems 可选字典项数组，由上层 React 组件 useDict 提供
 */
export function resolveWorkstationTypeLabel(
  type: number,
  dictItems?: ReadonlyArray<DictItem>
): string {
  const { labels } = buildDictLabelIndex(dictItems ? [...dictItems] : undefined);
  return resolveDictLabel(String(type), labels, WORKSTATION_TYPE_LABELS);
}

/**
 * 获取工位类型颜色。
 * 颜色按数字静态取色（口径：颜色仍可静态）；仅 label 在传入字典时被覆盖。
 * 不传 dictItems 时返回 WORKSTATION_TYPE_COLORS 原对象（保持既有引用相等语义）。
 * @param type 类型值
 * @param dictItems 可选字典项数组
 */
export function getWorkstationTypeColor(type: number, dictItems?: ReadonlyArray<DictItem>) {
  const color =
    WORKSTATION_TYPE_COLORS[type as keyof typeof WORKSTATION_TYPE_COLORS] ||
    WORKSTATION_TYPE_COLORS[0];
  if (!dictItems) return color;
  return { ...color, label: resolveWorkstationTypeLabel(type, dictItems) };
}

/**
 * 获取地图标记颜色
 */
export function getMapMarkerColor(type: "city" | "building", status?: number) {
  if (type === "city") {
    return MAP_MARKER_COLORS.city;
  }

  if (status === 1) {
    return { ...MAP_MARKER_COLORS.building, main: MAP_MARKER_COLORS.building.stopped };
  }

  return MAP_MARKER_COLORS.building;
}

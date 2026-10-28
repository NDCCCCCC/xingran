/**
 * 颜色预设常量（v1.24 P82 自 ColorSwitcher.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，常量数组同文件会破坏 Fast Refresh 边界）
 */

/**
 * 颜色预设类型
 */
export interface ColorPreset {
	name: string;
	primary: string;
	category?: "primary" | "accent" | "neutral" | "custom";
}

/**
 * 预设主色调方案
 */
export const primaryColorPresets: ColorPreset[] = [
	// 主要色系
	{ name: "蓝色", primary: "#2563eb", category: "primary" },
	{ name: "靛蓝", primary: "#4f46e5", category: "primary" },
	{ name: "紫色", primary: "#7c3aed", category: "primary" },
	{ name: "粉色", primary: "#db2777", category: "accent" },

	// 功能色系
	{ name: "红色", primary: "#dc2626", category: "accent" },
	{ name: "橙色", primary: "#ea580c", category: "accent" },
	{ name: "琥珀", primary: "#d97706", category: "accent" },
	{ name: "绿色", primary: "#16a34a", category: "primary" },

	// 青色系
	{ name: "青色", primary: "#0891b2", category: "primary" },
	{ name: "蓝绿", primary: "#0d9488", category: "primary" },
];

/**
 * 预设侧边栏颜色方案
 */
export const sidebarColorPresets: ColorPreset[] = [
	// 深色系（配浅色文字）
	{ name: "深蓝灰", primary: "#1E293B", category: "neutral" },
	{ name: "深灰", primary: "#374151", category: "neutral" },
	{ name: "墨黑", primary: "#0F172A", category: "neutral" },
	{ name: "靛蓝深", primary: "#1E1B4B", category: "primary" },
	{ name: "森林绿", primary: "#14532D", category: "primary" },
	{ name: "酒红", primary: "#450A0A", category: "accent" },

	// 浅色系（配深色文字）
	{ name: "纯白", primary: "#FFFFFF", category: "neutral" },
	{ name: "浅灰", primary: "#F8FAFC", category: "neutral" },
	{ name: "象牙白", primary: "#FFFBEB", category: "neutral" },
	{ name: "淡蓝", primary: "#EFF6FF", category: "primary" },
	{ name: "薄荷绿", primary: "#F0FDF4", category: "primary" },
	{ name: "樱花粉", primary: "#FFF1F2", category: "accent" },
];

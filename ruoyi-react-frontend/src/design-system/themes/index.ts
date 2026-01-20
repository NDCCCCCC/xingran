/**
 * 主题系统导出（重构版 - 支持明暗双模式）
 * Unified Theme System Export - Support Light/Dark Modes
 */

import type { ThemeConfig, ThemeType } from '@/types/theme';
import { getMinimalTheme } from './minimal';
import { getGlassmorphismTheme } from './glassmorphism';
import { getNeumorphismTheme } from './neumorphism';
import { getFlat2Theme } from './flat2.0';
import { getLuxuryQuietTheme } from './luxury-quiet';
import { getLuminance, getContrastTextColor, getHoverBackgroundColor, generateColorVariants } from '@/design-system/utils/color';

/**
 * 主题变体集合（包含 light 和 dark）
 */
export interface ThemeVariants {
	light: ThemeConfig;
	dark: ThemeConfig;
}

// 重新导出 ColorMode 类型供外部使用
export type { ColorMode };

/**
 * 所有主题配置（支持模式切换）
 */
export function getTheme(type: ThemeType, mode: ColorMode = 'light'): ThemeConfig {
	switch (type) {
		case 'minimal':
			return getMinimalTheme(mode);
		case 'glassmorphism':
			return getGlassmorphismTheme(mode);
		case 'neumorphism':
			return getNeumorphismTheme(mode);
		case 'flat2.0':
			return getFlat2Theme(mode);
		case 'luxury-quiet':
			return getLuxuryQuietTheme(mode);
		default:
			return getMinimalTheme(mode);
	}
}

/**
 * 主题预设信息（用于UI展示）
 */
export const themePresets: Array<{
	id: ThemeType;
	name: string;
	icon: string;
	description: string;
	preview: {
		light: string;
		dark: string;
	};
}> = [
	{
		id: 'minimal',
		name: '极简现代',
		icon: '◐',
		description: '干净的线条、大量留白、高对比度',
		preview: {
			light: 'linear-gradient(135deg, #ffffff 0%, #f5f5f5 100%)',
			dark: 'linear-gradient(135deg, #0a0a0a 0%, #171717 100%)',
		},
	},
	{
		id: 'glassmorphism',
		name: '玻璃拟态',
		icon: '◑',
		description: '半透明背景、模糊效果、现代感',
		preview: {
			light: 'linear-gradient(135deg, rgba(167, 139, 250, 0.3) 0%, rgba(139, 92, 246, 0.3) 100%)',
			dark: 'linear-gradient(135deg, rgba(10, 10, 20, 0.85) 0%, rgba(26, 26, 46, 0.75) 100%)',
		},
	},
	{
		id: 'neumorphism',
		name: '新拟态',
		icon: '◒',
		description: '柔和阴影浮雕效果、圆润设计',
		preview: {
			light: 'linear-gradient(135deg, #e0e5ec 0%, #d1d9e6 100%)',
			dark: 'linear-gradient(135deg, #1a1f2e 0%, #12151f 100%)',
		},
	},
	{
		id: 'flat2.0',
		name: '扁平化2.0',
		icon: '◓',
		description: '扁平设计+微妙渐变、鲜艳配色',
		preview: {
			light: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
			dark: 'linear-gradient(135deg, #0a0f1e 0%, #1a1f2e 100%)',
		},
	},
	{
		id: 'luxury-quiet',
		name: '静谧奢华',
		icon: '◆',
		description: '深邃炭灰配铜金点缀，优雅奢华质感',
		preview: {
			light: 'linear-gradient(135deg, #fafafa 0%, #f5f5f5 50%, #d4af37 100%)',
			dark: 'linear-gradient(135deg, #0a0e12 0%, #1a1f26 50%, #d4af37 100%)',
		},
	},
];

/**
 * 应用主题CSS变量到DOM
 * @param theme 主题配置
 * @param mode 颜色模式
 */
export function applyThemeVariables(theme: ThemeConfig, mode: ColorMode = 'light'): void {
	const root = document.documentElement;

	// 应用颜色变量
	applyColorVariables(theme.colors);

	// 应用间距变量
	applySpacingVariables(theme.spacing);

	// 应用阴影变量
	applyShadowVariables(theme.shadows);

	// 应用圆角变量
	applyRadiusVariables(theme.radius);

	// 应用特殊效果变量
	applyEffectVariables(theme.effects);

	// 设置 data 属性
	root.setAttribute('data-theme', theme.id);
	root.setAttribute('data-color-mode', mode);
}

/**
 * 应用颜色变量
 */
function applyColorVariables(colors: ThemeConfig['colors']): void {
	const root = document.documentElement;

	// 背景色
	root.style.setProperty('--theme-bg-primary', colors.background.primary);
	root.style.setProperty('--theme-bg-secondary', colors.background.secondary);
	root.style.setProperty('--theme-bg-tertiary', colors.background.tertiary);
	root.style.setProperty('--theme-bg-surface', colors.background.surface);
	root.style.setProperty('--theme-bg-elevated', colors.background.elevated);

	// 文字色
	root.style.setProperty('--theme-text-primary', colors.text.primary);
	root.style.setProperty('--theme-text-secondary', colors.text.secondary);
	root.style.setProperty('--theme-text-tertiary', colors.text.tertiary);
	root.style.setProperty('--theme-text-disabled', colors.text.disabled);
	root.style.setProperty('--theme-text-inverse', colors.text.inverse);

	// 主色
	if (colors.primary[0]) {
		root.style.setProperty('--theme-primary-50', colors.primary[0]);
		root.style.setProperty('--theme-primary-100', colors.primary[1] || colors.primary[0]);
		root.style.setProperty('--theme-primary-500', colors.primary[2] || colors.primary[0]);
		root.style.setProperty('--theme-primary-700', colors.primary[3] || colors.primary[0]);
	}

	// 边框色
	root.style.setProperty('--theme-border-primary', colors.border.primary);
	root.style.setProperty('--theme-border-secondary', colors.border.secondary);
	root.style.setProperty('--theme-border-divider', colors.border.divider);

	// 功能色
	root.style.setProperty('--theme-success', colors.success[1]);
	root.style.setProperty('--theme-warning', colors.warning[1]);
	root.style.setProperty('--theme-error', colors.error[1]);
	root.style.setProperty('--theme-info', colors.info[1]);
}

/**
 * 应用间距变量
 */
function applySpacingVariables(spacing: ThemeConfig['spacing']): void {
	const root = document.documentElement;

	root.style.setProperty('--theme-spacing-xs', spacing.xs);
	root.style.setProperty('--theme-spacing-sm', spacing.sm);
	root.style.setProperty('--theme-spacing-md', spacing.md);
	root.style.setProperty('--theme-spacing-lg', spacing.lg);
	root.style.setProperty('--theme-spacing-xl', spacing.xl);
	root.style.setProperty('--theme-spacing-2xl', spacing['2xl']);
}

/**
 * 应用阴影变量
 */
function applyShadowVariables(shadows: ThemeConfig['shadows']): void {
	const root = document.documentElement;

	root.style.setProperty('--theme-shadow-xs', shadows.xs);
	root.style.setProperty('--theme-shadow-sm', shadows.sm);
	root.style.setProperty('--theme-shadow-md', shadows.md);
	root.style.setProperty('--theme-shadow-lg', shadows.lg);
	root.style.setProperty('--theme-shadow-xl', shadows.xl);
	root.style.setProperty('--theme-shadow-2xl', shadows['2xl']);
	root.style.setProperty('--theme-shadow-inner', shadows.inner);
}

/**
 * 应用圆角变量
 */
function applyRadiusVariables(radius: ThemeConfig['radius']): void {
	const root = document.documentElement;

	root.style.setProperty('--theme-radius-sm', radius.sm);
	root.style.setProperty('--theme-radius-md', radius.md);
	root.style.setProperty('--theme-radius-lg', radius.lg);
	root.style.setProperty('--theme-radius-xl', radius.xl);
	root.style.setProperty('--theme-radius-2xl', radius['2xl']);
	root.style.setProperty('--theme-radius-full', radius.full);
}

/**
 * 应用特殊效果变量
 */
export function applyEffectVariables(effects: ThemeConfig['effects']): void {
	const root = document.documentElement;

	// 玻璃拟态效果
	if (effects.glass) {
		root.style.setProperty('--theme-glass-blur', effects.glass.blur);
		root.style.setProperty('--theme-glass-opacity', effects.glass.opacity);
		root.style.setProperty('--theme-glass-border', effects.glass.border);
	}

	// 新拟态效果
	if (effects.neumorphic) {
		root.style.setProperty('--theme-neu-light', effects.neumorphic.light);
		root.style.setProperty('--theme-neu-dark', effects.neumorphic.dark);
		root.style.setProperty('--theme-neu-radius', effects.neumorphic.radius);
	}

	// 过渡效果
	if (effects.transition) {
		root.style.setProperty('--theme-transition-fast', effects.transition.fast);
		root.style.setProperty('--theme-transition-base', effects.transition.base);
		root.style.setProperty('--theme-transition-slow', effects.transition.slow);
	}
}

/**
 * 应用自定义主题色（覆盖主题预设）
 * @param primaryColor 主色调（HEX格式）
 */
export function applyPrimaryColor(primaryColor: string | unknown): void {
	// 类型安全检查
	if (!primaryColor || typeof primaryColor !== 'string') {
		console.warn('applyPrimaryColor: Invalid color value', primaryColor);
		return;
	}

	const root = document.documentElement;

	// 生成颜色变体
	const variants = generateColorVariants(primaryColor);

	root.style.setProperty('--theme-primary', variants.primary);
	root.style.setProperty('--theme-primary-hover', variants.primaryHover);
	root.style.setProperty('--theme-primary-light', variants.primaryLight);
	root.style.setProperty('--theme-primary-lighter', variants.primaryLighter);

	// 同步更新侧边栏强调色
	root.style.setProperty('--sidebar-accent', variants.primary);
}

/**
 * 应用自定义侧边栏背景色
 * @param backgroundColor 背景色（HEX格式）
 */
export function applySidebarBackgroundColor(backgroundColor: string | unknown): void {
	// 类型安全检查
	if (!backgroundColor || typeof backgroundColor !== 'string') {
		console.warn('applySidebarBackgroundColor: Invalid color value', backgroundColor);
		return;
	}

	const root = document.documentElement;

	// 设置侧边栏背景色
	root.style.setProperty('--sidebar-bg', backgroundColor);

	// 自动计算并设置文字颜色
	const textColor = getContrastTextColor(backgroundColor);
	const hoverBackgroundColor = getHoverBackgroundColor(backgroundColor);

	root.style.setProperty('--sidebar-text', textColor);
	root.style.setProperty('--sidebar-text-hover', '#FFFFFF');
	root.style.setProperty('--sidebar-text-active', '#FFFFFF');
	root.style.setProperty('--sidebar-bg-hover', hoverBackgroundColor);
	root.style.setProperty('--sidebar-bg-active', getHoverBackgroundColor(hoverBackgroundColor));

	// 根据背景深浅调整边框色
	const borderColor = getLuminance(backgroundColor) < 0.5 ? hoverBackgroundColor : '#E2E8F0';
	root.style.setProperty('--sidebar-border', borderColor);
}

/**
 * 主题状态管理（重构版 - 衍生状态）
 * Theme State Management (Refactored - Derived State)
 *
 * 主题配置现在从 SettingsStore 衍生
 * 这个 Store 负责将配置应用到 DOM
 */

import { create } from 'zustand';
import type { ThemeType, ThemeConfig } from '@/types/theme';
import type { ThemeConfiguration } from '@/types/config';
import { themes, applyThemeVariables, applyEffectVariables } from '@/design-system/themes';

interface ThemeState {
	// 从 SettingsStore 衍生的配置
	configuration: ThemeConfiguration;

	// 当前应用的主题
	appliedTheme: ThemeType;
	appliedMode: 'light' | 'dark';
}

interface ThemeActions {
	// 从 SettingsStore 同步配置
	syncFromSettings: (config: ThemeConfiguration) => void;

	// 运行时临时预览（不保存到 Settings）
	previewTheme: (theme: ThemeType) => void;
	previewMode: (mode: 'light' | 'dark') => void;

	// 应用配置到 DOM
	applyToDOM: () => void;

	// 保存预览的配置到 SettingsStore
	savePreview: () => void;

	// 取消预览，恢复到 SettingsStore 的配置
	resetPreview: () => void;
}

type ThemeStore = ThemeState & ThemeActions;

export const useThemeStore = create<ThemeStore>()(
	(set, get) => ({
		// 初始状态
		configuration: {
			mode: 'light',
			style: 'minimal',
		},
		appliedTheme: 'minimal',
		appliedMode: 'light',

		// 从 SettingsStore 同步配置
		syncFromSettings: (config) => {
			set({ configuration: config });
			get().applyToDOM();
		},

		// 预览主题（临时，不保存）
		previewTheme: (theme) => {
			set({
				appliedTheme: theme,
				configuration: {
					...get().configuration,
					style: theme,
				},
			});
			get().applyToDOM();
		},

		// 预览模式（临时，不保存）
		previewMode: (mode) => {
			set({
				appliedMode: mode,
				configuration: {
					...get().configuration,
					mode,
				},
			});
			get().applyToDOM();
		},

		// 应用到 DOM
		applyToDOM: () => {
			const { configuration } = get();
			const themeConfig = themes[configuration.style];

			// 应用 CSS 变量
			applyThemeVariables(themeConfig);
			applyEffectVariables(themeConfig.effects);

			// 设置 data 属性
			document.documentElement.setAttribute('data-theme', configuration.style);
			document.documentElement.setAttribute('data-color-mode', configuration.mode);
		},

		// 保存预览的配置（需要触发 SettingsStore 更新）
		savePreview: () => {
			const { configuration } = get();

			// 触发 settings-changed 事件，让 ConfigProvider 处理保存
			if (typeof window !== 'undefined') {
				window.dispatchEvent(new CustomEvent('save-theme-settings', {
					detail: configuration,
				}));
			}
		},

		// 取消预览
		resetPreview: () => {
			// 触发重新从 SettingsStore 同步
			if (typeof window !== 'undefined') {
				window.dispatchEvent(new Event('reset-theme-preview'));
			}
		},
	})
);

// 监听 SettingsStore 变化
if (typeof window !== 'undefined') {
	window.addEventListener('settings-changed', ((event: CustomEvent) => {
		const preferences = event.detail;
		const syncTheme = useThemeStore.getState().syncFromSettings;
		syncTheme(preferences.theme);
	}) as EventListener);
}

/**
 * 主题 Hook
 * 提供更方便的主题访问和操作
 */
export function useTheme() {
	const { configuration, appliedTheme, appliedMode, syncFromSettings, previewTheme, previewMode, savePreview, resetPreview } = useThemeStore();

	return {
		// 当前配置
		theme: configuration.style,
		mode: configuration.mode,
		config: themes[configuration.style],

		// 应用状态
		appliedTheme,
		appliedMode,

		// 操作方法
		setTheme: syncFromSettings,
		previewTheme,
		previewMode,
		savePreview,
		resetPreview,

		// 便捷属性
		isMinimal: appliedTheme === 'minimal',
		isGlassmorphism: appliedTheme === 'glassmorphism',
		isNeumorphism: appliedTheme === 'neumorphism',
		isFlat2: appliedTheme === 'flat2.0',
		isLuxuryQuiet: appliedTheme === 'luxury-quiet',
		isLight: appliedMode === 'light',
		isDark: appliedMode === 'dark',
	};
}

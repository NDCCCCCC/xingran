/**
 * 布局状态管理（重构版 - 衍生状态）
 * Layout State Management (Refactored - Derived State)
 *
 * 布局配置现在从 SettingsStore 衍生
 * 这个 Store 负责将布局配置应用到 DOM 和组件
 */

import { create } from 'zustand';
import { useEffect } from 'react';
import type { LayoutType, LayoutConfig, DensityMode } from '@/types/layout';
import type { LayoutConfiguration } from '@/types/config';
import { defaultLayoutConfiguration } from '@/types/config';

interface LayoutState {
	// 从 SettingsStore 衍生的配置
	configuration: LayoutConfiguration;

	// 当前应用的状态
	currentLayout: LayoutType;
	sidebarCollapsed: boolean;
	density: DensityMode;
}

interface LayoutActions {
	// 从 SettingsStore 同步配置
	syncFromSettings: (config: LayoutConfiguration) => void;

	// 运行时临时操作（不保存到 Settings）
	toggleSidebar: () => void;
	setSidebarCollapsed: (collapsed: boolean) => void;
	setDensity: (density: DensityMode) => void;
	setLayout: (layout: LayoutType) => void;

	// 保存当前状态到 SettingsStore
	saveState: () => void;

	// 应用到 DOM
	applyToDOM: () => void;
}

type LayoutStore = LayoutState & LayoutActions;

// 布局配置
const layoutConfigs: Record<LayoutType, LayoutConfig> = {
	classic: {
		id: 'classic',
		name: '经典布局',
		description: '左侧导航+顶部栏+内容区',
		features: {
			sidebar: {
				collapsible: true,
				width: 256,
				collapsedWidth: 64,
				position: 'left',
				defaultCollapsed: false,
			},
			header: {
				fixed: true,
				height: 64,
				showBreadcrumb: true,
				showUserInfo: true,
			},
			tabs: {
				enabled: false,
				position: 'top',
				closable: false,
				draggable: false,
				persist: false,
			},
			content: {
				padding: '24px',
				centered: false,
				scrollable: true,
			},
		},
	},
	hybrid: {
		id: 'hybrid',
		name: '混合式布局',
		description: '可折叠侧边栏+多标签页系统',
		features: {
			sidebar: {
				collapsible: true,
				width: 240,
				collapsedWidth: 56,
				position: 'left',
				defaultCollapsed: false,
			},
			header: {
				fixed: true,
				height: 56,
				showBreadcrumb: false,
				showUserInfo: true,
			},
			tabs: {
				enabled: true,
				position: 'top',
				closable: true,
				draggable: true,
				persist: true,
			},
			content: {
				padding: '16px',
				centered: false,
				scrollable: true,
			},
		},
	},
	innovative: {
		id: 'innovative',
		name: '创新布局',
		description: '创新导航方式+模块化面板',
		features: {
			sidebar: {
				collapsible: false,
				width: 80,
				collapsedWidth: 80,
				position: 'left',
				defaultCollapsed: false,
			},
			header: {
				fixed: false,
				height: 0,
				showBreadcrumb: false,
				showUserInfo: false,
			},
			tabs: {
				enabled: true,
				position: 'bottom',
				closable: true,
				draggable: false,
				persist: true,
			},
			content: {
				padding: '0',
				centered: true,
				scrollable: true,
			},
		},
	},
};

export const useLayoutStore = create<LayoutStore>()(
	(set, get) => ({
		// 初始状态 - 从 localStorage 同步读取，避免首次渲染闪烁
		configuration: (() => {
			try {
				const stored = localStorage.getItem('settings-storage');
				if (stored) {
					const parsed = JSON.parse(stored);
					return parsed.state?.preferences?.layout || defaultLayoutConfiguration;
				}
			} catch (e) {
				console.error('Failed to read layout from localStorage:', e);
			}
			return defaultLayoutConfiguration;
		})(),

		currentLayout: (() => {
			try {
				const stored = localStorage.getItem('settings-storage');
				if (stored) {
					const parsed = JSON.parse(stored);
					return parsed.state?.preferences?.layout?.type || 'classic';
				}
			} catch (e) {
				console.error('Failed to read layout type from localStorage:', e);
			}
			return 'classic';
		})(),

		sidebarCollapsed: (() => {
			try {
				const stored = localStorage.getItem('settings-storage');
				if (stored) {
					const parsed = JSON.parse(stored);
					return parsed.state?.preferences?.layout?.sidebar?.collapsed ?? false;
				}
			} catch (e) {
				console.error('Failed to read sidebarCollapsed from localStorage:', e);
			}
			return false;
		})(),

		density: (() => {
			try {
				const stored = localStorage.getItem('settings-storage');
				if (stored) {
					const parsed = JSON.parse(stored);
					return parsed.state?.preferences?.layout?.density || 'comfortable';
				}
			} catch (e) {
				console.error('Failed to read density from localStorage:', e);
			}
			return 'comfortable';
		})(),

		// 从 SettingsStore 同步配置
		syncFromSettings: (config) => {
			set({
				configuration: config,
				currentLayout: config.type,
				sidebarCollapsed: config.sidebar.collapsed,
				density: config.density,
			});
			get().applyToDOM();
		},

		// 运行时切换侧边栏（临时，不保存）
		toggleSidebar: () => {
			set((state) => {
				const newCollapsed = !state.sidebarCollapsed;
				const newConfig = {
					...state.configuration,
					sidebar: {
						...state.configuration.sidebar,
						collapsed: newCollapsed,
					},
				};
				get().applyToDOM();
				return {
					sidebarCollapsed: newCollapsed,
					configuration: newConfig,
				};
			});
		},

		// 运行时设置侧边栏折叠状态（临时，不保存）
		setSidebarCollapsed: (collapsed) => {
			set((state) => {
				const newConfig = {
					...state.configuration,
					sidebar: {
						...state.configuration.sidebar,
						collapsed,
					},
				};
				get().applyToDOM();
				return {
					sidebarCollapsed: collapsed,
					configuration: newConfig,
				};
			});
		},

		// 运行时设置密度模式（临时，不保存）
		setDensity: (density) => {
			set((state) => {
				const newConfig = {
					...state.configuration,
					density,
				};
				get().applyToDOM();
				return {
					density,
					configuration: newConfig,
				};
			});
		},

		// 运行时设置布局类型（临时，不保存）
		setLayout: (layout) => {
			set((state) => {
				const newConfig = {
					...state.configuration,
					type: layout,
				};
				get().applyToDOM();
				return {
					currentLayout: layout,
					configuration: newConfig,
				};
			});
		},

		// 保存当前状态到 SettingsStore
		saveState: () => {
			const { configuration } = get();

			// 触发 save-layout-settings 事件
			if (typeof window !== 'undefined') {
				window.dispatchEvent(new CustomEvent('save-layout-settings', {
					detail: configuration,
				}));
			}
		},

		// 应用到 DOM
		applyToDOM: () => {
			const { currentLayout, density, sidebarCollapsed } = get();
			document.documentElement.setAttribute('data-layout', currentLayout);
			document.documentElement.setAttribute('data-density', density);
			document.documentElement.setAttribute('data-sidebar-collapsed', String(sidebarCollapsed));
		},
	})
);

// 监听 SettingsStore 变化
if (typeof window !== 'undefined') {
	window.addEventListener('settings-changed', ((event: CustomEvent) => {
		const preferences = event.detail;
		const syncLayout = useLayoutStore.getState().syncFromSettings;
		syncLayout(preferences.layout);
	}) as EventListener);
}

/**
 * 布局 Hook
 * 提供更方便的布局访问和操作
 */
export function useLayout() {
	const { currentLayout, configuration, sidebarCollapsed, density, syncFromSettings, toggleSidebar, setSidebarCollapsed, setDensity, setLayout, saveState } = useLayoutStore();

	// 同步 data-layout 和 data-density 属性
	useEffect(() => {
		const store = useLayoutStore.getState();
		document.documentElement.setAttribute('data-layout', store.currentLayout);
		document.documentElement.setAttribute('data-density', store.density);
	}, [currentLayout, density]);

	const layoutConfig = layoutConfigs[currentLayout];

	return {
		layout: currentLayout,
		layoutConfig,
		configuration,
		sidebarCollapsed,
		density,
		syncFromSettings,
		toggleSidebar,
		setSidebarCollapsed,
		setDensity,
		setLayout,
		saveState,
		isClassic: currentLayout === 'classic',
		isHybrid: currentLayout === 'hybrid',
		isInnovative: currentLayout === 'innovative',
		isCompact: density === 'compact',
		isComfortable: density === 'comfortable',
		isSpacious: density === 'spacious',
	};
}

/**
 * 导出布局配置
 */
export { layoutConfigs };

/**
 * 配置服务
 * Configuration Service
 *
 * 负责用户配置的获取、更新、缓存和数据迁移
 */

import type { UserPreferences, BackendUserPreferences, ThemeConfiguration, LayoutConfiguration, DataConfiguration } from '@/types/config';
import { defaultUserPreferences, defaultThemeConfiguration, defaultLayoutConfiguration, defaultDataConfiguration } from '@/types/config';
import { getUserPreferences, updateUserPreferences as updateUserPreferencesAPI } from '@/lib/profileApi';

/**
 * 配置服务类
 */
class ConfigService {
	private cache: UserPreferences | null = null;
	private cacheExpiry: number = 0;
	private readonly CACHE_TTL = 5 * 60 * 1000; // 5分钟缓存

	/**
	 * 获取用户偏好设置
	 */
	async getUserPreferences(): Promise<UserPreferences> {
		// 检查缓存
		if (this.cache && Date.now() < this.cacheExpiry) {
			return { ...this.cache };
		}

		try {
			// 从服务器获取
			const response = await getUserPreferences();

			// 转换为前端格式
			const normalized = this.fromBackendFormat(response.data);

			// 更新缓存
			this.cache = normalized;
			this.cacheExpiry = Date.now() + this.CACHE_TTL;

			return normalized;

		} catch (error) {
			console.error('Failed to fetch user preferences:', error);

			// 返回默认值
			return { ...defaultUserPreferences };
		}
	}

	/**
	 * 更新用户偏好设置
	 */
	async updateUserPreferences(preferences: UserPreferences): Promise<void> {
		// 转换为后端格式
		const backendFormat = this.toBackendFormat(preferences);

		try {
			// 保存到服务器
			await updateUserPreferencesAPI(backendFormat);

			// 更新缓存
			this.cache = preferences;
			this.cacheExpiry = Date.now() + this.CACHE_TTL;

		} catch (error) {
			console.error('Failed to update user preferences:', error);
			throw error;
		}
	}

	/**
	 * 数据迁移 - 处理不同版本的配置格式
	 */
	migratePreferences(prefs: any): UserPreferences {
		const version = prefs.version || 1;

		// 版本 1 → 版本 2（旧数据结构）
		if (version === 1) {
			return {
				version: 2,
				theme: {
					mode: prefs.theme === 'dark' ? 'dark' : 'light',
					style: prefs.themeStyle || 'minimal',
				},
				layout: {
					type: prefs.layoutType || 'classic',
					sidebar: {
						collapsed: prefs.sidebarCollapsed || false,
						width: 240,
						collapsedWidth: 64,
					},
					density: prefs.density || 'comfortable',
				},
				data: {
					defaultPageSize: prefs.pageSize || 10,
					pageSizeOptions: [10, 20, 50, 100],
				},
				language: prefs.language || 'zh-CN',
			};
		}

		// 版本 2（当前版本）- 直接返回
		return prefs as UserPreferences;
	}

	/**
	 * 从后端格式转换为前端格式
	 */
	private fromBackendFormat(backend: BackendUserPreferences): UserPreferences {
		return {
			version: 2,
			theme: {
				mode: backend.theme === 'dark' ? 'dark' : 'light',
				style: (backend.themeStyle as any) || 'minimal',
			},
			layout: {
				type: (backend.layoutType as any) || 'classic',
				sidebar: {
					collapsed: backend.sidebarCollapsed || false,
					width: backend.sidebarWidth || 240,
					collapsedWidth: backend.sidebarCollapsedWidth || 64,
				},
				density: (backend.layoutDensity as any) || 'comfortable',
			},
			data: {
				defaultPageSize: backend.pageSize || 10,
				pageSizeOptions: [10, 20, 50, 100],
			},
			language: backend.language || 'zh-CN',
		};
	}

	/**
	 * 转换为后端格式
	 */
	private toBackendFormat(prefs: UserPreferences): BackendUserPreferences {
		return {
			// 主题
			theme: prefs.theme.mode,
			themeStyle: prefs.theme.style,

			// 布局
			layoutType: prefs.layout.type,
			layoutDensity: prefs.layout.density,
			sidebarWidth: prefs.layout.sidebar.width,
			sidebarCollapsedWidth: prefs.layout.sidebar.collapsedWidth,
			sidebarCollapsed: prefs.layout.sidebar.collapsed,

			// 数据
			pageSize: prefs.data.defaultPageSize,

			// 语言
			language: prefs.language || 'zh-CN',
		};
	}

	/**
	 * 规范化数据 - 确保所有字段都存在
	 */
	private normalizePreferences(raw: Partial<UserPreferences>): UserPreferences {
		return {
			version: raw.version || 2,
			theme: {
				...defaultThemeConfiguration,
				...raw.theme,
			},
			layout: {
				...defaultLayoutConfiguration,
				...raw.layout,
				sidebar: {
					...defaultLayoutConfiguration.sidebar,
					...raw.layout?.sidebar,
				},
			},
			data: {
				...defaultDataConfiguration,
				...raw.data,
			},
			language: raw.language || 'zh-CN',
		};
	}

	/**
	 * 获取默认配置
	 */
	getDefaultPreferences(): UserPreferences {
		return { ...defaultUserPreferences };
	}

	/**
	 * 清除缓存
	 */
	clearCache(): void {
		this.cache = null;
		this.cacheExpiry = 0;
	}
}

// 导出单例
export const configService = new ConfigService();

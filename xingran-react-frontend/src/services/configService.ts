/**
 * 配置服务
 * Configuration Service
 *
 * 负责用户配置的获取、更新、缓存和数据迁移
 */

import type { UserPreferences, BackendUserPreferences } from "@/types/config";
import {
  defaultUserPreferences,
  defaultThemeConfiguration,
  defaultLayoutConfiguration,
  defaultDataConfiguration,
} from "@/types/config";
import { getUserPreferences } from "@/lib/profileApi";
import { put } from "@/lib/api";

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
      const backendData = await getUserPreferences();

      // 转换为前端格式
      const normalized = this.fromBackendFormat(backendData);

      // 更新缓存
      this.cache = normalized;
      this.cacheExpiry = Date.now() + this.CACHE_TTL;

      return normalized;
    } catch (error) {
      console.error("Failed to fetch user preferences:", error);

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
      // 保存到服务器 - 直接使用 put 调用后端 API
      await put("/system/settings/preferences", backendFormat);

      // 更新缓存
      this.cache = preferences;
      this.cacheExpiry = Date.now() + this.CACHE_TTL;
    } catch (error) {
      console.error("Failed to update user preferences:", error);
      throw error;
    }
  }

  /**
   * 数据迁移 - 处理不同版本的配置格式
   */
  migratePreferences(prefs: Partial<UserPreferences> | Record<string, unknown>): UserPreferences {
    // 使用类型断言访问旧版本属性
    const legacyPrefs = prefs as Record<string, unknown>;
    const version = (legacyPrefs.version as number) || 1;

    // 版本 1 → 版本 2（旧数据结构）
    if (version === 1) {
      // 验证布局类型
      const validLayoutTypes = ["classic", "hybrid", "innovative"];
      const layoutType =
        legacyPrefs.layoutType && validLayoutTypes.includes(legacyPrefs.layoutType as string)
          ? (legacyPrefs.layoutType as "classic" | "hybrid" | "innovative")
          : defaultLayoutConfiguration.type;

      // 验证密度模式
      const validDensityModes = ["compact", "comfortable", "spacious"];
      const density =
        legacyPrefs.density && validDensityModes.includes(legacyPrefs.density as string)
          ? (legacyPrefs.density as "compact" | "comfortable" | "spacious")
          : defaultLayoutConfiguration.density;

      const partial: Partial<UserPreferences> = {
        version: 2,
        // v1 旧数据的 themeStyle 字段随多主题移除而静默丢弃（Phase 65 · D-01）
        theme: {
          mode: legacyPrefs.theme === "dark" ? "dark" : "light",
        },
        layout: {
          type: layoutType,
          sidebar: {
            collapsed: Boolean(legacyPrefs.sidebarCollapsed),
            width:
              typeof legacyPrefs.sidebarWidth === "number"
                ? (legacyPrefs.sidebarWidth as number)
                : defaultLayoutConfiguration.sidebar.width,
            collapsedWidth:
              typeof legacyPrefs.sidebarCollapsedWidth === "number"
                ? (legacyPrefs.sidebarCollapsedWidth as number)
                : defaultLayoutConfiguration.sidebar.collapsedWidth,
          },
          density: density,
        },
        data: {
          defaultPageSize:
            typeof legacyPrefs.pageSize === "number"
              ? (legacyPrefs.pageSize as number)
              : defaultDataConfiguration.defaultPageSize,
          pageSizeOptions: [10, 20, 50, 100],
        },
        language: legacyPrefs.language as "zh-CN" | "en-US" | undefined,
      };
      return this.normalizePreferences(partial);
    }

    // 版本 2（当前版本）
    return this.normalizePreferences(prefs as Partial<UserPreferences>);
  }

  /**
   * 从后端格式转换为前端格式
   */
  private fromBackendFormat(backend: BackendUserPreferences): UserPreferences {
    // 验证布局类型是否有效
    const validLayoutTypes: Array<"classic" | "hybrid" | "innovative"> = [
      "classic",
      "hybrid",
      "innovative",
    ];
    const layoutType =
      backend.layoutType &&
      validLayoutTypes.includes(backend.layoutType as "classic" | "hybrid" | "innovative")
        ? (backend.layoutType as "classic" | "hybrid" | "innovative")
        : defaultLayoutConfiguration.type;

    // 验证密度模式是否有效
    const validDensityModes: Array<"compact" | "comfortable" | "spacious"> = [
      "compact",
      "comfortable",
      "spacious",
    ];
    const density =
      backend.layoutDensity &&
      validDensityModes.includes(backend.layoutDensity as "compact" | "comfortable" | "spacious")
        ? (backend.layoutDensity as "compact" | "comfortable" | "spacious")
        : defaultLayoutConfiguration.density;

    const partial: Partial<UserPreferences> = {
      version: 2,
      // 后端 themeStyle / customPrimaryColor / customSidebarColor 为 @Deprecated
      // 契约字段（Phase 65 · D-05 后端零改动），前端仅映射 mode，其余静默丢弃
      theme: {
        mode: backend.theme === "dark" ? "dark" : "light",
      },
      layout: {
        type: layoutType,
        sidebar: {
          collapsed: backend.sidebarCollapsed,
          width: backend.sidebarWidth ?? defaultLayoutConfiguration.sidebar.width,
          collapsedWidth:
            backend.sidebarCollapsedWidth ?? defaultLayoutConfiguration.sidebar.collapsedWidth,
        },
        density: density,
      },
      data: {
        defaultPageSize: backend.pageSize,
        pageSizeOptions: [10, 20, 50, 100], // 使用默认值
      },
      language: backend.language as "zh-CN" | "en-US",
    };

    return this.normalizePreferences(partial);
  }

  /**
   * 转换为后端格式
   */
  private toBackendFormat(prefs: UserPreferences): BackendUserPreferences {
    const result: BackendUserPreferences = {
      // 主题 - 确保有默认值
      // （themeStyle / customPrimaryColor / customSidebarColor 已随 Phase 65
      //  多主题移除不再发送；后端字段为 optional 可直接缺省）
      theme: prefs.theme?.mode || "light",

      // 布局
      layoutType: prefs.layout?.type || "classic",
      layoutDensity: prefs.layout?.density || "comfortable",
      sidebarWidth: prefs.layout?.sidebar?.width || 280,
      sidebarCollapsedWidth: prefs.layout?.sidebar?.collapsedWidth || 64,
      sidebarCollapsed: prefs.layout?.sidebar?.collapsed || false,

      // 数据
      pageSize: prefs.data?.defaultPageSize || 10,

      // 语言 - 确保有默认值
      language: (prefs.language as "zh-CN" | "en-US") || "zh-CN",
    };

    return result;
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
      language: (raw.language as "zh-CN" | "en-US") || "zh-CN",
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

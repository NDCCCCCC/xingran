import { useState, useCallback, useMemo, useEffect } from "react";
import { App } from "antd";
import { columnConfigApi } from "@/lib/columnConfigApi";
import type { ColumnConfigItem, UserColumnConfig } from "@/lib/columnConfigApi";

export interface ColumnConfig {
  key: string;
  label: string;
  visible: boolean;
  order: number;
  width?: number;
  group?: string;
}

export interface UseColumnConfigOptions {
  pageKey: string;
  defaultColumns: ColumnConfig[];
  enableCache?: boolean;
}

const CACHE_PREFIX = "column_config";
const CACHE_EXPIRY = 5 * 60 * 1000; // 5 minutes

// localStorage 缓存工具函数
const getFromLocalStorage = (pageKey: string): ColumnConfig[] | null => {
  try {
    const key = `${CACHE_PREFIX}:${pageKey}`;
    const cached = localStorage.getItem(key);
    if (!cached) return null;

    const { data, timestamp } = JSON.parse(cached);
    if (Date.now() - timestamp > CACHE_EXPIRY) {
      localStorage.removeItem(key);
      return null;
    }

    return data;
  } catch (error) {
    console.error("Failed to read from localStorage:", error);
    return null;
  }
};

const saveToLocalStorage = (pageKey: string, config: ColumnConfig[]) => {
  try {
    const key = `${CACHE_PREFIX}:${pageKey}`;
    localStorage.setItem(key, JSON.stringify({
      data: config,
      timestamp: Date.now(),
    }));
  } catch (error) {
    console.error("Failed to save to localStorage:", error);
  }
};

const removeFromLocalStorage = (pageKey: string) => {
  try {
    const key = `${CACHE_PREFIX}:${pageKey}`;
    localStorage.removeItem(key);
  } catch (error) {
    console.error("Failed to remove from localStorage:", error);
  }
};

// 转换后端数据格式为前端格式
const transformToColumnConfig = (userConfigs: UserColumnConfig[], defaultConfigs: ColumnConfig[]): ColumnConfig[] => {
  const configMap = new Map<string, ColumnConfig>();

  // 首先添加默认配置（作为标签和分组的来源）
  defaultConfigs.forEach(col => {
    configMap.set(col.key, { ...col });
  });

  // 然后覆盖用户配置
  userConfigs.forEach((userCol, index) => {
    const existing = configMap.get(userCol.columnKey);
    if (existing) {
      configMap.set(userCol.columnKey, {
        ...existing,
        visible: userCol.visible,
        order: index + 1,
        width: userCol.width || existing.width,
      });
    }
  });

  // 按顺序排序
  return Array.from(configMap.values())
    .map((col, idx) => ({ ...col, order: idx + 1 }))
    .sort((a, b) => a.order - b.order);
};

export function useColumnConfig(options: UseColumnConfigOptions) {
  const { pageKey, defaultColumns, enableCache = true } = options;
  const { message } = App.useApp();
  const [config, setConfig] = useState<ColumnConfig[]>(defaultColumns);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  // 加载配置
  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      // 健全性检查：可见列数过少视为损坏配置，回退默认（防御被误清空的缓存/服务端数据）
      const minVisible = Math.max(
        1,
        Math.floor(defaultColumns.filter(c => c.visible).length / 2)
      );
      const isConfigSane = (cfg: ColumnConfig[] | null | undefined): boolean => {
        if (!cfg || cfg.length === 0) return false;
        const visibleCount = cfg.filter(c => c.visible).length;
        return visibleCount >= minVisible;
      };

      // 尝试从缓存加载
      if (enableCache) {
        const cached = getFromLocalStorage(pageKey);
        if (cached && isConfigSane(cached)) {
          setConfig(cached);
        } else if (cached) {
          console.warn(
            `[useColumnConfig] Cached config for "${pageKey}" has <${minVisible} visible columns, falling back to default`
          );
          removeFromLocalStorage(pageKey);
        }
      }

      // 从服务器加载
      const response = await columnConfigApi.getByPageKey(pageKey);
      const userConfig = response.data;

      if (userConfig && userConfig.length > 0) {
        const transformedConfig = transformToColumnConfig(userConfig, defaultColumns);
        if (isConfigSane(transformedConfig)) {
          setConfig(transformedConfig);
          if (enableCache) {
            saveToLocalStorage(pageKey, transformedConfig);
          }
        } else {
          console.warn(
            `[useColumnConfig] Server config for "${pageKey}" has <${minVisible} visible columns, falling back to default`
          );
          setConfig(defaultColumns);
          if (enableCache) {
            saveToLocalStorage(pageKey, defaultColumns);
          }
        }
      } else {
        // 使用默认配置
        setConfig(defaultColumns);
        if (enableCache) {
          saveToLocalStorage(pageKey, defaultColumns);
        }
      }
    } catch (error) {
      console.error("Failed to load column config:", error);
      // 失败时使用默认配置
      setConfig(defaultColumns);
    } finally {
      setLoading(false);
    }
  }, [pageKey, defaultColumns, enableCache]);

  // 保存配置（防抖）
  const saveConfig = useCallback(async (newConfig: ColumnConfig[]) => {
    setSaving(true);
    try {
      const columnConfigs: ColumnConfigItem[] = newConfig.map((col, index) => ({
        columnKey: col.key,
        visible: col.visible,
        width: col.width || 0,
      }));

      await columnConfigApi.save({
        pageKey,
        columnConfigs,
      });

      setConfig(newConfig);

      if (enableCache) {
        saveToLocalStorage(pageKey, newConfig);
      }

      message.success("列配置保存成功");
    } catch (error) {
      console.error("Failed to save column config:", error);
      message.error("保存列配置失败，请稍后重试");
      throw error;
    } finally {
      setSaving(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [pageKey, enableCache]);

  // 重置配置
  const resetConfig = useCallback(async () => {
    setSaving(true);
    try {
      await columnConfigApi.reset(pageKey);

      setConfig(defaultColumns);

      if (enableCache) {
        removeFromLocalStorage(pageKey);
      }

      message.success("列配置已重置");
    } catch (error) {
      console.error("Failed to reset column config:", error);
      message.error("重置列配置失败，请稍后重试");
      throw error;
    } finally {
      setSaving(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [pageKey, defaultColumns, enableCache]);

  // 切换列可见性
  const toggleColumn = useCallback((key: string, visible: boolean) => {
    setConfig(prev => prev.map(col =>
      col.key === key ? { ...col, visible } : col
    ));
  }, []);

  // 更新列宽度
  const updateColumnWidth = useCallback((key: string, width: number) => {
    setConfig(prev => prev.map(col =>
      col.key === key ? { ...col, width } : col
    ));
  }, []);

  // 更新列顺序
  const updateColumnOrder = useCallback((newOrder: ColumnConfig[]) => {
    const reordered = newOrder.map((col, index) => ({ ...col, order: index + 1 }));
    setConfig(reordered);
  }, []);

  // 获取可见列
  const visibleColumns = useMemo(() => {
    return config
      .filter(col => col.visible)
      .sort((a, b) => a.order - b.order);
  }, [config]);

  // 组件挂载时加载配置
  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  return {
    config,
    visibleColumns,
    loading,
    saving,
    loadConfig,
    saveConfig,
    resetConfig,
    toggleColumn,
    updateColumnWidth,
    updateColumnOrder,
  };
}

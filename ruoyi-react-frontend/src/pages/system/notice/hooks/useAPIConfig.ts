import { useState, useCallback } from "react";
import { getAPINotificationConfigList } from "@/lib/notificationConfigApi";
import type { APINotificationConfig } from "@/lib/notificationConfigApi";

interface UseAPIConfigResult {
  apiConfigs: APINotificationConfig[];
  loadingAPIConfigs: boolean;
  loadAPIConfigs: () => Promise<void>;
}

/**
 * API配置数据 Hook
 * 处理API通知配置的加载
 */
export function useAPIConfig(): UseAPIConfigResult {
  const [apiConfigs, setApiConfigs] = useState<APINotificationConfig[]>([]);
  const [loadingAPIConfigs, setLoadingAPIConfigs] = useState(false);

  // 加载API配置列表
  const loadAPIConfigs = useCallback(async () => {
    setLoadingAPIConfigs(true);
    try {
      const result = await getAPINotificationConfigList({
        page: 1,
        pageSize: 100,
      }) as { data: { list: APINotificationConfig[] } };
      // 只显示状态为正常(0)的配置
      setApiConfigs(result.data.list.filter((c: APINotificationConfig) => c.status === 0));
    } catch (error) {
      console.error("加载API配置失败:", error);
    } finally {
      setLoadingAPIConfigs(false);
    }
  }, []);

  return {
    apiConfigs,
    loadingAPIConfigs,
    loadAPIConfigs,
  };
}

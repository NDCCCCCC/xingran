/**
 * AD 域配置管理 Hook
 * 用于统一管理 AD 域相关页面的配置选择逻辑
 */

import { useState, useEffect } from "react";
import { message } from "antd";
import {
  getADConfigList,
  type ADConfig,
} from "@/lib/adDomainApi";

export interface UseADConfigsOptions {
  /** 是否只获取启用状态的配置，默认 true */
  enabledOnly?: boolean;
  /** 初始时默认选择第一个配置，默认 true */
  autoSelectFirst?: boolean;
}

export function useADConfigs(options: UseADConfigsOptions = {}) {
  const {
    enabledOnly = true,
    autoSelectFirst = true,
  } = options;

  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [selectedConfig, setSelectedConfig] = useState<string>("");
  const [loading, setLoading] = useState(false);

  // 获取 AD 配置列表
  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 100 };
      if (enabledOnly) {
        params.status = 0; // 0=启用
      }

      const res = await getADConfigList(params);
      if (res.code === 0 && res.data && res.data.list && res.data.list.length > 0) {
        setConfigs(res.data.list);

        // 自动选择第一个配置（如果还未选择）
        if (autoSelectFirst && !selectedConfig) {
          setSelectedConfig(res.data.list[0].id);
        }
      }
    } catch {
      message.error("获取AD配置失败");
    } finally {
      setLoading(false);
    }
  };

  // 组件挂载时获取配置
  useEffect(() => {
    fetchConfigs();
  }, []);

  // 手动刷新配置列表
  const refreshConfigs = () => {
    fetchConfigs();
  };

  // 清除选择
  const clearSelection = () => {
    setSelectedConfig("");
  };

  return {
    configs,
    selectedConfig,
    setSelectedConfig,
    loading,
    refreshConfigs,
    clearSelection,
  };
}

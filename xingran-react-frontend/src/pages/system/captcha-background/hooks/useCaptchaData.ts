/**
 * Captcha Background Data Hook
 * 验证码背景数据管理 Hook
 */

import { useState, useCallback } from "react";
import type { CaptchaBackground } from "@/types/captcha";
import type { FormInstance } from "antd/es/form";
import * as captchaService from "@/services/captcha";
import type { CaptchaStatistics } from "../types";

export interface UseCaptchaDataReturn {
  backgrounds: CaptchaBackground[];
  loading: boolean;
  total: number;
  statistics: CaptchaStatistics | null;

  loadBackgrounds: (
    params?: Record<string, unknown>,
    searchValues?: Record<string, unknown>
  ) => Promise<void>;
  loadStatistics: () => Promise<void>;
}

export function useCaptchaData(
  searchForm: FormInstance<unknown>,
  setTotal: (total: number) => void
): UseCaptchaDataReturn {
  const [backgrounds, setBackgrounds] = useState<CaptchaBackground[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotalLocal] = useState(0);
  const [statistics, setStatistics] = useState<CaptchaStatistics | null>(null);

  const loadBackgrounds = useCallback(
    async (params: Record<string, unknown> = {}, searchValues?: Record<string, unknown>) => {
      setLoading(true);
      try {
        const values = searchValues || (searchForm.getFieldsValue() as Record<string, unknown>);
        const result = await captchaService.getCaptchaBackgroundList({
          current: (params.current as number) || 1,
          pageSize: (params.pageSize as number) || 10,
          ...(values as object),
        });
        setBackgrounds(result.items);
        setTotalLocal(result.total);
        setTotal(result.total);
      } catch (error) {
        console.error("加载背景图列表失败:", error);
      } finally {
        setLoading(false);
      }
    },
    [searchForm, setTotal]
  );

  const loadStatistics = useCallback(async () => {
    try {
      const stats = await captchaService.getCaptchaBackgroundStatistics();
      setStatistics(stats);
    } catch (error) {
      console.error("加载统计信息失败:", error);
    }
  }, []);

  return {
    backgrounds,
    loading,
    total: total,
    statistics,
    loadBackgrounds,
    loadStatistics,
  };
}

/**
 * Captcha Background Data Hook
 * 验证码背景数据管理 Hook
 */

import { useState, useCallback } from 'react';
import type { CaptchaBackground } from '@/types/captcha';
import * as captchaService from '@/services/captcha';
import type { CaptchaStatistics } from '../types';

export interface UseCaptchaDataReturn {
  backgrounds: CaptchaBackground[];
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  statistics: CaptchaStatistics | null;

  setTotal: React.Dispatch<React.SetStateAction<number>>;
  setCurrent: React.Dispatch<React.SetStateAction<number>>;
  setPageSize: React.Dispatch<React.SetStateAction<number>>;
  setStatistics: React.Dispatch<React.SetStateAction<CaptchaStatistics | null>>;

  loadBackgrounds: (params?: any, searchValues?: any) => Promise<void>;
  loadStatistics: () => Promise<void>;
}

export function useCaptchaData(searchForm: any): UseCaptchaDataReturn {
  const [backgrounds, setBackgrounds] = useState<CaptchaBackground[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [statistics, setStatistics] = useState<CaptchaStatistics | null>(null);

  const loadBackgrounds = useCallback(
    async (params: any = {}, searchValues?: any) => {
      setLoading(true);
      try {
        const values = searchValues || searchForm.getFieldsValue();
        const result = await captchaService.getCaptchaBackgroundList({
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
          ...values,
        });
        setBackgrounds(result.items);
        setTotal(result.total);
      } catch (error) {
        console.error('加载背景图列表失败:', error);
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize, searchForm]
  );

  const loadStatistics = useCallback(async () => {
    try {
      const stats = await captchaService.getCaptchaBackgroundStatistics();
      setStatistics(stats);
    } catch (error) {
      console.error('加载统计信息失败:', error);
    }
  }, []);

  return {
    backgrounds,
    loading,
    total,
    current,
    pageSize,
    statistics,
    setTotal,
    setCurrent,
    setPageSize,
    setStatistics,
    loadBackgrounds,
    loadStatistics,
  };
}

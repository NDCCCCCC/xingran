/**
 * useRPAProgress - RPA 执行进度实时推送 Hook
 *
 * 通过 WebSocket 订阅 RPA 任务执行进度更新
 */

import { useEffect, useCallback, useRef, useMemo } from "react";
import { useNoticeStore } from "@/store/noticeStore";
import type { RPAProgressMessage } from "@/types/rpa";

/**
 * RPA 进度更新数据
 */
export interface RPAProgressData {
  executionId: string;
  taskId: string;
  taskName: string;
  step: number;
  total: number;
  message: string;
  status: string;
  timestamp: number;
}

/**
 * Hook 选项
 */
export interface UseRPAProgressOptions {
  /** 是否启用订阅（默认 true） */
  enabled?: boolean;
  /** 过滤：只订阅指定执行ID的进度 */
  executionId?: string;
  /** 过滤：只订阅指定任务的进度 */
  taskId?: string;
  /** 进度回调 */
  onProgress?: (data: RPAProgressData) => void;
  /** 完成回调 */
  onCompleted?: (data: RPAProgressData) => void;
  /** 失败回调 */
  onFailed?: (data: RPAProgressData) => void;
}

/**
 * RPA 进度 Hook
 *
 * @example
 * ```tsx
 * // 基本用法 - 订阅所有 RPA 进度
 * const { progressData, isConnected } = useRPAProgress();
 *
 * // 订阅特定执行记录的进度
 * const { progressData } = useRPAProgress({
 *   executionId: 'uuid',
 *   onProgress: (data) => console.log('进度:', data),
 *   onCompleted: (data) => message.success('任务完成'),
 *   onFailed: (data) => message.error('任务失败'),
 * });
 * ```
 */
export function useRPAProgress(options?: UseRPAProgressOptions) {
  const { wsConnected, onRPAProgress } = useNoticeStore();
  const progressDataRef = useRef<Map<string, RPAProgressData>>(new Map());
  const unsubscribeRef = useRef<(() => void) | null>(null);

  // 使用 useMemo 缓存回调函数引用，避免 useEffect 无限循环
  // 遵循 Vercel React Best Practices: rerender-dependencies
  const callbacks = useMemo(
    () => ({
      onProgress: options?.onProgress,
      onCompleted: options?.onCompleted,
      onFailed: options?.onFailed,
    }),
    [options?.onProgress, options?.onCompleted, options?.onFailed]
  );

  // 处理 RPA 进度消息
  const handleRPAProgressMessage = useCallback(
    // eslint-disable-next-line react-hooks/preserve-manual-memoization
    (message: RPAProgressMessage) => {
      const data: RPAProgressData = {
        executionId: message.executionId,
        taskId: message.taskId,
        taskName: message.taskName,
        step: message.step,
        total: message.total,
        message: message.message,
        status: message.status,
        timestamp: message.timestamp,
      };

      // 应用过滤条件
      if (options?.executionId && data.executionId !== options.executionId) {
        return;
      }
      if (options?.taskId && data.taskId !== options.taskId) {
        return;
      }

      // 更新进度数据
      progressDataRef.current.set(data.executionId, data);

      // 触发回调
      switch (message.type) {
        case "rpa_progress":
          callbacks.onProgress?.(data);
          break;
        case "rpa_completed":
          callbacks.onCompleted?.(data);
          break;
        case "rpa_failed":
          callbacks.onFailed?.(data);
          break;
      }
    },
    [options?.executionId, options?.taskId, callbacks]
  );

  // 订阅 RPA 进度事件
  useEffect(() => {
    if (options?.enabled === false) {
      unsubscribeRef.current?.();
      unsubscribeRef.current = null;
      return;
    }

    // 订阅 RPA 进度事件
    unsubscribeRef.current = onRPAProgress(handleRPAProgressMessage);

    return () => {
      unsubscribeRef.current?.();
      unsubscribeRef.current = null;
    };
  }, [
    handleRPAProgressMessage,
    options?.enabled,
    options?.executionId,
    options?.taskId,
    onRPAProgress,
  ]);

  // 获取指定执行记录的进度
  const getProgress = useCallback((executionId: string): RPAProgressData | undefined => {
    return progressDataRef.current.get(executionId);
  }, []);

  // 获取所有进度数据
  const getAllProgress = useCallback((): RPAProgressData[] => {
    return Array.from(progressDataRef.current.values());
  }, []);

  // 清除指定执行记录的进度
  const clearProgress = useCallback((executionId: string) => {
    progressDataRef.current.delete(executionId);
  }, []);

  // 清除所有进度数据
  const clearAllProgress = useCallback(() => {
    progressDataRef.current.clear();
  }, []);

  return {
    /** WebSocket 连接状态 */
    isConnected: wsConnected,
    /** 获取指定执行记录的进度 */
    getProgress,
    /** 获取所有进度数据 */
    getAllProgress,
    /** 清除指定执行记录的进度 */
    clearProgress,
    /** 清除所有进度数据 */
    clearAllProgress,
  };
}

export default useRPAProgress;

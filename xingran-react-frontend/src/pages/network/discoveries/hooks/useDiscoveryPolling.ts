/**
 * Device Discovery Polling Hook
 * 设备发现轮询管理 Hook
 */

import { useEffect, useRef } from "react";
import type { DeviceDiscovery } from "@/types";

export interface UseDiscoveryPollingParams {
  discoveries: DeviceDiscovery[];
  onPoll: () => void;
}

export function useDiscoveryPolling({ discoveries, onPoll }: UseDiscoveryPollingParams) {
  const pollingTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // 用 ref 保存 discoveries，避免 effect 依赖数组因引用变化而频繁重建定时器
  const discoveriesRef = useRef(discoveries);
  // React Compiler 禁止在渲染期间写 ref，用 useEffect 保证初始化时机
  useEffect(() => {
    discoveriesRef.current = discoveries;
  }, [discoveries]);

  useEffect(() => {
    const runningTasks = discoveriesRef.current.filter((d) => d.status === "running");

    if (runningTasks.length > 0 && !pollingTimerRef.current) {
      const timer = setInterval(() => {
        const currentRunning = discoveriesRef.current.filter((d) => d.status === "running");
        if (currentRunning.length === 0) {
          if (pollingTimerRef.current) {
            clearInterval(pollingTimerRef.current);
            pollingTimerRef.current = null;
          }
          return;
        }
        onPoll();
      }, 3000);
      pollingTimerRef.current = timer;
    } else if (runningTasks.length === 0 && pollingTimerRef.current) {
      clearInterval(pollingTimerRef.current);
      pollingTimerRef.current = null;
    }

    return () => {
      if (pollingTimerRef.current) {
        clearInterval(pollingTimerRef.current);
        pollingTimerRef.current = null;
      }
    };
  }, [onPoll]);
}

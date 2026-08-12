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

  useEffect(() => {
    const runningTasks = discoveries.filter(d => d.status === "running");

    if (runningTasks.length > 0 && !pollingTimerRef.current) {
      const timer = setInterval(() => {
        const currentRunning = discoveries.filter(d => d.status === "running");
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
  }, [discoveries, onPoll]);
}

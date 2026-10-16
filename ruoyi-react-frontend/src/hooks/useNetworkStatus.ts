/**
 * useNetworkStatus - 网络状态检测 Hook
 *
 * 检测网络连接状态，提供在线/离线状态和离线历史
 */

import { useState, useEffect, useCallback } from "react";

export interface UseNetworkStatusReturn {
  /** 当前是否在线 */
  isOnline: boolean;
  /** 是否曾经离线过（用于显示恢复提示） */
  wasOffline: boolean;
  /** 重置 wasOffline 状态 */
  resetWasOffline: () => void;
}

/**
 * 网络状态检测 Hook
 *
 * @example
 * ```tsx
 * const { isOnline, wasOffline, resetWasOffline } = useNetworkStatus();
 *
 * if (!isOnline) {
 *   return <OfflineBanner />;
 * }
 *
 * if (wasOffline) {
 *   return <ConnectionRestoredBanner onClose={resetWasOffline} />;
 * }
 * ```
 */
export function useNetworkStatus(): UseNetworkStatusReturn {
  const [isOnline, setIsOnline] = useState<boolean>(() => {
    // SSR 兼容：服务端渲染时默认为 true
    if (typeof navigator === "undefined") {
      return true;
    }
    return navigator.onLine;
  });

  const [wasOffline, setWasOffline] = useState<boolean>(false);

  // 重置 wasOffline 状态
  const resetWasOffline = useCallback(() => {
    setWasOffline(false);
  }, []);

  useEffect(() => {
    // 处理网络恢复
    const handleOnline = () => {
      setIsOnline(true);
    };

    // 处理网络断开
    const handleOffline = () => {
      setIsOnline(false);
      setWasOffline(true);
    };

    // 添加事件监听器
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    // 清理函数
    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, []);

  return {
    isOnline,
    wasOffline,
    resetWasOffline,
  };
}

export default useNetworkStatus;

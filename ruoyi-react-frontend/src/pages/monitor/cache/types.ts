/**
 * Cache 类型定义
 */

export interface CacheInfo {
  key: string;
  value: string;
  ttl: number;
  size: number;
  type: string;
  location: string;  // "l1", "l2", "both"
  createdAt: string;
  updatedAt: string;
}

export interface CacheMonitor {
  l1: {
    status: {
      connected: boolean;
      type: string;
    };
    stats: {
      keyCount: number;
      usedMemory: number;
      hitRate: number;
      hitCount: number;
      missCount: number;
    };
  };
  l2: {
    status: {
      connected: boolean;
      type: string;
      version?: string;
      uptime?: string;
    };
    stats: {
      keyCount: number;
      usedMemory: number;
      hitRate: number;
      hitCount: number;
      missCount: number;
    };
  };
}

export interface CacheSearchForm {
  key: string;
  type: string;
  level: string;  // "all", "l1", "l2"
}

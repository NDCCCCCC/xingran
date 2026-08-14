/**
 * 双层缓存管理器
 * 实现 Memory + localStorage 双层缓存
 * 提供缓存统计和管理功能
 */

// ==================== 常量定义 ====================

/** 默认内存缓存过期时间（30分钟） */
const DEFAULT_MEMORY_TTL = 30 * 60 * 1000;

/** 默认 localStorage 缓存过期时间（7天） */
const DEFAULT_STORAGE_TTL = 7 * 24 * 60 * 60 * 1000;

/** localStorage 存储键前缀 */
const DEFAULT_STORAGE_PREFIX = "dual_level_cache_";

/** 定期清理间隔（5分钟） */
const CLEANUP_INTERVAL = 5 * 60 * 1000;

/** 日志前缀 */
const LOG_PREFIX = "[DualLevelCache]";

// ==================== 类型定义 ====================

interface CacheItem<T> {
  data: T;
  timestamp: number;
  expiresAt: number;
}

interface CacheConfig {
  memoryTTL: number;
  storageTTL: number;
  storagePrefix: string;
}

interface CacheStats {
  memoryHits: number;
  storageHits: number;
  misses: number;
}

interface CacheStatsResult {
  memoryHits: number;
  storageHits: number;
  misses: number;
  totalHits: number;
  totalRequests: number;
  hitRate: string;
  memorySize: number;
  storageSize: number;
  totalSize: number;
}

// 默认配置
const DEFAULT_CONFIG: CacheConfig = {
  memoryTTL: DEFAULT_MEMORY_TTL,
  storageTTL: DEFAULT_STORAGE_TTL,
  storagePrefix: DEFAULT_STORAGE_PREFIX,
};

// ==================== 内存缓存存储 ====================

class MemoryCache<T> {
  private cache: Map<string, CacheItem<T>>;
  private readonly ttl: number;

  constructor(ttl: number) {
    this.cache = new Map();
    this.ttl = ttl;
  }

  set(key: string, data: T): void {
    const now = Date.now();
    this.cache.set(key, {
      data,
      timestamp: now,
      expiresAt: now + this.ttl,
    });
  }

  get(key: string): T | null {
    const item = this.cache.get(key);
    if (!item) {
      return null;
    }

    if (Date.now() > item.expiresAt) {
      this.cache.delete(key);
      return null;
    }

    return item.data;
  }

  has(key: string): boolean {
    return this.get(key) !== null;
  }

  delete(key: string): void {
    this.cache.delete(key);
  }

  clear(): void {
    this.cache.clear();
  }

  cleanup(): void {
    const now = Date.now();
    for (const [key, item] of this.cache.entries()) {
      if (now > item.expiresAt) {
        this.cache.delete(key);
      }
    }
  }

  get size(): number {
    return this.cache.size;
  }
}

// ==================== localStorage 缓存存储 ====================

class StorageCache<T> {
  private readonly prefix: string;
  private readonly ttl: number;

  constructor(prefix: string, ttl: number) {
    this.prefix = prefix;
    this.ttl = ttl;
  }

  private getStorageKey(key: string): string {
    return `${this.prefix}${key}`;
  }

  private isExpired(item: CacheItem<T>): boolean {
    return Date.now() > item.expiresAt;
  }

  set(key: string, data: T): void {
    try {
      const now = Date.now();
      const item: CacheItem<T> = {
        data,
        timestamp: now,
        expiresAt: now + this.ttl,
      };
      localStorage.setItem(this.getStorageKey(key), JSON.stringify(item));
    } catch (error) {
      console.warn(`${LOG_PREFIX} localStorage write failed:`, error);
    }
  }

  get(key: string): T | null {
    try {
      const raw = localStorage.getItem(this.getStorageKey(key));
      if (!raw) {
        return null;
      }

      const item: CacheItem<T> = JSON.parse(raw);

      if (this.isExpired(item)) {
        this.delete(key);
        return null;
      }

      return item.data;
    } catch (error) {
      console.warn(`${LOG_PREFIX} localStorage read failed:`, error);
      return null;
    }
  }

  has(key: string): boolean {
    return this.get(key) !== null;
  }

  delete(key: string): void {
    try {
      localStorage.removeItem(this.getStorageKey(key));
    } catch (error) {
      console.warn(`${LOG_PREFIX} localStorage delete failed:`, error);
    }
  }

  clear(): void {
    try {
      const keys = Object.keys(localStorage);
      for (const key of keys) {
        if (key.startsWith(this.prefix)) {
          localStorage.removeItem(key);
        }
      }
    } catch (error) {
      console.warn(`${LOG_PREFIX} localStorage clear failed:`, error);
    }
  }

  cleanup(): void {
    try {
      const now = Date.now();
      const keys = Object.keys(localStorage);

      for (const storageKey of keys) {
        if (!storageKey.startsWith(this.prefix)) {
          continue;
        }

        try {
          const raw = localStorage.getItem(storageKey);
          if (!raw) {
            continue;
          }

          const item: CacheItem<T> = JSON.parse(raw);
          if (now > item.expiresAt) {
            localStorage.removeItem(storageKey);
          }
        } catch {
          // 损坏的条目
          localStorage.removeItem(storageKey);
        }
      }
    } catch (error) {
      console.warn(`${LOG_PREFIX} localStorage cleanup failed:`, error);
    }
  }

  getSize(): number {
    try {
      const keys = Object.keys(localStorage);
      return keys.filter((key) => key.startsWith(this.prefix)).length;
    } catch {
      return 0;
    }
  }
}

// ==================== 双层缓存类 ====================

export class DualLevelCache<T> {
  private memoryCache: MemoryCache<T>;
  private storageCache: StorageCache<T>;
  private cleanupTimer: ReturnType<typeof setInterval> | null = null;
  private readonly config: CacheConfig;
  private stats: CacheStats = {
    memoryHits: 0,
    storageHits: 0,
    misses: 0,
  };

  constructor(config: Partial<CacheConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.memoryCache = new MemoryCache<T>(this.config.memoryTTL);
    this.storageCache = new StorageCache<T>(this.config.storagePrefix, this.config.storageTTL);

    // 定期清理过期缓存
    this.cleanupTimer = setInterval(() => {
      this.memoryCache.cleanup();
      this.storageCache.cleanup();
    }, CLEANUP_INTERVAL);

    // 初始化时清理一次
    this.cleanup();
  }

  /**
   * 生成缓存键
   */
  generateKey(params: Record<string, unknown>): string {
    const sortedParams = Object.keys(params)
      .sort()
      .map((key) => `${key}=${params[key]}`)
      .join("&");
    return btoa(unescape(encodeURIComponent(sortedParams)));
  }

  /**
   * 获取缓存（优先内存，后localStorage）
   */
  get(key: string): T | null {
    // 先查内存缓存
    const memValue = this.memoryCache.get(key);
    if (memValue) {
      this.stats.memoryHits++;
      return memValue;
    }

    // 再查 localStorage
    const storageValue = this.storageCache.get(key);
    if (storageValue) {
      this.stats.storageHits++;
      // 回填到内存缓存
      this.memoryCache.set(key, storageValue);
      return storageValue;
    }

    this.stats.misses++;
    return null;
  }

  /**
   * 设置缓存（同时写入内存和localStorage）
   */
  set(key: string, data: T, persistToStorage = true): void {
    this.memoryCache.set(key, data);
    if (persistToStorage) {
      this.storageCache.set(key, data);
    }
  }

  /**
   * 检查缓存是否存在
   */
  has(key: string): boolean {
    return this.memoryCache.has(key) || this.storageCache.has(key);
  }

  /**
   * 删除缓存
   */
  delete(key: string): void {
    this.memoryCache.delete(key);
    this.storageCache.delete(key);
  }

  /**
   * 清空所有缓存
   */
  clear(): void {
    this.memoryCache.clear();
    this.storageCache.clear();
  }

  /**
   * 清理过期缓存
   */
  cleanup(): void {
    this.memoryCache.cleanup();
    this.storageCache.cleanup();
  }

  /**
   * 销毁缓存
   */
  destroy(): void {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
    this.clear();
  }

  /**
   * 获取缓存统计信息
   */
  getStats(): CacheStatsResult {
    const totalHits = this.stats.memoryHits + this.stats.storageHits;
    const totalRequests = totalHits + this.stats.misses;

    return {
      memoryHits: this.stats.memoryHits,
      storageHits: this.stats.storageHits,
      misses: this.stats.misses,
      totalHits,
      totalRequests,
      hitRate: totalRequests > 0 ? `${((totalHits / totalRequests) * 100).toFixed(2)}%` : "0%",
      memorySize: this.memoryCache.size,
      storageSize: this.storageCache.getSize(),
      totalSize: this.memoryCache.size + this.storageCache.getSize(),
    };
  }

  /**
   * 重置统计信息
   */
  resetStats(): void {
    this.stats = {
      memoryHits: 0,
      storageHits: 0,
      misses: 0,
    };
  }

  /**
   * 打印统计信息
   */
  printStats(): void {
    const stats = this.getStats();
    console.table({
      "Memory Hits": stats.memoryHits,
      "Storage Hits": stats.storageHits,
      Misses: stats.misses,
      "Total Hits": stats.totalHits,
      "Total Requests": stats.totalRequests,
      "Hit Rate": stats.hitRate,
      "Memory Size": stats.memorySize,
      "Storage Size": stats.storageSize,
      "Total Size": stats.totalSize,
    });
  }
}

// 导出单例实例
let cacheInstance: DualLevelCache<unknown> | null = null;

export const getDualLevelCache = <T>(): DualLevelCache<T> => {
  if (!cacheInstance) {
    cacheInstance = new DualLevelCache<unknown>();
  }
  return cacheInstance as DualLevelCache<T>;
};

export const clearDualLevelCache = (): void => {
  if (cacheInstance) {
    cacheInstance.destroy();
    cacheInstance = null;
  }
};

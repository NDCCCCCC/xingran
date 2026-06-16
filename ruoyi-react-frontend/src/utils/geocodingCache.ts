/**
 * 百度地图地理编码缓存工具
 * 使用内存缓存 + localStorage 实现双层缓存
 * 内存缓存：快速访问
 * localStorage：持久化存储，跨会话可用
 */

// 缓存项接口
interface CacheItem<T> {
  data: T;
  timestamp: number;
  expiresAt: number;
}

// 缓存配置
interface CacheConfig {
  // 内存缓存过期时间（毫秒）
  memoryTTL: number;
  // localStorage 缓存过期时间（毫秒）
  storageTTL: number;
  // localStorage 存储键前缀
  storagePrefix: string;
}

// 默认配置：地理坐标数据变化极小，可以设置较长的过期时间
const DEFAULT_CONFIG: CacheConfig = {
  memoryTTL: 30 * 60 * 1000, // 30分钟
  storageTTL: 7 * 24 * 60 * 60 * 1000, // 7天
  storagePrefix: "baidu_geocoding_",
};

// 内存缓存存储
class MemoryCache<T> {
  private cache: Map<string, CacheItem<T>>;
  private ttl: number;

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
    if (!item) return null;

    // 检查是否过期
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

  // 清理过期项
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

// localStorage 缓存存储
class StorageCache<T> {
  private prefix: string;
  private ttl: number;

  constructor(prefix: string, ttl: number) {
    this.prefix = prefix;
    this.ttl = ttl;
  }

  private getStorageKey(key: string): string {
    return `${this.prefix}${key}`;
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
    } catch (e) {
      // localStorage 可能已满或被禁用，静默失败
      console.warn("localStorage write failed:", e);
    }
  }

  get(key: string): T | null {
    try {
      const raw = localStorage.getItem(this.getStorageKey(key));
      if (!raw) return null;

      const item: CacheItem<T> = JSON.parse(raw);

      // 检查是否过期
      if (Date.now() > item.expiresAt) {
        this.delete(key);
        return null;
      }

      return item.data;
    } catch (e) {
      console.warn("localStorage read failed:", e);
      return null;
    }
  }

  has(key: string): boolean {
    return this.get(key) !== null;
  }

  delete(key: string): void {
    try {
      localStorage.removeItem(this.getStorageKey(key));
    } catch (e) {
      console.warn("localStorage delete failed:", e);
    }
  }

  clear(): void {
    try {
      // 清理所有带前缀的项
      const keys = Object.keys(localStorage);
      keys.forEach(key => {
        if (key.startsWith(this.prefix)) {
          localStorage.removeItem(key);
        }
      });
    } catch (e) {
      console.warn("localStorage clear failed:", e);
    }
  }

  // 清理过期项
  cleanup(): void {
    try {
      const now = Date.now();
      const keys = Object.keys(localStorage);
      keys.forEach(storageKey => {
        if (!storageKey.startsWith(this.prefix)) return;

        try {
          const raw = localStorage.getItem(storageKey);
          if (!raw) return;

          const item: CacheItem<T> = JSON.parse(raw);
          if (now > item.expiresAt) {
            localStorage.removeItem(storageKey);
          }
        } catch (e) {
          // 解析失败，删除该项
          localStorage.removeItem(storageKey);
        }
      });
    } catch (e) {
      console.warn("localStorage cleanup failed:", e);
    }
  }
}

// 地理编码缓存类
export class GeocodingCache<T> {
  private memoryCache: MemoryCache<T>;
  private storageCache: StorageCache<T>;
  private cleanupTimer: ReturnType<typeof setInterval> | null;

  constructor(config: Partial<CacheConfig> = {}) {
    const finalConfig = { ...DEFAULT_CONFIG, ...config };

    this.memoryCache = new MemoryCache<T>(finalConfig.memoryTTL);
    this.storageCache = new StorageCache<T>(finalConfig.storagePrefix, finalConfig.storageTTL);

    // 定期清理过期缓存（每5分钟）
    this.cleanupTimer = setInterval(() => {
      this.memoryCache.cleanup();
      this.storageCache.cleanup();
    }, 5 * 60 * 1000);

    // 初始化时清理一次过期缓存
    this.memoryCache.cleanup();
    this.storageCache.cleanup();
  }

  /**
   * 生成缓存键
   * @param params 缓存参数
   */
  generateKey(params: Record<string, unknown>): string {
    // 将参数排序后生成键，确保相同参数生成相同键
    const sortedParams = Object.keys(params)
      .sort()
      .map(key => `${key}=${params[key]}`)
      .join("&");
    return btoa(unescape(encodeURIComponent(sortedParams)));
  }

  /**
   * 获取缓存
   * 优先从内存缓存获取，若不存在则从 localStorage 获取
   */
  get(key: string): T | null {
    // 先查内存缓存
    const memValue = this.memoryCache.get(key);
    if (memValue) {
      return memValue;
    }

    // 再查 localStorage
    const storageValue = this.storageCache.get(key);
    if (storageValue) {
      // 回填到内存缓存
      this.memoryCache.set(key, storageValue);
      return storageValue;
    }

    return null;
  }

  /**
   * 设置缓存
   * 同时写入内存缓存和 localStorage
   */
  set(key: string, data: T): void {
    this.memoryCache.set(key, data);
    this.storageCache.set(key, data);
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
   * 获取或设置缓存（缓存穿透保护）
   * @param key 缓存键
   * @param factory 数据获取函数（缓存未命中时调用）
   */
  async getOrSet(key: string, factory: () => Promise<T>): Promise<T> {
    const cached = this.get(key);
    if (cached !== null) {
      return cached;
    }

    const data = await factory();
    this.set(key, data);
    return data;
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
  getStats() {
    return {
      memorySize: this.memoryCache.size,
      // localStorage 大小需要遍历计算，比较耗时，这里只返回内存大小
    };
  }
}

// 导出单例实例（使用默认配置）
let cacheInstance: GeocodingCache<unknown> | null = null;

export const getGeocodingCache = <T>(): GeocodingCache<T> => {
  if (!cacheInstance) {
    cacheInstance = new GeocodingCache<unknown>();
  }
  return cacheInstance as GeocodingCache<T>;
};

export const clearGeocodingCache = (): void => {
  if (cacheInstance) {
    cacheInstance.destroy();
    cacheInstance = null;
  }
};

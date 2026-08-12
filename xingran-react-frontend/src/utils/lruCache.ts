/**
 * 简单的 LRU (Least Recently Used) 缓存实现
 * 用于 memoize 纯函数（如格式化时间戳），避免重复计算
 * 容量超限时移除最久未访问的项
 */
export class LRUCache<K, V> {
  private cache = new Map<K, V>();

  constructor(private readonly capacity: number) {
    if (capacity <= 0) throw new Error("LRUCache capacity must be > 0");
  }

  get(key: K): V | undefined {
    const value = this.cache.get(key);
    if (value === undefined) return undefined;
    // Move to end (most recent)
    this.cache.delete(key);
    this.cache.set(key, value);
    return value;
  }

  set(key: K, value: V): void {
    if (this.cache.has(key)) {
      this.cache.delete(key);
    } else if (this.cache.size >= this.capacity) {
      // Evict oldest (first key in iteration order)
      const oldestKey = this.cache.keys().next().value;
      if (oldestKey !== undefined) this.cache.delete(oldestKey);
    }
    this.cache.set(key, value);
  }

  has(key: K): boolean {
    return this.cache.has(key);
  }

  clear(): void {
    this.cache.clear();
  }

  get size(): number {
    return this.cache.size;
  }
}

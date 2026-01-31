/**
 * 缓存统计类
 * @deprecated 此文件已弃用，统计功能已集成到 BaiduMapCache 类中。
 * 请使用 BaiduMapCache.getStats() 和 BaiduMapCache.resetStats() 代替。
 * 此文件保留仅为向后兼容，将在未来版本中移除。
 */

export class CacheStats {
  private stats = {
    memoryHits: 0,
    storageHits: 0,
    apiCalls: 0,
    misses: 0,
  };

  /**
   * 记录内存缓存命中
   * @deprecated 使用 BaiduMapCache.getStats() 代替
   */
  recordMemoryHit(): void {
    this.stats.memoryHits++;
    console.log('[CacheStats] Memory cache hit');
  }

  /**
   * 记录localStorage缓存命中
   * @deprecated 使用 BaiduMapCache.getStats() 代替
   */
  recordStorageHit(): void {
    this.stats.storageHits++;
    console.log('[CacheStats] localStorage cache hit');
  }

  /**
   * 记录API调用
   * @deprecated 使用 BaiduMapCache.getStats() 代替
   */
  recordAPICall(): void {
    this.stats.apiCalls++;
    console.log('[CacheStats] API call');
  }

  /**
   * 记录缓存未命中
   * @deprecated 使用 BaiduMapCache.getStats() 代替
   */
  recordMiss(): void {
    this.stats.misses++;
    console.log('[CacheStats] Cache miss');
  }

  /**
   * 获取统计信息
   * @deprecated 使用 BaiduMapCache.getStats() 代替
   */
  getStats() {
    const totalHits = this.stats.memoryHits + this.stats.storageHits;
    const totalRequests = totalHits + this.stats.apiCalls;

    return {
      memoryHits: this.stats.memoryHits,
      storageHits: this.stats.storageHits,
      apiCalls: this.stats.apiCalls,
      misses: this.stats.misses,
      totalHits,
      totalRequests,
      hitRate: totalRequests > 0 ? ((totalHits / totalRequests) * 100).toFixed(2) + '%' : '0%',
    };
  }

  /**
   * 重置统计
   * @deprecated 使用 BaiduMapCache.resetStats() 代替
   */
  reset(): void {
    this.stats = {
      memoryHits: 0,
      storageHits: 0,
      apiCalls: 0,
      misses: 0,
    };
    console.log('[CacheStats] Stats reset');
  }

  /**
   * 打印统计信息
   * @deprecated 使用 BaiduMapCache.printStats() 代替
   */
  print(): void {
    const stats = this.getStats();
    console.table({
      'Memory Hits': stats.memoryHits,
      'Storage Hits': stats.storageHits,
      'API Calls': stats.apiCalls,
      'Misses': stats.misses,
      'Total Hits': stats.totalHits,
      'Total Requests': stats.totalRequests,
      'Hit Rate': stats.hitRate,
    });
  }
}

// 导出全局单例
// @deprecated 使用 BaiduMapCache 实例的统计功能代替
export const cacheStats = new CacheStats();

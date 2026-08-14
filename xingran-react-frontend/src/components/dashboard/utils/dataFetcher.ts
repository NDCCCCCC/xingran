/**
 * DataFetcher - 数据获取器
 *
 * 统一的数据获取接口，支持API、WebSocket和静态数据
 */

import { get, post } from "@/lib/api";
import type {
  DataSourceConfig,
  ApiDataSourceConfig,
  WebSocketDataSourceConfig,
  StaticDataSourceConfig,
} from "@/types/dashboard";
import type { BaseResponse } from "@/types";

/**
 * 数据获取结果
 */
export interface DataFetchResult<T = unknown> {
  /** 数据 */
  data: T;
  /** 时间戳 */
  timestamp: number;
  /** 是否有错误 */
  error?: string;
}

/**
 * 数据获取器类
 */
export class DataFetcher {
  /** API 请求缓存 */
  private apiCache = new Map<string, DataFetchResult<unknown>>();

  /** WebSocket 连接缓存 */
  private wsConnections = new Map<string, WebSocket>();

  /** 缓存过期时间（毫秒） */
  private cacheExpiry = 60000; // 默认1分钟

  /**
   * 获取数据
   */
  async fetch<T = unknown>(dataSource: DataSourceConfig): Promise<DataFetchResult<T>> {
    // 使用类型守卫判断数据源类型
    if ("type" in dataSource && dataSource.type === "api") {
      return this.fetchFromAPI(dataSource as ApiDataSourceConfig);
    }
    if ("type" in dataSource && dataSource.type === "websocket") {
      return this.fetchFromWebSocket(dataSource as WebSocketDataSourceConfig);
    }
    if ("type" in dataSource && dataSource.type === "static") {
      return this.fetchStatic(dataSource as StaticDataSourceConfig);
    }

    return {
      data: null as T,
      timestamp: Date.now(),
      error: "无效的数据源配置",
    };
  }

  /**
   * 从API获取数据
   */
  private async fetchFromAPI<T = unknown>(
    config: ApiDataSourceConfig
  ): Promise<DataFetchResult<T>> {
    try {
      // 检查缓存
      const cacheKey = this.getCacheKey(config);
      const cached = this.apiCache.get(cacheKey);
      if (cached && Date.now() - cached.timestamp < this.cacheExpiry) {
        return cached as DataFetchResult<T>;
      }

      let response: BaseResponse<unknown>;

      // 根据HTTP方法发送请求
      if (config.method === "GET") {
        response = await get(config.endpoint, config.params);
      } else {
        response = await post(config.endpoint, config.params ?? config.body ?? {});
      }

      // 提取数据：标准API响应格式为 { code: 0, data: {...} }
      let actualData: unknown = response;

      // 如果响应有 code 字段，说明是标准API响应
      if (response && typeof response === "object" && "code" in response) {
        if (response.code === 0 && response.data !== undefined) {
          actualData = response.data;
        } else {
          // 业务错误
          return {
            data: null as T,
            timestamp: Date.now(),
            error: response.message || "API请求失败",
          };
        }
      }

      const result: DataFetchResult<T> = {
        data: actualData as T,
        timestamp: Date.now(),
      };

      // 更新缓存 - 缓存提取后的实际数据
      this.apiCache.set(cacheKey, result as DataFetchResult<unknown>);

      return result;
    } catch (error) {
      return {
        data: null as T,
        timestamp: Date.now(),
        error: (error as Error).message,
      };
    }
  }

  /**
   * 从WebSocket获取数据
   */
  private fetchFromWebSocket<T = unknown>(
    config: WebSocketDataSourceConfig
  ): Promise<DataFetchResult<T>> {
    return new Promise((resolve) => {
      try {
        // 检查是否已有连接
        let ws = this.wsConnections.get(config.channel);

        if (!ws || ws.readyState !== WebSocket.OPEN) {
          const wsUrl = this.getWebSocketUrl(config.channel);
          ws = new WebSocket(wsUrl);
          this.wsConnections.set(config.channel, ws);
        }

        // 设置一次性消息处理
        const handler = (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data);
            resolve({
              data: data as T,
              timestamp: Date.now(),
            });
            ws?.removeEventListener("message", handler);
          } catch {
            // 忽略解析错误
          }
        };

        ws.addEventListener("message", handler);

        // 超时处理
        setTimeout(() => {
          ws?.removeEventListener("message", handler);
          resolve({
            data: null as T,
            timestamp: Date.now(),
            error: "WebSocket请求超时",
          });
        }, 5000);
      } catch (error) {
        resolve({
          data: null as T,
          timestamp: Date.now(),
          error: (error as Error).message,
        });
      }
    });
  }

  /**
   * 获取静态数据
   */
  private fetchStatic<T = unknown>(config: StaticDataSourceConfig): Promise<DataFetchResult<T>> {
    return Promise.resolve({
      data: config.data as T,
      timestamp: Date.now(),
    });
  }

  /**
   * 获取缓存键
   */
  private getCacheKey(config: ApiDataSourceConfig): string {
    return `${config.method}:${config.endpoint}:${JSON.stringify(config.params ?? {})}`;
  }

  /**
   * 获取WebSocket URL
   */
  private getWebSocketUrl(channel: string): string {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = import.meta.env.VITE_WS_HOST || window.location.host;
    const basePath = import.meta.env.VITE_WS_BASE_PATH || "/ws";
    return `${protocol}//${host}${basePath}/${channel}`;
  }

  /**
   * 清除缓存
   */
  clearCache(pattern?: string): void {
    if (pattern) {
      for (const key of this.apiCache.keys()) {
        if (key.includes(pattern)) {
          this.apiCache.delete(key);
        }
      }
    } else {
      this.apiCache.clear();
    }
  }

  /**
   * 关闭WebSocket连接
   */
  closeWebSocket(channel?: string): void {
    if (channel) {
      const ws = this.wsConnections.get(channel);
      if (ws) {
        ws.close();
        this.wsConnections.delete(channel);
      }
    } else {
      this.wsConnections.forEach((ws) => ws.close());
      this.wsConnections.clear();
    }
  }

  /**
   * 设置缓存过期时间
   */
  setCacheExpiry(ms: number): void {
    this.cacheExpiry = ms;
  }
}

// 导出单例
export const dataFetcher = new DataFetcher();

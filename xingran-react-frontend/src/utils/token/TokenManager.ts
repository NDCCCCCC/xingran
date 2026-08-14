/**
 * Token 管理核心类
 * 使用国密 SM4 加密保护 Token 存储
 *
 * 职责：
 * 1. 管理 Token 的存储和获取
 * 2. 自动刷新 Token（Silent Refresh）
 * 3. 防止并发刷新（使用刷新锁）
 * 4. 处理刷新失败
 */

import type { SecureTokenStorage, TokenMeta, TokenRefreshResponse } from "./SecureTokenStorage";
import { TokenRefreshError } from "./SecureTokenStorage";
import { post } from "@/lib/api";

/**
 * Token 刷新锁（防止并发刷新）
 */
type RefreshLock = {
  promise: Promise<TokenRefreshResponse>;
  timestamp: number;
};

/**
 * TokenManager 配置
 */
export interface TokenManagerConfig {
  /**
   * 刷新 Token 的 API 端点
   */
  refreshEndpoint: string;

  /**
   * 提前刷新时间（秒）
   * 在 Token 过期前多少秒开始刷新
   */
  refreshBeforeSeconds: number;

  /**
   * 刷新超时时间（毫秒）
   */
  refreshTimeout: number;
}

/**
 * Token 管理核心类
 */
export class TokenManager {
  private storage: SecureTokenStorage;
  private config: TokenManagerConfig;
  private refreshLock: RefreshLock | null = null;
  private refreshTimer: number | null = null;

  constructor(storage: SecureTokenStorage, config: TokenManagerConfig) {
    this.storage = storage;
    this.config = config;
  }

  /**
   * 初始化 Token（登录成功后调用）
   */
  async initializeTokens(
    accessToken: string,
    refreshToken: string,
    expiresIn: number
  ): Promise<void> {
    const now = Date.now();
    const meta: TokenMeta = {
      expiresAt: now + expiresIn * 1000,
      issuedAt: now,
      expiresIn,
    };

    // 存储 Token
    this.storage.setAccessToken(accessToken);
    await this.storage.setRefreshToken(refreshToken);
    this.storage.setTokenMeta(meta);

    // 启动自动刷新定时器
    this.scheduleRefresh();
  }

  /**
   * 获取当前 AccessToken
   * 如果即将过期，会自动刷新
   */
  async getAccessToken(): Promise<string> {
    const token = this.storage.getAccessToken();

    if (!token) {
      throw new TokenRefreshError("No access token available", "INVALID_TOKEN");
    }

    // 检查是否需要刷新
    if (this.storage.isAccessTokenExpiringWithin(this.config.refreshBeforeSeconds)) {
      await this.refreshToken();
    }

    return this.storage.getAccessToken()!;
  }

  /**
   * 获取 RefreshToken
   */
  async getRefreshToken(): Promise<string | null> {
    return this.storage.getRefreshToken();
  }

  /**
   * 刷新 Token
   * 使用刷新锁防止并发刷新
   */
  async refreshToken(): Promise<TokenRefreshResponse> {
    // 如果已有刷新在进行中，返回同一个 Promise
    if (this.refreshLock) {
      // 检查锁是否超时
      if (Date.now() - this.refreshLock.timestamp > this.config.refreshTimeout) {
        this.refreshLock = null; // 锁超时，清除
      } else {
        return this.refreshLock.promise;
      }
    }

    // 创建刷新 Promise（动态导入避免循环依赖）
    const refreshPromise = this.doRefresh();

    // 设置刷新锁
    this.refreshLock = {
      promise: refreshPromise,
      timestamp: Date.now(),
    };

    try {
      const result = await refreshPromise;
      return result;
    } finally {
      // 清除刷新锁
      this.refreshLock = null;
    }
  }

  /**
   * 执行实际的 Token 刷新
   */
  private async doRefresh(): Promise<TokenRefreshResponse> {
    const refreshToken = await this.storage.getRefreshToken();

    if (!refreshToken) {
      throw new TokenRefreshError("No refresh token available", "INVALID_TOKEN");
    }

    try {
      // post 已在顶部静态导入（@/lib/api）。原本用动态 import 注释说"避免循环依赖"，
      // 但实际依赖链 api.ts → authStore → TokenManager 是单向的，没有循环。
      const response = (await post(this.config.refreshEndpoint, { refreshToken })) as {
        data: TokenRefreshResponse;
      };

      const { accessToken, refreshToken: newRefreshToken, expiresIn } = response.data;

      // 更新存储
      await this.initializeTokens(accessToken, newRefreshToken, expiresIn);

      return {
        accessToken,
        refreshToken: newRefreshToken,
        expiresIn,
      };
    } catch (error: unknown) {
      console.error("[TokenManager] 刷新 Token 失败:", error);

      // 根据错误类型分类
      const axiosError = error as { response?: { status?: number }; code?: string };
      if (axiosError.response?.status === 401) {
        throw new TokenRefreshError("Refresh token expired", "INVALID_TOKEN");
      } else if (axiosError.code === "NETWORK_ERROR") {
        throw new TokenRefreshError("Network error during refresh", "NETWORK_ERROR");
      } else {
        throw new TokenRefreshError("Server error during refresh", "SERVER_ERROR");
      }
    }
  }

  /**
   * 调度下一次刷新
   */
  private scheduleRefresh(): void {
    // 清除现有定时器
    if (this.refreshTimer !== null) {
      clearTimeout(this.refreshTimer);
    }

    const meta = this.storage.getTokenMeta();
    if (!meta) {
      return;
    }

    const now = Date.now();
    const expiresAt = meta.expiresAt;
    const refreshBefore = this.config.refreshBeforeSeconds * 1000;

    // 计算刷新时间（过期前 N 秒）
    const refreshTime = expiresAt - refreshBefore;

    // 如果已经过期，立即刷新
    if (refreshTime <= now) {
      this.refreshToken().catch((error) => {
        console.error("[TokenManager] 自动刷新失败:", error);
      });
      return;
    }

    // 设置定时器
    const delay = refreshTime - now;
    this.refreshTimer = window.setTimeout(() => {
      this.refreshToken().catch((error) => {
        console.error("[TokenManager] 定时刷新失败:", error);
      });
    }, delay);
  }

  /**
   * 清除所有 Token
   */
  async clearTokens(): Promise<void> {
    // 清除定时器
    if (this.refreshTimer !== null) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }

    // 清除刷新锁
    this.refreshLock = null;

    // 清除存储
    await this.storage.clear();
  }

  /**
   * 检查是否已认证
   */
  isAuthenticated(): boolean {
    return this.storage.getAccessToken() !== null;
  }

  /**
   * 获取 Token 剩余有效时间（秒）
   */
  getTokenRemainingTime(): number {
    const meta = this.storage.getTokenMeta();
    if (!meta || !meta.expiresAt) {
      return 0;
    }
    const now = Date.now();
    const remaining = Math.max(0, Math.floor((meta.expiresAt - now) / 1000));
    return remaining;
  }
}

// 重新导出错误类型
export type { TokenRefreshError };

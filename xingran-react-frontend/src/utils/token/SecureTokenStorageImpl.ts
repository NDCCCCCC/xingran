/**
 * 安全 Token 存储实现
 * 使用国密 SM4-CBC 加密保护 RefreshToken
 *
 * 特性：
 * - AccessToken: 存储在内存中
 * - RefreshToken: SM4 加密后存储在 sessionStorage
 * - TokenMeta: 持久化到 sessionStorage (JSON, 非敏感: 仅含 expiresAt)
 *
 * 修复历史:
 *   - P1-S1 (前端审查): 原 deriveStorageKey/deriveStorageIV 的输入字符串
 *     前 16 字节恰好相同("xingran-next-sec"), 导致派生的 Key 与 IV 完全
 *     相同 (78696e67...), SM4-CBC Key=IV 安全性归零。现用 hash 派生确保不同。
 *   - P1-R3 (前端审查): tokenMeta 仅存内存, 页面刷新后丢失导致
 *     isAccessTokenExpiringWithin 永远 false, 自动刷新定时器失效。
 *     现 tokenMeta 持久化到 sessionStorage。
 *
 * 安全说明: 前端 JS 中的密钥无法真正保密(可被 F12 提取), 此 SM4 加密
 * 的真实作用是"混淆 sessionStorage 内容, 增加爬虫/简单脚本的提取成本",
 * 不是对抗有动机攻击者的真正保护。真正的 RefreshToken 安全依赖后端
 * (一次性/绑定/IP 校验)。这里保留加密以维持现有架构兼容。
 */

import { encryptSM4CBC, decryptSM4CBC } from "@/utils/sm4";
import { hexToBase64, base64ToHex } from "@/utils/encoding";
import type { SecureTokenStorage, TokenMeta } from "./SecureTokenStorage";

/**
 * 存储密钥常量（使用简短键名避免暴露）
 */
const STORAGE_KEYS = {
  REFRESH_TOKEN: "rt", // 加密存储
  TOKEN_META: "tm", // JSON 存储 (非敏感: 仅 expiresAt)
} as const;

/**
 * 简单字符串散列 (djb2 变体) → 32 hex 字符
 * 用于从不同 salt 派生 Key / IV, 保证两者不同。
 * 非密码学强度, 但足够确保 Key≠IV 且每次派生结果确定。
 */
function deriveHex(input: string, salt: string): string {
  // 双轮 djb2, 输出 32 位; 拼接 4 轮不同 seed 凑够 32 hex 字符 (16 字节)
  const rounds = 4;
  let result = "";
  for (let r = 0; r < rounds; r++) {
    let h = 0x811c9dc5 ^ (r * 0x01000193 + 1); // FNV offset basis 随轮次变化
    const s = input + "|" + salt + "|" + r;
    for (let i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i);
      h = Math.imul(h, 0x01000193); // FNV prime (32-bit)
    }
    // 转为无符号 32 位 hex
    const hex = (h >>> 0).toString(16).padStart(8, "0");
    result += hex;
  }
  return result.substring(0, 32); // 确保正好 32 hex 字符 (16 字节)
}

/**
 * 派生存储加密密钥 (16 字节, 32 hex)
 * Key 使用 "-key-" salt, 与 IV 的 "-iv-" salt 确保不同。
 */
function deriveStorageKey(): string {
  return deriveHex("xingran-next-secure-storage", "sm4-key-2024");
}

/**
 * 派生存储 IV (16 字节, 32 hex)
 * IV 使用 "-iv-" salt, 与 Key 不同。
 */
function deriveStorageIV(): string {
  return deriveHex("xingran-next-secure-storage", "sm4-iv-2024");
}

// 派生密钥和 IV（单例, 启动时计算一次）
const STORAGE_ENCRYPTION_KEY = deriveStorageKey();
const STORAGE_ENCRYPTION_IV = deriveStorageIV();

/**
 * 安全 Token 存储实现类
 */
export class SecureTokenStorageImpl implements SecureTokenStorage {
  private accessToken: string | null = null;
  private tokenMeta: TokenMeta | null = null;

  /**
   * 存储 AccessToken（内存）
   */
  setAccessToken(token: string): void {
    this.accessToken = token;
  }

  /**
   * 获取 AccessToken（从内存）
   */
  getAccessToken(): string | null {
    return this.accessToken;
  }

  /**
   * 存储 RefreshToken（sessionStorage，SM4 加密）
   */
  async setRefreshToken(token: string): Promise<void> {
    try {
      // 使用 SM4-CBC 加密 token
      const encryptedHex = await encryptSM4CBC(
        token,
        STORAGE_ENCRYPTION_KEY,
        STORAGE_ENCRYPTION_IV
      );

      // 转换为 Base64 存储
      const encryptedBase64 = hexToBase64(encryptedHex);

      // 存储到 sessionStorage（会话级别，关闭标签页自动清除）
      sessionStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, encryptedBase64);
    } catch (error) {
      console.error("[SecureTokenStorage] SM4 加密 RefreshToken 失败:", error);
      throw new Error("Failed to encrypt refresh token");
    }
  }

  /**
   * 获取 RefreshToken（从 sessionStorage，SM4 解密）
   */
  async getRefreshToken(): Promise<string | null> {
    try {
      const encryptedBase64 = sessionStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);

      if (!encryptedBase64) {
        return null;
      }

      // 从 Base64 转换为 Hex
      const encryptedHex = base64ToHex(encryptedBase64);

      // 使用 SM4-CBC 解密
      const decryptedToken = await decryptSM4CBC(
        encryptedHex,
        STORAGE_ENCRYPTION_KEY,
        STORAGE_ENCRYPTION_IV
      );

      return decryptedToken;
    } catch (error) {
      console.error("[SecureTokenStorage] SM4 解密 RefreshToken 失败:", error);
      // 解密失败时清除损坏的数据
      sessionStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
      return null;
    }
  }

  /**
   * 存储 Token 元数据 (内存 + sessionStorage 持久化)
   * P1-R3: 持久化 expiresAt, 使刷新页面后 isAccessTokenExpiringWithin 仍可用。
   */
  setTokenMeta(meta: TokenMeta): void {
    this.tokenMeta = meta;
    try {
      sessionStorage.setItem(STORAGE_KEYS.TOKEN_META, JSON.stringify(meta));
    } catch {
      // sessionStorage 不可用时仅内存兜底
    }
  }

  /**
   * 获取 Token 元数据 (优先内存, 回落 sessionStorage)
   */
  getTokenMeta(): TokenMeta | null {
    if (this.tokenMeta) {
      return this.tokenMeta;
    }
    // P1-R3: 刷新页面后从 sessionStorage 恢复
    try {
      const raw = sessionStorage.getItem(STORAGE_KEYS.TOKEN_META);
      if (raw) {
        this.tokenMeta = JSON.parse(raw) as TokenMeta;
        return this.tokenMeta;
      }
    } catch {
      // 损坏的 JSON 忽略
    }
    return null;
  }

  /**
   * 清除所有 Token
   */
  async clear(): Promise<void> {
    this.accessToken = null;
    this.tokenMeta = null;
    sessionStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    sessionStorage.removeItem(STORAGE_KEYS.TOKEN_META);
  }

  /**
   * 检查 AccessToken 是否即将过期
   * @param seconds 秒数，默认 30 秒
   */
  isAccessTokenExpiringWithin(seconds: number = 30): boolean {
    const meta = this.getTokenMeta();
    if (!meta || !meta.expiresAt) {
      return false;
    }

    const now = Date.now();
    const expiresAt = meta.expiresAt;
    const threshold = seconds * 1000;

    return expiresAt - now <= threshold;
  }
}

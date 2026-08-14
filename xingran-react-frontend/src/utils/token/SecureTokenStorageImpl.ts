/**
 * 安全 Token 存储实现
 * 使用国密 SM4-CBC 加密保护 RefreshToken
 *
 * 特性：
 * - AccessToken: 存储在内存中
 * - RefreshToken: SM4 加密后存储在 sessionStorage
 * - TokenMeta: 存储在内存中
 */

import { encryptSM4CBC, decryptSM4CBC, generateSM4Key, generateIV } from "@/utils/sm4";
import { hexToBase64, base64ToHex, generateRandomHex } from "@/utils/encoding";
import type { SecureTokenStorage, TokenMeta } from "./SecureTokenStorage";

/**
 * 存储密钥常量（使用简短键名避免暴露）
 */
const STORAGE_KEYS = {
  REFRESH_TOKEN: "rt", // 加密存储
  TOKEN_META: "tm", // JSON 存储
} as const;

/**
 * 派生存储加密密钥
 * 使用应用标识符派生固定密钥，用于 sessionStorage 加密
 * 注意：这个密钥仅用于前端存储加密，不是传输加密
 */
function deriveStorageKey(): string {
  // 使用固定的应用标识符派生密钥
  // 实际生产环境可以使用浏览器指纹增强安全性
  const appIdentifier = "xingran-next-secure-storage";
  const keyBase = appIdentifier + "-sm4-key-2024";
  // 将字符串转换为 32 字符的十六进制密钥
  let hexKey = "";
  for (let i = 0; i < keyBase.length; i++) {
    hexKey += keyBase.charCodeAt(i).toString(16).padStart(2, "0");
  }
  // 补齐或截断到 32 字符（16 字节）
  while (hexKey.length < 32) {
    hexKey += "0" + hexKey;
  }
  return hexKey.substring(0, 32);
}

/**
 * 派生存储 IV
 */
function deriveStorageIV(): string {
  const ivBase = "xingran-next-secure-iv-2024";
  let hexIV = "";
  for (let i = 0; i < ivBase.length; i++) {
    hexIV += ivBase.charCodeAt(i).toString(16).padStart(2, "0");
  }
  while (hexIV.length < 32) {
    hexIV += "0" + hexIV;
  }
  return hexIV.substring(0, 32);
}

// 派生密钥和 IV（单例）
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
   * 存储 Token 元数据
   */
  setTokenMeta(meta: TokenMeta): void {
    this.tokenMeta = meta;
  }

  /**
   * 获取 Token 元数据
   */
  getTokenMeta(): TokenMeta | null {
    return this.tokenMeta;
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
    if (!this.tokenMeta || !this.tokenMeta.expiresAt) {
      return false;
    }

    const now = Date.now();
    const expiresAt = this.tokenMeta.expiresAt;
    const threshold = seconds * 1000;

    return expiresAt - now <= threshold;
  }
}

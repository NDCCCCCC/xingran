/**
 * 前端安全工具模块
 * 提供数据完整性校验、XSS防护、CSRF token管理等安全功能
 */

/**
 * 使用 SubtleCrypto API 生成数据的 SHA-256 哈希
 * 用于验证存储在 localStorage 中的数据完整性
 */
export async function generateHash(data: unknown): Promise<string> {
  // 检查 crypto.subtle 是否可用（需要 secure context：HTTPS 或 localhost）
  if (!crypto.subtle) {
    console.warn("[Security] crypto.subtle 不可用，使用简单的哈希替代");
    // 使用简单的字符串哈希作为后备方案
    const jsonString = JSON.stringify(data);
    let hash = 0;
    for (let i = 0; i < jsonString.length; i++) {
      const char = jsonString.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32bit integer
    }
    return Math.abs(hash).toString(16);
  }

  const jsonString = JSON.stringify(data);
  const encoder = new TextEncoder();
  const dataBuffer = encoder.encode(jsonString);
  const hashBuffer = await crypto.subtle.digest("SHA-256", dataBuffer);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
  return hashHex;
}

/**
 * 验证数据的完整性
 * @param data 要验证的数据
 * @param expectedHash 期望的哈希值
 * @returns 数据是否未被篡改
 */
export async function verifyDataIntegrity(data: unknown, expectedHash: string): Promise<boolean> {
  const actualHash = await generateHash(data);
  return actualHash === expectedHash;
}

/**
 * 安全的存储工具（带完整性校验）
 */
export const SecureStorage = {
  /**
   * 安全地存储数据到 localStorage
   * @param key 存储键名
   * @param data 要存储的数据
   */
  async setItem<T>(key: string, data: T): Promise<void> {
    try {
      const hash = await generateHash(data);
      const storageData = {
        data,
        hash,
        timestamp: Date.now(),
      };
      localStorage.setItem(key, JSON.stringify(storageData));
    } catch (error) {
      console.error("[SecureStorage] 存储失败:", error);
      throw error;
    }
  },

  /**
   * 安全地从 localStorage 读取数据
   * @param key 存储键名
   * @returns 存储的数据，如果数据被篡改则返回 null
   */
  async getItem<T>(key: string): Promise<T | null> {
    try {
      const item = localStorage.getItem(key);
      if (!item) {
        return null;
      }

      const storageData = JSON.parse(item) as { data: T; hash: string; timestamp: number };

      // 验证数据完整性
      const isValid = await verifyDataIntegrity(storageData.data, storageData.hash);
      if (!isValid) {
        console.warn(`[SecureStorage] 检测到数据篡改: ${key}`);
        // 删除被篡改的数据
        localStorage.removeItem(key);
        return null;
      }

      return storageData.data;
    } catch (error) {
      console.error("[SecureStorage] 读取失败:", error);
      return null;
    }
  },

  /**
   * 删除存储的数据
   */
  removeItem(key: string): void {
    localStorage.removeItem(key);
  },

  /**
   * 清空所有安全存储
   */
  clear(): void {
    localStorage.clear();
  },
};

/**
 * XSS 防护：转义 HTML 特殊字符
 * 用于防止 XSS 攻击，特别是在显示用户输入的内容时
 */
export function escapeHtml(unsafe: unknown): string {
  if (typeof unsafe !== "string") {
    return String(unsafe);
  }

  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

/**
 * 检查字符串是否可能包含 XSS 攻击代码
 */
export function containsXSS(str: string): boolean {
  const xssPatterns = [
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    /javascript:/gi,
    /on\w+\s*=/gi, // 如 onclick=, onload= 等
    /<iframe/gi,
    /<embed/gi,
    /<object/gi,
  ];

  return xssPatterns.some((pattern) => pattern.test(str));
}

/**
 * 获取安全的 CSP（Content Security Policy）配置
 * 返回用于生产环境的 CSP 策略
 */
export function getCSPConfig(): string {
  // 根据环境变量决定是否启用严格模式
  const isDevelopment = import.meta.env.DEV;
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "";

  if (isDevelopment) {
    // 开发环境：宽松的策略，允许热更新和开发工具
    return [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'", // 开发环境需要 unsafe-eval 用于 Vite HMR
      `connect-src 'self' ${apiBaseUrl}`,
      "img-src 'self' data: blob:",
      "style-src 'self' 'unsafe-inline'", // 开发环境需要 unsafe-inline
      "font-src 'self' data:",
      "frame-ancestors 'none'",
    ].join("; ");
  } else {
    // 生产环境：严格的策略
    return [
      "default-src 'self'",
      "script-src 'self'", // 仅允许同源脚本
      `connect-src 'self' ${apiBaseUrl}`,
      "img-src 'self' data: https:",
      "style-src 'self' 'unsafe-inline'", // 大多数UI组件库需要 unsafe-inline
      "font-src 'self' data:",
      "object-src 'none'", // 禁止插件
      "base-uri 'self'",
      "form-action 'self'",
      "frame-ancestors 'none'",
      "upgrade-insecure-requests",
    ].join("; ");
  }
}

/**
 * 安全的 URL 验证
 * 检查 URL 是否是合法的同源 URL 或允许的白名单 URL
 */
export function isSecureUrl(url: string, allowedDomains: string[] = []): boolean {
  try {
    const parsedUrl = new URL(url, window.location.origin);
    const currentOrigin = window.location.origin;

    // 检查是否是同源
    if (parsedUrl.origin === currentOrigin) {
      return true;
    }

    // 检查是否在白名单中
    if (allowedDomains.some((domain) => parsedUrl.origin === domain || parsedUrl.hostname.endsWith(domain))) {
      return true;
    }

    return false;
  } catch {
    return false;
  }
}

/**
 * 清理对象中的潜在 XSS 风险字段
 * 递归地转义对象中所有字符串字段的值
 */
export function sanitizeObject<T extends Record<string, unknown>>(obj: T): T {
  const sanitized = { ...obj };

  for (const key in sanitized) {
    const value = sanitized[key];

    if (typeof value === "string") {
      // 检查是否包含 XSS
      if (containsXSS(value)) {
        console.warn(`[Security] 检测到潜在 XSS 风险字段: ${key}`);
        (sanitized as Record<string, unknown>)[key] = escapeHtml(value);
      }
    } else if (typeof value === "object" && value !== null && !Array.isArray(value)) {
      // 递归处理嵌套对象
      (sanitized as Record<string, unknown>)[key] = sanitizeObject(value as Record<string, unknown>);
    } else if (Array.isArray(value)) {
      // 处理数组
      (sanitized as Record<string, unknown>)[key] = value.map((item) => {
        if (typeof item === "string") {
          return containsXSS(item) ? escapeHtml(item) : item;
        } else if (typeof item === "object" && item !== null) {
          return sanitizeObject(item as Record<string, unknown>);
        }
        return item;
      });
    }
  }

  return sanitized;
}

/**
 * 生成安全的随机字符串
 * 用于生成 nonce 等安全相关场景
 */
export function generateSecureRandom(length: number = 16): string {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const randomValues = new Uint8Array(length);
  crypto.getRandomValues(randomValues);

  let result = "";
  for (let i = 0; i < length; i++) {
    result += chars[randomValues[i] % chars.length];
  }
  return result;
}

/**
 * 安全的日志输出（生产环境可禁用）
 */
export function secureLog(category: "info" | "warn" | "error", message: string, data?: unknown): void {
  const isProduction = import.meta.env.PROD;

  if (isProduction) {
    // 生产环境：不输出敏感信息
    if (category === "error") {
      console.error(`[${category}] ${message}`);
    }
    // 生产环境不输出 info 和 warn，也不输出 data
  } else {
    // 开发环境：完整输出
    if (data) {
      console[category](`[${category}] ${message}`, data);
    } else {
      console[category](`[${category}] ${message}`);
    }
  }
}

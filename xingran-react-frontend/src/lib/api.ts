/**
 * API 客户端（重构版 - TokenManager 集成）
 * 集成 TokenManager 实现自动 Token 刷新和 401 重试
 */

import axios from "axios";
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from "axios";
import { getAppMessage } from "@/utils/antdMessage";
import { getTokenManager } from "@/store/authStore";
import { useMenuStore } from "@/store/menuStore";
import {
  generateIV,
  generateSM4Key,
  encryptRequestBody,
  hexToBase64,
  decryptSM4CBC,
  base64ToHex,
} from "@/utils/sm4";
import { fetchPublicKey, encryptWithSM2, clearPublicKeyCache } from "@/utils/sm2";
import { handleHttpResponseError, handleNetworkError } from "@/utils/errorHandler";
import type { BaseResponse } from "@/types/base";
import { clearEncryptionConfigCache } from "@/services/encryptionConfig";
import { LOGIN } from "@/constants/routes";

// 工具函数：生成请求ID
function generateRequestId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

// 工具函数：生成随机 nonce（防重放）
function generateNonce(): string {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return Array.from(array, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

// SM4 密钥和 IV 存储（用于响应解密）
// Key: X-Request-ID, Value: { sm4KeyHex: string, ivHex: string }
const encryptionKeyStore = new Map<string, { sm4KeyHex: string; ivHex: string }>();

/**
 * 原始 axios 实例（无拦截器）
 * 专门用于获取加密配置，避免无限循环
 */
const rawAxios: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api/v1",
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

// 是否启用请求体加密（动态配置，默认禁用以避免循环依赖）
let ENABLE_REQUEST_ENCRYPTION = false;

// 加密白名单
const ENCRYPTION_WHITELIST: string[] = [];

// 加密黑名单
const ENCRYPTION_BLACKLIST: string[] = [
  "/system/auth/public-key",
  "/system/auth/captcha",
  "/system/auth/encryption-config", // 防止无限循环：加密配置端点本身不能被加密
  "/upload",
];

// Token 认证白名单（不需要 Token 的 API）
const AUTH_WHITELIST: string[] = [
  "/system/auth/login",
  "/system/auth/public-key",
  "/system/auth/captcha",
  "/system/auth/refresh", // 刷新 Token 接口本身不需要 AccessToken
];

// 需要更长超时时间的接口
const LONG_TIMEOUT_ENDPOINTS: string[] = [
  "/ad-domain/configs/",
  "/ad-domain/users/batch-sync", // 批量同步AD用户可能需要较长时间
];

/**
 * 是否正在刷新 Token
 */
let isRefreshing = false;

/**
 * 等待刷新的请求队列
 */
const refreshQueue: Array<{
  resolve: (value: any) => void;
  reject: (reason?: any) => void;
}> = [];

/**
 * 处理刷新队列
 */
function processRefreshQueue(error?: any) {
  refreshQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else {
      resolve(null);
    }
  });
  refreshQueue.length = 0;
}

/**
 * 初始化加密配置
 * 从后端获取当前加密状态并应用到 ENABLE_REQUEST_ENCRYPTION 变量
 * 注意：必须在应用启动时调用，使用 rawAxios 避免拦截器循环
 */
export async function initEncryptionConfig(): Promise<void> {
  const MAX_RETRIES = 3;
  let lastError: Error | null = null;

  for (let i = 0; i < MAX_RETRIES; i++) {
    try {
      const response = await rawAxios.get<{
        code: number;
        data: { enabled: boolean; key: string; source: string };
      }>("/system/auth/encryption-config", {
        timeout: 3000, // 3秒超时
      });

      if (response.data?.code === 0) {
        ENABLE_REQUEST_ENCRYPTION = response.data.data.enabled;
        return; // Success, exit retry loop
      }
    } catch (error) {
      lastError = error as Error;
      console.warn(`[Encryption Config] 加载失败，重试 ${i + 1}/${MAX_RETRIES}:`, error);
      await new Promise((resolve) => setTimeout(resolve, 1000 * (i + 1))); // Exponential backoff
    }
  }

  // All retries failed - use secure default
  console.error("[Encryption Config] 所有重试失败，保持安全默认值:", lastError);
  ENABLE_REQUEST_ENCRYPTION = true; // Fail secure: enable encryption
}

/**
 * 刷新加密配置
 * 供配置更新页面调用，清除缓存并重新加载配置
 * 使用 rawAxios 避免拦截器循环依赖
 * @returns 是否成功获取并应用后端最新配置；失败时保留当前设置
 */
export async function refreshEncryptionConfig(): Promise<boolean> {
  // 清除前端缓存
  clearEncryptionConfigCache();

  // 使用 rawAxios 重新获取配置（避免拦截器循环）
  try {
    const response = await rawAxios.get<{
      code: number;
      data: { enabled: boolean; key: string; source: string };
    }>("/system/auth/encryption-config", {
      timeout: 5000,
    });

    if (response.data && response.data.code === 0) {
      ENABLE_REQUEST_ENCRYPTION = response.data.data.enabled;
      return true;
    }

    console.error("[Encryption Config] 刷新配置失败：服务端返回非成功码，保持当前设置");
    return false;
  } catch (error) {
    console.error("[Encryption Config] 刷新配置失败，保持当前设置:", error);
    // 失败时不改变当前设置
    return false;
  }
}

// 创建axios实例
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api/v1",
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

// 根据接口设置超时时间
function getTimeout(url: string): number {
  if (LONG_TIMEOUT_ENDPOINTS.some((endpoint) => url.includes(endpoint))) {
    return 35 * 60 * 1000;
  }
  return 30000;
}

/**
 * 检查请求是否需要加密
 */
function shouldEncryptRequest(url: string, method: string): boolean {
  if (!ENABLE_REQUEST_ENCRYPTION) {
    return false;
  }

  if (!["POST", "PUT", "PATCH"].includes(method.toUpperCase())) {
    return false;
  }

  if (ENCRYPTION_BLACKLIST.some((prefix) => url.startsWith(prefix))) {
    return false;
  }

  if (ENCRYPTION_WHITELIST.length > 0) {
    return ENCRYPTION_WHITELIST.some((prefix) => url.startsWith(prefix));
  }

  return true;
}

// 请求拦截器
api.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    config.timeout = getTimeout(config.url || "");

    // 检查是否需要 Token（跳过白名单 API）
    const needsAuth = !AUTH_WHITELIST.some((prefix) => config.url?.startsWith(prefix));

    if (needsAuth) {
      // 使用 TokenManager 获取 Token（自动刷新）
      const tokenManager = getTokenManager();

      try {
        const token = await tokenManager.getAccessToken();

        if (token && config.headers) {
          config.headers.set("Authorization", `Bearer ${token}`);
        }
      } catch (error) {
        // Token 获取失败，可能是刷新失败
        console.error("[Request] 获取 AccessToken 失败:", error);

        // 跳过登录接口的错误处理
        if (!config.url?.includes("/system/auth/login")) {
          // 跳转到登录页
          window.location.href = LOGIN;
        }
      }
    }

    config.headers.set("X-Request-ID", generateRequestId());

    if (config.data && shouldEncryptRequest(config.url || "", config.method || "")) {
      try {
        const publicKey = await fetchPublicKey();
        const sm4KeyHex = generateSM4Key();
        const ivHex = generateIV();

        const encryptedDataHex = await encryptRequestBody(config.data, sm4KeyHex, ivHex);
        const encryptedSM4Key = await encryptWithSM2(sm4KeyHex, publicKey);

        const requestId = config.headers.get("X-Request-ID") as string;
        encryptionKeyStore.set(requestId, { sm4KeyHex, ivHex });

        config.data = {
          encrypted: true,
          data: hexToBase64(encryptedDataHex),
          sm4Key: encryptedSM4Key,
          iv: hexToBase64(ivHex),
          timestamp: Math.floor(Date.now() / 1000),
          nonce: generateNonce(),
        };

        config.headers.set("X-Request-Encrypted", "true");
      } catch (error) {
        console.error("[Request Encryption] 加密失败:", error);
        if (import.meta.env.MODE === "production") {
          return Promise.reject(error);
        }
        console.warn("[Request Encryption] 回退到明文传输（仅开发环境）");
      }
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  async (response: AxiosResponse) => {
    let { data } = response;

    const responseHeaders = response.headers || {};

    // 检查后端响应加密标志
    const isResponseEncrypted =
      responseHeaders["x-response-encrypted"] === "true" ||
      responseHeaders["X-Response-Encrypted"] === "true";

    // 检查请求体加密的响应（前端发起的加密请求，后端会加密响应）
    const isEncrypted = !!(data && data.encrypted && data.data && data.iv);

    // 如果响应被加密但没有 encrypted 标志，说明是后端中间件自动加密的
    const needsBackendDecryption = isResponseEncrypted && !data?.encrypted;

    // 处理后端中间件自动加密的响应（需要解密）
    if (needsBackendDecryption) {
      console.warn("[API Response] 检测到后端加密响应，尝试解密...");
      try {
        const requestId = responseHeaders["x-request-id"] || responseHeaders["X-Request-ID"] || "";

        if (!data?.data || !data?.iv) {
          console.error("[Response Decryption] 后端加密响应缺少必要字段");
          getAppMessage().error("响应解密失败：格式错误");
          return Promise.reject(new Error("Missing encrypted data fields"));
        }

        const encryptedDataBase64 = data.data;
        const ivBase64 = data.iv;

        // 使用请求时存储的密钥解密
        const keyInfo = encryptionKeyStore.get(requestId);
        if (!keyInfo) {
          console.error("[Response Decryption] 找不到请求的加密密钥:", requestId);
          getAppMessage().error("响应解密失败：找不到加密密钥");
          return Promise.reject(new Error("Encryption keys not found"));
        }

        const encryptedDataHex = base64ToHex(encryptedDataBase64);
        const ivHex = base64ToHex(ivBase64);

        const decryptedJson = await decryptSM4CBC(encryptedDataHex, keyInfo.sm4KeyHex, ivHex);
        data = JSON.parse(decryptedJson);

        encryptionKeyStore.delete(requestId);
      } catch (error) {
        console.error("[Response Decryption] 后端加密响应解密失败:", error);
        getAppMessage().error("响应解密失败: " + (error as Error).message);
        return Promise.reject(error);
      }
    }

    // 处理前端发起加密请求的响应（原有逻辑）
    if (isEncrypted) {
      try {
        const requestId = response.config.headers.get("X-Request-ID") as string;

        if (!requestId) {
          console.error("[Response Decryption] 响应头中缺少请求 ID");
          getAppMessage().error("响应解密失败：缺少请求ID");
          return Promise.reject(new Error("Missing request ID"));
        }

        const keyInfo = encryptionKeyStore.get(requestId);
        if (!keyInfo) {
          console.error("[Response Decryption] 找不到请求的加密密钥:", requestId);
          getAppMessage().error("响应解密失败：找不到加密密钥");
          return Promise.reject(new Error("Encryption keys not found"));
        }

        const encryptedDataBase64 = data.data;
        const ivBase64 = data.iv;

        const encryptedDataHex = base64ToHex(encryptedDataBase64);
        const ivHex = base64ToHex(ivBase64);

        const decryptedJson = await decryptSM4CBC(encryptedDataHex, keyInfo.sm4KeyHex, ivHex);

        data = JSON.parse(decryptedJson);

        encryptionKeyStore.delete(requestId);
      } catch (error) {
        console.error("[Response Decryption] 解密失败:", error);
        getAppMessage().error("响应解密失败: " + (error as Error).message);
        return Promise.reject(error);
      }
    }

    if (data && typeof data === "object" && data.code === 0) {
      return data;
    } else if (data && typeof data === "object" && data.code !== undefined) {
      getAppMessage().error(data.message || "请求失败");
      return Promise.reject(new Error(data.message || "请求失败"));
    } else {
      console.error("[Response] 响应格式无效:", {
        url: response.config.url,
        method: response.config.method,
        dataType: typeof data,
        dataKeys: typeof data === "object" ? Object.keys(data) : "N/A",
        data,
      });
      getAppMessage().error("响应格式错误");
      return Promise.reject(new Error("Invalid response format"));
    }
  },
  async (error) => {
    const { response, config } = error;

    // 处理 401 未授权错误
    if (response?.status === 401 && config) {
      // 【登录请求短路】登录接口的 401 是凭据错误（用户名/密码错误），
      // 绝不能进入下面的 token 刷新→失败→跳转链路，否则会丢失后端原始错误信息
      // 并把用户"自动刷新"回登录页，看不到任何提示。
      // 此处提取后端 message 原样 reject，由登录页 catch 显示内联提示。
      if (config.url?.includes("/system/auth/login")) {
        const respData = response?.data;
        const backendMessage = (respData &&
          typeof respData === "object" &&
          (respData.message || respData.msg)) as string | undefined;
        const loginError = new Error(backendMessage || "用户名或密码错误");
        return Promise.reject(loginError);
      }

      // 清空菜单缓存
      useMenuStore.getState().clearMenus();

      // 如果是刷新 token 请求失败，直接登出
      if (config.url?.includes("/system/auth/refresh")) {
        const tokenManager = getTokenManager();
        await tokenManager.clearTokens();
        window.location.href = LOGIN;
        return Promise.reject(error);
      }

      // 如果已经在刷新，加入队列
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          refreshQueue.push({ resolve, reject });
        })
          .then(() => {
            // 刷新成功，重试原请求
            return api(config);
          })
          .catch((err) => {
            return Promise.reject(err);
          });
      }

      // 开始刷新
      isRefreshing = true;

      const tokenManager = getTokenManager();

      try {
        // 尝试刷新 Token
        await tokenManager.refreshToken();

        // 刷新成功，处理队列
        processRefreshQueue();

        // 重试原请求
        return api(config);
      } catch (refreshError) {
        // 刷新失败，清除所有状态
        await tokenManager.clearTokens();
        clearPublicKeyCache();

        // 处理队列（全部失败）
        processRefreshQueue(refreshError);

        // 跳转到登录页
        window.location.href = LOGIN;

        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    // 其他错误处理
    if (response) {
      const apiError = handleHttpResponseError(response.status, response.data);
      return Promise.reject(apiError);
    } else if (error.code === "ECONNABORTED") {
      const networkError = handleNetworkError(error);
      return Promise.reject(networkError);
    } else {
      const networkError = handleNetworkError(error);
      return Promise.reject(networkError);
    }
  }
);

// 封装常用请求方法
// 注意：这些方法返回 Promise<BaseResponse<T>>，其中 T 是响应中 data 字段的类型
export function get<T = unknown>(url: string, params?: unknown): Promise<BaseResponse<T>> {
  return api.get(url, { params });
}

export function post<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
  return api.post(url, data);
}

export function put<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
  return api.put(url, data);
}

export function del<T = unknown>(url: string): Promise<BaseResponse<T>> {
  return api.delete(url);
}

export function upload<T = unknown>(url: string, file: File): Promise<BaseResponse<T>> {
  const formData = new FormData();
  formData.append("file", file);

  return api.post(url, formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

export function postFormData<T = unknown>(
  url: string,
  formData: FormData
): Promise<BaseResponse<T>> {
  return api.post(url, formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
}

export function postLongRequest<T = unknown>(
  url: string,
  data?: unknown,
  timeout: number = 300000
): Promise<T> {
  return api.post(url, data, { timeout });
}

// 类型化 API 函数（别名，与上方函数相同，保留向后兼容）
// 推荐直接使用 get/post/put/del，它们已返回 BaseResponse<T>

export const getTyped = get;
export const postTyped = post;
export const putTyped = put;
export const patchTyped = <T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> =>
  api.patch(url, data);
export const deleteTyped = del;

// 导出axios实例
export { api };

export default api;

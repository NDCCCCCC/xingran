import axios from 'axios';
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';
import { useAuthStore } from '@/store/authStore';
import { generateIV, generateSM4Key, encryptRequestBody, hexToBase64, decryptSM4CBC, base64ToHex } from '@/utils/sm4';
import { fetchPublicKey, encryptWithSM2, clearPublicKeyCache } from '@/utils/sm2';
import {
  handleHttpResponseError,
  handleNetworkError,
  handleParseError,
} from '@/utils/errorHandler';

// SM4 密钥和 IV 存储（用于响应解密）
// Key: X-Request-ID, Value: { sm4KeyHex: string, ivHex: string }
const encryptionKeyStore = new Map<string, { sm4KeyHex: string; ivHex: string }>();

// 是否启用请求体加密
const ENABLE_REQUEST_ENCRYPTION = import.meta.env.VITE_ENABLE_REQUEST_ENCRYPTION === 'true';

// 加密白名单
const ENCRYPTION_WHITELIST: string[] = [];

// 加密黑名单
const ENCRYPTION_BLACKLIST: string[] = [
  '/system/auth/login',
  '/system/auth/public-key',
  '/system/auth/captcha',
  '/upload',
];

// 创建axios实例
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1',
  timeout: 30000, // 默认30秒
  headers: {
    'Content-Type': 'application/json',
  },
});

// 需要更长超时时间的接口
const LONG_TIMEOUT_ENDPOINTS: string[] = [
  '/ad-domain/configs/',
];

// 根据接口设置超时时间
function getTimeout(url: string): number {
  for (const endpoint of LONG_TIMEOUT_ENDPOINTS) {
    if (url.includes(endpoint)) {
      return 35 * 60 * 1000;
    }
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

  if (!['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
    return false;
  }

  for (const prefix of ENCRYPTION_BLACKLIST) {
    if (url.startsWith(prefix)) {
      return false;
    }
  }

  if (ENCRYPTION_WHITELIST.length > 0) {
    for (const prefix of ENCRYPTION_WHITELIST) {
      if (url.startsWith(prefix)) {
        return true;
      }
    }
    return false;
  }

  return true;
}

// 请求拦截器
api.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    config.timeout = getTimeout(config.url || '');

    const token = localStorage.getItem('accessToken');
    if (token && config.headers) {
      config.headers.set('Authorization', `Bearer ${token}`);
    }

    config.headers.set('X-Request-ID', generateRequestId());

    if (config.data && shouldEncryptRequest(config.url || '', config.method || '')) {
      try {
        const publicKey = await fetchPublicKey();
        const sm4KeyHex = generateSM4Key();
        const ivHex = generateIV();

        const encryptedDataHex = await encryptRequestBody(config.data, sm4KeyHex, ivHex);
        const encryptedSM4Key = await encryptWithSM2(sm4KeyHex, publicKey);

        const requestId = config.headers.get('X-Request-ID') as string;
        encryptionKeyStore.set(requestId, { sm4KeyHex, ivHex });

        config.data = {
          encrypted: true,
          data: hexToBase64(encryptedDataHex),
          sm4Key: encryptedSM4Key,
          iv: hexToBase64(ivHex),
          timestamp: Math.floor(Date.now() / 1000),
          nonce: generateNonce(),
        };

        config.headers.set('X-Request-Encrypted', 'true');
      } catch (error) {
        console.error('[Request Encryption] Encryption failed:', error);
        if (import.meta.env.MODE === 'production') {
          return Promise.reject(error);
        }
        console.warn('[Request Encryption] Falling back to plaintext (development only)');
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

    const isEncrypted = !!(data && data.encrypted && data.data && data.iv);

    if (isEncrypted) {
      try {
        const requestId = response.config.headers.get('X-Request-ID') as string;

        if (!requestId) {
          console.error('[Response Decryption] No request ID in response headers');
          message.error('响应解密失败：缺少请求ID');
          return Promise.reject(new Error('Missing request ID'));
        }

        const keyInfo = encryptionKeyStore.get(requestId);
        if (!keyInfo) {
          console.error('[Response Decryption] No encryption keys for request:', requestId);
          message.error('响应解密失败：找不到加密密钥');
          return Promise.reject(new Error('Encryption keys not found'));
        }

        const encryptedDataBase64 = data.data;
        const ivBase64 = data.iv;

        const encryptedDataHex = base64ToHex(encryptedDataBase64);
        const ivHex = base64ToHex(ivBase64);

        const decryptedJson = await decryptSM4CBC(encryptedDataHex, keyInfo.sm4KeyHex, ivHex);

        data = JSON.parse(decryptedJson);

        encryptionKeyStore.delete(requestId);
      } catch (error) {
        console.error('[Response Decryption] Decryption failed:', error);
        message.error('响应解密失败: ' + (error as Error).message);
        return Promise.reject(error);
      }
    }

    if (data && typeof data === 'object' && data.code === 0) {
      return data;
    } else if (data && typeof data === 'object' && data.code !== undefined) {
      message.error(data.message || '请求失败');
      return Promise.reject(new Error(data.message || '请求失败'));
    } else {
      console.error('[Response] Invalid response format:', {
        url: response.config.url,
        method: response.config.method,
        dataType: typeof data,
        dataKeys: typeof data === 'object' ? Object.keys(data) : 'N/A',
        data,
      });
      message.error('响应格式错误');
      return Promise.reject(new Error('Invalid response format'));
    }
  },
  async (error) => {
    const { response } = error;

    if (response) {
      // 使用统一的 HTTP 响应错误处理
      const apiError = handleHttpResponseError(response.status, response.data);
      return Promise.reject(apiError);
    } else if (error.code === 'ECONNABORTED') {
      // 使用统一的网络错误处理
      const networkError = handleNetworkError(error);
      return Promise.reject(networkError);
    } else {
      // 其他网络异常
      const networkError = handleNetworkError(error);
      return Promise.reject(networkError);
    }
  }
);

// 生成请求ID
function generateRequestId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

// 生成随机 nonce（防重放）
function generateNonce(): string {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
}

// 封装GET请求
export function get<T = unknown>(url: string, params?: unknown): Promise<T> {
  return api.get(url, { params });
}

// 封装POST请求
export function post<T = unknown>(url: string, data?: unknown): Promise<T> {
  return api.post(url, data);
}

// 封装PUT请求
export function put<T = unknown>(url: string, data?: unknown): Promise<T> {
  return api.put(url, data);
}

// 封装DELETE请求
export function del<T = unknown>(url: string): Promise<T> {
  return api.delete(url);
}

// 封装文件上传
export function upload<T = unknown>(url: string, file: File): Promise<T> {
  const formData = new FormData();
  formData.append('file', file);

  return api.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}

// 封装FormData POST请求
export function postFormData<T = unknown>(url: string, formData: FormData): Promise<T> {
  return api.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}

// 封装长时间运行请求
export function postLongRequest<T = unknown>(url: string, data?: unknown, timeout: number = 300000): Promise<T> {
  return api.post(url, data, { timeout });
}

// 导出axios实例
export { api };

export default api;
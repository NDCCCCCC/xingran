import axios from 'axios';
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';
import { useAuthStore } from '@/store/authStore';
import { generateIV, generateSM4Key, encryptRequestBody, hexToBase64, decryptSM4CBC, base64ToHex } from '@/utils/sm4';
import { fetchPublicKey, encryptWithSM2, clearPublicKeyCache } from '@/utils/sm2';

// SM4 密钥和 IV 存储（用于响应解密）
// Key: X-Request-ID, Value: { sm4KeyHex: string, ivHex: string }
const encryptionKeyStore = new Map<string, { sm4KeyHex: string; ivHex: string }>();

// 是否启用请求体加密（从环境变量读取）
const ENABLE_REQUEST_ENCRYPTION = import.meta.env.VITE_ENABLE_REQUEST_ENCRYPTION === 'true';

// 需要加密的路径（白名单）- 如果配置了白名单，则只加密白名单内的接口
const ENCRYPTION_WHITELIST: string[] = [
  // 示例: '/api/v1/network/credentials',
  //       '/api/v1/system/configs',
];

// 不需要加密的路径（黑名单，优先级高于白名单）
// 注意：由于 baseURL 是 /api/v1，这里使用相对路径
const ENCRYPTION_BLACKLIST: string[] = [
  '/system/auth/login',
  '/system/auth/public-key',
  '/system/auth/captcha',  // 验证码相关接口不加密
  '/upload',
];

// 创建axios实例
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * 检查请求是否需要加密
 * @param url 请求URL
 * @param method 请求方法
 * @returns 是否需要加密
 */
function shouldEncryptRequest(url: string, method: string): boolean {
  // 全局开关
  if (!ENABLE_REQUEST_ENCRYPTION) {
    return false;
  }

  // 只加密 POST/PUT/PATCH
  if (!['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
    return false;
  }

  // 检查黑名单
  for (const prefix of ENCRYPTION_BLACKLIST) {
    if (url.startsWith(prefix)) {
      return false;
    }
  }

  // 检查白名单（如果配置了白名单，则只加密白名单内的接口）
  if (ENCRYPTION_WHITELIST.length > 0) {
    for (const prefix of ENCRYPTION_WHITELIST) {
      if (url.startsWith(prefix)) {
        return true;
      }
    }
    return false;
  }

  // 默认加密所有 POST/PUT/PATCH 请求
  return true;
}

// 请求拦截器
api.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    // 从localStorage获取token
    const token = localStorage.getItem('accessToken');
    if (token && config.headers) {
      config.headers.set('Authorization', `Bearer ${token}`);
    }

    // 添加请求ID
    config.headers.set('X-Request-ID', generateRequestId());

    // 请求体加密
    if (config.data && shouldEncryptRequest(config.url || '', config.method || '')) {
      try {
        // 1. 获取 SM2 公钥（用于加密 SM4 密钥）
        const publicKey = await fetchPublicKey();

        // 2. 生成随机 SM4 密钥和 IV
        // 生成 32 字符的十六进制字符串（代表 16 字节）
        const sm4KeyHex = generateSM4Key(); // 32字符十六进制
        const ivHex = generateIV();          // 32字符十六进制，用于 SM4-CBC

        // 3. 使用 SM4-CBC 加密请求体（异步）
        const encryptedDataHex = await encryptRequestBody(config.data, sm4KeyHex, ivHex);

        // 4. 使用 SM2 公钥加密 SM4 密钥的十六进制字符串
        // 后端会自动检测十六进制格式并解码为 16 字节再加密
        const encryptedSM4Key = await encryptWithSM2(sm4KeyHex, publicKey);

        // 5. 存储密钥信息（用于响应解密）
        const requestId = config.headers.get('X-Request-ID') as string;
        encryptionKeyStore.set(requestId, { sm4KeyHex, ivHex });

        // 6. 组装加密请求体
        config.data = {
          encrypted: true,
          data: hexToBase64(encryptedDataHex), // SM4-CBC 加密的数据 (Base64)
          sm4Key: encryptedSM4Key,             // SM2 加密后的 SM4 密钥 (Base64)
          iv: hexToBase64(ivHex),              // SM4-CBC 的 IV (Base64)
          timestamp: Math.floor(Date.now() / 1000), // 转换为秒级时间戳
          nonce: generateNonce(),                       // 随机数（防重放）
        };

        // 添加加密标识头
        config.headers.set('X-Request-Encrypted', 'true');
      } catch (error) {
        console.error('[Request Encryption] Encryption failed:', error);
        // 回退: 如果加密失败，继续使用明文（仅开发环境）
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

    // 检查响应是否加密（只需要 encrypted: true 和 data, iv 字段）
    const isEncrypted = data && data.encrypted === true && data.data && data.iv;

    if (isEncrypted) {
      try {
        // 获取请求 ID
        const requestId = response.config.headers.get('X-Request-ID') as string;

        if (!requestId) {
          console.error('[Response Decryption] No request ID in response headers');
          message.error('响应解密失败：缺少请求ID');
          return Promise.reject(new Error('Missing request ID'));
        }

        // 获取存储的密钥信息
        const keyInfo = encryptionKeyStore.get(requestId);
        if (!keyInfo) {
          console.error('[Response Decryption] No encryption keys for request:', requestId);
          message.error('响应解密失败：找不到加密密钥');
          return Promise.reject(new Error('Encryption keys not found'));
        }

        // 解密响应数据
        const encryptedDataBase64 = data.data;
        const ivBase64 = data.iv;

        // Base64 转十六进制
        const encryptedDataHex = base64ToHex(encryptedDataBase64);
        const ivHex = base64ToHex(ivBase64);

        // SM4-CBC 解密
        const decryptedJson = await decryptSM4CBC(encryptedDataHex, keyInfo.sm4KeyHex, ivHex);

        // 解析 JSON
        data = JSON.parse(decryptedJson);

        // 清理密钥存储
        encryptionKeyStore.delete(requestId);
      } catch (error) {
        console.error('[Response Decryption] Decryption failed:', error);
        message.error('响应解密失败: ' + (error as Error).message);
        return Promise.reject(error);
      }
    }

    // 检查业务状态码 - 根据后端实际返回调整
    if (data && typeof data === 'object' && data.code === 0) {
      return data;
    } else if (data && typeof data === 'object' && data.code !== undefined) {
      // 业务错误
      message.error(data.message || '请求失败');
      return Promise.reject(new Error(data.message || '请求失败'));
    } else {
      // 没有 code 字段，可能是加密响应但解密失败
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
      const { status, data } = response;

      // 尝试从后端响应中获取错误消息
      let errorMessage = '请求失败';
      if (data?.message) {
        errorMessage = data.message;
      }

      switch (status) {
        case 400:
          // 如果是 SM2 解密失败，清除公钥缓存以便下次获取新密钥
          if (errorMessage.includes('SM2') || errorMessage.includes('解密')) {
            console.warn('[API] SM2 解密失败，清除公钥缓存');
            clearPublicKeyCache();
          }
          message.error(errorMessage);
          break;
        case 401:
          // 未授权，清除token并跳转到登录页
          localStorage.removeItem('accessToken');
          localStorage.removeItem('refreshToken');
          // 清除 Zustand store 中的认证状态
          useAuthStore.getState().logout();
          window.location.href = '/login';
          break;
        case 403:
          message.error(errorMessage);
          break;
        case 404:
          message.error(errorMessage);
          break;
        case 500:
          message.error(errorMessage);
          break;
        default:
          message.error(errorMessage || '网络错误，请稍后重试');
      }
    } else if (error.code === 'ECONNABORTED') {
      message.error('请求超时，请检查网络连接');
    } else {
      message.error('网络异常，请检查网络连接');
    }

    return Promise.reject(error);
  }
);

// 生成请求ID
function generateRequestId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

// 生成随机 nonce（防重放）
function generateNonce(): string {
  // 生成 16 字节（32 个十六进制字符）的随机数
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
}

// 封装GET请求
export function get<T = any>(url: string, params?: any): Promise<T> {
  return api.get(url, { params });
}

// 封装POST请求
export function post<T = any>(url: string, data?: any): Promise<T> {
  return api.post(url, data);
}

// 封装PUT请求
export function put<T = any>(url: string, data?: any): Promise<T> {
  return api.put(url, data);
}

// 封装DELETE请求
export function del<T = any>(url: string): Promise<T> {
  return api.delete(url);
}

// 封装文件上传
export function upload<T = any>(url: string, file: File): Promise<T> {
  const formData = new FormData();
  formData.append('file', file);

  return api.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}

// 封装FormData POST请求（用于多字段文件上传）
export function postFormData<T = any>(url: string, formData: FormData): Promise<T> {
  return api.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}

// 封装长时间运行请求（用于任务执行等耗时操作）
export function postLongRequest<T = any>(url: string, data?: any, timeout: number = 300000): Promise<T> {
  return api.post(url, data, { timeout });
}

// 导出axios实例以供自定义使用
export { api };

export default api;
/**
 * API 客户端（重构版 - TokenManager 集成）
 * 集成 TokenManager 实现自动 Token 刷新和 401 重试
 */

import axios from 'axios';
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';
import { getTokenManager } from '@/store/authStore';
import { useMenuStore } from '@/store/menuStore';
import { generateIV, generateSM4Key, encryptRequestBody, hexToBase64, decryptSM4CBC, base64ToHex } from '@/utils/sm4';
import { fetchPublicKey, encryptWithSM2, clearPublicKeyCache } from '@/utils/sm2';
import {
	handleHttpResponseError,
	handleNetworkError,
	handleParseError,
} from '@/utils/errorHandler';
import type { BaseResponse } from '@/types/base';

// 工具函数：生成请求ID
function generateRequestId(): string {
	return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

// 工具函数：生成随机 nonce（防重放）
function generateNonce(): string {
	const array = new Uint8Array(16);
	crypto.getRandomValues(array);
	return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
}

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

// Token 认证白名单（不需要 Token 的 API）
const AUTH_WHITELIST: string[] = [
	'/system/auth/login',
	'/system/auth/public-key',
	'/system/auth/captcha',
	'/system/auth/refresh', // 刷新 Token 接口本身不需要 AccessToken
];

// 需要更长超时时间的接口
const LONG_TIMEOUT_ENDPOINTS: string[] = [
	'/ad-domain/configs/',
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

// 创建axios实例
const api: AxiosInstance = axios.create({
	baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1',
	timeout: 30000,
	headers: {
		'Content-Type': 'application/json',
	},
});

// 根据接口设置超时时间
function getTimeout(url: string): number {
	if (LONG_TIMEOUT_ENDPOINTS.some(endpoint => url.includes(endpoint))) {
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

	if (!['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
		return false;
	}

	if (ENCRYPTION_BLACKLIST.some(prefix => url.startsWith(prefix))) {
		return false;
	}

	if (ENCRYPTION_WHITELIST.length > 0) {
		return ENCRYPTION_WHITELIST.some(prefix => url.startsWith(prefix));
	}

	return true;
}

// 请求拦截器
api.interceptors.request.use(
	async (config: InternalAxiosRequestConfig) => {
		config.timeout = getTimeout(config.url || '');

		// 检查是否需要 Token（跳过白名单 API）
		const needsAuth = !AUTH_WHITELIST.some(prefix => config.url?.startsWith(prefix));

		if (needsAuth) {
			// 使用 TokenManager 获取 Token（自动刷新）
			const tokenManager = getTokenManager();

			try {
				const token = await tokenManager.getAccessToken();

				if (token && config.headers) {
					config.headers.set('Authorization', `Bearer ${token}`);
				}
			} catch (error) {
				// Token 获取失败，可能是刷新失败
				console.error('[Request] 获取 AccessToken 失败:', error);

				// 跳过登录接口的错误处理
				if (!config.url?.includes('/system/auth/login')) {
					// 跳转到登录页
					window.location.href = '/login';
				}
			}
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
				console.error('[Request Encryption] 加密失败:', error);
				if (import.meta.env.MODE === 'production') {
					return Promise.reject(error);
				}
				console.warn('[Request Encryption] 回退到明文传输（仅开发环境）');
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
					console.error('[Response Decryption] 响应头中缺少请求 ID');
					message.error('响应解密失败：缺少请求ID');
					return Promise.reject(new Error('Missing request ID'));
				}

				const keyInfo = encryptionKeyStore.get(requestId);
				if (!keyInfo) {
					console.error('[Response Decryption] 找不到请求的加密密钥:', requestId);
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
				console.error('[Response Decryption] 解密失败:', error);
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
			console.error('[Response] 响应格式无效:', {
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
		const { response, config } = error;

		// 处理 401 未授权错误
		if (response?.status === 401 && config) {
			// 清空菜单缓存
			useMenuStore.getState().clearMenus();

			// 如果是刷新 token 请求失败，直接登出
			if (config.url?.includes('/system/auth/refresh')) {
				const tokenManager = getTokenManager();
				await tokenManager.clearTokens();
				window.location.href = '/login';
				return Promise.reject(error);
			}

			// 如果已经在刷新，加入队列
			if (isRefreshing) {
				return new Promise((resolve, reject) => {
					refreshQueue.push({ resolve, reject });
				}).then(() => {
					// 刷新成功，重试原请求
					return api(config);
				}).catch((err) => {
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
				window.location.href = '/login';

				return Promise.reject(refreshError);
			} finally {
				isRefreshing = false;
			}
		}

		// 其他错误处理
		if (response) {
			const apiError = handleHttpResponseError(response.status, response.data);
			return Promise.reject(apiError);
		} else if (error.code === 'ECONNABORTED') {
			const networkError = handleNetworkError(error);
			return Promise.reject(networkError);
		} else {
			const networkError = handleNetworkError(error);
			return Promise.reject(networkError);
		}
	}
);

// 封装常用请求方法
export function get<T = unknown>(url: string, params?: unknown): Promise<T> {
	return api.get(url, { params });
}

export function post<T = unknown>(url: string, data?: unknown): Promise<T> {
	return api.post(url, data);
}

export function put<T = unknown>(url: string, data?: unknown): Promise<T> {
	return api.put(url, data);
}

export function del<T = unknown>(url: string): Promise<T> {
	return api.delete(url);
}

export function upload<T = unknown>(url: string, file: File): Promise<T> {
	const formData = new FormData();
	formData.append('file', file);

	return api.post(url, formData, {
		headers: {
			'Content-Type': 'multipart/form-data',
		},
	});
}

export function postFormData<T = unknown>(url: string, formData: FormData): Promise<T> {
	return api.post(url, formData, {
		headers: {
			'Content-Type': 'multipart/form-data',
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

// 类型化 API 函数

export function getTyped<T>(url: string, params?: unknown): Promise<BaseResponse<T>> {
	return api.get(url, { params });
}

export function postTyped<T>(url: string, data?: unknown): Promise<BaseResponse<T>> {
	return api.post(url, data);
}

export function putTyped<T>(url: string, data?: unknown): Promise<BaseResponse<T>> {
	return api.put(url, data);
}

export function patchTyped<T>(url: string, data?: unknown): Promise<BaseResponse<T>> {
	return api.patch(url, data);
}

export function deleteTyped<T>(url: string): Promise<BaseResponse<T>> {
	return api.delete(url);
}

// 导出axios实例
export { api };

export default api;

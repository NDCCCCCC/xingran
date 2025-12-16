import axios from 'axios';
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';

// 创建axios实例
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:9000/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config: AxiosRequestConfig) => {
    // 从localStorage获取token
    const token = localStorage.getItem('accessToken');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // 添加请求ID
    config.headers = {
      ...config.headers,
      'X-Request-ID': generateRequestId(),
    };

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response: AxiosResponse) => {
    const { data } = response;

    // 检查业务状态码
    if (data.code === 0) {
      return data;
    } else {
      // 业务错误
      message.error(data.message || '请求失败');
      return Promise.reject(new Error(data.message || '请求失败'));
    }
  },
  async (error) => {
    const { response } = error;

    if (response) {
      const { status } = response;

      switch (status) {
        case 401:
          // 未授权，清除token并跳转到登录页
          localStorage.removeItem('accessToken');
          localStorage.removeItem('refreshToken');
          window.location.href = '/login';
          break;
        case 403:
          message.error('没有权限访问');
          break;
        case 404:
          message.error('请求的资源不存在');
          break;
        case 500:
          message.error('服务器内部错误');
          break;
        default:
          message.error('网络错误，请稍后重试');
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

export default api;
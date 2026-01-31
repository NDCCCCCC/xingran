import { message } from 'antd';
import { useAuthStore } from '@/store/authStore';
import { clearPublicKeyCache } from './sm2';

export interface ErrorResponse {
  response?: {
    data?: {
      message?: string;
      msg?: string;
      error?: string;
    };
  };
  message?: string;
  msg?: string;
  error?: string;
}

/**
 * HTTP 状态码错误类型
 */
export enum HttpErrorType {
  BAD_REQUEST = 400,
  UNAUTHORIZED = 401,
  FORBIDDEN = 403,
  NOT_FOUND = 404,
  METHOD_NOT_ALLOWED = 405,
  CONFLICT = 409,
  INTERNAL_ERROR = 500,
  NOT_IMPLEMENTED = 501,
  BAD_GATEWAY = 502,
  SERVICE_UNAVAILABLE = 503,
}

/**
 * API 响应数据接口
 */
export interface ApiResponseData {
  code?: number;
  message?: string;
  timestamp?: number;
  request_id?: string;
}

/**
 * 统一错误处理工具
 * 提供一致的错误提示和日志记录
 */
export class ErrorHandler {
  /**
   * 处理API错误
   * @param error 错误对象
   * @param context 错误上下文（操作名称）
   * @param showMessage 是否显示错误提示
   */
  static handleApiError(error: unknown, context: string, showMessage: boolean = true): void {
    // 获取错误消息
    let errorMessage = '操作失败';

    // 类型守卫，检查是否为 ErrorResponse 类型
    if (error && typeof error === 'object') {
      const err = error as ErrorResponse;
      if (err?.response?.data?.message) {
        errorMessage = err.response.data.message;
      } else if (err?.response?.data?.msg) {
        errorMessage = err.response.data.msg;
      } else if (err?.response?.data?.error) {
        errorMessage = err.response.data.error;
      } else if (err?.message) {
        errorMessage = err.message;
      } else if (err?.msg) {
        errorMessage = err.msg;
      } else if (err?.error) {
        errorMessage = err.error;
      }
    } else if (error && typeof error === 'string') {
      errorMessage = error;
    }

    // 控制台日志
    console.error(`[${context}]`, error);

    // 显示错误提示
    if (showMessage) {
      message.error(`${context}失败: ${errorMessage}`);
    }
  }

  /**
   * 处理成功消息
   * @param action 操作名称
   * @param isEdit 是否是编辑操作
   */
  static handleSuccess(action: string, isEdit: boolean = false): void {
    message.success(isEdit ? `更新${action}成功` : `${action}成功`);
  }

  /**
   * 创建操作结果处理器
   * @param action 操作名称
   */
  static createResultHandler(action: string) {
    return {
      success: (isEdit: boolean = false) => {
        this.handleSuccess(action, isEdit);
      },
      error: (error: unknown) => {
        this.handleApiError(error, action);
      },
    };
  }
}

/**
 * 便捷的错误处理函数
 */
export function handleApiError(error: unknown, context: string, showMessage: boolean = true): void {
  ErrorHandler.handleApiError(error, context, showMessage);
}

/**
 * 便捷的成功处理函数
 */
export function handleSuccess(action: string, isEdit: boolean = false): void {
  ErrorHandler.handleSuccess(action, isEdit);
}

/**
 * 异步操作包装器
 * 自动处理错误和成功消息
 */
export async function withErrorHandling<T>(
  operation: () => Promise<T>,
  options: {
    successMessage?: string;
    errorMessage?: string;
    onSuccess?: (result: T) => void;
    onError?: (error: unknown) => void;
  } = {}
): Promise<T | null> {
  try {
    const result = await operation();
    if (options.successMessage) {
      message.success(options.successMessage);
    }
    options.onSuccess?.(result);
    return result;
  } catch (error) {
    if (options.errorMessage) {
      ErrorHandler.handleApiError(error, options.errorMessage);
    }
    options.onError?.(error);
    return null;
  }
}

// ==================== HTTP 响应级别错误处理 ====================

/**
 * 默认错误消息映射
 */
const DEFAULT_ERROR_MESSAGES: Record<HttpErrorType, string> = {
  [HttpErrorType.BAD_REQUEST]: '请求参数错误',
  [HttpErrorType.UNAUTHORIZED]: '登录已过期，请重新登录',
  [HttpErrorType.FORBIDDEN]: '没有权限访问',
  [HttpErrorType.NOT_FOUND]: '请求的资源不存在',
  [HttpErrorType.METHOD_NOT_ALLOWED]: '请求方法不允许',
  [HttpErrorType.CONFLICT]: '数据冲突，请刷新后重试',
  [HttpErrorType.INTERNAL_ERROR]: '服务器内部错误',
  [HttpErrorType.NOT_IMPLEMENTED]: '功能未实现',
  [HttpErrorType.BAD_GATEWAY]: '网关错误',
  [HttpErrorType.SERVICE_UNAVAILABLE]: '服务暂时不可用',
};

/**
 * 处理 HTTP 响应错误
 * @param status HTTP 状态码
 * @param responseData 响应数据
 * @returns Error 对象
 */
export function handleHttpResponseError(
  status: number,
  responseData?: ApiResponseData
): Error {
  const errorType = getErrorTypeByStatus(status);
  const errorMessage = responseData?.message || DEFAULT_ERROR_MESSAGES[errorType];

  // 特殊处理 400 错误中的 SM2 解密失败
  if (status === 400 && errorMessage.includes('SM2') || errorMessage.includes('解密')) {
    console.warn('[ErrorHandler] SM2 解密失败，清除公钥缓存');
    clearPublicKeyCache();
  }

  // 特殊处理 401 未授权 - 自动登出
  if (status === 401) {
    handleUnauthorized();
  }

  // 显示错误消息
  message.error(errorMessage);

  const error = new Error(errorMessage);
  (error as any).status = status;
  (error as any).type = errorType;
  return error;
}

/**
 * 处理网络错误
 * @param error 原始错误对象
 * @returns Error 对象
 */
export function handleNetworkError(error: any): Error {
  let errorMessage: string;

  if (error.code === 'ECONNABORTED') {
    errorMessage = '请求超时，请检查网络连接';
  } else {
    errorMessage = '网络异常，请检查网络连接';
  }

  message.error(errorMessage);

  const networkError = new Error(errorMessage);
  (networkError as any).originalError = error;
  return networkError;
}

/**
 * 处理响应解析错误
 * @param error 原始错误对象
 * @returns Error 对象
 */
export function handleParseError(error: any): Error {
  const errorMessage = '响应解析失败';

  message.error(errorMessage);
  console.error('[ErrorHandler] Response parse error:', error);

  const parseError = new Error(errorMessage);
  (parseError as any).originalError = error;
  return parseError;
}

/**
 * 根据状态码获取错误类型
 */
function getErrorTypeByStatus(status: number): HttpErrorType {
  const statusToTypeMap: Record<number, HttpErrorType> = {
    400: HttpErrorType.BAD_REQUEST,
    401: HttpErrorType.UNAUTHORIZED,
    403: HttpErrorType.FORBIDDEN,
    404: HttpErrorType.NOT_FOUND,
    405: HttpErrorType.METHOD_NOT_ALLOWED,
    409: HttpErrorType.CONFLICT,
    500: HttpErrorType.INTERNAL_ERROR,
    501: HttpErrorType.NOT_IMPLEMENTED,
    502: HttpErrorType.BAD_GATEWAY,
    503: HttpErrorType.SERVICE_UNAVAILABLE,
  };

  return statusToTypeMap[status] || HttpErrorType.INTERNAL_ERROR;
}

/**
 * 处理未授权错误（登出逻辑）
 */
function handleUnauthorized(): void {
  // 清除本地存储的 token
  localStorage.removeItem('accessToken');
  localStorage.removeItem('refreshToken');

  // 清除 auth store 状态
  useAuthStore.getState().logout();

  // 跳转到登录页
  window.location.href = '/login';
}

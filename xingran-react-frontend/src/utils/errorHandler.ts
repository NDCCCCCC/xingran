import { getAppMessage } from "@/utils/antdMessage";
import { useAuthStore } from "@/store/authStore";
import { clearPublicKeyCache } from "./sm2";
import { LOGIN } from "@/constants/routes";

// ============================================================================
// 向后兼容性重新导出
// ============================================================================
// 以下导出仅用于保持向后兼容，支持现有代码从 @/utils/errorHandler 导入。
//
// ⚠️ 新代码请直接从定义位置导入：
//   import { isFormValidationError } from '@/types/common';
//
// 这些导出可能在未来版本中移除。
// ============================================================================

export { isFormValidationError } from "@/types/common";
export type { FormFieldError } from "@/types/common";

// 自定义错误类型，带有额外属性
interface EnhancedError extends Error {
  status?: number;
  type?: HttpErrorType;
  originalError?: unknown;
}

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
 * 从错误对象中提取错误消息
 */
function extractErrorMessage(error: unknown): string {
  if (typeof error === "string") {
    return error;
  }

  if (error && typeof error === "object") {
    const err = error as ErrorResponse;
    return err?.response?.data?.message
      ?? err?.response?.data?.msg
      ?? err?.response?.data?.error
      ?? err?.message
      ?? err?.msg
      ?? err?.error
      ?? "操作失败";
  }

  return "操作失败";
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
    const errorMessage = extractErrorMessage(error);
    console.error(`[${context}]`, error);

    if (showMessage) {
      getAppMessage().error(`${context}失败: ${errorMessage}`);
    }
  }

  /**
   * 处理成功消息
   * @param action 操作名称
   * @param isEdit 是否是编辑操作
   */
  static handleSuccess(action: string, isEdit: boolean = false): void {
    getAppMessage().success(isEdit ? `更新${action}成功` : `${action}成功`);
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
      getAppMessage().success(options.successMessage);
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
  [HttpErrorType.BAD_REQUEST]: "请求参数错误",
  [HttpErrorType.UNAUTHORIZED]: "登录已过期，请重新登录",
  [HttpErrorType.FORBIDDEN]: "没有权限访问",
  [HttpErrorType.NOT_FOUND]: "请求的资源不存在",
  [HttpErrorType.METHOD_NOT_ALLOWED]: "请求方法不允许",
  [HttpErrorType.CONFLICT]: "数据冲突，请刷新后重试",
  [HttpErrorType.INTERNAL_ERROR]: "服务器内部错误",
  [HttpErrorType.NOT_IMPLEMENTED]: "功能未实现",
  [HttpErrorType.BAD_GATEWAY]: "网关错误",
  [HttpErrorType.SERVICE_UNAVAILABLE]: "服务暂时不可用",
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
  if (status === 400 && (errorMessage.includes("SM2") || errorMessage.includes("解密"))) {
    console.warn("[ErrorHandler] SM2 解密失败，清除公钥缓存");
    clearPublicKeyCache();
  }

  // P1-S3: 401 未授权的自动登出已由 api.ts 响应拦截器完整处理
  // (L387-451, 含 login 短路/refresh 短路/刷新队列/失败跳转),
  // 该拦截器对所有 401 响应在到达此处之前已 return, 故此分支不可达。
  // 移除原 handleUnauthorized() 调用以消除死代码, 避免未来误以为
  // 这里是 401 处理的入口。

  // 显示错误消息
  getAppMessage().error(errorMessage);

  const error = new Error(errorMessage);
  (error as EnhancedError).status = status;
  (error as EnhancedError).type = errorType;
  return error;
}

/**
 * 处理网络错误
 * @param error 原始错误对象
 * @returns Error 对象
 */
export function handleNetworkError(error: unknown): Error {
  const errorMessage = (error as { code?: string })?.code === "ECONNABORTED"
    ? "请求超时，请检查网络连接"
    : "网络异常，请检查网络连接";

  getAppMessage().error(errorMessage);

  const networkError = new Error(errorMessage);
  (networkError as EnhancedError).originalError = error;
  return networkError;
}

/**
 * 处理响应解析错误
 * @param error 原始错误对象
 * @returns Error 对象
 */
export function handleParseError(error: unknown): Error {
  const errorMessage = "响应解析失败";

  getAppMessage().error(errorMessage);
  console.error("[ErrorHandler] Response parse error:", error);

  const parseError = new Error(errorMessage);
  (parseError as EnhancedError).originalError = error;
  return parseError;
}

/**
 * 根据状态码获取错误类型
 */
function getErrorTypeByStatus(status: number): HttpErrorType {
  const statusToTypeMap: Partial<Record<number, HttpErrorType>> = {
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

  return statusToTypeMap[status] ?? HttpErrorType.INTERNAL_ERROR;
}

/**
 * 处理未授权错误（登出逻辑）
 */
async function handleUnauthorized(): Promise<void> {
  // 清除 auth store 状态（内部已调用 tokenManager.clearTokens() 清理 sessionStorage 中的加密 token）
  await useAuthStore.getState().logout();

  // 跳转到登录页
  window.location.href = LOGIN;
}

// ==================== 类型安全的异步操作包装 ====================

/**
 * 异步操作结果类型
 * 用于替代 try-catch 的类型安全错误处理
 */
export type AsyncResult<T> =
  | { success: true; data: T }
  | { success: false; error: Error };

/**
 * 将未知错误转换为 Error 对象
 */
function toError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }
  if (typeof error === "string") {
    return new Error(error);
  }
  return new Error(String(error));
}

/**
 * 类型安全的异步包装器
 * 将可能抛出异常的函数包装为类型安全的 AsyncResult
 *
 * @example
 * const result = await safeAsync(() => await fetchData());
 * if (result.success) {
 *   console.log(result.data); // 类型安全，data 是 T 类型
 * } else {
 *   console.error(result.error); // error 是 Error 类型
 * }
 */
export async function safeAsync<T>(
  operation: () => Promise<T>
): Promise<AsyncResult<T>> {
  try {
    const data = await operation();
    return { success: true, data };
  } catch (error) {
    return { success: false, error: toError(error) };
  }
}

/**
 * 同步操作的类型安全包装器
 *
 * @example
 * const result = safeSync(() => JSON.parse(jsonString));
 * if (result.success) {
 *   console.log(result.data); // 类型安全的 data
 * }
 */
export function safeSync<T>(
  operation: () => T
): AsyncResult<T> {
  try {
    const data = operation();
    return { success: true, data };
  } catch (error) {
    return { success: false, error: toError(error) };
  }
}

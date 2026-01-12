import { message } from 'antd';

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

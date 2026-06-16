/**
 * 通用类型定义
 * 用于支持 TypeScript 严格模式迁移
 */

import type { FormInstance } from "antd";

// FormFieldError 类型定义（Ant Design 6 已移除此导出）
export interface FormFieldError {
  name: (string | number)[];
  errors: string[];
  warnings?: string[];
}

// ==================== 表单类型 ====================

/**
 * 通用表单实例类型
 * 替代 any 类型的 FormInstance
 */
export type GenericFormInstance = FormInstance;

/**
 * 表单字段值类型
 */
export type FormFieldValue = string | number | boolean | string[] | number[] | undefined | null;

/**
 * 表单数据类型
 */
export interface FormData {
  [key: string]: FormFieldValue;
}

// ==================== 错误类型 ====================

/**
 * 未知错误类型（用于 catch 块）
 * 比使用 any 更安全
 */
export interface UnknownError {
  message?: string;
  code?: string | number;
  response?: {
    data?: {
      message?: string;
      code?: string | number;
    };
    status?: number;
  };
  errorFields?: FormFieldError[]; // Ant Design 表单验证错误
}

/**
 * 错误类型守卫 - 检查是否为表单验证错误
 */
export function isFormValidationError(error: unknown): error is { errorFields: FormFieldError[] } {
  return (
    typeof error === "object" &&
    error !== null &&
    "errorFields" in error &&
    Array.isArray((error as { errorFields: FormFieldError[] }).errorFields)
  );
}

// ==================== 回调类型 ====================

/**
 * 无参数无返回值回调
 */
export type VoidCallback = () => void;

/**
 * 异步无返回值回调
 */
export type AsyncVoidCallback = () => Promise<void>;

/**
 * 通用成功回调
 */
export type SuccessCallback<T = void> = (data?: T) => void;

/**
 * 通用错误回调
 */
export type ErrorCallback = (error: Error | UnknownError) => void;

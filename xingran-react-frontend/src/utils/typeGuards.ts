/**
 * 类型守卫工具
 * 用于运行时类型检查和类型缩小
 */

import type { UnknownError } from "@/types/common";

/**
 * 检查是否为 Error 对象
 */
export function isError(error: unknown): error is Error {
  return (
    error instanceof Error ||
    (typeof error === "object" &&
      error !== null &&
      "message" in error &&
      typeof (error as Error).message === "string")
  );
}

/**
 * 获取错误消息
 */
export function getErrorMessage(error: unknown): string {
  if (isError(error)) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  return "发生未知错误";
}

/**
 * 检查对象是否有指定属性
 */
export function hasProperty<T extends PropertyKey>(
  obj: unknown,
  prop: T
): obj is Record<T, unknown> {
  return typeof obj === "object" && obj !== null && prop in obj;
}

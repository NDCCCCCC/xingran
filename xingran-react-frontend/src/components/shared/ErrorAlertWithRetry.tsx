/**
 * 错误态组件
 *
 * 接受 error / onRetry props,按错误码 1006 / 1007 / 500 分级展示文案。
 * - 1006: 设备未找到 → "该设备不存在或已被删除"
 * - 1007: token 失效 → 调用 authStore.logout() + 跳登录页
 * - 500: 服务器错误 → "服务暂不可用,请稍后重试"
 * - 其他: "查询失败:${error.message}"
 *
 * Alert 后跟 "重新加载" 按钮(调用 onRetry 回调)。
 *
 * 使用场景:React Query / API 调用失败的内联错误展示。
 */

import { useEffect, useRef, type FC } from "react";
import { Alert, Button, Space } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { useAuthStore } from "@/store/authStore";
import { LOGIN } from "@/constants/routes";

/** 业务错误对象,支持从 API 响应中提取的 code 字段 */
export interface ApiErrorShape {
  /** 业务错误码(对应后端 API 响应规范的 code 字段) */
  code?: number;
  /** 错误描述 */
  message?: string;
}

export interface ErrorAlertWithRetryProps {
  /** 错误对象(可以是 Error 实例或 ApiErrorShape) */
  error: Error | ApiErrorShape | null | undefined;
  /** 重试回调(可选) */
  onRetry?: () => void;
  /** 自定义错误描述(可选,覆盖默认的错误码描述) */
  description?: string;
}

/**
 * 从任意错误对象中提取业务错误码
 * 兼容后端响应结构:error.code / error.response.data.code / error.status
 */
function extractErrorCode(error: Error | ApiErrorShape | null | undefined): number | undefined {
  if (!error || typeof error !== "object") return undefined;

  const e = error as ApiErrorShape & {
    response?: { data?: { code?: number } };
    status?: number;
  };

  if (typeof e.code === "number") return e.code;
  if (e.response?.data && typeof e.response.data.code === "number") {
    return e.response.data.code;
  }
  if (typeof e.status === "number" && e.status >= 400) return e.status;
  return undefined;
}

/**
 * 从任意错误对象中提取错误描述
 */
function extractErrorMessage(error: Error | ApiErrorShape | null | undefined): string {
  if (!error) return "未知错误";
  if (error instanceof Error) return error.message || "未知错误";

  const e = error as ApiErrorShape & {
    response?: { data?: { message?: string; msg?: string } };
  };

  return (
    e.message
    ?? e.response?.data?.message
    ?? e.response?.data?.msg
    ?? "未知错误"
  );
}

const ErrorAlertWithRetry: FC<ErrorAlertWithRetryProps> = ({ error, onRetry, description }) => {
  const code = extractErrorCode(error);
  const messageText = extractErrorMessage(error);
  const logout = useAuthStore((s) => s.logout);

  // 1007 业务错误码:token 失效,自动登出并跳转登录页
  // ranRef 跨 render 持久化,即使父组件 state 变化导致重渲染也不会再触发 logout;
  // cancelledRef 在组件卸载时设为 true,避免在已卸载状态下操作 window.location
  const ranRef = useRef<number | null>(null);
  const cancelledRef = useRef(false);
  useEffect(() => {
    if (code !== 1007) return;
    if (ranRef.current === code) return; // 同一个 code 已经处理过,跳过
    ranRef.current = code;
    cancelledRef.current = false;
    // 调用 authStore.logout 清理 token / 用户状态 / 菜单
    logout()
      .catch(() => undefined)
      .finally(() => {
        if (!cancelledRef.current) {
          window.location.href = LOGIN;
        }
      });
    return () => {
      cancelledRef.current = true;
    };
  }, [code, logout]);

  // 根据错误码生成对应的 Alert 文案
  let alertMessage: string;
  switch (code) {
    case 1006:
      alertMessage = "该设备不存在或已被删除";
      break;
    case 1007:
      alertMessage = "登录已失效,正在跳转...";
      break;
    case 500:
      alertMessage = "服务暂不可用,请稍后重试";
      break;
    default:
      alertMessage = `查询失败:${messageText}`;
  }

  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      <Alert
        type="error"
        message={alertMessage}
        showIcon
        description={description ?? (code === undefined || code === 1006 || code === 500
          ? undefined
          : `错误码:${code}`)}
      />
      {onRetry && (
        <Button icon={<ReloadOutlined />} onClick={onRetry}>
          重新加载
        </Button>
      )}
    </Space>
  );
};

export default ErrorAlertWithRetry;

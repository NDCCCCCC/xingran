/**
 * RouteGuard - 路由级权限守卫组件
 *
 * 用于静态路由（如 /system/notice/:id）的客户端权限检查。
 * 注意：这不是安全边界——后端 API 必须独立校验数据归属。
 * RouteGuard 仅提供 UX 优化:
 *   - 无权限用户看到 "无权限访问" 页面而非 403 错误
 *   - 自动重定向到 fallback 路径
 *   - 阻止空白页或加载失败
 *
 * 设计决策:
 *   - 简化实现: 只做权限检查, 不做 ownership check (那是后端的事)
 *   - 不在 effect 中读 URL params (那是 useParams 的工作, 静态路由无法拿)
 *   - 失败时显示 inline 错误而非 toast (符合路由级 UX)
 */

import { Navigate } from "react-router-dom";
import { Result, Button } from "antd";
import { useMenuStore } from "@/store/menuStore";

export interface RouteGuardProps {
  /** 需要的权限点列表 (任一满足即可) */
  permissions: string[];
  /** 无权限时的重定向路径 (与 fallback 二选一) */
  fallback?: string;
  /** 无权限时显示的错误页面 (与 fallback 二选一; 优先 fallback) */
  fallbackElement?: React.ReactNode;
  /** 子元素 (要保护的实际路由内容) */
  children: React.ReactNode;
}

export function RouteGuard({ permissions, fallback, fallbackElement, children }: RouteGuardProps) {
  const { permissions: userPermissions } = useMenuStore();

  // 空 permissions 数组 = 无权限要求, 直接放行
  if (permissions.length === 0) {
    return <>{children}</>;
  }

  // 检查用户是否拥有任一所需权限
  const hasPermission = permissions.some((p) => userPermissions.includes(p));

  if (!hasPermission) {
    // 优先使用重定向 (更接近原生 SPA UX)
    if (fallback) {
      return <Navigate to={fallback} replace />;
    }

    // 否则显示 inline 错误页面 (告知用户无权访问)
    return (
      fallbackElement ?? (
        <Result
          status="403"
          title="403"
          subTitle="抱歉，您无权访问此页面。"
          extra={
            <Button type="primary" onClick={() => window.history.back()}>
              返回
            </Button>
          }
        />
      )
    );
  }

  return <>{children}</>;
}

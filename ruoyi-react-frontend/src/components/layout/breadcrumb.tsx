/**
 * 面包屑组件
 * 使用路由配置管理器动态生成面包屑
 */

import { useMemo } from "react";
import type { FC } from "react";
import { Breadcrumb } from "antd";
import { useLocation, Link } from "react-router-dom";
import { routeConfigManager } from "@/router/routeConfigManager";

const BreadcrumbComponent: FC = () => {
  const location = useLocation();

  const breadcrumbItems = useMemo(() => {
    // 如果路由配置管理器未初始化，返回空数组
    if (!routeConfigManager.isInitialized()) {
      return [];
    }

    // 使用路由配置管理器构建面包屑
    const breadcrumbPath = routeConfigManager.buildBreadcrumb(location.pathname);

    // 转换为 Ant Design Breadcrumb 所需格式
    return breadcrumbPath.map((item, index) => ({
      key: item.path,
      title:
        index === breadcrumbPath.length - 1 ? (
          <span>{item.title}</span>
        ) : (
          <Link to={item.path}>{item.title}</Link>
        ),
    }));
  }, [location.pathname]);

  // 如果没有面包屑项，不显示
  if (breadcrumbItems.length === 0) {
    return null;
  }

  return (
    <div
      className="px-6 py-3 border-b"
      style={{
        background: "var(--theme-bg-surface)",
        borderColor: "var(--theme-border-primary)",
      }}
    >
      <Breadcrumb
        items={breadcrumbItems}
        className="text-sm"
        style={{
          color: "var(--theme-text-secondary)",
        }}
      />
    </div>
  );
};

export default BreadcrumbComponent;

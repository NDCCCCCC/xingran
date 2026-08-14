/**
 * ExcelImport 懒加载包装器
 * 使用 React.lazy 延迟加载 ExcelImport 组件，减少首屏 bundle 大小
 */

import { lazy, Suspense } from "react";
import { Spin } from "antd";

// 懒加载 ExcelImport 组件
const ExcelImport = lazy(() =>
  import("./ExcelImport").then((module) => ({ default: module.default }))
);

export interface ExcelImportLazyProps {
  entityType: string;
  entityName: string;
  visible: boolean;
  onClose: () => void;
  onImportSuccess: () => void;
}

/**
 * ExcelImport 懒加载包装器组件
 * 提供统一的加载状态和错误处理
 */
export default function ExcelImportLazy(props: ExcelImportLazyProps) {
  return (
    <Suspense
      fallback={
        <div style={{ textAlign: "center", padding: "40px 0" }}>
          <Spin>
            <div style={{ minHeight: 80 }} />
          </Spin>
          <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载导入组件...</div>
        </div>
      }
    >
      <ExcelImport {...props} />
    </Suspense>
  );
}

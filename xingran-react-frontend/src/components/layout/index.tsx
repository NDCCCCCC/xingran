/**
 * 布局入口组件
 * 根据当前选择的布局类型动态渲染不同的布局组件
 */

import type { FC, ReactNode } from "react";
import { useLayoutStore } from "@/store/layoutStore";
import ClassicLayout from "./ClassicLayout";
import HybridLayout from "./HybridLayout";
import InnovativeLayout from "./InnovativeLayout";

interface LayoutProps {
  children: ReactNode;
}

const Layout: FC<LayoutProps> = ({ children }) => {
  const currentLayout = useLayoutStore((state) => state.currentLayout);

  if (currentLayout === "classic") {
    return <ClassicLayout key={currentLayout}>{children}</ClassicLayout>;
  }

  if (currentLayout === "hybrid") {
    return <HybridLayout key={currentLayout}>{children}</HybridLayout>;
  }

  if (currentLayout === "innovative") {
    return <InnovativeLayout key={currentLayout}>{children}</InnovativeLayout>;
  }

  // 默认使用经典布局
  return <ClassicLayout key={currentLayout}>{children}</ClassicLayout>;
};

export default Layout;

/**
 * 布局入口组件
 * 根据当前选择的布局类型动态渲染不同的布局组件
 */

import { useEffect, useState } from 'react';
import type { FC, ReactNode } from 'react';
import { useLayoutStore } from '@/store/layoutStore';
import ClassicLayout from './ClassicLayout';
import HybridLayout from './HybridLayout';
import InnovativeLayout from './InnovativeLayout';

interface LayoutProps {
  children: ReactNode;
}

const Layout: FC<LayoutProps> = ({ children }) => {
  const currentLayout = useLayoutStore((state) => state.currentLayout);
  const [isHydrated, setIsHydrated] = useState(false);

  // 确保 zustand persist 水合完成后再渲染
  useEffect(() => {
    // 使用 setTimeout 确保在下一个事件循环中检查状态
    const timer = setTimeout(() => {
      setIsHydrated(true);
    }, 0);

    return () => clearTimeout(timer);
  }, []);

  // 等待 store 水合完成
  if (!isHydrated) {
    return null;
  }

  if (currentLayout === 'classic') {
    return <ClassicLayout key={currentLayout}>{children}</ClassicLayout>;
  }

  if (currentLayout === 'hybrid') {
    return <HybridLayout key={currentLayout}>{children}</HybridLayout>;
  }

  if (currentLayout === 'innovative') {
    return <InnovativeLayout key={currentLayout}>{children}</InnovativeLayout>;
  }

  // 默认使用经典布局
  return <ClassicLayout key={currentLayout}>{children}</ClassicLayout>;
};

export default Layout;

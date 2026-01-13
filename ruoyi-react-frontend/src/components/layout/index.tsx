/**
 * 布局入口组件
 * 根据当前选择的布局类型动态渲染不同的布局组件
 */

import { useEffect, useState, useRef } from 'react';
import type { FC, ReactNode } from 'react';
import { useLayoutStore } from '@/store/layoutStore';
import ClassicLayout from './ClassicLayout';
import HybridLayout from './HybridLayout';
import InnovativeLayout from './InnovativeLayout';

interface LayoutProps {
  children: ReactNode;
}

const Layout: FC<LayoutProps> = ({ children }) => {
  // 直接从 store 获取布局类型，不使用本地状态
  // 避免使用 key 导致组件卸载/重新挂载
  const currentLayout = useLayoutStore((state) => state.currentLayout);
  const [isHydrated, setIsHydrated] = useState(false);
  const previousLayoutRef = useRef<string | null>(null);

  // 确保 zustand persist 水合完成后再渲染
  useEffect(() => {
    const unsubscribe = useLayoutStore.subscribe(() => {
      setIsHydrated(true);
    });

    // 立即检查是否已水合
    const state = useLayoutStore.getState();
    if (state.currentLayout) {
      setIsHydrated(true);
    }

    return unsubscribe;
  }, []);

  // 短暂延迟以避免闪烁，等待store完全水合
  if (!isHydrated) {
    return null;
  }

  // 检测布局是否真正切换（只在有前值且不同时才算切换）
  const layoutChanged = previousLayoutRef.current !== null && previousLayoutRef.current !== currentLayout;

  // 更新 ref 为当前值
  previousLayoutRef.current = currentLayout;

  // 只在布局真正切换时使用 key，避免不必要的重新挂载
  const layoutKey = layoutChanged ? currentLayout : undefined;

  if (currentLayout === 'classic') {
    return <ClassicLayout key={layoutKey}>{children}</ClassicLayout>;
  }

  if (currentLayout === 'hybrid') {
    return <HybridLayout key={layoutKey}>{children}</HybridLayout>;
  }

  if (currentLayout === 'innovative') {
    return <InnovativeLayout key={layoutKey}>{children}</InnovativeLayout>;
  }

  // 默认使用经典布局
  return <ClassicLayout key={layoutKey}>{children}</ClassicLayout>;
};

export default Layout;

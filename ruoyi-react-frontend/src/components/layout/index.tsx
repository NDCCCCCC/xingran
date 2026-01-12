/**
 * 布局入口组件
 * 根据当前选择的布局类型动态渲染不同的布局组件
 */

import React, { useEffect, useState } from 'react';
import { useLayoutStore } from '@/store/layoutStore';
import ClassicLayout from './ClassicLayout';
import HybridLayout from './HybridLayout';
import InnovativeLayout from './InnovativeLayout';

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [mountedLayout, setMountedLayout] = useState<string | null>(null);

  useEffect(() => {
    // 订阅store变化，确保获取水合后的值
    const unsubscribe = useLayoutStore.subscribe((state) => {
      setMountedLayout(state.currentLayout);
    });

    // 初始设置
    setMountedLayout(useLayoutStore.getState().currentLayout);

    return unsubscribe;
  }, []);

  // 短暂延迟以避免闪烁，等待store完全水合
  if (!mountedLayout) {
    return null;
  }

  // 根据当前布局类型渲染对应的布局组件
  // 使用currentLayout作为key，确保布局切换时正确重新挂载
  if (mountedLayout === 'classic') {
    return <ClassicLayout key={mountedLayout}>{children}</ClassicLayout>;
  }

  if (mountedLayout === 'hybrid') {
    return <HybridLayout key={mountedLayout}>{children}</HybridLayout>;
  }

  if (mountedLayout === 'innovative') {
    return <InnovativeLayout key={mountedLayout}>{children}</InnovativeLayout>;
  }

  // 默认使用经典布局
  return <ClassicLayout key="default">{children}</ClassicLayout>;
};

export default Layout;

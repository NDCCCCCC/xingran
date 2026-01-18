/**
 * 多标签页组件
 * 用于混合式布局，支持标签页的打开、关闭、切换等操作
 * 功能：
 * - 滚动导航（左右滚动按钮）
 * - 右键上下文菜单（关闭当前/其他/左侧/右侧标签）
 * - 激活标签样式（底部蓝色指示条，与侧边栏保持一致）
 */

import { useRef, useState, useMemo, useEffect, useCallback } from 'react';
import { Tabs, Dropdown, Button } from 'antd';
import type { MenuProps } from 'antd';
import { useTabs } from '@/store/tabsStore';
import { useNavigate } from 'react-router-dom';
import {
  CloseOutlined,
  CloseCircleOutlined,
  DownOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';

import type { FC, ReactNode } from 'react';

interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  tabKey: string;
}

const TabBar: FC = () => {
  const { tabs, activeTab, setActiveTab, removeTab, closeAllTabs, closeOtherTabs, closeLeftTabs, closeRightTabs } = useTabs();
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // 滚动状态
  const [scrollState, setScrollState] = useState({
    canScrollLeft: false,
    canScrollRight: false,
    scrollLeft: 0,
  });

  // 右键菜单状态
  const [contextMenuState, setContextMenuState] = useState<ContextMenuState>({
    visible: false,
    x: 0,
    y: 0,
    tabKey: '',
  });

  // 处理标签页点击
  const handleTabChange = useCallback((key: string) => {
    const tab = tabs.find(t => t.key === key);
    if (tab) {
      setActiveTab(key);
      navigate(tab.path);
    }
    // 关闭右键菜单
    setContextMenuState(prev => ({ ...prev, visible: false }));
  }, [tabs, setActiveTab, navigate]);

  // 处理标签页关闭
  const handleTabClose = useCallback((targetKey: string) => {
    // 如果关闭的是当前激活的标签，需要切换到前一个
    if (targetKey === activeTab && tabs.length > 1) {
      const currentIndex = tabs.findIndex(t => t.key === targetKey);
      if (currentIndex > 0) {
        const prevTab = tabs[currentIndex - 1];
        setActiveTab(prevTab.key);
        navigate(prevTab.path);
      } else if (tabs.length > 1) {
        const nextTab = tabs[1];
        setActiveTab(nextTab.key);
        navigate(nextTab.path);
      }
    }
    removeTab(targetKey);
    // 关闭右键菜单
    setContextMenuState(prev => ({ ...prev, visible: false }));
  }, [activeTab, tabs, setActiveTab, navigate, removeTab]);

  // 滚动控制函数
  const scrollTabs = useCallback((direction: 'left' | 'right') => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const scrollAmount = 200;
    const currentScroll = container.scrollLeft;
    container.scrollTo({
      left: direction === 'left' ? currentScroll - scrollAmount : currentScroll + scrollAmount,
      behavior: 'smooth'
    });
  }, []);

  // 检测滚动状态
  const checkScrollState = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollLeft, scrollWidth, clientWidth } = container;
    const canScrollLeft = scrollLeft > 0;
    const canScrollRight = scrollWidth > clientWidth && scrollLeft < scrollWidth - clientWidth - 1; // -1 容忍浮点误差

    // 开发环境调试输出
    if (process.env.NODE_ENV === 'development') {
      console.log('[TabBar] 滚动状态:', { scrollLeft, scrollWidth, clientWidth, canScrollLeft, canScrollRight });
    }

    setScrollState({
      canScrollLeft,
      canScrollRight,
      scrollLeft,
    });
  }, []);

  // 初始化和监听滚动状态
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    // 初始检测（多次延迟确保 DOM 完全渲染）
    const timers = [
      setTimeout(checkScrollState, 0),
      setTimeout(checkScrollState, 100),
      setTimeout(checkScrollState, 300),
    ];

    // 监听滚动事件
    const handleScroll = () => {
      checkScrollState();
    };
    container.addEventListener('scroll', handleScroll, { passive: true });

    // 监听容器尺寸变化
    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(checkScrollState);
    });
    resizeObserver.observe(container);

    // 监听窗口大小变化
    const handleResize = () => {
      checkScrollState();
    };
    window.addEventListener('resize', handleResize);

    return () => {
      timers.forEach(timer => clearTimeout(timer));
      container.removeEventListener('scroll', handleScroll);
      resizeObserver.disconnect();
      window.removeEventListener('resize', handleResize);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabs.length]); // 只在 tabs 数量变化时重新绑定；checkScrollState 引用不变，不需要作为依赖

  // 右键菜单处理
  const handleTabContextMenu = useCallback((tabKey: string, e: React.MouseEvent) => {
    e.preventDefault();
    setContextMenuState({ visible: true, x: e.clientX, y: e.clientY, tabKey });
  }, []);

  // 右键菜单项
  const contextMenuItems: MenuProps['items'] = useMemo(() => {
    const tab = tabs.find(t => t.key === contextMenuState.tabKey);
    const tabIndex = tabs.findIndex(t => t.key === contextMenuState.tabKey);
    const hasLeftTabs = tabIndex > 0;
    const hasRightTabs = tabIndex < tabs.length - 1;

    return [
      {
        key: 'close',
        label: '关闭当前标签',
        icon: <CloseOutlined />,
        disabled: !tab?.closable,
        onClick: () => {
          if (contextMenuState.tabKey) {
            handleTabClose(contextMenuState.tabKey);
          }
        },
      },
      { type: 'divider' },
      {
        key: 'closeOthers',
        label: '关闭其他标签',
        icon: <CloseCircleOutlined />,
        onClick: () => {
          if (contextMenuState.tabKey) {
            closeOtherTabs(contextMenuState.tabKey);
          }
          setContextMenuState(prev => ({ ...prev, visible: false }));
        },
      },
      {
        key: 'closeLeft',
        label: '关闭左侧标签',
        icon: <LeftOutlined />,
        disabled: !hasLeftTabs,
        onClick: () => {
          if (contextMenuState.tabKey) {
            closeLeftTabs(contextMenuState.tabKey);
          }
          setContextMenuState(prev => ({ ...prev, visible: false }));
        },
      },
      {
        key: 'closeRight',
        label: '关闭右侧标签',
        icon: <RightOutlined />,
        disabled: !hasRightTabs,
        onClick: () => {
          if (contextMenuState.tabKey) {
            closeRightTabs(contextMenuState.tabKey);
          }
          setContextMenuState(prev => ({ ...prev, visible: false }));
        },
      },
    ];
  }, [tabs, contextMenuState.tabKey, handleTabClose, closeOtherTabs, closeLeftTabs, closeRightTabs]);

  // 更多标签菜单
  const moreMenuItems: MenuProps['items'] = useMemo(() => {
    const closableTabs = tabs.filter(tab => tab.closable);
    const pinnedTabs = tabs.filter(tab => !tab.closable && tab.pinned);

    return [
      ...pinnedTabs.map(tab => ({
        key: tab.key,
        label: (
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {tab.icon}
            <span>{tab.title}</span>
          </div>
        ),
        onClick: () => handleTabChange(tab.key),
      })),
      ...(pinnedTabs.length > 0 && closableTabs.length > 0 ? [{ type: 'divider' as const }] : []),
      ...closableTabs.map(tab => ({
        key: tab.key,
        label: (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              {tab.icon}
              <span>{tab.title}</span>
            </div>
            <CloseOutlined
              style={{ fontSize: '12px' }}
              onClick={(e) => {
                e.stopPropagation();
                handleTabClose(tab.key);
              }}
            />
          </div>
        ),
        onClick: () => handleTabChange(tab.key),
      })),
      ...(closableTabs.length > 0 ? [{ type: 'divider' as const }] : []),
      {
        key: 'closeAll',
        label: '关闭所有',
        icon: <CloseCircleOutlined />,
        onClick: () => {
          closeAllTabs();
          setContextMenuState(prev => ({ ...prev, visible: false }));
        },
        danger: true,
      },
    ];
  }, [tabs, handleTabChange, handleTabClose, closeAllTabs]);

  // 自定义 Tab label（添加右键菜单）
  const createTabLabel = (tab: { title: string | ReactNode; key: string }) => (
    <span onContextMenu={(e) => handleTabContextMenu(tab.key, e)}>{tab.title}</span>
  );

  // 全局事件监听 - 关闭右键菜单
  useEffect(() => {
    const handleGlobalClick = () => {
      if (contextMenuState.visible) {
        setContextMenuState(prev => ({ ...prev, visible: false }));
      }
    };

    const handleScroll = () => {
      if (contextMenuState.visible) {
        setContextMenuState(prev => ({ ...prev, visible: false }));
      }
    };

    if (contextMenuState.visible) {
      document.addEventListener('click', handleGlobalClick);
      document.addEventListener('scroll', handleScroll, true);
      window.addEventListener('resize', handleScroll);

      return () => {
        document.removeEventListener('click', handleGlobalClick);
        document.removeEventListener('scroll', handleScroll, true);
        window.removeEventListener('resize', handleScroll);
      };
    }
  }, [contextMenuState.visible]);

  // 所有标签页项
  const allTabItems = tabs.map(tab => ({
    key: tab.key,
    label: createTabLabel(tab),
    closable: tab.closable,
    icon: tab.icon,
  }));

  if (tabs.length === 0) {
    return null;
  }

  return (
    <>
      <div
        ref={containerRef}
        className="border-b"
        style={{
          background: 'var(--theme-bg-surface)',
          borderColor: 'var(--theme-border-primary)',
          padding: '0 16px',
          transition: 'all var(--theme-transition-base)',
          display: 'flex',
          alignItems: 'center',
          height: '40px',
          minHeight: '40px',
          position: 'relative',
          zIndex: 1,
          flexShrink: 0,
        }}
      >
        {/* 左滚动按钮 */}
        <Button
          icon={<LeftOutlined />}
          disabled={!scrollState.canScrollLeft}
          onClick={() => scrollTabs('left')}
          className="tab-scroll-button"
        />

        {/* 滚动容器 */}
        <div
          ref={scrollContainerRef}
          className="tabs-scroll-container"
        >
          <Tabs
            activeKey={activeTab}
            onChange={handleTabChange}
            type="editable-card"
            hideAdd
            items={allTabItems}
            onEdit={(targetKey, action) => {
              if (action === 'remove') {
                handleTabClose(targetKey as string);
              }
            }}
            className="layout-tab-bar"
            style={{
              color: 'var(--theme-text-primary)',
            }}
          />
        </div>

        {/* 右滚动按钮 */}
        <Button
          icon={<RightOutlined />}
          disabled={!scrollState.canScrollRight}
          onClick={() => scrollTabs('right')}
          className="tab-scroll-button"
        />

        {/* "更多"下拉菜单 */}
        <Dropdown
          menu={{
            items: moreMenuItems,
            style: {
              zIndex: 1000,
            },
          }}
          trigger={['click']}
          placement="bottomRight"
          getPopupContainer={(triggerNode) => {
            return triggerNode?.parentNode?.parentNode?.parentNode as HTMLElement || document.body;
          }}
        >
          <Button
            type="text"
            icon={<DownOutlined />}
            style={{
              marginLeft: '8px',
              height: '32px',
              minWidth: '32px',
              padding: '0 8px',
              color: 'var(--theme-text-secondary)',
            }}
          >
            更多
          </Button>
        </Dropdown>
      </div>

      {/* 右键上下文菜单 - 使用 Dropdown 的 open 控制 */}
      <Dropdown
        menu={{ items: contextMenuItems }}
        trigger={[]}
        open={contextMenuState.visible}
        onOpenChange={(open) => setContextMenuState(prev => ({ ...prev, visible: open }))}
        placement="bottomLeft"
        getPopupContainer={() => document.body}
      >
        <div
          style={{
            position: 'fixed',
            left: contextMenuState.x,
            top: contextMenuState.y,
            width: 1,
            height: 1,
            pointerEvents: 'none',
          }}
        />
      </Dropdown>
    </>
  );
};

export default TabBar;

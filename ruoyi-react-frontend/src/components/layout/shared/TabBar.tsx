/**
 * 多标签页组件
 * 用于混合式布局，支持标签页的打开、关闭、切换等操作
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
} from '@ant-design/icons';

import type { FC } from 'react';

const TabBar: FC = () => {
  const { tabs, activeTab, setActiveTab, removeTab, closeAllTabs } = useTabs();
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);
  const [maxVisibleTabs, setMaxVisibleTabs] = useState(20);

  // 处理标签页点击
  const handleTabChange = useCallback((key: string) => {
    const tab = tabs.find(t => t.key === key);
    if (tab) {
      setActiveTab(key);
      navigate(tab.path);
    }
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
  }, [activeTab, tabs, setActiveTab, navigate, removeTab]);

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
        onClick: () => closeAllTabs(),
        danger: true,
      },
    ];
  }, [tabs, handleTabChange, handleTabClose, closeAllTabs]); // 添加所有在函数内使用的回调

  // 过滤掉固定标签（固定标签不可关闭）
  const closableTabs = tabs.filter(tab => tab.closable);
  const pinnedTabs = tabs.filter(tab => !tab.closable && tab.pinned);

  // 计算可见的标签页
  const visibleTabs = useMemo(() => {
    const allTabs = [...pinnedTabs, ...closableTabs];
    if (allTabs.length <= maxVisibleTabs) {
      return allTabs;
    }
    
    // 如果标签页数量超过限制，显示前N-1个标签页和"更多"按钮
    return allTabs.slice(0, maxVisibleTabs - 1);
  }, [tabs, pinnedTabs, closableTabs, maxVisibleTabs]);

  // 是否显示更多按钮
  const showMoreButton = useMemo(() => {
    const allTabs = [...pinnedTabs, ...closableTabs];
    return allTabs.length > maxVisibleTabs;
  }, [tabs, pinnedTabs, closableTabs, maxVisibleTabs]);

  // 动态计算最大可见标签页数量
  useEffect(() => {
    const calculateMaxVisibleTabs = () => {
      if (!containerRef.current) return;
      
      const containerWidth = containerRef.current.clientWidth;
      // 估算每个标签页的平均宽度（包括图标、文字、关闭按钮和边距）
      const estimatedTabWidth = 160; // 每个标签页约160px
      // "更多"按钮的宽度
      const moreButtonWidth = 80;
      // 左右边距
      const paddingWidth = 32;
      
      // 计算可用的宽度
      const availableWidth = containerWidth - paddingWidth - moreButtonWidth;
      // 计算最大可见标签页数量
      const calculatedMax = Math.floor(availableWidth / estimatedTabWidth);
      
      // 确保至少显示3个标签页
      const finalMax = Math.max(3, calculatedMax);
      
      setMaxVisibleTabs(finalMax);
    };

    calculateMaxVisibleTabs();
    
    // 监听窗口大小变化
    const handleResize = () => {
      calculateMaxVisibleTabs();
    };
    
    window.addEventListener('resize', handleResize);
    
    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  const items = [
    ...visibleTabs.map(tab => ({
      key: tab.key,
      label: tab.title,
      closable: tab.closable,
      icon: tab.icon,
    })),
  ];

  if (items.length === 0) {
    return null;
  }

  return (
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
      <div style={{
        flex: 1,
        overflow: 'hidden',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
      }}>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          type="editable-card"
          hideAdd
          items={items}
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

      {showMoreButton && (
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
      )}
    </div>
  );
};

export default TabBar;

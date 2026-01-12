/**
 * 多标签页组件
 * 用于混合式布局，支持标签页的打开、关闭、切换等操作
 */

import React, { useRef } from 'react';
import { Tabs, Dropdown } from 'antd';
import { useTabs } from '@/store/tabsStore';
import { useNavigate, useLocation } from 'react-router-dom';
import type { MenuProps } from 'antd';

const TabBar: React.FC = () => {
  const { tabs, activeTab, setActiveTab, removeTab, closeOtherTabs, closeAllTabs } = useTabs();
  const navigate = useNavigate();
  const location = useLocation();
  const tabsRef = useRef<HTMLDivElement>(null);

  // 处理标签页右键菜单
  const handleContextMenu = (tabKey: string, e: React.MouseEvent) => {
    e.preventDefault();

    const menuItems: MenuProps['items'] = [
      {
        key: 'close',
        label: '关闭',
        onClick: () => removeTab(tabKey),
      },
      {
        key: 'closeOthers',
        label: '关闭其他',
        onClick: () => closeOtherTabs(tabKey),
      },
      {
        key: 'closeAll',
        label: '关闭所有',
        onClick: () => closeAllTabs(),
      },
      ];

    // 简单的上下文菜单实现
    // 实际项目中可以使用 rc-menu 或其他组件库
  };

  // 处理标签页点击
  const handleTabChange = (key: string) => {
    const tab = tabs.find(t => t.key === key);
    if (tab) {
      setActiveTab(key);
      navigate(tab.path);
    }
  };

  // 处理标签页关闭
  const handleTabClose = (targetKey: string) => {
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
  };

  // 过滤掉固定标签（固定标签不可关闭）
  const closableTabs = tabs.filter(tab => tab.closable);
  const pinnedTabs = tabs.filter(tab => !tab.closable && tab.pinned);

  const items = [
    ...pinnedTabs.map(tab => ({
      key: tab.key,
      label: tab.title,
      closable: false,
      icon: tab.icon,
    })),
    ...closableTabs.map(tab => ({
      key: tab.key,
      label: tab.title,
      closable: true,
      icon: tab.icon,
    })),
  ];

  if (items.length === 0) {
    return null;
  }

  return (
    <div
      ref={tabsRef}
      className="border-b"
      style={{
        background: 'var(--theme-bg-surface)',
        borderColor: 'var(--theme-border-primary)',
        padding: '0 24px',
        transition: 'all var(--theme-transition-base)',
      }}
    >
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
  );
};

export default TabBar;

/**
 * 混合式布局组件
 * 可折叠侧边栏 + 多标签页系统
 */

import { Layout as AntLayout } from "antd";
import Header from "./header";
import Sidebar from "./sidebar";
import TabBar from "./shared/TabBar";
import { useRouteTabs } from "./shared/useRouteTabs";

import type { FC, ReactNode } from "react";

const { Content } = AntLayout;

interface HybridLayoutProps {
  children: ReactNode;
}

const HybridLayout: FC<HybridLayoutProps> = ({ children }) => {
  // v1.23：标签跟踪逻辑抽取到 shared/useRouteTabs（与 ClassicLayout 共用）
  useRouteTabs();

  return (
    <AntLayout className="h-screen" style={{ background: "var(--theme-bg-secondary)" }}>
      <Sidebar />
      <AntLayout
        className="h-full"
        style={{ background: "var(--theme-bg-secondary)", position: "relative" }}
      >
        <Header />
        <TabBar />
        {/* 原型复刻（v1.23）：正文 = 奶油画布 + 白卡浮层（与 ClassicLayout 一致） */}
        <Content
          className="overflow-auto"
          style={{
            padding: "20px 28px 40px",
            background: "var(--theme-bg-secondary)",
          }}
        >
          {children}
        </Content>
      </AntLayout>
    </AntLayout>
  );
};

export default HybridLayout;

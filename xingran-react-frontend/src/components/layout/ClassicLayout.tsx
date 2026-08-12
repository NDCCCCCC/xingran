/**
 * 经典布局组件
 * 左侧导航 + 顶部栏 + 内容区
 */

import type { FC, ReactNode } from "react";
import { Layout as AntLayout } from "antd";
import Header from "./header";
import Sidebar from "./sidebar";
import Breadcrumb from "./breadcrumb";

const { Content } = AntLayout;

interface ClassicLayoutProps {
  children: ReactNode;
}

const ClassicLayout: FC<ClassicLayoutProps> = ({ children }) => {
  return (
    <AntLayout className="h-screen" style={{ background: "var(--theme-bg-secondary)" }}>
      <Sidebar />
      <AntLayout className="h-full" style={{ background: "var(--theme-bg-secondary)" }}>
        <Header />
        <Breadcrumb />
        <Content className="flex-1 m-6 overflow-auto">
          <div
            className="rounded-xl shadow-sm p-6 min-h-full"
            style={{
              background: "var(--theme-bg-surface)",
              border: "1px solid var(--theme-border-primary)",
              transition: "all var(--theme-transition-base)",
            }}
          >
            {children}
          </div>
        </Content>
      </AntLayout>
    </AntLayout>
  );
};

export default ClassicLayout;

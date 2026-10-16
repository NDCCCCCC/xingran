import type { FC, ReactNode } from "react";
import { Layout as AntLayout } from "antd";
import Header from "./header";
import Sidebar from "./sidebar";
import TabBar from "./shared/TabBar";
import { useRouteTabs } from "./shared/useRouteTabs";

const { Content } = AntLayout;

interface ClassicLayoutProps {
  children: ReactNode;
}

const ClassicLayout: FC<ClassicLayoutProps> = ({ children }) => {
  // v1.23：classic 也带多标签栏（原型 .tabs 纯文字 + 金色下划线）
  useRouteTabs();

  return (
    <AntLayout className="h-screen" style={{ background: "var(--theme-bg-secondary)" }}>
      <Sidebar />
      <AntLayout className="h-full" style={{ background: "var(--theme-bg-secondary)" }}>
        <Header />
        <TabBar />
        {/* 原型复刻（v1.23）：正文 = 奶油画布 + 白卡浮层，不再包大白卡；
            面包屑在 header 内，独立面包屑栏移除 */}
        <Content
          className="flex-1 overflow-auto"
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

export default ClassicLayout;

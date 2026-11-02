/**
 * 布局切换器组件
 * 提供用户界面来切换布局
 */

import type { FC, ReactNode } from "react";
import { Segmented, Tooltip } from "antd";
import { ApartmentOutlined, AppstoreOutlined, RocketOutlined } from "@ant-design/icons";
import { useLayout } from "@/store/layoutStore";
import type { LayoutType } from "@/types/layout";

const LayoutSwitcher: FC = () => {
  const { layout, setLayout } = useLayout();

  const layouts: Array<{
    value: LayoutType;
    icon: ReactNode;
    label: string;
    tooltip: string;
  }> = [
    {
      value: "classic",
      icon: <ApartmentOutlined />,
      label: "经典",
      tooltip: "左侧导航+顶部栏+内容区",
    },
    {
      value: "hybrid",
      icon: <AppstoreOutlined />,
      label: "混合",
      tooltip: "可折叠侧边栏+多标签页",
    },
    {
      value: "innovative",
      icon: <RocketOutlined />,
      label: "创新",
      tooltip: "创新导航+模块化面板",
    },
  ];

  return (
    <Segmented
      value={layout}
      onChange={(value) => setLayout(value as LayoutType)}
      options={layouts.map((item) => ({
        value: item.value,
        icon: (
          <Tooltip title={item.tooltip}>
            <span>{item.icon}</span>
          </Tooltip>
        ),
        label: <span className="hidden lg:inline">{item.label}</span>,
      }))}
      className="layout-switcher"
      style={{
        transition: "var(--theme-transition-base)",
      }}
    />
  );
};

export default LayoutSwitcher;

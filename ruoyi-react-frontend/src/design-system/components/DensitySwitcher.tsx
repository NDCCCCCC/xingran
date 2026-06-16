/**
 * 密度模式切换组件
 * 支持紧凑/舒适/宽松三种模式
 */

import type { FC, ReactNode } from "react";
import { Segmented } from "antd";
import { ColumnHeightOutlined, BorderOutlined, AppstoreOutlined } from "@ant-design/icons";
import { useLayout } from "@/store/layoutStore";
import type { DensityMode } from "@/types/layout";

interface DensityOption {
  label: string;
  value: DensityMode;
  icon: ReactNode;
  description: string;
}

const densityOptions: DensityOption[] = [
  {
    label: "紧凑",
    value: "compact",
    icon: <ColumnHeightOutlined />,
    description: "信息密集型",
  },
  {
    label: "舒适",
    value: "comfortable",
    icon: <BorderOutlined />,
    description: "平衡体验",
  },
  {
    label: "宽松",
    value: "spacious",
    icon: <AppstoreOutlined />,
    description: "宽敞布局",
  },
];

const DensitySwitcher: FC = () => {
  const { density, setDensity } = useLayout();

  return (
    <Segmented
      value={density}
      onChange={(value) => setDensity(value as DensityMode)}
      options={densityOptions.map((option) => ({
        label: (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "6px",
              fontSize: "13px",
            }}
          >
            <span style={{ fontSize: "14px" }}>{option.icon}</span>
            <span>{option.label}</span>
          </div>
        ),
        value: option.value,
      }))}
      size="small"
      style={{
        background: "var(--theme-bg-secondary)",
        border: "1px solid var(--theme-border-primary)",
        transition: "all var(--theme-transition-base)",
      }}
    />
  );
};

export default DensitySwitcher;

/**
 * 部门侧边栏组件
 * 用于运营管理页面的左侧部门树筛选
 */

import type { FC } from "react";
import { Layout } from "antd";
import type { Key } from "react";
import type { DataNode } from "antd/es/tree";
import DeptTree from "@/components/DeptTree";

const { Sider } = Layout;

interface DeptSidebarProps {
  /** 选中的部门 ID */
  selectedDeptId?: string;
  /** 部门选择回调 */
  onSelect?: (selectedKeys: Key[], info: { selected: boolean; node: DataNode }) => void;
  /** 侧边栏宽度 */
  width?: number;
  /** 是否仅显示外部组织 */
  externalOnly?: boolean;
  /** 自定义样式 */
  style?: React.CSSProperties;
}

/**
 * 运营管理页面通用的部门侧边栏
 */
export const DeptSidebar: FC<DeptSidebarProps> = ({
  selectedDeptId = "",
  onSelect,
  width = 360,
  externalOnly = true,
  style,
}) => {
  return (
    <Sider
      width={width}
      className="dept-list-sider"
      style={{ background: "#fff", padding: "0 16px 16px 0", ...style }}
    >
      <DeptTree
        onSelect={onSelect}
        selectedKeys={selectedDeptId ? [selectedDeptId] : []}
        externalOnly={externalOnly}
      />
    </Sider>
  );
};

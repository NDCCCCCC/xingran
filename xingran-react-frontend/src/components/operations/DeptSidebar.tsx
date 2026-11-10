/**
 * 部门侧边栏组件
 * 用于运营管理页面的左侧部门树筛选
 *
 * UI 统一整改：Sider 改为 transparent 透出全局奶油画布（--theme-bg-secondary），
 * 白卡由 DeptTree 自带的 <Card> 承担 —— 与系统管理用户页 xr-dept-panel 范本一致。
 * 去掉自设 #fff 白底，避免业务页「多一块白色背景」。
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
  /**
   * 后端 tree/dept-options 已按可见集收口后返回的部门 ID 集合（Phase 133 / DS-06）。
   * 以后端返回为准，前端不硬编码任何过滤规则；不传时完整渲染后端 tree 结果。
   */
  visibleDeptIds?: string[];
  /** 自定义样式 */
  style?: React.CSSProperties;
}

/**
 * 运营管理页面通用的部门侧边栏
 */
export const DeptSidebar: FC<DeptSidebarProps> = ({
  selectedDeptId = "",
  onSelect,
  // 统一范本：与系统管理用户页部门列一致（250px）
  width = 250,
  externalOnly = true,
  visibleDeptIds,
  style,
}) => {
  return (
    <Sider
      width={width}
      className="dept-list-sider xr-dept-panel"
      style={{ background: "transparent", padding: "0 14px 16px 0", ...style }}
    >
      <DeptTree
        onSelect={onSelect}
        selectedKeys={selectedDeptId ? [selectedDeptId] : []}
        externalOnly={externalOnly}
        visibleDeptIds={visibleDeptIds}
      />
    </Sider>
  );
};

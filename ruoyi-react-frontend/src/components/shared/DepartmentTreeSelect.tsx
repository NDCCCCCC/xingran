/**
 * DepartmentTreeSelect - 部门树下拉选择器（共享组件）
 *
 * 统一的部门树下拉框组件，支持：
 * - 树形结构展示
 * - 搜索功能
 * - 可清空
 * - 禁用状态
 * - 支持顶级部门选项（用于部门编辑）
 * - 支持两种数据格式：标准部门格式和 TreeSelect 格式
 */

import { TreeSelect } from "antd";
import type { TreeSelectProps } from "antd/es/tree-select";

export interface Department {
  id: string;
  deptName: string;
  children?: Department[];
}

// TreeSelect 原生格式（已转换的数据）
export interface TreeNode {
  title: string;
  value: string;
  key: string;
  children?: TreeNode[];
}

export interface DepartmentTreeSelectProps extends Omit<TreeSelectProps, "treeData"> {
  value?: string;
  onChange?: (value: string) => void;
  departments?: Department[];
  treeData?: TreeNode[];  // 直接传递已转换的 TreeSelect 格式数据
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
  className?: string;
  loading?: boolean;
}

/**
 * 将部门数据转换为 TreeSelect 需要的格式
 * 从二级部门开始显示完整路径（跳过顶级部门）
 * @param departments 部门数据
 * @param parentPath 父级路径（用于构建完整路径显示）
 */
function convertDeptTreeData(departments: Department[], parentPath: string[] = []): TreeNode[] {
  if (!departments || departments.length === 0) {
    return [];
  }
  const convertNodes = (nodes: Department[], currentPath: string[]): TreeNode[] => {
    return nodes.map(node => {
      // 构建显示路径：从二级部门开始（跳过顶级）
      let displayPath: string;
      if (currentPath.length === 0) {
        // 顶级部门，直接显示名称
        displayPath = node.deptName;
      } else {
        // 从二级开始显示完整路径
        const pathFromSecondLevel = [...currentPath.slice(1), node.deptName];
        displayPath = pathFromSecondLevel.join(" / ");
      }

      return {
        title: displayPath,
        value: node.id,
        key: node.id,
        children: node.children && node.children.length > 0 ? convertNodes(node.children, [...currentPath, node.deptName]) : undefined,
      };
    });
  };
  return convertNodes(departments, parentPath);
}

/**
 * 获取第一级节点的 key（用于默认展开）- 标准部门格式
 */
function getFirstLevelKeysFromDepartments(departments: Department[]): string[] {
  if (!departments || departments.length === 0) {
    return [];
  }
  return departments.map(dept => dept.id);
}

/**
 * 获取第一级节点的 key（用于默认展开）- TreeSelect 格式
 */
function getFirstLevelKeysFromTreeData(treeData: TreeNode[]): string[] {
  if (!treeData || treeData.length === 0) {
    return [];
  }
  return treeData.map(node => node.key);
}

/**
 * 标准的部门树下拉选择器
 */
export function DepartmentTreeSelect({
  value,
  onChange,
  departments,
  treeData,
  placeholder = "请选择部门",
  allowClear = true,
  disabled = false,
  loading = false,
  style,
  className,
  ...restProps
}: DepartmentTreeSelectProps) {
  // 优先使用 treeData（已转换的格式），否则使用 departments 并转换
  const finalTreeData = treeData || (departments ? convertDeptTreeData(departments) : []);

  // 获取默认展开的 keys
  let defaultExpandedKeys: string[] = [];
  if (treeData) {
    defaultExpandedKeys = getFirstLevelKeysFromTreeData(treeData);
  } else if (departments) {
    defaultExpandedKeys = getFirstLevelKeysFromDepartments(departments);
  }

  return (
    <TreeSelect
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      allowClear={allowClear}
      disabled={disabled || loading}
      loading={loading}
      showSearch
      treeLine={{ showLeafIcon: false }}
      treeData={finalTreeData}
      treeDefaultExpandedKeys={defaultExpandedKeys}
      className={className}
      style={{ width: "100%", ...style }}
      {...restProps}
    />
  );
}

/**
 * 带顶级部门选项的变体（用于部门编辑模态框）
 */
export interface DepartmentTreeSelectWithTopProps extends Omit<TreeSelectProps, "treeData"> {
  value?: string;
  onChange?: (value: string) => void;
  departments?: Department[];
  treeData?: TreeNode[];  // 直接传递已转换的 TreeSelect 格式数据
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
  className?: string;
  loading?: boolean;
  showTopLevel?: boolean;
  topLevelLabel?: string;
}

export function DepartmentTreeSelectWithTop({
  value,
  onChange,
  departments,
  treeData,
  placeholder = "请选择上级部门",
  allowClear = true,
  disabled = false,
  loading = false,
  style,
  className,
  showTopLevel = true,
  topLevelLabel = "顶级部门",
  ...restProps
}: DepartmentTreeSelectWithTopProps) {
  // 优先使用 treeData（已转换的格式），否则使用 departments 并转换
  const baseTreeData = treeData || (departments ? convertDeptTreeData(departments) : []);

  const finalTreeData = showTopLevel
    ? [
        {
          title: topLevelLabel,
          value: "",
          key: "",
          children: baseTreeData,
        },
      ]
    : baseTreeData;

  // 获取默认展开的 keys
  let defaultExpandedKeys: string[] = [];
  if (showTopLevel) {
    defaultExpandedKeys = [""];
  } else if (treeData) {
    defaultExpandedKeys = getFirstLevelKeysFromTreeData(treeData);
  } else if (departments) {
    defaultExpandedKeys = getFirstLevelKeysFromDepartments(departments);
  }

  return (
    <TreeSelect
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      allowClear={allowClear}
      disabled={disabled || loading}
      loading={loading}
      treeData={finalTreeData}
      showSearch
      treeLine={{ showLeafIcon: false }}
      treeDefaultExpandedKeys={defaultExpandedKeys}
      className={className}
      style={{ width: "100%", ...style }}
      {...restProps}
    />
  );
}

export default DepartmentTreeSelect;

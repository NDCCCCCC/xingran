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
 *
 * 受控组件(D-LOCKED, Phase 37 CONTEXT):
 * - 数据通过 props `departments: SimpleDept[]` 或 `treeData: TreeNode[]` 由调用方喂入。
 * - **禁止**在本组件内调数据获取 hook (职责正交: 调用方负责数据获取, 本组件只负责渲染)。
 *
 * 类型(D-LOCKED):
 * - 不再定义本地 `Department` 接口, 统一引用 canonical `SimpleDept` (来自 `@/lib/dutyApi`)。
 * - 树转换不再用本地旧转换函数, 改调用 `deptUtils.toFullPathTree`。
 */

import { TreeSelect } from "antd";
import type { TreeSelectProps } from "antd/es/tree-select";
import type { SimpleDept } from "@/lib/dutyApi";
import { toFullPathTree } from "@/utils/deptUtils";

// TreeSelect 原生格式（已转换的数据）
export interface TreeNode {
  title: string;
  value: string;
  key: string;
  children?: TreeNode[];
  isExternalOrg?: number;
}

/**
 * 向后兼容 alias:本地旧 Department 接口已删除(canonical 类型为 `SimpleDept`)。
 * 保留 re-export 让 floors 等下游消费方在批 2 迁移前继续编译通过。
 * 消费方应逐步迁移到直接 import `SimpleDept`。
 *
 * @deprecated 请改 import `SimpleDept` from `@/lib/dutyApi`。本 alias 将在批 2/后续清理时移除。
 */
export type Department = SimpleDept;

export interface DepartmentTreeSelectProps extends Omit<TreeSelectProps, "treeData"> {
  value?: string;
  onChange?: (value: string) => void;
  departments?: SimpleDept[];
  treeData?: TreeNode[];  // 直接传递已转换的 TreeSelect 格式数据
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
  className?: string;
  loading?: boolean;
}

/**
 * 获取第一级节点的 key（用于默认展开）- 标准部门格式
 */
function getFirstLevelKeysFromDepartments(departments: SimpleDept[]): string[] {
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
 *
 * 行为保持(D-LOCKED):
 * - `toFullPathTree({ startFromLevel: 2 })` 复现旧全路径转换函数的 `currentPath.slice(1)`
 *   语义——从二级开始拼全路径, 顶级祖先名不进 title。
 * - 保持受控模式 (value/onChange + 外部 departments/treeData)。
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
  // 优先使用 treeData（已转换的格式），否则使用 departments 并通过 toFullPathTree 转换。
  // startFromLevel=2 复现旧全路径转换函数的 slice(1) 行为, 保证 UI 文案不变。
  const finalTreeData = treeData ||
    (departments ? toFullPathTree(departments, { startFromLevel: 2 }) : []);

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
  departments?: SimpleDept[];
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
  // 行为保持: 同 DepartmentTreeSelect, startFromLevel=2 复现旧全路径转换函数语义。
  const baseTreeData = treeData ||
    (departments ? toFullPathTree(departments, { startFromLevel: 2 }) : []);

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

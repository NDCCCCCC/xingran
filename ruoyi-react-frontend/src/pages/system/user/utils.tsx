/**
 * User 工具函数
 */

import type { ReactElement } from "react";
import { Select } from "antd";
import type { DictItem } from "@/hooks/useDict";
import { GENDER_OPTIONS, STATUS_TAG_CONFIG } from "./constants";

const { Option } = Select;

/**
 * 渲染树形部门选项（用于普通 Select）
 *
 * 注意:本函数是 Select 专用渲染(渲染 `<Option>` 树形缩进),
 * **不是** TreeSelect DataNode 转换,语义独立,Phase 37 收敛时保留不动。
 * 如需 TreeSelect DataNode 转换请使用 `@/utils/deptUtils` 的 `toShortNameDataNode`(短名)或 `toFullPathTree`(全路径)。
 */
export function renderDeptTreeOptions(
  departments: { id: string; deptName: string; children?: unknown[] }[]
): ReactElement[] {
  if (!departments || departments.length === 0) {
    return [
      <Option key="" value="">
        暂无部门数据
      </Option>,
    ];
  }

  // 递归渲染树形选项
  const renderOptions = (
    nodes: { id: string; deptName: string; children?: unknown[] }[],
    level = 0
  ): ReactElement[] => {
    const options: ReactElement[] = [];

    nodes.forEach((node) => {
      options.push(
        <Option key={node.id} value={node.id}>
          {"　".repeat(level * 2)}
          {node.deptName}
        </Option>
      );

      // 递归渲染子部门
      if (node.children && node.children.length > 0) {
        options.push(
          ...renderOptions(
            node.children as { id: string; deptName: string; children?: unknown[] }[],
            level + 1
          )
        );
      }
    });

    return options;
  };

  return renderOptions(departments);
}

/**
 * 格式化性别显示
 *
 * Phase 69 DICT-03: 优先用页面传入的 useDict("sys_user_sex") 字典数据渲染 label
 * （字典管理页改 label 后表格随之变化）；字典空态/未命中时回退静态 GENDER_OPTIONS 映射。
 * 本函数非 hook 上下文，字典数据由调用方（index.tsx）透传。
 */
export function formatGender(gender: number, dict?: DictItem[]): string {
  if (dict && dict.length > 0) {
    const dictItem = dict.find((d) => d.dictValue === String(gender));
    if (dictItem?.dictLabel) {
      return dictItem.dictLabel;
    }
  }
  const option = GENDER_OPTIONS.find((opt) => opt.value === gender);
  return option?.label || "保密";
}

/**
 * 格式化状态显示
 */
export function formatStatus(status: number): { text: string; color: string } {
  return STATUS_TAG_CONFIG[status] || { text: "未知", color: "default" };
}

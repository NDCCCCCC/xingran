/**
 * Role Utilities
 * 角色管理工具函数
 */

import { Space, Tag } from "antd";
import { SafetyCertificateOutlined } from "@ant-design/icons";
import type { TreeDataNode } from "antd";
import type { Key } from "antd/es/table/interface";
import { formatDateTime } from "@/utils/datetime";
import { ModernTag } from "@/components/shared/ModernTag";

/** 处理树形数据 */
export function processTreeData(
  nodes: Record<string, unknown>[],
  keyField: string = "id",
  titleField: string = "title"
): TreeDataNode[] {
  return nodes.map((node) => ({
    key: (node.key as string) || (node[keyField] as string),
    title: (node.title as string) || (node[titleField] as string),
    value: (node.value as string) || (node[keyField] as string),
    children:
      node.children && Array.isArray(node.children) && node.children.length > 0
        ? processTreeData(node.children as Record<string, unknown>[], keyField, titleField)
        : undefined,
  }));
}

/** 渲染状态标签 */
export function renderStatusTag(status: number) {
  return (
    <ModernTag status={status === 0 ? "success" : "error"}>
      {status === 0 ? "正常" : "停用"}
    </ModernTag>
  );
}

/** 渲染角色名称 */
export function renderRoleName(text: string) {
  return (
    <Space>
      <SafetyCertificateOutlined />
      {text}
    </Space>
  );
}

/** 渲染权限字符标签 */
export function renderRoleKeyTag(text: string) {
  return <Tag color="blue">{text}</Tag>;
}

/** 格式化时间为本地字符串 */
export function formatLocalTime(time: string) {
  return formatDateTime(time);
}

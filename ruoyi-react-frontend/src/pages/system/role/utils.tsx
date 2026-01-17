/**
 * Role Utilities
 * 角色管理工具函数
 */

import { Tag, Space } from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import type { TreeDataNode } from 'antd';
import type { Key } from 'antd/es/table/interface';
import { formatDateTime } from '@/utils/datetime';

/** 处理树形数据 */
export function processTreeData(nodes: any[], keyField: string = 'id', titleField: string = 'title'): TreeDataNode[] {
  return nodes.map((node: any) => ({
    key: node.key || node[keyField],
    title: node.title || node[titleField],
    value: node.value || node[keyField],
    children: node.children && node.children.length > 0 ? processTreeData(node.children, keyField, titleField) : undefined,
  }));
}

/** 渲染状态标签 */
export function renderStatusTag(status: number) {
  const color = status === 0 ? 'success' : 'error';
  const text = status === 0 ? '正常' : '停用';
  return <Tag color={color}>{text}</Tag>;
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

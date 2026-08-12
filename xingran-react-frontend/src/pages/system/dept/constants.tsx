/**
 * Department Constants
 * 部门页面常量定义
 */

import { Tag } from "antd";

/** 状态选项 */
export const STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
] as const;

/** 外部机构选项 */
export const EXTERNAL_ORG_OPTIONS = [
  { label: "否", value: 0 },
  { label: "是", value: 1 },
] as const;

/** 渲染状态标签 */
export function renderStatusTag(status: number) {
  return (
    <Tag color={status === 0 ? "success" : "error"}>
      {status === 0 ? "正常" : "停用"}
    </Tag>
  );
}

/** 渲染外部机构标签 */
export function renderExternalOrgTag(isExternalOrg: number) {
  return (
    <Tag color={isExternalOrg === 1 ? "blue" : "default"}>
      {isExternalOrg === 1 ? "是" : "否"}
    </Tag>
  );
}

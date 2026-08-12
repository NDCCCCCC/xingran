/**
 * Knowledge Article Constants
 * 知识文章页面常量定义
 */

import { Tag } from "antd";
import { KnowledgeArticleStatus } from "@/lib/knowledgeApi";

/** 文章状态配置 */
export const STATUS_CONFIG = {
  [KnowledgeArticleStatus.Draft]: { text: "草稿", color: "default" },
  [KnowledgeArticleStatus.Published]: { text: "已发布", color: "green" },
} as const;

/** 文章状态选项 */
export const STATUS_OPTIONS = [
  { label: "草稿", value: KnowledgeArticleStatus.Draft },
  { label: "已发布", value: KnowledgeArticleStatus.Published },
] as const;

/** 渲染状态标签 */
export function renderStatusTag(status: KnowledgeArticleStatus) {
  const config = STATUS_CONFIG[status];
  return <Tag color={config?.color}>{config?.text}</Tag>;
}

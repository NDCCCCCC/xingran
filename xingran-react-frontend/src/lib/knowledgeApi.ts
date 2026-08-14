import { post } from "./api";
import type { BaseResponse, PageResponse } from "@/types";

// ==================== 类型定义 ====================

// 知识库文章状态枚举
export enum KnowledgeArticleStatus {
  Draft = 0, // 草稿
  Published = 1, // 已发布
}

// ==================== 知识库文章相关类型 ====================

export interface KnowledgeArticle {
  id: string;
  title: string;
  content: string;
  summary?: string;
  categoryId: string;
  status: KnowledgeArticleStatus;
  viewCount: number;
  likeCount: number;
  sourceWorkOrderId?: string;
  // 关联
  category?: KnowledgeCategory;
  tags?: KnowledgeTag[];
  sourceWorkOrder?: {
    id: string;
    title: string;
    workOrderNo?: string;
  } | null;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface KnowledgeCategory {
  id: string;
  categoryName: string;
  description?: string;
  parentId?: string;
  sortOrder: number;
  status: KnowledgeArticleStatus;
  parent?: KnowledgeCategory;
  children?: KnowledgeCategory[];
  articles?: KnowledgeArticle[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface KnowledgeTag {
  id: string;
  tagName: string;
  useCount: number;
  createdAt: string;
  articles?: KnowledgeArticle[];
}

// ==================== 请求参数类型 ====================

export interface KnowledgeArticleListRequest {
  current?: number;
  pageSize?: number;
  orderByColumn?: string;
  isAsc?: boolean;
  title?: string;
  categoryId?: string;
  tagId?: string;
  status?: number;
  createdBy?: string;
}

export interface KnowledgeArticleCreateRequest {
  title: string;
  content: string;
  summary?: string;
  categoryId: string;
  status?: number;
  tagIds?: string[];
  sourceWorkOrderId?: string;
}

export interface KnowledgeArticleUpdateRequest {
  title?: string;
  content?: string;
  summary?: string;
  categoryId?: string;
  status?: number;
  tagIds?: string[];
}

export interface KnowledgeCategoryListRequest {
  parentId?: string;
  status?: number;
}

export interface KnowledgeCategoryCreateRequest {
  categoryName: string;
  description?: string;
  parentId?: string;
  sortOrder?: number;
  status?: number;
}

export interface KnowledgeCategoryUpdateRequest {
  categoryName?: string;
  description?: string;
  parentId?: string;
  sortOrder?: number;
  status?: number;
}

export interface SearchKnowledgeRequest {
  keyword?: string;
  categoryId?: string;
  tagId?: string;
}

export interface ConvertWorkOrderToArticleRequest {
  title: string;
  content: string;
  summary?: string;
  categoryId: string;
  tagIds?: string[];
  status?: number;
}

// ==================== 知识库文章 API ====================

export function getKnowledgeArticleList(
  params: KnowledgeArticleListRequest
): Promise<BaseResponse<PageResponse<KnowledgeArticle>>> {
  return post("/knowledge/articles/list", params);
}

/** 知识库文章统计（总数 / 草稿 / 已发布 / 累计浏览 / 累计点赞） */
export interface KnowledgeArticleStatistics {
  total: number;
  draft: number;
  published: number;
  totalViews: number;
  totalLikes: number;
}

/**
 * 获取知识库文章统计（后端 COUNT 聚合，不受分页影响）
 */
export function getKnowledgeArticleStatistics(): Promise<BaseResponse<KnowledgeArticleStatistics>> {
  return post("/knowledge/articles/statistics", {});
}

export function getKnowledgeArticle(id: string): Promise<BaseResponse<KnowledgeArticle>> {
  return post(`/knowledge/articles/${id}`, {});
}

export function createKnowledgeArticle(
  data: KnowledgeArticleCreateRequest
): Promise<BaseResponse<KnowledgeArticle>> {
  return post("/knowledge/articles", data);
}

export function updateKnowledgeArticle(
  id: string,
  data: KnowledgeArticleUpdateRequest
): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/articles/${id}/update`, data);
}

export function deleteKnowledgeArticle(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/articles/${id}/delete`);
}

export function searchKnowledgeArticles(
  data: SearchKnowledgeRequest
): Promise<BaseResponse<{ list: KnowledgeArticle[]; total: number }>> {
  return post("/knowledge/articles/search", data);
}

export function likeKnowledgeArticle(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/articles/${id}/like`);
}

// ==================== 知识库分类 API ====================

export function getKnowledgeCategoryList(
  params?: KnowledgeCategoryListRequest
): Promise<BaseResponse<KnowledgeCategory[]>> {
  return post("/knowledge/categories/list", params || {});
}

export function getKnowledgeCategory(id: string): Promise<BaseResponse<KnowledgeCategory>> {
  return post(`/knowledge/categories/${id}`, {});
}

export function createKnowledgeCategory(
  data: KnowledgeCategoryCreateRequest
): Promise<BaseResponse<KnowledgeCategory>> {
  return post("/knowledge/categories", data);
}

export function updateKnowledgeCategory(
  id: string,
  data: KnowledgeCategoryUpdateRequest
): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/categories/${id}/update`, data);
}

export function deleteKnowledgeCategory(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/categories/${id}/delete`);
}

// ==================== 知识库标签 API ====================

export function getAllKnowledgeTags(): Promise<BaseResponse<KnowledgeTag[]>> {
  return post("/knowledge/tags/all", {});
}

export function createKnowledgeTag(data: { tagName: string }): Promise<BaseResponse<KnowledgeTag>> {
  return post("/knowledge/tags", data);
}

export function updateKnowledgeTag(
  id: string,
  data: { tagName: string }
): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/tags/${id}/update`, data);
}

export function deleteKnowledgeTag(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/knowledge/tags/${id}/delete`);
}

// ==================== 工单转知识库 API ====================

export function convertWorkOrderToArticle(
  workOrderId: string,
  data: ConvertWorkOrderToArticleRequest
): Promise<BaseResponse<KnowledgeArticle>> {
  return post(`/knowledge/workorders/${workOrderId}`, data);
}

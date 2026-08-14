/**
 * Knowledge Article Data Hook
 * 知识文章数据管理 Hook
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import { App } from "antd";
import {
  getKnowledgeArticleList,
  getKnowledgeArticleStatistics,
  createKnowledgeArticle,
  updateKnowledgeArticle,
  deleteKnowledgeArticle,
  likeKnowledgeArticle,
  type KnowledgeArticle,
  KnowledgeArticleStatus,
} from "@/lib/knowledgeApi";
import { getKnowledgeCategoryList, type KnowledgeCategory } from "@/lib/knowledgeApi";
import { getAllKnowledgeTags, type KnowledgeTag } from "@/lib/knowledgeApi";
import type { UnknownError } from "@/types/common";

export interface ArticleStatistics {
  total: number;
  draft: number;
  published: number;
  totalViews: number;
  totalLikes: number;
}

export interface UseArticleDataParams {
  current: number;
  pageSize: number;
}

export interface UseArticleDataReturn {
  articles: KnowledgeArticle[];
  categories: KnowledgeCategory[];
  tags: KnowledgeTag[];
  flatCategories: KnowledgeCategory[];
  loading: boolean;
  total: number;
  statistics: ArticleStatistics;

  setArticles: React.Dispatch<React.SetStateAction<KnowledgeArticle[]>>;
  setCategories: React.Dispatch<React.SetStateAction<KnowledgeCategory[]>>;
  setTags: React.Dispatch<React.SetStateAction<KnowledgeTag[]>>;

  fetchList: (
    page?: number,
    pageSize?: number,
    orderByColumn?: string,
    isAsc?: boolean
  ) => Promise<void>;
  fetchCategories: () => Promise<void>;
  fetchTags: () => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  handleLike: (id: string) => Promise<void>;
  handlePublish: (record: KnowledgeArticle) => Promise<void>;
  handleSave: (
    editingRecord: KnowledgeArticle | null,
    values: Record<string, unknown>
  ) => Promise<void>;
}

export function useArticleData(params: UseArticleDataParams): UseArticleDataReturn {
  const { current, pageSize } = params;

  const { message } = App.useApp();
  const [articles, setArticles] = useState<KnowledgeArticle[]>([]);
  const [categories, setCategories] = useState<KnowledgeCategory[]>([]);
  const [tags, setTags] = useState<KnowledgeTag[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  const [statistics, setStatistics] = useState<ArticleStatistics>({
    total: 0,
    draft: 0,
    published: 0,
    totalViews: 0,
    totalLikes: 0,
  });

  // 扁平化分类树
  const flatCategories = useMemo(() => {
    const flatten = (cats: KnowledgeCategory[]): KnowledgeCategory[] => {
      const result: KnowledgeCategory[] = [];
      cats.forEach((cat) => {
        result.push(cat);
        if (cat.children && cat.children.length > 0) {
          result.push(...flatten(cat.children));
        }
      });
      return result;
    };
    return flatten(categories);
  }, [categories]);

  // 获取统计数据（专用端点 COUNT 聚合，全局计数，不受分页/筛选影响）。
  // 旧实现用当前页 list 算 total/draft/published/totalViews/totalLikes，多页时严重偏小。
  const fetchStats = useCallback(async () => {
    try {
      const result = await getKnowledgeArticleStatistics();
      setStatistics({
        total: result.data?.total ?? 0,
        draft: result.data?.draft ?? 0,
        published: result.data?.published ?? 0,
        totalViews: result.data?.totalViews ?? 0,
        totalLikes: result.data?.totalLikes ?? 0,
      });
    } catch (error) {
      console.error("获取文章统计失败:", error);
    }
  }, []);

  const fetchList = useCallback(
    async (page?: number, pageSize?: number, orderByColumn?: string, isAsc?: boolean) => {
      setLoading(true);
      try {
        const result = await getKnowledgeArticleList({
          current: page ?? current,
          pageSize: pageSize ?? pageSize,
          ...(orderByColumn ? { orderByColumn, isAsc } : {}),
        });
        const list = result.data?.list ?? [];
        setArticles(list);
        setTotal(result.data?.total ?? 0);

        // 列表加载后顺带刷新统计（全局 COUNT）；搜索/分页/增删改均经 fetchList，统计始终为真实全局计数。
        fetchStats();
      } catch (error) {
        message.error("获取文章列表失败");
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize, fetchStats]
  );

  const fetchCategories = useCallback(async () => {
    try {
      const result = await getKnowledgeCategoryList();
      setCategories(result.data || []);
    } catch (error) {
      console.error("获取分类列表失败:", error);
    }
  }, []);

  const fetchTags = useCallback(async () => {
    try {
      const result = await getAllKnowledgeTags();
      setTags(result.data || []);
    } catch (error) {
      console.error("获取标签列表失败:", error);
    }
  }, []);

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteKnowledgeArticle(id);
        message.success("删除成功");
        fetchList();
      } catch (error: unknown) {
        const err = error as UnknownError;
        message.error(err.message || "删除失败");
      }
    },
    [fetchList]
  );

  const handleLike = useCallback(
    async (id: string) => {
      try {
        await likeKnowledgeArticle(id);
        message.success("点赞成功");
        fetchList();
      } catch (error: unknown) {
        const err = error as UnknownError;
        message.error(err.message || "点赞失败");
      }
    },
    [fetchList]
  );

  const handlePublish = useCallback(
    async (record: KnowledgeArticle) => {
      try {
        await updateKnowledgeArticle(record.id, { status: KnowledgeArticleStatus.Published });
        message.success("发布成功");
        fetchList();
      } catch (error: unknown) {
        const err = error as UnknownError;
        message.error(err.message || "发布失败");
      }
    },
    [fetchList]
  );

  const handleSave = useCallback(
    async (editingRecord: KnowledgeArticle | null, values: Record<string, unknown>) => {
      const articleData = {
        title: values.title as string,
        content: values.content as string,
        summary: values.summary as string | undefined,
        categoryId: values.categoryId as string,
        status: values.status as number | undefined,
        tagIds: values.tagIds as string[] | undefined,
      };
      if (editingRecord) {
        await updateKnowledgeArticle(editingRecord.id, articleData);
        message.success("更新成功");
      } else {
        await createKnowledgeArticle(articleData);
        message.success("创建成功");
      }
      fetchList();
    },
    [fetchList]
  );

  return {
    articles,
    categories,
    tags,
    flatCategories,
    loading,
    total,
    statistics,
    setArticles,
    setCategories,
    setTags,
    fetchList,
    fetchCategories,
    fetchTags,
    handleDelete,
    handleLike,
    handlePublish,
    handleSave,
  };
}

/**
 * Knowledge Article Data Hook
 * 知识文章数据管理 Hook
 */

import { useState, useCallback, useEffect, useMemo } from 'react';
import { message } from 'antd';
import {
  getKnowledgeArticleList,
  createKnowledgeArticle,
  updateKnowledgeArticle,
  deleteKnowledgeArticle,
  likeKnowledgeArticle,
  type KnowledgeArticle,
  KnowledgeArticleStatus,
} from '@/lib/knowledgeApi';
import {
  getKnowledgeCategoryList,
  type KnowledgeCategory,
} from '@/lib/knowledgeApi';
import {
  getAllKnowledgeTags,
  type KnowledgeTag,
} from '@/lib/knowledgeApi';

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

  fetchList: (page?: number, pageSize?: number) => Promise<void>;
  fetchCategories: () => Promise<void>;
  fetchTags: () => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  handleLike: (id: string) => Promise<void>;
  handlePublish: (record: KnowledgeArticle) => Promise<void>;
  handleSave: (editingRecord: KnowledgeArticle | null, values: any) => Promise<void>;
}

export function useArticleData(params: UseArticleDataParams): UseArticleDataReturn {
  const { current, pageSize } = params;

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

  const fetchList = useCallback(async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const result = await getKnowledgeArticleList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
      });
      const list = result.data?.list ?? [];
      setArticles(list);
      setTotal(result.data?.total ?? 0);

      // 计算统计数据
      setStatistics({
        total: list.length,
        draft: list.filter((a) => a.status === KnowledgeArticleStatus.Draft).length,
        published: list.filter((a) => a.status === KnowledgeArticleStatus.Published).length,
        totalViews: list.reduce((sum, a) => sum + (a.viewCount || 0), 0),
        totalLikes: list.reduce((sum, a) => sum + (a.likeCount || 0), 0),
      });
    } catch (error) {
      message.error('获取文章列表失败');
    } finally {
      setLoading(false);
    }
  }, [current, pageSize]);

  const fetchCategories = useCallback(async () => {
    try {
      const result = await getKnowledgeCategoryList();
      setCategories(result.data || []);
    } catch (error) {
      console.error('获取分类列表失败:', error);
    }
  }, []);

  const fetchTags = useCallback(async () => {
    try {
      const result = await getAllKnowledgeTags();
      setTags(result.data || []);
    } catch (error) {
      console.error('获取标签列表失败:', error);
    }
  }, []);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteKnowledgeArticle(id);
      message.success('删除成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '删除失败');
    }
  }, [fetchList]);

  const handleLike = useCallback(async (id: string) => {
    try {
      await likeKnowledgeArticle(id);
      message.success('点赞成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '点赞失败');
    }
  }, [fetchList]);

  const handlePublish = useCallback(async (record: KnowledgeArticle) => {
    try {
      await updateKnowledgeArticle(record.id, { status: KnowledgeArticleStatus.Published });
      message.success('发布成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '发布失败');
    }
  }, [fetchList]);

  const handleSave = useCallback(async (editingRecord: KnowledgeArticle | null, values: any) => {
    if (editingRecord) {
      await updateKnowledgeArticle(editingRecord.id, {
        title: values.title,
        content: values.content,
        summary: values.summary,
        categoryId: values.categoryId,
        status: values.status,
        tagIds: values.tagIds,
      });
      message.success('更新成功');
    } else {
      await createKnowledgeArticle({
        title: values.title,
        content: values.content,
        summary: values.summary,
        categoryId: values.categoryId,
        status: values.status,
        tagIds: values.tagIds,
      });
      message.success('创建成功');
    }
    fetchList();
  }, [fetchList]);

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

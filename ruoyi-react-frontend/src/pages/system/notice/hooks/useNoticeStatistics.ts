import { useState, useCallback } from "react";
import { getNoticeList } from "@/lib/noticeApi";

export interface NoticeStatistics {
  total: number;
  published: number;
  draft: number;
  scheduled: number;
}

interface UseNoticeStatisticsResult {
  statistics: NoticeStatistics;
  loadStatistics: () => Promise<void>;
}

/**
 * 通知统计管理 Hook
 * 处理通知统计数据的加载
 */
export function useNoticeStatistics(): UseNoticeStatisticsResult {
  const [statistics, setStatistics] = useState<NoticeStatistics>({
    total: 0,
    published: 0,
    draft: 0,
    scheduled: 0,
  });

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await getNoticeList({ current: 1, pageSize: 1000 });
      const allNotices = result.data?.list || [];
      const publishedCount = allNotices.filter((n) => (n as { publishStatus: number }).publishStatus === 1).length;
      const draftCount = allNotices.filter((n) => {
        const status = (n as { publishStatus: number }).publishStatus;
        return status === 0 || status === 2;
      }).length;
      const scheduledCount = allNotices.filter((n) => (n as { publishStatus: number }).publishStatus === 3).length;
      setStatistics({
        total: allNotices.length,
        published: publishedCount,
        draft: draftCount,
        scheduled: scheduledCount,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  return {
    statistics,
    loadStatistics,
  };
}

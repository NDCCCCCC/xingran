import { useState, useCallback } from "react";
import { getNoticeStatusStatistics } from "@/lib/noticeApi";

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

  // 加载统计数据: 调用专用统计端点(COUNT 聚合),不再用列表长度充当总数——
  // 后端 notice list 同样受 MaxPageSize 上限影响,通知数超过后会截断。
  // 同时修正旧实现的状态桶错误: scheduled 应为 publishStatus=2(定时发布),
  // 旧代码误用 3(已撤回); draft 应为 0(草稿),旧代码误把 2(定时)也算进 draft。
  const loadStatistics = useCallback(async () => {
    try {
      const result = await getNoticeStatusStatistics();
      setStatistics({
        total: result.data?.total ?? 0,
        published: result.data?.published ?? 0,
        draft: result.data?.draft ?? 0,
        scheduled: result.data?.scheduled ?? 0,
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

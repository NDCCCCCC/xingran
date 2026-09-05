/**
 * 恢复任务轮询 Hook (Phase 93 D-16)
 *
 * taskId 非空时立即查询一次任务详情，pending/running 状态每 3s 续查，
 * success/failed 终态自动停止；taskId 置 null/切换时任务态自动失效，
 * unmount 时清理 timer。不引入 WebSocket 推送（D-16 前端轮询方案）。
 *
 * 实现注记：task 状态只在异步回调中写入（effect 同步路径零 setState，
 * react-hooks cascading-render 规则）；对外暴露的 task/polling 由
 * taskId 与原始 task 派生——taskId 置 null 或切换时旧任务自动不可见。
 */

import { useEffect, useRef, useState } from "react";
import { get } from "@/lib/api";
import type { ConfigRestoreTask, RestoreTaskStatus } from "../types";

const POLL_INTERVAL_MS = 3000;

const isTerminal = (status: RestoreTaskStatus | undefined): boolean =>
  status === "success" || status === "failed";

export interface UseRestoreTaskResult {
  task: ConfigRestoreTask | null;
  polling: boolean;
}

export function useRestoreTask(taskId: string | null): UseRestoreTaskResult {
  const [task, setTask] = useState<ConfigRestoreTask | null>(null);
  // 记录最近一次成功查询的任务 id：taskId 切换后旧任务自动失效（派生过滤）
  const fetchedIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (!taskId) {
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | null = null;

    const fetchOnce = async () => {
      try {
        const result = await get<ConfigRestoreTask>(`/network/backups/restore-tasks/${taskId}`);
        if (cancelled) {
          return;
        }
        fetchedIdRef.current = taskId;
        setTask(result.data);
        if (isTerminal(result.data?.status) && timer) {
          // 终态：停止后续轮询（D-16）
          clearInterval(timer);
          timer = null;
        }
      } catch (error) {
        // 单次查询失败不中断轮询（后端瞬时错误/网络抖动），仅记录
        console.error("查询恢复任务失败:", error);
      }
    };

    void fetchOnce();
    timer = setInterval(() => {
      void fetchOnce();
    }, POLL_INTERVAL_MS);

    return () => {
      cancelled = true;
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
    };
  }, [taskId]);

  // 派生态：仅当任务 id 与当前 taskId 匹配时可见（taskId 置 null/切换 → null）
  const activeTask = task && taskId && task.id === taskId ? task : null;
  const polling = taskId !== null && !isTerminal(activeTask?.status);

  return { task: activeTask, polling };
}

/**
 * 备份差异对比 Hook
 */

import { useState, useCallback, useRef, type UIEvent } from "react";
import type { ConfigBackup } from "@/types";
import type { DiffResult } from "../types";
import { computeDiff } from "../utils";
import { post } from "@/lib/api";

interface UseBackupDiffReturn {
  diffModalVisible: boolean;
  diffResult: DiffResult | null;
  compareBackup1: ConfigBackup | null;
  compareBackup2: ConfigBackup | null;
  leftScrollRef: React.RefObject<HTMLDivElement | null>;
  rightScrollRef: React.RefObject<HTMLDivElement | null>;
  openDiffModal: (backup1: ConfigBackup, backup2: ConfigBackup) => Promise<void>;
  closeDiffModal: () => void;
  handleLeftScroll: (e: UIEvent<HTMLDivElement>) => void;
  handleRightScroll: (e: UIEvent<HTMLDivElement>) => void;
}

export function useBackupDiff(): UseBackupDiffReturn {
  const [diffModalVisible, setDiffModalVisible] = useState(false);
  const [diffResult, setDiffResult] = useState<DiffResult | null>(null);
  const [compareBackup1, setCompareBackup1] = useState<ConfigBackup | null>(null);
  const [compareBackup2, setCompareBackup2] = useState<ConfigBackup | null>(null);

  // 同步滚动的 refs
  const leftScrollRef = useRef<HTMLDivElement>(null);
  const rightScrollRef = useRef<HTMLDivElement>(null);
  const isLeftScrolling = useRef(false);
  const isRightScrolling = useRef(false);

  // 打开差异对比弹窗
  const openDiffModal = useCallback(async (backup1: ConfigBackup, backup2: ConfigBackup) => {
    try {
      // 获取两个备份的内容
      const [result1, result2] = await Promise.all([
        post<{ content: string }>("/network/backups/content", { id: backup1.id }),
        post<{ content: string }>("/network/backups/content", { id: backup2.id }),
      ]);

      const content1 = result1.data?.content || "";
      const content2 = result2.data?.content || "";

      const diff = computeDiff(content1, content2);
      diff.oldVersion = String(backup1.version ?? backup1.createdAt ?? "");
      diff.newVersion = String(backup2.version ?? backup2.createdAt ?? "");

      setDiffResult(diff);
      setCompareBackup1(backup1);
      setCompareBackup2(backup2);
      setDiffModalVisible(true);
    } catch (error) {
      console.error("加载备份内容失败:", error);
    }
  }, []);

  // 关闭差异对比弹窗
  const closeDiffModal = useCallback(() => {
    setDiffModalVisible(false);
    setDiffResult(null);
    setCompareBackup1(null);
    setCompareBackup2(null);
  }, []);

  // 左侧滚动处理
  const handleLeftScroll = useCallback((e: UIEvent<HTMLDivElement>) => {
    if (isLeftScrolling.current) return;
    isRightScrolling.current = true;
    const target = e.target as HTMLDivElement;
    if (rightScrollRef.current) {
      rightScrollRef.current.scrollTop = target.scrollTop;
    }
    setTimeout(() => {
      isRightScrolling.current = false;
    }, 0);
  }, []);

  // 右侧滚动处理
  const handleRightScroll = useCallback((e: UIEvent<HTMLDivElement>) => {
    if (isRightScrolling.current) return;
    isLeftScrolling.current = true;
    const target = e.target as HTMLDivElement;
    if (leftScrollRef.current) {
      leftScrollRef.current.scrollTop = target.scrollTop;
    }
    setTimeout(() => {
      isLeftScrolling.current = false;
    }, 0);
  }, []);

  return {
    diffModalVisible,
    diffResult,
    compareBackup1,
    compareBackup2,
    leftScrollRef,
    rightScrollRef,
    openDiffModal,
    closeDiffModal,
    handleLeftScroll,
    handleRightScroll,
  };
}

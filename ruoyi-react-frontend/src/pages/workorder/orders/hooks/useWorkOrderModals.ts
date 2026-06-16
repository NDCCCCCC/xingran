/**
 * WorkOrder 模态框和抽屉状态管理 Hook
 * ✅ 优化：移除动态导入，在顶层导入 API 函数
 */

import { useState, useCallback } from "react";
import type { WorkOrder } from "@/lib/workorderApi";
import type { FormInstance } from "antd/es/form";
// ✅ 优化：直接导入 API 函数，避免动态导入 (bundle-dynamic-imports)
import { getWorkOrderComments, getWorkOrderHistory } from "@/lib/workorderApi";

export interface Comment {
  id: string;
  content: string;
  isInternal: boolean;
  createdAt: string;
  user?: {
    id: string;
    nickName?: string;
    username?: string;
  };
}

export interface HistoryItem {
  id: string;
  action: string;
  remark?: string;
  oldValue?: string;
  newValue?: string;
  createdAt: string;
  operator?: {
    id: string;
    nickName?: string;
    username?: string;
  };
}

export interface UseWorkOrderModalsReturn {
  modalVisible: boolean;
  detailDrawerVisible: boolean;
  editingRecord: WorkOrder | null;
  selectedRecord: WorkOrder | null;
  comments: Comment[];
  history: HistoryItem[];
  commentInternal: boolean;
  setModalVisible: (visible: boolean) => void;
  setDetailDrawerVisible: (visible: boolean) => void;
  setEditingRecord: (record: WorkOrder | null) => void;
  setSelectedRecord: (record: WorkOrder | null) => void;
  setComments: (comments: Comment[]) => void;
  setHistory: (history: HistoryItem[]) => void;
  setCommentInternal: (internal: boolean) => void;
  openAddModal: () => void;
  openEditModal: (record: WorkOrder, editForm: FormInstance<unknown>) => void;
  openDetailDrawer: (record: WorkOrder) => Promise<void>;
  closeModals: () => void;
}

export function useWorkOrderModals(): UseWorkOrderModalsReturn {
  const [modalVisible, setModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<WorkOrder | null>(null);
  const [selectedRecord, setSelectedRecord] = useState<WorkOrder | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [commentInternal, setCommentInternal] = useState(false);

  // 打开新增模态框
  const openAddModal = useCallback(() => {
    setEditingRecord(null);
    setModalVisible(true);
  }, []);

  // 打开编辑模态框
  const openEditModal = useCallback((record: WorkOrder, editForm: FormInstance<unknown>) => {
    setEditingRecord(record);
    setModalVisible(true);
    // 在 modal 打开时设置表单值
    setTimeout(() => {
      editForm.setFieldsValue({
        title: record.title,
        categoryId: record.categoryId,
        type: record.type,
        priority: record.priority,
        description: record.description,
        solution: record.solution,
        deptId: record.deptId,
        assigneeId: record.assigneeId,
        expectedResolveAt: record.expectedResolveAt,
      });
    }, 0);
  }, []);

  // 打开详情抽屉
  const openDetailDrawer = useCallback(async (record: WorkOrder) => {
    setSelectedRecord(record);
    setDetailDrawerVisible(true);

    // 获取评论和历史
    try {
      // ✅ 优化：直接使用顶层导入的函数，移除动态导入 (bundle-dynamic-imports)
      const [commentsResult, historyResult] = await Promise.all([
        getWorkOrderComments(record.id),
        getWorkOrderHistory(record.id),
      ]);
      setComments(commentsResult.data || []);
      setHistory(historyResult.data || []);
    } catch (error) {
      console.error("获取评论和历史失败:", error);
    }
  }, []);

  // 关闭所有模态框
  const closeModals = useCallback(() => {
    setModalVisible(false);
    setDetailDrawerVisible(false);
    setEditingRecord(null);
    setSelectedRecord(null);
    setComments([]);
    setHistory([]);
  }, []);

  return {
    modalVisible,
    detailDrawerVisible,
    editingRecord,
    selectedRecord,
    comments,
    history,
    commentInternal,
    setModalVisible,
    setDetailDrawerVisible,
    setEditingRecord,
    setSelectedRecord,
    setComments,
    setHistory,
    setCommentInternal,
    openAddModal,
    openEditModal,
    openDetailDrawer,
    closeModals,
  };
}

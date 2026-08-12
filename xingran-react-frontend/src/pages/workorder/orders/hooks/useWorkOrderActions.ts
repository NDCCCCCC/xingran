/**
 * WorkOrder 操作 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type {
  WorkOrderType,
  WorkOrderPriority} from "@/lib/workorderApi";
import {
  createWorkOrder,
  updateWorkOrder,
  deleteWorkOrder,
  batchDeleteWorkOrders,
  assignToTodayDuty,
  updateWorkOrderStatus,
  addWorkOrderComment,
  getWorkOrderComments,
  type WorkOrder,
  type WorkOrderCreateRequest,
  type WorkOrderUpdateRequest
} from "@/lib/workorderApi";

export interface UseWorkOrderActionsOptions {
  fetchList: () => void;
  openDetailDrawer: (record: WorkOrder) => Promise<void>;
  selectedRecord: WorkOrder | null;
}

export interface UseWorkOrderActionsReturn {
  actionLoading: boolean;
  handleAdd: () => void;
  handleEdit: (record: WorkOrder) => void;
  handleDelete: (id: string) => Promise<void>;
  handleBatchDelete: (selectedRowKeys: React.Key[]) => Promise<void>;
  handleAssignToTodayDuty: (id: string) => Promise<void>;
  handleModalOk: (editForm: { validateFields: () => Promise<Record<string, unknown>> }, editingRecord: WorkOrder | null, setModalVisible: (visible: boolean) => void) => Promise<void>;
  handleAddComment: (commentForm: { validateFields: () => Promise<Record<string, unknown>>; resetFields: () => void }, commentInternal: boolean, setComments: (comments: unknown[]) => void) => Promise<void>;
  handleStatusChange: (status: number) => Promise<void>;
}

export function useWorkOrderActions(
  options: UseWorkOrderActionsOptions
): UseWorkOrderActionsReturn {
  const { fetchList, openDetailDrawer, selectedRecord } = options;
  const { message } = App.useApp();
  const [actionLoading, setActionLoading] = useState(false);

  const handleAdd = useCallback(() => {
    // 由 useWorkOrderModals 处理
  }, []);

  const handleEdit = useCallback((record: WorkOrder) => {
    // 由 useWorkOrderModals 处理
  }, []);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteWorkOrder(id);
      message.success("删除成功");
      fetchList();
    } catch (error) {
      message.error("删除失败");
    }
  }, [fetchList, message]);

  const handleBatchDelete = useCallback(async (selectedRowKeys: React.Key[]) => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工单");
      return;
    }
    try {
      await batchDeleteWorkOrders(selectedRowKeys as string[]);
      message.success("批量删除成功");
      fetchList();
    } catch (error) {
      message.error("批量删除失败");
    }
  }, [fetchList, message]);

  const handleAssignToTodayDuty = useCallback(async (id: string) => {
    try {
      const result = await assignToTodayDuty(id);
      // 后端返回成功，不管是否分配都显示成功消息
      if (result.data?.message?.includes("没有值班人员")) {
        // 没有值班人员，不分配，静默处理
        fetchList();
      } else {
        message.success("已分配给当天值班人员");
        fetchList();
      }
    } catch (error: unknown) {
      // 静默处理错误，不显示提示
      console.error("分配失败:", error);
      fetchList();
    }
  }, [fetchList, message]);

  const handleModalOk = useCallback(async (
    editForm: { validateFields: () => Promise<Record<string, unknown>> },
    editingRecord: WorkOrder | null,
    setModalVisible: (visible: boolean) => void
  ) => {
    try {
      const rawValues = await editForm.validateFields();
      const values = rawValues as {
        title: string;
        categoryId: string;
        type: WorkOrderType;
        priority: WorkOrderPriority;
        description?: string;
        solution?: string;
        deptId?: string;
        assigneeId?: string;
        expectedResolveAt?: string;
        isAutoAssigned?: boolean;
      };

      if (editingRecord) {
        const updateData: WorkOrderUpdateRequest = {
          title: values.title,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          description: values.description,
          solution: values.solution,
          deptId: values.deptId,
          assigneeId: values.assigneeId,
          expectedResolveAt: values.expectedResolveAt,
        };
        await updateWorkOrder(editingRecord.id, updateData);
        message.success("更新成功");
      } else {
        const createData: WorkOrderCreateRequest = {
          title: values.title,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          description: values.description,
          deptId: values.deptId,
          expectedResolveAt: values.expectedResolveAt,
          isAutoAssigned: values.isAutoAssigned,
          assigneeId: values.assigneeId,
        };
        await createWorkOrder(createData);
        message.success("创建成功");
      }
      setModalVisible(false);
      fetchList();
    } catch (error: unknown) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      message.error(editingRecord ? "更新失败" : "创建失败");
    }
  }, [fetchList, message]);

  const handleAddComment = useCallback(async (
    commentForm: { validateFields: () => Promise<Record<string, unknown>>; resetFields: () => void },
    commentInternal: boolean,
    setComments: (comments: unknown[]) => void
  ) => {
    try {
      const rawValues = await commentForm.validateFields();
      const values = rawValues as { content: string };
      if (!selectedRecord) return;

      await addWorkOrderComment(selectedRecord.id, {
        content: values.content,
        isInternal: commentInternal,
      });
      message.success("评论添加成功");
      commentForm.resetFields();

      // 重新获取评论列表
      const commentsResult = await getWorkOrderComments(selectedRecord.id);
      setComments(commentsResult.data || []);
    } catch (error) {
      message.error("评论添加失败");
    }
  }, [selectedRecord, message]);

  const handleStatusChange = useCallback(async (status: number) => {
    if (!selectedRecord) return;
    try {
      await updateWorkOrderStatus(selectedRecord.id, { status });
      message.success("状态更新成功");
      fetchList();
      // 重新获取工单详情
      await openDetailDrawer(selectedRecord);
    } catch (error: unknown) {
      if (error && typeof error === "object" && "message" in error && typeof error.message === "string") {
        message.error(error.message);
      } else {
        message.error("状态更新失败");
      }
    }
  }, [fetchList, openDetailDrawer, selectedRecord, message]);

  return {
    actionLoading,
    handleAdd,
    handleEdit,
    handleDelete,
    handleBatchDelete,
    handleAssignToTodayDuty,
    handleModalOk,
    handleAddComment,
    handleStatusChange,
  };
}

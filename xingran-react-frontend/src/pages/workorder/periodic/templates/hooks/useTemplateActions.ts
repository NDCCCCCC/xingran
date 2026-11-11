/**
 * Periodic Template Actions Hook
 * 周期性工单模板操作管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import type { WorkOrderType, PeriodicAssignType } from "@/lib/workorderApi";
import {
  createPeriodicTemplate,
  updatePeriodicTemplate,
  deletePeriodicTemplate,
  type PeriodicWorkOrderTemplate,
  type CreatePeriodicTemplateRequest,
  type UpdatePeriodicTemplateRequest,
} from "@/lib/workorderApi";

export interface UseTemplateActionsParams {
  onLoad: () => void;
}

export interface UseTemplateActionsReturn {
  // 编辑状态
  editingRecord: PeriodicWorkOrderTemplate | null;
  setEditingRecord: (record: PeriodicWorkOrderTemplate | null) => void;

  // 操作方法
  handleAdd: (editForm: FormInstance<unknown>) => void;
  handleEdit: (record: PeriodicWorkOrderTemplate, editForm: FormInstance<unknown>) => void;
  handleDelete: (id: string) => Promise<void>;
  handleSave: (editForm: FormInstance<unknown>) => Promise<void>;
}

export function useTemplateActions(params: UseTemplateActionsParams): UseTemplateActionsReturn {
  const { onLoad } = params;
  const { message } = App.useApp();

  const [editingRecord, setEditingRecord] = useState<PeriodicWorkOrderTemplate | null>(null);

  // 新增
  const handleAdd = useCallback((editForm: FormInstance<unknown>) => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      type: "fault",
      priority: 1,
      assignType: "duty_pool",
      notifyAssignee: true,
    });
  }, []);

  // 编辑
  const handleEdit = useCallback(
    (record: PeriodicWorkOrderTemplate, editForm: FormInstance<unknown>) => {
      setEditingRecord(record);
      setTimeout(() => {
        editForm.setFieldsValue({
          templateName: record.templateName,
          workOrderTitle: record.workOrderTitle,
          description: record.description,
          categoryId: record.categoryId,
          type: record.type,
          priority: record.priority,
          cronExpression: record.cronExpression,
          assignType: record.assignType,
          assignTargetId: record.assignTargetId,
          notifyAssignee: record.notifyAssignee,
        });
      }, 0);
    },
    []
  );

  // 删除
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deletePeriodicTemplate(id);
        message.success("删除成功");
        onLoad();
      } catch (error: unknown) {
        const err = error instanceof Error ? error.message : "删除失败";
        message.error(err);
      }
    },
    [onLoad, message]
  );

  // FIX broken-links-batch1：handleToggleEnabled / handleGenerateNow 已下线。
  // 它们分别调用 enable/disablePeriodicTemplate 与 generateWorkOrderNow，
  // 指向后端已废弃（注释、未注册）的路由，必然 404，故连同按钮一并移除。

  // 保存
  const handleSave = useCallback(
    async (editForm: FormInstance<unknown>) => {
      try {
        const values = (await editForm.validateFields()) as {
          templateName: string;
          workOrderTitle: string;
          description?: string;
          categoryId: string;
          type: string;
          priority: number;
          cronExpression: string;
          assignType: string;
          assignTargetId: string;
          notifyAssignee: boolean;
        };

        if (editingRecord) {
          const updateData: UpdatePeriodicTemplateRequest = {
            templateName: values.templateName,
            workOrderTitle: values.workOrderTitle,
            description: values.description,
            categoryId: values.categoryId,
            type: values.type as WorkOrderType,
            priority: values.priority,
            cronExpression: values.cronExpression,
            assignType: values.assignType as PeriodicAssignType,
            assignTargetId: values.assignTargetId,
            notifyAssignee: values.notifyAssignee,
          };
          await updatePeriodicTemplate(editingRecord.id, updateData);
          message.success("更新成功");
        } else {
          const createData: CreatePeriodicTemplateRequest = {
            templateName: values.templateName,
            workOrderTitle: values.workOrderTitle,
            description: values.description,
            categoryId: values.categoryId,
            type: values.type as WorkOrderType,
            priority: values.priority,
            cronExpression: values.cronExpression,
            assignType: values.assignType as PeriodicAssignType,
            assignTargetId: values.assignTargetId,
            notifyAssignee: values.notifyAssignee,
          };
          await createPeriodicTemplate(createData);
          message.success("创建成功");
        }
        setEditingRecord(null);
        onLoad();
      } catch (error: unknown) {
        if (error && typeof error === "object" && "errorFields" in error) {
          return;
        }
        message.error(editingRecord ? "更新失败" : "创建失败");
      }
    },
    [editingRecord, onLoad, message]
  );

  return {
    editingRecord,
    setEditingRecord,
    handleAdd,
    handleEdit,
    handleDelete,
    handleSave,
  };
}

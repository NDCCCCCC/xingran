/**
 * Phase 65 契约 — 危险删除唯一命令式入口（UI-SPEC §4）
 *
 * 全仓唯一的 Modal.confirm 合法调用点（§3 lint 拦截业务代码直调）。
 * 文案模板：确认删除 / 将删除${target}，删除后不可恢复，是否继续？/ 删除 / 取消 /
 * okButtonProps.danger；onOk 返回 Promise（antd 自动 loading 并防重）；
 * 成功 message.success("已删除")；失败 handleApiError（64 FIX-04 唯一出口）且
 * Modal 保持打开可重试（onOk reject 时 antd 不关闭）。
 * 声明式形态 ConfirmDelete.tsx 复用本函数，不自建第二套 Modal 逻辑。
 */
import { Modal } from "antd";
import type { ReactNode } from "react";
import { handleApiError } from "@/utils/errorHandler";
import { getAppMessage } from "@/utils/antdMessage";

export interface ConfirmDeleteOptions {
  /** 操作对象描述，进入确认文案。如 "用户 张三"、"选中的 3 项资产"；批量场景传 `选中的 {N} 项…` */
  target: string;
  /** 确认后执行的删除动作；抛错即失败路径（由调用方刷新列表等后续逻辑） */
  onDelete: () => Promise<void>;
  /** 确认标题，默认 "确认删除"；批量场景传 "确认批量删除"（63 兼容） */
  title?: string;
  /** extra 确认内容（如级联删除复选项），渲染在确认文案下方 */
  extra?: ReactNode;
}

/** 危险删除确认（唯一命令式入口） */
export function confirmDelete({ target, onDelete, title = "确认删除", extra }: ConfirmDeleteOptions): void {
  // extra 存在时用数组 children 组合（避免 .ts 文件内 JSX）
  const content: ReactNode = extra
    ? [`将删除${target}，删除后不可恢复，是否继续？`, extra]
    : `将删除${target}，删除后不可恢复，是否继续？`;

  Modal.confirm({
    title,
    content,
    okText: "删除",
    cancelText: "取消",
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await onDelete();
        getAppMessage().success("已删除");
      } catch (error) {
        // 失败反馈唯一出口（64 FIX-04）；rethrow 让 onOk 的 Promise reject，
        // antd 因此保持 Modal 打开、可重试
        handleApiError(error, "删除");
        throw error;
      }
    },
  });
}

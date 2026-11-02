/**
 * Phase 65 契约 — 危险删除唯一命令式入口（UI-SPEC §4）
 *
 * 全仓唯一的 Modal.confirm 合法调用点（§3 lint 拦截业务代码直调）。
 * 文案模板：确认删除 / 将删除${target}，删除后不可恢复，是否继续？/ 删除 / 取消 /
 * okButtonProps.danger；onOk 返回 Promise（antd 自动 loading 并防重）；
 * 成功 message.success("已删除")；失败 handleApiError（64 FIX-04 唯一出口）且
 * Modal 保持打开可重试（onOk reject 时 antd 不关闭）。
 * 声明式形态 ConfirmDelete.tsx 复用本函数，不自建第二套 Modal 逻辑。
 *
 * Phase 68.1 T1 向后兼容扩展（全部可选，既有调用方零改动）：
 * - actionLabel / actionContext / contentText / successText（Q4 同一扩展面）；
 * - 提交中禁取消 + aria-busy（instance.update，沿用 buildings/index.tsx 既有写法）；
 * - mask.closable:false / keyboard:false（§1.7；Phase 88-04 起改用 antd 6 新形态）；
 * - focusable:{autoFocusButton:"ok"}（antd 6.6.4 新式写法）。
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
  /** 确认按钮文案 + 成功动作词，默认 "删除"；非删除破坏性动作传如 "清空" */
  actionLabel?: string;
  /** 失败提示 context（Q4），默认 "删除"；批量场景传 "批量删除" */
  actionContext?: string;
  /** 正文模板，默认 "将删除{target}，删除后不可恢复，是否继续？"；{target} 占位符内部替换 */
  contentText?: string;
  /** 成功文案，默认 `已${actionLabel}`（默认动作词下即 "已删除"，与现状逐字一致） */
  successText?: string;
}

/** 危险删除确认（唯一命令式入口） */
export function confirmDelete({
  target,
  onDelete,
  title = "确认删除",
  extra,
  actionLabel = "删除",
  actionContext = "删除",
  contentText = "将删除{target}，删除后不可恢复，是否继续？",
  successText,
}: ConfirmDeleteOptions): void {
  const resolvedSuccessText = successText ?? `已${actionLabel}`;
  const body = contentText.replace("{target}", target);
  // extra 存在时用数组 children 组合（避免 .ts 文件内 JSX）
  const content: ReactNode = extra ? [body, extra] : body;

  // 可空 holder 引用：onOk 若被同步调用（测试 mock），实例尚未初始化，
  // 可选链安全跳过 update（真实 antd 场景下 onOk 在用户点击后触发，实例必已就绪）
  const instanceRef: { current?: ReturnType<typeof Modal.confirm> } = {};
  instanceRef.current = Modal.confirm({
    title,
    content,
    okText: actionLabel,
    cancelText: "取消",
    okButtonProps: { danger: true },
    mask: { closable: false },
    keyboard: false,
    focusable: { autoFocusButton: "ok" },
    onOk: async () => {
      // 提交中禁取消 + aria-busy（§1.7；沿用 confirmInstance.update 仓库既有写法）
      instanceRef.current?.update({
        cancelButtonProps: { disabled: true },
        okButtonProps: { danger: true, "aria-busy": true },
      });
      try {
        await onDelete();
        getAppMessage().success(resolvedSuccessText);
      } catch (error) {
        // 失败反馈唯一出口（64 FIX-04）；rethrow 让 onOk 的 Promise reject，
        // antd 因此保持 Modal 打开、可重试
        handleApiError(error, actionContext);
        throw error;
      } finally {
        instanceRef.current?.update({
          cancelButtonProps: { disabled: false },
          okButtonProps: { danger: true },
        });
      }
    },
  });
}

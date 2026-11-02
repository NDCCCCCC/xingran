/**
 * Phase 70 契约组件 — AppTextArea
 * 契约封装：默认 allowClear，其余 props 全量透传 antd Input.TextArea。
 * 契约来源：.planning/phases/70-textarea-w0/70-PLAN.md（T1 审计定稿：
 * 仅 allowClear 下沉默认；showCount/maxLength/autoSize 由调用方显式传）
 */
import { Input } from "antd";
import type { ComponentProps } from "react";

export type AppTextAreaProps = ComponentProps<typeof Input.TextArea>;

/** 契约封装：默认 allowClear，其余 props 全量透传 antd Input.TextArea */
export function AppTextArea({ allowClear = true, ...rest }: AppTextAreaProps) {
  return <Input.TextArea allowClear={allowClear} {...rest} />;
}

export default AppTextArea;

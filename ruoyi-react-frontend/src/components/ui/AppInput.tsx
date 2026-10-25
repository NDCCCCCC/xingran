/**
 * Phase 65 契约组件 — AppInput
 * 契约封装：默认 allowClear，其余 props 全量透传 antd Input。
 * 契约来源：.planning/phases/65-contracts/65-UI-SPEC.md §1
 */
import { Input } from "antd";
import type { ComponentProps } from "react";

export type AppInputProps = ComponentProps<typeof Input>;

/** 契约封装：默认 allowClear，其余 props 全量透传 antd Input */
export function AppInput({ allowClear = true, ...rest }: AppInputProps) {
  return <Input allowClear={allowClear} {...rest} />;
}

export default AppInput;

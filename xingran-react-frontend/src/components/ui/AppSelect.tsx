/**
 * Phase 65 契约组件 — AppSelect
 * 契约封装：默认 allowClear + showSearch 自动策略（options.length >= 8 自动开搜索），
 * 其余 props 全量透传 antd Select。
 * 契约来源：.planning/phases/65-contracts/65-UI-SPEC.md §2
 */
import { Select } from "antd";
import type { ComponentProps } from "react";

export type AppSelectProps = ComponentProps<typeof Select>;

/** 契约封装：默认 allowClear + showSearch 自动策略，其余全透传 */
export function AppSelect({ allowClear = true, showSearch, options, ...rest }: AppSelectProps) {
  const count = Array.isArray(options) ? options.length : 0;
  return (
    <Select
      allowClear={allowClear}
      showSearch={showSearch ?? count >= 8}
      optionFilterProp="label"
      options={options}
      {...rest}
    />
  );
}

export default AppSelect;

/**
 * Phase 65 契约组件统一出口（.planning/phases/65-contracts/65-UI-SPEC.md §1）
 *
 * 注意：`Input` / `Select` 别名导出仅供 Phase 67 存量迁移降险使用（只改 import 行
 * 的迁移策略）；**新代码必须用具名 AppInput / AppSelect**——lint 规则认 import
 * 路径不认局部名，别名不豁免拦截。
 */
export { AppInput } from "./AppInput";
export { AppSelect } from "./AppSelect";
// Phase 70：AppTextArea 独立具名导出（locked：不挂 AppInput.TextArea 静态别名、不加 TextArea 形态的别名导出）
export { AppTextArea } from "./AppTextArea";
export type { AppInputProps } from "./AppInput";
export type { AppSelectProps } from "./AppSelect";
export type { AppTextAreaProps } from "./AppTextArea";

// P67 降险别名：仅存量迁移期使用，新代码禁止
export { AppInput as Input } from "./AppInput";
export { AppSelect as Select } from "./AppSelect";

// Phase 66：RemoteSelect（防抖远程搜索下拉；fetcher+debounceMs=300+initialOptions+竞态守卫）
export { RemoteSelect } from "./RemoteSelect";
export type { RemoteSelectProps, RemoteSelectOption } from "./RemoteSelect";

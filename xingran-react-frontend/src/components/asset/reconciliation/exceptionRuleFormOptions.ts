/**
 * ExceptionRuleForm 选项构造(字典优先 + 静态回退)
 *
 * 从组件文件抽出的纯工具:组件文件若同时导出组件与函数,
 * react-refresh/only-export-components 规则会告警(lint:ci --max-warnings 0 直接失败)。
 */
import type { DictItem } from "@/hooks/useDict";
import { buildDictOptions, type DictOption } from "@/components/reconciliation/dictLabels";

/** 冲突类型静态回退选项(字典为空/未命中时才用) */
const CONFLICT_TYPE_OPTIONS: DictOption[] = [
  { label: "B 物理有/声明无", value: "B" },
  { label: "C 物理声明不匹配", value: "C" },
  { label: "D 仅声明", value: "D" },
  { label: "E 无关联", value: "E" },
  { label: "F AD 账号不一致", value: "F" },
];

/** 例外动作静态回退选项(字典为空/未命中时才用) */
const EXCEPTION_ACTION_OPTIONS: DictOption[] = [
  { label: "no_alert 不告警", value: "no_alert" },
  { label: "no_notice 不通知", value: "no_notice" },
  { label: "no_workorder 不转工单", value: "no_workorder" },
  { label: "skip_severity 降级", value: "skip_severity" },
  { label: "silence 静默", value: "silence" },
];

/** 严重度覆盖静态回退选项(字典为空/未命中时才用) */
const SEVERITY_OVERRIDE_OPTIONS: DictOption[] = [
  { label: "(不覆盖)", value: "" },
  { label: "low", value: "low" },
  { label: "medium", value: "medium" },
  { label: "high", value: "high" },
];

/**
 * severity_override 的后端硬约束:
 *   DB CHECK 约束 chk_recon_exc_severity_override 仅允许 low/medium/high,
 *   不含 critical(override 是"降级"语义,不允许升到 critical)。
 *   因此即使字典 asset_reconciliation_severity 里存在 critical,
 *   也不放进严重度覆盖下拉,避免提交被后端拒绝。
 */
const SEVERITY_OVERRIDE_EXCLUDED_VALUES = ["critical"];

/** 冲突类型选项:字典优先(管理员新增枚举项即可选到),字典为空回退静态常量 */
export function buildConflictTypeOptions(dict: DictItem[] | undefined): DictOption[] {
  const options = buildDictOptions(dict);
  return options.length > 0 ? options : CONFLICT_TYPE_OPTIONS;
}

/** 例外动作选项:字典优先,字典为空回退静态常量 */
export function buildExceptionActionOptions(dict: DictItem[] | undefined): DictOption[] {
  const options = buildDictOptions(dict);
  return options.length > 0 ? options : EXCEPTION_ACTION_OPTIONS;
}

/** 严重度覆盖选项:字典优先(剔除 critical),字典为空回退静态常量 */
export function buildSeverityOverrideOptions(dict: DictItem[] | undefined): DictOption[] {
  const options = buildDictOptions(dict).filter(
    (option) => !SEVERITY_OVERRIDE_EXCLUDED_VALUES.includes(option.value)
  );
  if (options.length === 0) return SEVERITY_OVERRIDE_OPTIONS;
  return [{ label: "(不覆盖)", value: "" }, ...options];
}

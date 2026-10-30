/**
 * 状态展示统一 helper（Phase 68 DEBT-02a）
 *
 * 唯一落点裁定（UI-SPEC §2.1 checker 修订）：必须位于 src/components/shared/，
 * 不可放 src/utils/ —— StatusTag 内部裸 <Tag> 依赖 shared 豁免块（Phase 65）。
 *
 * 文案唯一口径：enable「启用/停用」、result「成功/失败」、health「正常/异常」、
 * online「在线/离线」；正向 success、负向 error；「停用」不带「已」前缀、
 * 不写「禁用」。visible 域为反转例外（1=显示/0=隐藏）。
 */

import { Tag } from "antd";
import type { JSX } from "react";

/** 语义域 */
export type StatusDomain =
  | "enable" // 0=启用, 1=停用（通用 sys_status）
  | "online" // 0=在线, 1=离线（ad-domain computers）
  | "result" // 0=成功, 1=失败（job 日志/执行结果）
  | "health" // 0=正常, 1=异常（monitor server）
  | "workstation" // 0=空闲, 1=占用, 2=维护（工位三态，唯一多值域）
  | "visible"; // 1=显示, 0=隐藏（menu visible，反转例外）

export type StatusColor = "success" | "error" | "default" | "processing";

interface StatusEntry {
  label: string;
  color: StatusColor;
}

const STATUS_MAP: Record<StatusDomain, Record<number, StatusEntry>> = {
  enable: {
    0: { label: "启用", color: "success" },
    1: { label: "停用", color: "error" },
  },
  online: {
    0: { label: "在线", color: "success" },
    1: { label: "离线", color: "error" },
  },
  result: {
    0: { label: "成功", color: "success" },
    1: { label: "失败", color: "error" },
  },
  health: {
    0: { label: "正常", color: "success" },
    1: { label: "异常", color: "error" },
  },
  workstation: {
    0: { label: "空闲", color: "success" },
    1: { label: "占用", color: "error" },
    2: { label: "维护", color: "processing" },
  },
  visible: {
    1: { label: "显示", color: "success" },
    0: { label: "隐藏", color: "default" },
  },
};

/** 未知 status 的兜底文案 */
export const UNKNOWN_STATUS_LABEL = "未知";

/** 状态 → 统一文案（未知值返回「未知」，不 crash） */
// eslint-disable-next-line react-refresh/only-export-components -- v1.22 契约组件（68 §2.6「唯一落点裁定 shared/」），helper 与组件同文件是有意收拢，只登记不拆
export function statusLabel(status: number, domain: StatusDomain): string {
  return STATUS_MAP[domain][status]?.label ?? UNKNOWN_STATUS_LABEL;
}

/** 状态 → antd Tag 语义色（未知值返回 default） */
// eslint-disable-next-line react-refresh/only-export-components -- v1.22 契约组件（68 §2.6「唯一落点裁定 shared/」），helper 与组件同文件是有意收拢，只登记不拆
export function statusColor(status: number, domain: StatusDomain): StatusColor {
  return STATUS_MAP[domain][status]?.color ?? "default";
}

/** 统一状态标签组件（未知 status → 「未知」+ default，不 crash） */
export function StatusTag({
  status,
  domain,
}: {
  status: number;
  domain: StatusDomain;
}): JSX.Element {
  return <Tag color={statusColor(status, domain)}>{statusLabel(status, domain)}</Tag>;
}

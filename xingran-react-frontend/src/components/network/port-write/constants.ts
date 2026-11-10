/**
 * PortWriteModal / BulkWriteDrawer 共享常量 (Phase 53 W4)
 *
 * 本文件为 53-02 UI 组件 (PortWriteModal.tsx / BulkWriteDrawer.tsx) 提供共享常量,
 * 让 UI 组件只聚焦交互逻辑而不必重复硬编码字符串与数值。
 *
 * 来源:
 * - D-01: 5 个 action 中文标题
 * - D-02: reason 字段 5-200 字符校验上下限 + 预置常用原因
 * - D-03: description 保守 80 字符上限
 * - 55-01 WR-02: 共享 reason validator helpers (composeReason + validate*)
 *
 * 自洽性约束:
 *   PRESET_REASONS 每项 value 的字符数 ≥ REASON_MIN (5),
 *   避免 53-02 校验逻辑在选预设项时被 REASON_MIN 卡住。
 */

import type { FormInstance } from "antd";
import type { PortWriteAction } from "@/types/network";

/**
 * D-02 预置操作原因
 *
 * - 前 4 项覆盖运维常见场景 (故障排查 / 安全合规 / 业务变更 / 临时测试),
 *   value 字符数严格 ≥ REASON_MIN (5),与校验下限自洽。
 * - 第 5 项 '__custom__' 是 sentinel,选中时由 UI 展开 TextArea 让用户自定义输入。
 */
export const PRESET_REASONS = [
  { label: "故障排查处理", value: "故障排查处理" },
  { label: "安全合规要求", value: "安全合规要求" },
  { label: "业务变更需要", value: "业务变更需要" },
  { label: "临时测试验证", value: "临时测试验证" },
  { label: "其他...", value: "__custom__" },
] as const;

/**
 * D-01 action 的中文标题 (Phase 143: 15 → 18, 补物理参数三动作)
 *
 * 用于 PortWriteModal / SetAccessVlanModal / PortBindingModal 动态标题
 * (ACTION_TITLE[action] + ' - ' + interfaceName) 与 BulkWriteDrawer action 选择器的 label。
 */
export const ACTION_TITLE: Record<
  | "shutdown"
  | "undo_shutdown"
  | "description"
  | "dot1x_enable"
  | "dot1x_disable"
  | "set_access_vlan" // v1.20.1
  | "port_binding" // v1.20.1
  | "trunk_set_native" // v1.51 Phase 124
  | "trunk_set_allowed" // v1.51 Phase 124
  | "trunk_set_hybrid" // v1.51 Phase 124
  | "portsec_enable" // v1.51 Phase 125
  | "portsec_disable" // v1.51 Phase 125
  | "portsec_max_mac" // v1.51 Phase 125
  | "portsec_sticky" // v1.51 Phase 125
  | "portsec_static_bind" // v1.51 Phase 125
  | "set_speed" // v1.54 Phase 143
  | "set_duplex" // v1.54 Phase 143
  | "set_mtu", // v1.54 Phase 143
  string
> = {
  shutdown: "关闭端口",
  undo_shutdown: "启用端口",
  description: "修改描述",
  dot1x_enable: "启用 802.1X",
  dot1x_disable: "停用 802.1X",
  set_access_vlan: "修改 access VLAN",
  port_binding: "端口绑定",
  trunk_set_native: "设置 Trunk Native VLAN",
  trunk_set_allowed: "配置 Trunk 放行 VLAN",
  trunk_set_hybrid: "配置 Hybrid 端口",
  portsec_enable: "启用端口安全",
  portsec_disable: "停用端口安全",
  portsec_max_mac: "设置 MAC 学习上限",
  portsec_sticky: "启用 Sticky MAC",
  portsec_static_bind: "配置静态 MAC 绑定",
  set_speed: "调整端口速率",
  set_duplex: "调整端口双工模式",
  set_mtu: "调整端口 MTU",
};

// ==================== v1.54 Phase 143 物理参数表单常量 (PORT-01/02) ====================
// 与后端 portcollection.defaultPhysicalProfile() 三家交集一致；型号级差异由后端
// 能力画像决定，前端只做 UX 层白名单（后端 validator 是真相源）。

/** set_speed 速率选项 (Mbps)；auto 作为 speedMode 独立枚举，见 PHYSICAL_SPEED_MODE_AUTO */
export const PHYSICAL_SPEED_OPTIONS = [
  { label: "10 Mbps", value: 10 },
  { label: "100 Mbps", value: 100 },
  { label: "1000 Mbps (1G)", value: 1000 },
  { label: "10000 Mbps (10G)", value: 10000 },
] as const;

/** set_speed 自协商速率 speedMode 哨兵值 */
export const PHYSICAL_SPEED_MODE_AUTO = "auto";

/** set_duplex 双工选项 */
export const PHYSICAL_DUPLEX_OPTIONS = [
  { label: "半双工 (half)", value: "half" },
  { label: "全双工 (full)", value: "full" },
  { label: "自协商 (auto)", value: "auto" },
] as const;

/** set_mtu 范围（后端全局默认画像三家交集；型号级上下界以后端能力画像为准） */
export const PHYSICAL_MTU_MIN = 1280;
export const PHYSICAL_MTU_MAX = 9216;

/**
 * 物理批量无自动回滚的诚实警示（后端 dry-run Summary 与审计同源文案）。
 * B-4 / quick 261110-161：audit.before_value 只含端口状态、不含物理原值，
 * 人工恢复须依据设备 running-config / 采集快照与 audit.command_sent，
 * 不得误导为「按 before_value 恢复」。
 */
export const PHYSICAL_NO_ROLLBACK_HINT =
  "批量物理参数无自动回滚，失败需人工按设备 running-config / 采集快照与 audit.command_sent 恢复（audit.before_value 仅含端口状态，不含物理原值）。";

/** 取证级别徽标文案（默认放行 ≠ 有证据；unverified 不冒充真机结论） */
export const PHYSICAL_EVIDENCE_LABEL: Record<string, string> = {
  unverified: "未取证（默认放行）",
  model: "型号级依据",
  vendor: "厂商基线（未逐型号取证）",
};

// ==================== v1.51 Phase 124 trunk/hybrid 表单常量 ====================

/** trunk_set_allowed 的 mode 三选（契约 9835291c: add|remove|replace, 默认 replace） */
export const TRUNK_ALLOWED_MODE_OPTIONS = [
  { label: "追加 (add)", value: "add" },
  { label: "移除 (remove)", value: "remove" },
  { label: "替换 (replace)", value: "replace" },
] as const;

/** replace 厂商语义提示（portwriter 契约注释: 华为/H3C 无原生替换, 退化为叠加） */
export const TRUNK_REPLACE_HINT =
  "replace（替换）仅锐捷原生支持；华为/H3C 无原生替换语义，将退化为叠加处理，结果以预览与审计为准。";

/** hybrid tagged/untagged 重叠提示（华为拒绝 / H3C 静默优先 untagged，系统侧 400 为共同真相源） */
export const TRUNK_HYBRID_OVERLAP_HINT =
  "tagged 与 untagged VLAN 不可重叠：华为设备会拒绝，H3C 会静默优先 untagged 表项。Cisco 不支持 hybrid 端口模式（选择将返回 400）。";

/**
 * Cisco 设备不支持的 action 集（契约 adefb14c + Phase 143 裁定）。
 * Phase 143: 物理参数三动作渲染层未实现 + 无真机取证依据 → 后端能力矩阵显式标
 * vendorUnsupported，前端同步禁用/提示（迈普同口径，后端同样标 Unsupported）。
 */
export const CISCO_UNSUPPORTED_ACTIONS: readonly PortWriteAction[] = [
  "trunk_set_hybrid",
  "set_speed",
  "set_duplex",
  "set_mtu",
];

/** VLAN 列表最大条数（保守上限, 防误粘超长串; 后端同样有 sentinel 校验） */
export const VLAN_LIST_MAX = 256;

/**
 * 解析 VLAN 列表输入（逗号/空格/分号分隔）→ 去重升序数组。
 * 返回 null 表示存在非法 token（非数字 / 越界 1-4094 / 超长），由调用方标红提示。
 */
export function parseVlanList(input: string): number[] | null {
  const tokens = input
    .split(/[,;\s]+/)
    .map((t) => t.trim())
    .filter((t) => t.length > 0);
  if (tokens.length === 0) return null;
  if (tokens.length > VLAN_LIST_MAX) return null;
  const seen = new Set<number>();
  for (const token of tokens) {
    if (!/^\d{1,5}$/.test(token)) return null;
    const vlan = Number(token);
    if (!Number.isInteger(vlan) || vlan < 1 || vlan > 4094) return null;
    seen.add(vlan);
  }
  return Array.from(seen).sort((a, b) => a - b);
}

// ==================== v1.51 Phase 125 portsec 表单常量 ====================

/** skipPreflight 开关文案（lead 裁决 + portwriter 建议） */
export const SKIP_PREFLIGHT_HINT = "跳过在线预检（管理员强制下发，事件将留痕审计）";

/** portsec_static_bind MAC 输入提示（后端按厂商归一渲染: 华为/H3C 连字符、锐捷点分） */
export const PORTSEC_MAC_HINT =
  "支持 AA-BB-CC-DD-EE-FF、AA:BB:CC:DD:EE:FF、aabb.ccdd.eeff 三种格式，后端按设备厂商自动归一。";

/**
 * 校验并归一 MAC 输入（三种分隔格式 → 大写连字符形式）。
 * 返回 null 表示格式非法（非 12 位 hex）。
 */
export function parseMacAddress(input: string): string | null {
  const hex = input.trim().replace(/[:\-.]/g, "");
  if (!/^[0-9a-fA-F]{12}$/.test(hex)) return null;
  const up = hex.toUpperCase();
  return `${up.slice(0, 2)}-${up.slice(2, 4)}-${up.slice(4, 6)}-${up.slice(6, 8)}-${up.slice(
    8,
    10
  )}-${up.slice(10, 12)}`;
}

/** VLAN 列表文本可选校验（抽公共: drawer 与 modal 的 hybrid/allowed 复用） */
export function validateVlanListText(value: unknown): Promise<void> {
  if (value === undefined || value === null || String(value).trim() === "") {
    return Promise.resolve();
  }
  if (parseVlanList(String(value)) === null) {
    return Promise.reject(new Error("格式：逗号/空格分隔的 1-4094 整数（自动去重）"));
  }
  return Promise.resolve();
}

/**
 * D-02 reason 字段校验上下限
 *
 * - REASON_MIN=5: 与 PRESET_REASONS 每项 value 字符数自洽 (预设项最长 6 字符,最短 5 字符)
 * - REASON_MAX=200: 与后端 PortWriteRequest.reason 字符上限对齐 (保守值,实际后端可能更宽松)
 */
export const REASON_MIN = 5;
export const REASON_MAX = 200;

/**
 * D-03 description 字段保守上限
 *
 * 华为/H3C/锐捷端口描述字符上限因厂商版本不同 (华为 24/80 视版本),
 * 前端不做厂商分支,统一 80 字符保守上限;后端设备侧会拒超长,前端保守值减少往返。
 */
export const DESCRIPTION_MAX = 80;

/**
 * 55-01 WR-02: reason 字段 "其他..." 自定义选项的 sentinel 值。
 *
 * PRESET_REASONS 第 5 项 value 同步为该常量,UI 层用 === 比较展开自定义 TextArea。
 * 提到共享常量以便两个组件对齐 sentinel 字符串,避免魔数漂移。
 */
export const REASON_CUSTOM_SENTINEL = "__custom__";

/**
 * 严格 IPv4 regex (v1.20.1 BIND-07, 与后端 service ipv4Pattern 对齐)
 *
 * - 拒 0.x.x.x / 255.x.x.x 段 (每段 0-255,首段允许 1-254 业务地址)
 * - 接受形如 10.62.25.5 / 192.168.1.1 的 4 段十进制
 * - 客户端 regex 仅 UX hint, 后端 service 为真相源 (防绕过)
 * - 与 56-UI-SPEC §Copywriting Contract IPv4_REGEX 字面量严格一致
 */
export const IPV4_REGEX =
  /^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])$/;

/**
 * MAC 地址 regex (v1.20.1 BIND-07)
 *
 * - 接受冒号 / 连字符 / 无分隔符 三种格式 (canonical / hyphen / bare)
 * - 12 位 hex 字符 (大小写不敏感)
 * - 后端 NormalizeMACAddress 会归一化为 AA:BB:CC:DD:EE:FF
 * - 客户端 regex 仅 UX hint, 后端 service 为真相源
 */
export const MAC_REGEX = /^([0-9A-Fa-f]{2}[:-]?){5}[0-9A-Fa-f]{2}$/;

/**
 * 端口绑定 op 枚举 (v1.20.1 BIND-01/02)
 *
 * - PortBindingModal 的 op Radio.Group 用此常量渲染选项
 * - 后端 port_binding wrapper 接收 "add" | "remove" 二态字面量
 */
export const BIND_OPS = [
  { label: "新增绑定 (add)", value: "add" },
  { label: "删除绑定 (remove)", value: "remove" },
] as const;

/**
 * D-02 reason 字段 reasonSelect + reasonText 合并出最终 reason 值。
 *
 * - 选预设项 (value !== REASON_CUSTOM_SENTINEL): 直接用预设字符串
 * - 选"其他...": 用 reasonText TextArea 输入值 (trim 后)
 *
 * 返回 null 表示未填 (description action 可空场景)。
 */
export function composeReason(reasonSelect: unknown, reasonText: unknown): string | null {
  if (reasonSelect === REASON_CUSTOM_SENTINEL) {
    const text = typeof reasonText === "string" ? reasonText.trim() : "";
    return text.length > 0 ? text : null;
  }
  return typeof reasonSelect === "string" && reasonSelect.length > 0 ? reasonSelect : null;
}

/**
 * D-03 description action 时 reason 可空 — 若填了则校验长度。
 *
 * 55-01 WR-02 修复: 原签名错误接收 `(rule, reasonSelect, reasonText)` 第三参数
 * 实际拿不到值 (antd validator 只传 (rule, value))。现改用 form 参数 (FormInstance)
 * 调用 `form.getFieldValue("reasonText")` 跨字段取值,绕过 antd 调用约定不传跨字段值的限制。
 */
export function validateReasonOptional(
  _: unknown,
  value: unknown,
  form: FormInstance
): Promise<void> {
  const reasonSelect = value;
  const reasonText = form.getFieldValue("reasonText");
  const reason = composeReason(reasonSelect, reasonText);
  if (reason === null) return Promise.resolve();
  if (reason.length < REASON_MIN) {
    return Promise.reject(new Error(`操作原因至少 ${REASON_MIN} 个字符`));
  }
  if (reason.length > REASON_MAX) {
    return Promise.reject(new Error(`操作原因不超过 ${REASON_MAX} 个字符`));
  }
  return Promise.resolve();
}

/**
 * D-03 其他 4 action 时 reason 必填 + 长度 5-200。
 *
 * 55-01 WR-02 修复: 同 validateReasonOptional, 用 form 参数跨字段取值。
 */
export function validateReasonRequired(
  _: unknown,
  value: unknown,
  form: FormInstance
): Promise<void> {
  const reasonSelect = value;
  const reasonText = form.getFieldValue("reasonText");
  const reason = composeReason(reasonSelect, reasonText);
  if (reason === null || reason.length === 0) {
    return Promise.reject(new Error("请选择或输入操作原因"));
  }
  if (reason.length < REASON_MIN) {
    return Promise.reject(new Error(`操作原因至少 ${REASON_MIN} 个字符`));
  }
  if (reason.length > REASON_MAX) {
    return Promise.reject(new Error(`操作原因不超过 ${REASON_MAX} 个字符`));
  }
  return Promise.resolve();
}

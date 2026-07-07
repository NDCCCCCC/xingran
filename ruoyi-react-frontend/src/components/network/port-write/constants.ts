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
 *
 * 自洽性约束:
 *   PRESET_REASONS 每项 value 的字符数 ≥ REASON_MIN (5),
 *   避免 53-02 校验逻辑在选预设项时被 REASON_MIN 卡住。
 */

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
 * D-01 5 个 action 的中文标题
 *
 * 用于 PortWriteModal 动态标题 (ACTION_TITLE[action] + ' - ' + interfaceName)
 * 与 BulkWriteDrawer action 选择器的 label。
 */
export const ACTION_TITLE: Record<
  | "shutdown"
  | "undo_shutdown"
  | "description"
  | "dot1x_enable"
  | "dot1x_disable",
  string
> = {
  shutdown: "关闭端口",
  undo_shutdown: "取消关闭",
  description: "修改描述",
  dot1x_enable: "启用 802.1X",
  dot1x_disable: "停用 802.1X",
};

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

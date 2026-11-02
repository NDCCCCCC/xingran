/**
 * renderStatusTag 便捷函数（v1.24 P82 自 ModernTag.tsx 拆出：
 * react-refresh 要求组件文件只导出组件；非组件导出会破坏 Fast Refresh 边界）
 */
import { ModernTag } from "./ModernTag";
import { statusLabel } from "./statusHelpers";

/**
 * 渲染状态标签的便捷函数
 * @param status 状态值 (0=正常/启用, 1=停用/禁用)
 * @param config 可选配置
 */
export function renderStatusTag(
  status: number,
  config?: {
    /** 正常状态的文本 */
    normalText?: string;
    /** 停用状态的文本 */
    stopText?: string;
    /** 是否显示图标 */
    showIcon?: boolean;
    /** 是否使用现代化样式 */
    modern?: boolean;
  }
) {
  // Phase 68 DEBT-02a：域固定 enable，默认文案收敛为 statusHelpers 口径
  // （0=「启用」，去「正常/禁用」混写）；normalText/stopText 覆盖语义保留。
  const {
    normalText = statusLabel(0, "enable"),
    stopText = statusLabel(1, "enable"),
    showIcon = true,
    modern = true,
  } = config || {};

  return (
    <ModernTag status={status === 0 ? "success" : "error"} showIcon={showIcon} modern={modern}>
      {status === 0 ? normalText : stopText}
    </ModernTag>
  );
}

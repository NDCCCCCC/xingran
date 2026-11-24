/**
 * Phase 155 / LIVEVIEW-01~04：帧/状态消息解析与画布→viewport 坐标换算（纯函数）
 *
 * 设计原则：
 *   1. 全部为纯函数（无 DOM、无 React、无副作用），便于逐值断言（见 __tests__）。
 *   2. 任何畸形消息一律返回 null 而不是抛错 —— WS 帧流高频，解析失败不能
 *      打断渲染循环（协议 §6「帧必须可丢」）。
 *   3. 坐标换算是硬契约（协议 §9）：
 *        cdpX = (canvasX / canvasDisplayWidth)  * frame.deviceWidth
 *        cdpY = (canvasY / canvasDisplayHeight) * frame.deviceHeight
 *      canvasDisplayWidth/Height 取 getBoundingClientRect()（CSS 显示尺寸），
 *      **不是** canvas.width/height 属性（那是 JPEG 实际像素）。
 */

import type {
  LiveContext,
  LiveFrame,
  LiveFrameEnvelope,
  LiveInputCommand,
  LiveInputKind,
  LiveMode,
} from "./types";

// ==================== 帧 / 状态消息解析 ====================

/** 把任意值收敛为有限数字；非数字回落 fallback */
function toNumber(value: unknown, fallback = 0): number {
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : fallback;
}

/** 把任意值收敛为字符串（undefined/null → 空串） */
function toStr(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * 解析 `rpa_liveview_frame` 的 content 字符串。
 *
 * content 形状：`{ frame: {...}, context: {...} }`（协议 §2）。
 * 帧缺少 data/尺寸时视为无效帧返回 null（画不出来就别画）。
 */
export function parseLiveFrameMessage(content: unknown): LiveFrameEnvelope | null {
  if (typeof content !== "string" || content === "") return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(content);
  } catch {
    return null;
  }
  if (!isRecord(parsed)) return null;

  const rawFrame = parsed.frame;
  if (!isRecord(rawFrame)) return null;

  const data = toStr(rawFrame.data);
  if (data === "") return null;

  const frame: LiveFrame = {
    executionId: toStr(rawFrame.executionId),
    seq: toNumber(rawFrame.seq),
    timestamp: toNumber(rawFrame.timestamp),
    data,
    byteSize: toNumber(rawFrame.byteSize, data.length),
    width: toNumber(rawFrame.width),
    height: toNumber(rawFrame.height),
    deviceWidth: toNumber(rawFrame.deviceWidth),
    deviceHeight: toNumber(rawFrame.deviceHeight),
    scrollOffsetX: toNumber(rawFrame.scrollOffsetX),
    scrollOffsetY: toNumber(rawFrame.scrollOffsetY),
    pageScaleFactor: toNumber(rawFrame.pageScaleFactor, 1),
  };

  // 无尺寸的帧无法确定 canvas 逻辑尺寸，直接丢（上层继续等下一帧）。
  if (frame.width <= 0 || frame.height <= 0) return null;

  const context = isRecord(parsed.context) ? parseLiveContext(parsed.context) : undefined;
  return { frame, context };
}

/**
 * 解析 `rpa_liveview_state` 的 content 字符串。
 *
 * content 形状：`{ context: {...} }`（协议 §2）。返回 null 表示畸形消息。
 */
export function parseLiveStateMessage(content: unknown): LiveContext | null {
  if (typeof content !== "string" || content === "") return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(content);
  } catch {
    return null;
  }
  if (!isRecord(parsed)) return null;

  const rawCtx = parsed.context;
  if (!isRecord(rawCtx)) return null;
  return parseLiveContext(rawCtx);
}

/** 从任意对象收敛出 LiveContext（缺字段给安全默认值） */
export function parseLiveContext(raw: Record<string, unknown>): LiveContext {
  const varsRaw = raw.variables;
  let variables: Record<string, string> | undefined;
  if (isRecord(varsRaw)) {
    variables = {};
    for (const [k, v] of Object.entries(varsRaw)) {
      // 脱敏由 Worker 完成；这里只做字符串化，绝不猜测明文。
      variables[k] = toStr(v);
    }
  }

  const modeRaw = toStr(raw.mode);
  const mode: LiveMode | undefined =
    modeRaw === "auto" || modeRaw === "copilot" || modeRaw === "manual"
      ? (modeRaw as LiveMode)
      : undefined;

  return {
    executionId: toStr(raw.executionId),
    taskName: toStr(raw.taskName) || undefined,
    workerId: toStr(raw.workerId) || undefined,
    workerName: toStr(raw.workerName) || undefined,
    step: toNumber(raw.step),
    totalSteps: toNumber(raw.totalSteps),
    actionType: toStr(raw.actionType) || undefined,
    actionDesc: toStr(raw.actionDesc) || undefined,
    status: toStr(raw.status) || undefined,
    mode,
    takeoverBy: toStr(raw.takeoverBy) || undefined,
    takeoverAt: toNumber(raw.takeoverAt) || undefined,
    lastLog: toStr(raw.lastLog) || undefined,
    variables,
    startedAt: toNumber(raw.startedAt) || undefined,
  };
}

// ==================== 坐标换算（协议 §9 硬要求） ====================

/** 画布矩形（只取换算需要的两个维度，便于测试构造） */
export interface CanvasRectLike {
  left: number;
  top: number;
  width: number;
  height: number;
}

/** 一次指针事件的客户端坐标 + 当前帧的 device 尺寸 */
export interface PointerToViewportInput {
  clientX: number;
  clientY: number;
  rect: CanvasRectLike;
  deviceWidth: number;
  deviceHeight: number;
}

/**
 * 画布 CSS 坐标 → viewport CSS 像素（CDP Input.dispatch* 期望的坐标）。
 *
 * 公式（协议 §9）：
 *   x = ((clientX - rect.left) / rect.width)  * deviceWidth
 *   y = ((clientY - rect.top)  / rect.height) * deviceHeight
 *
 * rect.width/height 为 0（未布局 / 隐藏）时返回 null —— 避免除零产出 NaN
 * 指令被后端 400 拒绝。
 */
export function pointerToViewport(input: PointerToViewportInput): { x: number; y: number } | null {
  const { clientX, clientY, rect, deviceWidth, deviceHeight } = input;
  if (!Number.isFinite(clientX) || !Number.isFinite(clientY)) return null;
  if (!(rect.width > 0) || !(rect.height > 0)) return null;
  if (!(deviceWidth > 0) || !(deviceHeight > 0)) return null;

  return {
    x: ((clientX - rect.left) / rect.width) * deviceWidth,
    y: ((clientY - rect.top) / rect.height) * deviceHeight,
  };
}

// ==================== 修饰键位掩码（协议 §4） ====================

/** Alt=1, Ctrl=2, Meta=4, Shift=8 */
export const MODIFIER_ALT = 1;
export const MODIFIER_CTRL = 2;
export const MODIFIER_META = 4;
export const MODIFIER_SHIFT = 8;

/** 由 DOM 事件修饰键状态算出 CDP 位掩码 */
export function computeModifiers(flags: {
  altKey?: boolean;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
}): number {
  let mask = 0;
  if (flags.altKey) mask |= MODIFIER_ALT;
  if (flags.ctrlKey) mask |= MODIFIER_CTRL;
  if (flags.metaKey) mask |= MODIFIER_META;
  if (flags.shiftKey) mask |= MODIFIER_SHIFT;
  return mask;
}

// ==================== 帧率 / 带宽统计（可选但有用） ====================

/** 统计窗口（毫秒）：只在这个时间窗内算 fps / Mbps，避免长期均值失去意义 */
export const BANDWIDTH_WINDOW_MS = 3000;

export interface FrameStats {
  fps: number;
  mbps: number;
  samples: number;
}

/**
 * 由「帧时间戳 + 字节数」样本序列算出 fps / Mbps。
 *
 * @param samples 最近的帧样本（timestamp 毫秒，byteSize 字节数）
 * @param nowMs 当前时间（注入以便测试确定性）
 */
export function computeFrameStats(
  samples: Array<{ timestamp: number; byteSize: number }>,
  nowMs: number
): FrameStats {
  if (samples.length === 0) return { fps: 0, mbps: 0, samples: 0 };

  const windowStart = nowMs - BANDWIDTH_WINDOW_MS;
  const inWindow = samples.filter((s) => s.timestamp >= windowStart);
  if (inWindow.length === 0) return { fps: 0, mbps: 0, samples: 0 };

  const bytes = inWindow.reduce((sum, s) => sum + (s.byteSize > 0 ? s.byteSize : 0), 0);
  // 既有 fps（帧数/秒）也有带宽（比特/秒 → Mbps）。窗口取实际跨度，
  // 单帧时用整个窗口长度兜底（否则除零或得出荒谬的 1000fps）。
  const spanMs = Math.max(nowMs - inWindow[0].timestamp, 1);
  const denomMs = Math.max(spanMs, inWindow.length === 1 ? BANDWIDTH_WINDOW_MS : 0);

  const fps = (inWindow.length / denomMs) * 1000;
  const mbps = (bytes * 8) / (denomMs / 1000) / 1_000_000;

  return {
    fps: round2(fps),
    mbps: round2(mbps),
    samples: inWindow.length,
  };
}

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

// ==================== 动作类型中文映射 ====================

/**
 * 动作类型中文标签（与 ../constants.tsx 的 ACTION_TYPE_TEXT_MAP 同语义，
 * 额外补 screenshot/scroll/hover/press/download/iframe 等 RPA 脚本动作，
 * 因为实时画面里的 actionType 来自脚本步骤而非表单枚举）。
 */
export const ACTION_TYPE_LABELS: Record<string, string> = {
  navigate: "导航",
  click: "点击",
  fill: "输入",
  wait: "等待",
  select: "选择",
  upload: "上传",
  extract: "提取",
  assert: "断言",
  script: "自定义脚本",
  screenshot: "截图",
  scroll: "滚动",
  hover: "悬停",
  press: "按键",
  download: "下载",
  iframe: "切换 iframe",
  pause: "暂停待人工",
};

/** 动作类型 → 中文（未知类型原样展示，不吞掉信息） */
export function actionTypeLabel(actionType: string | undefined): string {
  if (!actionType) return "-";
  return ACTION_TYPE_LABELS[actionType] ?? actionType;
}

// ==================== 模式标签 ====================

export const LIVE_MODE_OPTIONS: Array<{ label: string; value: LiveMode }> = [
  { label: "自动（auto）", value: "auto" },
  { label: "副驾（copilot）", value: "copilot" },
  { label: "手动（manual）", value: "manual" },
];

export function liveModeLabel(mode: LiveMode | undefined): string {
  switch (mode) {
    case "auto":
      return "自动";
    case "manual":
      return "手动";
    case "copilot":
    default:
      return "副驾";
  }
}

// ==================== 输入指令构造 ====================

/** 构造一条输入指令（补齐 executionId / timestamp；operator 由后端注入） */
export function buildInputCommand(
  executionId: string,
  kind: LiveInputKind,
  extra: Partial<LiveInputCommand> = {},
  nowMs: number = Date.now()
): LiveInputCommand {
  return {
    executionId,
    kind,
    timestamp: nowMs,
    ...extra,
  };
}

/** 鼠标按键名归一（DOM 的 0/1/2 → 协议 left/middle/right） */
export function mouseButtonName(button: number): "left" | "middle" | "right" | "none" {
  switch (button) {
    case 0:
      return "left";
    case 1:
      return "middle";
    case 2:
      return "right";
    default:
      return "none";
  }
}

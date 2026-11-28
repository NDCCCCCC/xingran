/**
 * Phase 155 / LIVEVIEW-01~05：实时画面前端类型（与 155-PROTOCOL.md §§1-4 同构）
 *
 * 归属说明：本目录是 Phase 155 的唯一前端写面；类型只描述协议字段，
 * 不引入后端 worker 侧任何私有形状。
 */

/** 协作模式（协议 §3 mode 端点允许值；默认 copilot 由后端下发） */
export type LiveMode = "auto" | "copilot" | "manual";

/** 输入注入事件种类（协议 §4 kind） */
export type LiveInputKind =
  "mouse_move" | "mouse_down" | "mouse_up" | "mouse_wheel" | "key_down" | "key_up" | "text";

/** 下行输入体（协议 §4 LiveInputCommand；operator/timestamp 由页面补齐） */
export interface LiveInputCommand {
  executionId: string;
  kind: LiveInputKind;
  x?: number;
  y?: number;
  button?: "left" | "right" | "middle" | "none";
  clickCount?: number;
  modifiers?: number;
  deltaX?: number;
  deltaY?: number;
  key?: string;
  code?: string;
  text?: string;
  virtualKey?: number;
  operator?: string;
  timestamp?: number;
}

/**
 * 画面帧（协议 §2 frame）。
 *
 * ⚠ width/height 是 **JPEG 实际像素**，直接作为 canvas 的 width/height 属性；
 * deviceWidth/deviceHeight 是浏览器 viewport 的 DIP 尺寸，**只**用于画布坐标换算。
 * 二者不可混用（协议 §9 缩放映射硬要求）。
 */
export interface LiveFrame {
  executionId: string;
  seq: number;
  timestamp: number;
  /** base64 JPEG（不含 data: 前缀） */
  data: string;
  byteSize: number;
  width: number;
  height: number;
  deviceWidth: number;
  deviceHeight: number;
  scrollOffsetX?: number;
  scrollOffsetY?: number;
  pageScaleFactor?: number;
}

/** 叠加层情境（协议 §2 context；variables 已在 Worker 侧脱敏为 ***） */
export interface LiveContext {
  executionId: string;
  taskName?: string;
  workerId?: string;
  workerName?: string;
  step: number;
  totalSteps: number;
  actionType?: string;
  actionDesc?: string;
  status?: string;
  mode?: LiveMode;
  /**
   * 接管者**展示名**（nickname/username），仅用于 HUD 文案。
   *
   * ⚠ W7 / UI-FIX-02：不要用它判定「我是不是接管者」。展示名会随用户改昵称、
   * 随 token 分支（SM2/HS256）取值口径不同而漂移 —— 生产上正是它导致
   * 「已接管但遮罩不消失」。控制权判据只用 `takeoverById`（用户 ID）。
   */
  takeoverBy?: string;
  /** 接管者用户 ID（控制权权威判据） */
  takeoverById?: string;
  takeoverAt?: number;
  lastLog?: string;
  variables?: Record<string, string>;
  startedAt?: number;
}

/** `rpa_liveview_frame` 的 content 解析结果 */
export interface LiveFrameEnvelope {
  frame: LiveFrame;
  context?: LiveContext;
}

/** GET /liveview/state 的 data 形状（含 hub 侧统计快照） */
export interface LiveStateSnapshot {
  context: LiveContext | null;
  watchers?: number;
  stats?: {
    watchers?: number;
    frames?: number;
    dropped?: number;
    lastFrameMs?: number;
    /**
     * W7.1 / 缺陷 C：已推流但超过阈值仍零帧 —— 页面没有可画内容
     *（典型：停在 about:blank 或空 DOM）。前端据此把「等待画面…」换成
     *「当前页面暂无画面（可能停在空白页或正在加载）」并给「刷新状态」按钮。
     */
    emptyPageHint?: boolean;
    /** 已等待的毫秒数（给用户看「等了多久」，不出现内部术语） */
    emptyPageWaitingMs?: number;
  };
}

/** POST /liveview/watch 的 data 形状 */
export interface LiveWatchState {
  executionId?: string;
  mode?: LiveMode;
  maxStreams?: number;
  streaming?: number;
  takeoverBy?: string;
  /** 接管者用户 ID（前端据此判 isController） */
  takeoverById?: string;
}

/** 页面顶部「可观看执行」条目（status 放宽为 string：后端含 paused） */
export interface LiveExecutionSummary {
  id: string;
  taskName?: string;
  workerName?: string;
  status: string;
  step?: number;
  totalSteps?: number;
  startedAt?: string;
}

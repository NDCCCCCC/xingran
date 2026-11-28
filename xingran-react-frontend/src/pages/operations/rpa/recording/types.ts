/**
 * Phase W6 / REC-01：录制与拾取会话的前端类型
 *
 * 与后端 internal/services/rpa/recording_service.go 的类型同构
 * （RecordingPickResult / RecordingResultData / 状态取值）。
 */

/** 录制会话状态（与后端 RecordingSessionState 同值） */
export type RecordingState =
  "created" | "starting" | "recording" | "stopped" | "failed" | "expired" | "aborted";

/** 一级选择器候选（7 级阶梯的其中一级；与后端 RecordingTier 同构） */
export interface RecordingTier {
  tier: number;
  name: string;
  selector: string;
  ok: boolean;
  score: number;
}

/** 拾取结果（7 级阶梯；与后端 RecordingPickResult 同构） */
export interface RecordingPickResult {
  sessionId: string;
  found: boolean;
  tag?: string;
  type?: string;
  text?: string;
  tiers?: RecordingTier[];
  recommended?: string;
  recommendedTier?: number;
  recommendedScore?: number;
  frame?: string;
  message?: string;
}

/** 录制产出（与后端 RecordingResultData 同构） */
export interface RecordingResultData {
  actions?: Array<Record<string, unknown>>;
  steps?: string[];
  startUrl?: string;
  eventCount?: number;
}

/** 是否终态（终态后前端停止轮询） */
export function isTerminalRecordingState(state: string): boolean {
  return state === "stopped" || state === "failed" || state === "expired" || state === "aborted";
}

/**
 * Phase W6 / REC-01：录制与拾取会话的 API 封装
 *
 * 纪律（AGENTS.md「Frontend API Calling」）：全部经 `@/lib/api` 的 `post`/`get`
 * 包裹函数，禁止裸 axios —— 走统一鉴权、SM2/SM4 加密与 401 刷新重试。
 *
 * 后端端点（internal/api/v1/rpa/recording_handler.go）：
 *   POST /rpa/recordings/start            开始录制
 *   POST /rpa/recordings/:id/stop         停止录制
 *   POST /rpa/recordings/:id/pick         拾取元素
 *   POST /rpa/recordings/:id/keepalive    心跳
 *   POST /rpa/recordings/:id/abort        中止
 *   GET  /rpa/recordings/:id              会话快照（轮询）
 *
 * 画面帧不走这里：帧复用 155 liveview 通道（rpa_liveview_frame WS 消息，
 * executionId 位放 sessionId），坐标系与可丢广播语义共用一套（W6-PLAN §2.2）。
 */

import { get, post } from "@/lib/api";
import type { BaseResponse } from "@/types/base";
import type { RecordingPickResult, RecordingResultData } from "./types";

/** 会话快照（GET /rpa/recordings/:id 的 data 形状） */
export interface RecordingSessionSnapshot {
  sessionId: string;
  taskId?: string;
  state: string;
  stateText: string;
  message?: string;
  eventCount: number;
  result?: RecordingResultData;
  lastPick?: RecordingPickResult;
  createdAt?: string;
  updatedAt?: string;
}

/** 开始录制的响应 */
export interface RecordingStartResponse {
  sessionId: string;
  state: string;
  stateText: string;
  maxSessions?: number;
}

/** 开始录制 */
export function startRecording(params: {
  taskId?: string;
  workerId?: string;
  targetUrl?: string;
}): Promise<BaseResponse<RecordingStartResponse>> {
  return post<RecordingStartResponse>("/rpa/recordings/start", params);
}

/** 停止录制（产出由 Worker 异步上行，前端轮询拿结果） */
export function stopRecording(sessionId: string): Promise<unknown> {
  return post(`/rpa/recordings/${sessionId}/stop`, {});
}

/** 拾取元素（坐标为画布 viewport CSS 像素，与 155 坐标系一致） */
export function pickElement(sessionId: string, x: number, y: number): Promise<unknown> {
  return post(`/rpa/recordings/${sessionId}/pick`, { x, y });
}

/** 心跳（20s 一次，防止会话被闲置回收） */
export function keepaliveRecording(sessionId: string): Promise<unknown> {
  return post(`/rpa/recordings/${sessionId}/keepalive`, {});
}

/** 中止录制（放弃本次录制并释放执行机资源） */
export function abortRecording(sessionId: string, reason?: string): Promise<unknown> {
  return post(`/rpa/recordings/${sessionId}/abort`, { reason });
}

/** 拉取会话快照（轮询：状态 / 产出 / 最近拾取结果） */
export function fetchRecordingSession(
  sessionId: string
): Promise<BaseResponse<RecordingSessionSnapshot>> {
  return get<RecordingSessionSnapshot>(`/rpa/recordings/${sessionId}`);
}

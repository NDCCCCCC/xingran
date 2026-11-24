/**
 * Phase 155 / LIVEVIEW-01/04：帧与状态消息接入总线（**不新建 WebSocket**）
 *
 * 为什么是「运行时挂接」而不是新开一条 WS：
 *   后端 `NoticeHub` 的连接表是 `clients[userID] = client`（一用户一条连接，
 *   notice_hub.go:118）。若 live 页再自建第二条 WS 连到同一 URL，第二次注册会
 *   覆盖同一 userID 的条目 —— 铃铛（NotificationBell）的旧连接从此收不到任何
 *   推送且永不关闭，即「打开实时画面 = 通知铃铛静默失效 + FD 泄漏」。
 *   所以这里复用**同一条既有连接**：
 *
 *   1. `installLiveFrameTap()` 在进入 live 页时，把 store 的 `handleWsMessage`
 *      换成一个「先转发原实现、再分流 live 消息」的包装；
 *   2. 卸载时恢复原实现（引用相等，不触发任何 store 状态变更）。
 *
 *   noticeStore.ts 本身**不被修改**：这里是运行时对 store action 的引用替换，
 *   且 NotificationBell 每次消息都经 `useNoticeStore.getState().handleWsMessage`
 *   取最新引用（NotificationBell.tsx:80），因此包装立即生效。
 *
 * 未命中 live 类型时行为与原来逐字节一致（原实现先被调用，无副作用差异）。
 */

import type { WebSocketNoticeMessage } from "@/types/notice";
import { useNoticeStore } from "@/store/noticeStore";
import { parseLiveFrameMessage, parseLiveStateMessage } from "./liveFrameParser";
import type { LiveContext, LiveFrameEnvelope } from "./types";

/** 帧消息类型（协议 §1） */
export const LIVE_FRAME_TYPE = "rpa_liveview_frame";
/** 叠加层情境消息类型（协议 §1） */
export const LIVE_STATE_TYPE = "rpa_liveview_state";

/** 派人入参：从 WS 信封里提取的 `{ type, content }` */
export interface LiveWsPayload {
  type: string;
  content: string;
}

export type LiveFrameListener = (envelope: LiveFrameEnvelope) => void;
export type LiveStateListener = (context: LiveContext) => void;

/** 模块级监听器集合（与 noticeStore 的 RPA/portWrite 同款：不进响应式 state） */
const frameListeners = new Set<LiveFrameListener>();
const stateListeners = new Set<LiveStateListener>();

let tapRefCount = 0;
/** 挂接前 store 上原始的 handleWsMessage（卸载时按引用恢复） */
let originalHandleWsMessage: ((message: WebSocketNoticeMessage) => void) | null = null;

function dispatchFrame(payload: LiveWsPayload): void {
  const envelope = parseLiveFrameMessage(payload.content);
  if (!envelope) return;
  frameListeners.forEach((cb) => {
    try {
      cb(envelope);
    } catch (error) {
      console.error("[LiveView] 帧监听器异常:", error);
    }
  });
}

function dispatchState(payload: LiveWsPayload): void {
  const context = parseLiveStateMessage(payload.content);
  if (!context) return;
  stateListeners.forEach((cb) => {
    try {
      cb(context);
    } catch (error) {
      console.error("[LiveView] 情境监听器异常:", error);
    }
  });
}

/** 当前是否已挂接（观测/测试用） */
export function isLiveTapInstalled(): boolean {
  return tapRefCount > 0;
}

/**
 * 挂接 live 消息分流（幂等，引用计数）。
 *
 * 返回卸载函数；计数归零时才真正恢复 store 原实现。
 */
export function installLiveFrameTap(): () => void {
  tapRefCount += 1;
  if (tapRefCount === 1) {
    const current = useNoticeStore.getState().handleWsMessage;
    originalHandleWsMessage = current;

    const patched = (message: WebSocketNoticeMessage) => {
      // 1) 先走原分流（new_notice / rpa_progress / port_write / ad_drift 全不受影响）
      try {
        originalHandleWsMessage?.(message);
      } catch (error) {
        console.error("[LiveView] 原 WS 分流异常:", error);
      }
      // 2) 再分流 live 两类消息（其余类型静默跳过）
      const type = (message as unknown as { type?: unknown })?.type;
      const content = (message as unknown as { content?: unknown })?.content;
      if (typeof type !== "string" || type !== LIVE_FRAME_TYPE) {
        if (typeof type === "string" && type === LIVE_STATE_TYPE) {
          dispatchState({ type, content: typeof content === "string" ? content : "" });
        }
        return;
      }
      dispatchFrame({ type, content: typeof content === "string" ? content : "" });
    };

    useNoticeStore.setState({ handleWsMessage: patched });
  }

  let released = false;
  return () => {
    if (released) return;
    released = true;
    tapRefCount = Math.max(0, tapRefCount - 1);
    if (tapRefCount === 0 && originalHandleWsMessage) {
      useNoticeStore.setState({ handleWsMessage: originalHandleWsMessage });
      originalHandleWsMessage = null;
    }
  };
}

/** 订阅画面帧（返回退订函数） */
export function onLiveFrame(callback: LiveFrameListener): () => void {
  frameListeners.add(callback);
  return () => {
    frameListeners.delete(callback);
  };
}

/** 订阅叠加层情境（返回退订函数） */
export function onLiveState(callback: LiveStateListener): () => void {
  stateListeners.add(callback);
  return () => {
    stateListeners.delete(callback);
  };
}

/** 清空全部监听器（仅供测试 / 页面硬重置；生产路径由各 useEffect cleanup 负责） */
export function resetLiveListeners(): void {
  frameListeners.clear();
  stateListeners.clear();
}

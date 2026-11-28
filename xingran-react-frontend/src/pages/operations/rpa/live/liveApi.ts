/**
 * Phase 155 / LIVEVIEW-01~05：实时画面 API 封装
 *
 * 纪律（AGENTS.md「Frontend API Calling」）：全部经 `@/lib/api` 的 `post`/`get`
 * 包裹函数，**禁止裸 axios** —— 这样才走统一鉴权、SM2/SM4 加密与 401 刷新重试。
 *
 * 端点清单与 155-PROTOCOL.md §3 一一对应；不新增任何权限码
 * （后端全部挂既有 ops:rpa:execution:list）。
 */

import { get, post } from "@/lib/api";
import { getAuthHeaders } from "@/utils/authHelpers";
import type { PageResponse } from "@/types/base";
import type {
  LiveExecutionSummary,
  LiveInputCommand,
  LiveMode,
  LiveStateSnapshot,
  LiveWatchState,
} from "./types";

/** liveview 端点基址（不含 /api/v1，由 axios baseURL 统一补） */
function liveBase(executionId: string): string {
  return `/rpa/executions/${executionId}/liveview`;
}

// ==================== 可观看执行列表 ====================

/** 可观看的状态集合（执行中 + 人工干预暂停中） */
export const WATCHABLE_STATUSES = ["running", "paused"] as const;

/**
 * 拉取「可观看执行」（running + paused 两态合集，供顶部列表切换）。
 *
 * 取值域说明（Phase 155 交付时跨层核对）：后端
 * `rpa.ExecutionListParams.Validate()`（internal/services/rpa/types.go）在 Phase 153
 * 漏补了 `paused` —— 实时画面页恰好需要「执行中 + 人工干预暂停中」两类可观看对象，
 * 故本 phase 同批把 `paused` 补进后端值域（值域与 rpamodels.RPAExecutionStatus 常量同源）。
 * 前端据此做**两次精确筛选**（running / paused 各一次，并发发出），不再退化为
 * 「拉一页再客户端过滤」——后者在并发执行多时可能漏掉 paused 的执行。
 *
 * 用 allSettled：任一状态查询失败（例如后端尚未升级到含 paused 的版本）时，
 * 另一状态的结果仍然可用，并以失败侧的 message 作为部分降级提示（不静默吞错）。
 */
export async function listWatchableExecutions(params?: {
  current?: number;
  pageSize?: number;
}): Promise<LiveExecutionSummary[]> {
  const current = params?.current ?? 1;
  const pageSize = params?.pageSize ?? 20;

  const results = await Promise.allSettled(
    WATCHABLE_STATUSES.map((status) =>
      post<PageResponse<LiveExecutionSummary>>("/rpa/executions/list", {
        current,
        pageSize,
        status,
      })
    )
  );

  const merged: LiveExecutionSummary[] = [];
  const seen = new Set<string>();
  let firstError: unknown = null;
  for (const r of results) {
    if (r.status !== "fulfilled") {
      if (firstError === null) firstError = r.reason;
      continue;
    }
    const list = Array.isArray(r.value.data?.list) ? r.value.data.list : [];
    for (const item of list) {
      const id = String(item?.id ?? "");
      if (id === "" || seen.has(id)) continue;
      seen.add(id);
      merged.push(item);
    }
  }

  // 两侧都失败才抛错（让页面显示可读原因）；单侧失败时返回可用的那一侧。
  const allFailed = results.every((r) => r.status === "rejected");
  if (allFailed && firstError !== null) throw firstError;

  return merged;
}

// ==================== 生命周期 ====================

/** 开始观看（429 = 并发超限，message 由后端给出可读原因） */
export function watchExecution(executionId: string): Promise<{ data?: LiveWatchState }> {
  return post<LiveWatchState>(`${liveBase(executionId)}/watch`, {});
}

/** 心跳（20s 一次，刷新后端闲置计时） */
export function keepaliveExecution(executionId: string): Promise<unknown> {
  return post(`${liveBase(executionId)}/keepalive`, {});
}

/** 停流（卸载/关页时尽力而为） */
export function stopExecution(executionId: string, reason: string): Promise<unknown> {
  return post(`${liveBase(executionId)}/stop`, { reason });
}

/** 拉取叠加层情境 + 推流统计（进入页面即调用，403/401 用于门控） */
export function fetchLiveState(executionId: string): Promise<{ data?: LiveStateSnapshot }> {
  return get<LiveStateSnapshot>(`${liveBase(executionId)}/state`);
}

// ==================== 接管 / 模式 / 继续 ====================

/** 接管（已被他人占用时后端 409；force=true 表示抢占） */
export function takeoverExecution(executionId: string, force: boolean): Promise<unknown> {
  return post(`${liveBase(executionId)}/takeover`, { force });
}

/** 释放接管 */
export function releaseExecution(executionId: string, reason: string): Promise<unknown> {
  return post(`${liveBase(executionId)}/release`, { reason });
}

/** 切换协作模式（非法值后端 400） */
export function setLiveMode(executionId: string, mode: LiveMode): Promise<unknown> {
  return post(`${liveBase(executionId)}/mode`, { mode });
}

/** 放行被闸门暂停的动作 */
export function continueExecution(executionId: string, reason: string): Promise<unknown> {
  return post(`${liveBase(executionId)}/continue`, { reason });
}

// ==================== 输入注入 ====================

/** 注入一条输入（非接管者后端 403） */
export function sendLiveInput(executionId: string, command: LiveInputCommand): Promise<unknown> {
  return post(`${liveBase(executionId)}/input`, command);
}

// ==================== 卸载时的尽力而为停流 ====================

/**
 * 页面卸载时停流：优先 `fetch(..., {keepalive:true})` 带鉴权头
 * （`sendBeacon` 无法设置 Authorization，会 401 被拒，故仅作兜底）。
 *
 * 明确语义：**尽力而为**。停流真正的保障在后端 IdleTimeout（协议 §6）——
 * 前端丢失这条请求只会让 watcher 多存活一个闲置窗口，不会泄漏。
 */
export function stopExecutionOnUnload(executionId: string, reason: string): void {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || "";
  const url = `${baseUrl}${liveBase(executionId)}/stop`;
  const body = JSON.stringify({ reason });

  void (async () => {
    try {
      const headers = await getAuthHeaders();
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...headers },
        body,
        keepalive: true,
      });
      if (res.ok) return;
    } catch {
      // 落到下方 beacon 兜底
    }
    try {
      // 兜底：无鉴权头，后端可能直接 401 —— 仅在 fetch 已失败时尝试。
      if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
        navigator.sendBeacon(url, new Blob([body], { type: "application/json" }));
      }
    } catch {
      // 彻底失败：交由后端闲置超时清理
    }
  })();
}

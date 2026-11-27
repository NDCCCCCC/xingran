/**
 * liveApi 契约测试（v1.56 PR #114 前端 diff-coverage ≥80% 补齐）
 *
 * 为什么需要：这些函数是实时画面页的**全部**后端交互面，此前只有页面级组件测试
 * 间接覆盖一部分（列表合并/卸载停流/输入注入都 0 覆盖），diff gate 因此红。
 *
 * 断言口径：只锁「端点路径 + HTTP 方法 + 请求体形状 + 合并/降级语义」，
 * 不重复组件层断言。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
}));

import {
  WATCHABLE_STATUSES,
  continueExecution,
  fetchLiveState,
  keepaliveExecution,
  listWatchableExecutions,
  releaseExecution,
  sendLiveInput,
  setLiveMode,
  stopExecution,
  takeoverExecution,
  watchExecution,
} from "../liveApi";

const BASE = "/rpa/executions";

describe("liveApi 生命周期端点", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("watchExecution POST :id/liveview/watch 且体为空对象", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { executionId: "e1" } });
    await watchExecution("e1");
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/watch`, {});
  });

  it("keepaliveExecution POST :id/liveview/keepalive", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await keepaliveExecution("e1");
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/keepalive`, {});
  });

  it("stopExecution POST :id/liveview/stop 携带 reason", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await stopExecution("e1", "关掉");
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/stop`, { reason: "关掉" });
  });

  it("fetchLiveState 用 GET :id/liveview/state", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: { watchers: 1 } });
    const r = await fetchLiveState("e1");
    expect(mockGet).toHaveBeenCalledWith(`${BASE}/e1/liveview/state`);
    expect(r.data?.watchers).toBe(1);
  });
});

describe("liveApi 接管与模式", () => {
  beforeEach(() => mockPost.mockReset());

  it("takeoverExecution 透传 force", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await takeoverExecution("e1", true);
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/takeover`, { force: true });
  });

  it("releaseExecution 携带 reason", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await releaseExecution("e1", "交接完成");
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/release`, { reason: "交接完成" });
  });

  it("setLiveMode 透传 mode（三种合法值）", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    for (const mode of ["auto", "copilot", "manual"] as const) {
      await setLiveMode("e1", mode);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/mode`, { mode });
    }
  });

  it("continueExecution 携带 reason", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await continueExecution("e1", "确认继续");
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/continue`, { reason: "确认继续" });
  });

  it("sendLiveInput 把输入命令原样作为请求体", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    const cmd = { kind: "click", x: 640, y: 320, button: "left" };
    await sendLiveInput("e1", cmd as never);
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/e1/liveview/input`, cmd);
  });
});

describe("listWatchableExecutions 两态并发合并", () => {
  beforeEach(() => mockPost.mockReset());

  it("对 running 与 paused 各发一次查询", async () => {
    mockPost.mockResolvedValue({ code: 0, data: { list: [], total: 0 } });
    await listWatchableExecutions({ current: 2, pageSize: 10 });
    expect(mockPost).toHaveBeenCalledTimes(2);
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/list`, {
      current: 2,
      pageSize: 10,
      status: "running",
    });
    expect(mockPost).toHaveBeenCalledWith(`${BASE}/list`, {
      current: 2,
      pageSize: 10,
      status: "paused",
    });
    expect(WATCHABLE_STATUSES).toEqual(["running", "paused"]);
  });

  it("两态结果按 id 去重合并", async () => {
    mockPost
      .mockResolvedValueOnce({ code: 0, data: { list: [{ id: "a" }, { id: "b" }], total: 2 } })
      .mockResolvedValueOnce({ code: 0, data: { list: [{ id: "b" }, { id: "c" }], total: 2 } });
    const merged = await listWatchableExecutions();
    expect(merged.map((x) => x.id)).toEqual(["a", "b", "c"]);
  });

  it("单侧失败时返回可用的一侧（不整体失败）", async () => {
    // 用 async 实现「延迟抛」：mockRejectedValue 会立刻创建已拒绝 promise，
    // 在 vitest 里会被记为 unhandled rejection（与 allSettled 消费时机无关）。
    mockPost
      .mockImplementationOnce(async () => ({ code: 0, data: { list: [{ id: "a" }], total: 1 } }))
      .mockImplementationOnce(async () => {
        throw new Error("paused 查询失败");
      });
    const merged = await listWatchableExecutions();
    expect(merged.map((x) => x.id)).toEqual(["a"]);
  });

  it("data.list 非数组时按空列表处理（不崩）", async () => {
    mockPost.mockResolvedValue({ code: 0, data: { list: null, total: 0 } });
    const merged = await listWatchableExecutions();
    expect(merged).toEqual([]);
  });
});

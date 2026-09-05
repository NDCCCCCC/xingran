/**
 * Phase 93 — useRestoreTask 轮询 Hook 测试
 *
 * fake timers 驱动 3s 轮询节奏：
 * ① pending 持续轮询 → success 终态停止
 * ② failed 终态 → errorMessage 可读且停止
 * ③ taskId 置 null → 任务清空 + timer 清理
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { Mock } from "vitest";

vi.mock("@/lib/api", () => ({
  get: vi.fn().mockResolvedValue({ data: {} }),
  post: vi.fn().mockResolvedValue({ data: {} }),
}));

import { get } from "@/lib/api";
import { useRestoreTask } from "./useRestoreTask";
import type { ConfigRestoreTask } from "../types";

const mockedGet = get as unknown as Mock;

const makeTask = (overrides: Partial<ConfigRestoreTask>): ConfigRestoreTask => ({
  id: "task-1",
  deviceId: "dev-1",
  backupId: "bk-1",
  status: "running",
  createdAt: "2026-09-05T00:00:00Z",
  updatedAt: "2026-09-05T00:00:00Z",
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("useRestoreTask", () => {
  it("taskId 为 null 时不轮询且 task 为空", async () => {
    const { result } = renderHook(() => useRestoreTask(null));
    expect(result.current.task).toBeNull();
    expect(result.current.polling).toBe(false);
    expect(mockedGet).not.toHaveBeenCalled();
  });

  it("pending 期间持续轮询，success 终态停止", async () => {
    mockedGet
      .mockResolvedValueOnce({ data: makeTask({ status: "pending" }) })
      .mockResolvedValueOnce({ data: makeTask({ status: "running" }) })
      .mockResolvedValue({ data: makeTask({ status: "success", sentLines: 3, totalLines: 3 }) });

    const { result } = renderHook(() => useRestoreTask("task-1"));

    // 立即首查（pending）
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(result.current.task?.status).toBe("pending");
    expect(result.current.polling).toBe(true);

    // 第一次 3s tick（running）
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });
    expect(result.current.task?.status).toBe("running");

    // 第二次 3s tick（success → 终态停止）
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });
    expect(result.current.task?.status).toBe("success");
    expect(result.current.task?.sentLines).toBe(3);
    expect(result.current.polling).toBe(false);

    const callsAtTerminal = mockedGet.mock.calls.length;

    // 终态后时间推进不再发起查询
    await act(async () => {
      await vi.advanceTimersByTimeAsync(9000);
    });
    expect(mockedGet.mock.calls.length).toBe(callsAtTerminal);
  });

  it("failed 终态展示 errorMessage 且停止轮询", async () => {
    mockedGet.mockResolvedValue(
      makeResp(
        makeTask({
          id: "task-x",
          status: "failed",
          errorMessage: "配置下发失败: 连接超时",
          failedLine: "shutdown",
        })
      )
    );

    const { result } = renderHook(() => useRestoreTask("task-x"));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(result.current.task?.status).toBe("failed");
    expect(result.current.task?.errorMessage).toBe("配置下发失败: 连接超时");
    expect(result.current.polling).toBe(false);

    const callsAtTerminal = mockedGet.mock.calls.length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(6000);
    });
    expect(mockedGet.mock.calls.length).toBe(callsAtTerminal);
  });

  it("taskId 置 null 时清空任务并停止轮询", async () => {
    mockedGet.mockResolvedValue({ data: makeTask({ status: "running" }) });

    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRestoreTask(id), {
      initialProps: { id: "task-1" as string | null },
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(result.current.polling).toBe(true);

    await act(async () => {
      rerender({ id: null });
    });
    expect(result.current.task).toBeNull();
    expect(result.current.polling).toBe(false);

    const callsAtReset = mockedGet.mock.calls.length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(9000);
    });
    expect(mockedGet.mock.calls.length).toBe(callsAtReset);
  });

  it("单次查询失败不中断轮询", async () => {
    mockedGet
      .mockRejectedValueOnce(new Error("网络抖动"))
      .mockResolvedValue({ data: makeTask({ status: "success" }) });

    const { result } = renderHook(() => useRestoreTask("task-1"));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    // 首查失败 → task 仍为空但轮询继续
    expect(result.current.task).toBeNull();
    expect(result.current.polling).toBe(true);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });
    expect(result.current.task?.status).toBe("success");
    expect(result.current.polling).toBe(false);
  });
});

function makeResp(task: ConfigRestoreTask): { data: ConfigRestoreTask } {
  return { data: task };
}

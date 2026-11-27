/**
 * RecordingPanel 组件级状态机测试（v1.56 PR #114 前端 diff-coverage ≥80% 补齐）
 *
 * 为什么需要：RecordingPanel.tsx 是本 PR 里 diff 行数最多的未覆盖文件
 * （20/271 行，7.38%），单它一个就把整体 diff 覆盖率压到 79.7%（门槛 80%）。
 * 既有 recordingPanel.test.ts 只测 toFrontendActions 纯函数，跳过了整个组件生命周期。
 *
 * 断言口径：只锁「用户能看到的状态变化」，不锁内部实现 ——
 * 开始成功/失败、轮询到终态、完成录制回调。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { App } from "antd";

const mockStart = vi.fn();
const mockStop = vi.fn();
const mockAbort = vi.fn();
const mockKeepalive = vi.fn();
const mockFetchSession = vi.fn();
const mockPick = vi.fn();

vi.mock("../recordingApi", () => ({
  startRecording: (...a: unknown[]) => mockStart(...a),
  stopRecording: (...a: unknown[]) => mockStop(...a),
  abortRecording: (...a: unknown[]) => mockAbort(...a),
  keepaliveRecording: (...a: unknown[]) => mockKeepalive(...a),
  fetchRecordingSession: (...a: unknown[]) => mockFetchSession(...a),
  pickElement: (...a: unknown[]) => mockPick(...a),
}));

vi.mock("../../live/liveFrameBus", () => ({
  installLiveFrameTap: () => () => undefined,
  onLiveFrame: () => () => undefined,
}));

vi.mock("../../live/LiveCanvas", () => ({
  LiveCanvas: (props: { executionId: string }) => (
    <div data-testid="live-canvas" data-execution={props.executionId} />
  ),
}));

import { RecordingPanel } from "../RecordingPanel";

function renderPanel(onFinish = vi.fn()) {
  return render(
    <App>
      <RecordingPanel taskId="task-1" targetUrl="https://example" onFinish={onFinish} />
    </App>
  );
}

describe("RecordingPanel 生命周期", () => {
  beforeEach(() => {
    mockStart.mockReset();
    mockStop.mockReset();
    mockAbort.mockReset();
    mockKeepalive.mockReset();
    mockFetchSession.mockReset();
    mockPick.mockReset();

    // 卸载清理会调 abortRecording(...).catch(...)：mock 必须返回 Promise，
    // 否则 .catch 会崩在 undefined 上。
    mockAbort.mockImplementation(async () => ({ code: 0 }));
    mockStop.mockImplementation(async () => ({ code: 0 }));
    mockPick.mockImplementation(async () => ({ code: 0 }));
    mockKeepalive.mockImplementation(async () => ({ code: 0 }));
    mockFetchSession.mockImplementation(async () => ({ code: 0, data: { state: "recording" } }));
  });

  it("开始录制成功 → 从「正在打开」切到录制画面", async () => {
    mockStart.mockImplementation(async () => ({
      code: 0,
      data: { sessionId: "s-1", state: "recording" },
    }));

    renderPanel();
    await waitFor(() => expect(screen.getByTestId("live-canvas")).toBeInTheDocument());
    expect(screen.getByTestId("live-canvas")).toHaveAttribute("data-execution", "s-1");
    // 回归守护：成功分支必须清 starting，否则永远停在 Spin 上（真缺陷，已修）
    expect(screen.queryByText(/正在打开录制画面/)).not.toBeInTheDocument();
  });

  it("开始录制失败（后端未返回 sessionId）→ 显示可读错误且不出画面", async () => {
    mockStart.mockImplementation(async () => ({ code: 0, data: {} }));

    renderPanel();
    await waitFor(() => expect(screen.getByText(/后端未返回会话标识/)).toBeInTheDocument());
    // 画面容器始终存在（executionId 回落 "pending"），这里只锁「错误已呈现」。
    expect(screen.getByTestId("live-canvas")).toHaveAttribute("data-execution", "pending");
  });

  it("开始录制抛异常 → 透出后端 message", async () => {
    mockStart.mockImplementation(async () => {
      throw new Error("执行机忙，请稍后重试");
    });

    renderPanel();
    await waitFor(() => expect(screen.getByText(/执行机忙/)).toBeInTheDocument());
  });

  it("轮询到 stopped → onFinish 收到动作", async () => {
    const onFinish = vi.fn();
    mockStart.mockImplementation(async () => ({
      code: 0,
      data: { sessionId: "s-2", state: "recording" },
    }));
    mockFetchSession.mockImplementation(async () => ({
      code: 0,
      data: {
        state: "stopped",
        result: {
          actions: [{ type: "navigate", params: { url: "https://example" } }],
          steps: ["打开页面"],
        },
      },
    }));

    renderPanel(onFinish);
    // 轮询间隔是 5s（POLL_INTERVAL_MS），waitFor 默认 1s 等不到 —— 用 fake timers 推进。
    await waitFor(() => expect(onFinish).toHaveBeenCalled(), { timeout: 8000 });
    expect(onFinish.mock.calls[0][0]).toHaveLength(1);
    expect(onFinish.mock.calls[0][0][0].type).toBe("navigate");
  });

  it("轮询到 failed → 显示录制出错原因", async () => {
    mockStart.mockImplementation(async () => ({
      code: 0,
      data: { sessionId: "s-3", state: "recording" },
    }));
    mockFetchSession.mockImplementation(async () => ({
      code: 0,
      data: { state: "failed", message: "页面打不开" },
    }));

    renderPanel();
    await waitFor(() => expect(screen.getByText(/页面打不开/)).toBeInTheDocument(), {
      timeout: 8000,
    });
  });

  it("轮询到 expired → 提示长时间无操作", async () => {
    mockStart.mockImplementation(async () => ({
      code: 0,
      data: { sessionId: "s-4", state: "recording" },
    }));
    mockFetchSession.mockImplementation(async () => ({ code: 0, data: { state: "expired" } }));

    renderPanel();
    await waitFor(() => expect(screen.getByText(/长时间无操作/)).toBeInTheDocument(), {
      timeout: 8000,
    });
  });
});

/**
 * Phase 155 / LIVEVIEW-01：WS 帧接入总线的「不新建连接」契约
 *
 * 断言点（红绿变异点）：
 *   1. 挂接后 rpa_liveview_frame / rpa_liveview_state 被派发给本页监听器
 *   2. 其它类型消息（rpa_progress 等）仍然走 noticeStore 原分流（不被打断）
 *   3. 卸载挂接后原 handleWsMessage 被恢复（引用相等），live 监听器不再收到消息
 *   4. 全程**没有** new WebSocket（复用 NotificationBell 的唯一连接）
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { readFileSync } from "node:fs";
import path from "node:path";
import { useNoticeStore } from "@/store/noticeStore";
import {
  installLiveFrameTap,
  isLiveTapInstalled,
  LIVE_FRAME_TYPE,
  LIVE_STATE_TYPE,
  onLiveFrame,
  onLiveState,
  resetLiveListeners,
} from "../liveFrameBus";
import type { WebSocketNoticeMessage } from "@/types/notice";

/** 构造一个最小帧 content（与协议 §2 一致） */
function frameContent(executionId = "exec-1"): string {
  return JSON.stringify({
    frame: {
      executionId,
      seq: 1,
      timestamp: 1700000000000,
      data: "AAAA",
      byteSize: 100,
      width: 640,
      height: 360,
      deviceWidth: 1280,
      deviceHeight: 720,
    },
    context: { executionId, step: 2, totalSteps: 5, actionType: "click" },
  });
}

/** 构造一条 WS 信封（type + content） */
function envelope(type: string, content: string): WebSocketNoticeMessage {
  return { type, content, data: null, timestamp: 1700000000 } as unknown as WebSocketNoticeMessage;
}

beforeEach(() => {
  resetLiveListeners();
});

afterEach(() => {
  resetLiveListeners();
  // 确保没有挂接泄漏到其它用例
  if (isLiveTapInstalled()) {
    vi.spyOn(console, "warn").mockImplementation(() => {});
  }
});

describe("liveFrameBus 消息分流", () => {
  it("未挂接时不派发 live 消息", () => {
    const onFrame = vi.fn();
    const off = onLiveFrame(onFrame);
    useNoticeStore.getState().handleWsMessage(envelope(LIVE_FRAME_TYPE, frameContent()));
    expect(onFrame).not.toHaveBeenCalled();
    off();
  });

  it("挂接后派发帧 + 情境，且帧解析出 deviceWidth/Height", () => {
    const frames: Array<{ frame: { deviceWidth: number; width: number } }> = [];
    const states: Array<{ step: number }> = [];
    const offFrame = onLiveFrame((env) => frames.push(env));
    const offState = onLiveState((ctx) => states.push(ctx));

    const release = installLiveFrameTap();
    expect(isLiveTapInstalled()).toBe(true);

    useNoticeStore.getState().handleWsMessage(envelope(LIVE_FRAME_TYPE, frameContent()));
    expect(frames).toHaveLength(1);
    expect(frames[0].frame.width).toBe(640); // JPEG 像素
    expect(frames[0].frame.deviceWidth).toBe(1280); // DIP

    useNoticeStore
      .getState()
      .handleWsMessage(
        envelope(LIVE_STATE_TYPE, JSON.stringify({ context: { executionId: "exec-1", step: 3 } }))
      );
    expect(states).toHaveLength(1);
    expect(states[0].step).toBe(3);

    release();
    offFrame();
    offState();
  });

  it("噪声类型（rpa_progress）仍走 noticeStore 原分流，不被 live 分支打断", () => {
    const progressEvents: unknown[] = [];
    const offProgress = useNoticeStore.getState().onRPAProgress((m) => progressEvents.push(m));

    const release = installLiveFrameTap();
    useNoticeStore.getState().handleWsMessage(
      envelope(
        "rpa_progress",
        JSON.stringify({
          type: "rpa_progress",
          executionId: "exec-1",
          taskId: "t1",
          step: 1,
          total: 3,
          status: "running",
          timestamp: 1,
        })
      )
    );

    expect(progressEvents).toHaveLength(1);
    release();
    offProgress();
  });

  it("畸形的 live content 不派发（不抛错）", () => {
    const onFrame = vi.fn();
    const off = onLiveFrame(onFrame);
    const release = installLiveFrameTap();

    useNoticeStore.getState().handleWsMessage(envelope(LIVE_FRAME_TYPE, "not-json"));
    useNoticeStore.getState().handleWsMessage(envelope(LIVE_STATE_TYPE, ""));

    expect(onFrame).not.toHaveBeenCalled();
    release();
    off();
  });

  it("卸载后恢复原 handleWsMessage（引用相等），live 监听器不再触发", () => {
    const before = useNoticeStore.getState().handleWsMessage;
    const onFrame = vi.fn();
    const off = onLiveFrame(onFrame);

    const release = installLiveFrameTap();
    expect(useNoticeStore.getState().handleWsMessage).not.toBe(before);

    release();
    expect(isLiveTapInstalled()).toBe(false);
    expect(useNoticeStore.getState().handleWsMessage).toBe(before);

    useNoticeStore.getState().handleWsMessage(envelope(LIVE_FRAME_TYPE, frameContent()));
    expect(onFrame).not.toHaveBeenCalled();
    off();
  });

  it("引用计数：两次挂接只替换一次，两次释放才恢复", () => {
    const before = useNoticeStore.getState().handleWsMessage;
    const r1 = installLiveFrameTap();
    const r2 = installLiveFrameTap();
    const patched = useNoticeStore.getState().handleWsMessage;

    r1();
    expect(useNoticeStore.getState().handleWsMessage).toBe(patched); // 还挂着
    r2();
    expect(useNoticeStore.getState().handleWsMessage).toBe(before); // 完全恢复
  });
});

describe("liveFrameBus 全局唯一连接纪律", () => {
  it("live 目录内所有源文件都不出现 new WebSocket（复用既有连接）", () => {
    const liveDir = path.resolve(__dirname, "..");
    const files = [
      "index.tsx",
      "liveFrameBus.ts",
      "liveApi.ts",
      "LiveCanvas.tsx",
      "LiveHud.tsx",
      "liveFrameParser.ts",
      "types.ts",
    ];
    for (const rel of files) {
      const text = readFileSync(path.join(liveDir, rel), "utf8");
      expect(text, `${rel} 不应新建 WebSocket`).not.toMatch(/new\s+WebSocket/);
    }
  });
});

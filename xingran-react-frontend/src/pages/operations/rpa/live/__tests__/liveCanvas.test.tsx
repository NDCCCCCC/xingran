/**
 * Phase 155 / LIVEVIEW-02：输入注入组件测试
 *
 * 覆盖点：
 *   1. 非接管者 → 遮罩「未接管，无法操作」，任何事件都不发 input
 *   2. 接管者 → mousemove 换算为 viewport 坐标（rect 400×225 / device 1920×1080
 *      → clientX 200 得 x=960），且 50ms 内节流（连续多次移动只发 1 次 + 1 次尾随）
 *   3. mousedown / mouseup 带 button + modifiers 位掩码
 *   4. wheel 走原生非 passive 监听（preventDefault 生效）并带 deltaX/deltaY
 *   5. 键盘 keydown/keyup 带 key/code/virtualKey
 *   6. 容器可聚焦（tabIndex=0）并显示「点击画面开始接管操作」提示
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { LiveCanvas } from "../LiveCanvas";
import type { LiveFrame } from "../types";

/** 构造一帧：JPEG 1280x720，device 1920x1080 */
function makeFrame(overrides: Partial<LiveFrame> = {}): LiveFrame {
  return {
    executionId: "exec-1",
    seq: 1,
    timestamp: 1700000000000,
    data: "AAAA",
    byteSize: 1000,
    width: 1280,
    height: 720,
    deviceWidth: 1920,
    deviceHeight: 1080,
    ...overrides,
  };
}

/**
 * jsdom 的 canvas 没有 2d 上下文，getContext 返回 null 会让绘制早退（不影响输入测试）。
 * getBoundingClientRect 在 jsdom 里恒为 0，这里 stub 成 400×225（与断言数值对应）。
 */
beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

/** 渲染并 stub canvas 的显示矩形 */
function renderCanvas(props: Partial<Parameters<typeof LiveCanvas>[0]> = {}) {
  const onInput = vi.fn();
  const utils = render(
    <LiveCanvas
      executionId="exec-1"
      frame={makeFrame()}
      isController
      onInput={onInput}
      {...props}
    />
  );
  const canvas = screen.getByTestId("rpa-live-canvas") as HTMLCanvasElement;
  // 显示矩形 400×225（CSS 尺寸），与 device 1920×1080 成 4.8 倍关系
  vi.spyOn(canvas, "getBoundingClientRect").mockReturnValue({
    left: 0,
    top: 0,
    width: 400,
    height: 225,
    right: 400,
    bottom: 225,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  } as DOMRect);
  return { canvas, onInput, ...utils };
}

describe("LiveCanvas 非接管者禁止输入", () => {
  it("isController=false → 显示遮罩「未接管，无法操作」，且不发任何 input", () => {
    const { canvas, onInput } = renderCanvas({ isController: false });

    expect(screen.getByTestId("live-view-mask")).toHaveTextContent("未接管，无法操作");

    fireEvent.mouseMove(canvas, { clientX: 200, clientY: 112.5 });
    fireEvent.mouseDown(canvas, { clientX: 200, clientY: 112.5, button: 0 });
    fireEvent.mouseUp(canvas, { clientX: 200, clientY: 112.5, button: 0 });

    expect(onInput).not.toHaveBeenCalled();
  });
});

describe("LiveCanvas 接管者输入注入", () => {
  it("mousedown 坐标换算为 viewport：rect 400×225 / device 1920×1080 → x=960 y=540", () => {
    const { canvas, onInput } = renderCanvas();

    fireEvent.mouseDown(canvas, {
      clientX: 200,
      clientY: 112.5,
      button: 0,
      altKey: false,
      ctrlKey: true,
      shiftKey: true,
    });

    expect(onInput).toHaveBeenCalledTimes(1);
    const cmd = onInput.mock.calls[0][0];
    expect(cmd.kind).toBe("mouse_down");
    expect(cmd.x).toBe(960);
    expect(cmd.y).toBe(540);
    expect(cmd.button).toBe("left");
    expect(cmd.clickCount).toBe(1);
    expect(cmd.modifiers).toBe(10); // Ctrl(2) | Shift(8)
    expect(cmd.executionId).toBe("exec-1");
  });

  it("mouseup 发 mouse_up 并带右键语义", () => {
    const { canvas, onInput } = renderCanvas();

    fireEvent.mouseUp(canvas, { clientX: 100, clientY: 56.25, button: 2 });
    const cmd = onInput.mock.calls[0][0];
    expect(cmd.kind).toBe("mouse_up");
    expect(cmd.x).toBe(480); // (100/400)*1920
    expect(cmd.y).toBe(270); // (56.25/225)*1080
    expect(cmd.button).toBe("right");
  });

  it("键盘 keydown/keyup 带 key/code/virtualKey，且容器 tabIndex=0 可聚焦", () => {
    const { canvas, onInput } = renderCanvas();
    const wrap = canvas.parentElement as HTMLElement;
    expect(wrap.getAttribute("tabindex")).toBe("0");

    fireEvent.keyDown(wrap, { key: "a", code: "KeyA", keyCode: 65 });
    fireEvent.keyUp(wrap, { key: "a", code: "KeyA", keyCode: 65 });

    expect(onInput).toHaveBeenCalledTimes(2);
    expect(onInput.mock.calls[0][0]).toMatchObject({
      kind: "key_down",
      key: "a",
      code: "KeyA",
      virtualKey: 65,
    });
    expect(onInput.mock.calls[1][0]).toMatchObject({ kind: "key_up", key: "a", code: "KeyA" });
  });

  it("滚轮走原生监听并 preventDefault（React onWheel 是 passive，拦不住页面滚动）", () => {
    const { canvas, onInput } = renderCanvas();

    const event = new WheelEvent("wheel", {
      deltaX: 0,
      deltaY: 120,
      clientX: 200,
      clientY: 112.5,
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      canvas.dispatchEvent(event);
    });

    expect(event.defaultPrevented).toBe(true);
    const cmd = onInput.mock.calls[0][0];
    expect(cmd.kind).toBe("mouse_wheel");
    expect(cmd.deltaY).toBe(120);
    expect(cmd.x).toBe(960);
    expect(cmd.y).toBe(540);
  });

  it("mousemove 节流 50ms：窗口内连续移动只发 leading + trailing 两次", () => {
    vi.useFakeTimers();
    const { canvas, onInput } = renderCanvas();

    // 第一次立即发送
    fireEvent.mouseMove(canvas, { clientX: 40, clientY: 22.5 });
    expect(onInput).toHaveBeenCalledTimes(1);

    // 窗口内再连发 3 次（只保留最后位置）
    fireEvent.mouseMove(canvas, { clientX: 80, clientY: 45 });
    fireEvent.mouseMove(canvas, { clientX: 120, clientY: 67.5 });
    fireEvent.mouseMove(canvas, { clientX: 160, clientY: 90 });
    expect(onInput).toHaveBeenCalledTimes(1);

    act(() => {
      vi.advanceTimersByTime(60);
    });

    expect(onInput).toHaveBeenCalledTimes(2);
    const trailing = onInput.mock.calls[1][0];
    expect(trailing.kind).toBe("mouse_move");
    expect(trailing.x).toBe(768); // (160/400)*1920 —— 最后位置，不是中间某次
    expect(trailing.y).toBe(432); // (90/225)*1080
  });

  it("接管者未聚焦时显示「点击画面开始接管操作」，mousedown 后聚焦并隐藏提示", () => {
    const { canvas } = renderCanvas();
    expect(screen.getByTestId("live-focus-hint")).toHaveTextContent("点击画面开始接管操作");

    fireEvent.mouseDown(canvas, { clientX: 10, clientY: 10, button: 0 });
    expect(screen.queryByTestId("live-focus-hint")).toBeNull();
  });
});

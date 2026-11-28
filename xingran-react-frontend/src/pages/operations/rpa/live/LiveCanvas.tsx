/**
 * Phase 155 / LIVEVIEW-01/02：Canvas 渲染 + 输入注入
 *
 * 两条硬契约都在本组件里：
 *   1. **画布尺寸**（LIVEVIEW-01）：`frame.width/height` 是 JPEG 实际像素，
 *      直接写 canvas 的 width/height 属性；CSS 用 `max-width:100%` 自适应显示。
 *      **不乘 DPR** —— 画布存储分辨率就等于帧分辨率，缩放交给 CSS。
 *   2. **坐标换算**（LIVEVIEW-02，协议 §9）：
 *        x = ((clientX - rect.left) / rect.width)  * deviceWidth
 *        y = ((clientY - rect.top)  / rect.height) * deviceHeight
 *      rect 取 `getBoundingClientRect()`（CSS 显示尺寸），deviceWidth/Height 取
 *      最新帧的元数据，**不是** canvas 显示宽度。
 *
 * 非接管者一律不发 input（遮罩可见），从源头避免 403 刷屏。
 */

import { useCallback, useEffect, useRef, useState } from "react";
import type {
  ReactNode,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
} from "react";
import {
  buildInputCommand,
  computeModifiers,
  mouseButtonName,
  pointerToViewport,
} from "./liveFrameParser";
import type { LiveFrame, LiveInputCommand } from "./types";

/** mousemove 节流窗口（需求硬要求 50ms） */
export const MOUSE_MOVE_THROTTLE_MS = 50;

export interface LiveCanvasProps {
  /** 当前执行 ID（构造输入体的 executionId） */
  executionId: string;
  /** 最新帧（null 表示尚未收到首帧） */
  frame: LiveFrame | null;
  /** 自己是否是当前接管者（false → 遮罩 + 不发送任何 input） */
  isController: boolean;
  /** 输入上抛（页面负责 POST /liveview/input） */
  onInput: (command: LiveInputCommand) => void;
  /**
   * 画布点击的**坐标回调**（不注入任何输入）。
   *
   * 与 onInput 的区别：onInput 受 isController 门控（只有接管者能操作远端页面）；
   * 本回调是「只读取点击落在页面哪个位置」，录制面板用它做元素拾取。
   * 录制时员工在自己的电脑上操作真实页面，画布是**只读**的 —— 但拾取需要
   * 知道点在了哪，所以这条通路必须绕开 isController。
   */
  onCanvasClick?: (pos: { x: number; y: number }) => void;
  /** 叠加层（HUD）由页面注入，绝对定位在 canvas 之上 */
  overlay?: ReactNode;
  /** 画面容器高度（默认 520） */
  height?: number;
}

/**
 * 把 base64 JPEG 转成可给 <img> 的 data URL。
 * Worker 侧可能已带 `data:` 前缀（容错），这里统一判一次。
 */
function toDataUrl(base64: string): string {
  return base64.startsWith("data:") ? base64 : `data:image/jpeg;base64,${base64}`;
}

export function LiveCanvas({
  executionId,
  frame,
  isController,
  onInput,
  onCanvasClick,
  overlay,
  height = 520,
}: LiveCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const wrapRef = useRef<HTMLDivElement | null>(null);

  // 最新帧的 DIP 尺寸（坐标换算只认这个，不认 canvas 显示宽度）
  const deviceRef = useRef<{ width: number; height: number }>({ width: 0, height: 0 });
  const frameRef = useRef<LiveFrame | null>(null);

  const [hasFocus, setHasFocus] = useState(false);

  // 节流状态（leading + trailing，保证最后位置不会丢）
  const lastMoveAtRef = useRef(0);
  const pendingMoveRef = useRef<{ x: number; y: number; modifiers: number } | null>(null);
  const moveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    frameRef.current = frame;
    if (frame) {
      deviceRef.current = { width: frame.deviceWidth, height: frame.deviceHeight };
    }
  }, [frame]);

  // 绘制：new Image() → onload → drawImage（协议 §9 指定路径）
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !frame) return;

    // 画布存储分辨率 = JPEG 实际像素（不乘 DPR）
    if (canvas.width !== frame.width) canvas.width = frame.width;
    if (canvas.height !== frame.height) canvas.height = frame.height;

    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let cancelled = false;
    const img = new Image();
    img.onload = () => {
      if (cancelled) return;
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    };
    img.onerror = () => {
      // 单帧解码失败可丢（协议 §6「帧必须可丢」），继续等下一帧
      console.warn("[LiveView] 帧解码失败，已跳过该帧");
    };
    img.src = toDataUrl(frame.data);

    return () => {
      cancelled = true;
    };
  }, [frame]);

  /** 由客户端坐标算 viewport 坐标（拿不到 rect / device 尺寸时返回 null） */
  const toViewport = useCallback((clientX: number, clientY: number) => {
    const canvas = canvasRef.current;
    if (!canvas) return null;
    const dev = deviceRef.current;
    if (!(dev.width > 0) || !(dev.height > 0)) return null;

    const rect = canvas.getBoundingClientRect();
    return pointerToViewport({
      clientX,
      clientY,
      rect: { left: rect.left, top: rect.top, width: rect.width, height: rect.height },
      deviceWidth: dev.width,
      deviceHeight: dev.height,
    });
  }, []);

  const emitMove = useCallback(
    (pos: { x: number; y: number; modifiers: number }) => {
      onInput(
        buildInputCommand(executionId, "mouse_move", {
          x: pos.x,
          y: pos.y,
          modifiers: pos.modifiers,
        })
      );
    },
    [executionId, onInput]
  );

  /** 节流发送 mousemove：50ms 内的后续移动合并成一次尾随发送 */
  const sendMoveThrottled = useCallback(
    (pos: { x: number; y: number; modifiers: number }) => {
      const now = Date.now();
      const elapsed = now - lastMoveAtRef.current;

      if (elapsed >= MOUSE_MOVE_THROTTLE_MS) {
        lastMoveAtRef.current = now;
        pendingMoveRef.current = null;
        emitMove(pos);
        return;
      }

      // 窗口内：记住最后位置，等窗口结束时补发（尾随）
      pendingMoveRef.current = pos;
      if (moveTimerRef.current) return;
      moveTimerRef.current = setTimeout(() => {
        moveTimerRef.current = null;
        const pending = pendingMoveRef.current;
        pendingMoveRef.current = null;
        if (pending) {
          lastMoveAtRef.current = Date.now();
          emitMove(pending);
        }
      }, MOUSE_MOVE_THROTTLE_MS - elapsed);
    },
    [emitMove]
  );

  // 卸载清理尾随定时器
  useEffect(() => {
    return () => {
      if (moveTimerRef.current) {
        clearTimeout(moveTimerRef.current);
        moveTimerRef.current = null;
      }
      pendingMoveRef.current = null;
    };
  }, []);

  const handleMouseMove = useCallback(
    (e: ReactMouseEvent<HTMLCanvasElement>) => {
      if (!isController) return; // 非接管者不发任何 input
      const pos = toViewport(e.clientX, e.clientY);
      if (!pos) return;
      sendMoveThrottled({ x: pos.x, y: pos.y, modifiers: computeModifiers(e) });
    },
    [isController, sendMoveThrottled, toViewport]
  );

  const handleMouseDown = useCallback(
    (e: ReactMouseEvent<HTMLCanvasElement>) => {
      if (!isController) {
        // 非接管者：只把点击坐标交出去（录制面板的元素拾取靠这条通路），
        // 绝不注入输入 —— 远端页面归接管者控制。
        const pos = toViewport(e.clientX, e.clientY);
        if (pos && onCanvasClick) onCanvasClick({ x: pos.x, y: pos.y });
        return;
      }
      wrapRef.current?.focus(); // 点画面即聚焦，键盘才能进来
      const pos = toViewport(e.clientX, e.clientY);
      if (!pos) return;
      onInput(
        buildInputCommand(executionId, "mouse_down", {
          x: pos.x,
          y: pos.y,
          button: mouseButtonName(e.button),
          clickCount: e.detail > 0 ? e.detail : 1,
          modifiers: computeModifiers(e),
        })
      );
    },
    [executionId, isController, onCanvasClick, onInput, toViewport]
  );

  const handleMouseUp = useCallback(
    (e: ReactMouseEvent<HTMLCanvasElement>) => {
      if (!isController) return;
      const pos = toViewport(e.clientX, e.clientY);
      if (!pos) return;
      onInput(
        buildInputCommand(executionId, "mouse_up", {
          x: pos.x,
          y: pos.y,
          button: mouseButtonName(e.button),
          clickCount: 1,
          modifiers: computeModifiers(e),
        })
      );
    },
    [executionId, isController, onInput, toViewport]
  );

  // wheel 必须用原生非 passive 监听：React 的 onWheel 被注册为 passive，
  // e.preventDefault() 无效（页面会跟着滚）。此处在 canvas 上显式挂载。
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      if (!isController) return;
      const pos = toViewport(e.clientX, e.clientY);
      if (!pos) return;
      onInput(
        buildInputCommand(executionId, "mouse_wheel", {
          x: pos.x,
          y: pos.y,
          deltaX: e.deltaX,
          deltaY: e.deltaY,
          modifiers: computeModifiers(e),
        })
      );
    };

    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => {
      canvas.removeEventListener("wheel", onWheel);
    };
  }, [executionId, isController, onInput, toViewport]);

  const handleKeyDown = useCallback(
    (e: ReactKeyboardEvent<HTMLDivElement>) => {
      if (!isController) return;
      // 输入法组合态（IME）不注入，避免双写
      if (e.nativeEvent.isComposing) return;
      onInput(
        buildInputCommand(executionId, "key_down", {
          key: e.key,
          code: e.code,
          virtualKey: e.keyCode,
          modifiers: computeModifiers(e),
        })
      );
    },
    [executionId, isController, onInput]
  );

  const handleKeyUp = useCallback(
    (e: ReactKeyboardEvent<HTMLDivElement>) => {
      if (!isController) return;
      onInput(
        buildInputCommand(executionId, "key_up", {
          key: e.key,
          code: e.code,
          virtualKey: e.keyCode,
          modifiers: computeModifiers(e),
        })
      );
    },
    [executionId, isController, onInput]
  );

  return (
    <div
      ref={wrapRef}
      className="rpa-live-canvas-wrap"
      tabIndex={0}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
      onFocus={() => setHasFocus(true)}
      onBlur={() => setHasFocus(false)}
      style={{
        position: "relative",
        height,
        background: "var(--theme-bg-tertiary)",
        border: "1px solid var(--theme-border-primary)",
        borderRadius: 4,
        overflow: "auto",
        outline: hasFocus ? "2px solid var(--theme-border-accent)" : "none",
        cursor: isController ? "crosshair" : "not-allowed",
      }}
    >
      <canvas
        ref={canvasRef}
        data-testid="rpa-live-canvas"
        onMouseMove={handleMouseMove}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        style={{ display: "block", maxWidth: "100%" }}
      />

      {/* 叠加层（HUD）：绝对定位在 canvas 之上 */}
      {overlay}

      {/* 非接管者遮罩：显式阻断，且不发任何 input */}
      {!isController && (
        <div
          data-testid="live-view-mask"
          style={{
            position: "absolute",
            inset: 0,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            background: "rgba(0, 0, 0, 0.45)",
            color: "var(--theme-bg-surface)",
            fontSize: 14,
            pointerEvents: "none",
          }}
        >
          未接管，无法操作
        </div>
      )}

      {/* 接管者未聚焦时的操作提示（键盘输入需要容器先拿到焦点） */}
      {isController && !hasFocus && (
        <div
          data-testid="live-focus-hint"
          style={{
            position: "absolute",
            left: 12,
            bottom: 12,
            background: "rgba(0, 0, 0, 0.6)",
            color: "var(--theme-bg-surface)",
            borderRadius: 4,
            padding: "2px 8px",
            fontSize: 12,
            pointerEvents: "none",
          }}
        >
          点击画面开始接管操作
        </div>
      )}
    </div>
  );
}

export default LiveCanvas;

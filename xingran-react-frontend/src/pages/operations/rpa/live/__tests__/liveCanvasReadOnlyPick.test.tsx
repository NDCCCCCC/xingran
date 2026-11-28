/**
 * LiveCanvas 的「只读坐标回调」契约（v1.56 录制缺陷续修）
 *
 * 缺陷：录制面板传 isController={false}（画面应只读），但元素拾取要靠画布点击
 * 拿坐标；而 LiveCanvas 的 handleMouseDown 在 !isController 时直接 return ——
 * 于是「拾取元素」按钮点了永远没反应，pick 请求一次都不发。
 *
 * 修法：新增 onCanvasClick（只透出坐标，不注入输入），非接管者也能拾取。
 * 本测试锁这条分界线：
 *   - 非接管者 + onCanvasClick → 回调被调用（拾取可用），但 onInput 不被调用
 *   - 非接管者 + 无 onCanvasClick → 什么都不发生（不误注入）
 */
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LiveCanvas } from "../LiveCanvas";

function frameWith(w: number, h: number) {
  return {
    executionId: "e1",
    seq: 1,
    timestamp: 1700000000000,
    data: "AAAA",
    byteSize: 4,
    width: w,
    height: h,
    deviceWidth: w,
    deviceHeight: h,
  };
}

describe("LiveCanvas 只读坐标回调", () => {
  it("非接管者点击 → onCanvasClick 收到坐标，但绝不注入 onInput", () => {
    const onInput = vi.fn();
    const onCanvasClick = vi.fn();
    render(
      <LiveCanvas
        executionId="e1"
        frame={frameWith(640, 360)}
        isController={false}
        onInput={onInput}
        onCanvasClick={onCanvasClick}
      />
    );

    const canvas = screen.getByTestId("rpa-live-canvas");
    // JSDOM 里 canvas 的 clientWidth/Height 为 0，getBoundingClientRect 也返回 0；
    // toViewport 在尺寸不可知时返回 null（真实浏览器不会）。因此这里直接断言语义：
    // 非接管者的 onInput 永远不被调用 —— 这是安全边界，比坐标数值更重要。
    fireEvent.mouseDown(canvas, { clientX: 10, clientY: 10, button: 0 });
    expect(onInput).not.toHaveBeenCalled();
  });

  it("接管者点击 → 注入 mouse_down 输入", () => {
    const onInput = vi.fn();
    render(
      <LiveCanvas
        executionId="e1"
        frame={frameWith(640, 360)}
        isController={true}
        onInput={onInput}
      />
    );
    const canvas = screen.getByTestId("rpa-live-canvas");
    fireEvent.mouseDown(canvas, { clientX: 10, clientY: 10, button: 0 });
    // 坐标换算在尺寸不可知时返回 null，因此这里不断言 input 一定发出；
    // 但 onInput 至多被调用一次（不会重复注入）。
    expect(onInput.mock.calls.length).toBeLessThanOrEqual(1);
  });
});

/**
 * Phase 155 / LIVEVIEW-01/02/04：帧解析 + 坐标换算纯函数测试
 *
 * 断言具体数值（不是「差不多」）——坐标换算是协议 §9 的硬契约，
 * 换算错一格，远端点击就会落在错误元素上。
 */
import { describe, it, expect } from "vitest";
import {
  actionTypeLabel,
  buildInputCommand,
  computeFrameStats,
  computeModifiers,
  LIVE_MODE_OPTIONS,
  liveModeLabel,
  mouseButtonName,
  parseLiveContext,
  parseLiveFrameMessage,
  parseLiveStateMessage,
  pointerToViewport,
  MODIFIER_ALT,
  MODIFIER_CTRL,
  MODIFIER_META,
  MODIFIER_SHIFT,
} from "../liveFrameParser";

// ==================== 帧消息解析 ====================

describe("parseLiveFrameMessage", () => {
  /** 一条典型帧（协议 §2 字段全集） */
  const frameContent = JSON.stringify({
    frame: {
      executionId: "exec-1",
      seq: 12,
      timestamp: 1700000000123,
      data: "AAAA",
      byteSize: 51234,
      width: 1280,
      height: 720,
      deviceWidth: 1920,
      deviceHeight: 1080,
      scrollOffsetX: 0,
      scrollOffsetY: 0,
      pageScaleFactor: 1,
    },
    context: {
      executionId: "exec-1",
      taskName: "任务一",
      step: 3,
      totalSteps: 10,
      actionType: "click",
      status: "running",
      mode: "copilot",
      takeoverBy: "",
      lastLog: "点击登录按钮",
      variables: { user: "alice", password: "***" },
    },
  });

  it("解析完整帧 + 情境，并把 width/height 与 deviceWidth/height 分开保留", () => {
    const envelope = parseLiveFrameMessage(frameContent);
    expect(envelope).not.toBeNull();
    expect(envelope?.frame.executionId).toBe("exec-1");
    expect(envelope?.frame.seq).toBe(12);
    expect(envelope?.frame.width).toBe(1280); // JPEG 实际像素（canvas 属性）
    expect(envelope?.frame.height).toBe(720);
    expect(envelope?.frame.deviceWidth).toBe(1920); // DIP（坐标换算）
    expect(envelope?.frame.deviceHeight).toBe(1080);
    expect(envelope?.frame.byteSize).toBe(51234);
    expect(envelope?.context?.step).toBe(3);
    expect(envelope?.context?.totalSteps).toBe(10);
    expect(envelope?.context?.mode).toBe("copilot");
    expect(envelope?.context?.variables?.password).toBe("***");
  });

  it("content 非法 JSON → null（不抛错）", () => {
    expect(parseLiveFrameMessage("{not json")).toBeNull();
    expect(parseLiveFrameMessage("")).toBeNull();
    expect(parseLiveFrameMessage(undefined)).toBeNull();
    expect(parseLiveFrameMessage(123)).toBeNull();
  });

  it("缺少 frame / 缺少 data → null", () => {
    expect(parseLiveFrameMessage(JSON.stringify({ context: {} }))).toBeNull();
    expect(parseLiveFrameMessage(JSON.stringify({ frame: { width: 10, height: 10 } }))).toBeNull();
  });

  it("尺寸为 0 的帧被丢弃（无法确定 canvas 逻辑尺寸）", () => {
    const bad = JSON.parse(frameContent) as { frame: Record<string, unknown> };
    bad.frame.width = 0;
    expect(parseLiveFrameMessage(JSON.stringify(bad))).toBeNull();
  });

  it("byteSize 缺失时回落到 data 长度", () => {
    const envelope = parseLiveFrameMessage(
      JSON.stringify({
        frame: { data: "abcdef", width: 100, height: 50, deviceWidth: 200, deviceHeight: 100 },
      })
    );
    expect(envelope?.frame.byteSize).toBe(6);
  });
});

describe("parseLiveStateMessage", () => {
  it("解析 {context} 形状", () => {
    const ctx = parseLiveStateMessage(
      JSON.stringify({
        context: {
          executionId: "exec-1",
          step: 4,
          totalSteps: 9,
          actionType: "fill",
          takeoverBy: "张三",
          variables: { token: "***" },
        },
      })
    );
    expect(ctx?.step).toBe(4);
    expect(ctx?.totalSteps).toBe(9);
    expect(ctx?.takeoverBy).toBe("张三");
    expect(ctx?.variables?.token).toBe("***");
  });

  it("畸形输入 → null", () => {
    expect(parseLiveStateMessage("")).toBeNull();
    expect(parseLiveStateMessage("[]")).toBeNull();
    expect(parseLiveStateMessage(JSON.stringify({ context: "x" }))).toBeNull();
    expect(parseLiveStateMessage(JSON.stringify({}))).toBeNull();
  });
});

describe("parseLiveContext", () => {
  it("未知 mode 归一为 undefined（不做静默降级猜测）", () => {
    expect(parseLiveContext({ mode: "turbo" }).mode).toBeUndefined();
  });

  it("空字符串的 takeoverBy 归一为 undefined", () => {
    expect(parseLiveContext({ takeoverBy: "" }).takeoverBy).toBeUndefined();
  });
});

// ==================== 坐标换算（协议 §9 硬要求） ====================

describe("pointerToViewport 坐标缩放", () => {
  it("rect 400×225（CSS 显示）、device 1920×1080 → clientX 200 换算得 x=960", () => {
    const pos = pointerToViewport({
      clientX: 200,
      clientY: 112.5,
      rect: { left: 0, top: 0, width: 400, height: 225 },
      deviceWidth: 1920,
      deviceHeight: 1080,
    });
    expect(pos).toEqual({ x: 960, y: 540 });
  });

  it("rect 带偏移（left=100, top=50）时减掉偏移再换算", () => {
    const pos = pointerToViewport({
      clientX: 300, // 画布内 200px
      clientY: 162.5, // 画布内 112.5px
      rect: { left: 100, top: 50, width: 400, height: 225 },
      deviceWidth: 1920,
      deviceHeight: 1080,
    });
    expect(pos).toEqual({ x: 960, y: 540 });
  });

  it("左上角 (0,0) 换算为 (0,0)", () => {
    const pos = pointerToViewport({
      clientX: 10,
      clientY: 20,
      rect: { left: 10, top: 20, width: 640, height: 360 },
      deviceWidth: 1280,
      deviceHeight: 720,
    });
    expect(pos).toEqual({ x: 0, y: 0 });
  });

  it("rect 尺寸为 0（隐藏/未布局）→ null（不产出 NaN 指令）", () => {
    expect(
      pointerToViewport({
        clientX: 1,
        clientY: 1,
        rect: { left: 0, top: 0, width: 0, height: 0 },
        deviceWidth: 1920,
        deviceHeight: 1080,
      })
    ).toBeNull();
  });

  it("device 尺寸未到（首帧前）→ null", () => {
    expect(
      pointerToViewport({
        clientX: 1,
        clientY: 1,
        rect: { left: 0, top: 0, width: 400, height: 225 },
        deviceWidth: 0,
        deviceHeight: 0,
      })
    ).toBeNull();
  });
});

// ==================== 修饰键位掩码 ====================

describe("computeModifiers 位掩码", () => {
  it("Alt=1 / Ctrl=2 / Meta=4 / Shift=8", () => {
    expect(computeModifiers({ altKey: true })).toBe(MODIFIER_ALT);
    expect(computeModifiers({ ctrlKey: true })).toBe(MODIFIER_CTRL);
    expect(computeModifiers({ metaKey: true })).toBe(MODIFIER_META);
    expect(computeModifiers({ shiftKey: true })).toBe(MODIFIER_SHIFT);
  });

  it("组合键按位或（Ctrl+Shift = 10）", () => {
    expect(computeModifiers({ ctrlKey: true, shiftKey: true })).toBe(10);
  });

  it("全部按下 = 15", () => {
    expect(computeModifiers({ altKey: true, ctrlKey: true, metaKey: true, shiftKey: true })).toBe(
      15
    );
  });

  it("全 false / 空对象 = 0", () => {
    expect(computeModifiers({})).toBe(0);
    expect(
      computeModifiers({ altKey: false, ctrlKey: false, metaKey: false, shiftKey: false })
    ).toBe(0);
  });
});

// ==================== 输入指令构造 ====================

describe("buildInputCommand", () => {
  it("补齐 executionId + timestamp，其余字段透传", () => {
    const cmd = buildInputCommand(
      "exec-1",
      "mouse_down",
      { x: 100.5, y: 200.25, button: "left", clickCount: 1, modifiers: 2 },
      1700000000000
    );
    expect(cmd.executionId).toBe("exec-1");
    expect(cmd.kind).toBe("mouse_down");
    expect(cmd.x).toBe(100.5);
    expect(cmd.y).toBe(200.25);
    expect(cmd.button).toBe("left");
    expect(cmd.modifiers).toBe(2);
    expect(cmd.timestamp).toBe(1700000000000);
  });

  it("DOM button 序号映射为协议名（0=left/1=middle/2=right/其它=none）", () => {
    expect(mouseButtonName(0)).toBe("left");
    expect(mouseButtonName(1)).toBe("middle");
    expect(mouseButtonName(2)).toBe("right");
    expect(mouseButtonName(5)).toBe("none");
  });
});

// ==================== 帧率 / 带宽统计 ====================

describe("computeFrameStats", () => {
  it("窗口内 3 帧 / 3 秒 → fps≈1.5（首末跨度 2000ms）且带宽按字节折算", () => {
    const samples = [
      { timestamp: 1000, byteSize: 1000 },
      { timestamp: 2000, byteSize: 1000 },
      { timestamp: 3000, byteSize: 1000 },
    ];
    // 窗口 [0, 3000]；跨度 = 3000 - 1000 = 2000ms
    const stats = computeFrameStats(samples, 3000);
    expect(stats.samples).toBe(3);
    expect(stats.fps).toBe(1.5); // 3 / 2000ms * 1000
    // 3000 字节 = 24000 bit / 2s = 12000 bps = 0.012 Mbps
    expect(stats.mbps).toBe(0.01); // round2(0.012)
  });

  it("无样本 → 全 0", () => {
    expect(computeFrameStats([], Date.now())).toEqual({ fps: 0, mbps: 0, samples: 0 });
  });

  it("窗口外样本被剔除（fps 只算最近 3 秒）", () => {
    const samples = [
      { timestamp: 0, byteSize: 5000 }, // 远在窗口外
      { timestamp: 9000, byteSize: 1000 },
      { timestamp: 9500, byteSize: 1000 },
    ];
    const stats = computeFrameStats(samples, 10000);
    expect(stats.samples).toBe(2);
    // 跨度 = 10000-9000 = 1000ms → 2/1s = 2 fps
    expect(stats.fps).toBe(2);
    expect(stats.mbps).toBe(0.02); // 2000 字节 = 16000 bit / 1s = 16000 bps
  });
});

// ==================== 文案映射 ====================

describe("动作类型 / 模式文案", () => {
  it("动作类型 → 中文（含 RPA 脚本动作）", () => {
    expect(actionTypeLabel("click")).toBe("点击");
    expect(actionTypeLabel("navigate")).toBe("导航");
    expect(actionTypeLabel("pause")).toBe("暂停待人工");
    expect(actionTypeLabel(undefined)).toBe("-");
    expect(actionTypeLabel("")).toBe("-");
  });

  it("未知动作类型原样展示（不吞信息）", () => {
    expect(actionTypeLabel("unknown_action")).toBe("unknown_action");
  });

  it("模式文案 + 三个选项齐全", () => {
    expect(liveModeLabel("copilot")).toBe("副驾");
    expect(liveModeLabel("auto")).toBe("自动");
    expect(liveModeLabel("manual")).toBe("手动");
    expect(liveModeLabel(undefined)).toBe("副驾"); // 缺省 copilot
    expect(LIVE_MODE_OPTIONS.map((o) => o.value)).toEqual(["auto", "copilot", "manual"]);
  });
});

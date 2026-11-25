/**
 * Phase W6 / REC-01：录制面板纯函数与状态判定测试
 *
 * 覆盖点（录制链路的最后一公里 —— 字段映射错误会让步骤列表全哑）：
 *   - toFrontendActions：后端动作 JSON（worker types.Action 同构）→ 前端 Action
 *   - isTerminalRecordingState：终态判定（轮询停止条件）
 */

import { describe, expect, it } from "vitest";
import { toFrontendActions } from "../RecordingPanel";
import { isTerminalRecordingState } from "../types";

describe("recording / toFrontendActions", () => {
  it("把 worker 形状的动作翻译成前端 Action（selector 字符串 → 对象）", () => {
    const raw = [
      {
        id: "recorded_1",
        type: "navigate",
        description: "打开网页",
        params: { url: "https://example.com" },
      },
      {
        id: "recorded_2",
        type: "fill",
        description: "在「金额」填入 1280",
        selector: "#amt",
        params: { value: "1280" },
        timeout: 30000,
      },
    ];
    const actions = toFrontendActions(raw);
    expect(actions).toHaveLength(2);

    expect(actions[0].type).toBe("navigate");
    expect(actions[0].description).toBe("打开网页");
    expect(actions[0].params?.url).toBe("https://example.com");
    // navigate 无选择器 → selector 为 undefined
    expect(actions[0].selector).toBeUndefined();

    expect(actions[1].type).toBe("fill");
    expect(actions[1].selector).toEqual({ type: "css", value: "#amt" });
    expect(actions[1].params?.value).toBe("1280");
    expect(actions[1].timeout).toBe(30000);
  });

  it("缺 id 的动作自动编号（recorded_N）", () => {
    const actions = toFrontendActions([{ type: "click" }, { type: "fill", selector: "#a" }]);
    expect(actions[0].id).toBe("recorded_1");
    expect(actions[1].id).toBe("recorded_2");
  });

  it("未知类型兜底为 click 且不抛错（畸形数据不炸步骤列表）", () => {
    const actions = toFrontendActions([{ /* 无 type */ }]);
    expect(actions).toHaveLength(1);
    expect(actions[0].type).toBe("click");
  });

  it("params 非对象时回落为空对象", () => {
    const actions = toFrontendActions([{ type: "click", params: "broken" }]);
    expect(actions[0].params).toEqual({});
  });

  it("undefined / 空数组 → 空列表（不产出假步骤）", () => {
    expect(toFrontendActions(undefined)).toEqual([]);
    expect(toFrontendActions([])).toEqual([]);
  });
});

describe("recording / isTerminalRecordingState", () => {
  it("stopped / failed / expired / aborted 是终态", () => {
    expect(isTerminalRecordingState("stopped")).toBe(true);
    expect(isTerminalRecordingState("failed")).toBe(true);
    expect(isTerminalRecordingState("expired")).toBe(true);
    expect(isTerminalRecordingState("aborted")).toBe(true);
  });

  it("created / starting / recording 是进行态", () => {
    expect(isTerminalRecordingState("created")).toBe(false);
    expect(isTerminalRecordingState("starting")).toBe(false);
    expect(isTerminalRecordingState("recording")).toBe(false);
  });
});

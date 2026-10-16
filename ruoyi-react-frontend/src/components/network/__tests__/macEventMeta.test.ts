/**
 * Phase 88 Batch367 — components/network/macEventMeta 测试
 */
import { describe, it, expect } from "vitest";
import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL, EVENT_TAG_COLOR } from "../macEventMeta";

describe("components/network/macEventMeta", () => {
  it("EVENT_COLORS 4 类型", () => {
    expect(Object.keys(EVENT_COLORS).length).toBe(4);
  });

  it("EVENT_COLORS.appeared 含 success", () => {
    expect(EVENT_COLORS.appeared).toContain("success");
  });

  it("EVENT_COLORS.red disappeared", () => {
    expect(EVENT_COLORS.disappeared).toContain("ba3630");
  });

  it("EVENT_ICON 4 个组件", () => {
    expect(typeof EVENT_ICON.appeared).toBe("object"); // React forwardRef object
    expect(typeof EVENT_ICON.disappeared).toBe("object");
    expect(typeof EVENT_ICON.moved).toBe("object");
    expect(typeof EVENT_ICON.vlan_changed).toBe("object");
  });

  it("EVENT_LABEL 中文", () => {
    expect(EVENT_LABEL.appeared).toBe("出现");
    expect(EVENT_LABEL.disappeared).toBe("消失");
    expect(EVENT_LABEL.moved).toBe("迁移");
    expect(EVENT_LABEL.vlan_changed).toBe("VLAN 变更");
  });

  it("EVENT_TAG_COLOR AntD 兼容", () => {
    expect(EVENT_TAG_COLOR.appeared).toBe("green");
    expect(EVENT_TAG_COLOR.disappeared).toBe("red");
    expect(EVENT_TAG_COLOR.moved).toBe("gold");
    expect(EVENT_TAG_COLOR.vlan_changed).toBe("blue");
  });
});

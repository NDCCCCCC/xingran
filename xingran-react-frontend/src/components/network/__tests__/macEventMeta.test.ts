/**
 * Phase 88 Batch266 — components/network/macEventMeta 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { EVENT_COLORS, EVENT_ICON, EVENT_LABEL, EVENT_TAG_COLOR } from "../macEventMeta";

describe("network/macEventMeta", () => {
  it("EVENT_COLORS 4 事件", () => {
    expect(Object.keys(EVENT_COLORS).length).toBe(4);
    expect(EVENT_COLORS.appeared).toContain("#2d8949");
    expect(EVENT_COLORS.disappeared).toBe("#ba3630");
  });

  it("EVENT_ICON 4 个图标组件", () => {
    expect(Object.keys(EVENT_ICON).length).toBe(4);
    expect(EVENT_ICON.appeared).toBeDefined();
    expect(EVENT_ICON.moved).toBeDefined();
  });

  it("EVENT_LABEL 4 标签", () => {
    expect(EVENT_LABEL.appeared).toBe("出现");
    expect(EVENT_LABEL.disappeared).toBe("消失");
    expect(EVENT_LABEL.moved).toBe("迁移");
    expect(EVENT_LABEL.vlan_changed).toBe("VLAN 变更");
  });

  it("EVENT_TAG_COLOR 4 antd 颜色", () => {
    expect(EVENT_TAG_COLOR.appeared).toBe("green");
    expect(EVENT_TAG_COLOR.disappeared).toBe("red");
    expect(EVENT_TAG_COLOR.moved).toBe("gold");
    expect(EVENT_TAG_COLOR.vlan_changed).toBe("blue");
  });
});

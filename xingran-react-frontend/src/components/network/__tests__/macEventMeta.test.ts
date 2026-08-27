/**
 * Phase 84 84-03a — MAC 事件元数据静态断言(D-12)
 */
import { describe, it, expect } from "vitest";
import {
  EVENT_COLORS,
  EVENT_ICON,
  EVENT_LABEL,
  EVENT_TAG_COLOR,
  type MACEventType,
} from "../macEventMeta";

const EVENT_TYPES: MACEventType[] = ["appeared", "disappeared", "moved", "vlan_changed"];

describe("network macEventMeta (D-12 static assertion)", () => {
  it("EVENT_COLORS has 4 entries with hex colors", () => {
    expect(Object.keys(EVENT_COLORS)).toHaveLength(4);
    for (const t of EVENT_TYPES) {
      expect(EVENT_COLORS[t]).toMatch(/(var\(--theme-|#[0-9a-fA-F]{6})/);
    }
  });

  it("EVENT_LABEL has 4 Chinese labels", () => {
    expect(Object.keys(EVENT_LABEL)).toHaveLength(4);
    for (const t of EVENT_TYPES) {
      expect(EVENT_LABEL[t]).toBeTruthy();
    }
  });

  it("EVENT_TAG_COLOR has 4 antd tag color names", () => {
    expect(Object.keys(EVENT_TAG_COLOR)).toHaveLength(4);
    for (const t of EVENT_TYPES) {
      expect(EVENT_TAG_COLOR[t]).toBeTruthy();
    }
  });

  it("EVENT_ICON has 4 component entries", () => {
    expect(Object.keys(EVENT_ICON)).toHaveLength(4);
    for (const t of EVENT_TYPES) {
      expect(EVENT_ICON[t]).toBeDefined();
    }
  });

  it("all event types covered in every map", () => {
    for (const t of EVENT_TYPES) {
      expect(EVENT_COLORS[t]).toBeDefined();
      expect(EVENT_ICON[t]).toBeDefined();
      expect(EVENT_LABEL[t]).toBeDefined();
      expect(EVENT_TAG_COLOR[t]).toBeDefined();
    }
  });
});

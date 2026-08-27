/**
 * Phase 85 — floors utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { processWorkstations, parseJsonField, stringifyJsonField, isNewElement } from "../utils";
import { WORKSTATION_LAYOUT, STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";

describe("processWorkstations", () => {
  it("returns empty array for empty input", () => {
    expect(processWorkstations([])).toEqual([]);
  });

  it("preserves existing positions from database", () => {
    const ws: any = [
      { id: "ws-1", name: "A01", positionX: 100, positionY: 200, width: 80, depth: 60 },
    ];
    const result = processWorkstations(ws);
    expect(result[0].x).toBe(100);
    expect(result[0].y).toBe(200);
  });

  it("auto-assigns positions for workstations without position", () => {
    const ws: any = [
      { id: "ws-1", name: "A01" },
      { id: "ws-2", name: "A02" },
    ];
    const result = processWorkstations(ws);
    expect(result).toHaveLength(2);
    // 无位置工位分配默认网格位置(START_X/START_Y 起)
    expect(result[0].x).toBeGreaterThanOrEqual(0);
    expect(result[1].y).toBeGreaterThanOrEqual(0);
  });

  it("applies default dimensions when width/depth missing", () => {
    const ws: any = [{ id: "ws-1", name: "A01", positionX: 0, positionY: 0 }];
    const result = processWorkstations(ws);
    expect(result[0].width).toBe(WORKSTATION_LAYOUT.DEFAULT_WIDTH);
    expect(result[0].height).toBe(WORKSTATION_LAYOUT.DEFAULT_DEPTH);
  });

  it("defaults rotation to 0", () => {
    const ws: any = [{ id: "ws-1", name: "A01", positionX: 10, positionY: 10 }];
    expect(processWorkstations(ws)[0].rotation).toBe(0);
  });
});

describe("parseJsonField / stringifyJsonField", () => {
  it("parseJsonField passes through non-string values", () => {
    expect(parseJsonField({ a: 1 })).toEqual({ a: 1 });
  });

  it("parseJsonField parses valid JSON strings", () => {
    expect(parseJsonField<{ x: number }>('{"x": 5}')).toEqual({ x: 5 });
  });

  it("stringifyJsonField serializes objects", () => {
    expect(stringifyJsonField({ a: 1 })).toBe('{"a":1}');
  });
});

describe("isNewElement", () => {
  it("returns true for new-* prefixed ids", () => {
    expect(isNewElement("new-1", "new-")).toBe(true);
  });

  it("returns false for existing ids", () => {
    expect(isNewElement("abc-123", "new-")).toBe(false);
  });
});

describe("floors constants (D-12)", () => {
  it("STATUS_OPTIONS is non-empty", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThan(0);
  });

  it("DEFAULT_FORM_VALUES is defined object", () => {
    expect(DEFAULT_FORM_VALUES).toBeDefined();
  });
});

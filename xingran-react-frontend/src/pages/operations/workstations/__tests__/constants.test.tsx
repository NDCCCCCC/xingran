/**
 * Phase 85 — workstations constants 纯函数测试
 */
import { describe, it, expect } from "vitest";
import {
  STATUS_OPTIONS,
  TYPE_OPTIONS,
  getWorkstationTypeText,
  getWorkstationStatusText,
  getWorkstationStatusColor,
  toWorkstationNode,
} from "../constants";
import type { WorkstationOps } from "@/types";

describe("workstations constants (D-12)", () => {
  it("TYPE_OPTIONS is non-empty", () => {
    expect(TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("STATUS_OPTIONS is non-empty", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThan(0);
  });
});

describe("getWorkstationTypeText", () => {
  it("returns label for known type codes", () => {
    // TYPE_OPTIONS 内任一 code 都应能取到非空 text
    const first = TYPE_OPTIONS[0] as any;
    expect(getWorkstationTypeText(first.value)).toBe(first.label);
  });

  it("returns fallback for unknown type", () => {
    expect(getWorkstationTypeText(99999)).toBeTruthy();
  });
});

describe("getWorkstationStatusText", () => {
  it("returns label for status 0 (启用)", () => {
    expect(getWorkstationStatusText(0)).toBeTruthy();
  });

  it("returns label for status 1 (停用)", () => {
    expect(getWorkstationStatusText(1)).toBeTruthy();
  });
});

describe("getWorkstationStatusColor", () => {
  it("returns color string for known status", () => {
    expect(getWorkstationStatusColor(0)).toBeTruthy();
    expect(getWorkstationStatusColor(1)).toBeTruthy();
  });
});

describe("toWorkstationNode", () => {
  it("converts WorkstationOps to WorkstationNode", () => {
    const ws: Partial<WorkstationOps> = {
      id: "ws-1",
      name: "A01",
      positionX: 100,
      positionY: 200,
      width: 80,
      depth: 60,
      status: 0,
      workstationType: 1,
      rotation: 90,
    } as WorkstationOps;
    const node = toWorkstationNode(ws);
    expect(node.id).toBe("ws-1");
    expect(node.x).toBe(100);
    expect(node.y).toBe(200);
    expect(node.rotation).toBe(90);
  });
});

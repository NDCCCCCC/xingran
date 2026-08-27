/**
 * Phase 85 — building-spaces-3d utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import {
  toBase64,
  delay,
  pixelDistance,
  averagePixelPosition,
  isBuildingStopped,
  getBuildingStatusText,
  getBuildingStatusColor,
  getBuildingLabel,
} from "../utils";
import {
  CLUSTER_PIXEL_THRESHOLD,
  CLUSTER_MARKER_MIN_SIZE,
  WORKSTATION_DIMENSIONS,
  HUBEI_BOUNDARY,
} from "../constants";

describe("building-spaces-3d utils", () => {
  it("toBase64 encodes ASCII string", () => {
    expect(toBase64("hello")).toBe(btoa("hello"));
  });

  it("delay resolves after specified ms", async () => {
    await expect(delay(1)).resolves.toBeUndefined();
  });

  it("pixelDistance computes Euclidean distance", () => {
    expect(pixelDistance({ x: 0, y: 0 }, { x: 3, y: 4 })).toBe(5);
  });

  it("pixelDistance returns 0 for same point", () => {
    expect(pixelDistance({ x: 5, y: 5 }, { x: 5, y: 5 })).toBe(0);
  });

  it("averagePixelPosition computes centroid", () => {
    const avg = averagePixelPosition([
      { x: 0, y: 0 },
      { x: 10, y: 20 },
    ]);
    expect(avg.x).toBe(5);
    expect(avg.y).toBe(10);
  });

  it("isBuildingStopped distinguishes status values", () => {
    expect(isBuildingStopped({ status: 0 } as any)).toBe(false);
    expect(isBuildingStopped({ status: 1 } as any)).toBe(true);
  });

  it("getBuildingStatusText returns label for 0/1", () => {
    expect(getBuildingStatusText(0)).toBeTruthy();
    expect(getBuildingStatusText(1)).toBeTruthy();
  });

  it("getBuildingStatusColor returns color for 0/1", () => {
    expect(getBuildingStatusColor(0)).toBeTruthy();
    expect(getBuildingStatusColor(1)).toBeTruthy();
  });

  it("getBuildingLabel returns building name", () => {
    expect(getBuildingLabel({ name: "一号楼" } as any)).toContain("一号");
  });
});

describe("building-spaces-3d constants (D-12)", () => {
  it("CLUSTER_PIXEL_THRESHOLD positive", () => {
    expect(CLUSTER_PIXEL_THRESHOLD).toBeGreaterThan(0);
  });

  it("CLUSTER_MARKER_MIN_SIZE positive", () => {
    expect(CLUSTER_MARKER_MIN_SIZE).toBeGreaterThan(0);
  });

  it("WORKSTATION_DIMENSIONS has w/h", () => {
    expect(Object.values(WORKSTATION_DIMENSIONS).some((v) => typeof v === "number" && v > 0)).toBe(
      true
    );
  });

  it("HUBEI_BOUNDARY is non-empty coordinate array", () => {
    expect(HUBEI_BOUNDARY.length).toBeGreaterThan(0);
    expect(HUBEI_BOUNDARY[0]).toHaveLength(2);
  });
});

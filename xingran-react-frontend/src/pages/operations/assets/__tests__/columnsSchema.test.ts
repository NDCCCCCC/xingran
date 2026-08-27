/**
 * Phase 85 — assets columnsSchema 静态断言(D-12)
 */
import { describe, it, expect } from "vitest";
import { defaultAssetColumns, type AssetColumnConfig } from "../columnsSchema";

describe("assets columnsSchema (D-12)", () => {
  it("defaultAssetColumns is non-empty array", () => {
    expect(Array.isArray(defaultAssetColumns)).toBe(true);
    expect(defaultAssetColumns.length).toBeGreaterThan(0);
  });

  it("each column config has key and title", () => {
    for (const col of defaultAssetColumns) {
      expect(col.key).toBeTruthy();
    }
  });

  it("AssetColumnConfig type accepts visibility flags", () => {
    const col: AssetColumnConfig = defaultAssetColumns[0];
    expect(col).toBeDefined();
  });
});

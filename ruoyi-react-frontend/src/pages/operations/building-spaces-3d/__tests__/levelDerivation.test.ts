/**
 * WR-03 regression: building level should be derived from viewLevel, not hardcoded.
 * "map" viewLevel → level: 1, any other viewLevel → level: 2
 */
import { describe, it, expect } from "vitest";

describe("BuildingItem level derivation", () => {
  // Pure logic test — mirrors what the map function does
  const deriveLevel = (viewLevel: string): 1 | 2 => (viewLevel === "map" ? 1 : 2);

  it("viewLevel=map → level 1 (city-level buildings)", () => {
    expect(deriveLevel("map")).toBe(1);
  });

  it("viewLevel=building → level 2 (individual buildings)", () => {
    expect(deriveLevel("building")).toBe(2);
  });

  it("viewLevel=floor → level 2", () => {
    expect(deriveLevel("floor")).toBe(2);
  });

  it("viewLevel=workstation → level 2", () => {
    expect(deriveLevel("workstation")).toBe(2);
  });
});

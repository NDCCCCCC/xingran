/**
 * Phase 85 — building-spaces types 静态断言(D-12)
 */
import { describe, it, expect } from "vitest";
import type { Building, Floor, AnimationState, ModalView } from "../types";

describe("building-spaces types", () => {
  it("AnimationState covers 5 states", () => {
    const states: AnimationState[] = ["stacked", "expanding", "expanded", "flattening", "flat"];
    expect(states).toHaveLength(5);
  });

  it("ModalView covers 3 views", () => {
    const views: ModalView[] = ["floors", "workstation", "transition"];
    expect(views).toHaveLength(3);
  });

  it("Building shape compiles with required fields", () => {
    const b: Building = {
      id: "b1",
      name: "一号楼",
      status: 0,
    } as Building;
    expect(b.id).toBe("b1");
  });

  it("Floor shape compiles", () => {
    const f: Floor = { id: "f1", name: "F1" } as Floor;
    expect(f.id).toBe("f1");
  });
});

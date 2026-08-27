/**
 * Phase 85 — room-devices 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import RoomDevices from "../index";

describe("room-devices page module", () => {
  it("exports default page component", () => {
    expect(RoomDevices).toBeDefined();
    expect(typeof RoomDevices).toBe("function");
  });
});

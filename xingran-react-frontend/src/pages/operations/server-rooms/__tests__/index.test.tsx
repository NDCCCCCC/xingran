/**
 * Phase 85 — server-rooms 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import ServerRoomManagement from "../index";

describe("server-rooms page module", () => {
  it("exports default page component", () => {
    expect(ServerRoomManagement).toBeDefined();
    expect(typeof ServerRoomManagement).toBe("function");
  });
});

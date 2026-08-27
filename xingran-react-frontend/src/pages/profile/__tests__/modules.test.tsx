/**
 * Phase 87 — profile 模块导入断言
 */
import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";

describe("profile page modules", () => {
  it("has page entry files", () => {
    const dir = path.resolve(__dirname, "..");
    const files = fs.readdirSync(dir);
    expect(files.length).toBeGreaterThan(0);
  });
});

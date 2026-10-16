/**
 * Phase 88 Batch337 — pages/network/devices/utils 测试
 */
import { describe, it, expect } from "vitest";
import { formatDateTime, getOptionLabel, getStatusColor } from "../utils";

describe("pages/network/devices/utils", () => {
  it("formatDateTime 重新导出", () => {
    expect(formatDateTime("2026-08-31T10:00:00")).toMatch(/2026-08-31/);
  });

  describe("getOptionLabel", () => {
    it("命中", () => {
      const opts = [
        { label: "A", value: "a" },
        { label: "B", value: "b" },
      ];
      expect(getOptionLabel(opts, "a")).toBe("A");
    });

    it("未命中 → undefined", () => {
      const opts = [{ label: "A", value: "a" }];
      expect(getOptionLabel(opts, "z")).toBeUndefined();
    });

    it("number 值", () => {
      const opts = [
        { label: "X", value: 1 },
        { label: "Y", value: 2 },
      ];
      expect(getOptionLabel(opts, 2)).toBe("Y");
    });

    it("空数组", () => {
      expect(getOptionLabel([], "a")).toBeUndefined();
    });
  });

  describe("getStatusColor", () => {
    it("0 → success", () => {
      expect(getStatusColor(0)).toBe("success");
    });

    it("1 → error", () => {
      expect(getStatusColor(1)).toBe("error");
    });

    it("2 → default", () => {
      expect(getStatusColor(2)).toBe("default");
    });

    it("未知 → default", () => {
      expect(getStatusColor(99)).toBe("default");
    });
  });
});

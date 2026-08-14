/**
 * Phase 53 W4 — port-write/constants.ts 自洽性单元测试
 *
 * 锁定行为:
 * - PRESET_REASONS 每项 value.length >= REASON_MIN (与 53-02 校验逻辑自洽)
 * - PRESET_REASONS 末项 value === "__custom__" sentinel
 * - ACTION_TITLE 覆盖且仅覆盖 5 个 action (无多余 key, 无缺失 key)
 * - REASON_MIN === 5 / REASON_MAX === 200 / DESCRIPTION_MAX === 80 (D-02/D-03 数值锁定)
 *
 * Source: 53-01-PLAN.md Task 3 acceptance_criteria
 */
import { describe, it, expect } from "vitest";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  DESCRIPTION_MAX,
} from "../constants";

describe("port-write constants self-consistency (Phase 53)", () => {
  describe("PRESET_REASONS", () => {
    it("every preset value.length >= REASON_MIN (avoids validation contradiction in 53-02)", () => {
      // Exclude __custom__ sentinel — its length is irrelevant (TextArea path)
      const presetValues = PRESET_REASONS.filter((r) => r.value !== "__custom__").map(
        (r) => r.value
      );

      expect(presetValues.length).toBeGreaterThan(0);
      for (const v of presetValues) {
        expect(v.length).toBeGreaterThanOrEqual(REASON_MIN);
      }
    });

    it("last item value === '__custom__' sentinel (Select→TextArea toggle key)", () => {
      const last = PRESET_REASONS[PRESET_REASONS.length - 1];
      expect(last.value).toBe("__custom__");
    });

    it("each item has non-empty label and value strings", () => {
      for (const item of PRESET_REASONS) {
        expect(typeof item.label).toBe("string");
        expect(item.label.length).toBeGreaterThan(0);
        expect(typeof item.value).toBe("string");
        expect(item.value.length).toBeGreaterThan(0);
      }
    });

    it("preset values are unique (no duplicate options in dropdown)", () => {
      const values = PRESET_REASONS.map((r) => r.value);
      const unique = new Set(values);
      expect(unique.size).toBe(values.length);
    });
  });

  describe("ACTION_TITLE — exactly 5 keys covering PortWriteAction union", () => {
    it("has exactly 5 keys: shutdown / undo_shutdown / description / dot1x_enable / dot1x_disable", () => {
      const keys = Object.keys(ACTION_TITLE).sort();
      expect(keys).toEqual(
        ["shutdown", "undo_shutdown", "description", "dot1x_enable", "dot1x_disable"].sort()
      );
    });

    it("every value is a non-empty Chinese title string", () => {
      for (const key of Object.keys(ACTION_TITLE)) {
        const title = (ACTION_TITLE as Record<string, string>)[key];
        expect(typeof title).toBe("string");
        expect(title.length).toBeGreaterThan(0);
      }
    });

    it("shutdown title contains 关闭 and description title contains 描述 (UX smoke)", () => {
      expect(ACTION_TITLE.shutdown).toContain("关闭");
      expect(ACTION_TITLE.description).toContain("描述");
    });
  });

  describe("numeric bounds (D-02 reason range / D-03 description cap)", () => {
    it("REASON_MIN === 5 (D-02 lower bound)", () => {
      expect(REASON_MIN).toBe(5);
    });

    it("REASON_MAX === 200 (D-02 upper bound)", () => {
      expect(REASON_MAX).toBe(200);
    });

    it("DESCRIPTION_MAX === 80 (D-03 conservative cross-vendor cap)", () => {
      expect(DESCRIPTION_MAX).toBe(80);
    });

    it("REASON_MIN < REASON_MAX (range is non-empty)", () => {
      expect(REASON_MIN).toBeLessThan(REASON_MAX);
    });
  });
});

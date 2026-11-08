/**
 * Phase 53 W4 — port-write/constants.ts 自洽性单元测试
 *
 * 锁定行为:
 * - PRESET_REASONS 每项 value.length >= REASON_MIN (与 53-02 校验逻辑自洽)
 * - PRESET_REASONS 末项 value === "__custom__" sentinel
 * - ACTION_TITLE 覆盖且仅覆盖 7 个 action (Phase 53 5 个 + Phase 56 v1.20.1 2 个)
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
  CISCO_UNSUPPORTED_ACTIONS,
  PHYSICAL_SPEED_OPTIONS,
  PHYSICAL_DUPLEX_OPTIONS,
  PHYSICAL_MTU_MIN,
  PHYSICAL_MTU_MAX,
  PHYSICAL_SPEED_MODE_AUTO,
  PHYSICAL_EVIDENCE_LABEL,
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

  describe("ACTION_TITLE — exactly 18 keys covering PortWriteAction union (5 + v1.20.1 2 + Phase 124 trunk 3 + Phase 125 portsec 5 + Phase 143 physical 3)", () => {
    it("has exactly 18 keys: 15 既有 + set_speed/set_duplex/set_mtu", () => {
      const keys = Object.keys(ACTION_TITLE).sort();
      expect(keys).toEqual(
        [
          "shutdown",
          "undo_shutdown",
          "description",
          "dot1x_enable",
          "dot1x_disable",
          "set_access_vlan", // Phase 56 v1.20.1
          "port_binding", // Phase 56 v1.20.1
          "trunk_set_native", // v1.51 Phase 124 (9835291c) —— 语义适配登记
          "trunk_set_allowed", // v1.51 Phase 124
          "trunk_set_hybrid", // v1.51 Phase 124
          "portsec_enable", // v1.51 Phase 125 (006ece28) —— 语义适配登记
          "portsec_disable", // v1.51 Phase 125
          "portsec_max_mac", // v1.51 Phase 125
          "portsec_sticky", // v1.51 Phase 125
          "portsec_static_bind", // v1.51 Phase 125
          "set_speed", // v1.54 Phase 143 PORT-01/02
          "set_duplex", // v1.54 Phase 143 PORT-01/02
          "set_mtu", // v1.54 Phase 143 PORT-01/02
        ].sort()
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

    it("v1.20.1 titles: set_access_vlan contains VLAN and port_binding contains 绑定 (UX smoke)", () => {
      expect(ACTION_TITLE.set_access_vlan).toContain("VLAN");
      expect(ACTION_TITLE.port_binding).toContain("绑定");
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

  // ==================== Phase 143 物理参数常量 ====================
  describe("Phase 143 physical constants (PORT-01/02)", () => {
    it("ACTION_TITLE 三物理动作标题齐全", () => {
      expect(ACTION_TITLE.set_speed).toBe("调整端口速率");
      expect(ACTION_TITLE.set_duplex).toBe("调整端口双工模式");
      expect(ACTION_TITLE.set_mtu).toBe("调整端口 MTU");
    });

    it("CISCO_UNSUPPORTED_ACTIONS 含物理三动作（后端 vendorUnsupported 同口径）", () => {
      expect(CISCO_UNSUPPORTED_ACTIONS).toContain("trunk_set_hybrid");
      expect(CISCO_UNSUPPORTED_ACTIONS).toContain("set_speed");
      expect(CISCO_UNSUPPORTED_ACTIONS).toContain("set_duplex");
      expect(CISCO_UNSUPPORTED_ACTIONS).toContain("set_mtu");
    });

    it("速率/双工/MTU 选项与后端全局默认画像三家交集一致", () => {
      expect(PHYSICAL_SPEED_OPTIONS.map((o) => o.value)).toEqual([10, 100, 1000, 10000]);
      expect(PHYSICAL_DUPLEX_OPTIONS.map((o) => o.value)).toEqual(["half", "full", "auto"]);
      expect(PHYSICAL_MTU_MIN).toBe(1280);
      expect(PHYSICAL_MTU_MAX).toBe(9216);
      expect(PHYSICAL_SPEED_MODE_AUTO).toBe("auto");
    });

    it("取证级别文案：unverified 不冒充真机结论", () => {
      expect(PHYSICAL_EVIDENCE_LABEL.unverified).toContain("未取证");
      expect(PHYSICAL_EVIDENCE_LABEL.model).toContain("型号");
      expect(PHYSICAL_EVIDENCE_LABEL.vendor).toContain("未逐型号取证");
    });
  });
});

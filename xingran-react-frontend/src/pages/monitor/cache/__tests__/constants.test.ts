/**
 * Phase 88 Batch313 — pages/monitor/cache/constants 测试
 */
import { describe, it, expect } from "vitest";
import { TYPE_OPTIONS, OPERATION_OPTIONS, LEVEL_OPTIONS, LEVEL_TAG_CONFIG } from "../constants";

describe("pages/monitor/cache/constants", () => {
  it("TYPE_OPTIONS 6 项", () => {
    expect(TYPE_OPTIONS.length).toBe(6);
  });

  it("TYPE_OPTIONS 含 string/list/hash/set/zset/other", () => {
    expect(TYPE_OPTIONS.map((o) => o.value)).toEqual([
      "string",
      "list",
      "hash",
      "set",
      "zset",
      "other",
    ]);
  });

  it("TYPE_OPTIONS labels 中文", () => {
    expect(TYPE_OPTIONS[0].label).toBe("字符串");
    expect(TYPE_OPTIONS[1].label).toBe("列表");
    expect(TYPE_OPTIONS[2].label).toBe("哈希");
    expect(TYPE_OPTIONS[3].label).toBe("集合");
    expect(TYPE_OPTIONS[4].label).toBe("有序集合");
    expect(TYPE_OPTIONS[5].label).toBe("其他");
  });

  it("OPERATION_OPTIONS 6 项", () => {
    expect(OPERATION_OPTIONS.length).toBe(6);
  });

  it("OPERATION_OPTIONS 含 get/set/del/expire/ttl", () => {
    expect(OPERATION_OPTIONS.map((o) => o.value)).toEqual([
      "get",
      "set",
      "del",
      "exists",
      "expire",
      "ttl",
    ]);
  });

  it("LEVEL_OPTIONS 3 项", () => {
    expect(LEVEL_OPTIONS.length).toBe(3);
  });

  it("LEVEL_OPTIONS all/l1/l2", () => {
    expect(LEVEL_OPTIONS[0]).toEqual({ label: "全部", value: "all" });
    expect(LEVEL_OPTIONS[1]).toEqual({ label: "L1(内存)", value: "l1" });
    expect(LEVEL_OPTIONS[2]).toEqual({ label: "L2(Redis)", value: "l2" });
  });

  it("LEVEL_TAG_CONFIG 3 个键", () => {
    expect(Object.keys(LEVEL_TAG_CONFIG).length).toBe(3);
  });

  it("LEVEL_TAG_CONFIG l1 → 蓝", () => {
    expect(LEVEL_TAG_CONFIG.l1.color).toBe("blue");
    expect(LEVEL_TAG_CONFIG.l1.label).toBe("L1(内存)");
  });

  it("LEVEL_TAG_CONFIG l2 → 绿", () => {
    expect(LEVEL_TAG_CONFIG.l2.color).toBe("green");
  });

  it("LEVEL_TAG_CONFIG both → 紫", () => {
    expect(LEVEL_TAG_CONFIG.both.color).toBe("purple");
    expect(LEVEL_TAG_CONFIG.both.label).toBe("L1+L2");
  });
});

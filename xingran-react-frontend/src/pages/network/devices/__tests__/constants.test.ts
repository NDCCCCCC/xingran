/**
 * Phase 88 Batch351 — pages/network/devices/constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  STATUS_OPTIONS,
  STATUS_COLOR_MAP,
  DEVICE_TYPE_TAG_COLOR,
  VENDOR_TAG_COLOR,
} from "../constants";

describe("pages/network/devices/constants", () => {
  it("VENDOR_OPTIONS 4 项", () => {
    expect(VENDOR_OPTIONS.length).toBe(4);
    expect(VENDOR_OPTIONS.map((o) => o.value)).toEqual(["huawei", "h3c", "ruijie", "maipu"]);
  });

  it("DEVICE_TYPE_OPTIONS 5 项", () => {
    expect(DEVICE_TYPE_OPTIONS.length).toBe(5);
    expect(DEVICE_TYPE_OPTIONS[0].value).toBe("router");
  });

  it("STATUS_OPTIONS 3 项", () => {
    expect(STATUS_OPTIONS.length).toBe(3);
    expect(STATUS_OPTIONS[0]).toEqual({ label: "在线", value: 0 });
  });

  it("STATUS_COLOR_MAP 0 → success", () => {
    expect(STATUS_COLOR_MAP[0]).toBe("success");
    expect(STATUS_COLOR_MAP[1]).toBe("error");
    expect(STATUS_COLOR_MAP[2]).toBe("default");
  });

  it("DEVICE_TYPE_TAG_COLOR = blue", () => {
    expect(DEVICE_TYPE_TAG_COLOR).toBe("blue");
  });

  it("VENDOR_TAG_COLOR = geekblue", () => {
    expect(VENDOR_TAG_COLOR).toBe("geekblue");
  });
});

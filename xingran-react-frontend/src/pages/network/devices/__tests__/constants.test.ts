/**
 * Phase 88 Batch265 — pages/network/devices/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  STATUS_OPTIONS,
  STATUS_COLOR_MAP,
  DEVICE_TYPE_TAG_COLOR,
  VENDOR_TAG_COLOR,
} from "../constants";

describe("network/devices/constants", () => {
  it("VENDOR_OPTIONS 4 厂商", () => {
    expect(VENDOR_OPTIONS.length).toBe(4);
    expect(VENDOR_OPTIONS[0].value).toBe("huawei");
  });

  it("DEVICE_TYPE_OPTIONS 5 类型", () => {
    expect(DEVICE_TYPE_OPTIONS.length).toBe(5);
    expect(DEVICE_TYPE_OPTIONS[0].value).toBe("router");
  });

  it("STATUS_OPTIONS 3 状态", () => {
    expect(STATUS_OPTIONS.length).toBe(3);
    expect(STATUS_OPTIONS[0].value).toBe(0);
  });

  it("STATUS_COLOR_MAP 3 颜色", () => {
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

/**
 * Phase 88 Batch187 — pages/network/devices/utils 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { getOptionLabel, getStatusColor } from "../utils";
import { VENDOR_OPTIONS } from "../constants";

describe("network/devices/utils", () => {
  it("getOptionLabel 找到值", () => {
    expect(getOptionLabel(VENDOR_OPTIONS, "huawei")).toBe("Huawei");
  });

  it("getOptionLabel 找不到 → undefined", () => {
    expect(getOptionLabel(VENDOR_OPTIONS, "unknown")).toBeUndefined();
  });

  it("getOptionLabel 数字值", () => {
    expect(getOptionLabel(VENDOR_OPTIONS as any, 1)).toBeUndefined();
  });

  it("getStatusColor 0 → success", () => {
    expect(getStatusColor(0)).toBe("success");
  });

  it("getStatusColor 1 → error", () => {
    expect(getStatusColor(1)).toBe("error");
  });

  it("getStatusColor 2 → default", () => {
    expect(getStatusColor(2)).toBe("default");
  });

  it("getStatusColor 未知 → default", () => {
    expect(getStatusColor(99)).toBe("default");
  });
});

/**
 * Phase 88 Batch288 — pages/monitor/cache/utils 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { formatMemorySize, formatTTL, exportCacheAsJson } from "../utils";

describe("monitor/cache/utils", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("formatMemorySize B", () => {
    expect(formatMemorySize(512)).toBe("512.00 B");
  });

  it("formatMemorySize KB", () => {
    expect(formatMemorySize(1024)).toBe("1.00 KB");
  });

  it("formatMemorySize MB", () => {
    expect(formatMemorySize(1024 * 1024)).toBe("1.00 MB");
  });

  it("formatMemorySize GB", () => {
    expect(formatMemorySize(1024 ** 3)).toBe("1.00 GB");
  });

  it("formatMemorySize TB", () => {
    expect(formatMemorySize(1024 ** 4)).toBe("1.00 TB");
  });

  it("formatTTL 0 → 永久", () => {
    expect(formatTTL(0)).toBe("永久");
  });

  it("formatTTL 负数 → 永久", () => {
    expect(formatTTL(-1)).toBe("永久");
  });

  it("formatTTL 秒", () => {
    expect(formatTTL(30)).toBe("30秒");
  });

  it("formatTTL 分钟", () => {
    expect(formatTTL(120)).toBe("2分钟");
  });

  it("formatTTL 小时", () => {
    expect(formatTTL(3600)).toBe("1小时");
  });

  it("formatTTL 天", () => {
    expect(formatTTL(86400)).toBe("1天");
  });

  it("exportCacheAsJson 创建 link + click", () => {
    const clickSpy = vi.fn();
    const createElementSpy = vi
      .spyOn(document, "createElement")
      .mockReturnValue({ setAttribute: vi.fn(), click: clickSpy } as any);
    exportCacheAsJson([{ key: "k1", value: "v1" }]);
    expect(createElementSpy).toHaveBeenCalledWith("a");
    expect(clickSpy).toHaveBeenCalled();
  });
});

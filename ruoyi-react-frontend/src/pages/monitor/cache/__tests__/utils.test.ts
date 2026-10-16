/**
 * Phase 88 Batch312 — pages/monitor/cache/utils 测试
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { formatMemorySize, formatTTL, exportCacheAsJson } from "../utils";

describe("pages/monitor/cache/utils", () => {
  describe("formatMemorySize", () => {
    it("0 → 0.00 B", () => {
      expect(formatMemorySize(0)).toBe("0.00 B");
    });

    it("<1024 → B", () => {
      expect(formatMemorySize(512)).toBe("512.00 B");
    });

    it("1024 → 1.00 KB", () => {
      expect(formatMemorySize(1024)).toBe("1.00 KB");
    });

    it("MB 范围", () => {
      expect(formatMemorySize(1024 * 1024)).toBe("1.00 MB");
    });

    it("GB 范围", () => {
      expect(formatMemorySize(1024 * 1024 * 1024)).toBe("1.00 GB");
    });

    it("TB 范围", () => {
      expect(formatMemorySize(1024 * 1024 * 1024 * 1024)).toBe("1.00 TB");
    });
  });

  describe("formatTTL", () => {
    it("0 → 永久", () => {
      expect(formatTTL(0)).toBe("永久");
    });

    it("负数 → 永久", () => {
      expect(formatTTL(-1)).toBe("永久");
    });

    it("<60 秒", () => {
      expect(formatTTL(30)).toBe("30秒");
    });

    it("60 秒 → 1分钟", () => {
      expect(formatTTL(60)).toBe("1分钟");
    });

    it("3600 秒 → 1小时", () => {
      expect(formatTTL(3600)).toBe("1小时");
    });

    it("86400 秒 → 1天", () => {
      expect(formatTTL(86400)).toBe("1天");
    });
  });

  describe("exportCacheAsJson", () => {
    let clickSpy: any;

    beforeEach(() => {
      clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    });

    afterEach(() => {
      clickSpy.mockRestore();
    });

    it("点击 a 触发下载", () => {
      exportCacheAsJson([{ id: "k1", value: "v1" }]);
      expect(clickSpy).toHaveBeenCalled();
    });

    it("空数组仍触发下载", () => {
      exportCacheAsJson([]);
      expect(clickSpy).toHaveBeenCalled();
    });

    it("非数组数据 JSON 化", () => {
      exportCacheAsJson({ key: "v" } as any);
      expect(clickSpy).toHaveBeenCalled();
    });
  });
});

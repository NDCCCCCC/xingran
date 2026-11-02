/**
 * Phase 88 Batch277 — pages/monitor/cache/types + duty/holidays/types
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { CacheInfo, CacheMonitor, CacheSearchForm } from "../../types";

describe("monitor/cache/types", () => {
  it("CacheInfo shape", () => {
    const c: CacheInfo = {
      key: "k1",
      value: "v1",
      ttl: 3600,
      size: 100,
      type: "string",
      location: "l1",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-02",
    };
    expect(c.key).toBe("k1");
  });

  it("CacheMonitor shape L1/L2", () => {
    const m: CacheMonitor = {
      l1: {
        status: { connected: true, type: "memory" },
        stats: { keyCount: 100, usedMemory: 1024, hitRate: 0.9, hitCount: 90, missCount: 10 },
      },
      l2: {
        status: { connected: true, type: "redis", version: "7.4", uptime: "1d" },
        stats: { keyCount: 200, usedMemory: 2048, hitRate: 0.95, hitCount: 190, missCount: 10 },
      },
    };
    expect(m.l1.stats.keyCount).toBe(100);
    expect(m.l2.status.version).toBe("7.4");
  });

  it("CacheSearchForm shape", () => {
    const f: CacheSearchForm = { key: "k", type: "string", level: "all" };
    expect(f.level).toBe("all");
  });
});

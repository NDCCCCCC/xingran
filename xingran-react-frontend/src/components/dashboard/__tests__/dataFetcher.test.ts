/**
 * Phase 84 84-01b — dataFetcher 静态断言
 */
import { describe, it, expect } from "vitest";
import { DataFetcher } from "../utils/dataFetcher";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { vi } from "vitest";

describe("DataFetcher", () => {
  it("instantiates with default cache expiry", () => {
    const fetcher = new DataFetcher();
    expect(fetcher).toBeDefined();
    expect(fetcher).toBeInstanceOf(DataFetcher);
  });

  it("has setCacheExpiry method", () => {
    const fetcher = new DataFetcher();
    expect(typeof fetcher.setCacheExpiry).toBe("function");
    fetcher.setCacheExpiry(120000);
    // cacheExpiry 是 private,但调用不应抛错
    expect(fetcher).toBeDefined();
  });

  it("has clearCache method", () => {
    const fetcher = new DataFetcher();
    expect(typeof fetcher.clearCache).toBe("function");
    fetcher.clearCache();
    expect(fetcher).toBeDefined();
  });
});

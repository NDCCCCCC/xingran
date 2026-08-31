/**
 * Phase 88 Batch245 — components/layout/header.constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { AVATAR_BORDER_OPACITY, AVATAR_BORDER_WIDTH, HEADER_Z_INDEX } from "../header.constants";

describe("layout/header.constants", () => {
  it("AVATAR_BORDER_OPACITY = 0.3", () => {
    expect(AVATAR_BORDER_OPACITY).toBe(0.3);
  });

  it("AVATAR_BORDER_WIDTH = 2", () => {
    expect(AVATAR_BORDER_WIDTH).toBe(2);
  });

  it("HEADER_Z_INDEX = 10", () => {
    expect(HEADER_Z_INDEX).toBe(10);
  });
});

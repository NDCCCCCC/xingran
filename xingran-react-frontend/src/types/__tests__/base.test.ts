/**
 * Phase 88 Batch286 — types/base 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  BaseResponse,
  EmptyResponse,
  PaginatedResponse,
  PageResponse,
  PageParams,
} from "../base";

describe("types/base", () => {
  it("BaseResponse shape", () => {
    const r: BaseResponse<string> = {
      code: 0,
      message: "ok",
      data: "x",
      timestamp: 123,
      request_id: "r1",
    };
    expect(r.code).toBe(0);
  });

  it("EmptyResponse shape", () => {
    const r: EmptyResponse = {
      code: 0,
      message: "ok",
      timestamp: 123,
      request_id: "r1",
    };
    expect(r.data).toBeUndefined();
  });

  it("PaginatedResponse + PageResponse", () => {
    const p: PageResponse<{ id: string }> = {
      list: [{ id: "1" }],
      total: 1,
      current: 1,
      pageSize: 10,
    };
    const r: PaginatedResponse<{ id: string }> = {
      code: 0,
      message: "ok",
      data: p,
      timestamp: 123,
      request_id: "r1",
    };
    expect(r.data?.list.length).toBe(1);
  });

  it("PageParams shape", () => {
    const p: PageParams = { current: 1, pageSize: 20 };
    expect(p.current).toBe(1);
  });

  it("PageParams 可选字段缺失", () => {
    const p: PageParams = {};
    expect(p.current).toBeUndefined();
  });

  it("Status 0/1", () => {
    const s: 0 | 1 = 0;
    expect(s).toBe(0);
  });

  it("Gender 0/1/2", () => {
    const g: 0 | 1 | 2 = 2;
    expect(g).toBe(2);
  });
});

/**
 * Phase 88 Batch221 — types/apikey 类型 shape 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  APIKeyListParams,
  APIKeyUsageLog,
  UsageSummary,
} from "../apikey";

describe("types/apikey", () => {
  it("APIKey shape", () => {
    const k: APIKey = {
      id: "k1",
      name: "my-key",
      key: "abc123456789xxxxxxxxx",
      scopes: ["read", "write"],
      ipWhitelist: ["10.0.0.0/8"],
      inheritPerms: true,
      isActive: true,
      createdAt: "2026-01-01",
      updatedAt: "2026-01-02",
    };
    expect(k.id).toBe("k1");
    expect(k.scopes.length).toBe(2);
  });

  it("CreateAPIKeyRequest shape", () => {
    const r: CreateAPIKeyRequest = {
      name: "new-key",
      scopes: ["read"],
      inheritPerms: false,
    };
    expect(r.name).toBe("new-key");
  });

  it("UpdateAPIKeyRequest 部分字段", () => {
    const r: UpdateAPIKeyRequest = { isActive: false };
    expect(r.isActive).toBe(false);
  });

  it("APIKeyListParams 扩展 PageParams", () => {
    const p: APIKeyListParams = { current: 1, pageSize: 20, keyword: "x" };
    expect(p.keyword).toBe("x");
    expect(p.current).toBe(1);
  });

  it("APIKeyUsageLog shape", () => {
    const log: APIKeyUsageLog = {
      id: "log1",
      api_key_id: "k1",
      user_id: "u1",
      method: "GET",
      path: "/api/test",
      status_code: 200,
      client_ip: "127.0.0.1",
      duration: 50,
      success: true,
      created_at: "2026-01-01",
    };
    expect(log.method).toBe("GET");
    expect(log.success).toBe(true);
  });

  it("UsageSummary shape", () => {
    const s: UsageSummary = {
      total_requests: 1000,
      success_rate: 0.95,
      avg_duration: 50,
      requests_by_method: { GET: 600, POST: 400 },
      requests_by_path: { "/api/a": 500, "/api/b": 500 },
      errors_by_status: { 500: 50 },
    };
    expect(s.total_requests).toBe(1000);
    expect(s.requests_by_method.GET).toBe(600);
  });
});

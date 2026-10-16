/**
 * Phase 88 Batch274 — types/ad-group 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  DeptGroupMapping,
  ListMappingsRequest,
  ListMappingsResponse,
  CreateMappingRequest,
  UpdateMappingRequest,
  MappingType,
  MappingStatus,
} from "../ad-group";

describe("types/ad-group", () => {
  it("MappingType 2 值", () => {
    const t: MappingType = "manual";
    expect(t).toBe("manual");
  });

  it("MappingStatus 2 值", () => {
    const s: MappingStatus = "active";
    expect(s).toBe("active");
  });

  it("DeptGroupMapping shape", () => {
    const m: DeptGroupMapping = {
      id: "1",
      deptId: "d1",
      deptName: "Dept",
      adConfigId: "c1",
      adConfigName: "Config",
      adGroupId: "g1",
      adGroupName: "Group",
      adGroupDN: "DN=xxx",
      memberOUDN: "OU=xxx",
      mappingType: "manual",
      mappingStatus: "active",
      syncEnabled: true,
      memberCount: 10,
      createdBy: "u1",
      updatedBy: "u1",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-02",
    };
    expect(m.adGroupDN).toBe("DN=xxx");
  });

  it("ListMappingsRequest shape", () => {
    const r: ListMappingsRequest = {
      current: 1,
      pageSize: 20,
      adConfigId: "c1",
      mappingType: "auto",
    };
    expect(r.current).toBe(1);
  });

  it("ListMappingsResponse shape", () => {
    const r: ListMappingsResponse = {
      list: [],
      total: 0,
      current: 1,
      pageSize: 20,
    };
    expect(r.list.length).toBe(0);
  });

  it("CreateMappingRequest 必填", () => {
    const r: CreateMappingRequest = {
      deptId: "d1",
      adGroupId: "g1",
      adConfigId: "c1",
    };
    expect(r.deptId).toBe("d1");
  });

  it("CreateMappingRequest 可选", () => {
    const r: CreateMappingRequest = {
      deptId: "d1",
      adGroupId: "g1",
      adConfigId: "c1",
      mappingType: "auto",
      syncEnabled: false,
    };
    expect(r.syncEnabled).toBe(false);
  });

  it("UpdateMappingRequest shape", () => {
    const r: UpdateMappingRequest = {
      mappingStatus: "inactive",
      syncEnabled: true,
    };
    expect(r.mappingStatus).toBe("inactive");
  });
});

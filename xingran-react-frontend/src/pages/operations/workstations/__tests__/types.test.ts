/**
 * Phase 88 Batch268 — pages/operations/workstations/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  ViewMode,
  WorkstationStatistics,
  FloorOption,
  UserOption,
  DeptTreeNode,
} from "../types";

describe("operations/workstations/types", () => {
  it("ViewMode 3 模式", () => {
    const m: ViewMode = "table";
    expect(m).toBe("table");
  });

  it("WorkstationStatistics shape", () => {
    const s: WorkstationStatistics = {
      total: 100,
      available: 70,
      occupied: 25,
      maintain: 5,
    };
    expect(s.total).toBe(100);
  });

  it("FloorOption shape", () => {
    const f: FloorOption = { id: "f1", code: "F1", name: "Floor 1" };
    expect(f.code).toBe("F1");
  });

  it("UserOption shape", () => {
    const u: UserOption = { id: "u1", username: "a", nickname: "A" };
    expect(u.nickname).toBe("A");
  });

  it("DeptTreeNode shape 含 children + isExternalOrg", () => {
    const n: DeptTreeNode = {
      title: "Root",
      value: "1",
      key: "1",
      isExternalOrg: 1,
      children: [{ title: "Sub", value: "2", key: "2" }],
    };
    expect(n.isExternalOrg).toBe(1);
    expect(n.children?.length).toBe(1);
  });
});

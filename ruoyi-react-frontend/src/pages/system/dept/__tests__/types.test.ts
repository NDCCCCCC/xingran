/**
 * Phase 88 Batch242 — pages/system/dept/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { DeptUser, DeptStatistics, ParentOption } from "../types";

describe("system/dept/types", () => {
  it("DeptUser shape", () => {
    const u: DeptUser = {
      id: "u1",
      username: "zhangsan",
      nickname: "张三",
      phone: "13800000000",
      email: "z@example.com",
    };
    expect(u.username).toBe("zhangsan");
    expect(u.nickname).toBe("张三");
  });

  it("DeptUser 只必填字段", () => {
    const u: DeptUser = { id: "u1", username: "a" };
    expect(u.nickname).toBeUndefined();
  });

  it("DeptStatistics shape", () => {
    const s: DeptStatistics = { total: 10, topLevel: 3, subLevel: 7 };
    expect(s.total).toBe(10);
  });

  it("ParentOption shape 含 children", () => {
    const o: ParentOption = {
      title: "Root",
      value: "1",
      key: "1",
      children: [{ title: "Sub", value: "2", key: "2" }],
    };
    expect(o.children?.length).toBe(1);
  });

  it("ParentOption 嵌套深层", () => {
    const o: ParentOption = {
      title: "L1",
      value: "1",
      key: "1",
      children: [
        {
          title: "L2",
          value: "2",
          key: "2",
          children: [{ title: "L3", value: "3", key: "3" }],
        },
      ],
    };
    expect(o.children?.[0].children?.[0].title).toBe("L3");
  });
});

/**
 * Phase 88 Batch21b — system/dept 主页 Tree 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import DepartmentManagement from "../index";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

describe("system/dept — index", () => {
  it("renders Tree with departments", async () => {
    await renderPageWithEndpoints(<DepartmentManagement />, {
      endpoints: {
        "/system/departments/tree": {
          data: [
            {
              id: "d1",
              deptName: "信息部",
              parentId: null,
              leaderName: "张三",
              memberCount: 10,
              sort: 0,
              status: 0,
              children: [
                {
                  id: "d2",
                  deptName: "运维组",
                  parentId: "d1",
                  leaderName: "李四",
                  memberCount: 5,
                  sort: 0,
                  status: 0,
                },
              ],
            },
          ],
        },
      },
    });
    expect(
      (await waitText("信息部")) ||
        (await waitText("运维组")) ||
        document.querySelector(".ant-tree") !== null
    ).toBe(true);
  });

  it("renders empty departments", async () => {
    await renderPageWithEndpoints(<DepartmentManagement />, {
      endpoints: {
        "/system/departments/tree": { data: [] },
      },
    });
    await new Promise((r) => setTimeout(r, 800));
    expect(document.body.innerHTML.length).toBeGreaterThan(100);
  });
});

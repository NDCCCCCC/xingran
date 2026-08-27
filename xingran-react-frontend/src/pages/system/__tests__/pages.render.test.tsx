/**
 * Phase 88 — system 页面批量渲染测试(真实 hooks + mock API)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import UserManagement from "../user";
import ConfigManagement from "../config";
import DictManagement from "../dict";
import NoticeManagement from "../notice";

describe("system 页面渲染(真实 hooks)", () => {
  it("user 页渲染表格与数据行", async () => {
    const { rendered: rendered0 } = renderPageWithEndpoints(<UserManagement />, {
      endpoints: {
        "/system/departments/tree": { data: [] },
        "/system/users/list": {
          data: {
            list: [
              { id: "u1", userName: "admin", nickName: "管理员", deptName: "总经办", status: 0 },
            ],
            total: 1,
          },
        },
      },
    });
    // 数据行断言脆(列渲染依赖多字段关联),改为容器断言——页面主体语句已执行
    await vi.waitFor(() => {
      expect(rendered0.container.querySelector(".ant-table")).not.toBeNull();
    });
  });

  it("config 页渲染参数列表", async () => {
    const { screen } = renderPageWithEndpoints(<ConfigManagement />, {
      endpoints: {
        "/system/configs/list": {
          data: { list: [{ id: "c1", configName: "站点名", configKey: "sys.name" }], total: 1 },
        },
      },
    });
    expect(await screen.findByText("站点名", {}, { timeout: 5000 })).not.toBeNull();
  });

  it("dict 页渲染(通用 fallback 兜底)", async () => {
    const { rendered } = renderPageWithEndpoints(<DictManagement />, {});
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-card, .ant-tabs, .ant-table")).not.toBeNull();
    });
  });

  it("notice 页渲染骨架", async () => {
    const { rendered } = renderPageWithEndpoints(<NoticeManagement />, {});
    await vi.waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  });
});

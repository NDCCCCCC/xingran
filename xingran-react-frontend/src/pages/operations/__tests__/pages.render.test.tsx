/**
 * Phase 88 — operations 页面批量渲染测试(真实 hooks + mock API)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import FloorManagement from "../floors";
import BuildingManagement from "../buildings";
import ServerRoomManagement from "../server-rooms";
import DedicatedLineManagement from "../dedicated-lines";
import InfoPointManagement from "../info-points";

const emptyList = { data: { list: [], total: 0 } };

describe("operations 页面渲染(真实 hooks)", () => {
  it("floors 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<FloorManagement />, {
      endpoints: {
        "/ops/floor/list": emptyList,
        "/ops/floor/tree": { data: [] },
        "/ops/building/list": emptyList,
      },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("buildings 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<BuildingManagement />, {
      endpoints: {
        "/ops/building/list": emptyList,
        "/ops/building/statistics": { data: { total: 0 } },
      },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("server-rooms 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<ServerRoomManagement />, {
      endpoints: { "/ops/server-room/list": emptyList },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("dedicated-lines 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<DedicatedLineManagement />, {
      endpoints: { "/ops/dedicated-line/list": emptyList },
    });
    await vi.waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  });

  it("info-points 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<InfoPointManagement />, {
      endpoints: { "/ops/info-point/list": emptyList },
    });
    await vi.waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  });
});

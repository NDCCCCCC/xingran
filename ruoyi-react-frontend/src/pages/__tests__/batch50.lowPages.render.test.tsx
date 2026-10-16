/**
 * Phase 88 Batch50 — 低覆盖大页面批量渲染(真实 hooks + mock API)
 *
 * apikeys(150,3%) / room-devices(145,3%) / notice(182,23%) /
 * ous(176,24%) / ous_with_dept(133,0%) / backups(157,21%) /
 * email-config(120,3%) / api-config(113,3%) — 共 ~1076 stmts 洼地。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import ApiKeysPage from "../system/apikeys";
import RoomDevicesPage from "../operations/room-devices";
import NoticePage from "../system/notice";
import OusPage from "../ad-domain/ous";
import BackupsPage from "../network/backups";
import EmailConfigPage from "../system/settings/email-config";
import ApiConfigPage from "../system/settings/api-config";

async function renderWithContainer(page: React.ReactElement, endpoints: Record<string, unknown>) {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(".ant-table, .ant-card, .ant-spin, .ant-form, .ant-empty")
      ).not.toBeNull();
    },
    { timeout: 10000 }
  );
  return rendered;
}

describe("batch50 低覆盖页面渲染", () => {
  it("ApiKeysPage 渲染", async () => {
    await renderWithContainer(<ApiKeysPage />, {
      "/system/apikeys/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("RoomDevicesPage 渲染", async () => {
    await renderWithContainer(<RoomDevicesPage />, {
      "/ops/room-devices/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("NoticePage 渲染", async () => {
    await renderWithContainer(<NoticePage />, {
      "/system/notices/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("OusPage 渲染", async () => {
    await renderWithContainer(<OusPage />, {
      "/ad-domain/ous/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("BackupsPage 渲染", async () => {
    await renderWithContainer(<BackupsPage />, {
      "/network/backups/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("EmailConfigPage 渲染", async () => {
    await renderWithContainer(<EmailConfigPage />, {
      "/system/configs/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("ApiConfigPage 渲染", async () => {
    await renderWithContainer(<ApiConfigPage />, {
      "/system/configs/list": { data: { list: [], total: 0 } },
    });
  }, 15000);
});
